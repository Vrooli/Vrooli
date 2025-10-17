package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCORSMiddleware_AllMethods tests CORS middleware with all HTTP methods
func TestCORSMiddleware_AllMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodOptions,
		http.MethodHead,
	}

	for _, method := range methods {
		t.Run("CORS_"+method, func(t *testing.T) {
			// Create a simple test handler
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Wrap with CORS middleware
			handler := corsMiddleware(testHandler)

			// Create test request
			req := httptest.NewRequest(method, "/test", nil)
			req.Header.Set("Origin", "http://example.com")

			w := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(w, req)

			// Check CORS headers - should be specific origin for security
			origin := w.Header().Get("Access-Control-Allow-Origin")
			if origin == "" {
				t.Error("Expected Access-Control-Allow-Origin header to be set")
			}
			// Should NOT be wildcard for security
			if origin == "*" {
				t.Error("Access-Control-Allow-Origin should not be wildcard (*) for security")
			}

			if methods := w.Header().Get("Access-Control-Allow-Methods"); methods == "" {
				t.Error("Expected Access-Control-Allow-Methods header to be set")
			}

			if headers := w.Header().Get("Access-Control-Allow-Headers"); headers == "" {
				t.Error("Expected Access-Control-Allow-Headers header to be set")
			}
		})
	}
}

// TestOptionsHandler_Direct tests the OPTIONS handler directly
func TestOptionsHandler_Direct(t *testing.T) {
	t.Run("OptionsRequestHandling", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		w := httptest.NewRecorder()

		// Call options handler
		optionsHandler(w, req)

		// Should return 200
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestCORSMiddleware_WithoutOrigin tests CORS without Origin header
func TestCORSMiddleware_WithoutOrigin(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := corsMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// CORS headers should still be set
	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin == "" {
		t.Error("Expected Access-Control-Allow-Origin header to be set")
	}
	// Should NOT be wildcard for security
	if origin == "*" {
		t.Error("Access-Control-Allow-Origin should not be wildcard (*) for security")
	}
}

// TestCORSMiddleware_PreflightRequest tests preflight OPTIONS requests
func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := corsMiddleware(testHandler)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")

	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check all CORS headers
	headers := []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
	}

	for _, header := range headers {
		if value := w.Header().Get(header); value == "" {
			t.Errorf("Expected %s header to be set", header)
		}
	}
}

// TestHandlerChaining tests middleware with actual handlers
func TestHandlerChaining(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)

	t.Run("GETWithCORS", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/scenarios",
			Headers: map[string]string{
				"Origin": "http://example.com",
			},
		}

		w := makeHTTPRequest(env, req)

		// Check both response and CORS headers
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin == "" {
			t.Error("Expected Access-Control-Allow-Origin header to be set")
		}
		// Should NOT be wildcard for security
		if origin == "*" {
			t.Error("Access-Control-Allow-Origin should not be wildcard (*) for security")
		}
	})

	t.Run("POSTWithCORS", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-scenario-1/activate",
			Headers: map[string]string{
				"Origin": "http://example.com",
			},
		}

		w := makeHTTPRequest(env, req)

		// Should have CORS headers
		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin == "" {
			t.Error("Expected Access-Control-Allow-Origin header")
		}
	})
}

// TestHealthCheckComponents tests individual health check components
func TestHealthCheckComponents(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CheckScenarioDiscoveryComponents", func(t *testing.T) {
		result := checkScenarioDiscovery()

		// Should have connected field
		if _, ok := result["connected"]; !ok {
			t.Error("Expected 'connected' field in checkScenarioDiscovery result")
		}

		// Should have error field
		if _, ok := result["error"]; !ok {
			t.Error("Expected 'error' field in checkScenarioDiscovery result")
		}
	})

	t.Run("CheckFilesystemAccessComponents", func(t *testing.T) {
		result := checkFilesystemAccess()

		// Should have connected field
		if _, ok := result["connected"]; !ok {
			t.Error("Expected 'connected' field in checkFilesystemAccess result")
		}

		// Should have error field
		if _, ok := result["error"]; !ok {
			t.Error("Expected 'error' field in checkFilesystemAccess result")
		}
	})
}

// TestPresetManagementCheck tests preset management health check
func TestPresetManagementCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CheckPresetManagement", func(t *testing.T) {
		result := checkPresetManagement()

		// Should return connected true (in-memory presets always work)
		if connected, ok := result["connected"].(bool); !ok || !connected {
			t.Error("Expected preset management to be connected")
		}

		// Should not have error
		if errorField := result["error"]; errorField != nil {
			t.Errorf("Expected no error, got %v", errorField)
		}
	})
}

// TestCompleteRequestLifecycle tests complete request processing
func TestCompleteRequestLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	initializeDefaultPresets(env.Orchestrator)
	populateTestScenarios(env.Orchestrator)

	t.Run("CompleteWorkflow", func(t *testing.T) {
		// 1. Check initial status
		statusReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/status",
		}
		w := makeHTTPRequest(env, statusReq)
		if w.Code != http.StatusOK {
			t.Errorf("Initial status check failed: %d", w.Code)
		}

		// 2. List scenarios
		scenariosReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/scenarios",
		}
		w = makeHTTPRequest(env, scenariosReq)
		if w.Code != http.StatusOK {
			t.Errorf("List scenarios failed: %d", w.Code)
		}

		// 3. Activate a scenario
		activateReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-scenario-1/activate",
		}
		w = makeHTTPRequest(env, activateReq)
		if w.Code != http.StatusOK {
			t.Errorf("Activate scenario failed: %d", w.Code)
		}

		// 4. Check updated status
		w = makeHTTPRequest(env, statusReq)
		if w.Code != http.StatusOK {
			t.Errorf("Updated status check failed: %d", w.Code)
		}

		// 5. List presets
		presetsReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/presets",
		}
		w = makeHTTPRequest(env, presetsReq)
		if w.Code != http.StatusOK {
			t.Errorf("List presets failed: %d", w.Code)
		}

		// 6. Deactivate scenario
		deactivateReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-scenario-1/deactivate",
		}
		w = makeHTTPRequest(env, deactivateReq)
		if w.Code != http.StatusOK {
			t.Errorf("Deactivate scenario failed: %d", w.Code)
		}

		// 7. Final health check
		healthReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}
		w = makeHTTPRequest(env, healthReq)
		if w.Code != http.StatusOK {
			t.Errorf("Final health check failed: %d", w.Code)
		}
	})
}
