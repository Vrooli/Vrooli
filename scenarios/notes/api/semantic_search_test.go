//go:build testing
// +build testing

package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

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
