package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Task represents a todo-like task
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
}

// TaskStore provides thread-safe in-memory storage for tasks
type TaskStore struct {
	mu     sync.RWMutex
	tasks  map[string]Task
	nextID int
}

// NewTaskStore creates a new TaskStore
func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks:  make(map[string]Task),
		nextID: 1,
	}
}

// Create adds a new task to the store and returns it with a generated ID
func (s *TaskStore) Create(title, description string) Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("task-%d", s.nextID)
	s.nextID++

	task := Task{
		ID:          id,
		Title:       title,
		Description: description,
		Completed:   false,
		CreatedAt:   time.Now(),
	}
	s.tasks[id] = task
	return task
}

// Get retrieves a task by ID, returns false if not found
func (s *TaskStore) Get(id string) (Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[id]
	return task, ok
}

// GetAll returns all tasks as a slice
func (s *TaskStore) GetAll() []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// Delete removes a task by ID, returns false if not found
func (s *TaskStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return false
	}
	delete(s.tasks, id)
	return true
}

// Server handles HTTP requests for the task API
type Server struct {
	store *TaskStore
}

// NewServer creates a new Server with the given TaskStore
func NewServer(store *TaskStore) *Server {
	return &Server{store: store}
}

// ServeHTTP implements the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Route requests based on path and method
	path := r.URL.Path

	if path == "/tasks" || path == "/tasks/" {
		switch r.Method {
		case http.MethodGet:
			s.listTasks(w, r)
		case http.MethodPost:
			s.createTask(w, r)
		default:
			s.methodNotAllowed(w)
		}
		return
	}

	if strings.HasPrefix(path, "/tasks/") {
		id := strings.TrimPrefix(path, "/tasks/")
		if id == "" {
			s.notFound(w)
			return
		}
		switch r.Method {
		case http.MethodGet:
			s.getTask(w, r, id)
		case http.MethodDelete:
			s.deleteTask(w, r, id)
		default:
			s.methodNotAllowed(w)
		}
		return
	}

	s.notFound(w)
}

// listTasks handles GET /tasks
func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks := s.store.GetAll()
	s.jsonResponse(w, http.StatusOK, tasks)
}

// createTask handles POST /tasks
func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if input.Title == "" {
		s.jsonError(w, http.StatusBadRequest, "title is required")
		return
	}

	task := s.store.Create(input.Title, input.Description)
	s.jsonResponse(w, http.StatusCreated, task)
}

// getTask handles GET /tasks/{id}
func (s *Server) getTask(w http.ResponseWriter, r *http.Request, id string) {
	task, ok := s.store.Get(id)
	if !ok {
		s.jsonError(w, http.StatusNotFound, "task not found")
		return
	}
	s.jsonResponse(w, http.StatusOK, task)
}

// deleteTask handles DELETE /tasks/{id}
func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request, id string) {
	if !s.store.Delete(id) {
		s.jsonError(w, http.StatusNotFound, "task not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// jsonResponse writes a JSON response with the given status code
func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("error encoding JSON response: %v", err)
	}
}

// jsonError writes a JSON error response
func (s *Server) jsonError(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}

// notFound returns a 404 response
func (s *Server) notFound(w http.ResponseWriter) {
	s.jsonError(w, http.StatusNotFound, "not found")
}

// methodNotAllowed returns a 405 response
func (s *Server) methodNotAllowed(w http.ResponseWriter) {
	s.jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func main() {
	store := NewTaskStore()
	server := NewServer(store)

	addr := ":8080"
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, server); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
