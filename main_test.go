package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestTaskStoreCreate tests creating tasks in the store
func TestTaskStoreCreate(t *testing.T) {
	store := NewTaskStore()

	task := store.Create("Test Task", "Test Description")

	if task.ID == "" {
		t.Error("expected task to have an ID")
	}
	if task.Title != "Test Task" {
		t.Errorf("expected title 'Test Task', got '%s'", task.Title)
	}
	if task.Description != "Test Description" {
		t.Errorf("expected description 'Test Description', got '%s'", task.Description)
	}
	if task.Completed {
		t.Error("expected new task to not be completed")
	}
	if task.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

// TestTaskStoreGet tests retrieving a task by ID
func TestTaskStoreGet(t *testing.T) {
	store := NewTaskStore()

	created := store.Create("Test Task", "Description")

	// Test getting existing task
	task, ok := store.Get(created.ID)
	if !ok {
		t.Error("expected to find task")
	}
	if task.ID != created.ID {
		t.Errorf("expected ID '%s', got '%s'", created.ID, task.ID)
	}

	// Test getting non-existent task
	_, ok = store.Get("non-existent")
	if ok {
		t.Error("expected not to find non-existent task")
	}
}

// TestTaskStoreGetAll tests retrieving all tasks
func TestTaskStoreGetAll(t *testing.T) {
	store := NewTaskStore()

	// Empty store
	tasks := store.GetAll()
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}

	// Add some tasks
	store.Create("Task 1", "Desc 1")
	store.Create("Task 2", "Desc 2")

	tasks = store.GetAll()
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

// TestTaskStoreDelete tests deleting a task
func TestTaskStoreDelete(t *testing.T) {
	store := NewTaskStore()

	task := store.Create("Test Task", "Description")

	// Delete existing task
	if !store.Delete(task.ID) {
		t.Error("expected delete to return true for existing task")
	}

	// Verify it's deleted
	_, ok := store.Get(task.ID)
	if ok {
		t.Error("expected task to be deleted")
	}

	// Delete non-existent task
	if store.Delete("non-existent") {
		t.Error("expected delete to return false for non-existent task")
	}
}

// TestServerListTasks tests GET /tasks endpoint
func TestServerListTasks(t *testing.T) {
	store := NewTaskStore()
	server := NewServer(store)

	// Test empty list
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}

	var tasks []Task
	if err := json.Unmarshal(w.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}

	// Add a task and test again
	store.Create("Test Task", "Description")

	req = httptest.NewRequest(http.MethodGet, "/tasks", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if err := json.Unmarshal(w.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
}

// TestServerCreateTask tests POST /tasks endpoint
func TestServerCreateTask(t *testing.T) {
	store := NewTaskStore()
	server := NewServer(store)

	// Valid request
	body := bytes.NewBufferString(`{"title": "New Task", "description": "New Description"}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", body)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var task Task
	if err := json.Unmarshal(w.Body.Bytes(), &task); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if task.Title != "New Task" {
		t.Errorf("expected title 'New Task', got '%s'", task.Title)
	}
	if task.ID == "" {
		t.Error("expected task to have an ID")
	}
}

// TestServerCreateTaskInvalidJSON tests POST /tasks with invalid JSON
func TestServerCreateTaskInvalidJSON(t *testing.T) {
	store := NewTaskStore()
	server := NewServer(store)

	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", body)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestServerCreateTaskMissingTitle tests POST /tasks without title
func TestServerCreateTaskMissingTitle(t *testing.T) {
	store := NewTaskStore()
	server := NewServer(store)

	body := bytes.NewBufferString(`{"description": "No title"}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", body)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestServerGetTask tests GET /tasks/{id} endpoint
func TestServerGetTask(t *testing.T) {
	store := NewTaskStore()
	server := NewServer(store)

	task := store.Create("Test Task", "Description")

	// Get existing task
	req := httptest.NewRequest(http.MethodGet, "/tasks/"+task.ID, nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var retrieved Task
	if err := json.Unmarshal(w.Body.Bytes(), &retrieved); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if retrieved.ID != task.ID {
		t.Errorf("expected ID '%s', got '%s'", task.ID, retrieved.ID)
	}
}

// TestServerGetTaskNotFound tests GET /tasks/{id} for non-existent task
func TestServerGetTaskNotFound(t *testing.T) {
	store := NewTaskStore()
	server := NewServer(store)

	req := httptest.NewRequest(http.MethodGet, "/tasks/non-existent", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestServerDeleteTask tests DELETE /tasks/{id} endpoint
func TestServerDeleteTask(t *testing.T) {
	store := NewTaskStore()
	server := NewServer(store)

	task := store.Create("Test Task", "Description")

	// Delete existing task
	req := httptest.NewRequest(http.MethodDelete, "/tasks/"+task.ID, nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	// Verify it's deleted
	_, ok := store.Get(task.ID)
	if ok {
		t.Error("expected task to be deleted")
	}
}

// TestServerDeleteTaskNotFound tests DELETE /tasks/{id} for non-existent task
func TestServerDeleteTaskNotFound(t *testing.T) {
	store := NewTaskStore()
	server := NewServer(store)

	req := httptest.NewRequest(http.MethodDelete, "/tasks/non-existent", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestServerMethodNotAllowed tests unsupported methods
func TestServerMethodNotAllowed(t *testing.T) {
	store := NewTaskStore()
	server := NewServer(store)

	// PUT on /tasks should be 405
	req := httptest.NewRequest(http.MethodPut, "/tasks", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// TestServerNotFound tests unknown paths
func TestServerNotFound(t *testing.T) {
	store := NewTaskStore()
	server := NewServer(store)

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
