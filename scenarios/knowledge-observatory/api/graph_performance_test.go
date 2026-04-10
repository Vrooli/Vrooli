package main

import (
	"context"
	"testing"
	"time"

	"knowledge-observatory/internal/services/graph"
)

// TestGraphPerformanceBudget validates graph latency expectations. [REQ:KO-KG-003]
func TestGraphPerformanceBudget(t *testing.T) {
	service := &graph.Service{VectorStore: stubVectorStore{}, Embedder: stubEmbedder{}}
	start := time.Now()
	_, err := service.Graph(context.Background(), graph.Request{Center: "alpha", Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("graph exceeded budget: %s", elapsed)
	}
}
