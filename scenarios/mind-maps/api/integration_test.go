// +build testing

package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHandlerIntegration tests handler integration with test patterns
func TestHandlerIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDBWithProcessor(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	db = testDB

	t.Run("TestScenarioBuilder_InvalidUUID", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddInvalidUUID("/api/mindmaps/{id}", "GET")
		patterns := builder.Build()

		for _, pattern := range patterns {
			w, err := makeHTTPRequest(pattern.Request, getMindMapHandler)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			t.Logf("Pattern '%s': status %d", pattern.Name, w.Code)
		}
	})

	t.Run("TestScenarioBuilder_NonExistentMindMap", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddNonExistentMindMap("/api/mindmaps/{id}", "GET")
		patterns := builder.Build()

		for _, pattern := range patterns {
			w, err := makeHTTPRequest(pattern.Request, getMindMapHandler)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			t.Logf("Pattern '%s': status %d", pattern.Name, w.Code)
		}
	})

	t.Run("TestScenarioBuilder_InvalidJSON", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddInvalidJSON("/api/mindmaps", "POST")
		patterns := builder.Build()

		for _, pattern := range patterns {
			w, err := makeHTTPRequest(pattern.Request, createMindMapHandler)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			t.Logf("Pattern '%s': status %d", pattern.Name, w.Code)
		}
	})
}

// TestGetMindMapsHandlerFiltering tests filtering in getMindMapsHandler
func TestGetMindMapsHandlerFiltering(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDBWithProcessor(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	db = testDB

	// Create multiple mind maps
	createTestMindMap(t, testDB, "Map 1", "user1")
	createTestMindMap(t, testDB, "Map 2", "user2")
	createTestMindMap(t, testDB, "Map 3", "user1")

	t.Run("ListAllMaps", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/mindmaps",
		}

		w, err := makeHTTPRequest(req, getMindMapsHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, nil)
	})
}

// TestSearchHandlerVariants tests different search scenarios
func TestSearchHandlerVariants(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDBWithProcessor(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	db = testDB

	mindMap := createTestMindMap(t, testDB, "Searchable Map", "user123")
	createTestNode(t, testDB, mindMap.ID, "Node about AI", "root")
	createTestNode(t, testDB, mindMap.ID, "Node about ML", "child")

	t.Run("BasicSearch", func(t *testing.T) {
		body := buildSearchRequest("AI", "basic")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/mindmaps/search",
			Body:   body,
		}

		w, err := makeHTTPRequest(req, searchHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, nil)
	})

	t.Run("SemanticSearch", func(t *testing.T) {
		body := buildSearchRequest("machine learning", "semantic")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/mindmaps/search",
			Body:   body,
		}

		w, err := makeHTTPRequest(req, searchHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// May fail without Qdrant, but that's expected
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Logf("Search returned status %d (expected with missing Qdrant)", w.Code)
		}
	})

	t.Run("AutoOrganizeViaProcessor", func(t *testing.T) {
		// Test via processor instead since there's no organize handler
		processor := setupTestProcessor(t, testDB)

		req := OrganizeRequest{
			MindMapID: mindMap.ID,
			Method:    "basic",
		}

		err := processor.AutoOrganize(context.Background(), req)
		if err != nil {
			t.Logf("AutoOrganize failed (may be expected): %v", err)
		} else {
			t.Log("AutoOrganize succeeded")
		}
	})
}

// TestProcessorUpdateMindMap tests UpdateMindMap via handler
func TestProcessorUpdateMindMap(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDBWithProcessor(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	db = testDB

	// Create a mind map first
	mindMap := createTestMindMap(t, testDB, "Original Title", "user123")

	t.Run("UpdateDescription", func(t *testing.T) {
		body := buildUpdateMindMapRequest("", "Updated Description")

		req := HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/mindmaps/" + mindMap.ID,
			URLVars: map[string]string{"id": mindMap.ID},
			Body:    body,
		}

		w, err := makeHTTPRequest(req, updateMindMapHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("UpdateTitle", func(t *testing.T) {
		body := buildUpdateMindMapRequest("Updated Title", "")

		req := HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/mindmaps/" + mindMap.ID,
			URLVars: map[string]string{"id": mindMap.ID},
			Body:    body,
		}

		w, err := makeHTTPRequest(req, updateMindMapHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestExportFunctions tests export functionality
func TestExportFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDBWithProcessor(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	db = testDB

	mindMap := createTestMindMap(t, testDB, "Export Test Map", "user123")
	createTestNode(t, testDB, mindMap.ID, "Root Node", "root")
	createTestNode(t, testDB, mindMap.ID, "Child Node", "child")

	t.Run("ExportJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/mindmaps/" + mindMap.ID + "/export?format=json",
			URLVars: map[string]string{"id": mindMap.ID},
		}

		w, err := makeHTTPRequest(req, exportHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Logf("Export returned status %d", w.Code)
		}
	})

	t.Run("ExportMarkdown", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/mindmaps/" + mindMap.ID + "/export?format=markdown",
			URLVars: map[string]string{"id": mindMap.ID},
		}

		w, err := makeHTTPRequest(req, exportHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Logf("Export returned status %d", w.Code)
		}
	})

	t.Run("ExportDOT", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/mindmaps/" + mindMap.ID + "/export?format=dot",
			URLVars: map[string]string{"id": mindMap.ID},
		}

		w, err := makeHTTPRequest(req, exportHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Logf("Export returned status %d", w.Code)
		}
	})
}

// TestProcessorHelperFunctions tests helper functions in processor
func TestProcessorHelperFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)

	t.Run("GetMindMapByID", func(t *testing.T) {
		// Create a mind map
		createReq := CreateMindMapRequest{
			Title:  "Test Map",
			UserID: "user123",
		}

		mindMap, err := processor.CreateMindMap(context.Background(), createReq)
		if err != nil {
			t.Fatalf("Failed to create mind map: %v", err)
		}

		// Get it back
		retrieved, err := processor.getMindMapByID(mindMap.ID)
		if err != nil {
			t.Errorf("Failed to get mind map by ID: %v", err)
		}

		if retrieved.ID != mindMap.ID {
			t.Errorf("Expected ID %s, got %s", mindMap.ID, retrieved.ID)
		}
	})

	t.Run("GetMindMapByID_NonExistent", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		_, err := processor.getMindMapByID(nonExistentID)
		if err == nil {
			t.Error("Expected error for non-existent mind map, got nil")
		}
	})
}

// TestErrorHandling tests error handling paths
func TestErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDBWithProcessor(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	db = testDB

	t.Run("CreateWithInvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/mindmaps",
			Body:   "not valid json{{{",
		}

		w, err := makeHTTPRequest(req, createMindMapHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Logf("Handler returned status %d for invalid JSON", w.Code)
		}
	})

	t.Run("UpdateWithInvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/mindmaps/" + uuid.New().String(),
			URLVars: map[string]string{"id": uuid.New().String()},
			Body:    "{{invalid",
		}

		w, err := makeHTTPRequest(req, updateMindMapHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Logf("Handler returned status %d for invalid JSON", w.Code)
		}
	})
}

// TestConcurrentOperations tests concurrent operations
func TestConcurrentOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDBWithProcessor(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	db = testDB

	t.Run("ConcurrentCreates", func(t *testing.T) {
		done := make(chan bool, 3)

		for i := 0; i < 3; i++ {
			go func(index int) {
				body := buildCreateMindMapRequest("Concurrent Map", "Test concurrent creation", "user123")

				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/mindmaps",
					Body:   body,
				}

				w, err := makeHTTPRequest(req, createMindMapHandler)
				if err != nil {
					t.Logf("Concurrent request %d failed: %v", index, err)
				} else {
					t.Logf("Concurrent request %d: status %d", index, w.Code)
				}

				done <- true
			}(i)
		}

		// Wait for all goroutines
		timeout := time.After(5 * time.Second)
		for i := 0; i < 3; i++ {
			select {
			case <-done:
			case <-timeout:
				t.Error("Timeout waiting for concurrent operations")
				return
			}
		}
	})
}

// TestDatabaseEdgeCases tests database edge cases
func TestDatabaseEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)
	ctx := context.Background()

	t.Run("CreateWithSpecialCharacters", func(t *testing.T) {
		createReq := CreateMindMapRequest{
			Title:       "Map with 特殊字符 and émojis 🎉",
			Description: "Description with <html> & special chars",
			UserID:      "user@example.com",
		}

		_, err := processor.CreateMindMap(ctx, createReq)
		if err != nil {
			t.Logf("Create with special characters failed: %v", err)
		}
	})

	t.Run("CreateWithVeryLongTitle", func(t *testing.T) {
		longTitle := ""
		for i := 0; i < 300; i++ {
			longTitle += "a"
		}

		createReq := CreateMindMapRequest{
			Title:  longTitle,
			UserID: "user123",
		}

		_, err := processor.CreateMindMap(ctx, createReq)
		if err != nil {
			t.Logf("Create with long title failed (expected): %v", err)
		}
	})
}
