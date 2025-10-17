// +build testing

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
)

// ============================================================================
// Qdrant Integration Tests (run when Qdrant is available)
// ============================================================================

func TestStoreItemEmbedding_WithQdrant(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	if env.QdrantConn == nil {
		t.Skip("Qdrant not available for integration tests")
	}

	t.Run("Success_StoresInQdrant", func(t *testing.T) {
		item := createTestItem(t, env, "qdrant-store-test", "Qdrant Storage Test", "electronics")

		err := env.Service.StoreItemEmbedding(item)
		if err != nil {
			t.Fatalf("Failed to store embedding in Qdrant: %v", err)
		}

		// Verify it was stored by searching for it
		limitUint := uint32(1)
		searchResponse, err := env.Service.pointsClient.Scroll(context.Background(), &pb.ScrollPoints{
			CollectionName: "item-embeddings",
			Filter: &pb.Filter{
				Must: []*pb.Condition{
					{
						ConditionOneOf: &pb.Condition_Field{
							Field: &pb.FieldCondition{
								Key: "item_id",
								Match: &pb.Match{
									MatchValue: &pb.Match_Keyword{Keyword: item.ID},
								},
							},
						},
					},
				},
			},
			Limit: &limitUint,
		})

		if err != nil {
			t.Fatalf("Failed to verify stored embedding: %v", err)
		}

		if len(searchResponse.Result) == 0 {
			t.Error("Embedding was not found in Qdrant")
		}
	})

	t.Run("Success_UpdateExistingEmbedding", func(t *testing.T) {
		item := createTestItem(t, env, "qdrant-update-test", "Original Title", "books")

		// Store initial embedding
		if err := env.Service.StoreItemEmbedding(item); err != nil {
			t.Fatalf("Failed to store initial embedding: %v", err)
		}

		// Update item and store new embedding
		item.Title = "Updated Title for Better Recommendations"
		item.Description = "Completely new description with different semantics"

		if err := env.Service.StoreItemEmbedding(item); err != nil {
			t.Fatalf("Failed to update embedding: %v", err)
		}

		// Verify updated embedding exists
		limitUint := uint32(1)
		searchResponse, err := env.Service.pointsClient.Scroll(context.Background(), &pb.ScrollPoints{
			CollectionName: "item-embeddings",
			Filter: &pb.Filter{
				Must: []*pb.Condition{
					{
						ConditionOneOf: &pb.Condition_Field{
							Field: &pb.FieldCondition{
								Key: "item_id",
								Match: &pb.Match{
									MatchValue: &pb.Match_Keyword{Keyword: item.ID},
								},
							},
						},
					},
				},
			},
			Limit: &limitUint,
		})

		if err != nil {
			t.Fatalf("Failed to retrieve updated embedding: %v", err)
		}

		if len(searchResponse.Result) > 0 {
			// Verify the title was updated in payload
			if searchResponse.Result[0].Payload["title"].GetStringValue() != "Updated Title for Better Recommendations" {
				t.Error("Embedding payload was not updated")
			}
		}
	})

	t.Run("Success_DifferentCategories", func(t *testing.T) {
		categories := []string{"electronics", "books", "clothing", "food", "toys"}

		for _, category := range categories {
			item := createTestItem(t, env, fmt.Sprintf("cat-%s", category), fmt.Sprintf("Item in %s", category), category)

			if err := env.Service.StoreItemEmbedding(item); err != nil {
				t.Errorf("Failed to store embedding for category %s: %v", category, err)
			}
		}
	})
}

func TestGetSimilarItems_WithQdrant(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	if env.QdrantConn == nil {
		t.Skip("Qdrant not available for integration tests")
	}

	t.Run("Success_FindsSimilarItems", func(t *testing.T) {
		// Create items with similar content
		item1 := createTestItem(t, env, "similar-phone-1", "Smartphone with great camera", "electronics")
		item2 := createTestItem(t, env, "similar-phone-2", "Mobile phone with excellent camera", "electronics")
		item3 := createTestItem(t, env, "different-book", "Programming book about Go language", "books")

		// Store embeddings
		for _, item := range []*Item{item1, item2, item3} {
			if err := env.Service.StoreItemEmbedding(item); err != nil {
				t.Fatalf("Failed to store embedding: %v", err)
			}
		}

		// Wait a bit for Qdrant to index
		time.Sleep(100 * time.Millisecond)

		// Find items similar to item1
		similarItems, err := env.Service.GetSimilarItems(item1.ID, env.TestScenarioID, 10, 0.0)
		if err != nil {
			t.Fatalf("GetSimilarItems failed: %v", err)
		}

		// Should find item2 (similar phone) but not item1 itself
		if len(similarItems) == 0 {
			t.Error("Should find at least one similar item")
		}

		// Verify item1 is not in its own similar items
		for _, similar := range similarItems {
			if similar.ItemID == item1.ID {
				t.Error("Item should not be similar to itself")
			}
		}
	})

	t.Run("Success_RespectsLimit", func(t *testing.T) {
		// Create multiple items
		baseItem := createTestItem(t, env, "base-item", "Base item for similarity", "test")
		if err := env.Service.StoreItemEmbedding(baseItem); err != nil {
			t.Fatalf("Failed to store base item embedding: %v", err)
		}

		for i := 0; i < 5; i++ {
			item := createTestItem(t, env, fmt.Sprintf("sim-limit-%d", i), "Similar item "+fmt.Sprint(i), "test")
			if err := env.Service.StoreItemEmbedding(item); err != nil {
				t.Fatalf("Failed to store embedding: %v", err)
			}
		}

		time.Sleep(100 * time.Millisecond)

		// Request only 2 similar items
		similarItems, err := env.Service.GetSimilarItems(baseItem.ID, env.TestScenarioID, 2, 0.0)
		if err != nil {
			t.Fatalf("GetSimilarItems failed: %v", err)
		}

		if len(similarItems) > 2 {
			t.Errorf("Expected at most 2 similar items, got %d", len(similarItems))
		}
	})

	t.Run("Success_RespectsThreshold", func(t *testing.T) {
		item := createTestItem(t, env, "threshold-test", "Threshold test item", "test")
		if err := env.Service.StoreItemEmbedding(item); err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		// Very high threshold should return fewer items
		similarItemsHigh, err := env.Service.GetSimilarItems(item.ID, env.TestScenarioID, 10, 0.95)
		if err != nil {
			t.Fatalf("GetSimilarItems with high threshold failed: %v", err)
		}

		// Low threshold should return more items
		similarItemsLow, err := env.Service.GetSimilarItems(item.ID, env.TestScenarioID, 10, 0.1)
		if err != nil {
			t.Fatalf("GetSimilarItems with low threshold failed: %v", err)
		}

		// Low threshold should return at least as many as high threshold
		if len(similarItemsLow) < len(similarItemsHigh) {
			t.Error("Lower threshold should return more or equal items")
		}
	})

	t.Run("Success_ReturnsScores", func(t *testing.T) {
		item := createTestItem(t, env, "score-test", "Score verification item", "test")
		if err := env.Service.StoreItemEmbedding(item); err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}

		// Create similar item
		similar := createTestItem(t, env, "score-similar", "Score verification similar", "test")
		if err := env.Service.StoreItemEmbedding(similar); err != nil {
			t.Fatalf("Failed to store similar embedding: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		similarItems, err := env.Service.GetSimilarItems(item.ID, env.TestScenarioID, 10, 0.0)
		if err != nil {
			t.Fatalf("GetSimilarItems failed: %v", err)
		}

		// Verify all items have valid similarity scores
		for _, similar := range similarItems {
			if similar.SimilarityScore < 0 || similar.SimilarityScore > 1 {
				t.Errorf("Invalid similarity score: %f (should be between 0 and 1)", similar.SimilarityScore)
			}

			if similar.Title == "" {
				t.Error("Similar item should have title")
			}

			if similar.Category == "" {
				t.Error("Similar item should have category")
			}
		}
	})

	t.Run("Error_ReferenceItemNotInQdrant", func(t *testing.T) {
		// Create item but don't store embedding
		item := createTestItem(t, env, "no-embedding", "Item without embedding", "test")

		_, err := env.Service.GetSimilarItems(item.ID, env.TestScenarioID, 10, 0.7)
		if err == nil {
			t.Error("Expected error when reference item not in Qdrant")
		}
	})

	t.Run("Success_CrossScenarioIsolation", func(t *testing.T) {
		// Create items in different scenarios
		item1 := createTestItem(t, env, "iso-item-1", "Isolation test item", "test")
		if err := env.Service.StoreItemEmbedding(item1); err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}

		// Create item in different scenario (simulated by changing scenario ID)
		differentScenarioID := uuid.New().String()
		item2 := &Item{
			ID:          uuid.New().String(),
			ScenarioID:  differentScenarioID,
			ExternalID:  "iso-item-2",
			Title:       "Isolation test item similar",
			Description: "Should not appear in cross-scenario search",
			Category:    "test",
			CreatedAt:   time.Now(),
		}

		// Store in database
		metadataJSON := []byte("{}")
		_, err := env.DB.Exec(`
			INSERT INTO items (id, scenario_id, external_id, title, description, category, metadata, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, item2.ID, item2.ScenarioID, item2.ExternalID, item2.Title, item2.Description, item2.Category, metadataJSON, item2.CreatedAt)
		if err != nil {
			t.Fatalf("Failed to insert item: %v", err)
		}

		if err := env.Service.StoreItemEmbedding(item2); err != nil {
			t.Fatalf("Failed to store embedding for different scenario: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		// Search for similar items in original scenario
		similarItems, err := env.Service.GetSimilarItems(item1.ID, env.TestScenarioID, 10, 0.0)
		if err != nil {
			t.Fatalf("GetSimilarItems failed: %v", err)
		}

		// Should not find item2 (different scenario)
		for _, similar := range similarItems {
			if similar.ItemID == item2.ID {
				t.Error("Should not find items from different scenario")
			}
		}
	})
}

func TestSimilarHandler_WithQdrant(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	if env.QdrantConn == nil {
		t.Skip("Qdrant not available for integration tests")
	}

	t.Run("Success_CompleteFlow", func(t *testing.T) {
		// Create and ingest items
		item := createTestItem(t, env, "handler-test-1", "Handler test item", "electronics")
		if err := env.Service.StoreItemEmbedding(item); err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}

		similar := createTestItem(t, env, "handler-test-2", "Handler similar item", "electronics")
		if err := env.Service.StoreItemEmbedding(similar); err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		// Make HTTP request
		req := SimilarRequest{
			ItemExternalID: item.ExternalID,
			ScenarioID:     env.TestScenarioID,
			Limit:          5,
			Threshold:      0.0,
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/similar",
			Body:   req,
		})

		assertJSONResponse(t, w, http.StatusOK, func(response map[string]interface{}) bool {
			similarItems, ok := response["similar_items"].([]interface{})
			return ok && similarItems != nil
		})
	})

	t.Run("Success_CustomParameters", func(t *testing.T) {
		item := createTestItem(t, env, "custom-params", "Custom parameters test", "test")
		if err := env.Service.StoreItemEmbedding(item); err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		req := SimilarRequest{
			ItemExternalID: item.ExternalID,
			ScenarioID:     env.TestScenarioID,
			Limit:          3,
			Threshold:      0.5,
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/similar",
			Body:   req,
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Success_DefaultParameters", func(t *testing.T) {
		item := createTestItem(t, env, "default-params", "Default parameters test", "test")
		if err := env.Service.StoreItemEmbedding(item); err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		// Don't specify limit or threshold - should use defaults
		req := SimilarRequest{
			ItemExternalID: item.ExternalID,
			ScenarioID:     env.TestScenarioID,
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/similar",
			Body:   req,
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// ============================================================================
// Health Check Qdrant Status Tests
// ============================================================================

func TestHealthHandler_QdrantStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("WithQdrant", func(t *testing.T) {
		if env.QdrantConn == nil {
			t.Skip("Qdrant not available")
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		qdrantStatus, ok := response["qdrant"].(string)
		if !ok {
			t.Error("Response should include qdrant status")
		}

		if qdrantStatus != "connected" && qdrantStatus != "not_configured" {
			t.Logf("Qdrant status: %s", qdrantStatus)
		}
	})

	t.Run("WithoutQdrant", func(t *testing.T) {
		// Create service without Qdrant
		serviceNoQdrant := NewRecommendationService(env.DB, nil)
		router := setupTestRouter(serviceNoQdrant)

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		qdrantStatus := response["qdrant"].(string)
		if qdrantStatus != "not_configured" {
			t.Errorf("Expected 'not_configured', got '%s'", qdrantStatus)
		}
	})
}
