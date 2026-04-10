package main

import (
	"context"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/ports"
	"knowledge-observatory/internal/services/graph"
	"knowledge-observatory/internal/services/search"
)

type stubEmbedder struct {
	vector []float64
}

func (s stubEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if len(s.vector) > 0 {
		return s.vector, nil
	}
	return []float64{0.1, 0.2, 0.3}, nil
}

type stubVectorStore struct {
	collections []string
	results     []ports.VectorSearchResult
}

func (s stubVectorStore) EnsureCollection(ctx context.Context, collection string, vectorSize int) error {
	return nil
}

func (s stubVectorStore) UpsertPoint(ctx context.Context, collection string, id string, vector []float64, payload map[string]interface{}) error {
	return nil
}

func (s stubVectorStore) DeletePoint(ctx context.Context, collection string, id string) error {
	return nil
}

func (s stubVectorStore) Search(ctx context.Context, collection string, vector []float64, limit int, threshold float64, filter *ports.VectorFilter) ([]ports.VectorSearchResult, error) {
	if len(s.results) > 0 {
		return s.results, nil
	}
	return []ports.VectorSearchResult{
		{ID: "demo", Score: 0.9, Payload: map[string]interface{}{"content": "demo"}},
	}, nil
}

func (s stubVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	if len(s.collections) > 0 {
		return s.collections, nil
	}
	return []string{"default"}, nil
}

func (s stubVectorStore) CountPoints(ctx context.Context, collection string) (int, error) {
	return len(s.results), nil
}

func (s stubVectorStore) SamplePoints(ctx context.Context, collection string, limit int) ([]ports.VectorPoint, error) {
	return nil, nil
}

func newTestServer() *Server {
	return &Server{
		config: &Config{Port: "8080"},
		router: mux.NewRouter(),
	}
}

func newTestServerWithServices() *Server {
	srv := newTestServer()
	vs := stubVectorStore{collections: []string{"default"}}
	emb := stubEmbedder{}
	srv.searchService = &search.Service{VectorStore: vs, Embedder: emb}
	srv.graphService = &graph.Service{VectorStore: vs, Embedder: emb}
	srv.setupRoutes()
	return srv
}
