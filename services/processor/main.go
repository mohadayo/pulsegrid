package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Pipeline struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Source      string    `json:"source"`
	Destination string    `json:"destination"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	EventCount  int       `json:"event_count"`
}

type PipelineCreate struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type PipelineEvent struct {
	PipelineID string                 `json:"pipeline_id"`
	EventType  string                 `json:"event_type"`
	Payload    map[string]interface{} `json:"payload"`
}

type EventResponse struct {
	ID         string    `json:"id"`
	PipelineID string    `json:"pipeline_id"`
	EventType  string    `json:"event_type"`
	Status     string    `json:"status"`
	ProcessedAt time.Time `json:"processed_at"`
}

type Store struct {
	mu        sync.RWMutex
	pipelines map[string]*Pipeline
}

func NewStore() *Store {
	return &Store{pipelines: make(map[string]*Pipeline)}
}

var store = NewStore()
var logger *log.Logger

func init() {
	logger = log.New(os.Stdout, "[processor] ", log.LstdFlags)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"service":   "processor",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func createPipelineHandler(w http.ResponseWriter, r *http.Request) {
	var req PipelineCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Source == "" || req.Destination == "" {
		writeError(w, http.StatusBadRequest, "name, source, and destination are required")
		return
	}

	p := &Pipeline{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Source:      req.Source,
		Destination: req.Destination,
		Status:      "active",
		CreatedAt:   time.Now().UTC(),
		EventCount:  0,
	}

	store.mu.Lock()
	store.pipelines[p.ID] = p
	store.mu.Unlock()

	logger.Printf("Created pipeline: %s (%s)", p.Name, p.ID)
	writeJSON(w, http.StatusCreated, p)
}

func listPipelinesHandler(w http.ResponseWriter, r *http.Request) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	pipelines := make([]*Pipeline, 0, len(store.pipelines))
	for _, p := range store.pipelines {
		pipelines = append(pipelines, p)
	}

	logger.Printf("Listed %d pipelines", len(pipelines))
	writeJSON(w, http.StatusOK, pipelines)
}

func getPipelineHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	store.mu.RLock()
	p, ok := store.pipelines[id]
	store.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func sendEventHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	store.mu.Lock()
	p, ok := store.pipelines[id]
	if !ok {
		store.mu.Unlock()
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}

	var event PipelineEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		store.mu.Unlock()
		writeError(w, http.StatusBadRequest, "invalid event body")
		return
	}

	p.EventCount++
	store.mu.Unlock()

	resp := EventResponse{
		ID:          uuid.New().String(),
		PipelineID:  id,
		EventType:   event.EventType,
		Status:      "processed",
		ProcessedAt: time.Now().UTC(),
	}

	logger.Printf("Processed event %s for pipeline %s", resp.ID, id)
	writeJSON(w, http.StatusCreated, resp)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /pipelines", createPipelineHandler)
	mux.HandleFunc("GET /pipelines", listPipelinesHandler)
	mux.HandleFunc("GET /pipelines/{id}", getPipelineHandler)
	mux.HandleFunc("POST /pipelines/{id}/events", sendEventHandler)

	logger.Printf("Starting processor on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		logger.Fatalf("Server failed: %v", err)
	}
}
