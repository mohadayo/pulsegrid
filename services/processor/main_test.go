package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupMux() *http.ServeMux {
	store = NewStore()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /pipelines", createPipelineHandler)
	mux.HandleFunc("GET /pipelines", listPipelinesHandler)
	mux.HandleFunc("GET /pipelines/{id}", getPipelineHandler)
	mux.HandleFunc("POST /pipelines/{id}/events", sendEventHandler)
	return mux
}

func TestHealthEndpoint(t *testing.T) {
	mux := setupMux()
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "healthy" {
		t.Fatalf("expected healthy, got %v", body["status"])
	}
	if body["service"] != "processor" {
		t.Fatalf("expected processor, got %v", body["service"])
	}
}

func TestCreatePipeline(t *testing.T) {
	mux := setupMux()
	payload := `{"name":"test-pipe","source":"kafka","destination":"s3"}`
	req := httptest.NewRequest("POST", "/pipelines", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var p Pipeline
	json.NewDecoder(w.Body).Decode(&p)
	if p.Name != "test-pipe" {
		t.Fatalf("expected test-pipe, got %s", p.Name)
	}
	if p.Status != "active" {
		t.Fatalf("expected active, got %s", p.Status)
	}
}

func TestCreatePipelineMissingFields(t *testing.T) {
	mux := setupMux()
	payload := `{"name":"test-pipe"}`
	req := httptest.NewRequest("POST", "/pipelines", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListPipelines(t *testing.T) {
	mux := setupMux()

	payload := `{"name":"p1","source":"kafka","destination":"s3"}`
	req := httptest.NewRequest("POST", "/pipelines", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	req = httptest.NewRequest("GET", "/pipelines", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var pipelines []Pipeline
	json.NewDecoder(w.Body).Decode(&pipelines)
	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}
}

func TestGetPipeline(t *testing.T) {
	mux := setupMux()

	payload := `{"name":"p1","source":"kafka","destination":"s3"}`
	req := httptest.NewRequest("POST", "/pipelines", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var created Pipeline
	json.NewDecoder(w.Body).Decode(&created)

	req = httptest.NewRequest("GET", "/pipelines/"+created.ID, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetPipelineNotFound(t *testing.T) {
	mux := setupMux()
	req := httptest.NewRequest("GET", "/pipelines/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSendEvent(t *testing.T) {
	mux := setupMux()

	payload := `{"name":"p1","source":"kafka","destination":"s3"}`
	req := httptest.NewRequest("POST", "/pipelines", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var created Pipeline
	json.NewDecoder(w.Body).Decode(&created)

	eventPayload := `{"pipeline_id":"` + created.ID + `","event_type":"data_ingested","payload":{"rows":100}}`
	req = httptest.NewRequest("POST", "/pipelines/"+created.ID+"/events", bytes.NewBufferString(eventPayload))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var resp EventResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "processed" {
		t.Fatalf("expected processed, got %s", resp.Status)
	}
}

func TestSendEventPipelineNotFound(t *testing.T) {
	mux := setupMux()
	eventPayload := `{"pipeline_id":"fake","event_type":"test","payload":{}}`
	req := httptest.NewRequest("POST", "/pipelines/fake/events", bytes.NewBufferString(eventPayload))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
