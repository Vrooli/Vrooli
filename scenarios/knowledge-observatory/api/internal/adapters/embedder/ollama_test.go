package embedder

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOllamaEmbedRequiresBaseURL(t *testing.T) {
	client := &Ollama{}
	_, err := client.Embed(context.Background(), "hello")
	if err == nil || !strings.Contains(err.Error(), "base url") {
		t.Fatalf("expected base url error, got %v", err)
	}
}

func TestOllamaEmbedUsesDefaultModel(t *testing.T) {
	var got embeddingRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(embeddingResponse{Embedding: []float64{0.2, 0.4}})
	}))
	defer server.Close()

	client := &Ollama{BaseURL: server.URL, Client: server.Client()}
	out, err := client.Embed(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Model != "nomic-embed-text" {
		t.Fatalf("expected default model, got %q", got.Model)
	}
	if len(out) != 2 {
		t.Fatalf("expected embedding length 2, got %d", len(out))
	}
}

func TestOllamaEmbedHandlesNonOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	defer server.Close()

	client := &Ollama{BaseURL: server.URL, Client: server.Client()}
	_, err := client.Embed(context.Background(), "test")
	if err == nil || !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected status error, got %v", err)
	}
}
