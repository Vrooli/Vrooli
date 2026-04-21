package aisearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEmbedder_Embed_Success(t *testing.T) {
	want := []float64{0.1, 0.2, 0.3, 0.4}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("expected path /api/embeddings, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req embeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "nomic-embed-text" {
			t.Errorf("expected model nomic-embed-text, got %s", req.Model)
		}
		if req.Prompt != "hello world" {
			t.Errorf("expected prompt 'hello world', got %q", req.Prompt)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(embeddingResponse{Embedding: want})
	}))
	defer server.Close()

	e := NewEmbedder(server.URL, "nomic-embed-text")
	got, err := e.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d dims, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("dim %d: got %f want %f", i, got[i], want[i])
		}
	}
}

func TestEmbedder_Embed_EmptyURL(t *testing.T) {
	e := NewEmbedder("", "nomic-embed-text")
	if _, err := e.Embed(context.Background(), "x"); err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestEmbedder_Embed_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	e := NewEmbedder(server.URL, "nomic-embed-text")
	_, err := e.Embed(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error for server 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain 500, got %v", err)
	}
}

func TestEmbedder_DefaultModel(t *testing.T) {
	e := NewEmbedder("http://localhost:11434", "")
	if e.Model != "nomic-embed-text" {
		t.Errorf("expected default model nomic-embed-text, got %s", e.Model)
	}
}

func TestEmbedder_Available_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(embeddingResponse{Embedding: []float64{0.1}})
	}))
	defer server.Close()

	e := NewEmbedder(server.URL, "nomic-embed-text")
	if !e.Available(context.Background()) {
		t.Error("expected Available to return true")
	}
}

func TestEmbedder_Available_EmptyURL(t *testing.T) {
	e := NewEmbedder("", "nomic-embed-text")
	if e.Available(context.Background()) {
		t.Error("expected Available to return false for empty URL")
	}
}

func TestEmbedder_Available_Unreachable(t *testing.T) {
	e := NewEmbedder("http://127.0.0.1:1", "nomic-embed-text")
	e.Client = &http.Client{Timeout: 100 * time.Millisecond}
	if e.Available(context.Background()) {
		t.Error("expected Available to return false for unreachable server")
	}
}
