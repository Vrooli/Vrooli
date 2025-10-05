package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestHandleGetScenarios_Comprehensive tests the GetScenarios handler comprehensively
func TestHandleGetScenarios_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("EmptyScenarios", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/scenarios",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		scenarios, ok := resp["scenarios"].([]interface{})
		if !ok {
			t.Fatal("Expected scenarios array in response")
		}

		if len(scenarios) != 0 {
			t.Errorf("Expected 0 scenarios, got %d", len(scenarios))
		}
	})

	t.Run("WithScenarios", func(t *testing.T) {
		populateTestScenarios(env.Orchestrator)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/scenarios",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		scenarios, ok := resp["scenarios"].([]interface{})
		if !ok {
			t.Fatal("Expected scenarios array in response")
		}

		if len(scenarios) != 3 {
			t.Errorf("Expected 3 scenarios, got %d", len(scenarios))
		}

		// Verify scenario structure
		scenario := scenarios[0].(map[string]interface{})
		if _, ok := scenario["id"]; !ok {
			t.Error("Expected 'id' field in scenario")
		}
		if _, ok := scenario["name"]; !ok {
			t.Error("Expected 'name' field in scenario")
		}
		if _, ok := scenario["isActive"]; !ok {
			t.Error("Expected 'isActive' field in scenario")
		}
	})
}

// TestHandleActivateScenario_Comprehensive tests scenario activation comprehensively
func TestHandleActivateScenario_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)

	t.Run("SuccessfulActivation", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-scenario-1/activate",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		if success, ok := resp["success"].(bool); !ok || !success {
			t.Error("Expected success=true in response")
		}

		if newState, ok := resp["newState"].(string); !ok || newState != "active" {
			t.Errorf("Expected newState='active', got '%v'", resp["newState"])
		}

		// Verify scenario is actually active
		scenario, exists := env.Orchestrator.GetScenario("test-scenario-1")
		if !exists {
			t.Fatal("Scenario should exist")
		}
		if !scenario.IsActive {
			t.Error("Scenario should be active")
		}
	})

	t.Run("NonExistentScenario", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/non-existent-scenario/activate",
		}

		w := makeHTTPRequest(env, req)
		assertErrorResponse(t, w, http.StatusNotFound, "Scenario not found")
	})

	t.Run("AlreadyActiveScenario", func(t *testing.T) {
		// First activation
		env.Orchestrator.ActivateScenario("test-scenario-2")

		// Second activation should still succeed (idempotent)
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-scenario-2/activate",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		if success, ok := resp["success"].(bool); !ok || !success {
			t.Error("Expected success=true even when already active")
		}
	})
}

// TestHandleDeactivateScenario_Comprehensive tests scenario deactivation comprehensively
func TestHandleDeactivateScenario_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)

	t.Run("SuccessfulDeactivation", func(t *testing.T) {
		// First activate
		env.Orchestrator.ActivateScenario("test-scenario-1")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-scenario-1/deactivate",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		if success, ok := resp["success"].(bool); !ok || !success {
			t.Error("Expected success=true in response")
		}

		if newState, ok := resp["newState"].(string); !ok || newState != "inactive" {
			t.Errorf("Expected newState='inactive', got '%v'", resp["newState"])
		}

		// Verify scenario is actually inactive
		scenario, exists := env.Orchestrator.GetScenario("test-scenario-1")
		if !exists {
			t.Fatal("Scenario should exist")
		}
		if scenario.IsActive {
			t.Error("Scenario should be inactive")
		}
	})

	t.Run("NonExistentScenario", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/non-existent-scenario/deactivate",
		}

		w := makeHTTPRequest(env, req)
		assertErrorResponse(t, w, http.StatusNotFound, "Scenario not found")
	})

	t.Run("AlreadyInactiveScenario", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-scenario-3/deactivate",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		if success, ok := resp["success"].(bool); !ok || !success {
			t.Error("Expected success=true even when already inactive")
		}
	})
}

// TestHandleGetPresets_Comprehensive tests preset listing comprehensively
func TestHandleGetPresets_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	initializeDefaultPresets(env.Orchestrator)

	t.Run("ListAllPresets", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/presets",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		presets, ok := resp["presets"].([]interface{})
		if !ok {
			t.Fatal("Expected presets array in response")
		}

		if len(presets) < 1 {
			t.Error("Expected at least 1 default preset")
		}

		// Verify preset structure
		preset := presets[0].(map[string]interface{})
		if _, ok := preset["id"]; !ok {
			t.Error("Expected 'id' field in preset")
		}
		if _, ok := preset["name"]; !ok {
			t.Error("Expected 'name' field in preset")
		}
		if _, ok := preset["description"]; !ok {
			t.Error("Expected 'description' field in preset")
		}
	})
}

// TestHandleGetActivePresets_Comprehensive tests active preset listing
func TestHandleGetActivePresets_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	initializeDefaultPresets(env.Orchestrator)
	populateTestScenarios(env.Orchestrator)

	t.Run("NoActivePresets", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/presets/active",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		activePresets, ok := resp["activePresets"].([]interface{})
		if !ok {
			t.Fatal("Expected activePresets array in response")
		}

		if len(activePresets) != 0 {
			t.Errorf("Expected 0 active presets, got %d", len(activePresets))
		}
	})

	t.Run("WithActivePresets", func(t *testing.T) {
		// Apply a preset to make it active
		presets := env.Orchestrator.GetPresets()
		if len(presets) > 0 {
			env.Orchestrator.ApplyPreset(presets[0].ID)
		}

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/presets/active",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		activePresets, ok := resp["activePresets"].([]interface{})
		if !ok {
			t.Fatal("Expected activePresets array in response")
		}

		if len(activePresets) < 1 {
			t.Error("Expected at least 1 active preset")
		}
	})
}

// TestHandleApplyPreset_Comprehensive tests preset application
func TestHandleApplyPreset_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	initializeDefaultPresets(env.Orchestrator)
	populateTestScenarios(env.Orchestrator)

	t.Run("SuccessfulPresetApplication", func(t *testing.T) {
		presets := env.Orchestrator.GetPresets()
		if len(presets) == 0 {
			t.Skip("No presets available")
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/presets/" + presets[0].ID + "/apply",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		if success, ok := resp["success"].(bool); !ok || !success {
			t.Error("Expected success=true in response")
		}

		if _, ok := resp["preset"]; !ok {
			t.Error("Expected 'preset' field in response")
		}
	})

	t.Run("NonExistentPreset", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/presets/non-existent-preset/apply",
		}

		w := makeHTTPRequest(env, req)
		assertErrorResponse(t, w, http.StatusNotFound, "Preset not found")
	})
}

// TestHandleGetStatus_Comprehensive tests status endpoint
func TestHandleGetStatus_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)

	t.Run("StatusWithoutActivity", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/status",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := resp["totalScenarios"]; !ok {
			t.Error("Expected 'totalScenarios' field in status response")
		}

		if _, ok := resp["activeScenarios"]; !ok {
			t.Error("Expected 'activeScenarios' field in status response")
		}

		if _, ok := resp["uptime"]; !ok {
			t.Error("Expected 'uptime' field in status response")
		}

		if _, ok := resp["recentActivity"]; !ok {
			t.Error("Expected 'recentActivity' field in status response")
		}
	})

	t.Run("StatusWithActivity", func(t *testing.T) {
		// Activate a scenario to generate activity
		env.Orchestrator.ActivateScenario("test-scenario-1")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/status",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		if activeScenarios, ok := resp["activeScenarios"].(float64); !ok || activeScenarios < 1 {
			t.Errorf("Expected at least 1 active scenario, got %v", resp["activeScenarios"])
		}

		if recentActivity, ok := resp["recentActivity"].([]interface{}); ok {
			if len(recentActivity) == 0 {
				t.Error("Expected recent activity after activation")
			}
		}
	})
}

// TestHandleStopAll_Comprehensive tests stop-all functionality
func TestHandleStopAll_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)

	t.Run("StopAllWhenNoneActive", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/stop-all",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		if success, ok := resp["success"].(bool); !ok || !success {
			t.Error("Expected success=true in response")
		}

		deactivated, ok := resp["deactivated"].([]interface{})
		if !ok {
			t.Fatal("Expected 'deactivated' array in response")
		}

		if len(deactivated) != 0 {
			t.Error("Expected 0 deactivated scenarios")
		}
	})

	t.Run("StopAllWhenSomeActive", func(t *testing.T) {
		// Activate some scenarios
		env.Orchestrator.ActivateScenario("test-scenario-1")
		env.Orchestrator.ActivateScenario("test-scenario-2")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/stop-all",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		if success, ok := resp["success"].(bool); !ok || !success {
			t.Error("Expected success=true in response")
		}

		deactivated, ok := resp["deactivated"].([]interface{})
		if !ok {
			t.Fatal("Expected 'deactivated' array in response")
		}

		if len(deactivated) != 2 {
			t.Errorf("Expected 2 deactivated scenarios, got %d", len(deactivated))
		}

		// Verify all scenarios are now inactive
		scenarios := env.Orchestrator.GetScenarios()
		for _, s := range scenarios {
			if s.IsActive {
				t.Errorf("Scenario %s should be inactive after stop-all", s.ID)
			}
		}
	})
}

// TestHealthHandler_Comprehensive tests health endpoint
func TestHealthHandler_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("HealthCheck", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		// Health endpoint returns status which can be "healthy", "degraded", or "unhealthy"
		if status, ok := resp["status"].(string); !ok {
			t.Error("Expected 'status' field in health response")
		} else if status != "healthy" && status != "degraded" && status != "unhealthy" {
			t.Errorf("Expected status to be 'healthy', 'degraded', or 'unhealthy', got '%s'", status)
		}

		// Check for metrics which contains uptime
		if metrics, ok := resp["metrics"].(map[string]interface{}); ok {
			if _, ok := metrics["uptime_seconds"]; !ok {
				t.Error("Expected 'uptime_seconds' in metrics")
			}
		} else {
			t.Error("Expected 'metrics' field in health response")
		}
	})
}

// TestCORSMiddleware_Comprehensive tests CORS headers comprehensively
func TestCORSMiddleware_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("CORSHeaders", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/scenarios",
		}

		w := makeHTTPRequest(env, req)

		// Check CORS headers
		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin='*', got '%s'", origin)
		}

		if methods := w.Header().Get("Access-Control-Allow-Methods"); methods == "" {
			t.Error("Expected Access-Control-Allow-Methods header to be set")
		}

		if headers := w.Header().Get("Access-Control-Allow-Headers"); headers == "" {
			t.Error("Expected Access-Control-Allow-Headers header to be set")
		}
	})
}

// TestErrorHandling tests various error scenarios
func TestErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := &HandlerTestSuite{
		HandlerName: "MaintenanceOrchestrator",
		BaseURL:     "/api/v1",
	}

	patterns := []ErrorTestPattern{
		nonExistentScenarioPattern("POST", "/api/v1/scenarios/non-existent/activate"),
		nonExistentScenarioPattern("POST", "/api/v1/scenarios/non-existent/deactivate"),
		nonExistentPresetPattern("/api/v1/presets/non-existent/apply"),
	}

	suite.RunErrorTests(t, env, patterns)
}

// TestJSONResponseFormats tests JSON response consistency
func TestJSONResponseFormats(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)

	endpoints := []struct {
		name   string
		method string
		path   string
	}{
		{"Scenarios", "GET", "/api/v1/scenarios"},
		{"Presets", "GET", "/api/v1/presets"},
		{"ActivePresets", "GET", "/api/v1/presets/active"},
		{"Status", "GET", "/api/v1/status"},
		{"Health", "GET", "/health"},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name+"_JSONFormat", func(t *testing.T) {
			req := HTTPTestRequest{
				Method: endpoint.method,
				Path:   endpoint.path,
			}

			w := makeHTTPRequest(env, req)

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("%s: Expected Content-Type 'application/json', got '%s'",
					endpoint.name, w.Header().Get("Content-Type"))
			}

			var result map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Errorf("%s: Failed to parse JSON response: %v", endpoint.name, err)
			}
		})
	}
}
