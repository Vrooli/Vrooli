package generation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewAnalyzer(t *testing.T) {
	t.Run("with explicit root", func(t *testing.T) {
		analyzer := NewAnalyzer("/custom/root")
		if analyzer.vrooliRoot != "/custom/root" {
			t.Errorf("expected vrooliRoot '/custom/root', got %q", analyzer.vrooliRoot)
		}
	})

	t.Run("with empty root uses fallback", func(t *testing.T) {
		analyzer := NewAnalyzer("")
		if analyzer.vrooliRoot == "" {
			t.Errorf("expected vrooliRoot to be set from fallback")
		}
	})
}

func TestFormatDisplayName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-app", "My App"},
		{"hello-world", "Hello World"},
		{"simple", "Simple"},
		{"qr-code-generator", "QR Code Generator"},
		{"api-server", "API Server"},
		{"ui-framework", "UI Framework"},
		{"cli-tool", "CLI Tool"},
		{"ai-assistant", "AI Assistant"},
		{"ml-model", "ML Model"},
		{"nlp-processor", "NLP Processor"},
		{"prd-generator", "PRD Generator"},
		{"my-api-client", "My API Client"},
		{"", ""},
		{"a", "A"},
		{"a-b-c", "A B C"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := FormatDisplayName(tt.input)
			if result != tt.expected {
				t.Errorf("FormatDisplayName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAnalyzeScenario_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	analyzer := NewAnalyzer(tmpDir)

	_, err := analyzer.AnalyzeScenario("nonexistent-scenario")
	if err == nil {
		t.Errorf("expected error for nonexistent scenario")
	}
}

func TestAnalyzeScenario_BasicScenario(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioName := "test-scenario"
	scenarioPath := filepath.Join(tmpDir, "scenarios", scenarioName)

	// Create minimal scenario structure
	if err := os.MkdirAll(scenarioPath, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	analyzer := NewAnalyzer(tmpDir)
	metadata, err := analyzer.AnalyzeScenario(scenarioName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metadata.Name != scenarioName {
		t.Errorf("expected Name %q, got %q", scenarioName, metadata.Name)
	}
	if metadata.DisplayName != "Test Scenario" {
		t.Errorf("expected DisplayName 'Test Scenario', got %q", metadata.DisplayName)
	}
	if metadata.Version != "1.0.0" {
		t.Errorf("expected Version '1.0.0', got %q", metadata.Version)
	}
	if metadata.Author != "Vrooli Team" {
		t.Errorf("expected Author 'Vrooli Team', got %q", metadata.Author)
	}
	if metadata.HasUI {
		t.Errorf("expected HasUI false without UI build")
	}
}

func TestAnalyzeScenario_WithServiceJSON(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioName := "test-scenario"
	scenarioPath := filepath.Join(tmpDir, "scenarios", scenarioName)
	vrooliDir := filepath.Join(scenarioPath, ".vrooli")

	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	serviceJSON := ServiceJSON{}
	serviceJSON.Service.Name = scenarioName
	serviceJSON.Service.DisplayName = "Custom Display Name"
	serviceJSON.Service.Description = "A custom description"
	serviceJSON.Service.Version = "2.0.0"
	serviceJSON.Service.Category = "productivity"
	serviceJSON.Service.Tags = []string{"tag1", "tag2"}
	serviceJSON.Service.License = "Apache-2.0"
	serviceJSON.Service.Maintainers = []struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}{{Name: "John Doe", Email: "john@example.com"}}
	serviceJSON.Ports.API.Range = "4000-4010"
	serviceJSON.Ports.UI.Range = "5000-5010"

	data, _ := json.Marshal(serviceJSON)
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write service.json: %v", err)
	}

	analyzer := NewAnalyzer(tmpDir)
	metadata, err := analyzer.AnalyzeScenario(scenarioName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metadata.DisplayName != "Custom Display Name" {
		t.Errorf("expected DisplayName from service.json, got %q", metadata.DisplayName)
	}
	if metadata.Description != "A custom description" {
		t.Errorf("expected Description from service.json, got %q", metadata.Description)
	}
	if metadata.Version != "2.0.0" {
		t.Errorf("expected Version from service.json, got %q", metadata.Version)
	}
	if metadata.Author != "John Doe" {
		t.Errorf("expected Author from service.json maintainers, got %q", metadata.Author)
	}
	if metadata.License != "Apache-2.0" {
		t.Errorf("expected License from service.json, got %q", metadata.License)
	}
	if metadata.APIPort != 4000 {
		t.Errorf("expected APIPort 4000, got %d", metadata.APIPort)
	}
	if metadata.UIPort != 5000 {
		t.Errorf("expected UIPort 5000, got %d", metadata.UIPort)
	}
	if metadata.Category != "productivity" {
		t.Errorf("expected Category 'productivity', got %q", metadata.Category)
	}
	if len(metadata.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(metadata.Tags))
	}
}

func TestAnalyzeScenario_WithUIPackageJSON(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioName := "test-scenario"
	scenarioPath := filepath.Join(tmpDir, "scenarios", scenarioName)
	uiDir := filepath.Join(scenarioPath, "ui")

	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	pkgJSON := UIPackageJSON{
		Name:        scenarioName,
		Version:     "3.0.0",
		Description: "Description from package.json",
		Author:      "Package Author",
		License:     "GPL-3.0",
	}

	data, _ := json.Marshal(pkgJSON)
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	analyzer := NewAnalyzer(tmpDir)
	metadata, err := analyzer.AnalyzeScenario(scenarioName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// package.json values should be used for fallbacks when service.json is missing
	if metadata.Description != "Description from package.json" {
		t.Errorf("expected Description from package.json, got %q", metadata.Description)
	}
	if metadata.Version != "3.0.0" {
		t.Errorf("expected Version from package.json, got %q", metadata.Version)
	}
	if metadata.License != "GPL-3.0" {
		t.Errorf("expected License from package.json, got %q", metadata.License)
	}
}

func TestAnalyzeScenario_WithBuiltUI(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioName := "test-scenario"
	scenarioPath := filepath.Join(tmpDir, "scenarios", scenarioName)
	distDir := filepath.Join(scenarioPath, "ui", "dist")

	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatalf("failed to create dist dir: %v", err)
	}

	// Create index.html to indicate built UI
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("failed to write index.html: %v", err)
	}

	analyzer := NewAnalyzer(tmpDir)
	metadata, err := analyzer.AnalyzeScenario(scenarioName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !metadata.HasUI {
		t.Errorf("expected HasUI true with built UI")
	}
	if metadata.UIDistPath != distDir {
		t.Errorf("expected UIDistPath %q, got %q", distDir, metadata.UIDistPath)
	}
}

func TestValidateScenarioForDesktop_NoUI(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioName := "test-scenario"
	scenarioPath := filepath.Join(tmpDir, "scenarios", scenarioName)

	if err := os.MkdirAll(scenarioPath, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	analyzer := NewAnalyzer(tmpDir)
	err := analyzer.ValidateScenarioForDesktop(scenarioName)
	if err == nil {
		t.Errorf("expected error for scenario without UI")
	}
}

func TestValidateScenarioForDesktop_WithUI(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioName := "test-scenario"
	scenarioPath := filepath.Join(tmpDir, "scenarios", scenarioName)
	distDir := filepath.Join(scenarioPath, "ui", "dist")

	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatalf("failed to create dist dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("failed to write index.html: %v", err)
	}

	analyzer := NewAnalyzer(tmpDir)
	err := analyzer.ValidateScenarioForDesktop(scenarioName)
	if err != nil {
		t.Errorf("expected no error for scenario with UI, got: %v", err)
	}
}

func TestCreateDesktopConfigFromMetadata_NoUI(t *testing.T) {
	metadata := &ScenarioMetadata{
		Name:  "test-scenario",
		HasUI: false,
	}

	analyzer := NewAnalyzer("")
	_, err := analyzer.CreateDesktopConfigFromMetadata(metadata, "basic")
	if err == nil {
		t.Errorf("expected error for metadata without UI")
	}
}

func TestCreateDesktopConfigFromMetadata_StaticServer(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario")
	distPath := filepath.Join(scenarioPath, "ui", "dist")

	if err := os.MkdirAll(distPath, 0o755); err != nil {
		t.Fatalf("failed to create dist dir: %v", err)
	}

	metadata := &ScenarioMetadata{
		Name:         "test-scenario",
		DisplayName:  "Test Scenario",
		Description:  "A test app",
		Version:      "1.0.0",
		Author:       "Test Author",
		License:      "MIT",
		AppID:        "com.test.app",
		HasUI:        true,
		UIDistPath:   distPath,
		UIPort:       5173,
		APIPort:      3000,
		ScenarioPath: scenarioPath,
	}

	analyzer := NewAnalyzer(tmpDir)
	config, err := analyzer.CreateDesktopConfigFromMetadata(metadata, "basic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Without API directory, should be static server
	if config.ServerType != "static" {
		t.Errorf("expected ServerType 'static', got %q", config.ServerType)
	}
	if config.AppName != "test-scenario" {
		t.Errorf("expected AppName 'test-scenario', got %q", config.AppName)
	}
	if config.AppDisplayName != "Test Scenario" {
		t.Errorf("expected AppDisplayName 'Test Scenario', got %q", config.AppDisplayName)
	}
	if config.Framework != "electron" {
		t.Errorf("expected Framework 'electron', got %q", config.Framework)
	}
	if config.TemplateType != "basic" {
		t.Errorf("expected TemplateType 'basic', got %q", config.TemplateType)
	}
}

func TestCreateDesktopConfigFromMetadata_ExternalServer(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario")
	distPath := filepath.Join(scenarioPath, "ui", "dist")
	apiPath := filepath.Join(scenarioPath, "api")

	if err := os.MkdirAll(distPath, 0o755); err != nil {
		t.Fatalf("failed to create dist dir: %v", err)
	}
	if err := os.MkdirAll(apiPath, 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	metadata := &ScenarioMetadata{
		Name:         "test-scenario",
		DisplayName:  "Test Scenario",
		Description:  "A test app",
		Version:      "1.0.0",
		Author:       "Test Author",
		License:      "MIT",
		AppID:        "com.test.app",
		HasUI:        true,
		UIDistPath:   distPath,
		UIPort:       5173,
		APIPort:      4000,
		ScenarioPath: scenarioPath,
	}

	analyzer := NewAnalyzer(tmpDir)
	config, err := analyzer.CreateDesktopConfigFromMetadata(metadata, "advanced")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With API directory, should be external server
	if config.ServerType != "external" {
		t.Errorf("expected ServerType 'external', got %q", config.ServerType)
	}
	if config.ExternalServerURL != "http://localhost:5173" {
		t.Errorf("expected ExternalServerURL 'http://localhost:5173', got %q", config.ExternalServerURL)
	}
	if config.ExternalAPIURL != "http://localhost:4000" {
		t.Errorf("expected ExternalAPIURL 'http://localhost:4000', got %q", config.ExternalAPIURL)
	}
	// System tray should be enabled for advanced template
	if systemTray, ok := config.Features["systemTray"].(bool); !ok || !systemTray {
		t.Errorf("expected systemTray feature enabled for advanced template")
	}
}

func TestCreateDesktopConfigFromMetadata_DefaultPorts(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario")
	distPath := filepath.Join(scenarioPath, "ui", "dist")
	apiPath := filepath.Join(scenarioPath, "api")

	if err := os.MkdirAll(distPath, 0o755); err != nil {
		t.Fatalf("failed to create dist dir: %v", err)
	}
	if err := os.MkdirAll(apiPath, 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	// Metadata with zero ports (should use defaults)
	metadata := &ScenarioMetadata{
		Name:         "test-scenario",
		HasUI:        true,
		UIDistPath:   distPath,
		UIPort:       0,
		APIPort:      0,
		ScenarioPath: scenarioPath,
	}

	analyzer := NewAnalyzer(tmpDir)
	config, err := analyzer.CreateDesktopConfigFromMetadata(metadata, "basic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.ExternalServerURL != "http://localhost:3000" {
		t.Errorf("expected default ExternalServerURL 'http://localhost:3000', got %q", config.ExternalServerURL)
	}
	if config.ExternalAPIURL != "http://localhost:4000" {
		t.Errorf("expected default ExternalAPIURL 'http://localhost:4000', got %q", config.ExternalAPIURL)
	}
}

func TestScenarioMetadata_Fields(t *testing.T) {
	metadata := &ScenarioMetadata{
		Name:            "test",
		DisplayName:     "Test",
		Description:     "Test desc",
		Version:         "1.0.0",
		Author:          "Author",
		License:         "MIT",
		AppID:           "com.test",
		HasUI:           true,
		UIDistPath:      "/path/to/dist",
		UIPort:          5173,
		APIPort:         3000,
		ScenarioPath:    "/path/to/scenario",
		Category:        "productivity",
		Tags:            []string{"tag1"},
		ServiceJSONPath: "/path/to/service.json",
		PackageJSONPath: "/path/to/package.json",
	}

	if metadata.Name != "test" {
		t.Errorf("expected Name 'test'")
	}
	if metadata.HasUI != true {
		t.Errorf("expected HasUI true")
	}
	if len(metadata.Tags) != 1 {
		t.Errorf("expected 1 tag")
	}
}
