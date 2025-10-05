// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Health Check Tests
// ============================================================================

func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		assertJSONResponse(t, w, http.StatusOK, func(response map[string]interface{}) bool {
			status, ok := response["status"].(string)
			return ok && (status == "healthy" || status == "unhealthy")
		})
	})
}

// ============================================================================
// Ingest Handler Tests
// ============================================================================

func TestIngestHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_ItemsOnly", func(t *testing.T) {
		req := IngestRequest{
			ScenarioID: env.TestScenarioID,
			Items: []Item{
				{
					ExternalID:  "item-1",
					Title:       "Test Item 1",
					Description: "Description for item 1",
					Category:    "electronics",
					Metadata:    map[string]interface{}{"price": 99.99},
				},
				{
					ExternalID:  "item-2",
					Title:       "Test Item 2",
					Description: "Description for item 2",
					Category:    "books",
					Metadata:    map[string]interface{}{"price": 19.99},
				},
			},
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/ingest",
			Body:   req,
		})

		assertJSONResponse(t, w, http.StatusOK, func(response map[string]interface{}) bool {
			itemsProcessed, ok := response["items_processed"].(float64)
			return ok && itemsProcessed == 2
		})
	})

	t.Run("Success_InteractionsOnly", func(t *testing.T) {
		// First create items
		item1 := createTestItem(t, env, "item-int-1", "Item for Interaction", "test")

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
					UserID:           "user-1",
					ItemExternalID:   item1.ExternalID,
					InteractionType:  "view",
					InteractionValue: nil,
				},
			},
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/ingest",
			Body:   req,
		})

		assertJSONResponse(t, w, http.StatusOK, func(response map[string]interface{}) bool {
			interactionsProcessed, ok := response["interactions_processed"].(float64)
			return ok && interactionsProcessed == 1
		})
	})

	t.Run("Success_ItemsAndInteractions", func(t *testing.T) {
		value := 4.5
		req := IngestRequest{
			ScenarioID: env.TestScenarioID,
			Items: []Item{
				{
					ExternalID:  "item-combo-1",
					Title:       "Combo Item",
					Description: "Item with interaction",
					Category:    "combo",
				},
			},
			Interactions: []struct {
				UserID             string                 `json:"user_id" binding:"required"`
				ItemExternalID     string                 `json:"item_external_id" binding:"required"`
				InteractionType    string                 `json:"interaction_type" binding:"required"`
				InteractionValue   *float64               `json:"interaction_value,omitempty"`
				Context            map[string]interface{} `json:"context,omitempty"`
			}{
				{
					UserID:           "user-combo",
					ItemExternalID:   "item-combo-1",
					InteractionType:  "purchase",
					InteractionValue: &value,
					Context:          map[string]interface{}{"source": "web"},
				},
			},
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/ingest",
			Body:   req,
		})

		assertJSONResponse(t, w, http.StatusOK, func(response map[string]interface{}) bool {
			itemsProcessed, ok1 := response["items_processed"].(float64)
			interactionsProcessed, ok2 := response["interactions_processed"].(float64)
			return ok1 && ok2 && itemsProcessed == 1 && interactionsProcessed == 1
		})
	})

	t.Run("Error_MissingScenarioID", func(t *testing.T) {
		req := map[string]interface{}{
			"items": []map[string]interface{}{
				{"external_id": "item-1", "title": "Test"},
			},
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/ingest",
			Body:   req,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "scenario_id")
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/recommendations/ingest",
			bytes.NewBufferString("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("Error_EmptyBody", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/recommendations/ingest",
			bytes.NewBufferString(""))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("EdgeCase_EmptyItemsArray", func(t *testing.T) {
		req := IngestRequest{
			ScenarioID: env.TestScenarioID,
			Items:      []Item{},
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/ingest",
			Body:   req,
		})

		assertJSONResponse(t, w, http.StatusOK, func(response map[string]interface{}) bool {
			itemsProcessed, ok := response["items_processed"].(float64)
			return ok && itemsProcessed == 0
		})
	})

	t.Run("EdgeCase_DuplicateItems", func(t *testing.T) {
		// Insert same item twice (should use upsert logic)
		externalID := uuid.New().String()
		req := IngestRequest{
			ScenarioID: env.TestScenarioID,
			Items: []Item{
				{
					ExternalID:  externalID,
					Title:       "Original Title",
					Description: "Original Description",
					Category:    "test",
				},
				{
					ExternalID:  externalID,
					Title:       "Updated Title",
					Description: "Updated Description",
					Category:    "test",
				},
			},
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/ingest",
			Body:   req,
		})

		assertJSONResponse(t, w, http.StatusOK, func(response map[string]interface{}) bool {
			// Should process both but only create one item (upsert)
			itemsProcessed, ok := response["items_processed"].(float64)
			return ok && itemsProcessed == 2
		})
	})
}

// ============================================================================
// Recommend Handler Tests
// ============================================================================

func TestRecommendHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_WithRecommendations", func(t *testing.T) {
		// Setup: Create items and interactions
		item1 := createTestItem(t, env, "rec-item-1", "Recommendation Item 1", "electronics")
		item2 := createTestItem(t, env, "rec-item-2", "Recommendation Item 2", "electronics")

		// Create user and interactions for other users
		createTestInteraction(t, env, "other-user-1", item1.ExternalID, "purchase", 5.0)
		createTestInteraction(t, env, "other-user-1", item2.ExternalID, "view", 1.0)
		createTestInteraction(t, env, "other-user-2", item2.ExternalID, "purchase", 5.0)

		req := RecommendRequest{
			UserID:     "target-user",
			ScenarioID: env.TestScenarioID,
			Limit:      10,
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/get",
			Body:   req,
		})

		assertJSONResponse(t, w, http.StatusOK, func(response map[string]interface{}) bool {
			recommendations, ok := response["recommendations"].([]interface{})
			algorithm, algOk := response["algorithm_used"].(string)
			return ok && algOk && algorithm != "" && recommendations != nil
		})
	})

	t.Run("Success_WithExcludeItems", func(t *testing.T) {
		item1 := createTestItem(t, env, "exc-item-1", "Exclude Item 1", "test")
		createTestInteraction(t, env, "other-user-3", item1.ExternalID, "view", 1.0)

		req := RecommendRequest{
			UserID:       "exclude-test-user",
			ScenarioID:   env.TestScenarioID,
			ExcludeItems: []string{item1.ExternalID},
			Limit:        5,
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/get",
			Body:   req,
		})

		assertJSONResponse(t, w, http.StatusOK, func(response map[string]interface{}) bool {
			recommendations, ok := response["recommendations"].([]interface{})
			if !ok {
				return false
			}
			// Verify excluded item is not in recommendations
			for _, rec := range recommendations {
				recMap := rec.(map[string]interface{})
				if recMap["external_id"] == item1.ExternalID {
					return false
				}
			}
			return true
		})
	})

	t.Run("Success_DefaultLimit", func(t *testing.T) {
		req := RecommendRequest{
			UserID:     "limit-test-user",
			ScenarioID: env.TestScenarioID,
			// No limit specified, should default to 10
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/get",
			Body:   req,
		})

		assertJSONResponse(t, w, http.StatusOK, func(response map[string]interface{}) bool {
			recommendations, ok := response["recommendations"].([]interface{})
			return ok && len(recommendations) <= 10
		})
	})

	t.Run("Success_CustomAlgorithm", func(t *testing.T) {
		req := RecommendRequest{
			UserID:     "algo-test-user",
			ScenarioID: env.TestScenarioID,
			Algorithm:  "collaborative",
			Limit:      5,
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/get",
			Body:   req,
		})

		assertJSONResponse(t, w, http.StatusOK, func(response map[string]interface{}) bool {
			algorithm, ok := response["algorithm_used"].(string)
			return ok && algorithm == "collaborative"
		})
	})

	t.Run("Error_MissingUserID", func(t *testing.T) {
		req := map[string]interface{}{
			"scenario_id": env.TestScenarioID,
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/get",
			Body:   req,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "user_id")
	})

	t.Run("Error_MissingScenarioID", func(t *testing.T) {
		req := map[string]interface{}{
			"user_id": "test-user",
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/get",
			Body:   req,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "scenario_id")
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/recommendations/get",
			bytes.NewBufferString("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("EdgeCase_NoData", func(t *testing.T) {
		// Request recommendations with no interactions in DB
		req := RecommendRequest{
			UserID:     "no-data-user",
			ScenarioID: uuid.New().String(),
			Limit:      10,
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/get",
			Body:   req,
		})

		assertJSONResponse(t, w, http.StatusOK, func(response map[string]interface{}) bool {
			recommendations, ok := response["recommendations"].([]interface{})
			return ok && recommendations != nil
		})
	})
}

// ============================================================================
// Similar Handler Tests
// ============================================================================

func TestSimilarHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Skip if Qdrant is not available
	if env.QdrantConn == nil {
		t.Skip("Qdrant not available, skipping similarity tests")
	}

	t.Run("Success_DefaultParameters", func(t *testing.T) {
		// Create and store item with embedding
		item := createTestItem(t, env, "sim-item-1", "Similarity Test Item", "test")
		if err := env.Service.StoreItemEmbedding(item); err != nil {
			t.Skipf("Failed to store embedding: %v", err)
		}

		req := SimilarRequest{
			ItemExternalID: item.ExternalID,
			ScenarioID:     env.TestScenarioID,
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

	t.Run("Success_CustomLimitAndThreshold", func(t *testing.T) {
		item := createTestItem(t, env, "sim-item-2", "Custom Params Item", "test")
		if err := env.Service.StoreItemEmbedding(item); err != nil {
			t.Skipf("Failed to store embedding: %v", err)
		}

		req := SimilarRequest{
			ItemExternalID: item.ExternalID,
			ScenarioID:     env.TestScenarioID,
			Limit:          5,
			Threshold:      0.8,
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/similar",
			Body:   req,
		})

		assertJSONResponse(t, w, http.StatusOK, func(response map[string]interface{}) bool {
			similarItems, ok := response["similar_items"].([]interface{})
			return ok && len(similarItems) <= 5
		})
	})

	t.Run("Error_ItemNotFound", func(t *testing.T) {
		req := SimilarRequest{
			ItemExternalID: "non-existent-item",
			ScenarioID:     env.TestScenarioID,
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/similar",
			Body:   req,
		})

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("Error_MissingItemExternalID", func(t *testing.T) {
		req := map[string]interface{}{
			"scenario_id": env.TestScenarioID,
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/similar",
			Body:   req,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "item_external_id")
	})

	t.Run("Error_MissingScenarioID", func(t *testing.T) {
		req := map[string]interface{}{
			"item_external_id": "test-item",
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/similar",
			Body:   req,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "scenario_id")
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/recommendations/similar",
			bytes.NewBufferString("{invalid}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

// ============================================================================
// Service Method Tests
// ============================================================================

func TestCreateItem(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		item := &Item{
			ScenarioID:  env.TestScenarioID,
			ExternalID:  "create-test-1",
			Title:       "Create Test Item",
			Description: "Test description",
			Category:    "test",
			Metadata:    map[string]interface{}{"key": "value"},
		}

		err := env.Service.CreateItem(item)
		if err != nil {
			t.Fatalf("Failed to create item: %v", err)
		}

		if item.ID == "" {
			t.Error("Item ID should be set after creation")
		}

		if item.CreatedAt.IsZero() {
			t.Error("CreatedAt should be set after creation")
		}
	})

	t.Run("Upsert_OnConflict", func(t *testing.T) {
		externalID := "upsert-test-1"
		item1 := &Item{
			ScenarioID:  env.TestScenarioID,
			ExternalID:  externalID,
			Title:       "Original Title",
			Description: "Original Description",
			Category:    "test",
		}

		err := env.Service.CreateItem(item1)
		if err != nil {
			t.Fatalf("Failed to create item: %v", err)
		}

		// Create same item with updated data
		item2 := &Item{
			ScenarioID:  env.TestScenarioID,
			ExternalID:  externalID,
			Title:       "Updated Title",
			Description: "Updated Description",
			Category:    "updated",
		}

		err = env.Service.CreateItem(item2)
		if err != nil {
			t.Fatalf("Failed to upsert item: %v", err)
		}

		// Verify item was updated
		var title string
		err = env.DB.QueryRow(
			"SELECT title FROM items WHERE scenario_id = $1 AND external_id = $2",
			env.TestScenarioID, externalID).Scan(&title)
		if err != nil {
			t.Fatalf("Failed to query item: %v", err)
		}

		if title != "Updated Title" {
			t.Errorf("Expected title 'Updated Title', got '%s'", title)
		}
	})
}

func TestCreateUserInteraction(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		item := createTestItem(t, env, "interaction-item-1", "Interaction Item", "test")

		interaction := &UserInteraction{
			UserID:           "interaction-user-1",
			ItemID:           item.ExternalID,
			InteractionType:  "view",
			InteractionValue: 1.0,
			Context:          map[string]interface{}{"device": "mobile"},
		}

		err := env.Service.CreateUserInteraction(interaction)
		if err != nil {
			t.Fatalf("Failed to create interaction: %v", err)
		}

		if interaction.ID == "" {
			t.Error("Interaction ID should be set")
		}

		if interaction.Timestamp.IsZero() {
			t.Error("Timestamp should be set")
		}
	})

	t.Run("Success_CreatesUser", func(t *testing.T) {
		item := createTestItem(t, env, "interaction-item-2", "Interaction Item 2", "test")
		newUserExtID := uuid.New().String()

		interaction := &UserInteraction{
			UserID:           newUserExtID,
			ItemID:           item.ExternalID,
			InteractionType:  "purchase",
			InteractionValue: 5.0,
		}

		err := env.Service.CreateUserInteraction(interaction)
		if err != nil {
			t.Fatalf("Failed to create interaction: %v", err)
		}

		// Verify user was created
		var userID string
		err = env.DB.QueryRow(
			"SELECT id FROM users WHERE external_id = $1",
			newUserExtID).Scan(&userID)
		if err != nil {
			t.Errorf("User should have been created: %v", err)
		}
	})
}

func TestGenerateEmbedding(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_GeneratesVector", func(t *testing.T) {
		text := "Test product for embedding generation"
		embedding := env.Service.generateEmbedding(text)

		if len(embedding) != 384 {
			t.Errorf("Expected embedding size 384, got %d", len(embedding))
		}

		// Check that embedding is not all zeros
		hasNonZero := false
		for _, val := range embedding {
			if val != 0 {
				hasNonZero = true
				break
			}
		}

		if !hasNonZero {
			t.Error("Embedding should not be all zeros")
		}
	})

	t.Run("Consistency", func(t *testing.T) {
		text := "Consistent embedding test"
		embedding1 := env.Service.generateEmbedding(text)
		embedding2 := env.Service.generateEmbedding(text)

		// Same text should produce same embedding
		for i := range embedding1 {
			if embedding1[i] != embedding2[i] {
				t.Errorf("Embeddings should be consistent for same text")
				break
			}
		}
	})

	t.Run("EdgeCase_EmptyString", func(t *testing.T) {
		embedding := env.Service.generateEmbedding("")
		if len(embedding) != 384 {
			t.Errorf("Expected embedding size 384 for empty string, got %d", len(embedding))
		}
	})

	t.Run("EdgeCase_VeryLongText", func(t *testing.T) {
		longText := ""
		for i := 0; i < 1000; i++ {
			longText += "word "
		}
		embedding := env.Service.generateEmbedding(longText)
		if len(embedding) != 384 {
			t.Errorf("Expected embedding size 384 for long text, got %d", len(embedding))
		}
	})
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestEndToEndRecommendationFlow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Complete_RecommendationWorkflow", func(t *testing.T) {
		// Step 1: Ingest items
		ingestReq := IngestRequest{
			ScenarioID: env.TestScenarioID,
			Items: []Item{
				{
					ExternalID:  "e2e-item-1",
					Title:       "Smartphone X",
					Description: "High-end smartphone with great camera",
					Category:    "electronics",
				},
				{
					ExternalID:  "e2e-item-2",
					Title:       "Smartphone Y",
					Description: "Budget smartphone with good battery",
					Category:    "electronics",
				},
				{
					ExternalID:  "e2e-item-3",
					Title:       "Laptop Pro",
					Description: "Professional laptop for developers",
					Category:    "electronics",
				},
			},
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/ingest",
			Body:   ingestReq,
		})

		if w.Code != http.StatusOK {
			t.Fatalf("Failed to ingest items: %s", w.Body.String())
		}

		// Step 2: Ingest interactions
		value := 5.0
		interactReq := IngestRequest{
			ScenarioID: env.TestScenarioID,
			Interactions: []struct {
				UserID             string                 `json:"user_id" binding:"required"`
				ItemExternalID     string                 `json:"item_external_id" binding:"required"`
				InteractionType    string                 `json:"interaction_type" binding:"required"`
				InteractionValue   *float64               `json:"interaction_value,omitempty"`
				Context            map[string]interface{} `json:"context,omitempty"`
			}{
				{
					UserID:           "e2e-user-1",
					ItemExternalID:   "e2e-item-1",
					InteractionType:  "purchase",
					InteractionValue: &value,
				},
				{
					UserID:           "e2e-user-2",
					ItemExternalID:   "e2e-item-2",
					InteractionType:  "view",
					InteractionValue: nil,
				},
			},
		}

		w = makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/ingest",
			Body:   interactReq,
		})

		if w.Code != http.StatusOK {
			t.Fatalf("Failed to ingest interactions: %s", w.Body.String())
		}

		// Step 3: Get recommendations
		recReq := RecommendRequest{
			UserID:     "e2e-user-3",
			ScenarioID: env.TestScenarioID,
			Limit:      5,
		}

		w = makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/get",
			Body:   recReq,
		})

		if w.Code != http.StatusOK {
			t.Fatalf("Failed to get recommendations: %s", w.Body.String())
		}

		var response RecommendResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse recommendation response: %v", err)
		}

		if response.AlgorithmUsed == "" {
			t.Error("Algorithm used should be specified")
		}

		if response.GeneratedAt.IsZero() {
			t.Error("Generated at timestamp should be set")
		}
	})
}

// ============================================================================
// Performance Tests
// ============================================================================

func TestPerformance_IngestHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("BulkIngest_100Items", func(t *testing.T) {
		items := make([]Item, 100)
		for i := 0; i < 100; i++ {
			items[i] = Item{
				ExternalID:  uuid.New().String(),
				Title:       fmt.Sprintf("Bulk Item %d", i),
				Description: "Bulk description",
				Category:    "bulk",
			}
		}

		req := IngestRequest{
			ScenarioID: env.TestScenarioID,
			Items:      items,
		}

		start := time.Now()
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recommendations/ingest",
			Body:   req,
		})
		duration := time.Since(start)

		if w.Code != http.StatusOK {
			t.Fatalf("Bulk ingest failed: %s", w.Body.String())
		}

		t.Logf("Bulk ingest of 100 items took: %v", duration)

		if duration > 10*time.Second {
			t.Errorf("Bulk ingest took too long: %v (expected < 10s)", duration)
		}
	})
}

func TestPerformance_RecommendHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Setup test data
	for i := 0; i < 10; i++ {
		item := createTestItem(t, env,
			fmt.Sprintf("perf-item-%d", i),
			fmt.Sprintf("Performance Item %d", i),
			"test")
		createTestInteraction(t, env, "perf-user-1", item.ExternalID, "view", 1.0)
	}

	t.Run("ConcurrentRecommendations", func(t *testing.T) {
		req := RecommendRequest{
			UserID:     "perf-target-user",
			ScenarioID: env.TestScenarioID,
			Limit:      10,
		}

		start := time.Now()

		// Make 10 concurrent recommendation requests
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				w := makeHTTPRequest(env.Router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/recommendations/get",
					Body:   req,
				})
				if w.Code != http.StatusOK {
					t.Errorf("Recommendation request failed: %s", w.Body.String())
				}
				done <- true
			}()
		}

		// Wait for all requests to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		duration := time.Since(start)
		t.Logf("10 concurrent recommendation requests took: %v", duration)

		if duration > 5*time.Second {
			t.Errorf("Concurrent recommendations took too long: %v (expected < 5s)", duration)
		}
	})
}
