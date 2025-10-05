package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIntegrationAgentLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ListAgents", func(t *testing.T) {
		w := testHandlerWithRequest(t, agentsHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/agents",
		})

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var response AgentsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse agents response: %v", err)
		}

		initialTotal := response.Total
		t.Logf("Initial agent count: %d", initialTotal)

		// Verify consistency
		if response.Total != response.Running+response.Completed+response.Failed+response.Stopped {
			t.Error("Agent counts don't match total")
		}
	})

	t.Run("GetStatus", func(t *testing.T) {
		w := testHandlerWithRequest(t, statusHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/status",
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		data := response["data"].(map[string]interface{})

		// Verify all status fields are present
		if data["total"] == nil {
			t.Error("Expected total in status")
		}

		// Status counts should be non-negative
		checkNonNegative := func(field string) {
			val := data[field]
			if val == nil {
				t.Errorf("Expected %s field in status", field)
				return
			}
			floatVal := val.(float64)
			if floatVal < 0 {
				t.Errorf("Expected %s to be non-negative, got %f", field, floatVal)
			}
		}

		checkNonNegative("running")
		checkNonNegative("completed")
		checkNonNegative("failed")
		checkNonNegative("stopped")
	})

	t.Run("GetCapabilities", func(t *testing.T) {
		w := testHandlerWithRequest(t, capabilitiesHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/capabilities",
		})

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var response struct {
			Capabilities []CapabilityInfo `json:"capabilities"`
			Total        int              `json:"total"`
		}

		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse capabilities response: %v", err)
		}

		// Total should match number of capabilities
		if response.Total != len(response.Capabilities) {
			t.Errorf("Expected total (%d) to match capability count (%d)", response.Total, len(response.Capabilities))
		}
	})
}

func TestIntegrationSearchCapabilities(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SearchAndVerify", func(t *testing.T) {
		// First, get all capabilities
		w1 := testHandlerWithRequest(t, capabilitiesHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/capabilities",
		})

		var capResponse struct {
			Capabilities []CapabilityInfo `json:"capabilities"`
			Total        int              `json:"total"`
		}

		if err := json.Unmarshal(w1.Body.Bytes(), &capResponse); err != nil {
			t.Fatalf("Failed to parse capabilities: %v", err)
		}

		// If we have capabilities, try searching for the first one
		if capResponse.Total > 0 && len(capResponse.Capabilities) > 0 {
			testCap := capResponse.Capabilities[0].Name

			w2 := testHandlerWithRequest(t, searchByCapabilityHandler, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/agents/search?capability=" + testCap,
				QueryParams: map[string]string{
					"capability": testCap,
				},
			})

			response := assertJSONResponse(t, w2, http.StatusOK, map[string]interface{}{
				"success": true,
			})

			data := response["data"].(map[string]interface{})

			// Verify capability matches what we searched for
			if data["capability"] != testCap {
				t.Errorf("Expected capability to be %s, got %v", testCap, data["capability"])
			}
		}
	})

	t.Run("SearchNonExistentCapability", func(t *testing.T) {
		w := testHandlerWithRequest(t, searchByCapabilityHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/agents/search?capability=non-existent-capability-12345",
			QueryParams: map[string]string{
				"capability": "non-existent-capability-12345",
			},
		})

		// Should still return success with empty results
		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		data := response["data"].(map[string]interface{})
		agents := data["agents"].([]interface{})

		// Should return empty list
		if len(agents) != 0 {
			t.Errorf("Expected 0 agents for non-existent capability, got %d", len(agents))
		}
	})
}

func TestIntegrationCORSHeaders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	endpoints := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
	}{
		{"health", healthHandler, "GET", "/health"},
		{"version", versionHandler, "GET", "/api/v1/version"},
		{"agents", agentsHandler, "GET", "/api/v1/agents"},
		{"status", statusHandler, "GET", "/api/v1/status"},
		{"capabilities", capabilitiesHandler, "GET", "/api/v1/capabilities"},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			// Wrap with CORS middleware
			handler := corsMiddleware(endpoint.handler)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(endpoint.method, endpoint.path, nil)
			handler(w, req)

			// Verify CORS headers are present
			if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
				t.Errorf("Expected Access-Control-Allow-Origin to be '*', got '%s'", got)
			}
		})
	}
}

func TestIntegrationContentTypes(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	endpoints := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
	}{
		{"health", healthHandler, "GET", "/health"},
		{"version", versionHandler, "GET", "/api/v1/version"},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			w := testHandlerWithRequest(t, endpoint.handler, HTTPTestRequest{
				Method: endpoint.method,
				Path:   endpoint.path,
			})

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
			}
		})
	}
}

func TestIntegrationRateLimiting(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RateLimitMiddleware", func(t *testing.T) {
		// Test that rate limiter allows requests within limit
		handler := rateLimitMiddleware(apiRateLimiter, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		// Make several requests
		for i := 0; i < 5; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			handler(w, req)

			// These requests should succeed (though some may be delayed)
			if w.Code != http.StatusOK && w.Code != http.StatusTooManyRequests {
				t.Errorf("Request %d: unexpected status %d", i, w.Code)
			}
		}
	})
}

func TestIntegrationErrorResponses(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InvalidJSONConsistency", func(t *testing.T) {
		// Test that all POST endpoints handle invalid JSON consistently
		w := testHandlerWithRequest(t, agentsHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/agents",
			Body:   `{invalid json}`,
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
		}

		var response APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		if response.Success {
			t.Error("Expected success to be false for error response")
		}

		if response.Error == "" {
			t.Error("Expected error message to be present")
		}
	})

	t.Run("MethodNotAllowedConsistency", func(t *testing.T) {
		tests := []struct {
			name    string
			handler http.HandlerFunc
			method  string
			path    string
		}{
			{"scan_GET", scanHandler, "GET", "/api/v1/scan"},
			{"capabilities_POST", capabilitiesHandler, "POST", "/api/v1/capabilities"},
			{"agents_PATCH", agentsHandler, "PATCH", "/api/v1/agents"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				w := testHandlerWithRequest(t, tt.handler, HTTPTestRequest{
					Method: tt.method,
					Path:   tt.path,
				})

				expectedStatus := http.StatusMethodNotAllowed
				if w.Code != expectedStatus {
					t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
				}
			})
		}
	})
}
