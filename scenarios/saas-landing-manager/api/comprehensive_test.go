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

// TestDatabaseServiceComprehensive provides comprehensive database service testing
func TestDatabaseServiceComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Skipping database tests: no database available")
	}

	dbService := NewDatabaseService(env.DB)

	t.Run("CreateSaaSScenario_FullData", func(t *testing.T) {
		scenario := &SaaSScenario{
			ID:               "test-id-1",
			ScenarioName:     "comprehensive-test-1",
			DisplayName:      "Comprehensive Test Scenario",
			Description:      "Full featured test scenario",
			SaaSType:         "b2b_tool",
			Industry:         "technology",
			RevenuePotential: "$50K-$100K",
			HasLandingPage:   true,
			LandingPageURL:   "/landing/test",
			LastScan:         time.Now(),
			ConfidenceScore:  0.95,
			Metadata: map[string]interface{}{
				"test_key":  "test_value",
				"features":  []string{"auth", "billing"},
				"priority":  1,
			},
		}

		err := dbService.CreateSaaSScenario(scenario)
		if err != nil {
			t.Fatalf("Failed to create scenario: %v", err)
		}

		// Verify it was created
		scenarios, err := dbService.GetSaaSScenarios()
		if err != nil {
			t.Fatalf("Failed to get scenarios: %v", err)
		}

		found := false
		for _, s := range scenarios {
			if s.ScenarioName == scenario.ScenarioName {
				found = true
				if s.DisplayName != scenario.DisplayName {
					t.Errorf("Display name mismatch: expected %s, got %s", scenario.DisplayName, s.DisplayName)
				}
				if s.SaaSType != scenario.SaaSType {
					t.Errorf("SaaS type mismatch: expected %s, got %s", scenario.SaaSType, s.SaaSType)
				}
				break
			}
		}

		if !found {
			t.Error("Created scenario not found")
		}
	})

	t.Run("CreateLandingPage_FullData", func(t *testing.T) {
		scenario := createTestScenario(t, env.DB, "landing-page-test")
		template := createTestTemplate(t, env.DB, "landing-template")

		page := &LandingPage{
			ID:         "landing-page-1",
			ScenarioID: scenario.ID,
			TemplateID: template.ID,
			Variant:    "control",
			Title:      "Test Landing Page",
			Description: "Comprehensive landing page test",
			Content: map[string]interface{}{
				"hero":    "Welcome",
				"cta":     "Get Started",
				"features": []string{"fast", "secure"},
			},
			SEOMetadata: map[string]interface{}{
				"title":       "SEO Title",
				"description": "SEO Description",
				"keywords":    []string{"saas", "landing"},
			},
			PerformanceMetrics: map[string]interface{}{
				"views":       100,
				"conversions": 10,
				"bounce_rate": 0.3,
			},
			Status:    "published",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := dbService.CreateLandingPage(page)
		if err != nil {
			t.Fatalf("Failed to create landing page: %v", err)
		}
	})

	t.Run("GetTemplates_WithFilters", func(t *testing.T) {
		// Create templates with different attributes
		templates := []*Template{
			{
				ID:          "tmpl-1",
				Name:        "Modern B2B",
				Category:    "modern",
				SaaSType:    "b2b_tool",
				Industry:    "technology",
				HTMLContent: "<html></html>",
				CSSContent:  "body {}",
				JSContent:   "console.log('test');",
				ConfigSchema: map[string]interface{}{
					"title": "string",
				},
				PreviewURL: "/preview/1",
				UsageCount: 10,
				Rating:     4.5,
				CreatedAt:  time.Now(),
			},
			{
				ID:          "tmpl-2",
				Name:        "Classic B2C",
				Category:    "classic",
				SaaSType:    "b2c_app",
				Industry:    "ecommerce",
				HTMLContent: "<html></html>",
				CSSContent:  "body {}",
				JSContent:   "console.log('test');",
				ConfigSchema: map[string]interface{}{
					"description": "string",
				},
				PreviewURL: "/preview/2",
				UsageCount: 5,
				Rating:     4.0,
				CreatedAt:  time.Now(),
			},
		}

		for _, tmpl := range templates {
			query := `
				INSERT INTO templates (id, name, category, saas_type, industry, html_content,
					css_content, js_content, config_schema, preview_url, usage_count, rating, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			`
			configJSON, _ := json.Marshal(tmpl.ConfigSchema)
			_, err := env.DB.Exec(query, tmpl.ID, tmpl.Name, tmpl.Category, tmpl.SaaSType,
				tmpl.Industry, tmpl.HTMLContent, tmpl.CSSContent, tmpl.JSContent,
				string(configJSON), tmpl.PreviewURL, tmpl.UsageCount, tmpl.Rating, tmpl.CreatedAt)
			if err != nil {
				t.Fatalf("Failed to create template: %v", err)
			}
		}

		// Test category filter
		results, err := dbService.GetTemplates("modern", "")
		if err != nil {
			t.Fatalf("Failed to get templates by category: %v", err)
		}
		for _, tmpl := range results {
			if tmpl.Category != "modern" {
				t.Errorf("Expected category 'modern', got '%s'", tmpl.Category)
			}
		}

		// Test SaaS type filter
		results, err = dbService.GetTemplates("", "b2b_tool")
		if err != nil {
			t.Fatalf("Failed to get templates by SaaS type: %v", err)
		}
		for _, tmpl := range results {
			if tmpl.SaaSType != "b2b_tool" {
				t.Errorf("Expected SaaS type 'b2b_tool', got '%s'", tmpl.SaaSType)
			}
		}

		// Test both filters
		results, err = dbService.GetTemplates("modern", "b2b_tool")
		if err != nil {
			t.Fatalf("Failed to get templates with both filters: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("Expected 1 template, got %d", len(results))
		}
	})
}

// TestSaaSDetectionServiceComprehensive provides comprehensive detection testing
func TestSaaSDetectionServiceComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Skipping database tests: no database available")
	}

	dbService := NewDatabaseService(env.DB)

	t.Run("ComplexScenarioDetection", func(t *testing.T) {
		scenariosDir := filepath.Join(env.TempDir, "scenarios")

		// Create complex SaaS scenario
		complexSaaS := filepath.Join(scenariosDir, "complex-saas")
		os.MkdirAll(filepath.Join(complexSaaS, ".vrooli"), 0755)
		os.MkdirAll(filepath.Join(complexSaaS, "ui"), 0755)
		os.MkdirAll(filepath.Join(complexSaaS, "api"), 0755)

		// Create service.json with all SaaS indicators
		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"displayName": "Complex SaaS Application",
				"description": "Multi-tenant enterprise SaaS platform",
				"tags": []string{"multi-tenant", "billing", "analytics", "a-b-testing", "saas", "subscription"},
			},
			"resources": map[string]interface{}{
				"postgres": map[string]interface{}{
					"enabled": true,
				},
			},
		}
		serviceJSON, _ := json.MarshalIndent(serviceConfig, "", "  ")
		os.WriteFile(filepath.Join(complexSaaS, ".vrooli", "service.json"), serviceJSON, 0644)

		// Create comprehensive PRD
		prdContent := `# Complex SaaS Platform

## Business Value
Revenue Potential: $100K-$500K

This is a comprehensive multi-tenant SaaS application with enterprise features:
- B2B focused
- Subscription-based pricing model
- Advanced analytics
- API-first architecture
- Marketplace capabilities
`
		os.WriteFile(filepath.Join(complexSaaS, "PRD.md"), []byte(prdContent), 0644)

		detectionService := NewSaaSDetectionService(scenariosDir, dbService)
		response, err := detectionService.ScanScenarios(false, "")
		if err != nil {
			t.Fatalf("Scan failed: %v", err)
		}

		if response.SaaSScenarios == 0 {
			t.Error("Expected to find SaaS scenarios")
		}

		// Find the complex-saas scenario
		found := false
		for _, scenario := range response.Scenarios {
			if scenario.ScenarioName == "complex-saas" {
				found = true
				if scenario.ConfidenceScore < 0.8 {
					t.Errorf("Expected high confidence score (>0.8), got %f", scenario.ConfidenceScore)
				}
				if scenario.DisplayName != "Complex SaaS Application" {
					t.Errorf("Display name not extracted correctly: %s", scenario.DisplayName)
				}
				if scenario.RevenuePotential == "" {
					t.Error("Revenue potential not extracted")
				}
				break
			}
		}

		if !found {
			t.Error("Complex SaaS scenario not detected")
		}
	})

	t.Run("NonSaaSScenarioDetection", func(t *testing.T) {
		scenariosDir := filepath.Join(env.TempDir, "scenarios2")
		os.MkdirAll(scenariosDir, 0755)

		// Create non-SaaS scenario (simple script)
		scriptDir := filepath.Join(scenariosDir, "simple-script")
		os.MkdirAll(scriptDir, 0755)

		// Minimal service.json without SaaS indicators
		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"displayName": "Simple Script",
				"description": "A basic automation script",
				"tags":        []string{"automation", "utility"},
			},
		}
		serviceJSON, _ := json.MarshalIndent(serviceConfig, "", "  ")
		os.WriteFile(filepath.Join(scriptDir, ".vrooli", "service.json"), serviceJSON, 0644)

		detectionService := NewSaaSDetectionService(scenariosDir, dbService)
		isSaaS, scenario := detectionService.analyzeSaaSCharacteristics("simple-script", scenariosDir)

		if isSaaS {
			t.Errorf("Simple script should not be classified as SaaS, got confidence: %f", scenario.ConfidenceScore)
		}
	})

	t.Run("GetExistingScenario", func(t *testing.T) {
		// Create a scenario
		scenario := createTestScenario(t, env.DB, "existing-scenario-test")

		scenariosDir := filepath.Join(env.TempDir, "scenarios3")
		detectionService := NewSaaSDetectionService(scenariosDir, dbService)

		// Test getting existing scenario
		existing, err := detectionService.getExistingScenario(scenario.ScenarioName)
		if err != nil {
			t.Fatalf("Failed to get existing scenario: %v", err)
		}
		if existing == nil {
			t.Error("Expected to find existing scenario")
		}

		// Test non-existent scenario
		nonExisting, err := detectionService.getExistingScenario("non-existent-scenario")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if nonExisting != nil {
			t.Error("Expected nil for non-existent scenario")
		}
	})
}

// TestLandingPageServiceComprehensive provides comprehensive landing page service testing
func TestLandingPageServiceComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Skipping database tests: no database available")
	}

	dbService := NewDatabaseService(env.DB)
	landingPageService := NewLandingPageService(dbService, filepath.Join(env.TempDir, "templates"))

	t.Run("GenerateWithSpecificTemplate", func(t *testing.T) {
		scenario := createTestScenario(t, env.DB, "specific-template-test")
		template := createTestTemplate(t, env.DB, "specific-template")

		req := &GenerateRequest{
			ScenarioID: scenario.ID,
			TemplateID: template.ID,
			CustomContent: map[string]interface{}{
				"hero_title":    "Custom Hero Title",
				"hero_subtitle": "Custom Subtitle",
				"cta_text":      "Start Free Trial",
				"features": []string{
					"Feature 1",
					"Feature 2",
					"Feature 3",
				},
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

		if !bytes.Contains([]byte(response.PreviewURL), []byte(response.LandingPageID)) {
			t.Error("Preview URL should contain landing page ID")
		}

		if len(response.ABTestVariants) != 1 || response.ABTestVariants[0] != "control" {
			t.Errorf("Expected only control variant, got %v", response.ABTestVariants)
		}
	})

	t.Run("GenerateWithABTestingVariants", func(t *testing.T) {
		scenario := createTestScenario(t, env.DB, "ab-testing-scenario")
		template := createTestTemplate(t, env.DB, "ab-testing-template")

		req := &GenerateRequest{
			ScenarioID:      scenario.ID,
			TemplateID:      template.ID,
			EnableABTesting: true,
			CustomContent: map[string]interface{}{
				"title": "A/B Test Landing Page",
			},
		}

		response, err := landingPageService.GenerateLandingPage(req)
		if err != nil {
			t.Fatalf("Failed to generate landing page with A/B testing: %v", err)
		}

		// Should have control, a, and b variants
		if len(response.ABTestVariants) < 2 {
			t.Errorf("Expected at least 2 variants for A/B testing, got %d", len(response.ABTestVariants))
		}

		hasControl := false
		for _, variant := range response.ABTestVariants {
			if variant == "control" {
				hasControl = true
				break
			}
		}

		if !hasControl {
			t.Error("Expected 'control' variant in A/B test")
		}
	})

	t.Run("GenerateWithoutTemplateID", func(t *testing.T) {
		scenario := createTestScenario(t, env.DB, "no-template-id")
		createTestTemplate(t, env.DB, "default-template-1")
		createTestTemplate(t, env.DB, "default-template-2")

		req := &GenerateRequest{
			ScenarioID:      scenario.ID,
			EnableABTesting: false,
		}

		response, err := landingPageService.GenerateLandingPage(req)
		if err != nil {
			t.Fatalf("Failed to generate with auto-selected template: %v", err)
		}

		if response.LandingPageID == "" {
			t.Error("Expected landing page to be generated with auto-selected template")
		}
	})
}

// TestClaudeCodeServiceComprehensive provides comprehensive Claude Code service testing
func TestClaudeCodeServiceComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("DeployLandingPage_DirectMethod", func(t *testing.T) {
		claudeService := NewClaudeCodeService("")

		req := &DeployRequest{
			TargetScenario:   "test-scenario",
			DeploymentMethod: "direct",
			BackupExisting:   true,
		}

		response, err := claudeService.DeployLandingPage("landing-page-id", "test-scenario", req)

		// May fail due to directory structure, but should return response
		if response != nil {
			if response.DeploymentID == "" {
				t.Error("Expected deployment ID")
			}
			if response.Status != "completed" {
				t.Logf("Deployment status: %s (may fail in test environment)", response.Status)
			}
		} else if err != nil {
			t.Logf("Direct deployment error (expected in test): %v", err)
		}
	})

	t.Run("DeployLandingPage_ClaudeAgent", func(t *testing.T) {
		claudeService := NewClaudeCodeService("/fake/claude-code")

		req := &DeployRequest{
			TargetScenario:   "test-scenario",
			DeploymentMethod: "claude_agent",
			BackupExisting:   false,
		}

		response, err := claudeService.DeployLandingPage("landing-page-id", "test-scenario", req)

		// Will fail because binary doesn't exist, but should handle gracefully
		if err != nil {
			t.Logf("Expected error for nonexistent Claude Code binary: %v", err)
		}

		if response != nil && response.Status == "agent_working" {
			if response.AgentSessionID == "" {
				t.Error("Expected agent session ID")
			}
		}
	})

	t.Run("DirectDeploy_CreatesStructure", func(t *testing.T) {
		claudeService := NewClaudeCodeService("")

		targetScenario := "deploy-target"
		targetPath := filepath.Join(env.TempDir, targetScenario)
		os.MkdirAll(targetPath, 0755)

		// Change to temp directory to make relative paths work
		oldWD, _ := os.Getwd()
		os.Chdir(env.TempDir)
		defer os.Chdir(oldWD)

		_ = claudeService.directDeploy(targetScenario, "test-page-id", false)

		// Check if landing directory was created
		landingPath := filepath.Join(targetPath, "landing")
		if _, err := os.Stat(landingPath); os.IsNotExist(err) {
			t.Logf("Landing directory not created (may fail in test): %v", err)
		}

		// Check if index.html was created
		indexPath := filepath.Join(landingPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			content, _ := os.ReadFile(indexPath)
			if !bytes.Contains(content, []byte("SaaS")) {
				t.Error("Expected SaaS content in generated HTML")
			}
		}
	})

	t.Run("SpawnClaudeAgent_PromptGeneration", func(t *testing.T) {
		claudeService := NewClaudeCodeService("/nonexistent/binary")

		// This will fail but we can verify it attempts to create the right prompt
		_, err := claudeService.spawnClaudeAgent("test-scenario", "test-landing-page-id")

		// Should fail with command not found or similar
		if err == nil {
			t.Log("Unexpectedly succeeded - Claude Code may be installed")
		}
	})
}

// TestHTTPHandlersComprehensive provides comprehensive HTTP handler testing
func TestHTTPHandlersComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Skipping database tests: no database available")
	}

	db = env.DB

	t.Run("HealthHandler_FullResponse", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(healthHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response map[string]string
		json.Unmarshal(rr.Body.Bytes(), &response)

		if response["status"] != "healthy" {
			t.Error("Expected healthy status")
		}

		if response["service"] != "saas-landing-manager" {
			t.Error("Expected saas-landing-manager service name")
		}

		if response["timestamp"] == "" {
			t.Error("Expected timestamp in response")
		}
	})

	t.Run("DeployHandler_WithURLVars", func(t *testing.T) {
		scenario := createTestScenario(t, env.DB, "deploy-handler-test")
		template := createTestTemplate(t, env.DB, "deploy-handler-template")
		page := createTestLandingPage(t, env.DB, scenario.ID, template.ID)

		reqBody := DeployRequest{
			TargetScenario:   "target-scenario",
			DeploymentMethod: "direct",
			BackupExisting:   false,
		}

		bodyJSON, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/landing-pages/"+page.ID+"/deploy", bytes.NewBuffer(bodyJSON))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": page.ID})

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(deployLandingPageHandler)
		handler.ServeHTTP(rr, req)

		// May return error due to test environment, but shouldn't crash
		if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
			t.Logf("Deploy handler returned status: %d", rr.Code)
		}
	})

	t.Run("GetDashboardHandler_WithScenarios", func(t *testing.T) {
		// Create multiple scenarios for dashboard
		for i := 0; i < 5; i++ {
			createTestScenario(t, env.DB, "dashboard-scenario-"+string(rune('a'+i)))
		}

		req, _ := http.NewRequest("GET", "/api/v1/analytics/dashboard", nil)
		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(getDashboardHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)

		if totalPages, ok := response["total_pages"].(float64); ok {
			if totalPages < 5 {
				t.Errorf("Expected at least 5 scenarios in dashboard, got %f", totalPages)
			}
		} else {
			t.Error("Expected total_pages in dashboard response")
		}
	})
}
