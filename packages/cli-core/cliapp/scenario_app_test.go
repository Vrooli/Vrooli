package cliapp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScenarioAppConfigureCommandSavesConfig(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", configDir)

	app, err := NewScenarioApp(ScenarioOptions{
		Name:             "demo",
		ConfigDirEnvVars: []string{"CLI_CONFIG_DIR_OVERRIDE"},
		AllowAnonymous:   true,
	})
	if err != nil {
		t.Fatalf("NewScenarioApp: %v", err)
	}

	cmd := app.ConfigureCommand(nil, []string{"token"})
	if err := cmd.Run([]string{"api_base", "http://example.com"}); err != nil {
		t.Fatalf("configure api_base: %v", err)
	}
	if err := cmd.Run([]string{"token", "secret"}); err != nil {
		t.Fatalf("configure token: %v", err)
	}

	// Ensure config persisted to disk with the expected values.
	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "example.com") || !strings.Contains(string(data), "secret") {
		t.Fatalf("config file missing expected values: %s", string(data))
	}
}

func TestScenarioAppPreflightValidatesAPIBase(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", configDir)
	t.Setenv("API_BASE_ENV", "http://localhost:9999")

	app, err := NewScenarioApp(ScenarioOptions{
		Name:             "demo",
		ConfigDirEnvVars: []string{"CLI_CONFIG_DIR_OVERRIDE"},
		APIEnvVars:       []string{"API_BASE_ENV"},
		AllowAnonymous:   true,
	})
	if err != nil {
		t.Fatalf("NewScenarioApp: %v", err)
	}

	ran := false
	cmd := Command{Name: "run", NeedsAPI: true, Run: func(args []string) error { ran = true; return nil }}
	app.SetCommands([]CommandGroup{{Title: "Test", Commands: []Command{cmd}}})

	if err := app.CLI.Run([]string{"run"}); err != nil {
		t.Fatalf("expected run to succeed, got: %v", err)
	}
	if !ran {
		t.Fatalf("expected command to execute after preflight")
	}
}

func TestScenarioAppPreflightFailsWhenAPIBaseMissing(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", configDir)

	app, err := NewScenarioApp(ScenarioOptions{
		Name:             "demo",
		ConfigDirEnvVars: []string{"CLI_CONFIG_DIR_OVERRIDE"},
		AllowAnonymous:   true,
	})
	if err != nil {
		t.Fatalf("NewScenarioApp: %v", err)
	}

	cmd := Command{Name: "run", NeedsAPI: true, Run: func(args []string) error { return nil }}
	app.SetCommands([]CommandGroup{{Title: "Test", Commands: []Command{cmd}}})

	if err := app.CLI.Run([]string{"run"}); err == nil {
		t.Fatalf("expected preflight error for missing API base")
	}
}

func TestScenarioAppPrefersTokenFromEnv(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", configDir)
	t.Setenv("DEMO_API_TOKEN", "from-env")

	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app, err := NewScenarioApp(ScenarioOptions{
		Name:             "demo",
		DefaultAPIBase:   server.URL,
		ConfigDirEnvVars: []string{"CLI_CONFIG_DIR_OVERRIDE"},
		TokenEnvVars:     []string{"DEMO_API_TOKEN"},
	})
	if err != nil {
		t.Fatalf("NewScenarioApp: %v", err)
	}

	app.Config.Token = "config-token"
	if _, err := app.APIClient.Get("/health", nil); err != nil {
		t.Fatalf("api call failed: %v", err)
	}
	if authHeader != "Bearer from-env" {
		t.Fatalf("expected token from env, got %q", authHeader)
	}
}

func TestScenarioAppFailsWhenTokenMissingForAPICall(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", configDir)
	t.Setenv("DEMO_API_BASE", "http://localhost:9999")

	app, err := NewScenarioApp(ScenarioOptions{
		Name:             "demo",
		APIEnvVars:       []string{"DEMO_API_BASE"},
		ConfigDirEnvVars: []string{"CLI_CONFIG_DIR_OVERRIDE"},
	})
	if err != nil {
		t.Fatalf("NewScenarioApp: %v", err)
	}

	cmd := Command{Name: "run", NeedsAPI: true, Run: func(args []string) error { return nil }}
	app.SetCommands([]CommandGroup{{Title: "Test", Commands: []Command{cmd}}})

	if err := app.CLI.Run([]string{"run"}); err == nil || !strings.Contains(err.Error(), "API token is required") {
		t.Fatalf("expected missing token error, got %v", err)
	}
}

func TestScenarioAppHTTPTimeoutFromEnv(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", configDir)
	t.Setenv("DEMO_HTTP_TIMEOUT", "5s")

	app, err := NewScenarioApp(ScenarioOptions{
		Name:               "demo",
		ConfigDirEnvVars:   []string{"CLI_CONFIG_DIR_OVERRIDE"},
		HTTPTimeoutEnvVars: []string{"DEMO_HTTP_TIMEOUT"},
		AllowAnonymous:     true,
	})
	if err != nil {
		t.Fatalf("NewScenarioApp: %v", err)
	}

	if app.HTTPClient == nil || app.HTTPClient.Timeout() != 5*time.Second {
		t.Fatalf("expected http client timeout from env, got %v", app.HTTPClient.Timeout())
	}
}

func TestIsScenarioLocalCLIExecutablePath(t *testing.T) {
	root := writeScenarioRepoFixture(t, "scenarios")
	t.Setenv("VROOLI_ROOT", root)

	tests := []struct {
		name       string
		appName    string
		executable string
		want       bool
	}{
		{
			name:       "scenario-local path",
			appName:    "swarm-manager",
			executable: filepath.Join(root, "scenarios", "swarm-manager", "cli", "swarm-manager"),
			want:       true,
		},
		{
			name:       "installed bin path",
			appName:    "swarm-manager",
			executable: "/home/user/.vrooli/bin/swarm-manager",
			want:       false,
		},
		{
			name:       "different scenario local path",
			appName:    "swarm-manager",
			executable: filepath.Join(root, "scenarios", "scenario-to-desktop", "cli", "scenario-to-desktop"),
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isScenarioLocalCLIExecutablePath(tt.appName, tt.executable)
			if got != tt.want {
				t.Fatalf("isScenarioLocalCLIExecutablePath(%q, %q) = %t, want %t", tt.appName, tt.executable, got, tt.want)
			}
		})
	}
}

func TestIsScenarioLocalCLIExecutablePathUsesContractDefinedLayout(t *testing.T) {
	root := writeScenarioRepoFixture(t, "apps")
	t.Setenv("VROOLI_ROOT", root)

	if !isScenarioLocalCLIExecutablePath("swarm-manager", filepath.Join(root, "apps", "swarm-manager", "cli", "swarm-manager")) {
		t.Fatal("expected contract-defined app-local CLI path to match")
	}
	if isScenarioLocalCLIExecutablePath("swarm-manager", filepath.Join(root, "scenarios", "swarm-manager", "cli", "swarm-manager")) {
		t.Fatal("unexpected legacy scenarios path match under apps contract")
	}
}

func TestResolveScenarioLocalCLIContextUsesContractDefinedLayout(t *testing.T) {
	root := writeScenarioRepoFixture(t, "apps")
	t.Setenv("VROOLI_ROOT", root)

	relativeScenario, cliDir, ok := resolveScenarioLocalCLIContext("swarm-manager")
	if !ok {
		t.Fatal("expected contract-backed CLI context to resolve")
	}
	if relativeScenario != "apps/swarm-manager" {
		t.Fatalf("relativeScenario = %q, want %q", relativeScenario, "apps/swarm-manager")
	}
	if cliDir != filepath.Join(root, "apps", "swarm-manager", "cli") {
		t.Fatalf("cliDir = %q, want %q", cliDir, filepath.Join(root, "apps", "swarm-manager", "cli"))
	}
}

func writeScenarioRepoFixture(t *testing.T, scenarioDir string) string {
	t.Helper()

	root := t.TempDir()
	doc := map[string]any{
		"$schema": "schemas/repo-contract.schema.json",
		"version": "1.0.0",
		"platform": map[string]any{
			"mode":                          "cross_platform_go_native",
			"legacy_project_bash_supported": false,
		},
		"root": map[string]any{
			"markers": map[string]any{
				"required_dirs":  []string{".vrooli", scenarioDir, "resources", "packages", "cmd", "internal"},
				"required_files": []string{"go.mod"},
			},
		},
		"layout": map[string]any{
			"project_config_dir": ".vrooli",
			"scenario_dir":       scenarioDir,
			"resource_dir":       "resources",
			"package_dir":        "packages",
			"command_dir":        "cmd",
			"internal_dir":       "internal",
			"docs_dir":           "docs",
		},
		"scenario": map[string]any{
			"required_files": []string{".vrooli/service.json"},
			"well_known_paths": map[string]string{
				"service": ".vrooli/service.json",
				"api":     "api",
				"cli":     "cli",
			},
		},
		"resource": map[string]any{
			"manifest": "resource.json",
		},
		"globs": map[string]any{
			"syntax":         "doublestar",
			"root_relative":  true,
			"case_sensitive": true,
			"allow_absolute": false,
			"path_format":    "slash_normalized",
		},
		"environment": map[string]any{
			"variables": map[string]string{
				"repo_root":      "VROOLI_ROOT",
				"source_root":    "VROOLI_SOURCE_ROOT",
				"sandbox_id":     "VROOLI_SANDBOX_ID",
				"sandbox_merged": "VROOLI_SANDBOX_MERGED",
				"sandbox_scope":  "VROOLI_SANDBOX_SCOPE",
			},
		},
		"sandbox": map[string]any{
			"full_repo_scopes":      []string{"", ".", "/"},
			"scenario_scope_prefix": scenarioDir + "/",
		},
		"profiles": map[string]any{
			"mini_vrooli_bundle": map[string]any{
				"description": "test profile",
				"parameters":  []string{"scenario"},
				"include":     []string{scenarioDir + "/{scenario}"},
			},
		},
	}

	for _, dir := range []string{".vrooli", scenarioDir, "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, scenario := range []string{"swarm-manager", "scenario-to-desktop"} {
		servicePath := filepath.Join(root, scenarioDir, scenario, ".vrooli", "service.json")
		if err := os.MkdirAll(filepath.Dir(servicePath), 0o755); err != nil {
			t.Fatalf("mkdir service dir: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(root, scenarioDir, scenario, "cli"), 0o755); err != nil {
			t.Fatalf("mkdir cli dir: %v", err)
		}
		if err := os.WriteFile(servicePath, []byte(`{"service":{"name":"`+scenario+`"}}`), 0o644); err != nil {
			t.Fatalf("write service: %v", err)
		}
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), data, 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	return root
}
