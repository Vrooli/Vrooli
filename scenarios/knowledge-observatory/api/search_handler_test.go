package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"knowledge-observatory/internal/ports"
	"knowledge-observatory/internal/services/search"
)

type fakeSearchEmbedder struct{}

func (fakeSearchEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return []float64{0.1, 0.2, 0.3}, nil
}

type fakeSearchVectorStore struct {
	lastFilter *ports.VectorFilter
}

func (f *fakeSearchVectorStore) EnsureCollection(ctx context.Context, collection string, vectorSize int) error {
	return nil
}
func (f *fakeSearchVectorStore) UpsertPoint(ctx context.Context, collection string, id string, vector []float64, payload map[string]interface{}) error {
	return nil
}
func (f *fakeSearchVectorStore) DeletePoint(ctx context.Context, collection string, id string) error {
	return nil
}
func (f *fakeSearchVectorStore) Search(ctx context.Context, collection string, vector []float64, limit int, threshold float64, filter *ports.VectorFilter) ([]ports.VectorSearchResult, error) {
	f.lastFilter = filter
	return []ports.VectorSearchResult{
		{ID: "abc", Score: 0.9, Payload: map[string]interface{}{"content": "hello", "namespace": "ns1"}},
	}, nil
}
func (f *fakeSearchVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	return []string{"knowledge_chunks_v1"}, nil
}
func (f *fakeSearchVectorStore) CountPoints(ctx context.Context, collection string) (int, error) {
	return 0, nil
}
func (f *fakeSearchVectorStore) SamplePoints(ctx context.Context, collection string, limit int) ([]ports.VectorPoint, error) {
	return []ports.VectorPoint{}, nil
}

func TestHandleSearch(t *testing.T) {
	vs := &fakeSearchVectorStore{}
	srv := &Server{
		searchService: &search.Service{
			VectorStore: vs,
			Embedder:    fakeSearchEmbedder{},
		},
	}

	t.Run("accepts valid request", func(t *testing.T) {
		raw, _ := json.Marshal(map[string]interface{}{"query": "test query", "limit": 10})
		req := httptest.NewRequest("POST", "/api/v1/knowledge/search", bytes.NewReader(raw))
		rec := httptest.NewRecorder()

		srv.handleSearch(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}

		var decoded SearchResponse
		if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if decoded.Query != "test query" {
			t.Fatalf("query=%q", decoded.Query)
		}
		if len(decoded.Results) != 1 {
			t.Fatalf("results=%d", len(decoded.Results))
		}
	})

	t.Run("rejects invalid ingested_after", func(t *testing.T) {
		raw, _ := json.Marshal(map[string]interface{}{"query": "test query", "ingested_after": "nope"})
		req := httptest.NewRequest("POST", "/api/v1/knowledge/search", bytes.NewReader(raw))
		rec := httptest.NewRecorder()

		srv.handleSearch(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("passes filters to vector store", func(t *testing.T) {
		now := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
		raw, _ := json.Marshal(map[string]interface{}{
			"query":          "test query",
			"namespaces":     []string{"ns1"},
			"visibility":     []string{"shared"},
			"tags":           []string{"tag-a"},
			"ingested_after": now,
		})
		req := httptest.NewRequest("POST", "/api/v1/knowledge/search", bytes.NewReader(raw))
		rec := httptest.NewRecorder()

		srv.handleSearch(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}

		if vs.lastFilter == nil {
			t.Fatalf("expected filter to be passed")
		}
		if len(vs.lastFilter.Namespaces) != 1 || vs.lastFilter.Namespaces[0] != "ns1" {
			t.Fatalf("namespaces=%v", vs.lastFilter.Namespaces)
		}
		if len(vs.lastFilter.Visibility) != 1 || vs.lastFilter.Visibility[0] != "shared" {
			t.Fatalf("visibility=%v", vs.lastFilter.Visibility)
		}
		if len(vs.lastFilter.Tags) != 1 || vs.lastFilter.Tags[0] != "tag-a" {
			t.Fatalf("tags=%v", vs.lastFilter.Tags)
		}
		if vs.lastFilter.IngestedAfterMS == nil {
			t.Fatalf("expected ingested_after_ms set")
		}
	})
}
