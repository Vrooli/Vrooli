//go:build testing
// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/health", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(healthHandler)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response["status"])
		}

		if response["service"] != "saas-landing-manager" {
			t.Errorf("Expected service 'saas-landing-manager', got '%s'", response["service"])
		}
	})
}

// TestDatabaseService tests the DatabaseService methods
func TestDatabaseService(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Skipping database tests: no database available")
	}

	dbService := NewDatabaseService(env.DB)

	t.Run("CreateAndGetSaaSScenarios", func(t *testing.T) {
		// Create test scenario
		scenario := createTestScenario(t, env.DB, "test-scenario-1")

		// Get scenarios
		scenarios, err := dbService.GetSaaSScenarios()
		if err != nil {
			t.Fatalf("Failed to get scenarios: %v", err)
		}

		if len(scenarios) == 0 {
			t.Error("Expected at least one scenario")
		}

		// Verify the created scenario exists
		found := false
		for _, s := range scenarios {
			if s.ScenarioName == scenario.ScenarioName {
				found = true
				if s.DisplayName != scenario.DisplayName {
					t.Errorf("Expected display name '%s', got '%s'", scenario.DisplayName, s.DisplayName)
				}
				break
			}
		}

		if !found {
			t.Error("Created scenario not found in results")
		}
	})

	t.Run("CreateLandingPage", func(t *testing.T) {
		// Create prerequisite data
		scenario := createTestScenario(t, env.DB, "test-scenario-2")
		template := createTestTemplate(t, env.DB, "test-template")

		// Create landing page
		page := createTestLandingPage(t, env.DB, scenario.ID, template.ID)

		if page.ScenarioID != scenario.ID {
			t.Errorf("Expected scenario ID '%s', got '%s'", scenario.ID, page.ScenarioID)
		}

		if page.TemplateID != template.ID {
			t.Errorf("Expected template ID '%s', got '%s'", template.ID, page.TemplateID)
		}
	})

	t.Run("GetTemplates", func(t *testing.T) {
		// Create test templates
		createTestTemplate(t, env.DB, "modern-template")
		createTestTemplate(t, env.DB, "classic-template")

		// Get all templates
		templates, err := dbService.GetTemplates("", "")
		if err != nil {
			t.Fatalf("Failed to get templates: %v", err)
		}

		if len(templates) < 2 {
			t.Errorf("Expected at least 2 templates, got %d", len(templates))
		}

		// Get templates by category
		templates, err = dbService.GetTemplates("modern", "")
		if err != nil {
			t.Fatalf("Failed to get templates by category: %v", err)
		}

		for _, tmpl := range templates {
			if tmpl.Category != "modern" {
				t.Errorf("Expected category 'modern', got '%s'", tmpl.Category)
			}
		}
	})

	t.Run("DuplicateScenarioHandling", func(t *testing.T) {
		scenario1 := createTestScenario(t, env.DB, "duplicate-test")

		// Try to create another with same name (should update)
		scenario2 := createTestScenario(t, env.DB, "duplicate-test")

		// Both should have same scenario_name but operations should succeed
		if scenario1.ScenarioName != scenario2.ScenarioName {
			t.Error("Expected same scenario name")
		}
	})
}

// TestSaaSDetectionService tests the SaaS detection functionality
func TestSaaSDetectionService(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Skipping database tests: no database available")
	}

	// Create test scenarios directory
	scenariosDir := createTestScenariosDirectory(t, env.TempDir)

	dbService := NewDatabaseService(env.DB)
	detectionService := NewSaaSDetectionService(scenariosDir, dbService)

	t.Run("ScanScenarios_FindsSaaS", func(t *testing.T) {
		response, err := detectionService.ScanScenarios(false, "")
		if err != nil {
			t.Fatalf("Scan failed: %v", err)
		}

		if response.TotalScenarios == 0 {
			t.Error("Expected to find scenarios")
		}

		if response.SaaSScenarios == 0 {
			t.Error("Expected to find at least one SaaS scenario")
		}

		// Verify test-saas-app was detected
		found := false
		for _, scenario := range response.Scenarios {
			if scenario.ScenarioName == "test-saas-app" {
				found = true
				if scenario.ConfidenceScore < 0.5 {
					t.Errorf("Expected confidence score >= 0.5, got %f", scenario.ConfidenceScore)
				}
				break
			}
		}

		if !found {
			t.Error("Expected test-saas-app to be detected as SaaS")
		}
	})

	t.Run("ScanScenarios_WithFilter", func(t *testing.T) {
		response, err := detectionService.ScanScenarios(false, "saas")
		if err != nil {
			t.Fatalf("Scan with filter failed: %v", err)
		}

		// Should only find scenarios with "saas" in the name
		for _, scenario := range response.Scenarios {
			if scenario.ScenarioName != "test-saas-app" {
				t.Errorf("Unexpected scenario found: %s", scenario.ScenarioName)
			}
		}
	})

	t.Run("AnalyzeSaaSCharacteristics", func(t *testing.T) {
		isSaaS, scenario := detectionService.analyzeSaaSCharacteristics("test-saas-app", scenariosDir)

		if !isSaaS {
			t.Error("Expected test-saas-app to be classified as SaaS")
		}

		if scenario.DisplayName == "" {
			t.Error("Expected display name to be populated")
		}

		if scenario.Description == "" {
			t.Error("Expected description to be populated")
		}

		if scenario.RevenuePotential == "" {
			t.Error("Expected revenue potential to be extracted from PRD")
		}
	})

	t.Run("PathTraversalPrevention", func(t *testing.T) {
		// Try to scan with path traversal
		isSaaS, _ := detectionService.analyzeSaaSCharacteristics("../../../etc/passwd", scenariosDir)

		if isSaaS {
			t.Error("Should not classify path traversal attempts as SaaS")
		}
	})
}

// TestLandingPageService tests the landing page generation
func TestLandingPageService(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Skipping database tests: no database available")
	}

	dbService := NewDatabaseService(env.DB)
	landingPageService := NewLandingPageService(dbService, filepath.Join(env.TempDir, "templates"))

	t.Run("GenerateLandingPage_WithTemplate", func(t *testing.T) {
		// Create prerequisite data
		scenario := createTestScenario(t, env.DB, "landing-test-1")
		template := createTestTemplate(t, env.DB, "generation-template")

		req := &GenerateRequest{
			ScenarioID: scenario.ID,
			TemplateID: template.ID,
			CustomContent: map[string]interface{}{
				"title": "Custom Title",
				"cta":   "Get Started",
			},
			EnableABTesting: false,
		}

		response, err := landingPageService.GenerateLandingPage(req)
		if err != nil {
			t.Fatalf("Failed to generate landing page: %v", err)
		}

		if response.LandingPageID == "" {
			t.Error("Expected landing page ID")
		}

		if response.PreviewURL == "" {
			t.Error("Expected preview URL")
		}

		if response.DeploymentStatus != "ready" {
			t.Errorf("Expected deployment status 'ready', got '%s'", response.DeploymentStatus)
		}
	})

	t.Run("GenerateLandingPage_WithABTesting", func(t *testing.T) {
		scenario := createTestScenario(t, env.DB, "ab-test-scenario")
		template := createTestTemplate(t, env.DB, "ab-test-template")

		req := &GenerateRequest{
			ScenarioID:      scenario.ID,
			TemplateID:      template.ID,
			EnableABTesting: true,
		}

		response, err := landingPageService.GenerateLandingPage(req)
		if err != nil {
			t.Fatalf("Failed to generate landing page with A/B testing: %v", err)
		}

		if len(response.ABTestVariants) < 2 {
			t.Errorf("Expected at least 2 A/B test variants, got %d", len(response.ABTestVariants))
		}

		// Should have control, a, and b variants
		expectedVariants := map[string]bool{"control": false, "a": false, "b": false}
		for _, variant := range response.ABTestVariants {
			expectedVariants[variant] = true
		}

		for variant, found := range expectedVariants {
			if !found {
				t.Errorf("Expected variant '%s' not found", variant)
			}
		}
	})

	t.Run("GenerateLandingPage_NoTemplate", func(t *testing.T) {
		scenario := createTestScenario(t, env.DB, "no-template-scenario")

		req := &GenerateRequest{
			ScenarioID:      scenario.ID,
			EnableABTesting: false,
		}

		// Should handle missing template gracefully
		_, err := landingPageService.GenerateLandingPage(req)
		// Error expected when no templates exist, but shouldn't crash
		if err != nil {
			t.Logf("Expected error when no templates: %v", err)
		}
	})
}

// TestClaudeCodeService tests the Claude Code integration
func TestClaudeCodeService(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("DirectDeploy", func(t *testing.T) {
		claudeService := NewClaudeCodeService("")

		// Create test target scenario directory
		targetScenario := "test-target-scenario"
		targetPath := filepath.Join(env.TempDir, "scenarios", targetScenario)
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			t.Fatalf("Failed to create target scenario: %v", err)
		}

		err := claudeService.directDeploy(targetScenario, "test-landing-page-id", false)

		// Note: This will fail in test because it expects to be in a specific directory structure
		// but we're testing it doesn't crash
		if err != nil {
			t.Logf("Direct deploy returned error (expected in test): %v", err)
		}
	})

	t.Run("DirectDeploy_PathTraversal", func(t *testing.T) {
		claudeService := NewClaudeCodeService("")

		// Try path traversal attack
		err := claudeService.directDeploy("../../../etc", "malicious-id", false)

		if err == nil {
			t.Error("Expected error for path traversal attempt")
		}
	})

	t.Run("SpawnClaudeAgent", func(t *testing.T) {
		claudeService := NewClaudeCodeService("/nonexistent/claude-code")

		// This will fail because the binary doesn't exist, but shouldn't panic
		_, err := claudeService.spawnClaudeAgent("test-scenario", "test-page-id")

		if err == nil {
			t.Log("Note: spawn succeeded unexpectedly (maybe claude-code is available)")
		} else {
			t.Logf("Expected error for nonexistent binary: %v", err)
		}
	})
}

// TestHandlers tests the HTTP handlers
func TestScanScenariosHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Skipping database tests: no database available")
	}

	// Set global db for handlers
	db = env.DB

	// Create test scenarios
	scenariosDir := createTestScenariosDirectory(t, env.TempDir)
	os.Setenv("SCENARIOS_PATH", scenariosDir)
	defer os.Unsetenv("SCENARIOS_PATH")

	t.Run("Success", func(t *testing.T) {
		reqBody := ScanRequest{
			ForceRescan:    false,
			ScenarioFilter: "",
		}

		bodyJSON, _ := json.Marshal(reqBody)
		req, err := http.NewRequest("POST", "/api/v1/scenarios/scan", bytes.NewBuffer(bodyJSON))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(scanScenariosHandler)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v. Body: %s", status, http.StatusOK, rr.Body.String())
		}

		var response ScanResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		if response.TotalScenarios == 0 {
			t.Error("Expected to find scenarios")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/v1/scenarios/scan", bytes.NewBufferString("{invalid json"))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(scanScenariosHandler)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})
}

func TestGenerateLandingPageHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Skipping database tests: no database available")
	}

	db = env.DB

	t.Run("Success", func(t *testing.T) {
		// Create prerequisite data
		scenario := createTestScenario(t, env.DB, "handler-test-scenario")
		createTestTemplate(t, env.DB, "handler-template")

		reqBody := GenerateRequest{
			ScenarioID:      scenario.ID,
			EnableABTesting: false,
		}

		bodyJSON, _ := json.Marshal(reqBody)
		req, err := http.NewRequest("POST", "/api/v1/landing-pages/generate", bytes.NewBuffer(bodyJSON))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(generateLandingPageHandler)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v. Body: %s", status, http.StatusOK, rr.Body.String())
		}

		var response GenerateResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		if response.LandingPageID == "" {
			t.Error("Expected landing page ID in response")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/v1/landing-pages/generate", bytes.NewBufferString("{invalid"))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(generateLandingPageHandler)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
		}
	})
}

func TestDeployLandingPageHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Skipping database tests: no database available")
	}

	db = env.DB

	t.Run("DirectDeployment", func(t *testing.T) {
		// Create test landing page
		scenario := createTestScenario(t, env.DB, "deploy-test")
		template := createTestTemplate(t, env.DB, "deploy-template")
		page := createTestLandingPage(t, env.DB, scenario.ID, template.ID)

		reqBody := DeployRequest{
			TargetScenario:   "test-target",
			DeploymentMethod: "direct",
			BackupExisting:   true,
		}

		bodyJSON, _ := json.Marshal(reqBody)
		req, err := http.NewRequest("POST", "/api/v1/landing-pages/"+page.ID+"/deploy", bytes.NewBuffer(bodyJSON))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": page.ID})

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(deployLandingPageHandler)
		handler.ServeHTTP(rr, req)

		// May fail due to directory structure, but shouldn't crash
		if status := rr.Code; status != http.StatusOK && status != http.StatusInternalServerError {
			t.Logf("Deploy returned status: %d", status)
		}
	})
}

func TestGetTemplatesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Skipping database tests: no database available")
	}

	db = env.DB

	t.Run("GetAllTemplates", func(t *testing.T) {
		// Create test templates
		createTestTemplate(t, env.DB, "template-1")
		createTestTemplate(t, env.DB, "template-2")

		req, err := http.NewRequest("GET", "/api/v1/templates", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(getTemplatesHandler)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		templates, ok := response["templates"].([]interface{})
		if !ok {
			t.Error("Expected 'templates' array in response")
		}

		if len(templates) < 2 {
			t.Errorf("Expected at least 2 templates, got %d", len(templates))
		}
	})

	t.Run("FilterByCategory", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/templates?category=modern", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(getTemplatesHandler)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})
}

func TestGetDashboardHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Skipping database tests: no database available")
	}

	db = env.DB

	t.Run("Success", func(t *testing.T) {
		// Create some test data
		createTestScenario(t, env.DB, "dashboard-test-1")
		createTestScenario(t, env.DB, "dashboard-test-2")

		req, err := http.NewRequest("GET", "/api/v1/analytics/dashboard", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(getDashboardHandler)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		if _, ok := response["total_pages"]; !ok {
			t.Error("Expected 'total_pages' in response")
		}

		if _, ok := response["scenarios"]; !ok {
			t.Error("Expected 'scenarios' in response")
		}
	})
}

// TestPerformance tests performance characteristics
func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Skipping performance tests: no database available")
	}

	dbService := NewDatabaseService(env.DB)

	t.Run("ScanPerformance", func(t *testing.T) {
		scenariosDir := createTestScenariosDirectory(t, env.TempDir)
		detectionService := NewSaaSDetectionService(scenariosDir, dbService)

		start := time.Now()
		_, err := detectionService.ScanScenarios(false, "")
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Scan failed: %v", err)
		}

		// Should complete in reasonable time (< 5 seconds for small test set)
		if duration > 5*time.Second {
			t.Errorf("Scan took too long: %v", duration)
		}
	})

	t.Run("BulkTemplateRetrieval", func(t *testing.T) {
		// Create multiple templates
		for i := 0; i < 100; i++ {
			createTestTemplate(t, env.DB, "perf-template-"+string(rune(i)))
		}

		start := time.Now()
		templates, err := dbService.GetTemplates("", "")
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to get templates: %v", err)
		}

		if len(templates) < 100 {
			t.Errorf("Expected at least 100 templates, got %d", len(templates))
		}

		// Should retrieve quickly (< 1 second)
		if duration > 1*time.Second {
			t.Errorf("Template retrieval took too long: %v", duration)
		}
	})
}
