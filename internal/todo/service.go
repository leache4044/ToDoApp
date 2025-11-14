package todo

// Service defines a high-level facade for working with Todo entities.
// It exposes operations for listing, retrieving, creating, updating,
// completing, and deleting todos without exposing storage details.
type Service interface {
	ListTodos() []*Todo
	GetTodo(id int) (*Todo, bool)
	// CreateTodo creates a new todo using the provided input.
	CreateTodo(input TodoInput) *Todo
	// UpdateTodo updates an existing todo identified by id.
	// The boolean indicates whether the todo was found.
	UpdateTodo(id int, input TodoInput) (*Todo, bool)
	// CompleteTodo marks the specified todo as completed.
	// The boolean indicates whether the todo was found.
	CompleteTodo(id int) (*Todo, bool)
	// DeleteTodo removes the todo with the given ID from the store.
	// It returns true if a todo was deleted, or false if none existed.
	DeleteTodo(id int) bool
}

// service is the concrete implementation of Service backed by a TodoStore.
type service struct {
	store *TodoStore
}

// NewService constructs a Service backed by the given TodoStore.
func NewService(store *TodoStore) Service {
	return &service{store: store}
}

// ListTodos returns all todos from the underlying store.
func (s *service) ListTodos() []*Todo {
	return s.store.GetAll()
}

// GetTodo returns a todo by ID from the underlying store.
// The boolean indicates whether a todo with the given ID exists.
func (s *service) GetTodo(id int) (*Todo, bool) {
	return s.store.GetByID(id)
}

// CreateTodo creates a new todo using the provided input.
func (s *service) CreateTodo(input TodoInput) *Todo {
	return s.store.Create(input)
}

// UpdateTodo updates an existing todo identified by id.
// The boolean indicates whether the todo was found.
func (s *service) UpdateTodo(id int, input TodoInput) (*Todo, bool) {
	return s.store.Update(id, input)
}

// CompleteTodo marks the specified todo as completed.
// The boolean indicates whether the todo was found.
func (s *service) CompleteTodo(id int) (*Todo, bool) {
	return s.store.Complete(id)
}

// DeleteTodo removes the todo with the given ID from the store.
// It returns true if a todo was deleted, or false if none existed.
func (s *service) DeleteTodo(id int) bool {
	return s.store.Delete(id)
}
