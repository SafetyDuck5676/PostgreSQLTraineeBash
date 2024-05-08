# BashRest

Данное приложение, предоставляет возможность использовать REST API для запуска команд. Используется база данных для хранения команд. Также написаны тесты.

API содержит следующий функционал:

- Создание новой команды. Запускает переданную bash-команду, сохраняет результат выполнения в БД.
- Получение списка команд
- Получение одной команды
- API может остановить выполнение команды
- Сохраняется вывод команды в БД по мере её выполнения.
- Докер сборка для запуска.

# Как запустить:
Сервис использует .env для конфиденциальных данных и .env.testing для тестов. В качестве примера оставил его в репо. Также используется Docker, POST, GET Методы, PostgreSQL для ДБ.

## Процесс запуска:
```
1. docker compose up -d --build
```
С помощью docker-compose и dockerfile, создается среда Go и PostgreSQL. Программа уже будет запущена внутри контейнера. У приложения несколько эндпоинтов:
```
1.1 POST localhost:8085/commands
1.2 GET localhost:8085/commands
1.3 POST localhost:8085/commands/{id}
1.4 GET localhost:8085/commands/stop
```
С помощью них можно получить вызов списка команды либо же, используя CURL либо другие методы POST, запросить команду или же остановить её. Пример CURL:
```
2. curl -X POST -H "Content-Type: application/json" -d '{"id":0, "command":"echo \"Hello World\" ", "result":" "}/' localhost:8085/commands
```

3. Параметр запуска находится в dockerfile в команде "CMD".

4. Тесты можно выполнить внутри контейнера. 
```
docker exec -it <container> bash
go test  
```

## Используемые технологии

Операционная система: Linux SUSe Leap

Язык программирования: Golang

Библиотеки: Стандартная библиотека

База данных: Postgres
