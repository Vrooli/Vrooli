package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleIngestDocument(t *testing.T) {
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

	upsertCount := 0
	mockQdrant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/collections/knowledge_chunks_v1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":{}}`))
			return
		case r.Method == "PUT" && r.URL.Path == "/collections/knowledge_chunks_v1/points":
			upsertCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer mockQdrant.Close()

	srv := &Server{
		config: &Config{
			QdrantURL: mockQdrant.URL,
			OllamaURL: mockOllama.URL,
		},
	}
	srv.setupServices()

	body := map[string]interface{}{
		"namespace":      "ecosystem-manager",
		"content":        "hello world this is a test document that should be chunked",
		"chunk_size":     10,
		"chunk_overlap":  2,
		"source":         "unit-test",
		"source_type":    "manual",
		"visibility":     "shared",
		"collection":     "knowledge_chunks_v1",
		"document_id":    "",
		"metadata":       map[string]interface{}{"k": "v"},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/knowledge/documents/ingest", bytes.NewReader(raw))
	rec := httptest.NewRecorder()

	srv.handleIngestDocument(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var decoded IngestDocumentResponse
	if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.DocumentID == "" {
		t.Fatalf("expected document_id")
	}
	if decoded.ChunkCount <= 0 {
		t.Fatalf("expected chunks > 0")
	}
	if len(decoded.RecordIDs) != decoded.ChunkCount {
		t.Fatalf("record_ids mismatch")
	}
	if upsertCount != decoded.ChunkCount {
		t.Fatalf("upserts=%d chunks=%d", upsertCount, decoded.ChunkCount)
	}
}

