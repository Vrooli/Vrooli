package aisearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sharedsearch "github.com/vrooli/ai-go/search"
)

func TestVectorStoreForPolicyCreateUsesPolicyDimensions(t *testing.T) {
	var gotSize int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			var req createCollectionRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			gotSize = req.Vectors.Size
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	vs := NewVectorStoreForPolicy(server.URL, "", "", sharedsearch.EmbeddingPolicy{
		Role:       "embedding.default",
		Model:      "fixture-embed-model:latest",
		Dimensions: 1234,
	})
	if err := vs.EnsureCollection(context.Background()); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}
	if gotSize != 1234 {
		t.Errorf("expected policy dimensions 1234, got %d", gotSize)
	}
}
