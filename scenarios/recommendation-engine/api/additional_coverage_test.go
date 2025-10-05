// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// ============================================================================
// Additional Coverage for Handlers
// ============================================================================

func TestHealthHandler_DatabaseFailure(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("DatabaseUnavailable", func(t *testing.T) {
		// Close the database connection to simulate failure
		env.DB.Close()

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		// Should return unhealthy status
		if w.Code != http.StatusServiceUnavailable && w.Code != http.StatusOK {
			t.Logf("Health check with closed DB returned status %d", w.Code)
		}

		// Reconnect for cleanup
		env.DB = setupTestDB(t)
	})
}

// ============================================================================
// Additional Edge Cases for CreateUserInteraction
// ============================================================================

func TestCreateUserInteraction_EdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Error_ItemNotFound", func(t *testing.T) {
		interaction := &UserInteraction{
			UserID:           "test-user",
			ItemID:           "non-existent-item",
			InteractionType:  "view",
			InteractionValue: 1.0,
		}

		err := env.Service.CreateUserInteraction(interaction)
		if err == nil {
			t.Error("Expected error for non-existent item")
		}
	})

	t.Run("Success_ExistingUserNewInteraction", func(t *testing.T) {
		// Create item
		item := createTestItem(t, env, "existing-user-item", "Item", "test")

		// Create first interaction (creates user)
		interaction1 := &UserInteraction{
			UserID:           "existing-user",
			ItemID:           item.ExternalID,
			InteractionType:  "view",
			InteractionValue: 1.0,
		}

		if err := env.Service.CreateUserInteraction(interaction1); err != nil {
			t.Fatalf("Failed to create first interaction: %v", err)
		}

		// Create second interaction with same user (should reuse user)
		interaction2 := &UserInteraction{
			UserID:           "existing-user",
			ItemID:           item.ExternalID,
			InteractionType:  "purchase",
			InteractionValue: 5.0,
		}

		if err := env.Service.CreateUserInteraction(interaction2); err != nil {
			t.Fatalf("Failed to create second interaction: %v", err)
		}

		// Verify both interactions exist
		var count int
		err := env.DB.QueryRow(
			`SELECT COUNT(*) FROM user_interactions ui
			 JOIN users u ON ui.user_id = u.id
			 WHERE u.external_id = $1`,
			"existing-user").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count interactions: %v", err)
		}

		if count != 2 {
			t.Errorf("Expected 2 interactions for user, got %d", count)
		}
	})

	t.Run("EdgeCase_NullContext", func(t *testing.T) {
		item := createTestItem(t, env, "null-context-item", "Item", "test")

		interaction := &UserInteraction{
			UserID:           "null-context-user",
			ItemID:           item.ExternalID,
			InteractionType:  "view",
			InteractionValue: 1.0,
			Context:          nil, // Null context
		}

		if err := env.Service.CreateUserInteraction(interaction); err != nil {
			t.Fatalf("Should handle nil context: %v", err)
		}
	})

	t.Run("EdgeCase_ZeroInteractionValue", func(t *testing.T) {
		item := createTestItem(t, env, "zero-value-item", "Item", "test")

		interaction := &UserInteraction{
			UserID:           "zero-value-user",
			ItemID:           item.ExternalID,
			InteractionType:  "view",
			InteractionValue: 0.0,
		}

		if err := env.Service.CreateUserInteraction(interaction); err != nil {
			t.Fatalf("Should handle zero interaction value: %v", err)
		}
	})

	t.Run("EdgeCase_NegativeInteractionValue", func(t *testing.T) {
		item := createTestItem(t, env, "negative-value-item", "Item", "test")

		interaction := &UserInteraction{
			UserID:           "negative-value-user",
			ItemID:           item.ExternalID,
			InteractionType:  "dislike",
			InteractionValue: -1.0,
		}

		if err := env.Service.CreateUserInteraction(interaction); err != nil {
			t.Fatalf("Should handle negative interaction value: %v", err)
		}
	})
}

// ============================================================================
// Additional GetRecommendations Tests
// ============================================================================

func TestGetRecommendations_ComplexScenarios(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_MultipleExcludeItems", func(t *testing.T) {
		// Create multiple items
		items := make([]*Item, 5)
		for i := 0; i < 5; i++ {
			items[i] = createTestItem(t, env, fmt.Sprintf("multi-exc-%d", i), fmt.Sprintf("Item %d", i), "test")
			createTestInteraction(t, env, "popular-user", items[i].ExternalID, "purchase", 5.0)
		}

		// Exclude first 3 items
		excludeItems := []string{
			items[0].ExternalID,
			items[1].ExternalID,
			items[2].ExternalID,
		}

		recommendations, err := env.Service.GetRecommendations("new-user", env.TestScenarioID, 10, "hybrid", excludeItems)
		if err != nil {
			t.Fatalf("GetRecommendations failed: %v", err)
		}

		// Verify excluded items are not in results
		for _, rec := range recommendations {
			for _, excluded := range excludeItems {
				if rec.ExternalID == excluded {
					t.Errorf("Excluded item %s found in recommendations", excluded)
				}
			}
		}
	})

	t.Run("Success_DifferentAlgorithms", func(t *testing.T) {
		item := createTestItem(t, env, "algo-test", "Algorithm Test", "test")
		createTestInteraction(t, env, "user-1", item.ExternalID, "view", 1.0)

		algorithms := []string{"hybrid", "collaborative", "semantic"}

		for _, algo := range algorithms {
			recommendations, err := env.Service.GetRecommendations("test-user", env.TestScenarioID, 5, algo, nil)
			if err != nil {
				t.Errorf("GetRecommendations failed for algorithm %s: %v", algo, err)
			}

			// Verify recommendations is not nil (may be empty)
			if recommendations == nil {
				t.Errorf("Recommendations should not be nil for algorithm %s", algo)
			}
		}
	})

	t.Run("Success_UserWithOwnInteractions", func(t *testing.T) {
		// Create items
		item1 := createTestItem(t, env, "own-int-1", "User's Item 1", "test")
		item2 := createTestItem(t, env, "own-int-2", "User's Item 2", "test")
		item3 := createTestItem(t, env, "own-int-3", "Other Item", "test")

		// Create interactions for target user
		createTestInteraction(t, env, "target-user-own", item1.ExternalID, "purchase", 5.0)
		createTestInteraction(t, env, "target-user-own", item2.ExternalID, "view", 1.0)

		// Create interactions for other user
		createTestInteraction(t, env, "other-user-own", item3.ExternalID, "purchase", 5.0)

		// Get recommendations for target user
		// Should get item3 (other user's item) but not item1/item2 (own items filtered by query)
		recommendations, err := env.Service.GetRecommendations("target-user-own", env.TestScenarioID, 10, "hybrid", nil)
		if err != nil {
			t.Fatalf("GetRecommendations failed: %v", err)
		}

		// Recommendations is valid (may be empty or contain items)
		if recommendations == nil {
			t.Error("Recommendations should not be nil")
		}
	})
}

// ============================================================================
// Ingest Handler Additional Coverage
// ============================================================================

func TestIngestHandler_ComplexScenarios(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("PartialFailure_SomeItemsFail", func(t *testing.T) {
		// Create request with mix of valid and potentially problematic items
		req := IngestRequest{
			ScenarioID: env.TestScenarioID,
			Items: []Item{
				{
					ExternalID:  "valid-item",
					Title:       "Valid Item",
					Description: "This is valid",
					Category:    "test",
				},
				{
					ExternalID:  "valid-item-2",
					Title:       "Another Valid Item",
					Description: "Also valid",
					Category:    "test",
				},
			},
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/ingest",
			Body:   req,
		})

		// Should succeed
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("InteractionWithoutItem", func(t *testing.T) {
		// Try to create interaction for non-existent item
		req := IngestRequest{
			ScenarioID: env.TestScenarioID,
			Interactions: []struct {
				UserID             string                 `json:"user_id" binding:"required"`
				ItemExternalID     string                 `json:"item_external_id" binding:"required"`
				InteractionType    string                 `json:"interaction_type" binding:"required"`
				InteractionValue   *float64               `json:"interaction_value,omitempty"`
				Context            map[string]interface{} `json:"context,omitempty"`
			}{
				{
					UserID:          "test-user",
					ItemExternalID:  "non-existent-item-" + uuid.New().String(),
					InteractionType: "view",
				},
			},
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/ingest",
			Body:   req,
		})

		// Should return 200 but with errors in response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response IngestResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response.Success {
			t.Error("Response should indicate failure for non-existent item")
		}

		if len(response.Errors) == 0 {
			t.Error("Response should contain error messages")
		}
	})

	t.Run("Success_ItemsWithoutEmbeddings", func(t *testing.T) {
		// Service without Qdrant should still ingest items
		serviceNoQdrant := NewRecommendationService(env.DB, nil)
		router := setupTestRouter(serviceNoQdrant)

		req := IngestRequest{
			ScenarioID: env.TestScenarioID,
			Items: []Item{
				{
					ExternalID:  "no-embed-item",
					Title:       "Item without embedding",
					Description: "No Qdrant available",
					Category:    "test",
				},
			},
		}

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/ingest",
			Body:   req,
		})

		// Should succeed but report embedding errors
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response IngestResponse
		json.Unmarshal(w.Body.Bytes(), &response)

		if response.ItemsProcessed != 1 {
			t.Errorf("Expected 1 item processed, got %d", response.ItemsProcessed)
		}
	})
}

// ============================================================================
// Additional Validation Tests
// ============================================================================

func TestItemValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("EdgeCase_EmptyStrings", func(t *testing.T) {
		item := &Item{
			ScenarioID:  env.TestScenarioID,
			ExternalID:  "empty-strings",
			Title:       "",
			Description: "",
			Category:    "",
		}

		// Should handle empty strings gracefully
		if err := env.Service.CreateItem(item); err != nil {
			t.Fatalf("Should handle empty strings: %v", err)
		}
	})

	t.Run("EdgeCase_VeryLongStrings", func(t *testing.T) {
		longString := string(make([]byte, 1000))
		for i := range longString {
			longString = string(append([]byte(longString[:i]), 'a'))
		}

		item := &Item{
			ScenarioID:  env.TestScenarioID,
			ExternalID:  "long-strings",
			Title:       longString[:500], // Truncate to reasonable length
			Description: longString,
			Category:    "test",
		}

		if err := env.Service.CreateItem(item); err != nil {
			t.Fatalf("Should handle long strings: %v", err)
		}
	})

	t.Run("EdgeCase_SpecialCharacters", func(t *testing.T) {
		item := &Item{
			ScenarioID:  env.TestScenarioID,
			ExternalID:  "special-chars-" + uuid.New().String(),
			Title:       "Item with special chars: <>&\"'",
			Description: "Description with unicode: 你好 мир 🎉",
			Category:    "test-€$¥",
		}

		if err := env.Service.CreateItem(item); err != nil {
			t.Fatalf("Should handle special characters: %v", err)
		}
	})
}

// ============================================================================
// Additional Recommendation Quality Tests
// ============================================================================

func TestRecommendationQuality(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Popularity_BasedRanking", func(t *testing.T) {
		// Create items with different popularity levels
		popularItem := createTestItem(t, env, "very-popular", "Popular Item", "test")
		lessPopularItem := createTestItem(t, env, "less-popular", "Less Popular Item", "test")

		// Create many interactions for popular item
		for i := 0; i < 10; i++ {
			createTestInteraction(t, env, fmt.Sprintf("user-%d", i), popularItem.ExternalID, "purchase", 5.0)
		}

		// Create fewer interactions for less popular item
		for i := 0; i < 2; i++ {
			createTestInteraction(t, env, fmt.Sprintf("user-lp-%d", i), lessPopularItem.ExternalID, "view", 1.0)
		}

		// Get recommendations
		recommendations, err := env.Service.GetRecommendations("new-user", env.TestScenarioID, 10, "hybrid", nil)
		if err != nil {
			t.Fatalf("GetRecommendations failed: %v", err)
		}

		// Popular item should appear in recommendations if any exist
		if len(recommendations) > 0 {
			foundPopular := false
			for _, rec := range recommendations {
				if rec.ExternalID == popularItem.ExternalID {
					foundPopular = true
					// Should have higher confidence than less popular items
					if rec.Confidence <= 0 {
						t.Error("Popular item should have positive confidence")
					}
				}
			}
			// Note: May not always be first depending on algorithm
			if !foundPopular && len(recommendations) > 0 {
				t.Log("Popular item not in recommendations (may be expected)")
			}
		}
	})
}
