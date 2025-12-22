package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"knowledge-observatory/internal/ports"
	"knowledge-observatory/internal/services/ingest"
)

type fakeIngestEmbedder struct{}

func (fakeIngestEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return []float64{0.1, 0.2, 0.3}, nil
}

type fakeIngestVectorStore struct {
	upserts int
}

func (f *fakeIngestVectorStore) EnsureCollection(ctx context.Context, collection string, vectorSize int) error {
	return nil
}
func (f *fakeIngestVectorStore) UpsertPoint(ctx context.Context, collection string, id string, vector []float64, payload map[string]interface{}) error {
	f.upserts++
	return nil
}
func (f *fakeIngestVectorStore) DeletePoint(ctx context.Context, collection string, id string) error {
	return nil
}
func (f *fakeIngestVectorStore) Search(ctx context.Context, collection string, vector []float64, limit int, threshold float64, filter *ports.VectorFilter) ([]ports.VectorSearchResult, error) {
	return nil, nil
}
func (f *fakeIngestVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	return []string{}, nil
}
func (f *fakeIngestVectorStore) CountPoints(ctx context.Context, collection string) (int, error) {
	return 0, nil
}
func (f *fakeIngestVectorStore) SamplePoints(ctx context.Context, collection string, limit int) ([]ports.VectorPoint, error) {
	return []ports.VectorPoint{}, nil
}

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
	vs := &fakeIngestVectorStore{}
	srv := &Server{
		config: &Config{Port: "8080"},
		ingestService: &ingest.Service{
			VectorStore: vs,
			Embedder:    fakeIngestEmbedder{},
			Metadata:    nil,
		},
	}

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
