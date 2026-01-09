package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"knowledge-observatory/internal/services/ingest"
)

func TestHandleIngestDocument(t *testing.T) {
	vs := &fakeIngestVectorStore{}
	srv := &Server{
		config: &Config{Port: "8080"},
		ingestService: &ingest.Service{
			VectorStore: vs,
			Embedder:    fakeIngestEmbedder{},
		},
	}

	body := map[string]interface{}{
		"namespace":     "ecosystem-manager",
		"content":       "hello world this is a test document that should be chunked",
		"chunk_size":    10,
		"chunk_overlap": 2,
		"source":        "unit-test",
		"source_type":   "manual",
		"visibility":    "shared",
		"collection":    "knowledge_chunks_v1",
		"document_id":   "",
		"metadata":      map[string]interface{}{"k": "v"},
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
	if vs.upserts != decoded.ChunkCount {
		t.Fatalf("upserts=%d chunks=%d", vs.upserts, decoded.ChunkCount)
	}
}
