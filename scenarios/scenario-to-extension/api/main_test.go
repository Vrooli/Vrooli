package main

import (
	"net/http"
	"os"
	"testing"
	"time"
)

// TestMain sets up test environment
func TestMain(m *testing.M) {
	// Set test environment variable to bypass lifecycle check
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")

	// Run tests
	code := m.Run()

	// Cleanup
	os.Exit(code)
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("Success", func(t *testing.T) {
		w, err := executeRequest(healthHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":   "healthy",
			"scenario": "scenario-to-extension",
		})

		if response == nil {
			t.Fatal("No response received")
		}

		// Validate resources
		resources, ok := response["resources"].(map[string]interface{})
		if !ok {
			t.Error("Expected resources field to be an object")
		} else {
			if _, exists := resources["browserless"]; !exists {
				t.Error("Expected browserless health check in resources")
			}
			if _, exists := resources["templates"]; !exists {
				t.Error("Expected templates health check in resources")
			}
		}

		// Validate stats
		stats, ok := response["stats"].(map[string]interface{})
		if !ok {
			t.Error("Expected stats field to be an object")
		} else {
			if _, exists := stats["total_builds"]; !exists {
				t.Error("Expected total_builds in stats")
			}
		}
	})

	t.Run("VerifyVersion", func(t *testing.T) {
		w, err := executeRequest(healthHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		version, ok := response["version"].(string)
		if !ok || version == "" {
			t.Error("Expected version field to be a non-empty string")
		}
	})
}

// TestGenerateExtensionHandler tests the extension generation endpoint
func TestGenerateExtensionHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("Success_FullGeneration", func(t *testing.T) {
		req := TestData.GenerateExtensionRequest("test-scenario", "Test App", "http://localhost:3000")

		w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated, nil)
		if response == nil {
			t.Fatal("No response received")
		}

		// Verify response contains required fields
		buildID, ok := response["build_id"].(string)
		if !ok || buildID == "" {
			t.Error("Expected build_id to be a non-empty string")
		}

		extensionPath, ok := response["extension_path"].(string)
		if !ok || extensionPath == "" {
			t.Error("Expected extension_path to be a non-empty string")
		}

		status, ok := response["status"].(string)
		if !ok || status != "building" {
			t.Errorf("Expected status to be 'building', got %v", status)
		}

		// Verify build was created
		if _, exists := builds[buildID]; !exists {
			t.Error("Expected build to be created in builds map")
		}
	})

	t.Run("Success_WithDefaults", func(t *testing.T) {
		req := ExtensionGenerateRequest{
			ScenarioName: "test-scenario",
			Config: ExtensionConfig{
				AppName:     "Test App",
				APIEndpoint: "http://localhost:3000",
			},
		}

		w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated, nil)
		if response == nil {
			t.Fatal("No response received")
		}

		buildID := response["build_id"].(string)
		build := builds[buildID]

		// Verify defaults were applied
		if build.Config.Version != "1.0.0" {
			t.Errorf("Expected default version 1.0.0, got %s", build.Config.Version)
		}
		if build.Config.AuthorName != "Vrooli Scenario Generator" {
			t.Errorf("Expected default author name, got %s", build.Config.AuthorName)
		}
		if build.Config.License != "MIT" {
			t.Errorf("Expected default license MIT, got %s", build.Config.License)
		}
		if build.TemplateType != "full" {
			t.Errorf("Expected default template type 'full', got %s", build.TemplateType)
		}
	})

	t.Run("Error_MissingScenarioName", func(t *testing.T) {
		req := ExtensionGenerateRequest{
			Config: ExtensionConfig{
				AppName:     "Test App",
				APIEndpoint: "http://localhost:3000",
			},
		}

		w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_MissingAppName", func(t *testing.T) {
		req := ExtensionGenerateRequest{
			ScenarioName: "test-scenario",
			Config: ExtensionConfig{
				APIEndpoint: "http://localhost:3000",
			},
		}

		w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_MissingAPIEndpoint", func(t *testing.T) {
		req := ExtensionGenerateRequest{
			ScenarioName: "test-scenario",
			Config: ExtensionConfig{
				AppName: "Test App",
			},
		}

		w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   `{"invalid": json}`,
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestGetExtensionStatusHandler tests the build status endpoint
func TestGetExtensionStatusHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("Success", func(t *testing.T) {
		// Create a test build
		buildID := "test_build_123"
		now := time.Now()
		builds[buildID] = &ExtensionBuild{
			BuildID:      buildID,
			ScenarioName: "test-scenario",
			TemplateType: "full",
			Status:       "building",
			CreatedAt:    now,
			BuildLog:     []string{"Starting build"},
			ErrorLog:     []string{},
		}

		w, err := executeRequest(getExtensionStatusHandler, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/extension/status/" + buildID,
			URLVars: map[string]string{"build_id": buildID},
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"build_id":      buildID,
			"scenario_name": "test-scenario",
			"status":        "building",
		})

		if response == nil {
			t.Fatal("No response received")
		}

		// Verify build log
		buildLog, ok := response["build_log"].([]interface{})
		if !ok || len(buildLog) == 0 {
			t.Error("Expected build_log to be a non-empty array")
		}
	})

	t.Run("Success_CompletedBuild", func(t *testing.T) {
		buildID := "test_build_completed"
		now := time.Now()
		completedAt := now.Add(5 * time.Second)
		builds[buildID] = &ExtensionBuild{
			BuildID:      buildID,
			ScenarioName: "test-scenario",
			TemplateType: "full",
			Status:       "ready",
			CreatedAt:    now,
			CompletedAt:  &completedAt,
			BuildLog:     []string{"Build completed"},
			ErrorLog:     []string{},
		}

		w, err := executeRequest(getExtensionStatusHandler, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/extension/status/" + buildID,
			URLVars: map[string]string{"build_id": buildID},
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "ready",
		})

		if response == nil {
			t.Fatal("No response received")
		}

		if _, exists := response["completed_at"]; !exists {
			t.Error("Expected completed_at field for completed build")
		}
	})

	t.Run("Error_BuildNotFound", func(t *testing.T) {
		w, err := executeRequest(getExtensionStatusHandler, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/extension/status/nonexistent",
			URLVars: map[string]string{"build_id": "nonexistent"},
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusNotFound)
	})
}

// TestTestExtensionHandler tests the extension testing endpoint
func TestTestExtensionHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("Success_DefaultSites", func(t *testing.T) {
		req := TestData.GenerateTestRequest("/path/to/extension", nil)

		w, err := executeRequest(testExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/test",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("No response received")
		}

		// Verify test results structure
		success, ok := response["success"].(bool)
		if !ok {
			t.Error("Expected success field to be a boolean")
		}

		testResults, ok := response["test_results"].([]interface{})
		if !ok {
			t.Error("Expected test_results to be an array")
		}
		if len(testResults) == 0 {
			t.Error("Expected at least one test result")
		}

		summary, ok := response["summary"].(map[string]interface{})
		if !ok {
			t.Error("Expected summary field to be an object")
		} else {
			if _, exists := summary["total_tests"]; !exists {
				t.Error("Expected total_tests in summary")
			}
			if _, exists := summary["success_rate"]; !exists {
				t.Error("Expected success_rate in summary")
			}
		}

		// For default implementation, success should be true
		if !success {
			t.Error("Expected success to be true for default test")
		}
	})

	t.Run("Success_CustomSites", func(t *testing.T) {
		sites := []string{"https://example.com", "https://test.com", "https://demo.com"}
		req := TestData.GenerateTestRequest("/path/to/extension", sites)

		w, err := executeRequest(testExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/test",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("No response received")
		}

		testResults, ok := response["test_results"].([]interface{})
		if !ok {
			t.Error("Expected test_results to be an array")
		}
		if len(testResults) != len(sites) {
			t.Errorf("Expected %d test results, got %d", len(sites), len(testResults))
		}
	})

	t.Run("Error_MissingExtensionPath", func(t *testing.T) {
		req := ExtensionTestRequest{
			TestSites: []string{"https://example.com"},
		}

		w, err := executeRequest(testExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/test",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		w, err := executeRequest(testExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/test",
			Body:   `{"invalid": json}`,
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestListTemplatesHandler tests the templates listing endpoint
func TestListTemplatesHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("Success", func(t *testing.T) {
		w, err := executeRequest(listTemplatesHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/extension/templates",
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("No response received")
		}

		templates, ok := response["templates"].([]interface{})
		if !ok {
			t.Fatal("Expected templates to be an array")
		}

		// Should have at least the 4 default templates
		if len(templates) < 4 {
			t.Errorf("Expected at least 4 templates, got %d", len(templates))
		}

		// Verify template structure
		for _, tmpl := range templates {
			template := tmpl.(map[string]interface{})
			if _, exists := template["name"]; !exists {
				t.Error("Expected name field in template")
			}
			if _, exists := template["display_name"]; !exists {
				t.Error("Expected display_name field in template")
			}
			if _, exists := template["description"]; !exists {
				t.Error("Expected description field in template")
			}
		}

		// Verify count matches
		count, ok := response["count"].(float64)
		if !ok {
			t.Error("Expected count field to be a number")
		}
		if int(count) != len(templates) {
			t.Errorf("Expected count %d to match templates length %d", int(count), len(templates))
		}
	})
}

// TestListBuildsHandler tests the builds listing endpoint
func TestListBuildsHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("Success_EmptyBuilds", func(t *testing.T) {
		w, err := executeRequest(listBuildsHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/extension/builds",
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("No response received")
		}

		buildsArray, ok := response["builds"].([]interface{})
		if !ok {
			t.Fatal("Expected builds to be an array")
		}

		if len(buildsArray) != 0 {
			t.Errorf("Expected 0 builds, got %d", len(buildsArray))
		}

		count, ok := response["count"].(float64)
		if !ok || int(count) != 0 {
			t.Errorf("Expected count to be 0, got %v", count)
		}
	})

	t.Run("Success_WithBuilds", func(t *testing.T) {
		// Create test builds
		buildID1 := "test_build_1"
		buildID2 := "test_build_2"

		builds[buildID1] = &ExtensionBuild{
			BuildID:      buildID1,
			ScenarioName: "test-scenario-1",
			Status:       "ready",
			CreatedAt:    time.Now(),
		}

		builds[buildID2] = &ExtensionBuild{
			BuildID:      buildID2,
			ScenarioName: "test-scenario-2",
			Status:       "building",
			CreatedAt:    time.Now(),
		}

		w, err := executeRequest(listBuildsHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/extension/builds",
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("No response received")
		}

		buildsArray, ok := response["builds"].([]interface{})
		if !ok {
			t.Fatal("Expected builds to be an array")
		}

		if len(buildsArray) != 2 {
			t.Errorf("Expected 2 builds, got %d", len(buildsArray))
		}

		count, ok := response["count"].(float64)
		if !ok || int(count) != 2 {
			t.Errorf("Expected count to be 2, got %v", count)
		}
	})
}

// TestHelperFunctions tests the internal helper functions
func TestHelperFunctions(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("GenerateBuildID", func(t *testing.T) {
		id1 := generateBuildID()
		id2 := generateBuildID()

		if id1 == "" {
			t.Error("Expected non-empty build ID")
		}

		if id1 == id2 {
			t.Error("Expected unique build IDs")
		}

		// Verify format
		if len(id1) < 10 {
			t.Error("Expected build ID to have reasonable length")
		}
	})

	t.Run("CountBuildsByStatus", func(t *testing.T) {
		// Clear builds
		builds = make(map[string]*ExtensionBuild)

		// Add test builds
		builds["build1"] = &ExtensionBuild{Status: "building"}
		builds["build2"] = &ExtensionBuild{Status: "building"}
		builds["build3"] = &ExtensionBuild{Status: "ready"}
		builds["build4"] = &ExtensionBuild{Status: "failed"}

		buildingCount := countBuildsByStatus("building")
		if buildingCount != 2 {
			t.Errorf("Expected 2 building builds, got %d", buildingCount)
		}

		readyCount := countBuildsByStatus("ready")
		if readyCount != 1 {
			t.Errorf("Expected 1 ready build, got %d", readyCount)
		}

		failedCount := countBuildsByStatus("failed")
		if failedCount != 1 {
			t.Errorf("Expected 1 failed build, got %d", failedCount)
		}

		nonexistentCount := countBuildsByStatus("nonexistent")
		if nonexistentCount != 0 {
			t.Errorf("Expected 0 nonexistent builds, got %d", nonexistentCount)
		}
	})

	t.Run("CheckTemplatesHealth", func(t *testing.T) {
		healthy := checkTemplatesHealth()
		if !healthy {
			t.Error("Expected templates to be healthy (manifest.json exists)")
		}
	})

	t.Run("GetInstallInstructions", func(t *testing.T) {
		instructions := getInstallInstructions("full")
		if instructions == "" {
			t.Error("Expected non-empty install instructions")
		}

		// Verify it contains essential information
		if !contains(instructions, "chrome://extensions") {
			t.Error("Expected instructions to mention chrome://extensions")
		}
		if !contains(instructions, "Load unpacked") {
			t.Error("Expected instructions to mention 'Load unpacked'")
		}
	})
}

// TestConfigLoading tests configuration loading
func TestConfigLoading(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	t.Run("DefaultConfig", func(t *testing.T) {
		// Clear environment variables
		os.Unsetenv("PORT")
		os.Unsetenv("API_ENDPOINT")

		cfg := loadConfig()

		if cfg.Port != 3201 {
			t.Errorf("Expected default port 3201, got %d", cfg.Port)
		}

		if cfg.APIEndpoint != "http://localhost:3201" {
			t.Errorf("Expected default API endpoint, got %s", cfg.APIEndpoint)
		}

		if cfg.TemplatesPath != "./templates" {
			t.Errorf("Expected default templates path, got %s", cfg.TemplatesPath)
		}
	})

	t.Run("EnvironmentOverrides", func(t *testing.T) {
		os.Setenv("PORT", "8080")
		os.Setenv("API_ENDPOINT", "http://custom:8080")
		os.Setenv("TEMPLATES_PATH", "/custom/templates")

		cfg := loadConfig()

		if cfg.Port != 8080 {
			t.Errorf("Expected port 8080, got %d", cfg.Port)
		}

		if cfg.APIEndpoint != "http://custom:8080" {
			t.Errorf("Expected custom API endpoint, got %s", cfg.APIEndpoint)
		}

		if cfg.TemplatesPath != "/custom/templates" {
			t.Errorf("Expected custom templates path, got %s", cfg.TemplatesPath)
		}

		// Cleanup
		os.Unsetenv("PORT")
		os.Unsetenv("API_ENDPOINT")
		os.Unsetenv("TEMPLATES_PATH")
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
