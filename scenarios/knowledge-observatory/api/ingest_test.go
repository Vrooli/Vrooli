package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHandleUpsertRecordValidation(t *testing.T) {
	srv := &Server{
		config: &Config{Port: "8080"},
		db:     nil,
	}

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name:       "rejects missing namespace",
			body:       map[string]interface{}{"content": "hello"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rejects missing content",
			body:       map[string]interface{}{"namespace": "test-scenario"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rejects invalid visibility",
			body:       map[string]interface{}{"namespace": "test-scenario", "content": "hello", "visibility": "nope"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/v1/knowledge/records/upsert", bytes.NewReader(raw))
			rec := httptest.NewRecorder()

			srv.handleUpsertRecord(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestHandleUpsertRecordSuccess(t *testing.T) {
	mockOllama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"embedding":[0.1,0.2,0.3]}`))
	}))
	defer mockOllama.Close()

	mockQdrant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/collections/knowledge_chunks_v1":
			w.WriteHeader(http.StatusNotFound)
			return
		case r.Method == "PUT" && r.URL.Path == "/collections/knowledge_chunks_v1":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":true}`))
			return
		case r.Method == "PUT" && r.URL.Path == "/collections/knowledge_chunks_v1/points":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer mockQdrant.Close()

	os.Setenv("OLLAMA_URL", mockOllama.URL)
	os.Setenv("QDRANT_URL", mockQdrant.URL)
	defer func() {
		os.Unsetenv("OLLAMA_URL")
		os.Unsetenv("QDRANT_URL")
	}()

	srv := &Server{
		config: &Config{
			QdrantURL: mockQdrant.URL,
			OllamaURL: mockOllama.URL,
		},
		db:     nil,
	}
	srv.setupServices()

	reqBody := map[string]interface{}{
		"namespace": "ecosystem-manager",
		"content":   "hello world",
		"metadata": map[string]interface{}{
			"foo": "bar",
		},
	}
	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/knowledge/records/upsert", bytes.NewReader(raw))
	rec := httptest.NewRecorder()

	srv.handleUpsertRecord(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var decoded UpsertRecordResponse
	if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if decoded.RecordID == "" {
		t.Fatalf("expected record_id to be set")
	}
	if decoded.Collection != "knowledge_chunks_v1" {
		t.Fatalf("collection=%q", decoded.Collection)
	}
	if decoded.Namespace != "ecosystem-manager" {
		t.Fatalf("namespace=%q", decoded.Namespace)
	}
	if decoded.ContentHash == "" {
		t.Fatalf("expected content_hash to be set")
	}
	if !decoded.Upserted {
		t.Fatalf("expected upserted=true")
	}
}
