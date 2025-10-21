package main

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("EmptyTestSites", func(t *testing.T) {
		req := ExtensionTestRequest{
			ExtensionPath: "/test/extension",
			TestSites:     []string{},
		}

		w, err := executeRequest(testExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/test",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		// Should use default site
		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("No response received")
		}

		testResults, ok := response["test_results"].([]interface{})
		if !ok || len(testResults) == 0 {
			t.Error("Expected at least one test result with default site")
		}
	})

	t.Run("NilTestSites", func(t *testing.T) {
		req := ExtensionTestRequest{
			ExtensionPath: "/test/extension",
			TestSites:     nil,
		}

		w, err := executeRequest(testExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/test",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		// Should use default site
		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("No response received")
		}

		testResults, ok := response["test_results"].([]interface{})
		if !ok || len(testResults) == 0 {
			t.Error("Expected at least one test result with default site")
		}
	})

	t.Run("EmptyScenarioName", func(t *testing.T) {
		req := ExtensionGenerateRequest{
			ScenarioName: "",
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

	t.Run("EmptyAppName", func(t *testing.T) {
		req := ExtensionGenerateRequest{
			ScenarioName: "test-scenario",
			Config: ExtensionConfig{
				AppName:     "",
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

	t.Run("EmptyAPIEndpoint", func(t *testing.T) {
		req := ExtensionGenerateRequest{
			ScenarioName: "test-scenario",
			Config: ExtensionConfig{
				AppName:     "Test App",
				APIEndpoint: "",
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

	t.Run("LongScenarioName", func(t *testing.T) {
		longName := "this-is-a-very-long-scenario-name-that-might-cause-issues-with-path-length-limits-and-should-be-handled-gracefully"
		req := ExtensionGenerateRequest{
			ScenarioName: longName,
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
		if build.ScenarioName != longName {
			t.Errorf("Expected scenario name to be preserved, got %s", build.ScenarioName)
		}
	})

	t.Run("SpecialCharactersInConfig", func(t *testing.T) {
		req := ExtensionGenerateRequest{
			ScenarioName: "test-scenario",
			Config: ExtensionConfig{
				AppName:     "Test App with \"quotes\" and 'apostrophes'",
				Description: "Description with <html> tags & special chars",
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
	})

	t.Run("MultipleBuildsTracking", func(t *testing.T) {
		// Create multiple builds and verify they're all tracked
		for i := 0; i < 5; i++ {
			req := TestData.GenerateExtensionRequest("test-scenario", "Test App", "http://localhost:3000")
			w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/extension/generate",
				Body:   req,
			})
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if w.Code != http.StatusCreated {
				t.Errorf("Expected status 201, got %d", w.Code)
			}
		}

		// List builds and verify count
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

		count, ok := response["count"].(float64)
		if !ok || int(count) < 5 {
			t.Errorf("Expected at least 5 builds, got %v", count)
		}
	})
}

// TestConcurrentAccess tests concurrent access scenarios
func TestConcurrentAccess(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("ConcurrentBuildGeneration", func(t *testing.T) {
		concurrency := 10
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(index int) {
				req := TestData.GenerateExtensionRequest("test-scenario", "Test App", "http://localhost:3000")
				w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/extension/generate",
					Body:   req,
				})
				if err != nil {
					t.Errorf("Failed to execute request: %v", err)
				}
				if w.Code != http.StatusCreated {
					t.Errorf("Expected status 201, got %d", w.Code)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < concurrency; i++ {
			<-done
		}

		// Small delay to ensure all builds are registered
		time.Sleep(10 * time.Millisecond)

		// Verify all builds were created
		buildsMux.RLock()
		buildCount := len(builds)
		buildsMux.RUnlock()

		if buildCount < concurrency {
			t.Errorf("Expected at least %d builds, got %d", concurrency, buildCount)
		}
	})

	t.Run("ConcurrentStatusChecks", func(t *testing.T) {
		// Create a build
		buildID := "test_concurrent_status"
		builds[buildID] = &ExtensionBuild{
			BuildID:      buildID,
			ScenarioName: "test-scenario",
			Status:       "building",
			CreatedAt:    time.Now(),
			BuildLog:     []string{},
			ErrorLog:     []string{},
		}

		concurrency := 20
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				w, err := executeRequest(getExtensionStatusHandler, HTTPTestRequest{
					Method:  "GET",
					Path:    "/api/v1/extension/status/" + buildID,
					URLVars: map[string]string{"build_id": buildID},
				})
				if err != nil {
					t.Errorf("Failed to execute request: %v", err)
				}
				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", w.Code)
				}
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < concurrency; i++ {
			<-done
		}
	})
}

// TestPerformance tests performance characteristics
func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("GenerateExtensionPerformance", func(t *testing.T) {
		req := TestData.GenerateExtensionRequest("test-scenario", "Test App", "http://localhost:3000")

		start := time.Now()
		w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   req,
		})
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		// Should complete quickly (handler returns immediately, processing happens in background)
		if duration > 100*time.Millisecond {
			t.Logf("Warning: Handler took %v (expected < 100ms)", duration)
		}
	})

	t.Run("ListBuildsPerformance", func(t *testing.T) {
		// Create 100 builds
		for i := 0; i < 100; i++ {
			buildID := generateBuildID()
			builds[buildID] = &ExtensionBuild{
				BuildID:      buildID,
				ScenarioName: "test-scenario",
				Status:       "ready",
				CreatedAt:    time.Now(),
				BuildLog:     []string{},
				ErrorLog:     []string{},
			}
		}

		start := time.Now()
		w, err := executeRequest(listBuildsHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/extension/builds",
		})
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Should complete quickly even with 100 builds
		if duration > 50*time.Millisecond {
			t.Logf("Warning: List builds took %v (expected < 50ms)", duration)
		}
	})
}

// TestConfigEdgeCases tests configuration edge cases
func TestConfigEdgeCases(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

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

		// Should use default port when invalid port is provided
		if cfg.Port != 3201 {
			t.Errorf("Expected default port 3201, got %d", cfg.Port)
		}
	})

	t.Run("BrowserlessURLOverride", func(t *testing.T) {
		customURL := "http://custom-browserless:3000"
		os.Setenv("BROWSERLESS_URL", customURL)
		defer os.Unsetenv("BROWSERLESS_URL")

		cfg := loadConfig()

		if cfg.BrowserlessURL != customURL {
			t.Errorf("Expected custom browserless URL %s, got %s", customURL, cfg.BrowserlessURL)
		}
	})

	t.Run("DebugMode", func(t *testing.T) {
		os.Setenv("DEBUG", "true")
		defer os.Unsetenv("DEBUG")

		cfg := loadConfig()

		if !cfg.Debug {
			t.Error("Expected debug mode to be enabled")
		}
	})

	t.Run("OutputPathCreation", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-config")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		customOutput := filepath.Join(tempDir, "custom", "output", "path")
		os.Setenv("OUTPUT_PATH", customOutput)
		defer os.Unsetenv("OUTPUT_PATH")

		cfg := loadConfig()

		// Verify directory was created
		if _, err := os.Stat(cfg.OutputPath); os.IsNotExist(err) {
			t.Error("Expected output directory to be created")
		}
	})
}

// TestBuildStateTransitions tests build state transitions
func TestBuildStateTransitions(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("BuildingToReady", func(t *testing.T) {
		buildID := "test_state_ready"
		extensionPath := filepath.Join(env.Config.OutputPath, "test-scenario-state", "platforms", "extension")

		build := &ExtensionBuild{
			BuildID:       buildID,
			ScenarioName:  "test-scenario",
			TemplateType:  "full",
			Status:        "building",
			ExtensionPath: extensionPath,
			Config: ExtensionConfig{
				AppName:     "Test App",
				APIEndpoint: "http://localhost:3000",
			},
			BuildLog:  []string{},
			ErrorLog:  []string{},
			CreatedAt: time.Now(),
		}

		builds[buildID] = build

		// Run generation
		generateExtension(build)

		// Verify state transition
		if build.Status != "ready" {
			t.Errorf("Expected status 'ready', got '%s'", build.Status)
		}

		if build.CompletedAt == nil {
			t.Error("Expected CompletedAt to be set")
		}

		if !build.CompletedAt.After(build.CreatedAt) {
			t.Error("Expected CompletedAt to be after CreatedAt")
		}
	})

	t.Run("BuildingToFailed", func(t *testing.T) {
		buildID := "test_state_failed"

		build := &ExtensionBuild{
			BuildID:       buildID,
			ScenarioName:  "test-scenario",
			TemplateType:  "full",
			Status:        "building",
			ExtensionPath: "/invalid/path\x00/extension",
			Config: ExtensionConfig{
				AppName:     "Test App",
				APIEndpoint: "http://localhost:3000",
			},
			BuildLog:  []string{},
			ErrorLog:  []string{},
			CreatedAt: time.Now(),
		}

		builds[buildID] = build

		// Run generation (should fail)
		generateExtension(build)

		// Verify state transition
		if build.Status != "failed" {
			t.Errorf("Expected status 'failed', got '%s'", build.Status)
		}

		if build.CompletedAt == nil {
			t.Error("Expected CompletedAt to be set even for failed builds")
		}

		if len(build.ErrorLog) == 0 {
			t.Error("Expected error log entries for failed build")
		}
	})
}
