package main

import (
	"net/http"
	"testing"
)

// TestGetWheelsHandler_AdditionalCoverage tests additional code paths
func TestGetWheelsHandler_AdditionalCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_WithMultipleWheels", func(t *testing.T) {
		resetTestState()
		initTestWheels()

		// Add additional wheels to test full list
		wheels = append(wheels, Wheel{
			ID:          "test-wheel-1",
			Name:        "Test Wheel 1",
			Description: "First test wheel",
			Options: []Option{
				{Label: "A", Color: "#FF0000", Weight: 1.0},
				{Label: "B", Color: "#00FF00", Weight: 1.0},
			},
			Theme:     "test",
			TimesUsed: 5,
		})

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

		if len(wheelsList) < 2 {
			t.Errorf("Expected at least 2 wheels, got %d", len(wheelsList))
		}
	})
}

// TestCreateWheelHandler_AdditionalCoverage tests additional code paths
func TestCreateWheelHandler_AdditionalCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_WithAllFields", func(t *testing.T) {
		resetTestState()

		newWheel := map[string]interface{}{
			"name":        "Complete Wheel",
			"description": "A wheel with all fields",
			"theme":       "neon",
			"options": []map[string]interface{}{
				{"label": "Option 1", "color": "#FF0000", "weight": 2.0},
				{"label": "Option 2", "color": "#00FF00", "weight": 1.5},
				{"label": "Option 3", "color": "#0000FF", "weight": 1.0},
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
		if createdWheel.Theme != "neon" {
			t.Errorf("Expected theme 'neon', got '%s'", createdWheel.Theme)
		}
		if len(createdWheel.Options) != 3 {
			t.Errorf("Expected 3 options, got %d", len(createdWheel.Options))
		}
	})

	t.Run("EdgeCase_MinimalWheel", func(t *testing.T) {
		resetTestState()

		newWheel := map[string]interface{}{
			"name": "Minimal",
			"options": []map[string]interface{}{
				{"label": "Yes", "color": "#00FF00", "weight": 1.0},
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

		if createdWheel.Name != "Minimal" {
			t.Errorf("Expected name 'Minimal', got '%s'", createdWheel.Name)
		}
	})
}

// TestGetHistoryHandler_AdditionalCoverage tests additional code paths
func TestGetHistoryHandler_AdditionalCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_MultipleHistoryItems", func(t *testing.T) {
		resetTestState()
		initTestWheels()

		// Add multiple history items
		for i := 0; i < 5; i++ {
			history = append(history, SpinResult{
				Result:    "Test Result",
				WheelID:   "test-wheel",
				SessionID: "session-test",
			})
		}

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

		if len(historyList) != 5 {
			t.Errorf("Expected 5 history items, got %d", len(historyList))
		}
	})
}

// TestSpinHandler_AdditionalCoverage tests additional code paths
func TestSpinHandler_AdditionalCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_SingleOptionWheel", func(t *testing.T) {
		resetTestState()

		spinRequest := map[string]interface{}{
			"wheel_id":   "single-option",
			"session_id": "test",
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

	t.Run("Success_ZeroWeights", func(t *testing.T) {
		resetTestState()

		spinRequest := map[string]interface{}{
			"wheel_id":   "zero-weights",
			"session_id": "test",
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

		// Should still return a result even with zero weights
		if result.Result == "" {
			t.Error("Expected a result, got empty string")
		}
	})

	t.Run("Success_MissingSessionID", func(t *testing.T) {
		resetTestState()
		initTestWheels()

		spinRequest := map[string]interface{}{
			"wheel_id": "yes-or-no",
			// No session_id provided
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
}

// TestDeleteWheelHandler_AdditionalCoverage tests additional code paths
func TestDeleteWheelHandler_AdditionalCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_DeleteCustomWheel", func(t *testing.T) {
		resetTestState()

		// Create a custom wheel first
		customWheel := Wheel{
			ID:          "custom-to-delete",
			Name:        "Custom Wheel",
			Description: "Will be deleted",
			Options: []Option{
				{Label: "A", Color: "#FF0000", Weight: 1.0},
			},
		}
		wheels = append(wheels, customWheel)

		initialCount := len(wheels)

		req := HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/wheels/custom-to-delete",
			URLVars: map[string]string{"id": "custom-to-delete"},
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
		}

		if len(wheels) != initialCount-1 {
			t.Errorf("Expected %d wheels after delete, got %d", initialCount-1, len(wheels))
		}
	})
}

// TestGetWheelHandler_AdditionalCoverage tests additional code paths
func TestGetWheelHandler_AdditionalCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_GetDinnerDecider", func(t *testing.T) {
		resetTestState()
		initTestWheels()

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/wheels/dinner-decider",
			URLVars: map[string]string{"id": "dinner-decider"},
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		var wheel Wheel
		assertJSONResponse(t, w, http.StatusOK, &wheel)

		if wheel.ID != "dinner-decider" {
			t.Errorf("Expected ID 'dinner-decider', got '%s'", wheel.ID)
		}
		if len(wheel.Options) == 0 {
			t.Error("Expected dinner-decider to have options")
		}
	})
}
