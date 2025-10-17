// +build testing

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestModels tests the data models
func TestModels(t *testing.T) {
	t.Run("SaaSScenario", func(t *testing.T) {
		scenario := SaaSScenario{
			ID:              uuid.New().String(),
			ScenarioName:    "test-scenario",
			DisplayName:     "Test Scenario",
			Description:     "A test scenario",
			SaaSType:        "b2b_tool",
			Industry:        "technology",
			RevenuePotential: "$10K-$50K",
			HasLandingPage:  false,
			LandingPageURL:  "",
			LastScan:        time.Now(),
			ConfidenceScore: 0.8,
			Metadata:        map[string]interface{}{"key": "value"},
		}

		if scenario.ScenarioName != "test-scenario" {
			t.Errorf("Expected scenario_name 'test-scenario', got '%s'", scenario.ScenarioName)
		}

		if scenario.ConfidenceScore != 0.8 {
			t.Errorf("Expected confidence score 0.8, got %f", scenario.ConfidenceScore)
		}
	})

	t.Run("LandingPage", func(t *testing.T) {
		page := LandingPage{
			ID:          uuid.New().String(),
			ScenarioID:  uuid.New().String(),
			TemplateID:  uuid.New().String(),
			Variant:     "control",
			Title:       "Test Page",
			Description: "Test description",
			Content: map[string]interface{}{
				"hero": "Welcome",
			},
			SEOMetadata:        map[string]interface{}{"title": "SEO Title"},
			PerformanceMetrics: map[string]interface{}{"views": 100},
			Status:             "active",
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		if page.Variant != "control" {
			t.Errorf("Expected variant 'control', got '%s'", page.Variant)
		}

		if page.Status != "active" {
			t.Errorf("Expected status 'active', got '%s'", page.Status)
		}
	})

	t.Run("Template", func(t *testing.T) {
		template := Template{
			ID:          uuid.New().String(),
			Name:        "Modern Template",
			Category:    "modern",
			SaaSType:    "b2b_tool",
			Industry:    "technology",
			HTMLContent: "<html></html>",
			CSSContent:  "body {}",
			JSContent:   "console.log('test');",
			ConfigSchema: map[string]interface{}{
				"title": "string",
			},
			PreviewURL: "/preview/modern",
			UsageCount: 10,
			Rating:     4.5,
			CreatedAt:  time.Now(),
		}

		if template.Category != "modern" {
			t.Errorf("Expected category 'modern', got '%s'", template.Category)
		}

		if template.Rating != 4.5 {
			t.Errorf("Expected rating 4.5, got %f", template.Rating)
		}
	})

	t.Run("ABTestResult", func(t *testing.T) {
		result := ABTestResult{
			ID:            uuid.New().String(),
			LandingPageID: uuid.New().String(),
			Variant:       "a",
			MetricName:    "conversion_rate",
			MetricValue:   0.15,
			Timestamp:     time.Now(),
			SessionID:     uuid.New().String(),
			UserAgent:     "Mozilla/5.0",
		}

		if result.Variant != "a" {
			t.Errorf("Expected variant 'a', got '%s'", result.Variant)
		}

		if result.MetricValue != 0.15 {
			t.Errorf("Expected metric value 0.15, got %f", result.MetricValue)
		}
	})
}

// TestRequestResponseTypes tests request and response structures
func TestRequestResponseTypes(t *testing.T) {
	t.Run("ScanRequest", func(t *testing.T) {
		req := ScanRequest{
			ForceRescan:     true,
			ScenarioFilter: "test",
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded ScanRequest
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.ForceRescan != req.ForceRescan {
			t.Error("ForceRescan not preserved")
		}

		if decoded.ScenarioFilter != req.ScenarioFilter {
			t.Error("ScenarioFilter not preserved")
		}
	})

	t.Run("GenerateRequest", func(t *testing.T) {
		req := GenerateRequest{
			ScenarioID:      uuid.New().String(),
			TemplateID:      uuid.New().String(),
			CustomContent:   map[string]interface{}{"key": "value"},
			EnableABTesting: true,
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded GenerateRequest
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.EnableABTesting != req.EnableABTesting {
			t.Error("EnableABTesting not preserved")
		}
	})

	t.Run("DeployRequest", func(t *testing.T) {
		req := DeployRequest{
			TargetScenario:   "test-scenario",
			DeploymentMethod: "claude_agent",
			BackupExisting:   true,
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded DeployRequest
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.TargetScenario != req.TargetScenario {
			t.Error("TargetScenario not preserved")
		}

		if decoded.DeploymentMethod != req.DeploymentMethod {
			t.Error("DeploymentMethod not preserved")
		}
	})
}

// TestSaaSDetectionWithoutDB tests SaaS detection logic without database
func TestSaaSDetectionWithoutDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("PathTraversal", func(t *testing.T) {
		// Create a temporary directory
		tempDir, err := os.MkdirTemp("", "saas-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		// Create a mock detection service (no DB needed for this test)
		detectionService := &SaaSDetectionService{
			scenariosPath: tempDir,
			dbService:     nil, // No DB needed for path validation
		}

		// Test path traversal prevention
		isSaaS, _ := detectionService.analyzeSaaSCharacteristics("../../../etc/passwd", tempDir)
		if isSaaS {
			t.Error("Path traversal should not be classified as SaaS")
		}

		isSaaS2, _ := detectionService.analyzeSaaSCharacteristics("..\\..\\windows\\system32", tempDir)
		if isSaaS2 {
			t.Error("Path traversal (Windows style) should not be classified as SaaS")
		}
	})

	t.Run("AnalyzeCharacteristics", func(t *testing.T) {
		// Create test scenario structure
		tempDir, err := os.MkdirTemp("", "saas-analyze")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		scenarioDir := filepath.Join(tempDir, "test-app")
		os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755)
		os.MkdirAll(filepath.Join(scenarioDir, "ui"), 0755)
		os.MkdirAll(filepath.Join(scenarioDir, "api"), 0755)

		// Create service.json with SaaS indicators
		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"displayName": "Test SaaS Application",
				"description": "A multi-tenant SaaS platform",
				"tags":        []string{"saas", "multi-tenant", "business-application"},
			},
			"resources": map[string]interface{}{
				"postgres": map[string]interface{}{
					"enabled": true,
				},
			},
		}

		serviceJSON, _ := json.MarshalIndent(serviceConfig, "", "  ")
		os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), serviceJSON, 0644)

		// Create PRD with revenue indicators
		prdContent := `# Test SaaS App

## Business Value
Revenue Potential: $25K-$100K

This is a subscription-based SaaS platform with enterprise pricing tiers.
`
		os.WriteFile(filepath.Join(scenarioDir, "PRD.md"), []byte(prdContent), 0644)

		detectionService := &SaaSDetectionService{
			scenariosPath: tempDir,
		}

		isSaaS, scenario := detectionService.analyzeSaaSCharacteristics("test-app", tempDir)

		if !isSaaS {
			t.Error("Expected test-app to be classified as SaaS")
		}

		if scenario.DisplayName != "Test SaaS Application" {
			t.Errorf("Expected display name 'Test SaaS Application', got '%s'", scenario.DisplayName)
		}

		if scenario.RevenuePotential == "" {
			t.Error("Expected revenue potential to be extracted")
		}

		if scenario.ConfidenceScore < 0.5 {
			t.Errorf("Expected confidence score >= 0.5, got %f", scenario.ConfidenceScore)
		}

		// Verify characteristics were recorded
		if characteristics, ok := scenario.Metadata["characteristics"].([]string); ok {
			if len(characteristics) == 0 {
				t.Error("Expected characteristics to be recorded")
			}
		}
	})

	t.Run("NoSaaSIndicators", func(t *testing.T) {
		// Create a simple scenario without SaaS indicators
		tempDir, err := os.MkdirTemp("", "saas-simple")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		scenarioDir := filepath.Join(tempDir, "simple-script")
		os.MkdirAll(scenarioDir, 0755)

		// Create minimal service.json
		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"displayName": "Simple Script",
				"description": "A simple utility script",
			},
		}

		serviceJSON, _ := json.MarshalIndent(serviceConfig, "", "  ")
		os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), serviceJSON, 0644)

		detectionService := &SaaSDetectionService{
			scenariosPath: tempDir,
		}

		isSaaS, scenario := detectionService.analyzeSaaSCharacteristics("simple-script", tempDir)

		if isSaaS {
			t.Errorf("Expected simple-script to NOT be classified as SaaS (score: %f)", scenario.ConfidenceScore)
		}
	})
}

// TestClaudeCodeServiceWithoutDB tests Claude Code service without database
func TestClaudeCodeServiceWithoutDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NewClaudeCodeService", func(t *testing.T) {
		// Test with custom binary
		service1 := NewClaudeCodeService("/custom/path/claude-code")
		if service1.claudeCodeBinary != "/custom/path/claude-code" {
			t.Errorf("Expected binary path '/custom/path/claude-code', got '%s'", service1.claudeCodeBinary)
		}

		// Test with empty binary (should use default)
		service2 := NewClaudeCodeService("")
		if service2.claudeCodeBinary != "resource-claude-code" {
			t.Errorf("Expected default binary 'resource-claude-code', got '%s'", service2.claudeCodeBinary)
		}
	})

	t.Run("DirectDeployPathValidation", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "deploy-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		service := NewClaudeCodeService("")

		// Test path traversal prevention
		err = service.directDeploy("../../../etc", "test-page", false)
		if err == nil {
			t.Error("Expected error for path traversal attempt")
		}

		err2 := service.directDeploy("..\\..\\windows", "test-page", false)
		if err2 == nil {
			t.Error("Expected error for Windows path traversal attempt")
		}
	})
}

// TestLandingPageServiceWithoutDB tests landing page service logic
func TestLandingPageServiceWithoutDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NewLandingPageService", func(t *testing.T) {
		service := NewLandingPageService(nil, "/tmp/templates")

		if service.templatesPath != "/tmp/templates" {
			t.Errorf("Expected templates path '/tmp/templates', got '%s'", service.templatesPath)
		}
	})
}

// TestDatabaseServiceConstructor tests service constructors
func TestDatabaseServiceConstructor(t *testing.T) {
	t.Run("NewDatabaseService", func(t *testing.T) {
		service := NewDatabaseService(nil)
		if service == nil {
			t.Error("Expected service to be created")
		}
	})

	t.Run("NewSaaSDetectionService", func(t *testing.T) {
		service := NewSaaSDetectionService("/tmp/scenarios", nil)
		if service.scenariosPath != "/tmp/scenarios" {
			t.Errorf("Expected scenarios path '/tmp/scenarios', got '%s'", service.scenariosPath)
		}
	})
}

// TestErrorHandling tests error handling paths
func TestErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ScanNonExistentDirectory", func(t *testing.T) {
		detectionService := &SaaSDetectionService{
			scenariosPath: "/nonexistent/path",
			dbService:     nil,
		}

		_, err := detectionService.ScanScenarios(false, "")
		if err == nil {
			t.Error("Expected error when scanning nonexistent directory")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		var req ScanRequest
		err := json.Unmarshal([]byte("{invalid json"), &req)
		if err == nil {
			t.Error("Expected error when unmarshaling invalid JSON")
		}
	})
}
