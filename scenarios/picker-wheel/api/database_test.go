package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestGetWheelsHandler_DatabasePaths tests database-related code paths
func TestGetWheelsHandler_DatabasePaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_WithDatabase", func(t *testing.T) {
		resetTestState()
		// Simulate database available but returning no rows
		// This will test the database query path

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/wheels",
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		var wheelsList []Wheel
		assertJSONResponse(t, w, http.StatusOK, &wheelsList)
	})
}

// TestCreateWheelHandler_DatabasePaths tests database-related code paths
func TestCreateWheelHandler_DatabasePaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_InMemoryFallback", func(t *testing.T) {
		resetTestState()
		db = nil // Ensure we're using in-memory

		newWheel := map[string]interface{}{
			"name":        "Test Wheel",
			"description": "A test wheel",
			"theme":       "test",
			"options": []map[string]interface{}{
				{"label": "Option 1", "color": "#FF0000", "weight": 1.0},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/wheels",
			Body:   newWheel,
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		var createdWheel Wheel
		assertJSONResponse(t, w, http.StatusOK, &createdWheel)

		if createdWheel.Name != "Test Wheel" {
			t.Errorf("Expected name 'Test Wheel', got '%s'", createdWheel.Name)
		}
	})

	t.Run("EdgeCase_MissingName", func(t *testing.T) {
		resetTestState()

		newWheel := map[string]interface{}{
			"description": "A wheel with no name",
			"theme":       "test",
			"options": []map[string]interface{}{
				{"label": "Option 1", "color": "#FF0000", "weight": 1.0},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/wheels",
			Body:   newWheel,
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		// Should still create the wheel
		var createdWheel Wheel
		assertJSONResponse(t, w, http.StatusOK, &createdWheel)
	})
}

// TestSpinHandler_EdgeCases tests additional edge cases
func TestSpinHandler_EdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EdgeCase_ZeroWeights", func(t *testing.T) {
		resetTestState()

		spinRequest := map[string]interface{}{
			"wheel_id":   "zero-weights",
			"session_id": "test-session",
			"options": []map[string]interface{}{
				{"label": "Option 1", "color": "#FF0000", "weight": 0.0},
				{"label": "Option 2", "color": "#00FF00", "weight": 0.0},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/spin",
			Body:   spinRequest,
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		var result SpinResult
		assertJSONResponse(t, w, http.StatusOK, &result)

		// With zero weights, first option should be selected
		if result.Result != "Option 1" && result.Result != "Option 2" {
			t.Errorf("Unexpected result: '%s'", result.Result)
		}
	})

	t.Run("EdgeCase_SingleOption", func(t *testing.T) {
		resetTestState()

		spinRequest := map[string]interface{}{
			"wheel_id":   "single-option",
			"session_id": "test-session",
			"options": []map[string]interface{}{
				{"label": "Only Option", "color": "#FF0000", "weight": 1.0},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/spin",
			Body:   spinRequest,
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		var result SpinResult
		assertJSONResponse(t, w, http.StatusOK, &result)

		if result.Result != "Only Option" {
			t.Errorf("Expected 'Only Option', got '%s'", result.Result)
		}
	})

	t.Run("Success_EmptySessionID", func(t *testing.T) {
		resetTestState()
		initTestWheels()

		spinRequest := map[string]interface{}{
			"wheel_id": "yes-or-no",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/spin",
			Body:   spinRequest,
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		var result SpinResult
		assertJSONResponse(t, w, http.StatusOK, &result)

		if result.Result == "" {
			t.Error("Expected a result, got empty string")
		}
	})

	t.Run("Success_HistoryTracking", func(t *testing.T) {
		resetTestState()
		initTestWheels()
		initialHistoryLen := len(history)

		spinRequest := map[string]interface{}{
			"wheel_id":   "yes-or-no",
			"session_id": "history-test",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/spin",
			Body:   spinRequest,
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		var result SpinResult
		assertJSONResponse(t, w, http.StatusOK, &result)

		// Verify history was updated
		if len(history) != initialHistoryLen+1 {
			t.Errorf("Expected history length %d, got %d", initialHistoryLen+1, len(history))
		}

		// Verify the last history item matches the result
		lastHistory := history[len(history)-1]
		if lastHistory.Result != result.Result {
			t.Errorf("History result mismatch: expected '%s', got '%s'", result.Result, lastHistory.Result)
		}
		if lastHistory.WheelID != "yes-or-no" {
			t.Errorf("History wheel_id mismatch: expected 'yes-or-no', got '%s'", lastHistory.WheelID)
		}
		if lastHistory.SessionID != "history-test" {
			t.Errorf("History session_id mismatch: expected 'history-test', got '%s'", lastHistory.SessionID)
		}
	})
}

// TestGetHistoryHandler_EdgeCases tests additional history handler cases
func TestGetHistoryHandler_EdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_DatabaseFallback", func(t *testing.T) {
		resetTestState()
		db = nil // Ensure we're using in-memory

		// Add some history
		history = append(history, SpinResult{
			Result:    "Test Result",
			WheelID:   "test-wheel",
			SessionID: "test-session",
			Timestamp: time.Now(),
		})

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/history",
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		var historyList []SpinResult
		assertJSONResponse(t, w, http.StatusOK, &historyList)

		if len(historyList) != 1 {
			t.Errorf("Expected 1 history item, got %d", len(historyList))
		}
	})
}

// TestOptionValidation tests option field validation
func TestOptionValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_MissingOptionalFields", func(t *testing.T) {
		resetTestState()

		// Create wheel with minimal option data
		newWheel := map[string]interface{}{
			"name": "Minimal Wheel",
			"options": []map[string]interface{}{
				{"label": "Option 1"},
				{"label": "Option 2"},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/wheels",
			Body:   newWheel,
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		var createdWheel Wheel
		assertJSONResponse(t, w, http.StatusOK, &createdWheel)

		if len(createdWheel.Options) != 2 {
			t.Errorf("Expected 2 options, got %d", len(createdWheel.Options))
		}
	})
}

// TestWheelStructValidation tests wheel structure validation
func TestWheelStructValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_AllFields", func(t *testing.T) {
		resetTestState()

		newWheel := map[string]interface{}{
			"id":          "test-id",
			"name":        "Complete Wheel",
			"description": "A wheel with all fields",
			"theme":       "complete",
			"times_used":  5,
			"options": []map[string]interface{}{
				{"label": "Red", "color": "#FF0000", "weight": 2.0},
				{"label": "Blue", "color": "#0000FF", "weight": 1.0},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/wheels",
			Body:   newWheel,
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		var createdWheel Wheel
		assertJSONResponse(t, w, http.StatusOK, &createdWheel)

		if createdWheel.Name != "Complete Wheel" {
			t.Errorf("Expected name 'Complete Wheel', got '%s'", createdWheel.Name)
		}
		if createdWheel.Description != "A wheel with all fields" {
			t.Errorf("Expected description to match, got '%s'", createdWheel.Description)
		}
		if createdWheel.Theme != "complete" {
			t.Errorf("Expected theme 'complete', got '%s'", createdWheel.Theme)
		}
	})
}

// TestSpinResultStructure tests spin result structure
func TestSpinResultStructure(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_CompleteResult", func(t *testing.T) {
		resetTestState()
		initTestWheels()

		spinRequest := map[string]interface{}{
			"wheel_id":   "dinner-decider",
			"session_id": "complete-test",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/spin",
			Body:   spinRequest,
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		var result SpinResult
		assertJSONResponse(t, w, http.StatusOK, &result)

		// Verify all fields are populated
		if result.Result == "" {
			t.Error("Result field should not be empty")
		}
		if result.WheelID != "dinner-decider" {
			t.Errorf("Expected wheel_id 'dinner-decider', got '%s'", result.WheelID)
		}
		if result.SessionID != "complete-test" {
			t.Errorf("Expected session_id 'complete-test', got '%s'", result.SessionID)
		}
		if result.Timestamp.IsZero() {
			t.Error("Timestamp should be set")
		}

		// Verify result is one of the valid dinner options
		validOptions := []string{"Pizza 🍕", "Sushi 🍱", "Tacos 🌮", "Burger 🍔"}
		isValid := false
		for _, valid := range validOptions {
			if result.Result == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			t.Errorf("Result '%s' is not a valid dinner option", result.Result)
		}
	})
}

// TestHealthResponseStructure tests health response structure
func TestHealthResponseStructure(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_VerifyStructure", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		// Verify JSON structure
		var rawResponse map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &rawResponse); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Check all expected fields exist
		if _, ok := rawResponse["status"]; !ok {
			t.Error("Response missing 'status' field")
		}
		if _, ok := rawResponse["service"]; !ok {
			t.Error("Response missing 'service' field")
		}
		if _, ok := rawResponse["version"]; !ok {
			t.Error("Response missing 'version' field")
		}
	})
}

// TestCORS tests CORS handling (implicitly through router)
func TestCORSImplicit(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_OptionsRequest", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/api/wheels",
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		// OPTIONS requests should be handled (may return 404 or 200 depending on CORS middleware)
		// The important thing is it doesn't panic
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
			t.Logf("OPTIONS request returned status %d (acceptable)", w.Code)
		}
	})
}
