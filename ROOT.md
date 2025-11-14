# windsuf Todo API

This repository is a small HATEOAS-style Todo API written in Go.

## Project layout

- **cmd/server**
  - Entry point for the HTTP server.
- **internal/todo**
  - Todo models, in-memory store, handlers, and router wiring.

## Running the server

From the repo root:

```bash
go run ./cmd/server
```

Then try:

- `GET http://localhost:8000/`
- `GET http://localhost:8000/todos`

