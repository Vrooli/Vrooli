package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"knowledge-observatory/internal/ports"
	"knowledge-observatory/internal/services/graph"
)

type fakeEmbedder struct{}

func (f *fakeEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return []float64{0.1, 0.2, 0.3}, nil
}

type fakeGraphVectorStore struct {
	lastFilter *ports.VectorFilter
}

func (f *fakeGraphVectorStore) EnsureCollection(ctx context.Context, collection string, vectorSize int) error {
	return nil
}
func (f *fakeGraphVectorStore) UpsertPoint(ctx context.Context, collection string, id string, vector []float64, payload map[string]interface{}) error {
	return nil
}
func (f *fakeGraphVectorStore) DeletePoint(ctx context.Context, collection string, id string) error {
	return nil
}
func (f *fakeGraphVectorStore) Search(ctx context.Context, collection string, vector []float64, limit int, threshold float64, filter *ports.VectorFilter) ([]ports.VectorSearchResult, error) {
	f.lastFilter = filter
	return []ports.VectorSearchResult{
		{ID: "n1", Score: 0.9, Payload: map[string]interface{}{"content": "alpha", "namespace": "ns1"}},
		{ID: "n2", Score: 0.8, Payload: map[string]interface{}{"content": "beta", "namespace": "ns1"}},
	}, nil
}
func (f *fakeGraphVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	return []string{"knowledge_chunks_v1"}, nil
}
func (f *fakeGraphVectorStore) CountPoints(ctx context.Context, collection string) (int, error) {
	return 0, nil
}
func (f *fakeGraphVectorStore) SamplePoints(ctx context.Context, collection string, limit int) ([]ports.VectorPoint, error) {
	return nil, nil
}

func TestHandleGraphPost(t *testing.T) {
	vs := &fakeGraphVectorStore{}
	srv := &Server{
		graphService: &graph.Service{
			VectorStore: vs,
			Embedder:    &fakeEmbedder{},
		},
	}

	body := map[string]interface{}{
		"center_concept": "research",
		"namespaces":     []string{"ns1"},
		"visibility":     []string{"shared"},
		"tags":           []string{"tag-a"},
		"depth":          1,
		"limit":          5,
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/knowledge/graph", bytes.NewReader(raw))
	rec := httptest.NewRecorder()

	srv.handleGraph(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	if vs.lastFilter == nil || len(vs.lastFilter.Namespaces) != 1 || vs.lastFilter.Namespaces[0] != "ns1" {
		t.Fatalf("expected namespace filter to be applied, got %#v", vs.lastFilter)
	}

	var decoded graph.Response
	if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Center != "research" {
		t.Fatalf("center=%q", decoded.Center)
	}
	if len(decoded.Nodes) < 2 {
		t.Fatalf("expected nodes, got %#v", decoded.Nodes)
	}
	if len(decoded.Edges) < 1 {
		t.Fatalf("expected edges, got %#v", decoded.Edges)
	}
}
