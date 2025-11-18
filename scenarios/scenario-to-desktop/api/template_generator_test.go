package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplateGeneratorInterpolatesPlaceholders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "desktop-output")

	config := &DesktopConfig{
		AppName:          "test-scenario",
		AppDisplayName:   "Test Scenario",
		AppDescription:   "Integration test",
		Version:          "1.0.0",
		Author:           "Test",
		License:          "MIT",
		AppID:            "com.vrooli.test",
		Framework:        "electron",
		TemplateType:     "basic",
		OutputPath:       outputDir,
		ServerType:       "external",
		ServerPath:       "https://example.com/apps/test/",
		APIEndpoint:      "https://example.com/apps/test/api",
		DeploymentMode:   "external-server",
		ScenarioName:     "test-scenario",
		AutoManageVrooli: true,
		VrooliBinaryPath: "vrooli",
		Features:         map[string]interface{}{},
		Window:           map[string]interface{}{},
		Platforms:        []string{"win"},
	}

	payload, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, payload, 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	generatorPath := filepath.Clean(filepath.Join("..", "templates", "build-tools", "dist", "template-generator.js"))
	cmd := exec.Command("node", generatorPath, configPath)
	cmd.Env = append(os.Environ(), "SKIP_DESKTOP_DEPENDENCY_INSTALL=1")

	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("template generator failed: %v\n%s", err, string(output))
	}

	mainContentBytes, err := os.ReadFile(filepath.Join(outputDir, "src", "main.ts"))
	if err != nil {
		t.Fatalf("failed to read generated main.ts: %v", err)
	}
	mainContent := string(mainContentBytes)

	if strings.Contains(mainContent, "{{") {
		t.Fatalf("generated main.ts still contains template tokens: %s", mainContent)
	}
	if !strings.Contains(mainContent, `DEPLOYMENT_MODE: "external-server"`) {
		t.Fatalf("deployment mode placeholder missing from generated file: %s", mainContent)
	}
	if !strings.Contains(mainContent, `SCENARIO_NAME: "test-scenario"`) {
		t.Fatalf("scenario name placeholder missing from generated file")
	}
}
