//go:build testing
// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%v'", response["status"])
		}

		if response["service"] != "smartnotes-api" {
			t.Errorf("Expected service 'smartnotes-api', got '%v'", response["service"])
		}
	})
}

func TestNotesHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("CreateNote_Success", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/notes",
			Body: map[string]interface{}{
				"title":   "Test Note",
				"content": "This is test content for the note",
			},
		})

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Errorf("Expected status 200 or 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["title"] != "Test Note" {
			t.Errorf("Expected title 'Test Note', got '%v'", response["title"])
		}

		if response["id"] == nil || response["id"] == "" {
			t.Error("Expected note ID to be set")
		}
	})

	t.Run("CreateNote_InvalidJSON", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/notes",
			Body:   `{"invalid": "json"`,
		})

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("CreateNote_EmptyBody", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/notes",
			Body:   "",
		})

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			t.Error("Expected error for empty body")
		}
	})

	t.Run("CreateNote_MissingTitle", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/notes",
			Body: map[string]interface{}{
				"content": "Content without title",
			},
		})

		// Note: The API might accept this, so we just verify it doesn't crash
		if w.Code >= 500 {
			t.Errorf("Server error: %d", w.Code)
		}
	})

	t.Run("GetNotes_Success", func(t *testing.T) {
		// Create a test note first
		testNote := createTestNote(t, env, "Get Test Note", "Content for get test")
		defer testNote.Cleanup()

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/notes",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var notes []map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &notes); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should have at least the note we created
		found := false
		for _, note := range notes {
			if note["id"] == testNote.Note.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Created note not found in list")
		}
	})

	t.Run("GetNote_Success", func(t *testing.T) {
		testNote := createTestNote(t, env, "Single Note Test", "Content for single note")
		defer testNote.Cleanup()

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/notes/%s", testNote.Note.ID),
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["id"] != testNote.Note.ID {
			t.Errorf("Expected note ID %s, got %v", testNote.Note.ID, response["id"])
		}
	})

	t.Run("GetNote_NotFound", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddNonExistentResource("/api/notes/%s", "GET").
			Build()

		RunErrorTests(t, env, patterns)
	})

	t.Run("UpdateNote_Success", func(t *testing.T) {
		testNote := createTestNote(t, env, "Update Test", "Original content")
		defer testNote.Cleanup()

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   fmt.Sprintf("/api/notes/%s", testNote.Note.ID),
			Body: map[string]interface{}{
				"title":   "Updated Title",
				"content": "Updated content",
			},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["title"] != "Updated Title" {
			t.Errorf("Expected title 'Updated Title', got '%v'", response["title"])
		}
	})

	t.Run("UpdateNote_NotFound", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddNonExistentResource("/api/notes/%s", "PUT").
			Build()

		RunErrorTests(t, env, patterns)
	})

	t.Run("DeleteNote_Success", func(t *testing.T) {
		testNote := createTestNote(t, env, "Delete Test", "To be deleted")

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "DELETE",
			Path:   fmt.Sprintf("/api/notes/%s", testNote.Note.ID),
		})

		if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
			t.Errorf("Expected status 200 or 204, got %d", w.Code)
		}

		// Verify note is actually deleted
		checkW := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/notes/%s", testNote.Note.ID),
		})

		if checkW.Code == http.StatusOK {
			t.Error("Note still exists after deletion")
		}
	})

	t.Run("DeleteNote_NotFound", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddNonExistentResource("/api/notes/%s", "DELETE").
			Build()

		RunErrorTests(t, env, patterns)
	})
}

func TestFoldersHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("CreateFolder_Success", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/folders",
			Body: map[string]interface{}{
				"name":  "Test Folder",
				"icon":  "📁",
				"color": "#6366f1",
			},
		})

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Errorf("Expected status 200 or 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["name"] != "Test Folder" {
			t.Errorf("Expected name 'Test Folder', got '%v'", response["name"])
		}
	})

	t.Run("GetFolders_Success", func(t *testing.T) {
		testFolder := createTestFolder(t, env, "Get Test Folder")
		defer testFolder.Cleanup()

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/folders",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var folders []map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &folders); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		found := false
		for _, folder := range folders {
			if folder["id"] == testFolder.Folder.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Created folder not found in list")
		}
	})

	t.Run("UpdateFolder_Success", func(t *testing.T) {
		testFolder := createTestFolder(t, env, "Original Folder")
		defer testFolder.Cleanup()

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   fmt.Sprintf("/api/folders/%s", testFolder.Folder.ID),
			Body: map[string]interface{}{
				"name": "Updated Folder Name",
			},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("DeleteFolder_Success", func(t *testing.T) {
		testFolder := createTestFolder(t, env, "Delete Me")

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "DELETE",
			Path:   fmt.Sprintf("/api/folders/%s", testFolder.Folder.ID),
		})

		if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
			t.Errorf("Expected status 200 or 204, got %d", w.Code)
		}
	})
}

func TestTagsHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("CreateTag_Success", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/tags",
			Body: map[string]interface{}{
				"name":  "test-tag",
				"color": "#10b981",
			},
		})

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Errorf("Expected status 200 or 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["name"] != "test-tag" {
			t.Errorf("Expected name 'test-tag', got '%v'", response["name"])
		}
	})

	t.Run("GetTags_Success", func(t *testing.T) {
		testTag := createTestTag(t, env, "get-test-tag")
		defer testTag.Cleanup()

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/tags",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var tags []map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &tags); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		found := false
		for _, tag := range tags {
			if tag["id"] == testTag.Tag.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Created tag not found in list")
		}
	})
}

func TestTemplatesHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("CreateTemplate_Success", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/templates",
			Body: map[string]interface{}{
				"name":        "Meeting Notes",
				"description": "Template for meeting notes",
				"content":     "# Meeting Notes\n\n## Attendees:\n## Agenda:\n",
				"category":    "business",
			},
		})

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Errorf("Expected status 200 or 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["name"] != "Meeting Notes" {
			t.Errorf("Expected name 'Meeting Notes', got '%v'", response["name"])
		}
	})

	t.Run("GetTemplates_Success", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/templates",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var templates []map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &templates); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should return an array (even if empty)
		if templates == nil {
			t.Error("Expected templates array, got nil")
		}
	})
}

func TestSearchHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Search_Success", func(t *testing.T) {
		// Create test notes
		testNote := createTestNote(t, env, "Searchable Note", "This contains searchable keywords for testing")
		defer testNote.Cleanup()

		// Give database time to update
		time.Sleep(100 * time.Millisecond)

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body: map[string]interface{}{
				"query": "searchable",
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

		// Verify response structure
		if response["results"] == nil {
			t.Error("Expected results field in response")
		}
	})

	t.Run("Search_EmptyQuery", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body: map[string]interface{}{
				"query": "",
				"limit": 10,
			},
		})

		// Should handle empty query gracefully
		if w.Code >= 500 {
			t.Errorf("Server error on empty query: %d", w.Code)
		}
	})
}

func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("LargeContent", func(t *testing.T) {
		// Create note with large content
		largeContent := string(make([]byte, 100000))
		for i := range largeContent {
			largeContent = largeContent[:i] + "a"
		}

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/notes",
			Body: map[string]interface{}{
				"title":   "Large Content Note",
				"content": largeContent[:10000], // Limit to reasonable size
			},
		})

		if w.Code >= 500 {
			t.Errorf("Server error on large content: %d", w.Code)
		}
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/notes",
			Body: map[string]interface{}{
				"title":   "Special <>&\"' Characters",
				"content": "Content with emojis 🎉🔥 and symbols @#$%",
			},
		})

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Errorf("Expected success with special characters, got %d", w.Code)
		}
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		// Test concurrent note creation
		done := make(chan bool, 5)

		for i := 0; i < 5; i++ {
			go func(index int) {
				w := makeHTTPRequest(env, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/notes",
					Body: map[string]interface{}{
						"title":   fmt.Sprintf("Concurrent Note %d", index),
						"content": fmt.Sprintf("Content %d", index),
					},
				})

				if w.Code >= 500 {
					t.Errorf("Server error in concurrent request %d: %d", index, w.Code)
				}
				done <- true
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < 5; i++ {
			<-done
		}
	})
}

func TestPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	patterns := []PerformanceTestPattern{
		CreateNotePerformance(500 * time.Millisecond),
		SearchPerformance(1 * time.Second),
		ListNotesPerformance(500*time.Millisecond, 10),
		ListNotesPerformance(1*time.Second, 50),
	}

	RunPerformanceTests(t, env, patterns)
}
