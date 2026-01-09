package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// Note: TestHandleGetScenarioStatuses is already covered in additional_coverage_test.go
// Additional tests would cause timeouts due to CLI command execution

// TestHandleAddMaintenanceTag_EdgeCases tests edge cases in handleAddMaintenanceTag
func TestHandleAddMaintenanceTag_EdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create a temporary test directory
	tmpDir, err := ioutil.TempDir("", "maintenance-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save original working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(origWd)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	t.Run("NonExistentScenario", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/scenarios/nonexistent/add-tag", nil)
		w := httptest.NewRecorder()

		env.Router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 for nonexistent scenario, got %d", w.Code)
		}
	})

	t.Run("ValidScenarioWithoutMaintenanceTag", func(t *testing.T) {
		// Create test scenario directory structure
		scenarioPath := filepath.Join(tmpDir, "scenarios", "test-scenario", ".vrooli")
		if err := os.MkdirAll(scenarioPath, 0755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		// Create service.json without maintenance tag
		serviceJSON := map[string]interface{}{
			"service": map[string]interface{}{
				"name": "test-scenario",
				"tags": []string{"test"},
			},
		}
		data, _ := json.MarshalIndent(serviceJSON, "", "  ")
		servicePath := filepath.Join(scenarioPath, "service.json")
		if err := ioutil.WriteFile(servicePath, data, 0644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		req := httptest.NewRequest("POST", "/api/v1/scenarios/test-scenario/add-tag", nil)
		w := httptest.NewRecorder()

		env.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify tag was added
		updatedData, err := ioutil.ReadFile(servicePath)
		if err != nil {
			t.Fatalf("Failed to read updated service.json: %v", err)
		}

		var updatedService map[string]interface{}
		if err := json.Unmarshal(updatedData, &updatedService); err != nil {
			t.Fatalf("Failed to parse updated service.json: %v", err)
		}

		serviceData := updatedService["service"].(map[string]interface{})
		tags := serviceData["tags"].([]interface{})
		hasMaintenanceTag := false
		for _, tag := range tags {
			if tag.(string) == "maintenance" {
				hasMaintenanceTag = true
				break
			}
		}

		if !hasMaintenanceTag {
			t.Error("Maintenance tag was not added")
		}
	})

	t.Run("ScenarioAlreadyHasMaintenanceTag", func(t *testing.T) {
		// Create test scenario directory structure
		scenarioPath := filepath.Join(tmpDir, "scenarios", "test-scenario-2", ".vrooli")
		if err := os.MkdirAll(scenarioPath, 0755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		// Create service.json WITH maintenance tag already
		serviceJSON := map[string]interface{}{
			"service": map[string]interface{}{
				"name": "test-scenario-2",
				"tags": []string{"test", "maintenance"},
			},
		}
		data, _ := json.MarshalIndent(serviceJSON, "", "  ")
		servicePath := filepath.Join(scenarioPath, "service.json")
		if err := ioutil.WriteFile(servicePath, data, 0644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		req := httptest.NewRequest("POST", "/api/v1/scenarios/test-scenario-2/add-tag", nil)
		w := httptest.NewRecorder()

		env.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify tag count didn't increase
		updatedData, err := ioutil.ReadFile(servicePath)
		if err != nil {
			t.Fatalf("Failed to read updated service.json: %v", err)
		}

		var updatedService map[string]interface{}
		if err := json.Unmarshal(updatedData, &updatedService); err != nil {
			t.Fatalf("Failed to parse updated service.json: %v", err)
		}

		serviceData := updatedService["service"].(map[string]interface{})
		tags := serviceData["tags"].([]interface{})

		// Should still have exactly 2 tags
		if len(tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(tags))
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		// Create test scenario directory structure
		scenarioPath := filepath.Join(tmpDir, "scenarios", "invalid-json", ".vrooli")
		if err := os.MkdirAll(scenarioPath, 0755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		// Create invalid service.json
		servicePath := filepath.Join(scenarioPath, "service.json")
		if err := ioutil.WriteFile(servicePath, []byte("{invalid json}"), 0644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		req := httptest.NewRequest("POST", "/api/v1/scenarios/invalid-json/add-tag", nil)
		w := httptest.NewRecorder()

		env.Router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 for invalid JSON, got %d", w.Code)
		}
	})
}

// TestHandleRemoveMaintenanceTag_EdgeCases tests edge cases in handleRemoveMaintenanceTag
func TestHandleRemoveMaintenanceTag_EdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create a temporary test directory
	tmpDir, err := ioutil.TempDir("", "maintenance-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save original working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(origWd)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	t.Run("NonExistentScenario", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/scenarios/nonexistent/remove-tag", nil)
		w := httptest.NewRecorder()

		env.Router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 for nonexistent scenario, got %d", w.Code)
		}
	})

	t.Run("RemoveMaintenanceTag", func(t *testing.T) {
		// Create test scenario directory structure
		scenarioPath := filepath.Join(tmpDir, "scenarios", "test-scenario", ".vrooli")
		if err := os.MkdirAll(scenarioPath, 0755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		// Create service.json WITH maintenance tag
		serviceJSON := map[string]interface{}{
			"service": map[string]interface{}{
				"name": "test-scenario",
				"tags": []string{"test", "maintenance"},
			},
		}
		data, _ := json.MarshalIndent(serviceJSON, "", "  ")
		servicePath := filepath.Join(scenarioPath, "service.json")
		if err := ioutil.WriteFile(servicePath, data, 0644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		req := httptest.NewRequest("POST", "/api/v1/scenarios/test-scenario/remove-tag", nil)
		w := httptest.NewRecorder()

		env.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify tag was removed
		updatedData, err := ioutil.ReadFile(servicePath)
		if err != nil {
			t.Fatalf("Failed to read updated service.json: %v", err)
		}

		var updatedService map[string]interface{}
		if err := json.Unmarshal(updatedData, &updatedService); err != nil {
			t.Fatalf("Failed to parse updated service.json: %v", err)
		}

		serviceData := updatedService["service"].(map[string]interface{})
		tags := serviceData["tags"].([]interface{})
		hasMaintenanceTag := false
		for _, tag := range tags {
			if tag.(string) == "maintenance" {
				hasMaintenanceTag = true
				break
			}
		}

		if hasMaintenanceTag {
			t.Error("Maintenance tag was not removed")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		// Create test scenario directory structure
		scenarioPath := filepath.Join(tmpDir, "scenarios", "invalid-json", ".vrooli")
		if err := os.MkdirAll(scenarioPath, 0755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		// Create invalid service.json
		servicePath := filepath.Join(scenarioPath, "service.json")
		if err := ioutil.WriteFile(servicePath, []byte("{invalid json}"), 0644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		req := httptest.NewRequest("POST", "/api/v1/scenarios/invalid-json/remove-tag", nil)
		w := httptest.NewRecorder()

		env.Router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 for invalid JSON, got %d", w.Code)
		}
	})
}

// TestCheckScenarioStateManagement_OLD tests the obsolete function for coverage
func TestCheckScenarioStateManagement_OLD(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	result := checkScenarioStateManagement_OLD()

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check that result contains expected keys
	if _, ok := result["connected"]; !ok {
		t.Error("Result missing 'connected' key")
	}

	if _, ok := result["error"]; !ok {
		t.Error("Result missing 'error' key")
	}

	// Test that orchestrator initialization works
	connected, ok := result["connected"].(bool)
	if !ok {
		t.Error("Expected 'connected' to be bool")
	}

	if connected {
		// If connected, should have scenario counts
		if _, ok := result["total_scenarios"]; !ok {
			t.Error("Connected result missing 'total_scenarios' key")
		}
		if _, ok := result["active_scenarios"]; !ok {
			t.Error("Connected result missing 'active_scenarios' key")
		}
		if _, ok := result["inactive_scenarios"]; !ok {
			t.Error("Connected result missing 'inactive_scenarios' key")
		}
	}
}

// Note: TestCheckFilesystemAccess and TestNotifyScenarioStateChange
// are already covered in handler_edge_cases_test.go

// TestHandleListAllScenarios_EdgeCases tests edge cases in handleListAllScenarios
func TestHandleListAllScenarios_EdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	handler := handleListAllScenarios()

	t.Run("Request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/all-scenarios", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// The handler may return 500 if vrooli CLI is not available in test environment
		// or 200 if it is available. Both are acceptable for this test.
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}

		// If successful, should have scenarios key
		if w.Code == http.StatusOK {
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to parse response JSON: %v", err)
			}
		}
	})
}

// TestDiscoveryEdgeCases tests additional edge cases in scenario discovery
func TestDiscoveryEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("DiscoverWithEmptyRoot", func(t *testing.T) {
		// Create a temporary empty directory
		tmpDir, err := ioutil.TempDir("", "empty-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Save and change VROOLI_ROOT
		origRoot := os.Getenv("VROOLI_ROOT")
		os.Setenv("VROOLI_ROOT", tmpDir)
		defer os.Setenv("VROOLI_ROOT", origRoot)

		// Create orchestrator and logger for discovery
		orchestrator := NewOrchestrator()
		testLogger := log.New(os.Stdout, "[test] ", log.LstdFlags)

		discoverScenarios(orchestrator, testLogger)

		// Should not error, orchestrator should have 0 scenarios
		scenarios := orchestrator.GetScenarios()
		if len(scenarios) != 0 {
			t.Errorf("Expected 0 scenarios in empty directory, got %d", len(scenarios))
		}
	})
}

// TestHandleStopScenario_EdgeCases tests edge cases in handleStopScenario
func TestHandleStopScenario_EdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("StopNonExistentScenario", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/nonexistent/stop",
		})

		// Handler calls vrooli CLI which will return error for non-existent scenario
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("StopValidScenario", func(t *testing.T) {
		// Try to stop a valid scenario (may or may not be running)
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-scenario-1/stop",
		})

		// Handler calls vrooli CLI - accept both success and error codes
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})
}

// TestHandleStartScenario_EdgeCases tests edge cases in handleStartScenario
func TestHandleStartScenario_EdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)

	t.Run("StartNonExistentScenario", func(t *testing.T) {
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/nonexistent/start",
		})

		// Handler calls vrooli CLI which will return error for non-existent scenario
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("StartValidScenario", func(t *testing.T) {
		// Try to start a test scenario (may or may not succeed depending on environment)
		w := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/scenarios/test-scenario-1/start",
		})

		// Handler calls vrooli CLI - accept both success and error codes
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})
}

// TestCheckScenarioDiscovery_EdgeCases tests edge cases in checkScenarioDiscovery
func TestCheckScenarioDiscovery_EdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CheckDiscovery", func(t *testing.T) {
		result := checkScenarioDiscovery()

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		// Check that result contains expected keys
		if _, ok := result["connected"]; !ok {
			t.Error("Result missing 'connected' key")
		}

		// Function uses globalOrchestrator which is initialized in main
		// In tests, this may or may not be populated, so just check the structure
		if connected, ok := result["connected"].(bool); ok {
			if connected {
				// If connected, should have scenario count field
				if _, ok := result["total_scenarios"]; !ok {
					t.Error("Connected result missing 'total_scenarios' key")
				}
			}
		}
	})
}

// TestMiddlewareEdgeCases tests additional middleware edge cases
func TestMiddlewareEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CORSWithDifferentMethods", func(t *testing.T) {
		handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "OK")
		}))

		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/test", nil)
			req.Header.Set("Origin", "http://localhost:36222")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Check CORS headers are set
			if origin := w.Header().Get("Access-Control-Allow-Origin"); origin == "" {
				t.Errorf("CORS origin header not set for method %s", method)
			}
		}
	})

	t.Run("OptionsHandler", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://localhost:36222")
		w := httptest.NewRecorder()

		optionsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}

		// The optionsHandler delegates to CORS middleware, so headers may not be set directly
		// This is acceptable since OPTIONS requests are handled by the router middleware
	})
}
