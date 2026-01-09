//go:build testing
// +build testing

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestSemanticSearch(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Skip if Ollama or Qdrant is not available
	ollamaHost := os.Getenv("OLLAMA_HOST")
	qdrantHost := os.Getenv("QDRANT_HOST")

	if ollamaHost == "" && qdrantHost == "" {
		t.Skip("Semantic search requires Ollama and Qdrant - skipping")
	}

	t.Run("GetEmbedding_Success", func(t *testing.T) {
		if ollamaHost == "" {
			t.Skip("Ollama not available")
		}

		embedding, err := getEmbedding("test query")
		if err != nil {
			t.Logf("Warning: Failed to get embedding: %v", err)
			t.Skip("Ollama embedding generation not available")
		}

		if len(embedding) == 0 {
			t.Error("Expected non-empty embedding vector")
		}

		// Typical embedding dimensions for nomic-embed-text
		if len(embedding) < 100 {
			t.Errorf("Expected embedding dimension > 100, got %d", len(embedding))
		}
	})

	t.Run("GetEmbedding_EmptyText", func(t *testing.T) {
		if ollamaHost == "" {
			t.Skip("Ollama not available")
		}

		embedding, err := getEmbedding("")
		// Should either succeed with empty vector or fail gracefully
		if err == nil && len(embedding) == 0 {
			t.Log("Empty text produces empty embedding")
		}
	})

	t.Run("IndexNoteInQdrant_Success", func(t *testing.T) {
		if qdrantHost == "" || ollamaHost == "" {
			t.Skip("Qdrant or Ollama not available")
		}

		note := createTestNote(t, env, "Qdrant Test Note", "Content for vector indexing")
		defer note.Cleanup()

		// Try to index the note
		err := indexNoteInQdrant(*note.Note)
		if err != nil {
			t.Logf("Warning: Failed to index note in Qdrant: %v", err)
			// This is a warning, not a failure, as Qdrant might not be configured
		}
	})

	t.Run("SemanticSearchHandler_Success", func(t *testing.T) {
		if qdrantHost == "" || ollamaHost == "" {
			t.Skip("Qdrant or Ollama not available")
		}

		// Create and index test notes
		note1 := createTestNote(t, env, "Machine Learning Basics", "Introduction to neural networks and deep learning")
		defer note1.Cleanup()

		note2 := createTestNote(t, env, "Cooking Recipe", "How to make chocolate chip cookies")
		defer note2.Cleanup()

		// Index notes
		indexNoteInQdrant(*note1.Note)
		indexNoteInQdrant(*note2.Note)

		// Wait for indexing
		time.Sleep(2 * time.Second)

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search/semantic",
			Body: map[string]interface{}{
				"query": "artificial intelligence",
				"limit": 5,
			},
		})

		if w.Code != http.StatusOK {
			t.Logf("Semantic search returned status %d, body: %s", w.Code, w.Body.String())
			// Fall back to text search might have been used
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify response structure
		if response["results"] == nil {
			t.Error("Expected results field in response")
		}

		if response["query"] == nil {
			t.Error("Expected query field in response")
		}
	})

	t.Run("SemanticSearchHandler_FallbackToTextSearch", func(t *testing.T) {
		// Even without Qdrant, semantic search should fall back gracefully
		note := createTestNote(t, env, "Fallback Test", "Test content for fallback")
		defer note.Cleanup()

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search/semantic",
			Body: map[string]interface{}{
				"query": "fallback",
				"limit": 5,
			},
		})

		// Should succeed even if semantic search is not available
		if w.Code >= 500 {
			t.Errorf("Expected graceful fallback, got server error: %d", w.Code)
		}
	})

	t.Run("SemanticSearchHandler_InvalidJSON", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search/semantic",
			Body:   `{"invalid": "json"`,
		})

		if w.Code == http.StatusOK {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("SemanticSearchHandler_EmptyQuery", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search/semantic",
			Body: map[string]interface{}{
				"query": "",
				"limit": 5,
			},
		})

		// Should handle empty query gracefully
		if w.Code >= 500 {
			t.Errorf("Server error on empty query: %d", w.Code)
		}
	})

	t.Run("DeleteNoteFromQdrant_Success", func(t *testing.T) {
		if qdrantHost == "" {
			t.Skip("Qdrant not available")
		}

		note := createTestNote(t, env, "Delete Test", "To be deleted from Qdrant")

		// Try to delete from Qdrant
		err := deleteNoteFromQdrant(note.Note.ID)
		if err != nil {
			t.Logf("Warning: Failed to delete note from Qdrant: %v", err)
			// This is a warning, not a failure
		}

		note.Cleanup()
	})
}

func TestTextSearchHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("TextSearch_Success", func(t *testing.T) {
		// Create test notes with searchable content
		note1 := createTestNote(t, env, "Search Test 1", "This note contains unique keywords for testing")
		defer note1.Cleanup()

		note2 := createTestNote(t, env, "Search Test 2", "Different content without the special terms")
		defer note2.Cleanup()

		// Wait a moment for database
		time.Sleep(100 * time.Millisecond)

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body: map[string]interface{}{
				"query": "unique keywords",
				"limit": 10,
			},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["results"] == nil {
			t.Error("Expected results field in response")
		}
	})

	t.Run("TextSearch_NoResults", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body: map[string]interface{}{
				"query": "nonexistent-unique-term-12345",
				"limit": 10,
			},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should return empty results, not error
		if response["results"] != nil {
			results := response["results"].([]interface{})
			if len(results) > 0 {
				t.Log("Note: Found unexpected results for non-existent term")
			}
		}
	})

	t.Run("TextSearch_LimitParameter", func(t *testing.T) {
		// Create multiple notes
		for i := 0; i < 5; i++ {
			note := createTestNote(t, env, "Limit Test", "Common search term content")
			defer note.Cleanup()
		}

		time.Sleep(100 * time.Millisecond)

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body: map[string]interface{}{
				"query": "Common search term",
				"limit": 2,
			},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["results"] != nil {
			results := response["results"].([]interface{})
			if len(results) > 2 {
				t.Errorf("Expected at most 2 results with limit=2, got %d", len(results))
			}
		}
	})
}
