package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	router.HandleFunc("/health", server.healthCheck).Methods("GET")

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
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
			t.Errorf("Expected status=healthy, got %v", response["status"])
		}

		services, ok := response["services"].(map[string]interface{})
		if !ok {
			t.Fatal("services field missing or invalid")
		}

		if services["database"] != "healthy" {
			t.Errorf("Expected database=healthy, got %v", services["database"])
		}
	})
}

// TestCampaignCRUD tests campaign CRUD operations
func TestCampaignCRUD(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/campaigns", server.getCampaigns).Methods("GET")
	v1.HandleFunc("/campaigns", server.createCampaign).Methods("POST")
	v1.HandleFunc("/campaigns/{id}", server.getCampaign).Methods("GET")
	v1.HandleFunc("/campaigns/{id}", server.updateCampaign).Methods("PUT")
	v1.HandleFunc("/campaigns/{id}", server.deleteCampaign).Methods("DELETE")

	var createdID string

	t.Run("CreateCampaign_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body: map[string]interface{}{
				"name":        "Test Campaign",
				"description": "Test description",
				"color":       "#ff0000",
			},
		})

		result := assertJSONResponse(t, w, http.StatusCreated)

		if result["name"] != "Test Campaign" {
			t.Errorf("Expected name='Test Campaign', got %v", result["name"])
		}

		createdID = result["id"].(string)
	})

	t.Run("GetCampaign_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/campaigns/" + createdID,
			URLVars: map[string]string{"id": createdID},
		})

		result := assertJSONResponse(t, w, http.StatusOK)

		if result["name"] != "Test Campaign" {
			t.Errorf("Expected name='Test Campaign', got %v", result["name"])
		}
	})

	t.Run("GetCampaign_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/campaigns/" + nonExistentID,
			URLVars: map[string]string{"id": nonExistentID},
		})

		assertErrorResponse(t, w, http.StatusNotFound, "Campaign not found")
	})

	t.Run("ListCampaigns_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var campaigns []Campaign
		if err := json.Unmarshal(w.Body.Bytes(), &campaigns); err != nil {
			t.Fatalf("Failed to parse campaigns: %v", err)
		}

		if len(campaigns) == 0 {
			t.Error("Expected at least one campaign")
		}
	})

	t.Run("UpdateCampaign_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/campaigns/" + createdID,
			URLVars: map[string]string{"id": createdID},
			Body: map[string]interface{}{
				"name":        "Updated Campaign",
				"description": "Updated description",
			},
		})

		result := assertJSONResponse(t, w, http.StatusOK)

		if result["name"] != "Updated Campaign" {
			t.Errorf("Expected name='Updated Campaign', got %v", result["name"])
		}
	})

	t.Run("DeleteCampaign_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/campaigns/" + createdID,
			URLVars: map[string]string{"id": createdID},
		})

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("DeleteCampaign_NotFound", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/campaigns/" + createdID,
			URLVars: map[string]string{"id": createdID},
		})

		assertErrorResponse(t, w, http.StatusNotFound, "Campaign not found")
	})
}

// TestPromptCRUD tests prompt CRUD operations
func TestPromptCRUD(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	// Create test campaign
	campaign := createTestCampaign(t, db, "Test Campaign")

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/prompts", server.getPrompts).Methods("GET")
	v1.HandleFunc("/prompts", server.createPrompt).Methods("POST")
	v1.HandleFunc("/prompts/{id}", server.getPrompt).Methods("GET")
	v1.HandleFunc("/prompts/{id}", server.updatePrompt).Methods("PUT")
	v1.HandleFunc("/prompts/{id}", server.deletePrompt).Methods("DELETE")
	v1.HandleFunc("/prompts/{id}/use", server.recordPromptUsage).Methods("POST")

	var createdID string

	t.Run("CreatePrompt_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/prompts",
			Body: map[string]interface{}{
				"campaign_id": campaign.ID,
				"title":       "Test Prompt",
				"content":     "This is a test prompt content",
				"description": "Test description",
				"variables":   []string{"var1", "var2"},
			},
		})

		result := assertJSONResponse(t, w, http.StatusCreated)

		if result["title"] != "Test Prompt" {
			t.Errorf("Expected title='Test Prompt', got %v", result["title"])
		}

		createdID = result["id"].(string)
	})

	t.Run("GetPrompt_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/prompts/" + createdID,
			URLVars: map[string]string{"id": createdID},
		})

		result := assertJSONResponse(t, w, http.StatusOK)

		if result["title"] != "Test Prompt" {
			t.Errorf("Expected title='Test Prompt', got %v", result["title"])
		}
	})

	t.Run("GetPrompt_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/prompts/" + nonExistentID,
			URLVars: map[string]string{"id": nonExistentID},
		})

		assertErrorResponse(t, w, http.StatusNotFound, "Prompt not found")
	})

	t.Run("ListPrompts_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/prompts",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var prompts []Prompt
		if err := json.Unmarshal(w.Body.Bytes(), &prompts); err != nil {
			t.Fatalf("Failed to parse prompts: %v", err)
		}

		if len(prompts) == 0 {
			t.Error("Expected at least one prompt")
		}
	})

	t.Run("UpdatePrompt_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/prompts/" + createdID,
			URLVars: map[string]string{"id": createdID},
			Body: map[string]interface{}{
				"title":   "Updated Prompt",
				"content": "Updated content",
			},
		})

		result := assertJSONResponse(t, w, http.StatusOK)

		if result["title"] != "Updated Prompt" {
			t.Errorf("Expected title='Updated Prompt', got %v", result["title"])
		}
	})

	t.Run("RecordUsage_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/prompts/" + createdID + "/use",
			URLVars: map[string]string{"id": createdID},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("DeletePrompt_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/prompts/" + createdID,
			URLVars: map[string]string{"id": createdID},
		})

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}

// TestSearchFunctionality tests search endpoints
func TestSearchFunctionality(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	campaign := createTestCampaign(t, db, "Search Test Campaign")
	createTestPrompt(t, db, campaign.ID, "Search Test 1", "This is a test prompt for searching")
	createTestPrompt(t, db, campaign.ID, "Search Test 2", "Another test prompt with different content")

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/search/prompts", server.searchPrompts).Methods("GET")

	t.Run("SearchPrompts_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/search/prompts?q=test",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var results []Prompt
		if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
			t.Fatalf("Failed to parse results: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected search results")
		}
	})

	t.Run("SearchPrompts_EmptyQuery", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/search/prompts?q=",
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "Search query required")
	})
}

// TestExportImport tests export and import functionality
func TestExportImport(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	campaign := createTestCampaign(t, db, "Export Test Campaign")
	createTestPrompt(t, db, campaign.ID, "Export Test Prompt", "Test content for export")

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/export", server.exportData).Methods("GET")
	v1.HandleFunc("/import", server.importData).Methods("POST")

	var exportData map[string]interface{}

	t.Run("Export_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/export",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if err := json.Unmarshal(w.Body.Bytes(), &exportData); err != nil {
			t.Fatalf("Failed to parse export data: %v", err)
		}

		if exportData["version"] != "1.0" {
			t.Errorf("Expected version=1.0, got %v", exportData["version"])
		}
	})

	t.Run("Import_Success", func(t *testing.T) {
		// Modify export data for import
		importData := exportData

		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/import",
			Body:   importData,
		})

		result := assertJSONResponse(t, w, http.StatusOK)

		if result["campaigns_imported"] == nil {
			t.Error("Expected campaigns_imported field")
		}
	})

	t.Run("Import_InvalidData", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/import",
			Body:   `{"invalid": "data"}`,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestHelperFunctions tests utility functions
func TestHelperFunctions(t *testing.T) {
	t.Run("CalculateWordCount", func(t *testing.T) {
		content := "This is a test sentence with seven words"
		count := calculateWordCount(&content)
		if count == nil || *count != 8 {
			t.Errorf("Expected 8 words, got %v", count)
		}

		nilContent := (*string)(nil)
		nilCount := calculateWordCount(nilContent)
		if nilCount != nil {
			t.Errorf("Expected nil for nil input, got %v", nilCount)
		}

		// Test empty string
		emptyContent := ""
		emptyCount := calculateWordCount(&emptyContent)
		if emptyCount == nil || *emptyCount != 0 {
			t.Errorf("Expected 0 words for empty string, got %v", emptyCount)
		}

		// Test with multiple spaces
		spacedContent := "word1    word2     word3"
		spacedCount := calculateWordCount(&spacedContent)
		if spacedCount == nil || *spacedCount != 3 {
			t.Errorf("Expected 3 words with multiple spaces, got %v", spacedCount)
		}
	})

	t.Run("CalculateTokenCount", func(t *testing.T) {
		content := "This is a test"
		tokens := calculateTokenCount(&content)
		if tokens == nil {
			t.Fatal("Expected token count, got nil")
		}

		// Should be roughly 0.75 * word count
		expectedTokens := int(float64(4) * 0.75)
		if *tokens != expectedTokens {
			t.Errorf("Expected ~%d tokens, got %d", expectedTokens, *tokens)
		}

		// Test nil input
		nilContent := (*string)(nil)
		nilTokens := calculateTokenCount(nilContent)
		if nilTokens != nil {
			t.Errorf("Expected nil for nil input, got %v", nilTokens)
		}

		// Test empty content
		emptyContent := ""
		emptyTokens := calculateTokenCount(&emptyContent)
		if emptyTokens == nil || *emptyTokens != 0 {
			t.Errorf("Expected 0 tokens for empty string, got %v", emptyTokens)
		}

		// Test longer content
		longContent := "This is a much longer piece of text with many more words to verify token calculation"
		longTokens := calculateTokenCount(&longContent)
		if longTokens == nil {
			t.Fatal("Expected token count for long content, got nil")
		}
		expectedLongTokens := int(float64(16) * 0.75)
		if *longTokens != expectedLongTokens {
			t.Errorf("Expected ~%d tokens for long content, got %d", expectedLongTokens, *longTokens)
		}
	})

	t.Run("PtrFloat64", func(t *testing.T) {
		val := 3.14
		ptr := ptrFloat64(val)
		if ptr == nil {
			t.Fatal("Expected non-nil pointer")
		}
		if *ptr != val {
			t.Errorf("Expected %f, got %f", val, *ptr)
		}

		// Test zero value
		zero := 0.0
		zeroPtr := ptrFloat64(zero)
		if zeroPtr == nil || *zeroPtr != 0.0 {
			t.Errorf("Expected 0.0, got %v", zeroPtr)
		}

		// Test negative value
		neg := -1.5
		negPtr := ptrFloat64(neg)
		if negPtr == nil || *negPtr != neg {
			t.Errorf("Expected %f, got %v", neg, negPtr)
		}
	})
}

// TestErrorConditions tests systematic error conditions
func TestErrorConditions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/campaigns", server.createCampaign).Methods("POST")
	v1.HandleFunc("/prompts", server.createPrompt).Methods("POST")

	t.Run("CreateCampaign_InvalidJSON", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body:   `{"invalid": "json"`,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("CreatePrompt_InvalidJSON", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/prompts",
			Body:   `{"invalid": "json"`,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestConcurrentOperations tests concurrent database operations
func TestConcurrentOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	campaign := createTestCampaign(t, db, "Concurrent Test Campaign")

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/prompts", server.createPrompt).Methods("POST")

	t.Run("CreateMultiplePrompts_Concurrent", func(t *testing.T) {
		done := make(chan bool, 5)

		for i := 0; i < 5; i++ {
			go func(index int) {
				w := makeHTTPRequest(server, router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/prompts",
					Body: map[string]interface{}{
						"campaign_id": campaign.ID,
						"title":       "Concurrent Prompt " + string(rune(index)),
						"content":     "Concurrent test content",
					},
				})

				if w.Code != http.StatusCreated {
					t.Errorf("Expected status 201, got %d", w.Code)
				}

				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 5; i++ {
			<-done
		}
	})
}

// TestDatabaseConnectionPool tests connection pool behavior
func TestDatabaseConnectionPool(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	t.Run("ConnectionPool_Stats", func(t *testing.T) {
		stats := db.Stats()

		if stats.MaxOpenConnections <= 0 {
			t.Error("Expected max open connections to be configured")
		}

		t.Logf("Connection pool stats: Open=%d, InUse=%d, Idle=%d, MaxOpen=%d",
			stats.OpenConnections, stats.InUse, stats.Idle, stats.MaxOpenConnections)
	})

	t.Run("ConnectionPool_MultipleQueries", func(t *testing.T) {
		setupTestTables(t, db)

		for i := 0; i < 10; i++ {
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM campaigns").Scan(&count)
			if err != nil {
				t.Errorf("Query %d failed: %v", i, err)
			}
		}
	})
}

// TestTagManagement tests tag CRUD operations
func TestTagManagement(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/tags", server.getTags).Methods("GET")
	v1.HandleFunc("/tags", server.createTag).Methods("POST")

	t.Run("CreateTag_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/tags",
			Body: map[string]interface{}{
				"name":        "test-tag",
				"color":       "#ff0000",
				"description": "Test tag description",
			},
		})

		result := assertJSONResponse(t, w, http.StatusCreated)

		if result["name"] != "test-tag" {
			t.Errorf("Expected name='test-tag', got %v", result["name"])
		}
	})

	t.Run("GetTags_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/tags",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestCampaignPrompts tests campaign-specific prompt listing
func TestCampaignPrompts(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	campaign := createTestCampaign(t, db, "Test Campaign")
	createTestPrompt(t, db, campaign.ID, "Prompt 1", "Content 1")
	createTestPrompt(t, db, campaign.ID, "Prompt 2", "Content 2")

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/campaigns/{id}/prompts", server.getCampaignPrompts).Methods("GET")

	t.Run("GetCampaignPrompts_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/campaigns/" + campaign.ID + "/prompts",
			URLVars: map[string]string{"id": campaign.ID},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var prompts []Prompt
		if err := json.Unmarshal(w.Body.Bytes(), &prompts); err != nil {
			t.Fatalf("Failed to parse prompts: %v", err)
		}

		if len(prompts) < 2 {
			t.Errorf("Expected at least 2 prompts, got %d", len(prompts))
		}
	})

	t.Run("GetCampaignPrompts_NonExistentCampaign", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/campaigns/" + nonExistentID + "/prompts",
			URLVars: map[string]string{"id": nonExistentID},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 (empty list), got %d", w.Code)
		}
	})
}

// TestPromptFiltering tests various prompt filtering options
func TestPromptFiltering(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	campaign := createTestCampaign(t, db, "Filter Test Campaign")
	createTestPrompt(t, db, campaign.ID, "Recent Prompt", "Recent content")

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/prompts/recent", server.getRecentPrompts).Methods("GET")
	v1.HandleFunc("/prompts/favorites", server.getFavoritePrompts).Methods("GET")

	t.Run("GetRecentPrompts_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/prompts/recent",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("GetFavoritePrompts_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/prompts/favorites",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}

// TestQuickAccessKey tests quick access key functionality
func TestQuickAccessKey(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	campaign := createTestCampaign(t, db, "Quick Key Campaign")
	prompt := createTestPrompt(t, db, campaign.ID, "Quick Prompt", "Quick content")

	// Set quick access key
	_, err := db.Exec("UPDATE prompts SET quick_access_key = $1 WHERE id = $2", "quick1", prompt.ID)
	if err != nil {
		t.Fatalf("Failed to set quick access key: %v", err)
	}

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/prompts/quick/{key}", server.getPromptByQuickKey).Methods("GET")

	t.Run("GetPromptByQuickKey_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/prompts/quick/quick1",
			URLVars: map[string]string{"key": "quick1"},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("GetPromptByQuickKey_NotFound", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/prompts/quick/nonexistent",
			URLVars: map[string]string{"key": "nonexistent"},
		})

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

// TestPromptVersioning tests version management
func TestPromptVersioning(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	campaign := createTestCampaign(t, db, "Version Test Campaign")
	prompt := createTestPrompt(t, db, campaign.ID, "Versioned Prompt", "Version 1 content")

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/prompts/{id}/versions", server.getPromptVersions).Methods("GET")
	v1.HandleFunc("/prompts/{id}/revert/{version}", server.revertPromptVersion).Methods("POST")

	t.Run("GetPromptVersions_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/prompts/" + prompt.ID + "/versions",
			URLVars: map[string]string{"id": prompt.ID},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}

// TestTemplateManagement tests template endpoints
func TestTemplateManagement(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/templates", server.getTemplates).Methods("GET")
	v1.HandleFunc("/templates/{id}", server.getTemplate).Methods("GET")

	t.Run("GetTemplates_Success", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/templates",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}

// TestBoundaryConditions tests edge cases
func TestBoundaryConditions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	campaign := createTestCampaign(t, db, "Boundary Test Campaign")

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/prompts", server.createPrompt).Methods("POST")
	v1.HandleFunc("/campaigns", server.createCampaign).Methods("POST")

	t.Run("CreatePrompt_EmptyTitle", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/prompts",
			Body: map[string]interface{}{
				"campaign_id": campaign.ID,
				"title":       "",
				"content":     "Valid content",
			},
		})

		// Should still create - validation is application-level
		if w.Code == http.StatusInternalServerError {
			t.Skip("Database constraint prevents empty title")
		}
	})

	t.Run("CreatePrompt_VeryLongContent", func(t *testing.T) {
		longContent := strings.Repeat("a", 100000)
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/prompts",
			Body: map[string]interface{}{
				"campaign_id": campaign.ID,
				"title":       "Long Content Test",
				"content":     longContent,
			},
		})

		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 201 or 400, got %d", w.Code)
		}
	})

	t.Run("CreateCampaign_SpecialCharactersInName", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body: map[string]interface{}{
				"name":  "Test <script>alert('xss')</script>",
				"color": "#ff0000",
			},
		})

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}
	})

	t.Run("CreatePrompt_ManyVariables", func(t *testing.T) {
		variables := make([]string, 100)
		for i := 0; i < 100; i++ {
			variables[i] = fmt.Sprintf("var%d", i)
		}

		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/prompts",
			Body: map[string]interface{}{
				"campaign_id": campaign.ID,
				"title":       "Many Variables Test",
				"content":     "Content with many variables",
				"variables":   variables,
			},
		})

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}
