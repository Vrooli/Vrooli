package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// [REQ:TMPL-PREVIEW-LINKS]
// NOTE: GetPreviewLinks now uses `vrooli scenario port` command which requires a running scenario.
// These unit tests are skipped in favor of integration tests in test/playbooks/ and api/integration_test.go
func TestGetPreviewLinks_Success(t *testing.T) {
	t.Skip("Requires running scenarios - tested via integration tests")
	tmpDir := t.TempDir()
	t.Setenv("GEN_OUTPUT_DIR", tmpDir)

	ts := &TemplateService{templatesDir: t.TempDir()}

	t.Run("REQ:TMPL-PREVIEW-LINKS/success-get-preview-links", func(t *testing.T) {
		// Create a mock generated scenario with service.json
		scenarioID := "test-preview-landing"
		scenarioPath := filepath.Join(tmpDir, scenarioID)
		if err := os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		// Write service.json with UI_PORT
		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"name":        scenarioID,
				"displayName": "Test Preview Landing",
			},
			"ports": map[string]interface{}{
				"API_PORT": 15000,
				"UI_PORT":  38000,
			},
		}
		svcData, _ := json.Marshal(serviceConfig)
		if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), svcData, 0o644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		// Get preview links
		result, err := ts.GetPreviewLinks(scenarioID)
		if err != nil {
			t.Fatalf("GetPreviewLinks() returned error: %v", err)
		}

		// Verify scenario_id
		if sid, ok := result["scenario_id"].(string); !ok || sid != scenarioID {
			t.Errorf("Expected scenario_id '%s', got '%v'", scenarioID, result["scenario_id"])
		}

		// Verify path
		if path, ok := result["path"].(string); !ok || path != scenarioPath {
			t.Errorf("Expected path '%s', got '%v'", scenarioPath, result["path"])
		}

		// Verify base_url
		expectedBaseURL := "http://localhost:38000"
		if baseURL, ok := result["base_url"].(string); !ok || baseURL != expectedBaseURL {
			t.Errorf("Expected base_url '%s', got '%v'", expectedBaseURL, result["base_url"])
		}

		// Verify links object
		links, ok := result["links"].(map[string]string)
		if !ok {
			t.Fatal("Expected links to be a map[string]string")
		}

		expectedLinks := map[string]string{
			"public":      "http://localhost:38000/",
			"admin":       "http://localhost:38000/admin",
			"admin_login": "http://localhost:38000/admin/login",
			"health":      "http://localhost:38000/health",
		}

		for key, expectedURL := range expectedLinks {
			if actualURL, ok := links[key]; !ok || actualURL != expectedURL {
				t.Errorf("Expected links['%s'] = '%s', got '%s'", key, expectedURL, actualURL)
			}
		}

		// Verify instructions exist
		if instructions, ok := result["instructions"].([]string); !ok || len(instructions) == 0 {
			t.Error("Expected non-empty instructions array")
		}

		// Verify notes exist
		if notes, ok := result["notes"].(string); !ok || notes == "" {
			t.Error("Expected non-empty notes string")
		}
	})

	t.Run("REQ:TMPL-PREVIEW-LINKS/different-port", func(t *testing.T) {
		// Test with a different UI port
		scenarioID := "different-port-test"
		scenarioPath := filepath.Join(tmpDir, scenarioID)
		if err := os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"name": scenarioID,
			},
			"ports": map[string]interface{}{
				"UI_PORT": 42000,
			},
		}
		svcData, _ := json.Marshal(serviceConfig)
		if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), svcData, 0o644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		result, err := ts.GetPreviewLinks(scenarioID)
		if err != nil {
			t.Fatalf("GetPreviewLinks() returned error: %v", err)
		}

		// Verify base_url uses custom port
		expectedBaseURL := "http://localhost:42000"
		if baseURL, ok := result["base_url"].(string); !ok || baseURL != expectedBaseURL {
			t.Errorf("Expected base_url '%s', got '%v'", expectedBaseURL, result["base_url"])
		}

		// Verify links use custom port
		links, ok := result["links"].(map[string]string)
		if !ok {
			t.Fatal("Expected links to be a map[string]string")
		}

		if publicURL, ok := links["public"]; !ok || publicURL != "http://localhost:42000/" {
			t.Errorf("Expected public to use port 42000, got '%s'", publicURL)
		}
	})
}

// [REQ:TMPL-PREVIEW-LINKS]
// NOTE: GetPreviewLinks now uses `vrooli scenario port` command which requires a running scenario.
// These unit tests are skipped in favor of integration tests in test/playbooks/ and api/integration_test.go
func TestGetPreviewLinks_ErrorHandling(t *testing.T) {
	t.Skip("Requires running scenarios - tested via integration tests")
	tmpDir := t.TempDir()
	t.Setenv("GEN_OUTPUT_DIR", tmpDir)

	ts := &TemplateService{templatesDir: t.TempDir()}

	t.Run("scenario not found", func(t *testing.T) {
		_, err := ts.GetPreviewLinks("non-existent-scenario")
		if err == nil {
			t.Error("Expected error for non-existent scenario, got nil")
		}
	})

	t.Run("missing service.json", func(t *testing.T) {
		// Create a scenario directory without service.json
		scenarioID := "missing-service-json"
		scenarioPath := filepath.Join(tmpDir, scenarioID)
		if err := os.MkdirAll(scenarioPath, 0o755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		_, err := ts.GetPreviewLinks(scenarioID)
		if err == nil {
			t.Error("Expected error for missing service.json, got nil")
		}
	})

	t.Run("missing UI_PORT in service.json", func(t *testing.T) {
		// Create a scenario with service.json but no UI_PORT
		scenarioID := "missing-ui-port"
		scenarioPath := filepath.Join(tmpDir, scenarioID)
		if err := os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		// Write service.json without UI_PORT
		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"name": scenarioID,
			},
			"ports": map[string]interface{}{
				"API_PORT": 15000,
			},
		}
		svcData, _ := json.Marshal(serviceConfig)
		if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), svcData, 0o644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		_, err := ts.GetPreviewLinks(scenarioID)
		if err == nil {
			t.Error("Expected error for missing UI_PORT, got nil")
		}
	})

	t.Run("invalid service.json format", func(t *testing.T) {
		// Create a scenario with malformed service.json
		scenarioID := "invalid-service-json"
		scenarioPath := filepath.Join(tmpDir, scenarioID)
		if err := os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		// Write invalid JSON
		if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), []byte("not valid json"), 0o644); err != nil {
			t.Fatalf("Failed to write invalid service.json: %v", err)
		}

		_, err := ts.GetPreviewLinks(scenarioID)
		if err == nil {
			t.Error("Expected error for invalid service.json, got nil")
		}
	})

	t.Run("empty scenario directory", func(t *testing.T) {
		// Create empty scenario directory
		scenarioID := "empty-scenario"
		scenarioPath := filepath.Join(tmpDir, scenarioID)
		if err := os.MkdirAll(scenarioPath, 0o755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		_, err := ts.GetPreviewLinks(scenarioID)
		if err == nil {
			t.Error("Expected error for empty scenario directory, got nil")
		}
	})
}

// [REQ:TMPL-PREVIEW-LINKS]
// NOTE: GetPreviewLinks now uses `vrooli scenario port` command which requires a running scenario.
// These unit tests are skipped in favor of integration tests in test/playbooks/ and api/integration_test.go
func TestGetPreviewLinks_LinkFormat(t *testing.T) {
	t.Skip("Requires running scenarios - tested via integration tests")
	tmpDir := t.TempDir()
	t.Setenv("GEN_OUTPUT_DIR", tmpDir)

	ts := &TemplateService{templatesDir: t.TempDir()}

	t.Run("verify link format and completeness", func(t *testing.T) {
		scenarioID := "link-format-test"
		scenarioPath := filepath.Join(tmpDir, scenarioID)
		if err := os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"name": scenarioID,
			},
			"ports": map[string]interface{}{
				"UI_PORT": 35000,
			},
		}
		svcData, _ := json.Marshal(serviceConfig)
		if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), svcData, 0o644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		result, err := ts.GetPreviewLinks(scenarioID)
		if err != nil {
			t.Fatalf("GetPreviewLinks() returned error: %v", err)
		}

		links, ok := result["links"].(map[string]string)
		if !ok {
			t.Fatal("Expected links to be a map[string]string")
		}

		// Verify all required link keys are present
		requiredLinks := []string{"public", "admin", "admin_login", "health"}
		for _, key := range requiredLinks {
			if _, ok := links[key]; !ok {
				t.Errorf("Expected link key '%s' to be present", key)
			}
		}

		// Verify all links start with http://localhost:35000
		baseURL := "http://localhost:35000"
		for key, url := range links {
			if len(url) < len(baseURL) || url[:len(baseURL)] != baseURL {
				t.Errorf("Expected link '%s' to start with '%s', got '%s'", key, baseURL, url)
			}
		}

		// Verify specific path formats
		if links["public"] != "http://localhost:35000/" {
			t.Errorf("Public landing should be base URL with trailing slash")
		}
		if links["admin"] != "http://localhost:35000/admin" {
			t.Errorf("Admin portal should be base URL + /admin")
		}
		if links["admin_login"] != "http://localhost:35000/admin/login" {
			t.Errorf("Admin login should be base URL + /admin/login")
		}
		if links["health"] != "http://localhost:35000/health" {
			t.Errorf("Health check should be base URL + /health")
		}
	})
}
