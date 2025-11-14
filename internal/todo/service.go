package todo

// NOTE: This file contains the Service facade over the TodoStore.

type Service interface {
	ListTodos() []*Todo
	GetTodo(id int) (*Todo, bool)
	CreateTodo(input TodoInput) *Todo
	UpdateTodo(id int, input TodoInput) (*Todo, bool)
	CompleteTodo(id int) (*Todo, bool)
	DeleteTodo(id int) bool
}

type service struct {
	store *TodoStore
}

func NewService(store *TodoStore) Service {
	return &service{store: store}
}

func (s *service) ListTodos() []*Todo {
	return s.store.GetAll()
}

func (s *service) GetTodo(id int) (*Todo, bool) {
	return s.store.GetByID(id)
}

func (s *service) CreateTodo(input TodoInput) *Todo {
	return s.store.Create(input)
}

func (s *service) UpdateTodo(id int, input TodoInput) (*Todo, bool) {
	return s.store.Update(id, input)
}

func (s *service) CompleteTodo(id int) (*Todo, bool) {
	return s.store.Complete(id)
}

func (s *service) DeleteTodo(id int) bool {
	return s.store.Delete(id)
}
