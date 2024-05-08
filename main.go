package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Command represents a bash command in the database
type Command struct {
	ID      int    `json:"id"`
	Command string `json:"command"`
	Result  string `json:"result"`
}

var (
	db             *sql.DB
	stopSignal     = make(chan struct{})
	stopSignalLock sync.Mutex
)

func main() {
	// Initialize database connection
	if err := initDB(".env"); err != nil {
		log.Fatal(err)
	}
	// Initialize router
	r := mux.NewRouter()

	// Define API endpoints
	r.HandleFunc("/commands", createCommand).Methods("POST")
	r.HandleFunc("/commands", getCommands).Methods("GET")
	r.HandleFunc("/commands/{id}", getCommand).Methods("GET")
	r.HandleFunc("/commands/stop", stopScript).Methods("POST")

	// Start HTTP server
	log.Fatal(http.ListenAndServe(":8085", r))
}

func initDB(env string) error {
	var err error
	// Connect to Postgres database
	// Load environment variables from .env file
	if err := godotenv.Load(env); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Get database connection details from environment variables
	connStr := "postgres://" + os.Getenv("DB_USER") + ":" + os.Getenv("DB_PASSWORD") + "@" + os.Getenv("DB_HOST") + "/" + os.Getenv("DB_NAME") + "?sslmode=disable"

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	// Create commands table if not exists
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS commands (
	id SERIAL PRIMARY KEY,
	command TEXT NOT NULL,
	result TEXT
)`)
	if err != nil {
		return err
	}

	return nil
}

func createCommand(w http.ResponseWriter, r *http.Request) {
	var cmd Command
	// Decode JSON body into Command struct
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	go func(cmd Command) {
		// Start executing bash command
		cmdExec := exec.Command("bash", "-c", cmd.Command)

		// Create a pipe to capture the command's output
		stdout, err := cmdExec.StdoutPipe()
		if err != nil {
			log.Println("Error creating pipe:", err)
			return
		}

		// Start the command
		if err := cmdExec.Start(); err != nil {
			log.Println("Error starting command:", err)
			return
		}

		// Read command output and write to database
		var result string
		buf := make([]byte, 4096) // Buffer to read output
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				result += string(buf[:n])
				// Write partial result to database
				if err := saveCommandResult(cmd, result); err != nil {
					log.Println("Error saving partial result:", err)
					// Optionally, you might choose to break or handle the error differently
				}
			}
			if err != nil {
				break // Exit loop on error
			}
		}

		// Wait for command to finish
		if err := cmdExec.Wait(); err != nil {
			log.Println("Error waiting for command:", err)
			return
		}

		// Write final result to database
		if err := saveCommandResult(cmd, result); err != nil {
			log.Println("Error saving final result:", err)
			return
		}
	}(cmd)

	// Respond with success
	w.WriteHeader(http.StatusCreated)
}

func saveCommandResult(cmd Command, result string) error {
	_, err := db.Exec("INSERT INTO commands (command, result) VALUES ($1, $2)", cmd.Command, result)
	return err
}

func getCommands(w http.ResponseWriter, r *http.Request) {
	// Query all commands from database
	rows, err := db.Query("SELECT * FROM commands")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var commands []Command
	// Iterate through query results and append to commands slice
	for rows.Next() {
		var cmd Command
		if err := rows.Scan(&cmd.ID, &cmd.Command, &cmd.Result); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		commands = append(commands, cmd)
	}

	// Encode commands slice to JSON and send response
	json.NewEncoder(w).Encode(commands)
}

func getCommand(w http.ResponseWriter, r *http.Request) {
	// Extract command ID from request parameters
	vars := mux.Vars(r)
	id := vars["id"]

	var cmd Command
	// Query command from database by ID
	row := db.QueryRow("SELECT * FROM commands WHERE id = $1", id)
	if err := row.Scan(&cmd.ID, &cmd.Command, &cmd.Result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Encode command struct to JSON and send response
	json.NewEncoder(w).Encode(cmd)
}

func stopScript(w http.ResponseWriter, r *http.Request) {
	// Signal the command to stop
	stopSignalLock.Lock()
	defer stopSignalLock.Unlock()
	select {
	case <-stopSignal:
		http.Error(w, "Command already stopped", http.StatusBadRequest)
	default:
		close(stopSignal)
	}
}
