package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// [REQ:KO-SS-001,KO-SS-002,KO-SS-003] Test server creation error handling
func TestNewServerIntegration(t *testing.T) {
	t.Run("handles missing database config gracefully", func(t *testing.T) {
		// Temporarily clear database environment variables
		oldDB := os.Getenv("DATABASE_URL")
		oldUser := os.Getenv("POSTGRES_USER")
		defer func() {
			if oldDB != "" {
				os.Setenv("DATABASE_URL", oldDB)
			}
			if oldUser != "" {
				os.Setenv("POSTGRES_USER", oldUser)
			}
		}()

		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("POSTGRES_USER")

		_, err := NewServer()
		if err == nil {
			t.Error("NewServer() should return error when database config is missing")
		}
	})
}

// [REQ:KO-SS-002] Test embedding generation with mock Ollama
func TestGenerateEmbeddingIntegration(t *testing.T) {
	// Create mock Ollama server
	mockOllama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Return mock embedding
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"embedding":[0.1,0.2,0.3,0.4,0.5]}`))
	}))
	defer mockOllama.Close()

	// Set mock Ollama URL
	os.Setenv("OLLAMA_URL", mockOllama.URL)
	defer os.Unsetenv("OLLAMA_URL")

	// Create mock server (no DB connection needed for this test)
	server := &Server{}

	tests := []struct {
		name      string
		text      string
		wantLen   int
		wantError bool
	}{
		{
			name:      "generates embedding for valid text",
			text:      "test query",
			wantLen:   5,
			wantError: false,
		},
		{
			name:      "handles empty text",
			text:      "",
			wantLen:   5,
			wantError: false,
		},
		{
			name:      "handles long text",
			text:      "this is a very long query that should still work fine with the embedding service",
			wantLen:   5,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			embedding, err := server.generateEmbedding(ctx, tt.text)
			if (err != nil) != tt.wantError {
				t.Errorf("generateEmbedding() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && len(embedding) != tt.wantLen {
				t.Errorf("generateEmbedding() returned %d dimensions, want %d", len(embedding), tt.wantLen)
			}
		})
	}
}

// [REQ:KO-SS-002] Test embedding generation error cases
func TestGenerateEmbeddingErrors(t *testing.T) {
	// Create failing mock Ollama server
	mockOllama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"model not found"}`))
	}))
	defer mockOllama.Close()

	os.Setenv("OLLAMA_URL", mockOllama.URL)
	defer os.Unsetenv("OLLAMA_URL")

	server := &Server{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := server.generateEmbedding(ctx, "test")
	if err == nil {
		t.Error("generateEmbedding() should return error for failing Ollama")
	}
}

// [REQ:KO-SS-003] Test Qdrant search with real resource
func TestSearchQdrantIntegration(t *testing.T) {
	// This test requires real Qdrant resource because getCollections() uses exec.Command("resource-qdrant")
	// Skip by default in unit test phase; enable explicitly with INTEGRATION_TEST=true
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("skipping Qdrant integration test - set INTEGRATION_TEST=true to run")
	}

	// Note: This test uses resource-qdrant CLI which connects to real Qdrant
	// Cannot mock HTTP because getCollections() uses exec.Command

	// Create mock Qdrant server
	mockQdrant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/collections" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result":{"collections":[{"name":"test_collection"}]}}`))
			return
		}
		if r.URL.Path == "/collections/test_collection/points/search" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result":[{"id":1,"score":0.95,"payload":{"text":"test result"}}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockQdrant.Close()

	os.Setenv("QDRANT_URL", mockQdrant.URL)
	defer os.Unsetenv("QDRANT_URL")

	server := &Server{}

	tests := []struct {
		name       string
		collection string
		vector     []float64
		limit      int
		threshold  float64
		wantLen    int
		wantError  bool
	}{
		{
			name:       "searches all collections",
			collection: "",
			vector:     []float64{0.1, 0.2, 0.3},
			limit:      10,
			threshold:  0.7,
			wantLen:    1,
			wantError:  false,
		},
		{
			name:       "searches specific collection",
			collection: "test_collection",
			vector:     []float64{0.1, 0.2, 0.3},
			limit:      5,
			threshold:  0.8,
			wantLen:    1,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			results, err := server.searchQdrant(ctx, tt.collection, tt.vector, tt.limit, tt.threshold)
			if (err != nil) != tt.wantError {
				t.Errorf("searchQdrant() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && len(results) != tt.wantLen {
				t.Errorf("searchQdrant() returned %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}

// [REQ:KO-SS-003] Test single collection search
func TestSearchSingleCollectionIntegration(t *testing.T) {
	// Create mock Qdrant server
	mockQdrant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result":[{"id":1,"score":0.95,"payload":{"text":"test result","collection":"test_collection"}}]}`))
	}))
	defer mockQdrant.Close()

	os.Setenv("QDRANT_URL", mockQdrant.URL)
	defer os.Unsetenv("QDRANT_URL")

	server := &Server{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := server.searchSingleCollection(ctx, "test_collection", []float64{0.1, 0.2, 0.3}, 10, 0.7)
	if err != nil {
		t.Errorf("searchSingleCollection() error = %v", err)
		return
	}
	if len(results) == 0 {
		t.Error("searchSingleCollection() returned no results")
	}
	if len(results) > 0 {
		// Verify result has metadata
		if results[0].Metadata == nil {
			t.Error("searchSingleCollection() result missing metadata")
		}
		// Collection name should be in metadata
		if coll, ok := results[0].Metadata["collection"]; ok {
			if coll != "test_collection" {
				t.Errorf("searchSingleCollection() metadata collection = %v, want test_collection", coll)
			}
		}
	}
}
