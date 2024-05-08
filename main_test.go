package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Set up test environment
	setup()
	code := m.Run()
	// Tear down test environment
	tearDown()
	os.Exit(code)
}

func setup() {
	// Initialize database for testing
	if err := initDB(".env.testing"); err != nil {
		panic(err)
	}
}

func tearDown() {
	// Close database connection after testing
	if db != nil {
		db.Close()
	}
}

func TestCreateCommand(t *testing.T) {
	// Create a mock HTTP request with a JSON body
	jsonStr := []byte(`{"command": "echo hello"}`)
	req, err := http.NewRequest("POST", "/commands", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createCommand)

	// Call the handler function
	handler.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Query the command from the database
	var cmd Command
	err = db.QueryRow("SELECT * FROM commands ORDER BY id DESC").Scan(&cmd.ID, &cmd.Command, &cmd.Result)
	assert.NoError(t, err)
	assert.Equal(t, "echo hello", cmd.Command)
	assert.Equal(t, "hello\n", cmd.Result)
}

func TestGetCommands(t *testing.T) {
	// Insert some test commands into the database
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM commands").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}

	// Create a mock HTTP request
	req, err := http.NewRequest("GET", "/commands", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getCommands)

	// Call the handler function
	handler.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Decode the response body
	var commands []Command
	err = json.NewDecoder(rr.Body).Decode(&commands)
	assert.NoError(t, err)

	// Check the number of commands returned
	assert.Len(t, commands, count)
}

func TestGetCommand(t *testing.T) {
	// Insert a test command into the database

	var lastInsertedID int
	err := db.QueryRow("INSERT INTO commands (command, result) VALUES($1, $2) RETURNING id", "echo command1", "result1").Scan(&lastInsertedID)
	assert.NoError(t, err)

	strValue := strconv.Itoa(lastInsertedID)
	// Create a mock HTTP request with the command ID

	req, err := http.NewRequest("GET", "/commands/"+strValue, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Create a new router
	r := mux.NewRouter()
	r.HandleFunc("/commands/{id}", getCommand)

	// Serve the request using the router
	r.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Decode the response body
	var cmd Command
	err = json.NewDecoder(rr.Body).Decode(&cmd)
	assert.NoError(t, err)

	// Check the content of the command
	assert.Equal(t, "echo command1", cmd.Command)
	assert.Equal(t, "result1", cmd.Result)
}
