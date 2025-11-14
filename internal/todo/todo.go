package todo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Todo struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	Links       Links     `json:"_links"`
}

type TodoInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Links struct {
	Self     *Link `json:"self,omitempty"`
	Update   *Link `json:"update,omitempty"`
	Delete   *Link `json:"delete,omitempty"`
	Complete *Link `json:"complete,omitempty"`
	Todos    *Link `json:"todos,omitempty"`
}

type Link struct {
	Href   string `json:"href"`
	Method string `json:"method,omitempty"`
}

type TodoCollection struct {
	Todos []Todo          `json:"todos"`
	Meta  CollectionMeta  `json:"_meta"`
	Links CollectionLinks `json:"_links"`
}

type CollectionMeta struct {
	Total      int `json:"total"`
	Count      int `json:"count"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

type CollectionLinks struct {
	Self   *Link `json:"self,omitempty"`
	First  *Link `json:"first,omitempty"`
	Last   *Link `json:"last,omitempty"`
	Next   *Link `json:"next,omitempty"`
	Prev   *Link `json:"prev,omitempty"`
	Create *Link `json:"create,omitempty"`
}

type APIRoot struct {
	Message string       `json:"message"`
	Links   APIRootLinks `json:"_links"`
}

type APIRootLinks struct {
	Self  *Link `json:"self"`
	Todos *Link `json:"todos"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Links   Links  `json:"_links"`
}

type TodoStore struct {
	todos  map[int]*Todo
	nextID int
	mu     sync.RWMutex
}

func NewTodoStore() *TodoStore {
	return &TodoStore{
		todos:  make(map[int]*Todo),
		nextID: 1,
	}
}

// GetAll returns all todos currently stored in memory.
func (s *TodoStore) GetAll() []*Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	todos := make([]*Todo, 0, len(s.todos))
	for _, todo := range s.todos {
		todos = append(todos, todo)
	}
	return todos
}

// GetByID returns a todo by its ID.
// The boolean indicates whether a todo with that ID exists.
func (s *TodoStore) GetByID(id int) (*Todo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	todo, exists := s.todos[id]
	return todo, exists
}

// Create adds a new todo to the store using the provided input.
func (s *TodoStore) Create(input TodoInput) *Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo := &Todo{
		ID:          s.nextID,
		Title:       input.Title,
		Description: input.Description,
		Completed:   false,
		CreatedAt:   time.Now(),
	}

	s.todos[s.nextID] = todo
	s.nextID++

	return todo
}

// Update modifies an existing todo identified by id.
// The boolean indicates whether the todo was found.
func (s *TodoStore) Update(id int, input TodoInput) (*Todo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo, exists := s.todos[id]
	if !exists {
		return nil, false
	}

	todo.Title = input.Title
	todo.Description = input.Description

	return todo, true
}

// Complete marks the todo with the given ID as completed.
// The boolean indicates whether the todo was found.
func (s *TodoStore) Complete(id int) (*Todo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo, exists := s.todos[id]
	if !exists {
		return nil, false
	}

	todo.Completed = true
	return todo, true
}

// Delete removes the todo with the given ID from the store.
// It returns true if a todo was deleted, or false if it did not exist.
func (s *TodoStore) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.todos[id]
	if !exists {
		return false
	}

	delete(s.todos, id)
	return true
}

// buildTodoLinks constructs the HATEOAS links for a single todo resource.
func buildTodoLinks(todo *Todo, baseURL string) Links {
	links := Links{
		Self: &Link{
			Href:   fmt.Sprintf("%s/todos/%d", baseURL, todo.ID),
			Method: "GET",
		},
		Update: &Link{
			Href:   fmt.Sprintf("%s/todos/%d", baseURL, todo.ID),
			Method: "PUT",
		},
		Delete: &Link{
			Href:   fmt.Sprintf("%s/todos/%d", baseURL, todo.ID),
			Method: "DELETE",
		},
		Todos: &Link{
			Href:   fmt.Sprintf("%s/todos", baseURL),
			Method: "GET",
		},
	}

	if !todo.Completed {
		links.Complete = &Link{
			Href:   fmt.Sprintf("%s/todos/%d/complete", baseURL, todo.ID),
			Method: "PATCH",
		}
	}

	return links
}

// buildCollectionLinks constructs HATEOAS links for a paginated todos collection.
func buildCollectionLinks(baseURL string, page, perPage, total int) CollectionLinks {
	totalPages := 1
	if total > 0 {
		totalPages = (total + perPage - 1) / perPage
	}

	links := CollectionLinks{
		Self: &Link{
			Href: fmt.Sprintf("%s/todos?page=%d&per_page=%d", baseURL, page, perPage),
		},
		First: &Link{
			Href: fmt.Sprintf("%s/todos?page=1&per_page=%d", baseURL, perPage),
		},
		Create: &Link{
			Href:   fmt.Sprintf("%s/todos", baseURL),
			Method: "POST",
		},
		Last: nil,
		Next: nil,
		Prev: nil,
	}

	if totalPages > 1 {
		links.Last = &Link{
			Href: fmt.Sprintf("%s/todos?page=%d&per_page=%d", baseURL, totalPages, perPage),
		}
	}

	if page < totalPages {
		links.Next = &Link{
			Href: fmt.Sprintf("%s/todos?page=%d&per_page=%d", baseURL, page+1, perPage),
		}
	}

	if page > 1 {
		links.Prev = &Link{
			Href: fmt.Sprintf("%s/todos?page=%d&per_page=%d", baseURL, page-1, perPage),
		}
	}

	return links
}

// buildErrorLinks constructs navigation links included in error responses.
func buildErrorLinks(baseURL string) Links {
	return Links{
		Todos: &Link{
			Href:   fmt.Sprintf("%s/todos", baseURL),
			Method: "GET",
		},
	}
}

// TodoAPI provides HTTP handlers for the Todo REST API.
type TodoAPI struct {
	service Service
	baseURL string
}

// NewTodoAPI constructs a new TodoAPI using the provided base URL and Service facade.
func NewTodoAPI(baseURL string, service Service) *TodoAPI {
	return &TodoAPI{
		service: service,
		baseURL: baseURL,
	}
}

// GetRoot handles GET / and returns the API root document with navigation links.
func (api *TodoAPI) GetRoot(w http.ResponseWriter, r *http.Request) {
	root := APIRoot{
		Message: "Welcome to the HATEOAS Todo API",
		Links: APIRootLinks{
			Self: &Link{
				Href: api.baseURL,
			},
			Todos: &Link{
				Href:   fmt.Sprintf("%s/todos", api.baseURL),
				Method: "GET",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(root)
}

// GetTodos handles GET /todos and returns a paginated list of todos.
func (api *TodoAPI) GetTodos(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	perPageStr := r.URL.Query().Get("per_page")

	page := 1
	perPage := 10

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	allTodos := api.service.ListTodos()
	total := len(allTodos)

	start := (page - 1) * perPage
	end := start + perPage

	if start >= total {
		start = total
	}
	if end > total {
		end = total
	}

	var paginatedTodos []Todo
	if start < total {
		for i := start; i < end; i++ {
			todo := *allTodos[i]
			todo.Links = buildTodoLinks(&todo, api.baseURL)
			paginatedTodos = append(paginatedTodos, todo)
		}
	}

	totalPages := (total + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}

	collection := TodoCollection{
		Todos: paginatedTodos,
		Meta: CollectionMeta{
			Total:      total,
			Count:      len(paginatedTodos),
			Page:       page,
			PerPage:    perPage,
			TotalPages: totalPages,
		},
		Links: buildCollectionLinks(api.baseURL, page, perPage, total),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collection)
}

// GetTodo handles GET /todos/{id} and returns a single todo by ID.
func (api *TodoAPI) GetTodo(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid todo ID", "The provided ID must be a valid integer")
		return
	}

	todo, exists := api.service.GetTodo(id)
	if !exists {
		api.sendError(w, http.StatusNotFound, "Todo not found", fmt.Sprintf("Todo with ID %d does not exist", id))
		return
	}

	todoResponse := *todo
	todoResponse.Links = buildTodoLinks(todo, api.baseURL)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todoResponse)
}

// CreateTodo handles POST /todos and creates a new todo from the request body.
func (api *TodoAPI) CreateTodo(w http.ResponseWriter, r *http.Request) {
	var input TodoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid JSON", "Request body must be valid JSON")
		return
	}

	if input.Title == "" {
		api.sendError(w, http.StatusBadRequest, "Validation error", "Title is required")
		return
	}

	todo := api.service.CreateTodo(input)
	todo.Links = buildTodoLinks(todo, api.baseURL)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", fmt.Sprintf("%s/todos/%d", api.baseURL, todo.ID))
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(todo)
}

// UpdateTodo handles PUT /todos/{id} and updates an existing todo.
func (api *TodoAPI) UpdateTodo(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid todo ID", "The provided ID must be a valid integer")
		return
	}

	var input TodoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid JSON", "Request body must be valid JSON")
		return
	}

	if input.Title == "" {
		api.sendError(w, http.StatusBadRequest, "Validation error", "Title is required")
		return
	}

	todo, exists := api.service.UpdateTodo(id, input)
	if !exists {
		api.sendError(w, http.StatusNotFound, "Todo not found", fmt.Sprintf("Todo with ID %d does not exist", id))
		return
	}

	todo.Links = buildTodoLinks(todo, api.baseURL)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

// CompleteTodo handles PATCH /todos/{id}/complete and marks a todo as completed.
func (api *TodoAPI) CompleteTodo(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid todo ID", "The provided ID must be a valid integer")
		return
	}

	todo, exists := api.service.CompleteTodo(id)
	if !exists {
		api.sendError(w, http.StatusNotFound, "Todo not found", fmt.Sprintf("Todo with ID %d does not exist", id))
		return
	}

	todo.Links = buildTodoLinks(todo, api.baseURL)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

// DeleteTodo handles DELETE /todos/{id} and removes the todo.
func (api *TodoAPI) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid todo ID", "The provided ID must be a valid integer")
		return
	}

	exists := api.service.DeleteTodo(id)
	if !exists {
		api.sendError(w, http.StatusNotFound, "Todo not found", fmt.Sprintf("Todo with ID %d does not exist", id))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// sendError writes a JSON error response with the given status code and message.
func (api *TodoAPI) sendError(w http.ResponseWriter, statusCode int, error, message string) {
	errorResponse := ErrorResponse{
		Error:   error,
		Message: message,
		Links:   buildErrorLinks(api.baseURL),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse)
}

// NewRouter constructs and configures the chi router for the Todo API.
// It wires the in-memory store, Service facade, middleware, routes,
// and seeds the store with some sample data.
func NewRouter(baseURL string) http.Handler {
	store := NewTodoStore()
	service := NewService(store)
	api := NewTodoAPI(baseURL, service)

	service.CreateTodo(TodoInput{Title: "Learn Go", Description: "Master the Go programming language"})
	service.CreateTodo(TodoInput{Title: "Build REST API", Description: "Create a HATEOAS-compliant REST API"})
	service.CreateTodo(TodoInput{Title: "Write Tests", Description: "Add comprehensive test coverage"})

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

			if r.Method == "OPTIONS" {
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	r.Get("/", api.GetRoot)
	r.Route("/todos", func(r chi.Router) {
		r.Get("/", api.GetTodos)
		r.Post("/", api.CreateTodo)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", api.GetTodo)
			r.Put("/", api.UpdateTodo)
			r.Delete("/", api.DeleteTodo)
			r.Patch("/complete", api.CompleteTodo)
		})
	})

	return r
}
