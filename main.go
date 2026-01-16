package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Task represents a todo item
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
}

// TaskStore provides thread-safe in-memory storage for tasks
type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]Task
}

// NewTaskStore creates a new TaskStore
func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks: make(map[string]Task),
	}
}

// GetAll returns all tasks
func (s *TaskStore) GetAll() []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// Get returns a task by ID
func (s *TaskStore) Get(id string) (Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[id]
	return task, exists
}

// Create adds a new task
func (s *TaskStore) Create(task Task) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[task.ID] = task
}

// Delete removes a task by ID
func (s *TaskStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[id]; !exists {
		return false
	}
	delete(s.tasks, id)
	return true
}

// Server handles HTTP requests for the task API
type Server struct {
	store *TaskStore
}

// NewServer creates a new Server with the given store
func NewServer(store *TaskStore) *Server {
	return &Server{store: store}
}

// handleTasks handles requests to /tasks
func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		s.listTasks(w, r)
	case http.MethodPost:
		s.createTask(w, r)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// listTasks returns all tasks as JSON
func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks := s.store.GetAll()
	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
	}
}

// createTask creates a new task from JSON body
func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		http.Error(w, `{"error":"title is required"}`, http.StatusBadRequest)
		return
	}

	task.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	task.CreatedAt = time.Now()

	s.store.Create(task)

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
	}
}

// handleTaskByID handles requests to /tasks/{id}
func (s *Server) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract ID from path (e.g., /tasks/abc123 -> abc123)
	id := r.URL.Path[len("/tasks/"):]
	if id == "" {
		http.Error(w, `{"error":"task ID required"}`, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getTask(w, r, id)
	case http.MethodDelete:
		s.deleteTask(w, r, id)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// getTask returns a single task by ID
func (s *Server) getTask(w http.ResponseWriter, r *http.Request, id string) {
	task, exists := s.store.Get(id)
	if !exists {
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
	}
}

// deleteTask deletes a task by ID
func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request, id string) {
	if !s.store.Delete(id) {
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SetupRoutes configures the HTTP routes
func (s *Server) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/tasks", s.handleTasks)
	mux.HandleFunc("/tasks/", s.handleTaskByID)
}

func main() {
	store := NewTaskStore()
	server := NewServer(store)

	mux := http.NewServeMux()
	server.SetupRoutes(mux)

	addr := ":8080"
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
