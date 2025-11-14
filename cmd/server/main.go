package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/efrem/windsurf/internal/todo"
)

func main() {
	port := ":8000"
	baseURL := "http://localhost:8000"

	r := todo.NewRouter(baseURL)

	fmt.Printf("🚀 HATEOAS Todo API server starting on %s\n", port)
	fmt.Printf("📖 Try: curl %s\n", baseURL)
	fmt.Printf("📝 Try: curl %s/todos\n", baseURL)

	log.Fatal(http.ListenAndServe(port, r))
}
