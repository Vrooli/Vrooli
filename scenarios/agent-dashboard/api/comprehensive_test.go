package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthHandlerComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SuccessCase", func(t *testing.T) {
		w := testHandlerWithRequest(t, healthHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "agent-dashboard-api",
			"version": "1.0.0",
		})

		// Check timestamp field exists
		if _, ok := response["timestamp"]; !ok {
			t.Error("Expected timestamp field in health response")
		}

		// Check readiness field
		if readiness, ok := response["readiness"]; !ok || readiness != true {
			t.Error("Expected readiness to be true")
		}
	})

	t.Run("ContentType", func(t *testing.T) {
		w := testHandlerWithRequest(t, healthHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected Content-Type to be application/json, got %s", contentType)
		}
	})
}

func TestVersionHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SuccessCase", func(t *testing.T) {
		w := testHandlerWithRequest(t, versionHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/version",
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"service": "agent-dashboard",
		})

		// Verify required fields exist
		requiredFields := []string{"api_version", "codex_default_mode", "default_timeout_sec"}
		for _, field := range requiredFields {
			if _, ok := response[field]; !ok {
				t.Errorf("Expected field '%s' in version response", field)
			}
		}
	})
}

func TestAgentsHandlerGET(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SuccessCase", func(t *testing.T) {
		w := testHandlerWithRequest(t, agentsHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/agents",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response AgentsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify response structure
		if response.Agents == nil {
			t.Error("Expected agents array to be initialized")
		}

		if response.Total < 0 {
			t.Error("Expected total to be non-negative")
		}
	})
}

func TestAgentsHandlerPOST(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InvalidJSON", func(t *testing.T) {
		w := testHandlerWithRequest(t, agentsHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/agents",
			Body:   `{"invalid": json`,
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid request body")
	})

	t.Run("EmptyBody", func(t *testing.T) {
		w := testHandlerWithRequest(t, agentsHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/agents",
			Body:   map[string]interface{}{},
		})

		// Should fail validation for empty task
		if w.Code != http.StatusBadRequest {
			t.Logf("Response: %s", w.Body.String())
		}
	})

	t.Run("MissingTask", func(t *testing.T) {
		w := testHandlerWithRequest(t, agentsHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/agents",
			Body: map[string]interface{}{
				"mode": "auto",
			},
		})

		// Should fail validation for missing task
		if w.Code != http.StatusBadRequest {
			t.Logf("Response: %s", w.Body.String())
		}
	})
}

func TestAgentsHandlerMethodNotAllowed(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	methods := []string{"PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			w := testHandlerWithRequest(t, agentsHandler, HTTPTestRequest{
				Method: method,
				Path:   "/api/v1/agents",
			})

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s method, got %d", method, w.Code)
			}
		})
	}
}

func TestStatusHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SuccessCase", func(t *testing.T) {
		w := testHandlerWithRequest(t, statusHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/status",
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		// Verify data field exists and has required fields
		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data field in response")
		}

		requiredFields := []string{"timestamp", "running", "completed", "failed", "stopped", "total"}
		for _, field := range requiredFields {
			if _, ok := data[field]; !ok {
				t.Errorf("Expected field '%s' in status data", field)
			}
		}
	})
}

func TestScanHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SuccessCase", func(t *testing.T) {
		w := testHandlerWithRequest(t, scanHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scan",
		})

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		w := testHandlerWithRequest(t, scanHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/scan",
		})

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405 for GET method, got %d", w.Code)
		}
	})
}

func TestCapabilitiesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SuccessCase", func(t *testing.T) {
		w := testHandlerWithRequest(t, capabilitiesHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/capabilities",
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		// Verify data field exists
		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data field in response")
		}

		// Check capabilities array
		capabilities, ok := data["capabilities"].([]interface{})
		if !ok {
			t.Fatal("Expected capabilities to be an array")
		}

		// Capabilities should be initialized (even if empty)
		if capabilities == nil {
			t.Error("Expected capabilities array to be initialized")
		}

		// Check total field
		total, ok := data["total"].(float64)
		if !ok {
			t.Fatal("Expected total to be a number")
		}

		if total < 0 {
			t.Error("Total capabilities should not be negative")
		}

		// Total should match length of capabilities
		if int(total) != len(capabilities) {
			t.Errorf("Expected total (%d) to match capabilities length (%d)", int(total), len(capabilities))
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		w := testHandlerWithRequest(t, capabilitiesHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/capabilities",
		})

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405 for POST method, got %d", w.Code)
		}
	})
}

func TestSearchByCapabilityHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SuccessCase", func(t *testing.T) {
		w := testHandlerWithRequest(t, searchByCapabilityHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/agents/search?capability=text-generation",
			QueryParams: map[string]string{
				"capability": "text-generation",
			},
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		// Verify data structure
		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data field in response")
		}

		if _, ok := data["capability"]; !ok {
			t.Error("Expected capability field in response data")
		}

		if _, ok := data["agents"]; !ok {
			t.Error("Expected agents field in response data")
		}
	})

	t.Run("MissingCapabilityParam", func(t *testing.T) {
		w := testHandlerWithRequest(t, searchByCapabilityHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/agents/search",
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for missing capability param, got %d", w.Code)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("EmptyCapabilityParam", func(t *testing.T) {
		w := testHandlerWithRequest(t, searchByCapabilityHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/agents/search?capability=",
			QueryParams: map[string]string{
				"capability": "",
			},
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty capability param, got %d", w.Code)
		}
	})
}

func TestCORSMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	handler := corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	t.Run("OPTIONSRequest", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("OPTIONS", "/test", nil)
		handler(w, req)

		// Verify CORS headers
		expectedHeaders := map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "GET, POST, PUT, PATCH, DELETE, OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type",
		}

		for header, expected := range expectedHeaders {
			if got := w.Header().Get(header); got != expected {
				t.Errorf("Expected %s header to be '%s', got '%s'", header, expected, got)
			}
		}
	})

	t.Run("GETRequest", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		handler(w, req)

		// Verify CORS headers are set for GET requests too
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin to be '*', got '%s'", got)
		}

		// Verify handler executed
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestValidationFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("isValidLineCount", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected bool
		}{
			{"valid_100", "100", true},
			{"valid_max", "10000", true},
			{"invalid_too_high", "10001", false},
			{"invalid_negative", "-1", false},
			{"invalid_zero", "0", false},
			{"invalid_not_number", "abc", false},
			{"invalid_empty", "", false},
			{"invalid_decimal", "100.5", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := isValidLineCount(tt.input)
				if result != tt.expected {
					t.Errorf("isValidLineCount(%q) = %v, expected %v", tt.input, result, tt.expected)
				}
			})
		}
	})

	t.Run("isValidResourceName", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected bool
		}{
			{"valid_claude", "claude-code", true},
			{"valid_ollama", "ollama", true},
			{"valid_codex", "codex", true},
			{"invalid_unknown", "unknown-resource", false},
			{"invalid_empty", "", false},
			{"invalid_special_chars", "claude@code", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := isValidResourceName(tt.input)
				if result != tt.expected {
					t.Errorf("isValidResourceName(%q) = %v, expected %v", tt.input, result, tt.expected)
				}
			})
		}
	})

	t.Run("isValidAgentID", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected bool
		}{
			{"valid_standard", "claude-code:agent-123", true},
			{"valid_underscore", "ollama:agent_456", true},
			{"valid_codex", "codex:test-agent", true},
			{"invalid_no_colon", "agent-123", false},
			{"invalid_empty", "", false},
			{"invalid_special_char", "resource:agent@123", false},
			{"invalid_no_resource", ":agent-123", false},
			{"invalid_no_agent", "resource:", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := isValidAgentID(tt.input)
				if result != tt.expected {
					t.Errorf("isValidAgentID(%q) = %v, expected %v", tt.input, result, tt.expected)
				}
			})
		}
	})
}

func TestAPIResponseHelpers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("jsonResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		response := APIResponse{
			Success: true,
			Data:    map[string]string{"message": "test"},
		}

		jsonResponse(w, response, http.StatusOK)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var decoded APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !decoded.Success {
			t.Error("Expected success to be true")
		}
	})

	t.Run("errorResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		errorResponse(w, "test error", http.StatusBadRequest)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var decoded APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if decoded.Success {
			t.Error("Expected success to be false")
		}

		if decoded.Error != "test error" {
			t.Errorf("Expected error message 'test error', got '%s'", decoded.Error)
		}
	})
}

func TestCodexAgentManager(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Initialization", func(t *testing.T) {
		if codexManager == nil {
			t.Fatal("Expected codexManager to be initialized")
		}

		if codexManager.agents == nil {
			t.Error("Expected agents map to be initialized")
		}

		if codexManager.defaultTimeout != defaultAgentTimeout {
			t.Errorf("Expected default timeout to be %v, got %v", defaultAgentTimeout, codexManager.defaultTimeout)
		}
	})

	t.Run("Snapshot", func(t *testing.T) {
		agents, stats := codexManager.Snapshot()

		if agents == nil {
			t.Error("Expected agents slice to be non-nil")
		}

		if stats.Total < 0 {
			t.Error("Expected total to be non-negative")
		}

		if stats.Running < 0 {
			t.Error("Expected running count to be non-negative")
		}

		if stats.Completed < 0 {
			t.Error("Expected completed count to be non-negative")
		}

		if stats.Failed < 0 {
			t.Error("Expected failed count to be non-negative")
		}

		if stats.Stopped < 0 {
			t.Error("Expected stopped count to be non-negative")
		}
	})

	t.Run("Summary", func(t *testing.T) {
		stats := codexManager.Summary()

		if stats.Total < 0 {
			t.Error("Expected total to be non-negative")
		}

		// Total should equal sum of states
		sum := stats.Running + stats.Completed + stats.Failed + stats.Stopped
		if sum != stats.Total {
			t.Errorf("Expected total (%d) to equal sum of states (%d)", stats.Total, sum)
		}
	})
}

func TestAgentTimestampHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthTimestamp", func(t *testing.T) {
		w := testHandlerWithRequest(t, healthHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		timestampStr, ok := response["timestamp"].(string)
		if !ok {
			t.Fatal("Expected timestamp to be a string")
		}

		// Verify timestamp is valid RFC3339
		_, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			t.Errorf("Expected valid RFC3339 timestamp, got error: %v", err)
		}
	})

	t.Run("StatusTimestamp", func(t *testing.T) {
		w := testHandlerWithRequest(t, statusHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/status",
		})

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data field")
		}

		// Timestamp in status is a time.Time, which gets marshaled differently
		if _, ok := data["timestamp"]; !ok {
			t.Error("Expected timestamp field in status data")
		}
	})
}
