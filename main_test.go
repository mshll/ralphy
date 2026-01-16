package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTaskStore(t *testing.T) {
	t.Run("NewTaskStore creates empty store", func(t *testing.T) {
		store := NewTaskStore()
		tasks := store.GetAll()
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})

	t.Run("Create and Get task", func(t *testing.T) {
		store := NewTaskStore()
		task := Task{ID: "test-1", Title: "Test Task"}
		store.Create(task)

		got, exists := store.Get("test-1")
		if !exists {
			t.Fatal("expected task to exist")
		}
		if got.Title != "Test Task" {
			t.Errorf("expected title 'Test Task', got '%s'", got.Title)
		}
	})

	t.Run("Get non-existent task returns false", func(t *testing.T) {
		store := NewTaskStore()
		_, exists := store.Get("non-existent")
		if exists {
			t.Error("expected task to not exist")
		}
	})

	t.Run("Delete existing task", func(t *testing.T) {
		store := NewTaskStore()
		store.Create(Task{ID: "test-1", Title: "Test"})

		deleted := store.Delete("test-1")
		if !deleted {
			t.Error("expected delete to return true")
		}

		_, exists := store.Get("test-1")
		if exists {
			t.Error("expected task to be deleted")
		}
	})

	t.Run("Delete non-existent task returns false", func(t *testing.T) {
		store := NewTaskStore()
		deleted := store.Delete("non-existent")
		if deleted {
			t.Error("expected delete to return false for non-existent task")
		}
	})

	t.Run("GetAll returns all tasks", func(t *testing.T) {
		store := NewTaskStore()
		store.Create(Task{ID: "1", Title: "Task 1"})
		store.Create(Task{ID: "2", Title: "Task 2"})

		tasks := store.GetAll()
		if len(tasks) != 2 {
			t.Errorf("expected 2 tasks, got %d", len(tasks))
		}
	})
}

func TestServerBasic(t *testing.T) {
	store := NewTaskStore()
	server := NewServer(store)
	mux := http.NewServeMux()
	server.SetupRoutes(mux)

	t.Run("GET /tasks returns empty array initially", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
		}

		var tasks []Task
		if err := json.NewDecoder(rec.Body).Decode(&tasks); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})

	t.Run("Server responds to requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})
}

func TestServerCreateTask(t *testing.T) {
	store := NewTaskStore()
	server := NewServer(store)
	mux := http.NewServeMux()
	server.SetupRoutes(mux)

	t.Run("POST /tasks creates a new task", func(t *testing.T) {
		body := bytes.NewBufferString(`{"title":"New Task","description":"Test description"}`)
		req := httptest.NewRequest(http.MethodPost, "/tasks", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", rec.Code)
		}

		var task Task
		if err := json.NewDecoder(rec.Body).Decode(&task); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if task.Title != "New Task" {
			t.Errorf("expected title 'New Task', got '%s'", task.Title)
		}
		if task.ID == "" {
			t.Error("expected task to have an ID")
		}
	})

	t.Run("POST /tasks with invalid JSON returns 400", func(t *testing.T) {
		body := bytes.NewBufferString(`{invalid json}`)
		req := httptest.NewRequest(http.MethodPost, "/tasks", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("POST /tasks without title returns 400", func(t *testing.T) {
		body := bytes.NewBufferString(`{"description":"No title"}`)
		req := httptest.NewRequest(http.MethodPost, "/tasks", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestServerGetTaskByID(t *testing.T) {
	store := NewTaskStore()
	server := NewServer(store)
	mux := http.NewServeMux()
	server.SetupRoutes(mux)

	// Create a task first
	task := Task{ID: "test-123", Title: "Test Task", Description: "Test"}
	store.Create(task)

	t.Run("GET /tasks/{id} returns task", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks/test-123", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var got Task
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if got.ID != "test-123" {
			t.Errorf("expected ID 'test-123', got '%s'", got.ID)
		}
	})

	t.Run("GET /tasks/{id} returns 404 for non-existent task", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks/non-existent", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})
}

func TestServerDeleteTask(t *testing.T) {
	store := NewTaskStore()
	server := NewServer(store)
	mux := http.NewServeMux()
	server.SetupRoutes(mux)

	// Create a task first
	task := Task{ID: "delete-me", Title: "To Delete"}
	store.Create(task)

	t.Run("DELETE /tasks/{id} returns 204", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/tasks/delete-me", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", rec.Code)
		}

		// Verify task is deleted
		_, exists := store.Get("delete-me")
		if exists {
			t.Error("expected task to be deleted")
		}
	})

	t.Run("DELETE /tasks/{id} returns 404 for non-existent task", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/tasks/non-existent", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})
}

func TestMethodNotAllowed(t *testing.T) {
	store := NewTaskStore()
	server := NewServer(store)
	mux := http.NewServeMux()
	server.SetupRoutes(mux)

	t.Run("PUT /tasks returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/tasks", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", rec.Code)
		}
	})

	t.Run("PATCH /tasks/{id} returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/tasks/123", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", rec.Code)
		}
	})
}
