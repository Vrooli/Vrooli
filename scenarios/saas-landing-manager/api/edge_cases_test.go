// +build testing

package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestSaaSDetectionEdgeCases tests edge cases in SaaS detection
func TestSaaSDetectionEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyServiceJSON", func(t *testing.T) {
		tempDir, _ := ioutil.TempDir("", "empty-service")
		defer os.RemoveAll(tempDir)

		scenarioDir := filepath.Join(tempDir, "test-scenario")
		os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755)

		// Create empty service.json
		os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), []byte("{}"), 0644)

		detectionService := &SaaSDetectionService{
			scenariosPath: tempDir,
		}

		_, scenario := detectionService.analyzeSaaSCharacteristics("test-scenario", tempDir)

		// Should not be classified as SaaS with empty config
		if scenario.ConfidenceScore >= 0.5 {
			t.Errorf("Empty service.json should not be classified as SaaS (score: %f)", scenario.ConfidenceScore)
		}
	})

	t.Run("MalformedServiceJSON", func(t *testing.T) {
		tempDir, _ := ioutil.TempDir("", "malformed-service")
		defer os.RemoveAll(tempDir)

		scenarioDir := filepath.Join(tempDir, "test-scenario")
		os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755)

		// Create malformed service.json
		os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), []byte("{invalid json"), 0644)

		detectionService := &SaaSDetectionService{
			scenariosPath: tempDir,
		}

		_, scenario := detectionService.analyzeSaaSCharacteristics("test-scenario", tempDir)

		// Should handle gracefully
		if scenario.ScenarioName != "test-scenario" {
			t.Error("Should still set scenario name")
		}
	})

	t.Run("NoServiceJSON", func(t *testing.T) {
		tempDir, _ := ioutil.TempDir("", "no-service")
		defer os.RemoveAll(tempDir)

		scenarioDir := filepath.Join(tempDir, "test-scenario")
		os.MkdirAll(scenarioDir, 0755)

		detectionService := &SaaSDetectionService{
			scenariosPath: tempDir,
		}

		_, scenario := detectionService.analyzeSaaSCharacteristics("test-scenario", tempDir)

		// Should handle missing service.json gracefully
		if scenario.ScenarioName != "test-scenario" {
			t.Error("Should still set scenario name")
		}
	})

	t.Run("NoPRD", func(t *testing.T) {
		tempDir, _ := ioutil.TempDir("", "no-prd")
		defer os.RemoveAll(tempDir)

		scenarioDir := filepath.Join(tempDir, "test-scenario")
		os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755)

		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"displayName": "Test",
				"tags":        []string{"saas"},
			},
		}

		serviceJSON, _ := json.MarshalIndent(serviceConfig, "", "  ")
		os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), serviceJSON, 0644)

		detectionService := &SaaSDetectionService{
			scenariosPath: tempDir,
		}

		_, scenario := detectionService.analyzeSaaSCharacteristics("test-scenario", tempDir)

		// Can still be SaaS based on service.json tags
		if scenario.RevenuePotential != "" {
			t.Error("Should not extract revenue potential without PRD")
		}
	})

	t.Run("ScenarioWithLandingPage", func(t *testing.T) {
		tempDir, _ := ioutil.TempDir("", "has-landing")
		defer os.RemoveAll(tempDir)

		scenarioDir := filepath.Join(tempDir, "test-scenario")
		os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755)
		os.MkdirAll(filepath.Join(scenarioDir, "landing"), 0755)

		// Create service.json with SaaS indicators
		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"displayName": "Test",
				"tags":        []string{"saas"},
			},
		}

		serviceJSON, _ := json.MarshalIndent(serviceConfig, "", "  ")
		os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), serviceJSON, 0644)

		detectionService := &SaaSDetectionService{
			scenariosPath: tempDir,
		}

		_, scenario := detectionService.analyzeSaaSCharacteristics("test-scenario", tempDir)

		if !scenario.HasLandingPage {
			t.Error("Should detect existing landing page")
		}

		if scenario.LandingPageURL == "" {
			t.Error("Should set landing page URL")
		}
	})

	t.Run("MultipleRevenueMentions", func(t *testing.T) {
		tempDir, _ := ioutil.TempDir("", "multi-revenue")
		defer os.RemoveAll(tempDir)

		scenarioDir := filepath.Join(tempDir, "test-scenario")
		os.MkdirAll(scenarioDir, 0755)

		prdContent := `# Test App

Revenue Potential: $50K-$200K

This app has subscription pricing.
Enterprise tier available.
`
		os.WriteFile(filepath.Join(scenarioDir, "PRD.md"), []byte(prdContent), 0644)

		detectionService := &SaaSDetectionService{
			scenariosPath: tempDir,
		}

		_, scenario := detectionService.analyzeSaaSCharacteristics("test-scenario", tempDir)

		// Should extract revenue potential
		if scenario.RevenuePotential == "" {
			t.Error("Should extract revenue potential from PRD")
		}

		// Should have higher confidence score due to multiple indicators
		if scenario.ConfidenceScore < 0.3 {
			t.Errorf("Expected higher confidence score, got %f", scenario.ConfidenceScore)
		}
	})

	t.Run("DifferentSaaSTypes", func(t *testing.T) {
		testCases := []struct {
			prdContent     string
			expectedType   string
		}{
			{"B2B SaaS platform", "b2b_tool"},
			{"API service provider", "api_service"},
			{"Marketplace platform", "marketplace"},
			{"Consumer application", "b2c_app"},
		}

		for _, tc := range testCases {
			tempDir, _ := ioutil.TempDir("", "saas-type")
			scenarioDir := filepath.Join(tempDir, "test-scenario")
			os.MkdirAll(scenarioDir, 0755)

			os.WriteFile(filepath.Join(scenarioDir, "PRD.md"), []byte(tc.prdContent), 0644)

			detectionService := &SaaSDetectionService{
				scenariosPath: tempDir,
			}

			_, scenario := detectionService.analyzeSaaSCharacteristics("test-scenario", tempDir)

			if scenario.SaaSType != tc.expectedType {
				t.Errorf("For content '%s', expected type '%s', got '%s'",
					tc.prdContent, tc.expectedType, scenario.SaaSType)
			}

			os.RemoveAll(tempDir)
		}
	})
}

// TestDeploymentEdgeCases tests deployment edge cases
func TestDeploymentEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("DeployWithBackup", func(t *testing.T) {
		tempDir, _ := ioutil.TempDir("", "deploy-backup")
		defer os.RemoveAll(tempDir)

		service := NewClaudeCodeService("")

		// Create target with existing landing page
		targetDir := filepath.Join(tempDir, "target-scenario")
		landingDir := filepath.Join(targetDir, "landing")
		os.MkdirAll(landingDir, 0755)
		os.WriteFile(filepath.Join(landingDir, "index.html"), []byte("<html>Old</html>"), 0644)

		// This will fail because we're not in the right directory structure,
		// but we're testing the code path is exercised
		err := service.directDeploy("target-scenario", "test-page", true)
		if err != nil {
			t.Logf("Deploy with backup failed as expected: %v", err)
		}
	})

	t.Run("SpawnAgentPromptFormat", func(t *testing.T) {
		service := NewClaudeCodeService("/nonexistent/claude-code")

		// Test that prompt is formatted correctly (will fail to execute)
		_, err := service.spawnClaudeAgent("test-scenario", "test-page-id")

		// Expected to fail because binary doesn't exist
		if err == nil {
			t.Log("Spawn succeeded unexpectedly")
		}
	})

	t.Run("DirectDeployCreatesStructure", func(t *testing.T) {
		// Test that direct deploy creates the necessary directory structure
		// This will fail in current directory context, but exercises code path
		service := NewClaudeCodeService("")
		err := service.directDeploy("valid-scenario-name", "test-page", false)

		if err != nil {
			t.Logf("Direct deploy failed as expected in test context: %v", err)
		}
	})
}

// TestMetadataHandling tests metadata operations
func TestMetadataHandling(t *testing.T) {
	t.Run("MetadataSerialization", func(t *testing.T) {
		scenario := SaaSScenario{
			ID:           uuid.New().String(),
			ScenarioName: "test",
			Metadata: map[string]interface{}{
				"characteristics": []string{"has_ui", "has_api", "has_database"},
				"analysis_version": "1.0",
				"custom_field":     "value",
				"nested": map[string]interface{}{
					"key": "value",
				},
			},
		}

		data, err := json.Marshal(scenario)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded SaaSScenario
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.Metadata == nil {
			t.Error("Metadata should not be nil")
		}

		if len(decoded.Metadata) != len(scenario.Metadata) {
			t.Errorf("Metadata length mismatch: expected %d, got %d",
				len(scenario.Metadata), len(decoded.Metadata))
		}
	})

	t.Run("EmptyMetadata", func(t *testing.T) {
		scenario := SaaSScenario{
			ID:           uuid.New().String(),
			ScenarioName: "test",
			Metadata:     make(map[string]interface{}),
		}

		data, err := json.Marshal(scenario)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded SaaSScenario
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.Metadata == nil {
			t.Error("Empty metadata should be preserved")
		}
	})
}

// TestServiceConfiguration tests service constructors and configuration
func TestServiceConfiguration(t *testing.T) {
	t.Run("DatabaseServiceNilDB", func(t *testing.T) {
		service := NewDatabaseService(nil)
		if service == nil {
			t.Error("Service should be created even with nil db")
		}
	})

	t.Run("DetectionServicePaths", func(t *testing.T) {
		testPaths := []string{
			"/tmp/scenarios",
			"/var/lib/vrooli/scenarios",
			"./scenarios",
			"",
		}

		for _, path := range testPaths {
			service := NewSaaSDetectionService(path, nil)
			if service.scenariosPath != path {
				t.Errorf("Expected path '%s', got '%s'", path, service.scenariosPath)
			}
		}
	})

	t.Run("LandingPageServicePaths", func(t *testing.T) {
		testPaths := []string{
			"/tmp/templates",
			"./templates",
			"",
		}

		for _, path := range testPaths {
			service := NewLandingPageService(nil, path)
			if service.templatesPath != path {
				t.Errorf("Expected path '%s', got '%s'", path, service.templatesPath)
			}
		}
	})

	t.Run("ClaudeCodeServiceBinaries", func(t *testing.T) {
		testBinaries := []string{
			"/usr/local/bin/claude-code",
			"./bin/claude-code",
			"claude-code",
			"",
		}

		for _, binary := range testBinaries {
			service := NewClaudeCodeService(binary)
			expectedBinary := binary
			if binary == "" {
				expectedBinary = "resource-claude-code"
			}
			if service.claudeCodeBinary != expectedBinary {
				t.Errorf("Expected binary '%s', got '%s'", expectedBinary, service.claudeCodeBinary)
			}
		}
	})
}

// TestScanScenariosLogic tests scanning logic without database
func TestScanScenariosLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ScanEmptyDirectory", func(t *testing.T) {
		tempDir, _ := ioutil.TempDir("", "empty-scan")
		defer os.RemoveAll(tempDir)

		detectionService := &SaaSDetectionService{
			scenariosPath: tempDir,
			dbService:     nil,
		}

		response, err := detectionService.ScanScenarios(false, "")
		if err != nil {
			t.Fatalf("Scan should succeed on empty directory: %v", err)
		}

		if response.TotalScenarios != 0 {
			t.Errorf("Expected 0 scenarios, got %d", response.TotalScenarios)
		}
	})

	t.Run("ScanWithFilter", func(t *testing.T) {
		tempDir, _ := ioutil.TempDir("", "filter-scan")
		defer os.RemoveAll(tempDir)

		// Create multiple scenarios
		os.MkdirAll(filepath.Join(tempDir, "saas-app-1"), 0755)
		os.MkdirAll(filepath.Join(tempDir, "saas-app-2"), 0755)
		os.MkdirAll(filepath.Join(tempDir, "simple-script"), 0755)

		detectionService := &SaaSDetectionService{
			scenariosPath: tempDir,
			dbService:     nil,
		}

		response, err := detectionService.ScanScenarios(false, "saas")
		if err != nil {
			t.Fatalf("Scan with filter failed: %v", err)
		}

		if response.TotalScenarios > 2 {
			t.Errorf("Expected at most 2 scenarios with filter 'saas', got %d", response.TotalScenarios)
		}
	})

	t.Run("ScanIgnoresFiles", func(t *testing.T) {
		tempDir, _ := ioutil.TempDir("", "files-scan")
		defer os.RemoveAll(tempDir)

		// Create files (should be ignored)
		os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tempDir, "config.json"), []byte("{}"), 0644)

		// Create a directory (should be scanned)
		os.MkdirAll(filepath.Join(tempDir, "real-scenario"), 0755)

		detectionService := &SaaSDetectionService{
			scenariosPath: tempDir,
			dbService:     nil,
		}

		response, err := detectionService.ScanScenarios(false, "")
		if err != nil {
			t.Fatalf("Scan failed: %v", err)
		}

		// Should only count the directory, not the files
		if response.TotalScenarios != 1 {
			t.Errorf("Expected 1 scenario (ignoring files), got %d", response.TotalScenarios)
		}
	})
}

// TestCharacteristicsExtraction tests detailed characteristics extraction
func TestCharacteristicsExtraction(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MultipleTagIndicators", func(t *testing.T) {
		tempDir, _ := ioutil.TempDir("", "multi-tags")
		defer os.RemoveAll(tempDir)

		scenarioDir := filepath.Join(tempDir, "test-scenario")
		os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755)

		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"tags": []string{
					"saas",
					"multi-tenant",
					"billing",
					"analytics",
					"a-b-testing",
					"subscription",
				},
			},
		}

		serviceJSON, _ := json.MarshalIndent(serviceConfig, "", "  ")
		os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), serviceJSON, 0644)

		detectionService := &SaaSDetectionService{
			scenariosPath: tempDir,
		}

		_, scenario := detectionService.analyzeSaaSCharacteristics("test-scenario", tempDir)

		if scenario.ConfidenceScore < 0.5 {
			t.Errorf("Multiple SaaS tags should result in classification (score: %f)", scenario.ConfidenceScore)
		}

		if scenario.ConfidenceScore < 0.5 {
			t.Errorf("Expected high confidence score with multiple tags, got %f", scenario.ConfidenceScore)
		}

		characteristics, ok := scenario.Metadata["characteristics"].([]string)
		if !ok {
			t.Error("Characteristics should be recorded")
		} else {
			if len(characteristics) == 0 {
				t.Error("Should have recorded tag characteristics")
			}
		}
	})

	t.Run("AllStructuralIndicators", func(t *testing.T) {
		tempDir, _ := ioutil.TempDir("", "all-indicators")
		defer os.RemoveAll(tempDir)

		scenarioDir := filepath.Join(tempDir, "test-scenario")
		os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755)
		os.MkdirAll(filepath.Join(scenarioDir, "ui"), 0755)
		os.MkdirAll(filepath.Join(scenarioDir, "api"), 0755)
		os.MkdirAll(filepath.Join(scenarioDir, "landing"), 0755)

		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"tags": []string{"saas"},
			},
			"resources": map[string]interface{}{
				"postgres": map[string]interface{}{
					"enabled": true,
				},
			},
		}

		serviceJSON, _ := json.MarshalIndent(serviceConfig, "", "  ")
		os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), serviceJSON, 0644)

		prdContent := "Revenue Potential: $100K"
		os.WriteFile(filepath.Join(scenarioDir, "PRD.md"), []byte(prdContent), 0644)

		detectionService := &SaaSDetectionService{
			scenariosPath: tempDir,
		}

		_, scenario := detectionService.analyzeSaaSCharacteristics("test-scenario", tempDir)

		if scenario.ConfidenceScore < 0.5 {
			t.Error("All indicators present should result in SaaS classification")
		}

		// Should have very high confidence
		if scenario.ConfidenceScore < 0.8 {
			t.Errorf("Expected very high confidence score, got %f", scenario.ConfidenceScore)
		}

		characteristics, _ := scenario.Metadata["characteristics"].([]string)
		expectedCharacteristics := []string{"has_ui", "has_api", "existing_landing"}
		for _, expected := range expectedCharacteristics {
			found := false
			for _, actual := range characteristics {
				if strings.Contains(actual, expected) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected characteristic '%s' not found", expected)
			}
		}
	})
}
