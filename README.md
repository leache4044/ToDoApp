# WindSurf Project

A simple Go web server project.

## Getting Started

### Prerequisites

- Go 1.21 or later

### Running the Application

1. Clone the repository
2. Run the application from the repo root:
   ```bash
   go run ./cmd/server
   ```
3. Open your browser and visit: http://localhost:8000

## Project Structure

- `cmd/server` - Main application entry point (HTTP server)
- `internal/todo` - Todo models, store, service facade, and HTTP handlers
- `go.mod` - Go module definition
