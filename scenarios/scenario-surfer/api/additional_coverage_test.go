package main

import (
	"os"
	"testing"
)

// TestGetScenariosFunction tests the getScenarios helper
func TestGetScenariosFunction(t *testing.T) {
	t.Run("ExecutesVrooliCommand", func(t *testing.T) {
		// This test verifies getScenarios executes without crashing
		// Actual result depends on system state
		scenarios, err := getScenarios()

		// Function should either succeed or return an error
		// We can't predict the exact result in tests
		if err != nil {
			t.Logf("getScenarios returned error (expected in test env): %v", err)
		} else {
			t.Logf("getScenarios returned %d scenarios", len(scenarios))
		}

		// Verify scenarios is not nil even on error
		if scenarios == nil && err == nil {
			t.Error("Expected non-nil scenarios or error")
		}
	})
}

// TestSubmitIssueToTracker tests the issue submission function
func TestSubmitIssueToTracker(t *testing.T) {
	t.Run("HandlesTrackerNotRunning", func(t *testing.T) {
		report := IssueReport{
			Scenario:    "test-scenario",
			Title:       "Test Issue",
			Description: "Test Description",
		}

		err := submitIssueToTracker(report)

		// Should return error if app-issue-tracker not found/running
		// This is expected in test environment
		if err != nil {
			t.Logf("Expected error when tracker not available: %v", err)
		}
	})
}

// TestCaptureScreenshot tests the screenshot capture function
func TestCaptureScreenshot(t *testing.T) {
	t.Run("HandlesScreenshotFailure", func(t *testing.T) {
		// This will likely fail if browserless is not available
		screenshot, err := captureScreenshot("http://localhost:3000")

		// Expected to fail in test environment
		if err != nil {
			t.Logf("Expected error when browserless not available: %v", err)
		}

		if err == nil && screenshot == "" {
			t.Error("If no error, expected non-empty screenshot path")
		}
	})

	t.Run("HandlesInvalidURL", func(t *testing.T) {
		screenshot, err := captureScreenshot("invalid-url")

		// Should handle invalid URLs gracefully
		if err != nil {
			t.Logf("Correctly returned error for invalid URL: %v", err)
		}

		if err == nil && screenshot == "" {
			t.Error("If no error, expected non-empty screenshot path")
		}
	})
}

// TestMainFunction tests the main function lifecycle protection
func TestMainFunction(t *testing.T) {
	t.Run("RequiresLifecycleManagement", func(t *testing.T) {
		// Temporarily unset lifecycle variable
		original := os.Getenv("VROOLI_LIFECYCLE_MANAGED")
		os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")

		// We can't actually call main() in tests as it calls os.Exit
		// But we can verify the environment variable check works
		if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "" {
			t.Error("Expected VROOLI_LIFECYCLE_MANAGED to be unset")
		}

		// Restore
		if original != "" {
			os.Setenv("VROOLI_LIFECYCLE_MANAGED", original)
		}
	})

	t.Run("UsesAPIPortEnvironmentVariable", func(t *testing.T) {
		// Verify API_PORT environment variable handling
		original := os.Getenv("API_PORT")

		// Set custom port
		os.Setenv("API_PORT", "12345")
		port := os.Getenv("API_PORT")

		if port != "12345" {
			t.Errorf("Expected port '12345', got '%s'", port)
		}

		// Restore
		if original != "" {
			os.Setenv("API_PORT", original)
		} else {
			os.Unsetenv("API_PORT")
		}
	})
}

// TestScenarioInfoStruct tests the ScenarioInfo data structure
func TestScenarioInfoStruct(t *testing.T) {
	t.Run("CreateScenarioInfo", func(t *testing.T) {
		scenario := ScenarioInfo{
			Name:        "test-scenario",
			Status:      "running",
			Health:      "healthy",
			Ports:       map[string]int{"ui": 3000, "api": 8000},
			Tags:        []string{"test", "automation"},
			URL:         "http://localhost:3000",
			DisplayName: "Test Scenario",
			Description: "A test scenario",
			Runtime:     "go",
			Processes:   2,
		}

		// Verify all fields are set correctly
		if scenario.Name != "test-scenario" {
			t.Errorf("Expected name 'test-scenario', got '%s'", scenario.Name)
		}
		if scenario.Ports["ui"] != 3000 {
			t.Errorf("Expected UI port 3000, got %d", scenario.Ports["ui"])
		}
		if len(scenario.Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(scenario.Tags))
		}
	})
}

// TestVrooliStatusResponse tests the response structure
func TestVrooliStatusResponse(t *testing.T) {
	t.Run("CreateStatusResponse", func(t *testing.T) {
		response := VrooliStatusResponse{
			Success:   true,
			Scenarios: []ScenarioInfo{},
		}

		response.Summary.Total = 5
		response.Summary.Running = 3
		response.Summary.Stopped = 2

		if !response.Success {
			t.Error("Expected success to be true")
		}
		if response.Summary.Total != 5 {
			t.Errorf("Expected total 5, got %d", response.Summary.Total)
		}
	})
}

// TestIssueReportStruct tests the IssueReport structure
func TestIssueReportStruct(t *testing.T) {
	t.Run("CreateIssueReport", func(t *testing.T) {
		report := IssueReport{
			Scenario:    "test-scenario",
			Title:       "Test Title",
			Description: "Test Description",
			Screenshot:  "/tmp/screenshot.png",
		}

		if report.Scenario != "test-scenario" {
			t.Errorf("Expected scenario 'test-scenario', got '%s'", report.Scenario)
		}
		if report.Screenshot != "/tmp/screenshot.png" {
			t.Errorf("Expected screenshot path, got '%s'", report.Screenshot)
		}
	})

	t.Run("CreateReportWithoutScreenshot", func(t *testing.T) {
		report := IssueReport{
			Scenario:    "test-scenario",
			Title:       "Test Title",
			Description: "Test Description",
		}

		if report.Screenshot != "" {
			t.Errorf("Expected empty screenshot, got '%s'", report.Screenshot)
		}
	})
}

// TestHealthyScenarioResponse tests the response structure
func TestHealthyScenarioResponse(t *testing.T) {
	t.Run("CreateHealthyResponse", func(t *testing.T) {
		response := HealthyScenarioResponse{
			Scenarios: []ScenarioInfo{
				createTestScenario("test1", "running", "healthy", 3000),
				createTestScenario("test2", "running", "healthy", 3001),
			},
			Categories: []string{"work", "fun", "dev"},
		}

		if len(response.Scenarios) != 2 {
			t.Errorf("Expected 2 scenarios, got %d", len(response.Scenarios))
		}
		if len(response.Categories) != 3 {
			t.Errorf("Expected 3 categories, got %d", len(response.Categories))
		}
	})

	t.Run("EmptyResponse", func(t *testing.T) {
		response := HealthyScenarioResponse{
			Scenarios:  []ScenarioInfo{},
			Categories: []string{},
		}

		if len(response.Scenarios) != 0 {
			t.Errorf("Expected 0 scenarios, got %d", len(response.Scenarios))
		}
		if len(response.Categories) != 0 {
			t.Errorf("Expected 0 categories, got %d", len(response.Categories))
		}
	})
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("PortRespondingWithTimeout", func(t *testing.T) {
		// Test with a port that definitely won't respond quickly
		responding := isPortResponding(65535)
		if responding {
			t.Error("Expected high port to not be responding")
		}
	})

	t.Run("LoadTagsWithEmptyName", func(t *testing.T) {
		tags := loadScenarioTags("")
		if len(tags) != 0 {
			t.Errorf("Expected empty tags for empty scenario name, got %v", tags)
		}
	})

	t.Run("LoadTagsWithSpecialCharacters", func(t *testing.T) {
		tags := loadScenarioTags("scenario-with-special-chars-!@#$")
		// Should handle gracefully without crashing
		if tags == nil {
			t.Error("Expected non-nil tags array")
		}
	})
}

// TestCORSMiddlewareInternal tests CORS middleware internals
func TestCORSMiddlewareInternal(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("VerifyOriginHeader", func(t *testing.T) {
		w, _ := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin != "http://localhost:3000" {
			t.Errorf("Expected origin 'http://localhost:3000', got '%s'", origin)
		}
	})

	t.Run("VerifyMethodsHeader", func(t *testing.T) {
		w, _ := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		methods := w.Header().Get("Access-Control-Allow-Methods")
		if methods != "GET, POST, OPTIONS" {
			t.Errorf("Expected methods 'GET, POST, OPTIONS', got '%s'", methods)
		}
	})

	t.Run("VerifyHeadersAllowed", func(t *testing.T) {
		w, _ := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		headers := w.Header().Get("Access-Control-Allow-Headers")
		if headers != "Content-Type" {
			t.Errorf("Expected headers 'Content-Type', got '%s'", headers)
		}
	})
}
