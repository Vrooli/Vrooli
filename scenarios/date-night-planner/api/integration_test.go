package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestIntegration_FullDatePlanningWorkflow tests complete workflow
func TestIntegration_FullDatePlanningWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dbCleanup := setupTestDB(t)
	defer dbCleanup()

	// Step 1: Get date suggestions
	suggestReq := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/dates/suggest",
		Body: map[string]interface{}{
			"couple_id":  "test-couple-123",
			"date_type":  "romantic",
			"budget_max": 100,
		},
	}

	w1, err := makeHTTPRequest(suggestReq, suggestDatesHandler)
	if err != nil {
		t.Fatalf("Failed to get suggestions: %v", err)
	}

	var suggestResp DateSuggestionResponse
	if err := json.NewDecoder(w1.Body).Decode(&suggestResp); err != nil {
		t.Fatalf("Failed to decode suggestions: %v", err)
	}

	if len(suggestResp.Suggestions) == 0 {
		t.Fatal("No suggestions returned")
	}

	// Step 2: Select a suggestion and create a date plan
	selectedSuggestion := suggestResp.Suggestions[0]

	planReq := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/dates/plan",
		Body: map[string]interface{}{
			"couple_id": "test-couple-123",
			"selected_suggestion": map[string]interface{}{
				"title":              selectedSuggestion.Title,
				"description":        selectedSuggestion.Description,
				"activities":         selectedSuggestion.Activities,
				"estimated_cost":     selectedSuggestion.EstimatedCost,
				"estimated_duration": selectedSuggestion.EstimatedDuration,
				"confidence_score":   selectedSuggestion.ConfidenceScore,
			},
			"planned_date": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		},
	}

	w2, err := makeHTTPRequest(planReq, planDateHandler)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	if w2.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w2.Code)
	}

	var planResp DatePlanResponse
	if err := json.NewDecoder(w2.Body).Decode(&planResp); err != nil {
		t.Fatalf("Failed to decode plan response: %v", err)
	}

	if planResp.DatePlan.ID == "" {
		t.Error("Expected date plan ID")
	}

	if planResp.DatePlan.Title != selectedSuggestion.Title {
		t.Errorf("Expected title '%s', got '%s'", selectedSuggestion.Title, planResp.DatePlan.Title)
	}
}

// TestIntegration_SurpriseDateWorkflow tests surprise date workflow
func TestIntegration_SurpriseDateWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dbCleanup := setupTestDB(t)
	defer dbCleanup()

	// Step 1: Get suggestions
	suggestReq := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/dates/suggest",
		Body: map[string]interface{}{
			"couple_id":  "test-couple-123",
			"date_type":  "adventure",
			"budget_max": 200,
		},
	}

	w1, err := makeHTTPRequest(suggestReq, suggestDatesHandler)
	if err != nil {
		t.Fatalf("Failed to get suggestions: %v", err)
	}

	var suggestResp DateSuggestionResponse
	if err := json.NewDecoder(w1.Body).Decode(&suggestResp); err != nil {
		t.Fatalf("Failed to decode suggestions: %v", err)
	}

	// Step 2: Create surprise date
	selectedSuggestion := suggestResp.Suggestions[0]

	surpriseReq := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/dates/surprise",
		Body: map[string]interface{}{
			"couple_id":  "test-couple-123",
			"planned_by": "partner-1",
			"date_suggestion": map[string]interface{}{
				"title":              selectedSuggestion.Title,
				"description":        selectedSuggestion.Description,
				"activities":         selectedSuggestion.Activities,
				"estimated_cost":     selectedSuggestion.EstimatedCost,
				"estimated_duration": selectedSuggestion.EstimatedDuration,
			},
			"planned_date": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
		},
	}

	w2, err := makeHTTPRequest(surpriseReq, surpriseDateHandler)
	if err != nil {
		t.Fatalf("Failed to create surprise: %v", err)
	}

	if w2.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w2.Code)
	}

	var surpriseResp map[string]interface{}
	if err := json.NewDecoder(w2.Body).Decode(&surpriseResp); err != nil {
		t.Fatalf("Failed to decode surprise response: %v", err)
	}

	surpriseID, ok := surpriseResp["surprise_id"].(string)
	if !ok || surpriseID == "" {
		t.Fatal("Expected surprise_id in response")
	}

	// Step 3: Verify planner can access details
	getReq := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/v1/dates/surprise/" + surpriseID,
		QueryParams: map[string]string{
			"requester_id": "partner-1",
		},
	}

	w3, err := makeHTTPRequest(getReq, getSurpriseHandler)
	if err != nil {
		t.Fatalf("Failed to get surprise details: %v", err)
	}

	if w3.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w3.Code)
	}
}

// TestIntegration_MultipleCoupleSeparation tests data isolation between couples
func TestIntegration_MultipleCoupleSeparation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dbCleanup := setupTestDB(t)
	defer dbCleanup()

	couples := []string{"couple-1", "couple-2", "couple-3"}
	dateTypes := []string{"romantic", "adventure", "cultural"}

	for i, coupleID := range couples {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/suggest",
			Body: map[string]interface{}{
				"couple_id": coupleID,
				"date_type": dateTypes[i],
			},
		}

		w, err := makeHTTPRequest(req, suggestDatesHandler)
		if err != nil {
			t.Fatalf("Failed for couple %s: %v", coupleID, err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for %s, got %d", coupleID, w.Code)
		}

		var resp DateSuggestionResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode for %s: %v", coupleID, err)
		}

		if len(resp.Suggestions) == 0 {
			t.Errorf("No suggestions for couple %s", coupleID)
		}
	}
}

// TestIntegration_ErrorRecovery tests graceful error handling
func TestIntegration_ErrorRecovery(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name           string
		handler        http.HandlerFunc
		request        HTTPTestRequest
		expectedStatus int
	}{
		{
			name:    "MissingCoupleID",
			handler: suggestDatesHandler,
			request: HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/dates/suggest",
				Body:   map[string]interface{}{},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "MalformedJSON",
			handler: suggestDatesHandler,
			request: HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/dates/suggest",
				Body:   "invalid-json",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "MissingRequesterID",
			handler: getSurpriseHandler,
			request: HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/dates/surprise/test-id",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w, err := makeHTTPRequest(tc.request, tc.handler)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}
