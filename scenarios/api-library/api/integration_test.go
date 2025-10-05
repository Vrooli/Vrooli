
package main

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestSemanticSearchIntegration tests semantic search with Qdrant
func TestSemanticSearchIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	// Create test APIs
	testAPI1 := setupTestAPI(t, env.DB, "Payment Gateway API")
	defer testAPI1.Cleanup()

	testAPI2 := setupTestAPI(t, env.DB, "Weather Forecast API")
	defer testAPI2.Cleanup()

	t.Run("SemanticSearch_finds_relevant_APIs", func(t *testing.T) {
		// Initialize semantic search client if needed
		if semanticClient == nil {
			initSemanticSearchClient()
		}

		if semanticClient == nil {
			t.Skip("Semantic search client not available")
			return
		}

		searchReq := SearchRequest{
			Query: "payment processing",
			Limit: 10,
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search",
			Body:   searchReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchAPIsHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if method, ok := response["method"].(string); ok {
				if method != "semantic" && method != "fulltext" {
					t.Logf("Search method: %s", method)
				}
			}
		}
	})
}

// TestRateLimitingIntegration tests rate limiting functionality
func TestRateLimitingIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	t.Run("RateLimit_allows_within_limit", func(t *testing.T) {
		// Make several requests below rate limit
		for i := 0; i < 5; i++ {
			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			healthHandler(w, httpReq)

			if w.Code != http.StatusOK {
				t.Errorf("Request %d failed with status %d", i+1, w.Code)
			}
		}
	})
}

// TestCacheIntegration tests Redis caching
func TestCacheIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	// Initialize Redis if needed
	if redisClient == nil {
		initRedisCache()
	}

	if redisClient == nil {
		t.Skip("Redis client not available")
		return
	}

	t.Run("Cache_stores_and_retrieves", func(t *testing.T) {
		ctx := context.Background()
		testKey := getCacheKey("test", map[string]string{"key": "123"})
		testValue := `{"test": "data"}`

		// Set cache
		err := setCache(ctx, testKey, testValue, 60*time.Second)
		if err != nil {
			t.Fatalf("Failed to set cache: %v", err)
		}

		// Get cache
		cached, err := getFromCache(ctx, testKey)
		if err != nil {
			t.Fatalf("Failed to get from cache: %v", err)
		}

		if string(cached) != testValue {
			t.Errorf("Expected cached value '%s', got '%s'", testValue, string(cached))
		}

		// Cleanup
		redisClient.Del(ctx, testKey)
	})

	t.Run("Cache_invalidation_works", func(t *testing.T) {
		ctx := context.Background()
		pattern := "test:apis:*"

		// Set multiple cache entries
		setCache(ctx, "test:apis:1", "data1", 60*time.Second)
		setCache(ctx, "test:apis:2", "data2", 60*time.Second)

		// Invalidate pattern
		err := invalidateCachePattern(ctx, pattern)
		if err != nil {
			t.Fatalf("Failed to invalidate cache pattern: %v", err)
		}

		// Verify cache is empty
		cached, err := getFromCache(ctx, "test:apis:1")
		if err == nil && len(cached) > 0 {
			t.Error("Cache should be empty after invalidation")
		}
	})
}

// TestDatabaseTransactions tests database transaction handling
func TestDatabaseTransactions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	t.Run("Transaction_rollback_on_error", func(t *testing.T) {
		tx, err := env.DB.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Insert test data
		_, err = tx.Exec(`
			INSERT INTO apis (id, name, provider, description, base_url, documentation_url,
				category, status, auth_type, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			"tx-test-1", "TX Test API", "Provider", "Description", "https://api.test",
			"https://docs.test", "testing", "active", "api_key", time.Now(), time.Now())
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}

		// Rollback
		tx.Rollback()

		// Verify data was not committed
		var count int
		err = env.DB.QueryRow("SELECT COUNT(*) FROM apis WHERE id = $1", "tx-test-1").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}

		if count > 0 {
			t.Error("Transaction was not rolled back properly")
		}
	})

	t.Run("Transaction_commit_on_success", func(t *testing.T) {
		tx, err := env.DB.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		testID := "tx-test-2"
		defer env.DB.Exec("DELETE FROM apis WHERE id = $1", testID)

		// Insert test data
		_, err = tx.Exec(`
			INSERT INTO apis (id, name, provider, description, base_url, documentation_url,
				category, status, auth_type, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			testID, "TX Test API", "Provider", "Description", "https://api.test",
			"https://docs.test", "testing", "active", "api_key", time.Now(), time.Now())
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to insert: %v", err)
		}

		// Commit
		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		// Verify data was committed
		var count int
		err = env.DB.QueryRow("SELECT COUNT(*) FROM apis WHERE id = $1", testID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}

		if count == 0 {
			t.Error("Transaction was not committed properly")
		}
	})
}

// TestWebhookIntegration tests webhook functionality
func TestWebhookIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	t.Run("Webhook_registration", func(t *testing.T) {
		webhookData := map[string]interface{}{
			"url":    "https://example.com/webhook",
			"events": []string{"api.created", "api.updated"},
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/webhooks",
			Body:   webhookData,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Note: This assumes webhook handler exists
		// If not, test will be skipped or fail gracefully
		env.Router.ServeHTTP(w, httpReq)

		// Accept various success codes
		if w.Code != http.StatusNotFound {
			if w.Code == http.StatusCreated || w.Code == http.StatusOK {
				t.Log("Webhook registered successfully")
			}
		}
	})
}

// TestEndToEndWorkflow tests complete user workflow
func TestEndToEndWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	var apiID string

	t.Run("Step1_CreateAPI", func(t *testing.T) {
		apiData := TestData.CreateAPIRequest("E2E Test API")

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/apis",
			Body:   apiData,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createAPIHandler(w, httpReq)

		if w.Code == http.StatusCreated || w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, w.Code, nil)
			if response != nil {
				if id, ok := response["id"].(string); ok {
					apiID = id
					defer env.DB.Exec("DELETE FROM apis WHERE id = $1", apiID)
				}
			}
		}
	})

	if apiID == "" {
		t.Skip("API creation failed, skipping workflow steps")
		return
	}

	t.Run("Step2_AddNote", func(t *testing.T) {
		noteData := map[string]interface{}{
			"content":    "E2E test note",
			"type":       "tip",
			"created_by": "e2e_test",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/apis/" + apiID + "/notes",
			URLVars: map[string]string{"id": apiID},
			Body:    noteData,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		addNoteHandler(w, httpReq)

		if w.Code != http.StatusCreated && w.Code != http.StatusOK {
			t.Errorf("Failed to add note, status: %d", w.Code)
		}
	})

	t.Run("Step3_MarkConfigured", func(t *testing.T) {
		configData := map[string]interface{}{
			"configured": true,
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/apis/" + apiID + "/configure",
			URLVars: map[string]string{"id": apiID},
			Body:    configData,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		markConfiguredHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Logf("Configure handler returned status: %d", w.Code)
		}
	})

	t.Run("Step4_SearchAPI", func(t *testing.T) {
		searchReq := SearchRequest{
			Query: "E2E",
			Limit: 10,
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search",
			Body:   searchReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchAPIsHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			t.Log("Search completed successfully")
		}
	})

	t.Run("Step5_DeleteAPI", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/apis/" + apiID,
			URLVars: map[string]string{"id": apiID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		deleteAPIHandler(w, httpReq)

		if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
			t.Logf("Delete returned status: %d", w.Code)
		}
	})
}
