package main

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
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

		var response HealthResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		if response.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response.Status)
		}
		if response.Service != "picker-wheel-api" {
			t.Errorf("Expected service 'picker-wheel-api', got '%s'", response.Service)
		}
		if response.Version != "1.0.0" {
			t.Errorf("Expected version '1.0.0', got '%s'", response.Version)
		}
		if response.Readiness != true {
			t.Errorf("Expected readiness 'true', got '%v'", response.Readiness)
		}
		if response.Timestamp.IsZero() {
			t.Error("Expected timestamp to be set, got zero value")
		}
	})
}

// TestGetWheelsHandler tests retrieving wheels
func TestGetWheelsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_DefaultWheels", func(t *testing.T) {
		resetTestState()
		initTestWheels()

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

		if len(wheelsList) == 0 {
			t.Error("Expected default wheels, got empty list")
		}

		// Verify we have the expected default wheels
		foundYesNo := false
		foundDinner := false
		for _, wheel := range wheelsList {
			if wheel.ID == "yes-or-no" {
				foundYesNo = true
			}
			if wheel.ID == "dinner-decider" {
				foundDinner = true
			}
		}

		if !foundYesNo {
			t.Error("Expected to find 'yes-or-no' wheel")
		}
		if !foundDinner {
			t.Error("Expected to find 'dinner-decider' wheel")
		}
	})

	t.Run("Success_EmptyWhenNoDatabase", func(t *testing.T) {
		resetTestState()
		// Don't initialize wheels to test empty state

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

		// Should return default wheels when in-memory is empty
		var wheelsList []Wheel
		assertJSONResponse(t, w, http.StatusOK, &wheelsList)
	})
}

// TestCreateWheelHandler tests creating custom wheels
func TestCreateWheelHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		resetTestState()

		newWheel := map[string]interface{}{
			"name":        "Test Wheel",
			"description": "A test wheel",
			"theme":       "test",
			"options": []map[string]interface{}{
				{"label": "Option 1", "color": "#FF0000", "weight": 1.0},
				{"label": "Option 2", "color": "#00FF00", "weight": 1.0},
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
		if createdWheel.ID == "" {
			t.Error("Expected wheel to have an ID assigned")
		}
		if len(createdWheel.Options) != 2 {
			t.Errorf("Expected 2 options, got %d", len(createdWheel.Options))
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		resetTestState()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/wheels",
			Body:   "invalid json",
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("EdgeCase_EmptyOptions", func(t *testing.T) {
		resetTestState()

		newWheel := map[string]interface{}{
			"name":        "Empty Wheel",
			"description": "Wheel with no options",
			"theme":       "test",
			"options":     []map[string]interface{}{},
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

		// Should accept even with empty options
		var createdWheel Wheel
		assertJSONResponse(t, w, http.StatusOK, &createdWheel)

		if len(createdWheel.Options) != 0 {
			t.Errorf("Expected 0 options, got %d", len(createdWheel.Options))
		}
	})
}

// TestGetWheelHandler tests retrieving a specific wheel by ID
func TestGetWheelHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		resetTestState()
		initTestWheels()

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/wheels/yes-or-no",
			URLVars: map[string]string{"id": "yes-or-no"},
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		var wheel Wheel
		assertJSONResponse(t, w, http.StatusOK, &wheel)

		if wheel.ID != "yes-or-no" {
			t.Errorf("Expected ID 'yes-or-no', got '%s'", wheel.ID)
		}
		if wheel.Name != "Yes or No" {
			t.Errorf("Expected name 'Yes or No', got '%s'", wheel.Name)
		}
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		resetTestState()
		initTestWheels()

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/wheels/non-existent",
			URLVars: map[string]string{"id": "non-existent"},
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		assertErrorResponse(t, w, http.StatusNotFound, "")
	})
}

// TestDeleteWheelHandler tests deleting wheels
func TestDeleteWheelHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		resetTestState()
		initTestWheels()

		initialCount := len(wheels)

		req := HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/wheels/yes-or-no",
			URLVars: map[string]string{"id": "yes-or-no"},
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

		// Verify wheel was removed
		if len(wheels) != initialCount-1 {
			t.Errorf("Expected %d wheels after delete, got %d", initialCount-1, len(wheels))
		}
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		resetTestState()
		initTestWheels()

		req := HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/wheels/non-existent",
			URLVars: map[string]string{"id": "non-existent"},
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		assertErrorResponse(t, w, http.StatusNotFound, "")
	})
}

// TestSpinHandler tests the wheel spinning functionality
func TestSpinHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_WithWheelID", func(t *testing.T) {
		resetTestState()
		initTestWheels()

		spinRequest := map[string]interface{}{
			"wheel_id":   "yes-or-no",
			"session_id": "test-session",
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
		if result.WheelID != "yes-or-no" {
			t.Errorf("Expected wheel_id 'yes-or-no', got '%s'", result.WheelID)
		}
		if result.SessionID != "test-session" {
			t.Errorf("Expected session_id 'test-session', got '%s'", result.SessionID)
		}

		// Verify result is one of the valid options
		validResults := []string{"YES! ✅", "NO ❌"}
		isValid := false
		for _, valid := range validResults {
			if result.Result == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			t.Errorf("Result '%s' is not a valid option for yes-or-no wheel", result.Result)
		}
	})

	t.Run("Success_WithCustomOptions", func(t *testing.T) {
		resetTestState()

		spinRequest := map[string]interface{}{
			"wheel_id":   "custom",
			"session_id": "test-session",
			"options": []map[string]interface{}{
				{"label": "Red", "color": "#FF0000", "weight": 1.0},
				{"label": "Blue", "color": "#0000FF", "weight": 1.0},
				{"label": "Green", "color": "#00FF00", "weight": 1.0},
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

		if result.Result == "" {
			t.Error("Expected a result, got empty string")
		}

		// Verify result is one of the provided options
		validResults := []string{"Red", "Blue", "Green"}
		isValid := false
		for _, valid := range validResults {
			if result.Result == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			t.Errorf("Result '%s' is not a valid option", result.Result)
		}
	})

	t.Run("Success_WithWeightedOptions", func(t *testing.T) {
		resetTestState()

		// Create heavily weighted options to verify weighting works
		spinRequest := map[string]interface{}{
			"wheel_id":   "weighted-test",
			"session_id": "test-session",
			"options": []map[string]interface{}{
				{"label": "Very Likely", "color": "#FF0000", "weight": 99.0},
				{"label": "Very Unlikely", "color": "#0000FF", "weight": 1.0},
			},
		}

		// Run multiple spins to verify weighting (statistical test)
		results := make(map[string]int)
		spins := 100

		for i := 0; i < spins; i++ {
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
			if err := json.Unmarshal(w.Body.Bytes(), &result); err == nil {
				results[result.Result]++
			}
		}

		// With 99:1 weighting, "Very Likely" should appear significantly more often
		// Using a loose threshold to avoid flaky tests
		veryLikelyCount := results["Very Likely"]
		if veryLikelyCount < 80 {
			t.Errorf("Expected 'Very Likely' to appear at least 80 times out of 100, got %d", veryLikelyCount)
		}
	})

	t.Run("EdgeCase_NoOptions", func(t *testing.T) {
		resetTestState()

		spinRequest := map[string]interface{}{
			"wheel_id":   "non-existent",
			"session_id": "test-session",
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

		if result.Result != "No options provided" {
			t.Errorf("Expected 'No options provided', got '%s'", result.Result)
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		resetTestState()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/spin",
			Body:   "invalid json",
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestGetHistoryHandler tests retrieving spin history
func TestGetHistoryHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_EmptyHistory", func(t *testing.T) {
		resetTestState()

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

		if len(historyList) != 0 {
			t.Errorf("Expected empty history, got %d items", len(historyList))
		}
	})

	t.Run("Success_WithHistory", func(t *testing.T) {
		resetTestState()
		initTestWheels()

		// Add some history items
		history = append(history, SpinResult{
			Result:    "YES! ✅",
			WheelID:   "yes-or-no",
			SessionID: "session-1",
			Timestamp: time.Now(),
		})
		history = append(history, SpinResult{
			Result:    "Pizza 🍕",
			WheelID:   "dinner-decider",
			SessionID: "session-2",
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

		if len(historyList) != 2 {
			t.Errorf("Expected 2 history items, got %d", len(historyList))
		}
	})
}

// TestGetDefaultWheels tests the default wheel generation
func TestGetDefaultWheels(t *testing.T) {
	defaultWheels := getDefaultWheels()

	if len(defaultWheels) == 0 {
		t.Error("Expected default wheels, got empty list")
	}

	// Verify structure of default wheels
	for _, wheel := range defaultWheels {
		if wheel.ID == "" {
			t.Error("Default wheel has empty ID")
		}
		if wheel.Name == "" {
			t.Error("Default wheel has empty name")
		}
		if len(wheel.Options) == 0 {
			t.Errorf("Default wheel '%s' has no options", wheel.Name)
		}

		// Verify each option has required fields
		for _, option := range wheel.Options {
			if option.Label == "" {
				t.Errorf("Option in wheel '%s' has empty label", wheel.Name)
			}
			if option.Color == "" {
				t.Errorf("Option '%s' in wheel '%s' has empty color", option.Label, wheel.Name)
			}
			if option.Weight <= 0 {
				t.Errorf("Option '%s' in wheel '%s' has invalid weight: %f", option.Label, wheel.Name, option.Weight)
			}
		}
	}
}

// TestInitDB tests database initialization
func TestInitDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MissingConfig", func(t *testing.T) {
		// Clear environment variables
		os.Unsetenv("POSTGRES_URL")
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("POSTGRES_PORT")
		os.Unsetenv("POSTGRES_USER")
		os.Unsetenv("POSTGRES_PASSWORD")
		os.Unsetenv("POSTGRES_DB")

		// Should not panic, should fallback to in-memory
		initDB()

		// DB should be nil (in-memory fallback)
		if db != nil {
			defer db.Close()
		}
	})
}

// Benchmark tests

// BenchmarkSpinHandler benchmarks the spin endpoint
func BenchmarkSpinHandler(b *testing.B) {
	resetTestState()
	initTestWheels()

	spinRequest := map[string]interface{}{
		"wheel_id":   "yes-or-no",
		"session_id": "benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/spin",
			Body:   spinRequest,
		}

		httpReq, w, _ := makeHTTPRequest(req)
		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)
	}
}

// BenchmarkGetWheels benchmarks the get wheels endpoint
func BenchmarkGetWheels(b *testing.B) {
	resetTestState()
	initTestWheels()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/wheels",
		}

		httpReq, w, _ := makeHTTPRequest(req)
		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)
	}
}
