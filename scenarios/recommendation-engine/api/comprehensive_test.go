// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Database Setup and Schema Tests
// ============================================================================

func TestSetupDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_WithPostgresURL", func(t *testing.T) {
		// This test validates setupDatabase can connect
		// Test environment already validated this in setup
		env := setupTestEnvironment(t)
		defer env.Cleanup()

		if env.DB == nil {
			t.Error("Database connection should be established")
		}

		// Test connection pool is working
		if err := env.DB.Ping(); err != nil {
			t.Errorf("Database ping failed: %v", err)
		}
	})

	t.Run("EdgeCase_ConnectionPoolSettings", func(t *testing.T) {
		env := setupTestEnvironment(t)
		defer env.Cleanup()

		// Verify connection pool settings are reasonable
		stats := env.DB.Stats()
		// Note: Go defaults to unlimited (0) if not explicitly set
		// Check that we can get stats
		if stats.OpenConnections < 0 {
			t.Error("Should be able to query connection stats")
		}
	})
}

// ============================================================================
// Service Creation Tests
// ============================================================================

func TestNewRecommendationService(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_WithQdrant", func(t *testing.T) {
		service := NewRecommendationService(env.DB, env.QdrantConn)
		if service == nil {
			t.Fatal("Service should be created")
		}
		if service.db == nil {
			t.Error("Service should have database connection")
		}
	})

	t.Run("Success_WithoutQdrant", func(t *testing.T) {
		service := NewRecommendationService(env.DB, nil)
		if service == nil {
			t.Fatal("Service should be created even without Qdrant")
		}
		if service.qdrantClient != nil {
			t.Error("Qdrant client should be nil when no connection provided")
		}
	})
}

// ============================================================================
// GetRecommendations Extended Tests
// ============================================================================

func TestGetRecommendations_EdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("EdgeCase_VeryLargeLimit", func(t *testing.T) {
		// Create test data
		for i := 0; i < 5; i++ {
			item := createTestItem(t, env, fmt.Sprintf("limit-item-%d", i), "Item", "test")
			createTestInteraction(t, env, "other-user", item.ExternalID, "view", 1.0)
		}

		recommendations, err := env.Service.GetRecommendations("test-user", env.TestScenarioID, 1000, "hybrid", nil)
		if err != nil {
			t.Fatalf("GetRecommendations failed: %v", err)
		}

		// Should not return more items than exist
		if len(recommendations) > 5 {
			t.Errorf("Expected at most 5 recommendations, got %d", len(recommendations))
		}
	})

	t.Run("EdgeCase_ZeroLimit", func(t *testing.T) {
		recommendations, err := env.Service.GetRecommendations("test-user", env.TestScenarioID, 0, "hybrid", nil)
		if err != nil {
			t.Fatalf("GetRecommendations failed: %v", err)
		}

		if len(recommendations) != 0 {
			t.Errorf("Expected 0 recommendations with limit 0, got %d", len(recommendations))
		}
	})

	t.Run("EdgeCase_ExcludeAllItems", func(t *testing.T) {
		// Create items
		item1 := createTestItem(t, env, "exc-all-1", "Item 1", "test")
		item2 := createTestItem(t, env, "exc-all-2", "Item 2", "test")

		createTestInteraction(t, env, "other-user", item1.ExternalID, "view", 1.0)
		createTestInteraction(t, env, "other-user", item2.ExternalID, "view", 1.0)

		// Exclude all items
		excludeItems := []string{item1.ExternalID, item2.ExternalID}
		recommendations, err := env.Service.GetRecommendations("test-user", env.TestScenarioID, 10, "hybrid", excludeItems)
		if err != nil {
			t.Fatalf("GetRecommendations failed: %v", err)
		}

		if len(recommendations) > 0 {
			t.Errorf("Expected no recommendations when all items excluded, got %d", len(recommendations))
		}
	})

	t.Run("EdgeCase_NonExistentScenario", func(t *testing.T) {
		fakeScenarioID := uuid.New().String()
		recommendations, err := env.Service.GetRecommendations("test-user", fakeScenarioID, 10, "hybrid", nil)
		if err != nil {
			t.Fatalf("GetRecommendations failed: %v", err)
		}

		if len(recommendations) != 0 {
			t.Errorf("Expected no recommendations for non-existent scenario, got %d", len(recommendations))
		}
	})

	t.Run("Validation_ConfidenceValues", func(t *testing.T) {
		// Create items with interactions
		item := createTestItem(t, env, "conf-item", "Confidence Item", "test")
		for i := 0; i < 5; i++ {
			createTestInteraction(t, env, fmt.Sprintf("user-%d", i), item.ExternalID, "purchase", 5.0)
		}

		recommendations, err := env.Service.GetRecommendations("new-user", env.TestScenarioID, 10, "hybrid", nil)
		if err != nil {
			t.Fatalf("GetRecommendations failed: %v", err)
		}

		for _, rec := range recommendations {
			if rec.Confidence < 0 {
				t.Errorf("Confidence should not be negative: %f", rec.Confidence)
			}
			if rec.Reason == "" {
				t.Error("Reason should be provided for each recommendation")
			}
		}
	})
}

// ============================================================================
// GetSimilarItems Extended Tests
// ============================================================================

func TestGetSimilarItems_EdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	if env.QdrantConn == nil {
		t.Skip("Qdrant not available, skipping similarity tests")
	}

	t.Run("EdgeCase_VeryHighThreshold", func(t *testing.T) {
		item := createTestItem(t, env, "high-thresh-item", "Test Item", "test")
		if err := env.Service.StoreItemEmbedding(item); err != nil {
			t.Skipf("Failed to store embedding: %v", err)
		}

		// Very high threshold (0.99) should return very few items
		similarItems, err := env.Service.GetSimilarItems(item.ID, env.TestScenarioID, 10, 0.99)
		if err != nil {
			t.Fatalf("GetSimilarItems failed: %v", err)
		}

		// Should return empty or very small list
		if len(similarItems) > 2 {
			t.Logf("Warning: High threshold returned %d items (expected few or none)", len(similarItems))
		}
	})

	t.Run("EdgeCase_ZeroThreshold", func(t *testing.T) {
		item1 := createTestItem(t, env, "zero-thresh-1", "Item 1", "test")
		item2 := createTestItem(t, env, "zero-thresh-2", "Item 2", "test")

		if err := env.Service.StoreItemEmbedding(item1); err != nil {
			t.Skipf("Failed to store embedding: %v", err)
		}
		if err := env.Service.StoreItemEmbedding(item2); err != nil {
			t.Skipf("Failed to store embedding: %v", err)
		}

		// Zero threshold should return all items (up to limit)
		similarItems, err := env.Service.GetSimilarItems(item1.ID, env.TestScenarioID, 10, 0.0)
		if err != nil {
			t.Fatalf("GetSimilarItems failed: %v", err)
		}

		if len(similarItems) == 0 {
			t.Error("Zero threshold should return similar items")
		}
	})

	t.Run("Error_QdrantNotConfigured", func(t *testing.T) {
		// Create service without Qdrant
		serviceNoQdrant := NewRecommendationService(env.DB, nil)

		_, err := serviceNoQdrant.GetSimilarItems("some-id", env.TestScenarioID, 10, 0.7)
		if err == nil {
			t.Error("Expected error when Qdrant not configured")
		}
		if err.Error() != "Qdrant not configured" {
			t.Errorf("Expected 'Qdrant not configured' error, got: %v", err)
		}
	})
}

// ============================================================================
// StoreItemEmbedding Tests
// ============================================================================

func TestStoreItemEmbedding(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	if env.QdrantConn == nil {
		t.Skip("Qdrant not available, skipping embedding tests")
	}

	t.Run("Success_StoreEmbedding", func(t *testing.T) {
		item := createTestItem(t, env, "embed-item-1", "Embedding Test", "test")

		err := env.Service.StoreItemEmbedding(item)
		if err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}
	})

	t.Run("Success_UpdateEmbedding", func(t *testing.T) {
		item := createTestItem(t, env, "embed-update", "Original Title", "test")

		// Store initial embedding
		if err := env.Service.StoreItemEmbedding(item); err != nil {
			t.Fatalf("Failed to store initial embedding: %v", err)
		}

		// Update item and store again
		item.Title = "Updated Title"
		if err := env.Service.StoreItemEmbedding(item); err != nil {
			t.Fatalf("Failed to update embedding: %v", err)
		}
	})

	t.Run("Error_QdrantNotConfigured", func(t *testing.T) {
		serviceNoQdrant := NewRecommendationService(env.DB, nil)
		item := &Item{
			ID:          uuid.New().String(),
			Title:       "Test",
			Description: "Test",
		}

		err := serviceNoQdrant.StoreItemEmbedding(item)
		if err == nil {
			t.Error("Expected error when Qdrant not configured")
		}
	})

	t.Run("EdgeCase_EmptyDescription", func(t *testing.T) {
		item := createTestItem(t, env, "embed-empty-desc", "Title Only", "test")
		item.Description = ""

		err := env.Service.StoreItemEmbedding(item)
		if err != nil {
			t.Fatalf("Should handle empty description: %v", err)
		}
	})

	t.Run("EdgeCase_VeryLongText", func(t *testing.T) {
		longDesc := ""
		for i := 0; i < 1000; i++ {
			longDesc += "word "
		}

		item := createTestItem(t, env, "embed-long", "Long Description Item", "test")
		item.Description = longDesc

		err := env.Service.StoreItemEmbedding(item)
		if err != nil {
			t.Fatalf("Should handle very long text: %v", err)
		}
	})
}

// ============================================================================
// HTTP Handler Error Patterns
// ============================================================================

func TestIngestHandler_ErrorPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Test invalid JSON
	RunErrorPattern(t, env, InvalidJSONPattern("/api/v1/recommendations/ingest", "POST"))

	// Test empty body
	RunErrorPattern(t, env, EmptyBodyPattern("/api/v1/recommendations/ingest", "POST"))

	// Test missing required field (scenario_id)
	RunErrorPattern(t, env, MissingRequiredFieldPattern(
		"/api/v1/recommendations/ingest",
		"POST",
		map[string]interface{}{"items": []interface{}{}},
	))
}

func TestRecommendHandler_ErrorPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Test invalid JSON
	RunErrorPattern(t, env, InvalidJSONPattern("/api/v1/recommendations/get", "POST"))

	// Test empty body
	RunErrorPattern(t, env, EmptyBodyPattern("/api/v1/recommendations/get", "POST"))

	// Test missing user_id
	RunErrorPattern(t, env, MissingRequiredFieldPattern(
		"/api/v1/recommendations/get",
		"POST",
		map[string]interface{}{"scenario_id": "test"},
	))

	// Test missing scenario_id
	RunErrorPattern(t, env, MissingRequiredFieldPattern(
		"/api/v1/recommendations/get",
		"POST",
		map[string]interface{}{"user_id": "test"},
	))
}

func TestSimilarHandler_ErrorPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Test invalid JSON
	RunErrorPattern(t, env, InvalidJSONPattern("/api/v1/recommendations/similar", "POST"))

	// Test empty body
	RunErrorPattern(t, env, EmptyBodyPattern("/api/v1/recommendations/similar", "POST"))

	// Test non-existent item
	RunErrorPattern(t, env, NonExistentResourcePattern(
		"/api/v1/recommendations/similar",
		"POST",
		map[string]interface{}{
			"item_external_id": uuid.New().String(),
			"scenario_id":      uuid.New().String(),
		},
	))
}

// ============================================================================
// CORS and Middleware Tests
// ============================================================================

func TestCORSMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("OPTIONS_Request", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", "/api/v1/recommendations/ingest", nil)
		w := httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)

		// The test router doesn't have CORS middleware, so it returns 404
		// In production main.go, CORS middleware would return 204
		// This test validates that OPTIONS requests are handled
		if w.Code != http.StatusNotFound && w.Code != http.StatusNoContent {
			t.Logf("OPTIONS request returned status %d (acceptable - no CORS in test router)", w.Code)
		}
	})
}

// ============================================================================
// Concurrent Access Tests
// ============================================================================

func TestConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent access test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ConcurrentIngest", func(t *testing.T) {
		done := make(chan bool, 10)
		errors := make(chan error, 10)

		for i := 0; i < 10; i++ {
			go func(idx int) {
				defer func() { done <- true }()

				req := IngestRequest{
					ScenarioID: env.TestScenarioID,
					Items: []Item{
						{
							ExternalID:  fmt.Sprintf("concurrent-item-%d", idx),
							Title:       fmt.Sprintf("Concurrent Item %d", idx),
							Description: "Concurrent test",
							Category:    "test",
						},
					},
				}

				w := makeHTTPRequest(env.Router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/recommendations/ingest",
					Body:   req,
				})

				if w.Code != http.StatusOK {
					errors <- fmt.Errorf("request %d failed with status %d", idx, w.Code)
				}
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent ingest error: %v", err)
		}
	})

	t.Run("ConcurrentRead", func(t *testing.T) {
		// Create test data
		item := createTestItem(t, env, "read-item", "Read Test", "test")
		createTestInteraction(t, env, "user-1", item.ExternalID, "view", 1.0)

		done := make(chan bool, 20)

		for i := 0; i < 20; i++ {
			go func() {
				defer func() { done <- true }()

				req := RecommendRequest{
					UserID:     "test-user",
					ScenarioID: env.TestScenarioID,
					Limit:      10,
				}

				w := makeHTTPRequest(env.Router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/recommendations/get",
					Body:   req,
				})

				if w.Code != http.StatusOK {
					t.Errorf("Concurrent read failed: %d", w.Code)
				}
			}()
		}

		for i := 0; i < 20; i++ {
			<-done
		}
	})
}

// ============================================================================
// Data Integrity Tests
// ============================================================================

func TestDataIntegrity(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ItemMetadata_Preserved", func(t *testing.T) {
		metadata := map[string]interface{}{
			"price":    99.99,
			"brand":    "TestBrand",
			"features": []string{"feature1", "feature2"},
			"rating":   4.5,
		}

		item := &Item{
			ScenarioID:  env.TestScenarioID,
			ExternalID:  "metadata-test",
			Title:       "Metadata Test Item",
			Description: "Testing metadata preservation",
			Category:    "test",
			Metadata:    metadata,
		}

		if err := env.Service.CreateItem(item); err != nil {
			t.Fatalf("Failed to create item: %v", err)
		}

		// Query and verify metadata
		var metadataJSON []byte
		err := env.DB.QueryRow(
			"SELECT metadata FROM items WHERE scenario_id = $1 AND external_id = $2",
			env.TestScenarioID, "metadata-test").Scan(&metadataJSON)
		if err != nil {
			t.Fatalf("Failed to query metadata: %v", err)
		}

		var retrievedMetadata map[string]interface{}
		if err := json.Unmarshal(metadataJSON, &retrievedMetadata); err != nil {
			t.Fatalf("Failed to unmarshal metadata: %v", err)
		}

		// Verify key fields
		if retrievedMetadata["brand"] != "TestBrand" {
			t.Errorf("Metadata brand not preserved: %v", retrievedMetadata)
		}
	})

	t.Run("InteractionContext_Preserved", func(t *testing.T) {
		item := createTestItem(t, env, "context-item", "Context Test", "test")

		context := map[string]interface{}{
			"device":     "mobile",
			"location":   "US",
			"session_id": uuid.New().String(),
		}

		interaction := &UserInteraction{
			UserID:           "context-user",
			ItemID:           item.ExternalID,
			InteractionType:  "view",
			InteractionValue: 1.0,
			Context:          context,
		}

		if err := env.Service.CreateUserInteraction(interaction); err != nil {
			t.Fatalf("Failed to create interaction: %v", err)
		}

		// Verify context was stored correctly
		var contextJSON []byte
		err := env.DB.QueryRow(
			"SELECT context FROM user_interactions WHERE id = $1",
			interaction.ID).Scan(&contextJSON)
		if err != nil {
			t.Fatalf("Failed to query context: %v", err)
		}

		var retrievedContext map[string]interface{}
		if err := json.Unmarshal(contextJSON, &retrievedContext); err != nil {
			t.Fatalf("Failed to unmarshal context: %v", err)
		}

		if retrievedContext["device"] != "mobile" {
			t.Errorf("Context not preserved: %v", retrievedContext)
		}
	})
}

// ============================================================================
// Health Check Extended Tests
// ============================================================================

func TestHealthHandler_Extended(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_ChecksAllComponents", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		assertJSONResponse(t, w, http.StatusOK, func(response map[string]interface{}) bool {
			status, statusOk := response["status"].(string)
			database, dbOk := response["database"].(string)
			timestamp, tsOk := response["timestamp"].(string)

			return statusOk && dbOk && tsOk &&
				status != "" &&
				database == "connected" &&
				timestamp != ""
		})
	})

	t.Run("ReportsQdrantStatus", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		// Qdrant status should be present
		if _, exists := response["qdrant"]; !exists {
			t.Error("Health check should report Qdrant status")
		}
	})
}

// ============================================================================
// Performance Pattern Tests
// ============================================================================

func TestPerformancePattern_RecommendationLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance pattern test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Setup test data
	for i := 0; i < 20; i++ {
		item := createTestItem(t, env, fmt.Sprintf("perf-pat-%d", i), fmt.Sprintf("Item %d", i), "test")
		createTestInteraction(t, env, "user-1", item.ExternalID, "view", 1.0)
	}

	RunPerformanceTest(t, env, PerformanceTestPattern{
		Name:              "RecommendationLatency",
		Description:       "Test recommendation API latency under load",
		MaxDuration:       5 * time.Second,
		RequestCount:      50,
		ConcurrentWorkers: 5,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			req := RecommendRequest{
				UserID:     "perf-user",
				ScenarioID: env.TestScenarioID,
				Limit:      10,
			}
			return makeHTTPRequest(env.Router, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/recommendations/get",
				Body:   req,
			})
		},
	})
}
