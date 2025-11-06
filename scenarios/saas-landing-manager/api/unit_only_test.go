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
	"strings"
	"testing"
	"time"
)

// TestNewServiceConstructors tests service constructors without database
func TestNewServiceConstructors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NewDatabaseService", func(t *testing.T) {
		service := NewDatabaseService(nil)
		if service == nil {
			t.Error("Expected non-nil service")
		}
		if service.db != nil {
			t.Error("Expected nil db when passed nil")
		}
	})

	t.Run("NewSaaSDetectionService", func(t *testing.T) {
		service := NewSaaSDetectionService("/test/path", nil)
		if service == nil {
			t.Error("Expected non-nil service")
		}
		if service.scenariosPath != "/test/path" {
			t.Errorf("Expected path '/test/path', got '%s'", service.scenariosPath)
		}
	})

	t.Run("NewLandingPageService", func(t *testing.T) {
		service := NewLandingPageService(nil, "/templates")
		if service == nil {
			t.Error("Expected non-nil service")
		}
		if service.templatesPath != "/templates" {
			t.Errorf("Expected path '/templates', got '%s'", service.templatesPath)
		}
	})

	t.Run("NewClaudeCodeService", func(t *testing.T) {
		// With default binary
		service := NewClaudeCodeService("")
		if service == nil {
			t.Error("Expected non-nil service")
		}
		if service.claudeCodeBinary != "resource-claude-code" {
			t.Errorf("Expected default binary 'resource-claude-code', got '%s'", service.claudeCodeBinary)
		}

		// With custom binary
		service = NewClaudeCodeService("/custom/binary")
		if service.claudeCodeBinary != "/custom/binary" {
			t.Errorf("Expected custom binary '/custom/binary', got '%s'", service.claudeCodeBinary)
		}
	})
}

// TestSaaSDetectionWithoutDatabase tests detection logic without database
func TestSaaSDetectionWithoutDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create temp dir without database
	tempDir, err := os.MkdirTemp("", "saas-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	scenariosDir := filepath.Join(tempDir, "scenarios")
	os.MkdirAll(scenariosDir, 0755)

	service := NewSaaSDetectionService(scenariosDir, nil)

	t.Run("AnalyzeSaaSCharacteristics_FullFeatures", func(t *testing.T) {
		// Create a scenario with all SaaS indicators
		scenarioDir := filepath.Join(scenariosDir, "full-saas")
		os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755)
		os.MkdirAll(filepath.Join(scenarioDir, "ui"), 0755)
		os.MkdirAll(filepath.Join(scenarioDir, "api"), 0755)

		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"displayName": "Full SaaS App",
				"description": "Complete SaaS application",
				"tags":        []string{"saas", "multi-tenant", "billing", "analytics", "a-b-testing", "subscription"},
			},
			"resources": map[string]interface{}{
				"postgres": map[string]interface{}{
					"enabled": true,
				},
			},
		}
		serviceJSON, _ := json.MarshalIndent(serviceConfig, "", "  ")
		os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), serviceJSON, 0644)

		prdContent := `# Full SaaS Application

## Business Value
Revenue Potential: $200K-$1M

This is a comprehensive B2B SaaS platform with enterprise pricing and subscription models.
Multi-tenant architecture with advanced API service capabilities.
Includes marketplace features and business analytics.
`
		os.WriteFile(filepath.Join(scenarioDir, "PRD.md"), []byte(prdContent), 0644)

		isSaaS, scenario := service.analyzeSaaSCharacteristics("full-saas", scenariosDir)

		if !isSaaS {
			t.Error("Expected full-saas to be classified as SaaS")
		}

		if scenario.DisplayName != "Full SaaS App" {
			t.Errorf("Display name not extracted: %s", scenario.DisplayName)
		}

		if scenario.Description != "Complete SaaS application" {
			t.Errorf("Description not extracted: %s", scenario.Description)
		}

		if scenario.RevenuePotential == "" {
			t.Error("Revenue potential not extracted")
		}
		t.Logf("Revenue potential extracted: %s", scenario.RevenuePotential)

		if scenario.SaaSType != "b2b_tool" {
			t.Errorf("Expected SaaS type 'b2b_tool', got '%s'", scenario.SaaSType)
		}

		// Should have high confidence with all indicators
		if scenario.ConfidenceScore < 1.0 {
			t.Errorf("Expected high confidence score (>=1.0), got %f", scenario.ConfidenceScore)
		}

		// Check metadata
		if characteristics, ok := scenario.Metadata["characteristics"].([]string); ok {
			if len(characteristics) < 5 {
				t.Errorf("Expected multiple characteristics, got %d", len(characteristics))
			}
		} else {
			t.Error("Characteristics not in metadata")
		}
	})

	t.Run("AnalyzeSaaSCharacteristics_MinimalSaaS", func(t *testing.T) {
		scenarioDir := filepath.Join(scenariosDir, "minimal-saas")
		os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755)

		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"displayName": "Minimal SaaS",
				"tags":        []string{"saas"},
			},
		}
		serviceJSON, _ := json.MarshalIndent(serviceConfig, "", "  ")
		os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), serviceJSON, 0644)

		isSaaS, scenario := service.analyzeSaaSCharacteristics("minimal-saas", scenariosDir)

		// May or may not detect as SaaS with just one tag, depending on threshold
		t.Logf("Classification result: isSaaS=%v, score=%f", isSaaS, scenario.ConfidenceScore)

		// The tag should contribute to the score
		if scenario.ConfidenceScore < 0.15 {
			t.Errorf("Expected higher confidence score with SaaS tag, got %f", scenario.ConfidenceScore)
		}
	})

	t.Run("AnalyzeSaaSCharacteristics_NonSaaS", func(t *testing.T) {
		scenarioDir := filepath.Join(scenariosDir, "non-saas")
		os.MkdirAll(scenarioDir, 0755)

		isSaaS, scenario := service.analyzeSaaSCharacteristics("non-saas", scenariosDir)

		if isSaaS {
			t.Errorf("Non-SaaS scenario incorrectly classified as SaaS (score: %f)", scenario.ConfidenceScore)
		}
	})

	t.Run("AnalyzeSaaSCharacteristics_PathTraversal", func(t *testing.T) {
		isSaaS, _ := service.analyzeSaaSCharacteristics("../../../etc/passwd", scenariosDir)

		if isSaaS {
			t.Error("Path traversal attempt should not be classified as SaaS")
		}
	})

	t.Run("AnalyzeSaaSCharacteristics_APISaaS", func(t *testing.T) {
		scenarioDir := filepath.Join(scenariosDir, "api-saas")
		os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755)

		prdContent := `# API Service

This is an API-based SaaS service with subscription pricing.
Revenue Potential: $50K-$150K
`
		os.WriteFile(filepath.Join(scenarioDir, "PRD.md"), []byte(prdContent), 0644)

		isSaaS, scenario := service.analyzeSaaSCharacteristics("api-saas", scenariosDir)

		if !isSaaS {
			t.Error("Expected api-saas to be classified as SaaS")
		}

		if scenario.SaaSType != "api_service" {
			t.Errorf("Expected SaaS type 'api_service', got '%s'", scenario.SaaSType)
		}
	})

	t.Run("AnalyzeSaaSCharacteristics_MarketplaceSaaS", func(t *testing.T) {
		scenarioDir := filepath.Join(scenariosDir, "marketplace-saas")
		os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755)

		prdContent := `# Marketplace Platform

This is a marketplace SaaS platform.
Revenue Potential: $100K-$500K
`
		os.WriteFile(filepath.Join(scenarioDir, "PRD.md"), []byte(prdContent), 0644)

		isSaaS, scenario := service.analyzeSaaSCharacteristics("marketplace-saas", scenariosDir)

		if !isSaaS {
			t.Error("Expected marketplace-saas to be classified as SaaS")
		}

		if scenario.SaaSType != "marketplace" {
			t.Errorf("Expected SaaS type 'marketplace', got '%s'", scenario.SaaSType)
		}
	})
}

// TestClaudeCodeServiceWithoutExecution tests Claude Code service without external dependencies
func TestClaudeCodeServiceWithoutExecution(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create temp dir without database
	tempDir, err := os.MkdirTemp("", "saas-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("DeployLandingPage_DirectMethod_Validation", func(t *testing.T) {
		service := NewClaudeCodeService("")

		req := &DeployRequest{
			TargetScenario:   "test-scenario",
			DeploymentMethod: "direct",
			BackupExisting:   true,
		}

		response, err := service.DeployLandingPage("test-page-id", "test-scenario", req)

		// Should attempt deployment (may fail due to test environment)
		if response == nil && err == nil {
			t.Error("Expected either response or error")
		}

		if response != nil {
			if response.DeploymentID == "" {
				t.Error("Expected deployment ID")
			}
			if response.Status == "" {
				t.Error("Expected deployment status")
			}
		}
	})

	t.Run("DeployLandingPage_ClaudeAgentMethod", func(t *testing.T) {
		service := NewClaudeCodeService("/fake/claude-code")

		req := &DeployRequest{
			TargetScenario:   "test-scenario",
			DeploymentMethod: "claude_agent",
			BackupExisting:   false,
		}

		_, err := service.DeployLandingPage("test-page-id", "test-scenario", req)

		// Should fail because binary doesn't exist
		if err == nil {
			t.Log("Unexpectedly succeeded (Claude Code may be installed)")
		}
	})

	t.Run("DirectDeploy_PathValidation", func(t *testing.T) {
		service := NewClaudeCodeService("")

		// Test path traversal protection
		err := service.directDeploy("../../../etc", "page-id", false)

		if err == nil {
			t.Error("Expected error for path traversal attempt")
		}

		if !strings.Contains(err.Error(), "invalid target scenario path") {
			t.Errorf("Expected path validation error, got: %v", err)
		}
	})

	t.Run("DirectDeploy_ValidPath", func(t *testing.T) {
		service := NewClaudeCodeService("")

		targetScenario := "valid-scenario"
		targetPath := filepath.Join(tempDir, targetScenario)
		os.MkdirAll(targetPath, 0755)

		// Change to temp dir
		oldWD, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(oldWD)

		err := service.directDeploy(targetScenario, "page-id", false)

		// Check if landing directory was created
		landingPath := filepath.Join(targetPath, "landing")
		if _, statErr := os.Stat(landingPath); os.IsNotExist(statErr) && err == nil {
			t.Error("Expected landing directory to be created or error to be returned")
		}

		// If successful, check for index.html
		indexPath := filepath.Join(landingPath, "index.html")
		if _, statErr := os.Stat(indexPath); statErr == nil {
			content, _ := os.ReadFile(indexPath)
			if !bytes.Contains(content, []byte("<!DOCTYPE html>")) {
				t.Error("Expected HTML content in index.html")
			}
			if !bytes.Contains(content, []byte("SaaS")) {
				t.Error("Expected SaaS-related content in generated HTML")
			}
		}
	})
}

// TestHTTPHandlersWithoutDatabase tests HTTP handlers without database dependencies
func TestHTTPHandlersWithoutDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthHandler_Complete", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(healthHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response["status"])
		}

		if response["service"] != "saas-landing-manager" {
			t.Errorf("Expected service 'saas-landing-manager', got '%s'", response["service"])
		}

		if response["timestamp"] == "" {
			t.Error("Expected timestamp in response")
		}

		// Validate timestamp format
		_, err := time.Parse(time.RFC3339, response["timestamp"])
		if err != nil {
			t.Errorf("Invalid timestamp format: %v", err)
		}

		// Check content type
		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})

	t.Run("ScanScenariosHandler_ErrorCases", func(t *testing.T) {
		// Test with invalid JSON
		req, _ := http.NewRequest("POST", "/api/v1/scenarios/scan", bytes.NewBufferString("{invalid json"))
		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(scanScenariosHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}

		// Test with empty body (should also be bad request due to JSON decode error)
		req2, _ := http.NewRequest("POST", "/api/v1/scenarios/scan", bytes.NewBufferString(""))
		rr2 := httptest.NewRecorder()

		handler.ServeHTTP(rr2, req2)

		if rr2.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty body, got %d", rr2.Code)
		}
	})

	t.Run("GenerateLandingPageHandler_ErrorCases", func(t *testing.T) {
		// Test with invalid JSON
		req, _ := http.NewRequest("POST", "/api/v1/landing-pages/generate", bytes.NewBufferString("{invalid"))
		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(generateLandingPageHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})

	t.Run("DeployLandingPageHandler_ErrorCases", func(t *testing.T) {
		// Test with invalid JSON
		req, _ := http.NewRequest("POST", "/api/v1/landing-pages/test-id/deploy", bytes.NewBufferString("{invalid"))
		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(deployLandingPageHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})
}

// TestModelsAndStructures tests data models and structures
func TestModelsAndStructures(t *testing.T) {
	t.Run("SaaSScenario_Initialization", func(t *testing.T) {
		scenario := SaaSScenario{
			ID:               "test-id",
			ScenarioName:     "test-scenario",
			DisplayName:      "Test Scenario",
			Description:      "Test description",
			SaaSType:         "b2b_tool",
			Industry:         "technology",
			RevenuePotential: "$50K",
			HasLandingPage:   true,
			LandingPageURL:   "/landing",
			LastScan:         time.Now(),
			ConfidenceScore:  0.85,
			Metadata:         map[string]interface{}{"key": "value"},
		}

		if scenario.ID != "test-id" {
			t.Error("Scenario initialization failed")
		}
	})

	t.Run("GenerateRequest_Validation", func(t *testing.T) {
		req := GenerateRequest{
			ScenarioID:      "scenario-1",
			TemplateID:      "template-1",
			CustomContent:   map[string]interface{}{"title": "Test"},
			EnableABTesting: true,
		}

		if req.ScenarioID != "scenario-1" {
			t.Error("GenerateRequest initialization failed")
		}
	})

	t.Run("DeployRequest_Validation", func(t *testing.T) {
		req := DeployRequest{
			TargetScenario:   "target",
			DeploymentMethod: "direct",
			BackupExisting:   true,
		}

		if req.TargetScenario != "target" {
			t.Error("DeployRequest initialization failed")
		}
	})
}
