package main

import (
	"context"
	"testing"
	"time"

	"knowledge-observatory/internal/services/search"
)

// TestSearchPerformanceBudget validates search latency expectations. [REQ:KO-SS-004]
func TestSearchPerformanceBudget(t *testing.T) {
	service := &search.Service{VectorStore: stubVectorStore{}, Embedder: stubEmbedder{}}
	start := time.Now()
	_, err := service.Search(context.Background(), search.Request{Query: "hello", Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("search exceeded budget: %s", elapsed)
	}
}
