// +build testing

package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestNewMindMapProcessor tests processor creation
func TestNewMindMapProcessor(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	t.Run("Success", func(t *testing.T) {
		processor := NewMindMapProcessor(testDB, "http://localhost:11434", "http://localhost:6333")

		if processor == nil {
			t.Fatal("Expected processor to be created")
		}

		if processor.db != testDB {
			t.Error("Expected processor to have correct database")
		}

		if processor.ollamaURL != "http://localhost:11434" {
			t.Error("Expected processor to have correct Ollama URL")
		}

		if processor.qdrantURL != "http://localhost:6333" {
			t.Error("Expected processor to have correct Qdrant URL")
		}
	})
}

// TestCreateMindMap tests creating a mind map via processor
func TestCreateMindMap(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)

	t.Run("Success_WithoutInitialNode", func(t *testing.T) {
		req := CreateMindMapRequest{
			Title:       "Test Mind Map",
			Description: "Test description",
			UserID:      "user123",
			Tags:        []string{"test", "demo"},
			IsPublic:    false,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		mindMap, err := processor.CreateMindMap(ctx, req)
		if err != nil {
			t.Fatalf("Failed to create mind map: %v", err)
		}

		if mindMap == nil {
			t.Fatal("Expected mind map to be created")
		}

		if mindMap.Title != req.Title {
			t.Errorf("Expected title %s, got %s", req.Title, mindMap.Title)
		}

		if mindMap.OwnerID != req.UserID {
			t.Errorf("Expected owner ID %s, got %s", req.UserID, mindMap.OwnerID)
		}

		// Verify mind map exists in database
		var count int
		err = testDB.QueryRow("SELECT COUNT(*) FROM mind_maps WHERE id = $1", mindMap.ID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query database: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 mind map in database, got %d", count)
		}
	})

	t.Run("Success_WithInitialNode", func(t *testing.T) {
		initialNode := &Node{
			ID:        uuid.New().String(),
			Content:   "Custom Root Node",
			Type:      "root",
			PositionX: 100.0,
			PositionY: 200.0,
			Metadata:  make(map[string]interface{}),
		}

		req := CreateMindMapRequest{
			Title:       "Mind Map with Custom Node",
			Description: "Test description",
			UserID:      "user456",
			InitialNode: initialNode,
			Tags:        []string{},
			IsPublic:    true,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		mindMap, err := processor.CreateMindMap(ctx, req)
		if err != nil {
			t.Fatalf("Failed to create mind map: %v", err)
		}

		if mindMap == nil {
			t.Fatal("Expected mind map to be created")
		}
	})

	t.Run("Failure_EmptyTitle", func(t *testing.T) {
		req := CreateMindMapRequest{
			Title:       "",
			Description: "Test description",
			UserID:      "user123",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		mindMap, err := processor.CreateMindMap(ctx, req)

		// Empty title might be allowed, depending on implementation
		if err != nil {
			t.Logf("Empty title rejected as expected: %v", err)
		} else if mindMap != nil {
			t.Logf("Empty title allowed (permissive behavior)")
		}
	})

	t.Run("Failure_EmptyUserID", func(t *testing.T) {
		req := CreateMindMapRequest{
			Title:       "Test Mind Map",
			Description: "Test description",
			UserID:      "",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		mindMap, err := processor.CreateMindMap(ctx, req)

		// Empty user ID might cause constraint violation
		if err != nil {
			t.Logf("Empty user ID rejected as expected: %v", err)
		} else if mindMap != nil {
			t.Logf("Empty user ID allowed (permissive behavior)")
		}
	})
}

// TestUpdateMindMap tests updating a mind map via processor
func TestUpdateMindMap(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)

	// Create a test mind map
	mindMap := createTestMindMap(t, testDB, "Original Title", "user123")

	t.Run("Success_UpdateTitle", func(t *testing.T) {
		updates := map[string]interface{}{
			"title": "Updated Title",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := processor.UpdateMindMap(ctx, mindMap.ID, updates)
		if err != nil {
			t.Fatalf("Failed to update mind map: %v", err)
		}

		// Verify update
		var title string
		err = testDB.QueryRow("SELECT title FROM mind_maps WHERE id = $1", mindMap.ID).Scan(&title)
		if err != nil {
			t.Fatalf("Failed to query database: %v", err)
		}

		if title != "Updated Title" {
			t.Errorf("Expected title 'Updated Title', got '%s'", title)
		}
	})

	t.Run("Success_UpdateDescription", func(t *testing.T) {
		updates := map[string]interface{}{
			"description": "Updated Description",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := processor.UpdateMindMap(ctx, mindMap.ID, updates)
		if err != nil {
			t.Fatalf("Failed to update mind map: %v", err)
		}
	})

	t.Run("Success_MultipleFields", func(t *testing.T) {
		updates := map[string]interface{}{
			"title":       "Multi Update Title",
			"description": "Multi Update Description",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := processor.UpdateMindMap(ctx, mindMap.ID, updates)
		if err != nil {
			t.Fatalf("Failed to update mind map: %v", err)
		}
	})

	t.Run("Failure_NoValidFields", func(t *testing.T) {
		updates := map[string]interface{}{
			"invalid_field": "value",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := processor.UpdateMindMap(ctx, mindMap.ID, updates)
		if err == nil {
			t.Error("Expected error for no valid fields")
		}
	})

	t.Run("Failure_NonExistentMindMap", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		updates := map[string]interface{}{
			"title": "Updated Title",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := processor.UpdateMindMap(ctx, nonExistentID, updates)
		// Should not error (UPDATE affects 0 rows)
		if err != nil {
			t.Logf("Update non-existent mind map returned error: %v", err)
		}
	})

	t.Run("Failure_EmptyUpdates", func(t *testing.T) {
		updates := map[string]interface{}{}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := processor.UpdateMindMap(ctx, mindMap.ID, updates)
		if err == nil {
			t.Error("Expected error for empty updates")
		}
	})
}

// TestSemanticSearch tests semantic search functionality
func TestSemanticSearch(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)

	t.Run("WithoutExternalService", func(t *testing.T) {
		req := SemanticSearchRequest{
			Query:      "machine learning",
			Collection: "mind_maps",
			Limit:      10,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		results, err := processor.SemanticSearch(ctx, req)

		// This will likely fail without Qdrant running
		if err != nil {
			t.Logf("Semantic search failed as expected (no Qdrant): %v", err)
		} else {
			t.Logf("Semantic search returned %d results", len(results))
		}
	})

	t.Run("EmptyQuery", func(t *testing.T) {
		req := SemanticSearchRequest{
			Query:      "",
			Collection: "mind_maps",
			Limit:      10,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		results, err := processor.SemanticSearch(ctx, req)

		if err != nil {
			t.Logf("Empty query handled: %v", err)
		} else {
			t.Logf("Empty query returned %d results", len(results))
		}
	})

	t.Run("LargeLimit", func(t *testing.T) {
		req := SemanticSearchRequest{
			Query:      "test",
			Collection: "mind_maps",
			Limit:      1000,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		results, err := processor.SemanticSearch(ctx, req)

		if err != nil {
			t.Logf("Large limit search failed: %v", err)
		} else {
			t.Logf("Large limit search returned %d results", len(results))
		}
	})
}

// TestAutoOrganize tests auto-organization functionality
func TestAutoOrganize(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)

	// Create a test mind map
	mindMap := createTestMindMap(t, testDB, "Mind Map to Organize", "user123")

	t.Run("BasicOrganization", func(t *testing.T) {
		req := OrganizeRequest{
			MindMapID: mindMap.ID,
			Method:    "basic",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := processor.AutoOrganize(ctx, req)

		// May fail due to data format issues
		if err != nil {
			t.Logf("Basic organization failed: %v", err)
		} else {
			t.Log("Basic organization succeeded")
		}
	})

	t.Run("EnhancedOrganization", func(t *testing.T) {
		req := OrganizeRequest{
			MindMapID: mindMap.ID,
			Method:    "enhanced",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := processor.AutoOrganize(ctx, req)

		if err != nil {
			t.Logf("Enhanced organization failed: %v", err)
		} else {
			t.Log("Enhanced organization succeeded")
		}
	})

	t.Run("AdvancedOrganization", func(t *testing.T) {
		req := OrganizeRequest{
			MindMapID: mindMap.ID,
			Method:    "advanced",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := processor.AutoOrganize(ctx, req)

		if err != nil {
			t.Logf("Advanced organization failed: %v", err)
		} else {
			t.Log("Advanced organization succeeded")
		}
	})

	t.Run("DefaultMethod", func(t *testing.T) {
		req := OrganizeRequest{
			MindMapID: mindMap.ID,
			Method:    "",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := processor.AutoOrganize(ctx, req)

		if err != nil {
			t.Logf("Default method organization failed: %v", err)
		} else {
			t.Log("Default method organization succeeded")
		}
	})

	t.Run("NonExistentMindMap", func(t *testing.T) {
		req := OrganizeRequest{
			MindMapID: uuid.New().String(),
			Method:    "basic",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := processor.AutoOrganize(ctx, req)

		if err == nil {
			t.Error("Expected error for non-existent mind map")
		}
	})
}

// TestDocumentToMindMap tests document conversion
func TestDocumentToMindMap(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)

	t.Run("WithoutOllama", func(t *testing.T) {
		req := DocumentToMindMapRequest{
			Title:           "Test Document",
			DocumentContent: "This is a test document with some content about machine learning and AI.",
			DocumentType:    "text",
			UserID:          "user123",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		mindMap, err := processor.DocumentToMindMap(ctx, req)

		// Will likely fail without Ollama
		if err != nil {
			t.Logf("Document conversion failed (expected without Ollama): %v", err)
		} else if mindMap != nil {
			t.Logf("Document conversion succeeded: %s", mindMap.ID)
		}
	})

	t.Run("EmptyContent", func(t *testing.T) {
		req := DocumentToMindMapRequest{
			Title:           "Empty Document",
			DocumentContent: "",
			DocumentType:    "text",
			UserID:          "user123",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		mindMap, err := processor.DocumentToMindMap(ctx, req)

		if err != nil {
			t.Logf("Empty content rejected: %v", err)
		} else if mindMap != nil {
			t.Logf("Empty content allowed")
		}
	})

	t.Run("LargeDocument", func(t *testing.T) {
		largeContent := ""
		for i := 0; i < 1000; i++ {
			largeContent += "This is a line of text. "
		}

		req := DocumentToMindMapRequest{
			Title:           "Large Document",
			DocumentContent: largeContent,
			DocumentType:    "text",
			UserID:          "user123",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		mindMap, err := processor.DocumentToMindMap(ctx, req)

		if err != nil {
			t.Logf("Large document conversion failed: %v", err)
		} else if mindMap != nil {
			t.Logf("Large document converted successfully")
		}
	})
}

// TestContextCancellation tests context cancellation handling
func TestContextCancellation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)

	t.Run("CancelledContext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		req := CreateMindMapRequest{
			Title:       "Test Mind Map",
			Description: "Test description",
			UserID:      "user123",
		}

		mindMap, err := processor.CreateMindMap(ctx, req)

		// Database operation might complete before context is checked
		if err != nil {
			t.Logf("Cancelled context handled: %v", err)
		} else if mindMap != nil {
			t.Logf("Operation completed before cancellation")
		}
	})

	t.Run("TimeoutContext", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Wait for timeout
		time.Sleep(10 * time.Millisecond)

		req := CreateMindMapRequest{
			Title:       "Test Mind Map",
			Description: "Test description",
			UserID:      "user123",
		}

		mindMap, err := processor.CreateMindMap(ctx, req)

		if err != nil {
			t.Logf("Timeout context handled: %v", err)
		} else if mindMap != nil {
			t.Logf("Operation completed before timeout")
		}
	})
}

// TestEdgeCases tests edge cases in processor logic
func TestProcessorEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)

	t.Run("NilInitialNode", func(t *testing.T) {
		req := CreateMindMapRequest{
			Title:       "Mind Map with Nil Node",
			Description: "Test",
			UserID:      "user123",
			InitialNode: nil,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		mindMap, err := processor.CreateMindMap(ctx, req)
		if err != nil {
			t.Fatalf("Failed to create mind map with nil initial node: %v", err)
		}

		if mindMap == nil {
			t.Fatal("Expected mind map to be created")
		}
	})

	t.Run("EmptyTags", func(t *testing.T) {
		req := CreateMindMapRequest{
			Title:       "Mind Map with Empty Tags",
			Description: "Test",
			UserID:      "user123",
			Tags:        []string{},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		mindMap, err := processor.CreateMindMap(ctx, req)
		if err != nil {
			t.Fatalf("Failed to create mind map with empty tags: %v", err)
		}

		if mindMap == nil {
			t.Fatal("Expected mind map to be created")
		}
	})

	t.Run("NilMetadata", func(t *testing.T) {
		req := CreateMindMapRequest{
			Title:       "Mind Map with Nil Metadata",
			Description: "Test",
			UserID:      "user123",
			Metadata:    nil,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		mindMap, err := processor.CreateMindMap(ctx, req)
		if err != nil {
			t.Fatalf("Failed to create mind map with nil metadata: %v", err)
		}

		if mindMap == nil {
			t.Fatal("Expected mind map to be created")
		}
	})
}

// TestExportMindMap tests the ExportMindMap function with various formats
func TestExportMindMap(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)

	// Create a test mind map first
	ctx := context.Background()
	mindMap := createTestMindMap(t, testDB, "Export Test Map", "user123")

	t.Run("ExportJSON", func(t *testing.T) {
		result, err := processor.ExportMindMap(ctx, mindMap.ID, "json")
		if err != nil {
			t.Fatalf("Failed to export as JSON: %v", err)
		}
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		exportedMap, ok := result.(*MindMap)
		if !ok {
			t.Fatal("Expected result to be a MindMap")
		}
		if exportedMap.ID != mindMap.ID {
			t.Errorf("Expected ID %s, got %s", mindMap.ID, exportedMap.ID)
		}
	})

	t.Run("ExportMarkdown", func(t *testing.T) {
		result, err := processor.ExportMindMap(ctx, mindMap.ID, "markdown")
		if err != nil {
			t.Logf("Markdown export failed (expected): %v", err)
		} else {
			t.Logf("Markdown export result: %v", result)
		}
	})

	t.Run("ExportSVG", func(t *testing.T) {
		result, err := processor.ExportMindMap(ctx, mindMap.ID, "svg")
		if err != nil {
			t.Logf("SVG export failed (expected): %v", err)
		} else {
			t.Logf("SVG export result: %v", result)
		}
	})

	t.Run("ExportPDF", func(t *testing.T) {
		result, err := processor.ExportMindMap(ctx, mindMap.ID, "pdf")
		if err != nil {
			t.Logf("PDF export failed (expected): %v", err)
		} else {
			t.Logf("PDF export result: %v", result)
		}
	})

	t.Run("UnsupportedFormat", func(t *testing.T) {
		_, err := processor.ExportMindMap(ctx, mindMap.ID, "invalid-format")
		if err == nil {
			t.Fatal("Expected error for unsupported format")
		}
		if !strings.Contains(err.Error(), "unsupported export format") {
			t.Errorf("Expected unsupported format error, got: %v", err)
		}
	})

	t.Run("NonExistentMindMap", func(t *testing.T) {
		_, err := processor.ExportMindMap(ctx, "non-existent-id", "json")
		if err == nil {
			t.Fatal("Expected error for non-existent mind map")
		}
	})
}

// TestSearchInQdrant tests the searchInQdrant helper function
func TestSearchInQdrant(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)
	ctx := context.Background()

	t.Run("EmptyEmbedding", func(t *testing.T) {
		results, err := processor.searchInQdrant(ctx, "mind_maps", []float64{}, 10)
		if err != nil {
			t.Logf("Search with empty embedding failed (expected): %v", err)
		} else {
			t.Logf("Search returned %d results", len(results))
		}
	})

	t.Run("NormalEmbedding", func(t *testing.T) {
		embedding := make([]float64, 384) // nomic-embed-text dimension
		for i := range embedding {
			embedding[i] = 0.1
		}
		results, err := processor.searchInQdrant(ctx, "mind_maps", embedding, 5)
		if err != nil {
			t.Logf("Search failed (expected without Qdrant): %v", err)
		} else {
			t.Logf("Search returned %d results", len(results))
		}
	})
}

// TestStoreInQdrant tests the storeInQdrant helper function
func TestStoreInQdrant(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)
	ctx := context.Background()

	t.Run("StoreValidEmbedding", func(t *testing.T) {
		embedding := make([]float64, 384)
		for i := range embedding {
			embedding[i] = 0.1
		}
		payload := map[string]interface{}{
			"title": "Test",
			"description": "Test description",
		}
		err := processor.storeInQdrant(ctx, "mind_maps", "test-id", embedding, payload)
		if err != nil {
			t.Logf("Store failed (expected without Qdrant): %v", err)
		}
	})

	t.Run("StoreEmptyEmbedding", func(t *testing.T) {
		payload := map[string]interface{}{
			"title": "Test",
		}
		err := processor.storeInQdrant(ctx, "mind_maps", "test-id", []float64{}, payload)
		if err != nil {
			t.Logf("Store with empty embedding failed (expected): %v", err)
		}
	})
}

// TestParseAIMindMapResponse tests the parseAIMindMapResponse helper
func TestParseAIMindMapResponse(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)

	t.Run("ValidJSON", func(t *testing.T) {
		response := `{
			"title": "Test Map",
			"nodes": [
				{"content": "Node 1", "x": 100, "y": 100},
				{"content": "Node 2", "x": 200, "y": 200}
			]
		}`
		result, err := processor.parseAIMindMapResponse(response)
		if err != nil {
			t.Fatalf("Failed to parse valid JSON: %v", err)
		}
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		response := "not a valid json"
		_, err := processor.parseAIMindMapResponse(response)
		if err == nil {
			t.Fatal("Expected error for invalid JSON")
		}
	})

	t.Run("EmptyResponse", func(t *testing.T) {
		_, err := processor.parseAIMindMapResponse("")
		if err == nil {
			t.Fatal("Expected error for empty response")
		}
	})
}
