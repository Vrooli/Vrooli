package main

import (
	"net/http"
	"testing"
)

func TestHandleGetScenarios(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("EmptyScenarios", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/scenarios",
		})

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

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/scenarios",
		})

		resp := assertJSONResponse(t, w, http.StatusOK)
		scenarios, ok := resp["scenarios"].([]interface{})
		if !ok {
			t.Fatal("Expected scenarios array in response")
		}
		if len(scenarios) != 3 {
			t.Errorf("Expected 3 scenarios, got %d", len(scenarios))
		}

		// Validate structure
		if len(scenarios) > 0 {
			scenario := scenarios[0].(map[string]interface{})
			requiredFields := []string{"id", "name", "displayName", "description", "isActive", "endpoint", "tags"}
			for _, field := range requiredFields {
				if _, exists := scenario[field]; !exists {
					t.Errorf("Expected field '%s' in scenario response", field)
				}
			}
		}
	})
}

func TestHandleActivateScenario(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-scenario-1/activate",
		})

		resp := assertJSONResponse(t, w, http.StatusOK)
		if success, ok := resp["success"].(bool); !ok || !success {
			t.Error("Expected success: true")
		}
		if scenario, ok := resp["scenario"].(string); !ok || scenario != "test-scenario-1" {
			t.Errorf("Expected scenario: test-scenario-1, got %v", resp["scenario"])
		}
		if newState, ok := resp["newState"].(string); !ok || newState != "active" {
			t.Errorf("Expected newState: active, got %v", resp["newState"])
		}

		// Verify scenario is actually active
		scenario, _ := env.Orchestrator.GetScenario("test-scenario-1")
		if !scenario.IsActive {
			t.Error("Scenario should be active")
		}
	})

	t.Run("NonExistentScenario", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/nonexistent/activate",
		})

		assertErrorResponse(t, w, http.StatusNotFound, "Scenario not found")
	})
}

func TestHandleDeactivateScenario(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)

	t.Run("Success", func(t *testing.T) {
		// First activate it
		env.Orchestrator.ActivateScenario("test-scenario-1")

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-scenario-1/deactivate",
		})

		resp := assertJSONResponse(t, w, http.StatusOK)
		if success, ok := resp["success"].(bool); !ok || !success {
			t.Error("Expected success: true")
		}
		if newState, ok := resp["newState"].(string); !ok || newState != "inactive" {
			t.Errorf("Expected newState: inactive, got %v", resp["newState"])
		}

		// Verify scenario is actually inactive
		scenario, _ := env.Orchestrator.GetScenario("test-scenario-1")
		if scenario.IsActive {
			t.Error("Scenario should be inactive")
		}
	})

	t.Run("NonExistentScenario", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/nonexistent/deactivate",
		})

		assertErrorResponse(t, w, http.StatusNotFound, "Scenario not found")
	})
}

func TestHandleGetPresets(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("EmptyPresets", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/presets",
		})

		resp := assertJSONResponse(t, w, http.StatusOK)
		presets, ok := resp["presets"].([]interface{})
		if !ok {
			t.Fatal("Expected presets array in response")
		}
		if len(presets) != 0 {
			t.Errorf("Expected 0 presets, got %d", len(presets))
		}
	})

	t.Run("WithPresets", func(t *testing.T) {
		populateTestPresets(env.Orchestrator)

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/presets",
		})

		resp := assertJSONResponse(t, w, http.StatusOK)
		presets, ok := resp["presets"].([]interface{})
		if !ok {
			t.Fatal("Expected presets array in response")
		}
		if len(presets) != 2 {
			t.Errorf("Expected 2 presets, got %d", len(presets))
		}

		// Validate structure
		if len(presets) > 0 {
			preset := presets[0].(map[string]interface{})
			requiredFields := []string{"id", "name", "description", "states", "isDefault", "isActive"}
			for _, field := range requiredFields {
				if _, exists := preset[field]; !exists {
					t.Errorf("Expected field '%s' in preset response", field)
				}
			}
		}
	})
}

func TestHandleGetActivePresets(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestPresets(env.Orchestrator)

	t.Run("NoActivePresets", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/presets/active",
		})

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
		// Activate a preset
		preset, _ := env.Orchestrator.GetPreset("preset-1")
		preset.IsActive = true

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/presets/active",
		})

		resp := assertJSONResponse(t, w, http.StatusOK)
		activePresets, ok := resp["activePresets"].([]interface{})
		if !ok {
			t.Fatal("Expected activePresets array in response")
		}
		if len(activePresets) != 1 {
			t.Errorf("Expected 1 active preset, got %d", len(activePresets))
		}
	})
}

func TestHandleApplyPreset(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)
	populateTestPresets(env.Orchestrator)

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/presets/preset-1/apply",
		})

		resp := assertJSONResponse(t, w, http.StatusOK)
		if success, ok := resp["success"].(bool); !ok || !success {
			t.Error("Expected success: true")
		}
		if preset, ok := resp["preset"].(string); !ok || preset != "preset-1" {
			t.Errorf("Expected preset: preset-1, got %v", resp["preset"])
		}

		// Verify scenarios were activated
		scenario1, _ := env.Orchestrator.GetScenario("test-scenario-1")
		scenario2, _ := env.Orchestrator.GetScenario("test-scenario-2")
		if !scenario1.IsActive {
			t.Error("test-scenario-1 should be active")
		}
		if !scenario2.IsActive {
			t.Error("test-scenario-2 should be active")
		}
	})

	t.Run("NonExistentPreset", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/presets/nonexistent/apply",
		})

		assertErrorResponse(t, w, http.StatusNotFound, "Preset not found")
	})

	t.Run("TogglePreset", func(t *testing.T) {
		// Create fresh environment for this test
		env2 := setupTestEnvironment(t)
		defer env2.Cleanup()

		populateTestScenarios(env2.Orchestrator)
		populateTestPresets(env2.Orchestrator)

		// Apply preset-1 (activate)
		makeHTTPRequest(env2, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/presets/preset-1/apply",
		})

		// Apply preset-1 again (deactivate)
		w := makeHTTPRequest(env2, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/presets/preset-1/apply",
		})

		resp := assertJSONResponse(t, w, http.StatusOK)
		if success, ok := resp["success"].(bool); !ok || !success {
			t.Error("Expected success: true")
		}

		// Verify scenarios were deactivated
		scenario1, _ := env2.Orchestrator.GetScenario("test-scenario-1")
		if scenario1.IsActive {
			t.Error("test-scenario-1 should be inactive after toggling preset")
		}
	})
}

func TestHandleGetStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)

	t.Run("Success", func(t *testing.T) {
		// Activate one scenario
		env.Orchestrator.ActivateScenario("test-scenario-1")

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/status",
		})

		resp := assertJSONResponse(t, w, http.StatusOK)

		requiredFields := []string{"health", "maintenanceState", "totalScenarios", "activeScenarios", "inactiveScenarios", "recentActivity", "uptime"}
		for _, field := range requiredFields {
			if _, exists := resp[field]; !exists {
				t.Errorf("Expected field '%s' in status response", field)
			}
		}

		if total, ok := resp["totalScenarios"].(float64); !ok || int(total) != 3 {
			t.Errorf("Expected totalScenarios: 3, got %v", resp["totalScenarios"])
		}
		if active, ok := resp["activeScenarios"].(float64); !ok || int(active) != 1 {
			t.Errorf("Expected activeScenarios: 1, got %v", resp["activeScenarios"])
		}
		if inactive, ok := resp["inactiveScenarios"].(float64); !ok || int(inactive) != 2 {
			t.Errorf("Expected inactiveScenarios: 2, got %v", resp["inactiveScenarios"])
		}
	})
}

func TestHandleStopAll(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)
	populateTestPresets(env.Orchestrator)

	t.Run("Success", func(t *testing.T) {
		// Activate some scenarios and presets
		env.Orchestrator.ActivateScenario("test-scenario-1")
		env.Orchestrator.ActivateScenario("test-scenario-2")
		preset, _ := env.Orchestrator.GetPreset("preset-1")
		preset.IsActive = true

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/stop-all",
		})

		resp := assertJSONResponse(t, w, http.StatusOK)
		if success, ok := resp["success"].(bool); !ok || !success {
			t.Error("Expected success: true")
		}

		deactivated, ok := resp["deactivated"].([]interface{})
		if !ok {
			t.Fatal("Expected deactivated array in response")
		}
		if len(deactivated) != 2 {
			t.Errorf("Expected 2 deactivated scenarios, got %d", len(deactivated))
		}

		// Verify all scenarios are inactive
		_, active, _ := env.Orchestrator.GetStatus()
		if active != 0 {
			t.Errorf("Expected 0 active scenarios, got %d", active)
		}

		// Verify all presets are inactive
		preset, _ = env.Orchestrator.GetPreset("preset-1")
		if preset.IsActive {
			t.Error("Preset should be inactive")
		}
	})
}

func TestHandleGetScenarioPresetAssignments(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)
	populateTestPresets(env.Orchestrator)

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/scenarios/test-scenario-1/preset-assignments",
		})

		resp := assertJSONResponse(t, w, http.StatusOK)
		assignments, ok := resp["assignments"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected assignments map in response")
		}

		// test-scenario-1 should be in preset-1
		if val, ok := assignments["preset-1"].(bool); !ok || !val {
			t.Error("Expected test-scenario-1 to be assigned to preset-1")
		}
	})
}

func TestHandleUpdateScenarioPresetAssignments(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)
	populateTestPresets(env.Orchestrator)

	t.Run("Success", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"assignments": map[string]bool{
				"preset-1": false, // Remove from preset-1
				"preset-2": true,  // Add to preset-2
			},
		}

		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-scenario-1/preset-assignments",
			Body:   requestBody,
		})

		resp := assertJSONResponse(t, w, http.StatusOK)
		if success, ok := resp["success"].(bool); !ok || !success {
			t.Error("Expected success: true")
		}

		// Verify preset states were updated
		preset1, _ := env.Orchestrator.GetPreset("preset-1")
		preset2, _ := env.Orchestrator.GetPreset("preset-2")

		if _, exists := preset1.States["test-scenario-1"]; exists {
			t.Error("test-scenario-1 should not be in preset-1")
		}
		if !preset2.States["test-scenario-1"] {
			t.Error("test-scenario-1 should be in preset-2")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/scenarios/test-scenario-1/preset-assignments",
			Body:    "invalid json",
			Headers: map[string]string{"Content-Type": "text/plain"},
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("HealthCheck", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		resp := assertJSONResponse(t, w, http.StatusOK)

		requiredFields := []string{"status", "service", "timestamp", "readiness", "version", "dependencies", "metrics"}
		for _, field := range requiredFields {
			if _, exists := resp[field]; !exists {
				t.Errorf("Expected field '%s' in health response", field)
			}
		}

		if service, ok := resp["service"].(string); !ok || service != serviceName {
			t.Errorf("Expected service: %s, got %v", serviceName, resp["service"])
		}

		if version, ok := resp["version"].(string); !ok || version != apiVersion {
			t.Errorf("Expected version: %s, got %v", apiVersion, resp["version"])
		}

		// Check dependencies structure
		deps, ok := resp["dependencies"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected dependencies to be a map")
		}

		expectedDeps := []string{"scenario_discovery", "preset_management", "state_management", "filesystem"}
		for _, depName := range expectedDeps {
			if _, exists := deps[depName]; !exists {
				t.Errorf("Expected dependency '%s' in health response", depName)
			}
		}
	})
}

func TestCORSMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("CORSHeaders", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/scenarios",
		})

		// Check CORS headers
		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin: *, got %s", origin)
		}

		if methods := w.Header().Get("Access-Control-Allow-Methods"); methods == "" {
			t.Error("Expected Access-Control-Allow-Methods header to be set")
		}

		if headers := w.Header().Get("Access-Control-Allow-Headers"); headers == "" {
			t.Error("Expected Access-Control-Allow-Headers header to be set")
		}
	})
}
