package todo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	testBaseURL        = "http://localhost:8000"
	testExampleBaseURL = "http://example.com"
	contentTypeHeader  = "Content-Type"
	contentTypeJSON    = "application/json"
	todosPath          = "/todos"
	todosIDFormat      = "/todos/%d"
	deleteMsgFormat    = "failed to unmarshal created todo: %v"
)

func TestTodoStoreCreateAndGet(t *testing.T) {
	store := NewTodoStore()

	created := store.Create(TodoInput{Title: "Test", Description: "Desc"})
	if created.ID != 1 {
		t.Fatalf("expected first todo ID to be 1, got %d", created.ID)
	}
	if created.Completed {
		t.Fatalf("expected new todo to be not completed")
	}

	fetched, ok := store.GetByID(created.ID)
	if !ok {
		t.Fatalf("expected todo with ID %d to exist", created.ID)
	}
	if fetched.Title != "Test" || fetched.Description != "Desc" {
		t.Fatalf("unexpected fetched todo: %+v", fetched)
	}
}

func TestTodoStoreNegativePaths(t *testing.T) {
	store := NewTodoStore()

	if todo, ok := store.Update(999, TodoInput{Title: "X", Description: "Y"}); ok || todo != nil {
		t.Fatalf("expected Update on missing ID to return (nil, false), got (%+v, %v)", todo, ok)
	}

	if todo, ok := store.Complete(999); ok || todo != nil {
		t.Fatalf("expected Complete on missing ID to return (nil, false), got (%+v, %v)", todo, ok)
	}

	if deleted := store.Delete(999); deleted {
		t.Fatalf("expected Delete on missing ID to return false")
	}
}

func TestTodoStoreUpdateCompleteDelete(t *testing.T) {
	store := NewTodoStore()
	created := store.Create(TodoInput{Title: "Original", Description: "Original desc"})

	updated, ok := store.Update(created.ID, TodoInput{Title: "Updated", Description: "Updated desc"})
	if !ok {
		t.Fatalf("expected update to succeed")
	}
	if updated.Title != "Updated" || updated.Description != "Updated desc" {
		t.Fatalf("unexpected updated todo: %+v", updated)
	}

	completed, ok := store.Complete(created.ID)
	if !ok {
		t.Fatalf("expected complete to succeed")
	}
	if !completed.Completed {
		t.Fatalf("expected todo to be marked completed")
	}

	deleted := store.Delete(created.ID)
	if !deleted {
		t.Fatalf("expected delete to succeed")
	}
	if _, ok := store.GetByID(created.ID); !ok {
		// ok, should not exist anymore
	} else {
		t.Fatalf("expected todo to be removed after delete")
	}
}

func TestServiceDelegatesToStore(t *testing.T) {
	store := NewTodoStore()
	service := NewService(store)

	created := service.CreateTodo(TodoInput{Title: "Svc", Description: "Svc desc"})
	if created.ID == 0 {
		t.Fatalf("expected created todo to have non-zero ID")
	}

	list := service.ListTodos()
	if len(list) != 1 {
		t.Fatalf("expected 1 todo from ListTodos, got %d", len(list))
	}

	got, ok := service.GetTodo(created.ID)
	if !ok || got.ID != created.ID {
		t.Fatalf("expected to get todo with ID %d, got %+v, ok=%v", created.ID, got, ok)
	}

	updated, ok := service.UpdateTodo(created.ID, TodoInput{Title: "Svc2", Description: "Svc2 desc"})
	if !ok || updated.Title != "Svc2" {
		t.Fatalf("expected UpdateTodo to modify title, got %+v, ok=%v", updated, ok)
	}

	completed, ok := service.CompleteTodo(created.ID)
	if !ok || !completed.Completed {
		t.Fatalf("expected CompleteTodo to mark as completed, got %+v, ok=%v", completed, ok)
	}

	if !service.DeleteTodo(created.ID) {
		t.Fatalf("expected DeleteTodo to return true")
	}
	if _, ok := service.GetTodo(created.ID); ok {
		t.Fatalf("expected todo to be gone after DeleteTodo")
	}
}

func TestBuildTodoLinksIncludesCompleteWhenNotCompleted(t *testing.T) {
	todo := &Todo{ID: 42, Completed: false}
	baseURL := testExampleBaseURL

	links := buildTodoLinks(todo, baseURL)

	if links.Self == nil || links.Self.Href == "" {
		t.Fatalf("expected self link to be set")
	}
	if links.Complete == nil {
		t.Fatalf("expected complete link when todo is not completed")
	}
}

func TestBuildTodoLinksOmitsCompleteWhenCompleted(t *testing.T) {
	todo := &Todo{ID: 42, Completed: true}
	baseURL := testExampleBaseURL

	links := buildTodoLinks(todo, baseURL)

	if links.Complete != nil {
		t.Fatalf("expected no complete link when todo is already completed")
	}
}

func TestBuildCollectionLinksPagination(t *testing.T) {
	baseURL := testExampleBaseURL
	page := 2
	perPage := 10
	total := 35

	links := buildCollectionLinks(baseURL, page, perPage, total)

	if links.Self == nil || links.First == nil || links.Last == nil {
		t.Fatalf("expected self, first, and last links to be set")
	}
	if links.Next == nil {
		t.Fatalf("expected next link on non-final page")
	}
	if links.Prev == nil {
		t.Fatalf("expected prev link on page > 1")
	}
}

func TestGetRootHandler(t *testing.T) {
	r := NewRouter(testBaseURL)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var root APIRoot
	if err := json.Unmarshal(rec.Body.Bytes(), &root); err != nil {
		t.Fatalf("failed to unmarshal root response: %v", err)
	}
	if root.Message == "" {
		t.Fatalf("expected root message to be set")
	}
}

func TestCreateAndGetTodoHandlers(t *testing.T) {
	r := NewRouter(testBaseURL)
	body := `{"title":"FromHandler","description":"via HTTP"}`
	req := httptest.NewRequest(http.MethodPost, todosPath, strings.NewReader(body))
	req.Header.Set(contentTypeHeader, contentTypeJSON)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", rec.Code, rec.Body.String())
	}

	var created Todo
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to unmarshal created todo: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected created todo to have non-zero ID")
	}

	getPath := fmt.Sprintf(todosIDFormat, created.ID)
	getReq := httptest.NewRequest(http.MethodGet, getPath, nil)
	getRec := httptest.NewRecorder()

	r.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 from GET todo, got %d; body=%s", getRec.Code, getRec.Body.String())
	}
}

func TestGetTodosPaginationHandler(t *testing.T) {
	r := NewRouter(testBaseURL)
	req := httptest.NewRequest(http.MethodGet, todosPath+"?page=1&per_page=2", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var collection TodoCollection
	if err := json.Unmarshal(rec.Body.Bytes(), &collection); err != nil {
		t.Fatalf("failed to unmarshal todos collection: %v", err)
	}
	if collection.Meta.PerPage != 2 {
		t.Fatalf("expected per_page meta to be 2, got %d", collection.Meta.PerPage)
	}
}

func TestCreateTodoValidationError(t *testing.T) {
	r := NewRouter(testBaseURL)
	body := `{"title":""}`
	req := httptest.NewRequest(http.MethodPost, todosPath, strings.NewReader(body))
	req.Header.Set(contentTypeHeader, contentTypeJSON)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for validation error, got %d", rec.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error == "" {
		t.Fatalf("expected error message in validation error response")
	}
}

func TestGetTodoNotFound(t *testing.T) {
	r := NewRouter(testBaseURL)
	req := httptest.NewRequest(http.MethodGet, "/todos/9999", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for missing todo, got %d", rec.Code)
	}
}

func TestGetTodoBadID(t *testing.T) {
	r := NewRouter(testBaseURL)
	req := httptest.NewRequest(http.MethodGet, "/todos/not-an-int", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for bad ID, got %d", rec.Code)
	}
}

func TestUpdateTodoHandler(t *testing.T) {
	r := NewRouter(testBaseURL)

	// First create a todo
	createBody := `{"title":"Before","description":"before"}`
	createReq := httptest.NewRequest(http.MethodPost, todosPath, strings.NewReader(createBody))
	createReq.Header.Set(contentTypeHeader, contentTypeJSON)
	createRec := httptest.NewRecorder()

	r.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 from create, got %d", createRec.Code)
	}

	var created Todo
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf(deleteMsgFormat, err)
	}

	// Now update it
	updateBody := `{"title":"After","description":"after"}`
	updatePath := fmt.Sprintf(todosIDFormat, created.ID)
	updateReq := httptest.NewRequest(http.MethodPut, updatePath, strings.NewReader(updateBody))
	updateReq.Header.Set(contentTypeHeader, contentTypeJSON)
	updateRec := httptest.NewRecorder()

	r.ServeHTTP(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 from update, got %d; body=%s", updateRec.Code, updateRec.Body.String())
	}

	var updated Todo
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to unmarshal updated todo: %v", err)
	}
	if updated.Title != "After" {
		t.Fatalf("expected updated title 'After', got %q", updated.Title)
	}
}

func TestUpdateTodoHandlerErrors(t *testing.T) {
	r := NewRouter(testBaseURL)

	badIDReq := httptest.NewRequest(http.MethodPut, "/todos/bad-id", strings.NewReader(`{"title":"X","description":"Y"}`))
	badIDReq.Header.Set(contentTypeHeader, contentTypeJSON)
	badIDRec := httptest.NewRecorder()

	r.ServeHTTP(badIDRec, badIDReq)

	if badIDRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for bad ID, got %d", badIDRec.Code)
	}

	invalidJSONReq := httptest.NewRequest(http.MethodPut, "/todos/1", strings.NewReader("{invalid-json"))
	invalidJSONReq.Header.Set(contentTypeHeader, contentTypeJSON)
	invalidJSONRec := httptest.NewRecorder()

	r.ServeHTTP(invalidJSONRec, invalidJSONReq)

	if invalidJSONRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid JSON, got %d", invalidJSONRec.Code)
	}

	validationReq := httptest.NewRequest(http.MethodPut, "/todos/1", strings.NewReader(`{"title":""}`))
	validationReq.Header.Set(contentTypeHeader, contentTypeJSON)
	validationRec := httptest.NewRecorder()

	r.ServeHTTP(validationRec, validationReq)

	if validationRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for validation error, got %d", validationRec.Code)
	}

	notFoundReq := httptest.NewRequest(http.MethodPut, "/todos/9999", strings.NewReader(`{"title":"X","description":"Y"}`))
	notFoundReq.Header.Set(contentTypeHeader, contentTypeJSON)
	notFoundRec := httptest.NewRecorder()

	r.ServeHTTP(notFoundRec, notFoundReq)

	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for missing todo, got %d", notFoundRec.Code)
	}
}

func TestRouterUnknownRouteAndOptions(t *testing.T) {
	r := NewRouter(testBaseURL)

	// Unknown route should return 404
	reqNotFound := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	recNotFound := httptest.NewRecorder()

	r.ServeHTTP(recNotFound, reqNotFound)

	if recNotFound.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for unknown route, got %d", recNotFound.Code)
	}

	// OPTIONS request should return immediately with CORS headers set
	reqOptions := httptest.NewRequest(http.MethodOptions, todosPath, nil)
	recOptions := httptest.NewRecorder()

	r.ServeHTTP(recOptions, reqOptions)

	if got := recOptions.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin '*' for OPTIONS, got %q", got)
	}
}

func TestCompleteTodoHandler(t *testing.T) {
	r := NewRouter(testBaseURL)

	// Create a todo
	body := `{"title":"CompleteMe","description":"please"}`
	createReq := httptest.NewRequest(http.MethodPost, todosPath, strings.NewReader(body))
	createReq.Header.Set(contentTypeHeader, contentTypeJSON)
	createRec := httptest.NewRecorder()

	r.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 from create, got %d", createRec.Code)
	}

	var created Todo
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to unmarshal created todo: %v", err)
	}

	// Complete it
	completePath := fmt.Sprintf("/todos/%d/complete", created.ID)
	completeReq := httptest.NewRequest(http.MethodPatch, completePath, nil)
	completeRec := httptest.NewRecorder()

	r.ServeHTTP(completeRec, completeReq)

	if completeRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 from complete, got %d; body=%s", completeRec.Code, completeRec.Body.String())
	}

	var completed Todo
	if err := json.Unmarshal(completeRec.Body.Bytes(), &completed); err != nil {
		t.Fatalf("failed to unmarshal completed todo: %v", err)
	}
	if !completed.Completed {
		t.Fatalf("expected todo to be marked completed")
	}
}

func TestCompleteTodoHandlerErrors(t *testing.T) {
	r := NewRouter(testBaseURL)

	badIDReq := httptest.NewRequest(http.MethodPatch, "/todos/bad-id/complete", nil)
	badIDRec := httptest.NewRecorder()

	r.ServeHTTP(badIDRec, badIDReq)

	if badIDRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for bad ID, got %d", badIDRec.Code)
	}

	notFoundReq := httptest.NewRequest(http.MethodPatch, "/todos/9999/complete", nil)
	notFoundRec := httptest.NewRecorder()

	r.ServeHTTP(notFoundRec, notFoundReq)

	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for missing todo, got %d", notFoundRec.Code)
	}
}

func TestDeleteTodoHandler(t *testing.T) {
	r := NewRouter(testBaseURL)

	// Create a todo
	body := `{"title":"DeleteMe","description":"please"}`
	createReq := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()

	r.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 from create, got %d", createRec.Code)
	}

	var created Todo
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to unmarshal created todo: %v", err)
	}

	// Delete it
	deletePath := fmt.Sprintf(todosIDFormat, created.ID)
	deleteReq := httptest.NewRequest(http.MethodDelete, deletePath, nil)
	deleteRec := httptest.NewRecorder()

	r.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 from delete, got %d; body=%s", deleteRec.Code, deleteRec.Body.String())
	}

	// Ensure subsequent GET returns 404
	getReq := httptest.NewRequest(http.MethodGet, deletePath, nil)
	getRec := httptest.NewRecorder()

	r.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 when getting deleted todo, got %d", getRec.Code)
	}
}

func TestDeleteTodoHandlerErrors(t *testing.T) {
	r := NewRouter(testBaseURL)

	badIDReq := httptest.NewRequest(http.MethodDelete, "/todos/not-an-int", nil)
	badIDRec := httptest.NewRecorder()

	r.ServeHTTP(badIDRec, badIDReq)

	if badIDRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for bad ID, got %d", badIDRec.Code)
	}

	notFoundReq := httptest.NewRequest(http.MethodDelete, "/todos/9999", nil)
	notFoundRec := httptest.NewRecorder()

	r.ServeHTTP(notFoundRec, notFoundReq)

	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for missing todo, got %d", notFoundRec.Code)
	}
}
