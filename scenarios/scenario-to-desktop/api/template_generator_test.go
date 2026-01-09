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

func TestTemplateGeneratorInjectsBundledRuntimeConfig(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "desktop-output")

	manifestPath := filepath.Join(tempDir, "bundle.json")
	if err := os.WriteFile(manifestPath, []byte(`{"schema_version":"desktop.v0.1"}`), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	config := &DesktopConfig{
		AppName:                  "bundled-scenario",
		AppDisplayName:           "Bundled Scenario",
		AppDescription:           "Bundled runtime test",
		Version:                  "2.0.0",
		Author:                   "Test",
		License:                  "MIT",
		AppID:                    "com.vrooli.bundled",
		Framework:                "electron",
		TemplateType:             "basic",
		OutputPath:               outputDir,
		ServerType:               "external",
		ServerPath:               "https://example.com/apps/bundled/",
		APIEndpoint:              "https://example.com/apps/bundled/api",
		DeploymentMode:           "bundled",
		ScenarioName:             "bundled-scenario",
		AutoManageVrooli:         false,
		VrooliBinaryPath:         "vrooli",
		Features:                 map[string]interface{}{},
		Window:                   map[string]interface{}{},
		Platforms:                []string{"win"},
		BundleManifestPath:       manifestPath,
		BundleRuntimeRoot:        "custom-bundle-root",
		BundleIPC:                &BundleIPCConfig{Host: "10.0.0.5", Port: 49100, AuthTokenRel: "runtime/custom-token"},
		BundleUISvcID:            "ui-service",
		BundleUIPortName:         "ui-port",
		BundleTelemetryUploadURL: "https://telemetry.example.com/upload",
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

	if !strings.Contains(mainContent, `DEPLOYMENT_MODE: "bundled"`) {
		t.Fatalf("deployment mode placeholder missing from generated file: %s", mainContent)
	}
	if !strings.Contains(mainContent, `SUPPORTED: true`) {
		t.Fatalf("bundled runtime flag missing from generated file: %s", mainContent)
	}

	expectedSnippets := []string{
		`ROOT: "custom-bundle-root"`,
		`IPC_HOST: "10.0.0.5"`,
		`IPC_PORT: 49100`,
		`TOKEN_REL: "runtime/custom-token"`,
		`UI_SERVICE: "ui-service"`,
		`UI_PORT_NAME: "ui-port"`,
		`TELEMETRY_UPLOAD_URL: "https://telemetry.example.com/upload"`,
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(mainContent, snippet) {
			t.Fatalf("expected bundled runtime snippet missing: %q", snippet)
		}
	}
}
