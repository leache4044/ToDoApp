package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/efrem/windsurf/internal/todo"
)

// main is the entrypoint for the Todo API HTTP server.
// It configures the listen port and base URL, builds the router,
// and starts the HTTP server on port 8000.
func main() {
	port := ":8000"
	baseURL := "http://localhost:8000"

	r := todo.NewRouter(baseURL)

	fmt.Printf("🚀 HATEOAS Todo API server starting on %s\n", port)
	fmt.Printf("📖 Try: curl %s\n", baseURL)
	fmt.Printf("📝 Try: curl %s/todos\n", baseURL)

	log.Fatal(http.ListenAndServe(port, r))
}
