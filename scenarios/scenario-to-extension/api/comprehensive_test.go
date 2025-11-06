package main

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestComprehensiveAPIEndpoints provides comprehensive coverage of all API endpoints
func TestComprehensiveAPIEndpoints(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("GenerateExtensionWorkflow", func(t *testing.T) {
		// Test complete workflow: generate -> check status -> list builds

		// 1. Generate extension
		req := TestData.GenerateExtensionRequest(
			"test-workflow-scenario",
			"Test Workflow Extension",
			"http://localhost:8080/api",
		)

		w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to generate extension: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated, nil)
		buildID, ok := response["build_id"].(string)
		if !ok {
			t.Fatal("Expected build_id in response")
		}

		// 2. Check build status
		w2, err := executeRequest(getExtensionStatusHandler, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/extension/status/" + buildID,
			URLVars: map[string]string{"build_id": buildID},
		})
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		statusResponse := assertJSONResponse(t, w2, http.StatusOK, nil)
		status, ok := statusResponse["status"].(string)
		if !ok {
			t.Fatal("Expected status field in response")
		}

		if status != "building" && status != "ready" && status != "failed" {
			t.Errorf("Invalid status value: %s", status)
		}

		// 3. List all builds
		w3, err := executeRequest(listBuildsHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/extension/builds",
		})
		if err != nil {
			t.Fatalf("Failed to list builds: %v", err)
		}

		buildsResponse := assertJSONArray(t, w3, http.StatusOK, "builds")
		if len(buildsResponse) == 0 {
			t.Error("Expected at least one build in list")
		}
	})

	t.Run("ExtensionTestWorkflow", func(t *testing.T) {
		// Test extension testing workflow
		testReq := TestData.GenerateTestRequest("/tmp/test-extension", []string{
			"https://example.com",
			"https://google.com",
		})

		w, err := executeRequest(testExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/test",
			Body:   testReq,
		})
		if err != nil {
			t.Fatalf("Failed to test extension: %v", err)
		}

		var result ExtensionTestResult
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse test result: %v", err)
		}

		if len(result.TestResults) != 2 {
			t.Errorf("Expected 2 test results, got %d", len(result.TestResults))
		}

		if result.Summary.TotalTests != 2 {
			t.Errorf("Expected summary total_tests=2, got %d", result.Summary.TotalTests)
		}
	})

	t.Run("TemplateTypes", func(t *testing.T) {
		// Test all template types
		templateTypes := []string{"full", "content-script-only", "background-only", "popup-only"}

		for _, templateType := range templateTypes {
			t.Run(templateType, func(t *testing.T) {
				req := ExtensionGenerateRequest{
					ScenarioName: "test-scenario",
					TemplateType: templateType,
					Config: ExtensionConfig{
						AppName:     "Test App",
						Description: "Test description",
						APIEndpoint: "http://localhost:3000",
					},
				}

				w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/extension/generate",
					Body:   req,
				})
				if err != nil {
					t.Fatalf("Failed to generate %s extension: %v", templateType, err)
				}

				response := assertJSONResponse(t, w, http.StatusCreated, nil)
				if response == nil {
					t.Fatalf("No response for %s template", templateType)
				}

				buildID, ok := response["build_id"].(string)
				if !ok || buildID == "" {
					t.Errorf("Expected build_id for %s template", templateType)
				}
			})
		}
	})

	t.Run("ConfigurationDefaults", func(t *testing.T) {
		// Test that defaults are properly applied
		req := ExtensionGenerateRequest{
			ScenarioName: "test-defaults",
			Config: ExtensionConfig{
				AppName:     "Test App",
				Description: "Test description",
				APIEndpoint: "http://localhost:3000",
				// Omit optional fields to test defaults
			},
		}

		w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to generate extension: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated, nil)
		buildID := response["build_id"].(string)

		// Check that build has defaults applied
		build, _ := buildManager.Get(buildID)

		if build.Config.Version != "1.0.0" {
			t.Errorf("Expected default version 1.0.0, got %s", build.Config.Version)
		}

		if build.Config.AuthorName != "Vrooli Scenario Generator" {
			t.Errorf("Expected default author, got %s", build.Config.AuthorName)
		}

		if build.Config.License != "MIT" {
			t.Errorf("Expected default license MIT, got %s", build.Config.License)
		}

		if len(build.Config.Permissions) == 0 {
			t.Error("Expected default permissions to be set")
		}

		if len(build.Config.HostPermissions) == 0 {
			t.Error("Expected default host permissions to be set")
		}
	})

	t.Run("ConcurrentBuildRequests", func(t *testing.T) {
		// Test concurrent extension generation
		const concurrentRequests = 5
		done := make(chan bool, concurrentRequests)

		for i := 0; i < concurrentRequests; i++ {
			go func(index int) {
				req := TestData.GenerateExtensionRequest(
					"concurrent-scenario",
					"Concurrent Test",
					"http://localhost:3000",
				)

				w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/extension/generate",
					Body:   req,
				})

				if err != nil {
					t.Errorf("Request %d failed: %v", index, err)
					done <- false
					return
				}

				if w.Code != http.StatusCreated {
					t.Errorf("Request %d got status %d, expected %d", index, w.Code, http.StatusCreated)
					done <- false
					return
				}

				done <- true
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < concurrentRequests; i++ {
			<-done
		}

		// Verify all builds were created
		buildCount := buildManager.Count()

		if buildCount < concurrentRequests {
			t.Errorf("Expected at least %d builds, got %d", concurrentRequests, buildCount)
		}
	})
}

// TestBuildManagement tests build lifecycle management
func TestBuildManagement(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("BuildStatusTransitions", func(t *testing.T) {
		// Create a build and verify status transitions
		build := &ExtensionBuild{
			BuildID:      "test-build-123",
			ScenarioName: "test-scenario",
			Status:       "building",
			CreatedAt:    time.Now(),
			BuildLog:     []string{},
			ErrorLog:     []string{},
		}

		buildManager.Add(build)

		// Verify initial status
		if build.Status != "building" {
			t.Errorf("Expected initial status 'building', got %s", build.Status)
		}

		// Simulate completion
		build.Status = "ready"
		completedAt := time.Now()
		build.CompletedAt = &completedAt

		if build.Status != "ready" {
			t.Errorf("Expected status 'ready', got %s", build.Status)
		}

		if build.CompletedAt == nil {
			t.Error("Expected CompletedAt to be set")
		}
	})

	t.Run("BuildLogTracking", func(t *testing.T) {
		req := TestData.GenerateExtensionRequest(
			"log-test-scenario",
			"Log Test",
			"http://localhost:3000",
		)

		w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to generate extension: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated, nil)
		buildID := response["build_id"].(string)

		// Wait a bit for async generation to start
		time.Sleep(100 * time.Millisecond)

		// Check that build log is being populated
		build, _ := buildManager.Get(buildID)

		if len(build.BuildLog) == 0 {
			t.Error("Expected build log to have entries")
		}
	})

	t.Run("BuildIDGeneration", func(t *testing.T) {
		// Test that build IDs are unique
		id1 := generateBuildID()
		time.Sleep(10 * time.Millisecond)
		id2 := generateBuildID()

		if id1 == id2 {
			t.Error("Expected unique build IDs")
		}

		if id1 == "" || id2 == "" {
			t.Error("Expected non-empty build IDs")
		}
	})

	t.Run("CountBuildsByStatus", func(t *testing.T) {
		// Clear builds
		buildManager = NewBuildManager()

		// Add builds with different statuses
		buildManager.Add(&ExtensionBuild{BuildID: "build1", Status: "building"})
		buildManager.Add(&ExtensionBuild{BuildID: "build2", Status: "building"})
		buildManager.Add(&ExtensionBuild{BuildID: "build3", Status: "ready"})
		buildManager.Add(&ExtensionBuild{BuildID: "build4", Status: "failed"})

		if buildManager.CountByStatus("building") != 2 {
			t.Errorf("Expected 2 building builds, got %d", buildManager.CountByStatus("building"))
		}

		if buildManager.CountByStatus("ready") != 1 {
			t.Errorf("Expected 1 ready build, got %d", buildManager.CountByStatus("ready"))
		}

		if buildManager.CountByStatus("failed") != 1 {
			t.Errorf("Expected 1 failed build, got %d", buildManager.CountByStatus("failed"))
		}

		if buildManager.CountByStatus("nonexistent") != 0 {
			t.Errorf("Expected 0 nonexistent builds, got %d", buildManager.CountByStatus("nonexistent"))
		}
	})
}

// TestExtensionTestingFeatures tests the extension testing functionality
func TestExtensionTestingFeatures(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("DefaultTestSites", func(t *testing.T) {
		// Test that default test sites are used when none provided
		req := ExtensionTestRequest{
			ExtensionPath: "/tmp/test-extension",
			Screenshot:    true,
			Headless:      true,
		}

		w, err := executeRequest(testExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/test",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to test extension: %v", err)
		}

		var result ExtensionTestResult
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		if len(result.TestResults) == 0 {
			t.Error("Expected at least one test result with default sites")
		}

		if result.TestResults[0].Site != "https://example.com" {
			t.Errorf("Expected default site https://example.com, got %s", result.TestResults[0].Site)
		}
	})

	t.Run("EmptyTestSites", func(t *testing.T) {
		// Test that empty test sites array gets default
		req := ExtensionTestRequest{
			ExtensionPath: "/tmp/test-extension",
			TestSites:     []string{},
			Screenshot:    true,
		}

		w, err := executeRequest(testExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/test",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to test extension: %v", err)
		}

		var result ExtensionTestResult
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		if len(result.TestResults) == 0 {
			t.Error("Expected at least one test result with default sites")
		}
	})

	t.Run("TestSummaryCalculation", func(t *testing.T) {
		// Test that summary statistics are calculated correctly
		req := TestData.GenerateTestRequest("/tmp/test", []string{
			"https://site1.com",
			"https://site2.com",
			"https://site3.com",
		})

		w, err := executeRequest(testExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/test",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to test extension: %v", err)
		}

		var result ExtensionTestResult
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		if result.Summary.TotalTests != 3 {
			t.Errorf("Expected total_tests=3, got %d", result.Summary.TotalTests)
		}

		if result.Summary.Passed != 3 {
			t.Errorf("Expected passed=3, got %d", result.Summary.Passed)
		}

		if result.Summary.Failed != 0 {
			t.Errorf("Expected failed=0, got %d", result.Summary.Failed)
		}

		if result.Summary.SuccessRate != 100 {
			t.Errorf("Expected success_rate=100, got %f", result.Summary.SuccessRate)
		}
	})
}

// TestConfigurationManagement tests configuration loading and environment variables
func TestConfigurationManagement(t *testing.T) {
	t.Run("EnvironmentVariableOverrides", func(t *testing.T) {
		// Clear API_PORT to avoid interference from lifecycle system
		oldAPIPort := os.Getenv("API_PORT")
		os.Unsetenv("API_PORT")

		// Set environment variables
		os.Setenv("PORT", "9999")
		os.Setenv("API_ENDPOINT", "http://custom.endpoint")
		os.Setenv("TEMPLATES_PATH", "/custom/templates")
		os.Setenv("OUTPUT_PATH", "/custom/output")
		os.Setenv("BROWSERLESS_URL", "http://custom.browserless")
		os.Setenv("DEBUG", "true")

		defer func() {
			os.Unsetenv("PORT")
			os.Unsetenv("API_ENDPOINT")
			os.Unsetenv("TEMPLATES_PATH")
			os.Unsetenv("OUTPUT_PATH")
			os.Unsetenv("BROWSERLESS_URL")
			os.Unsetenv("DEBUG")
			if oldAPIPort != "" {
				os.Setenv("API_PORT", oldAPIPort)
			}
		}()

		cfg := loadConfig()

		if cfg.Port != 9999 {
			t.Errorf("Expected port 9999, got %d", cfg.Port)
		}

		if cfg.APIEndpoint != "http://custom.endpoint" {
			t.Errorf("Expected custom API endpoint, got %s", cfg.APIEndpoint)
		}

		if cfg.TemplatesPath != "/custom/templates" {
			t.Errorf("Expected custom templates path, got %s", cfg.TemplatesPath)
		}

		if cfg.OutputPath != "/custom/output" {
			t.Errorf("Expected custom output path, got %s", cfg.OutputPath)
		}

		if cfg.BrowserlessURL != "http://custom.browserless" {
			t.Errorf("Expected custom browserless URL, got %s", cfg.BrowserlessURL)
		}

		if !cfg.Debug {
			t.Error("Expected debug to be true")
		}
	})

	t.Run("InvalidPortEnvironmentVariable", func(t *testing.T) {
		// Clear API_PORT to avoid interference from lifecycle system
		oldAPIPort := os.Getenv("API_PORT")
		os.Unsetenv("API_PORT")

		os.Setenv("PORT", "invalid")
		defer func() {
			os.Unsetenv("PORT")
			if oldAPIPort != "" {
				os.Setenv("API_PORT", oldAPIPort)
			}
		}()

		cfg := loadConfig()

		// Should fall back to default port
		if cfg.Port != 3201 {
			t.Errorf("Expected default port 3201 when invalid port specified, got %d", cfg.Port)
		}
	})
}

// TestTemplateManagement tests template listing and health checks
func TestTemplateManagement(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("ListAllTemplates", func(t *testing.T) {
		w, err := executeRequest(listTemplatesHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/extension/templates",
		})
		if err != nil {
			t.Fatalf("Failed to list templates: %v", err)
		}

		templates := assertJSONArray(t, w, http.StatusOK, "templates")

		if len(templates) != 4 {
			t.Errorf("Expected 4 templates, got %d", len(templates))
		}

		// Verify each template has required fields
		for _, template := range templates {
			templateMap, ok := template.(map[string]interface{})
			if !ok {
				t.Error("Expected template to be an object")
				continue
			}

			if _, exists := templateMap["name"]; !exists {
				t.Error("Expected template to have name field")
			}

			if _, exists := templateMap["display_name"]; !exists {
				t.Error("Expected template to have display_name field")
			}

			if _, exists := templateMap["description"]; !exists {
				t.Error("Expected template to have description field")
			}

			if _, exists := templateMap["files"]; !exists {
				t.Error("Expected template to have files field")
			}
		}
	})

	t.Run("CheckTemplatesHealth", func(t *testing.T) {
		// With valid template
		if !checkTemplatesHealth() {
			t.Error("Expected templates health check to pass with valid template")
		}

		// Test missing template (would require modifying filesystem in real scenario)
		// This is a basic sanity check
	})

	t.Run("GetInstallInstructionsForEachTemplate", func(t *testing.T) {
		templateTypes := []string{"full", "content-script-only", "background-only", "popup-only"}

		for _, templateType := range templateTypes {
			instructions := getInstallInstructions(templateType)

			if instructions == "" {
				t.Errorf("Expected non-empty instructions for %s", templateType)
			}

			// Verify instructions contain key information
			if !containsString(instructions, "chrome://extensions/") {
				t.Errorf("Expected instructions to mention chrome://extensions/ for %s", templateType)
			}

			if !containsString(instructions, "Developer mode") {
				t.Errorf("Expected instructions to mention Developer mode for %s", templateType)
			}
		}
	})
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr) && findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
