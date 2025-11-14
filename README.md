# WindSurf Project

A simple Go web server project.
It implements HATEOAS for a todo list.
It uses CHI for routing.
It uses chi middleware for logging and error handling.

## Getting Started

### Prerequisites

- Go 1.21 or later

### Running the Application

1. Clone the repository
2. From the repo root, run:
   ```bash
   go run ./cmd/server
   ```
3. Open your browser and visit: http://localhost:8000

## Project Structure

- `cmd/server` - Main application entry point (Todo HTTP API server)
- `internal/todo` - Todo models, store, service facade, and HTTP handlers
- `go.mod` - Go module definition

## Logging & Error Handling

- `cmd/server/main.go` uses `log.Fatal` around `http.ListenAndServe` to log and exit on server startup errors.
- The `internal/todo` router, built with chi, configures middleware:
  - `middleware.Logger` to log each HTTP request.
  - `middleware.Recoverer` to recover from panics and return `500` instead of crashing the server.

## Testing & Coverage

- Run all tests:
  ```bash
  go test ./...
  ```
- Run tests with coverage (per-package summary):
  ```bash
  go test ./... -cover
  ```
- The main logic package `internal/todo` currently achieves around **97%** statement coverage, including store, service, HTTP handlers, and router behavior.
