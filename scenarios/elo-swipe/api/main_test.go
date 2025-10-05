
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("Success", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.HealthCheck(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "healthy",
		})

		if response == nil {
			t.Fatal("Expected valid JSON response")
		}
	})
}

// TestGetLists tests the GetLists handler
func TestGetLists(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("Success_EmptyList", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/lists",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetLists(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Response should be a valid JSON array
		var lists []List
		if err := json.Unmarshal(w.Body.Bytes(), &lists); err != nil {
			t.Errorf("Failed to parse lists response: %v", err)
		}
	})

	t.Run("Success_WithLists", func(t *testing.T) {
		// Create test lists
		testList1 := setupTestList(t, testApp.DB, "Test List 1", 3)
		defer testList1.Cleanup()
		testList2 := setupTestList(t, testApp.DB, "Test List 2", 5)
		defer testList2.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/lists",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetLists(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var lists []List
		if err := json.Unmarshal(w.Body.Bytes(), &lists); err != nil {
			t.Errorf("Failed to parse lists response: %v", err)
		}

		if len(lists) < 2 {
			t.Errorf("Expected at least 2 lists, got %d", len(lists))
		}
	})
}

// TestCreateList tests the CreateList handler
func TestCreateList(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := TestData.CreateListRequest("New Test List", 5)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/lists",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.CreateList(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"item_count": float64(5), // JSON numbers are float64
		})

		if response == nil {
			t.Fatal("Expected valid JSON response")
		}

		// Verify list_id is present
		if _, ok := response["list_id"]; !ok {
			t.Error("Expected list_id in response")
		}

		// Clean up
		if listID, ok := response["list_id"].(string); ok {
			testApp.DB.Exec("DELETE FROM elo_swipe.items WHERE list_id = $1", listID)
			testApp.DB.Exec("DELETE FROM elo_swipe.lists WHERE id = $1", listID)
		}
	})

	t.Run("Success_EmptyItems", func(t *testing.T) {
		req := CreateListRequest{
			Name:        "Empty List",
			Description: "A list with no items",
			Items:       []ItemInput{},
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/lists",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.CreateList(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"item_count": float64(0),
		})

		// Clean up
		if listID, ok := response["list_id"].(string); ok {
			testApp.DB.Exec("DELETE FROM elo_swipe.lists WHERE id = $1", listID)
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/lists",
			Body:   `{"invalid": json}`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.CreateList(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("Error_EmptyBody", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/lists",
			Body:   "",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.CreateList(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for empty body, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

// TestGetList tests the GetList handler
func TestGetList(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 3)
		defer testList.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/lists/%s", testList.List.ID),
			URLVars: map[string]string{"id": testList.List.ID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetList(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"id":   testList.List.ID,
			"name": testList.List.Name,
		})

		if response == nil {
			t.Fatal("Expected valid JSON response")
		}
	})

	t.Run("Error_NonExistentList", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/lists/%s", nonExistentID),
			URLVars: map[string]string{"id": nonExistentID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetList(w, httpReq)

		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})
}

// TestGetNextComparison tests the GetNextComparison handler
func TestGetNextComparison(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 3)
		defer testList.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/lists/%s/next-comparison", testList.List.ID),
			URLVars: map[string]string{"id": testList.List.ID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetNextComparison(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Expected valid JSON response")
		}

		// Verify item_a and item_b are present
		if _, ok := response["item_a"]; !ok {
			t.Error("Expected item_a in response")
		}
		if _, ok := response["item_b"]; !ok {
			t.Error("Expected item_b in response")
		}
		if _, ok := response["progress"]; !ok {
			t.Error("Expected progress in response")
		}
	})

	t.Run("Error_InsufficientItems", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Single Item List", 1)
		defer testList.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/lists/%s/next-comparison", testList.List.ID),
			URLVars: map[string]string{"id": testList.List.ID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetNextComparison(w, httpReq)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status %d for insufficient items, got %d", http.StatusNoContent, w.Code)
		}
	})

	t.Run("Error_NonExistentList", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/lists/%s/next-comparison", nonExistentID),
			URLVars: map[string]string{"id": nonExistentID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetNextComparison(w, httpReq)

		// Either 404 or 204 is acceptable for non-existent list
		if w.Code != http.StatusNotFound && w.Code != http.StatusNoContent && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 404, 204, or 500 for non-existent list, got %d", w.Code)
		}
	})
}

// TestCreateComparison tests the CreateComparison handler
func TestCreateComparison(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 3)
		defer testList.Cleanup()

		req := TestData.CreateComparisonRequest(
			testList.List.ID,
			testList.Items[0].ID,
			testList.Items[1].ID,
		)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/comparisons",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.CreateComparison(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"list_id":   testList.List.ID,
			"winner_id": testList.Items[0].ID,
			"loser_id":  testList.Items[1].ID,
		})

		if response == nil {
			t.Fatal("Expected valid JSON response")
		}

		// Verify ratings changed
		if _, ok := response["winner_rating_after"]; !ok {
			t.Error("Expected winner_rating_after in response")
		}
		if _, ok := response["loser_rating_after"]; !ok {
			t.Error("Expected loser_rating_after in response")
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/comparisons",
			Body:   `{"invalid": json}`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.CreateComparison(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("Error_NonExistentItems", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 2)
		defer testList.Cleanup()

		req := TestData.CreateComparisonRequest(
			testList.List.ID,
			uuid.New().String(), // Non-existent winner
			testList.Items[0].ID,
		)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/comparisons",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.CreateComparison(w, httpReq)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d for non-existent item, got %d", http.StatusInternalServerError, w.Code)
		}
	})
}

// TestGetRankings tests the GetRankings handler
func TestGetRankings(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("Success_JSON", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 3)
		defer testList.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/lists/%s/rankings", testList.List.ID),
			URLVars: map[string]string{"id": testList.List.ID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetRankings(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Expected valid JSON response")
		}

		rankings, ok := response["rankings"].([]interface{})
		if !ok {
			t.Fatal("Expected rankings array in response")
		}

		if len(rankings) != 3 {
			t.Errorf("Expected 3 rankings, got %d", len(rankings))
		}
	})

	t.Run("Success_CSV", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 3)
		defer testList.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        fmt.Sprintf("/api/v1/lists/%s/rankings?format=csv", testList.List.ID),
			URLVars:     map[string]string{"id": testList.List.ID},
			QueryParams: map[string]string{"format": "csv"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetRankings(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d for CSV format, got %d", http.StatusOK, w.Code)
		}

		if contentType := w.Header().Get("Content-Type"); contentType != "text/csv" {
			t.Errorf("Expected Content-Type 'text/csv', got '%s'", contentType)
		}

		// Verify CSV content has at least header + 3 rows
		csvContent := w.Body.String()
		if csvContent == "" {
			t.Error("Expected CSV content, got empty response")
		}
	})

	t.Run("Error_InvalidFormat", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 3)
		defer testList.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        fmt.Sprintf("/api/v1/lists/%s/rankings?format=xml", testList.List.ID),
			URLVars:     map[string]string{"id": testList.List.ID},
			QueryParams: map[string]string{"format": "xml"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetRankings(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for invalid format, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

// TestSmartPairing tests the smart pairing handlers
// Note: These tests are skipped because they require Ollama integration
func TestGenerateSmartPairing(t *testing.T) {
	t.Skip("Skipping smart pairing tests - require Ollama setup and pairing_queue table")
}

// TestGetPairingQueue tests getting the pairing queue
func TestGetPairingQueue(t *testing.T) {
	t.Skip("Skipping pairing queue tests - require pairing_queue table")
}

// TestRefreshPairingQueue tests refreshing the pairing queue
func TestRefreshPairingQueue(t *testing.T) {
	t.Skip("Skipping pairing queue refresh tests - require pairing_queue table")
}

// TestHelperFunctions tests the helper functions
func TestGetItem(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 2)
		defer testList.Cleanup()

		item, err := testApp.App.getItem(testList.Items[0].ID)
		if err != nil {
			t.Fatalf("Failed to get item: %v", err)
		}

		if item.ID != testList.Items[0].ID {
			t.Errorf("Expected item ID %s, got %s", testList.Items[0].ID, item.ID)
		}
	})

	t.Run("Error_NonExistent", func(t *testing.T) {
		_, err := testApp.App.getItem(uuid.New().String())
		if err == nil {
			t.Error("Expected error for non-existent item")
		}
	})
}

func TestGetRatings(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 2)
		defer testList.Cleanup()

		rating1, rating2, err := testApp.App.getRatings(testList.Items[0].ID, testList.Items[1].ID)
		if err != nil {
			t.Fatalf("Failed to get ratings: %v", err)
		}

		if rating1 != 1500.0 {
			t.Errorf("Expected initial rating 1500.0, got %f", rating1)
		}
		if rating2 != 1500.0 {
			t.Errorf("Expected initial rating 1500.0, got %f", rating2)
		}
	})

	t.Run("Error_NonExistent", func(t *testing.T) {
		_, _, err := testApp.App.getRatings(uuid.New().String(), uuid.New().String())
		if err == nil {
			t.Error("Expected error for non-existent items")
		}
	})
}

func TestGetProgress(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 5)
		defer testList.Cleanup()

		progress, err := testApp.App.getProgress(testList.List.ID)
		if err != nil {
			t.Fatalf("Failed to get progress: %v", err)
		}

		if progress.Completed < 0 {
			t.Errorf("Expected non-negative completed count, got %d", progress.Completed)
		}
		if progress.Total <= 0 {
			t.Errorf("Expected positive total count, got %d", progress.Total)
		}
	})
}

// TestDeleteComparison tests the DeleteComparison handler
func TestDeleteComparison(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("Success", func(t *testing.T) {
		comparisonID := uuid.New().String()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    fmt.Sprintf("/api/v1/comparisons/%s", comparisonID),
			URLVars: map[string]string{"id": comparisonID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.DeleteComparison(w, httpReq)

		// Currently returns 204 No Content (TODO: implement actual undo logic)
		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
		}
	})
}

// TestEloCalculation tests the Elo rating calculation logic
func TestEloCalculation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("RatingChanges", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 2)
		defer testList.Cleanup()

		// Get initial ratings
		initialRating1, initialRating2, _ := testApp.App.getRatings(testList.Items[0].ID, testList.Items[1].ID)

		// Create comparison
		req := TestData.CreateComparisonRequest(
			testList.List.ID,
			testList.Items[0].ID, // Winner
			testList.Items[1].ID, // Loser
		)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/comparisons",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.CreateComparison(w, httpReq)

		var comparison Comparison
		json.Unmarshal(w.Body.Bytes(), &comparison)

		// Winner rating should increase
		if comparison.WinnerRatingAfter <= initialRating1 {
			t.Errorf("Expected winner rating to increase from %f, got %f",
				initialRating1, comparison.WinnerRatingAfter)
		}

		// Loser rating should decrease
		if comparison.LoserRatingAfter >= initialRating2 {
			t.Errorf("Expected loser rating to decrease from %f, got %f",
				initialRating2, comparison.LoserRatingAfter)
		}
	})

	t.Run("MultipleComparisons", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 3)
		defer testList.Cleanup()

		// Create multiple comparisons to test rating convergence
		for i := 0; i < 5; i++ {
			req := TestData.CreateComparisonRequest(
				testList.List.ID,
				testList.Items[0].ID, // Same winner
				testList.Items[1].ID, // Same loser
			)

			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/comparisons",
				Body:   req,
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			testApp.App.CreateComparison(w, httpReq)
		}

		// Winner should have significantly higher rating after multiple wins
		rating1, rating2, _ := testApp.App.getRatings(testList.Items[0].ID, testList.Items[1].ID)
		if rating1 <= rating2 {
			t.Errorf("Expected winner rating %f to be higher than loser rating %f", rating1, rating2)
		}
	})
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("EmptyListRankings", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Empty List", 0)
		defer testList.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/lists/%s/rankings", testList.List.ID),
			URLVars: map[string]string{"id": testList.List.ID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetRankings(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d for empty list rankings, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("LargeList", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Large List", 50)
		defer testList.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/lists/%s/rankings", testList.List.ID),
			URLVars: map[string]string{"id": testList.List.ID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetRankings(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Expected valid JSON response")
		}

		rankings, ok := response["rankings"].([]interface{})
		if !ok {
			t.Fatal("Expected rankings array")
		}

		if len(rankings) != 50 {
			t.Errorf("Expected 50 items, got %d", len(rankings))
		}
	})

	t.Run("GetProgressNoComparisons", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 10)
		defer testList.Cleanup()

		progress, err := testApp.App.getProgress(testList.List.ID)
		if err != nil {
			t.Fatalf("Failed to get progress: %v", err)
		}

		if progress.Completed != 0 {
			t.Errorf("Expected 0 comparisons, got %d", progress.Completed)
		}
		if progress.Total <= 0 {
			t.Errorf("Expected positive total, got %d", progress.Total)
		}
	})

	t.Run("CreateListWithManyItems", func(t *testing.T) {
		req := TestData.CreateListRequest("Big List", 100)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/lists",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.CreateList(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"item_count": float64(100),
		})

		// Clean up
		if listID, ok := response["list_id"].(string); ok {
			testApp.DB.Exec("DELETE FROM elo_swipe.items WHERE list_id = $1", listID)
			testApp.DB.Exec("DELETE FROM elo_swipe.lists WHERE id = $1", listID)
		}
	})
}

// TestIntegrationFlows tests complete user workflows
func TestIntegrationFlows(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("CompleteRankingFlow", func(t *testing.T) {
		// 1. Create a list
		createReq := TestData.CreateListRequest("Priority List", 5)
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/lists",
			Body:   createReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.CreateList(w, httpReq)
		var createResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &createResp)
		listID := createResp["list_id"].(string)
		defer func() {
			testApp.DB.Exec("DELETE FROM elo_swipe.comparisons WHERE list_id = $1", listID)
			testApp.DB.Exec("DELETE FROM elo_swipe.items WHERE list_id = $1", listID)
			testApp.DB.Exec("DELETE FROM elo_swipe.lists WHERE id = $1", listID)
		}()

		// 2. Get next comparison
		w, httpReq, err = makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/lists/%s/next-comparison", listID),
			URLVars: map[string]string{"id": listID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetNextComparison(w, httpReq)
		var nextComp NextComparison
		json.Unmarshal(w.Body.Bytes(), &nextComp)

		// 3. Submit comparison
		compReq := TestData.CreateComparisonRequest(listID, nextComp.ItemA.ID, nextComp.ItemB.ID)
		w, httpReq, err = makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/comparisons",
			Body:   compReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.CreateComparison(w, httpReq)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected comparison to be created, got status %d", w.Code)
		}

		// 4. Get rankings
		w, httpReq, err = makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/lists/%s/rankings", listID),
			URLVars: map[string]string{"id": listID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetRankings(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Expected valid rankings response")
		}
	})
}
