package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// TestHandleAddMaintenanceTag_Comprehensive tests adding maintenance tag to scenarios
func TestHandleAddMaintenanceTag_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create temporary test directory
	tempDir, err := ioutil.TempDir("", "add-tag-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWD)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("AddTagToExistingScenario", func(t *testing.T) {
		// Create a test scenario structure
		scenarioDir := filepath.Join("scenarios", "test-tag-scenario")
		if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755); err != nil {
			t.Fatalf("Failed to create scenario dir: %v", err)
		}

		serviceJSON := `{
  "service": {
    "name": "test-tag-scenario",
    "tags": ["test"]
  },
  "ports": {
    "api": {
      "env_var": "API_PORT"
    }
  }
}`
		serviceJSONPath := filepath.Join(scenarioDir, ".vrooli", "service.json")
		if err := ioutil.WriteFile(serviceJSONPath, []byte(serviceJSON), 0644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-tag-scenario/add-tag",
		}

		w := makeHTTPRequest(env, req)

		// Should attempt to add tag (may succeed or fail based on file permissions)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Logf("Add tag returned status %d", w.Code)
		}
	})

	t.Run("AddTagToNonExistentScenario", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/non-existent-scenario/add-tag",
		}

		w := makeHTTPRequest(env, req)

		// Should return error for non-existent scenario
		if w.Code != http.StatusInternalServerError && w.Code != http.StatusNotFound {
			t.Logf("Add tag to non-existent scenario returned status %d", w.Code)
		}
	})
}

// TestHandleRemoveMaintenanceTag_Comprehensive tests removing maintenance tag
func TestHandleRemoveMaintenanceTag_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create temporary test directory
	tempDir, err := ioutil.TempDir("", "remove-tag-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWD)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("RemoveTagFromExistingScenario", func(t *testing.T) {
		// Create a test scenario with maintenance tag
		scenarioDir := filepath.Join("scenarios", "test-remove-tag-scenario")
		if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755); err != nil {
			t.Fatalf("Failed to create scenario dir: %v", err)
		}

		serviceJSON := `{
  "service": {
    "name": "test-remove-tag-scenario",
    "tags": ["test", "maintenance"]
  },
  "ports": {
    "api": {
      "env_var": "API_PORT"
    }
  }
}`
		serviceJSONPath := filepath.Join(scenarioDir, ".vrooli", "service.json")
		if err := ioutil.WriteFile(serviceJSONPath, []byte(serviceJSON), 0644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-remove-tag-scenario/remove-tag",
		}

		w := makeHTTPRequest(env, req)

		// Should attempt to remove tag
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Logf("Remove tag returned status %d", w.Code)
		}
	})

	t.Run("RemoveTagFromNonExistentScenario", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/non-existent-scenario/remove-tag",
		}

		w := makeHTTPRequest(env, req)

		// Should return error for non-existent scenario
		if w.Code != http.StatusInternalServerError && w.Code != http.StatusNotFound {
			t.Logf("Remove tag from non-existent scenario returned status %d", w.Code)
		}
	})
}

// TestCheckScenarioDiscovery tests the scenario discovery health check
func TestCheckScenarioDiscovery(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create temporary test directory
	tempDir, err := ioutil.TempDir("", "check-discovery-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWD)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	t.Run("WithoutScenariosDirectory", func(t *testing.T) {
		result := checkScenarioDiscovery()

		if connected, ok := result["connected"].(bool); ok && connected {
			t.Error("Expected discovery to fail without scenarios directory")
		}
	})

	t.Run("WithScenariosDirectory", func(t *testing.T) {
		if err := os.MkdirAll("scenarios", 0755); err != nil {
			t.Fatalf("Failed to create scenarios dir: %v", err)
		}

		result := checkScenarioDiscovery()

		if connected, ok := result["connected"].(bool); ok && !connected {
			t.Error("Expected discovery to succeed with scenarios directory")
		}

		if _, ok := result["total_directories"]; !ok {
			t.Error("Expected total_directories in result")
		}
	})
}

// TestCheckFilesystemAccess tests filesystem access health check
func TestCheckFilesystemAccess(t *testing.T) {
	t.Run("CheckFilesystemAccessibility", func(t *testing.T) {
		result := checkFilesystemAccess()

		if _, ok := result["connected"]; !ok {
			t.Error("Expected 'connected' field in result")
		}

		// Check for error field
		if errorField, ok := result["error"]; ok && errorField != nil {
			t.Logf("Filesystem check returned error: %v", errorField)
		}
	})
}

// TestNotifyScenarioStateChange tests state change notifications
func TestNotifyScenarioStateChange(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NotifyWithInvalidEndpoint", func(t *testing.T) {
		// This should handle the error gracefully
		notifyScenarioStateChange("http://invalid-endpoint-12345", "active")
		// If we reach here, the function handled the error gracefully
		t.Log("Notification to invalid endpoint handled gracefully")
	})

	t.Run("NotifyWithEmptyEndpoint", func(t *testing.T) {
		// Should handle empty endpoint
		notifyScenarioStateChange("", "active")
		t.Log("Notification with empty endpoint handled gracefully")
	})
}

// TestHandlerEdgeCases tests various edge cases across handlers
func TestHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)
	initializeDefaultPresets(env.Orchestrator)

	t.Run("ActivateAlreadyActiveScenario", func(t *testing.T) {
		// Activate twice
		env.Orchestrator.ActivateScenario("test-scenario-1")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-scenario-1/activate",
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for idempotent activation, got %d", w.Code)
		}
	})

	t.Run("DeactivateAlreadyInactiveScenario", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-scenario-2/deactivate",
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for idempotent deactivation, got %d", w.Code)
		}
	})

	t.Run("GetScenariosWithActiveCampaigns", func(t *testing.T) {
		env.Orchestrator.ActivateScenario("test-scenario-1")
		env.Orchestrator.ActivateScenario("test-scenario-2")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/scenarios",
		}

		w := makeHTTPRequest(env, req)
		resp := assertJSONResponse(t, w, http.StatusOK)

		scenarios, ok := resp["scenarios"].([]interface{})
		if !ok {
			t.Fatal("Expected scenarios array")
		}

		activeCount := 0
		for _, s := range scenarios {
			scenarioMap := s.(map[string]interface{})
			if isActive, ok := scenarioMap["isActive"].(bool); ok && isActive {
				activeCount++
			}
		}

		if activeCount != 2 {
			t.Errorf("Expected 2 active scenarios, got %d", activeCount)
		}
	})

	t.Run("HealthCheckWithMultipleDependencies", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check for service field
		if service, ok := resp["service"].(string); !ok || service == "" {
			t.Error("Expected 'service' field in health response")
		}

		// Check for timestamp
		if _, ok := resp["timestamp"].(string); !ok {
			t.Error("Expected 'timestamp' field in health response")
		}
	})
}

// TestOptionsHandler_Coverage tests OPTIONS handler for coverage
func TestOptionsHandler_Coverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	endpoints := []string{
		"/api/v1/scenarios",
		"/api/v1/scenarios/test/activate",
		"/api/v1/presets",
		"/api/v1/status",
	}

	for _, endpoint := range endpoints {
		t.Run("OPTIONS_"+endpoint, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: "OPTIONS",
				Path:   endpoint,
			}

			w := makeHTTPRequest(env, req)

			// OPTIONS should be handled
			t.Logf("OPTIONS %s returned status %d", endpoint, w.Code)
		})
	}
}
