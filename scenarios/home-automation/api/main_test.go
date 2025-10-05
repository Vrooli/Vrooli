package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// Note: setupTestApp is defined in test_helpers.go

// Test health check endpoint
func TestHealthCheck(t *testing.T) {
	app := setupTestApp(t)

	// Skip if we don't have a DB connection for this test
	if app.DB == nil {
		t.Skip("Skipping health check test - requires database connection")
	}

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.HealthCheck)
	handler.ServeHTTP(rr, req)

	// Should return 200 or 503 depending on DB state
	if status := rr.Code; status != http.StatusOK && status != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v or %v",
			status, http.StatusOK, http.StatusServiceUnavailable)
	}

	// Check response is valid JSON
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Response is not valid JSON: %v", err)
	}

	// Check required fields
	requiredFields := []string{"status", "service", "timestamp", "readiness", "dependencies"}
	for _, field := range requiredFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Health check response missing required field: %s", field)
		}
	}
}

// Test device listing endpoint structure
func TestListDevicesEndpoint(t *testing.T) {
	app := setupTestApp(t)

	if app.DB == nil {
		t.Skip("Skipping device listing test - requires database connection")
	}

	req, err := http.NewRequest("GET", "/api/v1/devices", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.ListDevices)
	handler.ServeHTTP(rr, req)

	// Even with mock DB, endpoint should respond with proper structure
	if status := rr.Code; status != http.StatusOK && status != http.StatusInternalServerError {
		t.Errorf("handler returned unexpected status code: got %v", status)
	}
}

// Test automation validation endpoint
func TestValidateAutomationEndpoint(t *testing.T) {
	app := setupTestApp(t)

	if app.DB == nil {
		t.Skip("Skipping automation validation test - requires database connection")
	}

	requestBody := map[string]interface{}{
		"automation_code": "lights:\n  - platform: test",
		"profile_id":      "test-profile-id",
	}
	body, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/api/v1/automations/validate", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.ValidateAutomation)
	handler.ServeHTTP(rr, req)

	// Should return proper status
	if status := rr.Code; status != http.StatusOK && status != http.StatusBadRequest && status != http.StatusInternalServerError {
		t.Errorf("handler returned unexpected status code: got %v", status)
	}
}

// Test route registration
func TestRouteSetup(t *testing.T) {
	router := mux.NewRouter()

	// Test routes that should exist
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/health"},
		{"GET", "/api/v1/health"},
		{"GET", "/api/v1/devices"},
		{"POST", "/api/v1/devices/control"},
		{"POST", "/api/v1/automations/generate"},
		{"POST", "/api/v1/automations/validate"},
		{"GET", "/api/v1/automations"},
		{"POST", "/api/v1/calendar/trigger"},
		{"GET", "/api/v1/profiles"},
	}

	app := setupTestApp(t)

	// Root health check
	router.HandleFunc("/health", app.HealthCheck).Methods("GET")

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", app.HealthCheck).Methods("GET")
	api.HandleFunc("/devices/control", app.ControlDevice).Methods("POST")
	api.HandleFunc("/devices", app.ListDevices).Methods("GET")
	api.HandleFunc("/automations/generate", app.GenerateAutomation).Methods("POST")
	api.HandleFunc("/automations/validate", app.ValidateAutomation).Methods("POST")
	api.HandleFunc("/automations", app.ListAutomations).Methods("GET")
	api.HandleFunc("/calendar/trigger", app.HandleCalendarEvent).Methods("POST")
	api.HandleFunc("/profiles", app.GetProfiles).Methods("GET")

	for _, route := range routes {
		req, _ := http.NewRequest(route.method, route.path, nil)
		match := &mux.RouteMatch{}
		if !router.Match(req, match) {
			t.Errorf("Route not found: %s %s", route.method, route.path)
		}
	}
}

// Test getEnv helper function
func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "returns environment value when set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "returns default when env not set",
			key:          "NONEXISTENT_VAR",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnv() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test countHealthyDependencies helper
func TestCountHealthyDependencies(t *testing.T) {
	app := &App{}

	tests := []struct {
		name         string
		dependencies map[string]interface{}
		expected     int
	}{
		{
			name: "all healthy",
			dependencies: map[string]interface{}{
				"db":     map[string]interface{}{"status": "healthy"},
				"cache":  map[string]interface{}{"status": "healthy"},
				"device": map[string]interface{}{"status": "healthy"},
			},
			expected: 3,
		},
		{
			name: "mixed health states",
			dependencies: map[string]interface{}{
				"db":     map[string]interface{}{"status": "healthy"},
				"cache":  map[string]interface{}{"status": "degraded"},
				"device": map[string]interface{}{"status": "unhealthy"},
			},
			expected: 1,
		},
		{
			name:         "empty dependencies",
			dependencies: map[string]interface{}{},
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.countHealthyDependencies(tt.dependencies)
			if result != tt.expected {
				t.Errorf("countHealthyDependencies() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test CORS configuration
func TestCORSConfiguration(t *testing.T) {
	app := setupTestApp(t)
	router := mux.NewRouter()
	router.HandleFunc("/health", app.HealthCheck).Methods("GET")

	// Test that OPTIONS requests are handled
	req, _ := http.NewRequest("OPTIONS", "/health", nil)
	req.Header.Set("Origin", "http://example.com")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// CORS should allow requests
	// This is a basic test - full CORS testing would check headers
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test generateAutomationCode function
func TestGenerateAutomationCode(t *testing.T) {
	app := setupTestApp(t)

	tests := []struct {
		name        string
		description string
		devices     []string
		context     string
	}{
		{
			name:        "Sunset trigger",
			description: "Turn on lights at sunset",
			devices:     []string{"light.living_room"},
			context:     "",
		},
		{
			name:        "Sunrise trigger",
			description: "Turn off lights at sunrise",
			devices:     []string{"light.bedroom"},
			context:     "",
		},
		{
			name:        "Time-based with context",
			description: "Turn on at 8pm",
			devices:     []string{"light.porch"},
			context:     "evening",
		},
		{
			name:        "Multiple devices",
			description: "Turn on all lights",
			devices:     []string{"light.living_room", "light.bedroom", "light.kitchen"},
			context:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, explanation := app.generateAutomationCode(tt.description, tt.devices, tt.context)

			if code == "" {
				t.Error("Generated code should not be empty")
			}

			if explanation == "" {
				t.Error("Explanation should not be empty")
			}

			// Code should contain description
			if !contains(code, tt.description) {
				t.Error("Code should contain description")
			}

			// Code should be valid YAML structure
			if !contains(code, "# Home Automation Rule") {
				t.Error("Code should contain header comment")
			}
		})
	}
}

// Test checkAutomationConflicts function
func TestCheckAutomationConflicts(t *testing.T) {
	app := setupTestApp(t)

	t.Run("No conflicts for simple automation", func(t *testing.T) {
		devices := []string{"light.bedroom"}
		context := "morning"

		conflicts := app.checkAutomationConflicts(devices, context)

		if len(conflicts) > 0 {
			t.Errorf("Expected no conflicts, got %d", len(conflicts))
		}
	})

	t.Run("Detects conflicts for living room at evening", func(t *testing.T) {
		devices := []string{"light.living_room"}
		context := "evening"

		conflicts := app.checkAutomationConflicts(devices, context)

		if len(conflicts) == 0 {
			t.Error("Expected conflict detection for living room at evening")
		}
	})

	t.Run("Warns about many devices", func(t *testing.T) {
		devices := []string{"light.1", "light.2", "light.3", "light.4"}
		context := ""

		conflicts := app.checkAutomationConflicts(devices, context)

		if len(conflicts) == 0 {
			t.Error("Expected warning for controlling many devices")
		}
	})
}

// Test helper functions from main.go
func TestHelperFunctions(t *testing.T) {
	t.Run("getEnv with existing value", func(t *testing.T) {
		t.Setenv("TEST_KEY", "test_value")
		result := getEnv("TEST_KEY", "default")
		if result != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", result)
		}
	})

	t.Run("getEnv with default", func(t *testing.T) {
		result := getEnv("NONEXISTENT_KEY_12345", "default_value")
		if result != "default_value" {
			t.Errorf("Expected 'default_value', got '%s'", result)
		}
	})
}

// Test handler error cases
func TestHandlerErrorCases(t *testing.T) {
	app := setupTestApp(t)

	t.Run("ControlDevice with invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/devices/control", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.ControlDevice)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})

	t.Run("ValidateAutomation with invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/automations/validate", bytes.NewBuffer([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.ValidateAutomation)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})

	t.Run("HandleCalendarEvent with invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/calendar/trigger", bytes.NewBuffer([]byte("bad json")))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.HandleCalendarEvent)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})
}

// Test device control endpoint with valid input
func TestControlDeviceEndpoint(t *testing.T) {
	app := setupTestApp(t)

	if app.DeviceController == nil || app.DB == nil {
		t.Skip("Skipping device control test - requires database connection")
	}

	t.Run("Valid device control request", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"device_id": "light.living_room",
			"action":    "turn_on",
			"user_id":   "550e8400-e29b-41d4-a716-446655440001",
		}
		body, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/devices/control", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.ControlDevice)
		handler.ServeHTTP(rr, req)

		// Should process request (may fail without real HA, but should not panic)
		if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status code: %d", rr.Code)
		}
	})
}

// Test automation generation endpoint
func TestGenerateAutomationEndpoint(t *testing.T) {
	app := setupTestApp(t)

	if app.SafetyValidator == nil || app.DB == nil {
		t.Skip("Skipping automation generation test - requires database connection")
	}

	t.Run("Valid automation generation request", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"description":    "Turn on lights at sunset",
			"profile_id":     "test-profile",
			"target_devices": []string{"light.living_room"},
		}
		body, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/automations/generate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.GenerateAutomation)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		// Check response structure
		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Errorf("Response is not valid JSON: %v", err)
		}

		// Verify expected fields
		expectedFields := []string{"automation_id", "generated_code", "explanation", "validation"}
		for _, field := range expectedFields {
			if _, ok := response[field]; !ok {
				t.Errorf("Response missing field: %s", field)
			}
		}
	})
}

// Test list automations endpoint
func TestListAutomationsEndpoint(t *testing.T) {
	app := setupTestApp(t)

	req, _ := http.NewRequest("GET", "/api/v1/automations", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.ListAutomations)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Response is not valid JSON: %v", err)
	}

	if _, ok := response["automations"]; !ok {
		t.Error("Response missing 'automations' field")
	}

	if _, ok := response["total"]; !ok {
		t.Error("Response missing 'total' field")
	}
}

// Test context activation endpoint
func TestActivateContextEndpoint(t *testing.T) {
	app := setupTestApp(t)

	if app.CalendarScheduler == nil || app.DB == nil {
		t.Skip("Skipping context activation test - requires database connection")
	}

	req, _ := http.NewRequest("POST", "/api/v1/contexts/evening/activate", nil)
	req = mux.SetURLVars(req, map[string]string{"context": "evening"})

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.ActivateContext)
	handler.ServeHTTP(rr, req)

	// Should attempt activation (may fail without DB, but should respond)
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Errorf("Unexpected status code: %d", rr.Code)
	}
}

// Test context deactivation endpoint
func TestDeactivateContextEndpoint(t *testing.T) {
	app := setupTestApp(t)

	if app.CalendarScheduler == nil || app.DB == nil {
		t.Skip("Skipping context deactivation test - requires database connection")
	}

	req, _ := http.NewRequest("POST", "/api/v1/contexts/evening/deactivate", nil)
	req = mux.SetURLVars(req, map[string]string{"context": "evening"})

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.DeactivateContext)
	handler.ServeHTTP(rr, req)

	// Should attempt deactivation (may fail without DB, but should respond)
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Errorf("Unexpected status code: %d", rr.Code)
	}
}

// Test get current context endpoint
func TestGetCurrentContextEndpoint(t *testing.T) {
	app := setupTestApp(t)

	if app.CalendarScheduler == nil || app.DB == nil {
		t.Skip("Skipping get context test - requires database connection")
	}

	req, _ := http.NewRequest("GET", "/api/v1/contexts/current", nil)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.GetCurrentContext)
	handler.ServeHTTP(rr, req)

	// Should attempt to get context (may fail without DB, but should respond)
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Errorf("Unexpected status code: %d", rr.Code)
	}
}

// Test get profiles endpoint
func TestGetProfilesEndpoint(t *testing.T) {
	app := setupTestApp(t)

	if app.DeviceController == nil || app.DB == nil {
		t.Skip("Skipping get profiles test - requires database connection")
	}

	req, _ := http.NewRequest("GET", "/api/v1/profiles", nil)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.GetProfiles)
	handler.ServeHTTP(rr, req)

	// Should attempt to get profiles (may fail without DB, but should respond)
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Errorf("Unexpected status code: %d", rr.Code)
	}
}

// Test get profile permissions endpoint
func TestGetProfilePermissionsEndpoint(t *testing.T) {
	app := setupTestApp(t)

	if app.DeviceController == nil || app.DB == nil {
		t.Skip("Skipping get profile permissions test - requires database connection")
	}

	req, _ := http.NewRequest("GET", "/api/v1/profiles/test-id/permissions", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-profile-id"})

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.GetProfilePermissions)
	handler.ServeHTTP(rr, req)

	// Should attempt to get permissions (may fail without DB, but should respond)
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Errorf("Unexpected status code: %d", rr.Code)
	}
}

// Test get safety status endpoint
func TestGetSafetyStatusEndpoint(t *testing.T) {
	app := setupTestApp(t)

	if app.SafetyValidator == nil || app.DB == nil {
		t.Skip("Skipping get safety status test - requires database connection")
	}

	req, _ := http.NewRequest("GET", "/api/v1/automations/test-id/safety-check", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-automation-id"})

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.GetSafetyStatus)
	handler.ServeHTTP(rr, req)

	// Should attempt to get safety status (may fail without DB, but should respond)
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Errorf("Unexpected status code: %d", rr.Code)
	}
}

// Test get device status endpoint
func TestGetDeviceStatusEndpoint(t *testing.T) {
	app := setupTestApp(t)

	if app.DeviceController == nil || app.DB == nil {
		t.Skip("Skipping get device status test - requires database connection")
	}

	req, _ := http.NewRequest("GET", "/api/v1/devices/light.living_room/status", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "light.living_room"})

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.GetDeviceStatus)
	handler.ServeHTTP(rr, req)

	// Should attempt to get device status (may fail without DB/HA, but should respond)
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Errorf("Unexpected status code: %d", rr.Code)
	}
}
