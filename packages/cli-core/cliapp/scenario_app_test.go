package cliapp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
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

func TestNewStandardScenarioAppUsesStandardWiring(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", configDir)
	t.Setenv("DEMO_HTTP_TIMEOUT", "7s")

	app, err := NewStandardScenarioApp(StandardScenarioOptions{
		Name:                    "demo",
		Version:                 "0.1.0",
		Description:             "Demo CLI",
		ExtraAPIEnvVars:         []string{"API_BASE_URL"},
		ExtraConfigDirEnvVars:   []string{"CLI_CONFIG_DIR_OVERRIDE"},
		ExtraHTTPTimeoutEnvVars: []string{"DEMO_HTTP_TIMEOUT"},
		AllowAnonymous:          true,
	})
	if err != nil {
		t.Fatalf("NewStandardScenarioApp: %v", err)
	}

	if app.HTTPClient == nil || app.HTTPClient.Timeout() != 7*time.Second {
		t.Fatalf("expected standard http timeout wiring, got %v", app.HTTPClient.Timeout())
	}

	commands := app.CLI.commandGroups()
	if len(commands) < 3 {
		t.Fatalf("expected meta + standard command groups, got %d", len(commands))
	}
	if commands[1].Title != "Health" {
		t.Fatalf("expected health command group, got %q", commands[1].Title)
	}
	if commands[2].Title != "Configuration" {
		t.Fatalf("expected configuration command group, got %q", commands[2].Title)
	}
}

func TestScenarioAppStandardBaseCommandGroupsCanDisableDefaultCommands(t *testing.T) {
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

	disableStatus := false
	groups := app.StandardBaseCommandGroups(StandardBaseCommandOptions{
		IncludeStatusCommand: &disableStatus,
	})
	if len(groups) != 1 {
		t.Fatalf("expected only configuration group, got %d groups", len(groups))
	}
	if groups[0].Title != "Configuration" {
		t.Fatalf("expected configuration group, got %q", groups[0].Title)
	}
}

func TestScenarioAppAPIPathHelpersFromRootBase(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", configDir)

	app, err := NewScenarioApp(ScenarioOptions{
		Name:             "demo",
		DefaultAPIBase:   "http://example.com",
		ConfigDirEnvVars: []string{"CLI_CONFIG_DIR_OVERRIDE"},
		AllowAnonymous:   true,
	})
	if err != nil {
		t.Fatalf("NewScenarioApp: %v", err)
	}

	if got := app.APIBase(); got != "http://example.com/api/v1" {
		t.Fatalf("APIBase() = %q, want %q", got, "http://example.com/api/v1")
	}
	if got := app.APIRootBase(); got != "http://example.com" {
		t.Fatalf("APIRootBase() = %q, want %q", got, "http://example.com")
	}
	if got := app.APIPath("/tasks"); got != "/api/v1/tasks" {
		t.Fatalf("APIPath(/tasks) = %q, want %q", got, "/api/v1/tasks")
	}
	if got := app.APIRootPath("/health"); got != "/health" {
		t.Fatalf("APIRootPath(/health) = %q, want %q", got, "/health")
	}
}

func TestScenarioAppAPIPathHelpersFromVersionedBase(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", configDir)

	app, err := NewScenarioApp(ScenarioOptions{
		Name:             "demo",
		DefaultAPIBase:   "http://example.com/api/v1",
		ConfigDirEnvVars: []string{"CLI_CONFIG_DIR_OVERRIDE"},
		AllowAnonymous:   true,
	})
	if err != nil {
		t.Fatalf("NewScenarioApp: %v", err)
	}

	if got := app.APIBase(); got != "http://example.com/api/v1" {
		t.Fatalf("APIBase() = %q, want %q", got, "http://example.com/api/v1")
	}
	if got := app.APIRootBase(); got != "http://example.com" {
		t.Fatalf("APIRootBase() = %q, want %q", got, "http://example.com")
	}
	if got := app.APIPath("/tasks"); got != "/tasks" {
		t.Fatalf("APIPath(/tasks) = %q, want %q", got, "/tasks")
	}
}

func TestScenarioAppStandardStatusCommandUsesRootHealth(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", configDir)

	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy","service":"demo-api","readiness":true,"dependencies":{"postgres":{"connected":true}}}`))
	}))
	defer server.Close()

	app, err := NewScenarioApp(ScenarioOptions{
		Name:             "demo",
		DefaultAPIBase:   server.URL + "/api/v1",
		ConfigDirEnvVars: []string{"CLI_CONFIG_DIR_OVERRIDE"},
		AllowAnonymous:   true,
	})
	if err != nil {
		t.Fatalf("NewScenarioApp: %v", err)
	}

	var stdout bytes.Buffer
	if err := app.runStandardStatus(nil, &stdout); err != nil {
		t.Fatalf("runStandardStatus: %v", err)
	}
	if requestedPath != "/health" {
		t.Fatalf("requested path = %q, want %q", requestedPath, "/health")
	}
	output := stdout.String()
	for _, needle := range []string{"Status: healthy", "Ready: true", "Service: demo-api", "postgres: connected"} {
		if !strings.Contains(output, needle) {
			t.Fatalf("status output missing %q in %q", needle, output)
		}
	}
}

func TestScenarioAppStandardStatusCommandSupportsJSON(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", configDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy","service":"demo-api","readiness":true}`))
	}))
	defer server.Close()

	app, err := NewScenarioApp(ScenarioOptions{
		Name:             "demo",
		DefaultAPIBase:   server.URL,
		ConfigDirEnvVars: []string{"CLI_CONFIG_DIR_OVERRIDE"},
		AllowAnonymous:   true,
	})
	if err != nil {
		t.Fatalf("NewScenarioApp: %v", err)
	}

	var stdout bytes.Buffer
	if err := app.runStandardStatus([]string{"--json"}, &stdout); err != nil {
		t.Fatalf("runStandardStatus --json: %v", err)
	}
	if !strings.Contains(stdout.String(), "\"status\": \"healthy\"") {
		t.Fatalf("expected pretty JSON output, got %q", stdout.String())
	}
}

func TestScenarioAppGetUsesVersionedAPIPath(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", configDir)

	var requestedPath string
	var requestedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		requestedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	app, err := NewScenarioApp(ScenarioOptions{
		Name:             "demo",
		DefaultAPIBase:   server.URL,
		ConfigDirEnvVars: []string{"CLI_CONFIG_DIR_OVERRIDE"},
		AllowAnonymous:   true,
	})
	if err != nil {
		t.Fatalf("NewScenarioApp: %v", err)
	}

	query := url.Values{"limit": []string{"10"}}
	if _, err := app.Get("/tasks", query); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if requestedPath != "/api/v1/tasks" {
		t.Fatalf("requested path = %q, want %q", requestedPath, "/api/v1/tasks")
	}
	if requestedQuery != "limit=10" {
		t.Fatalf("requested query = %q, want %q", requestedQuery, "limit=10")
	}
}

func TestScenarioAppGetRootUsesRootAPIPath(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", configDir)

	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()

	app, err := NewScenarioApp(ScenarioOptions{
		Name:             "demo",
		DefaultAPIBase:   server.URL + "/api/v1",
		ConfigDirEnvVars: []string{"CLI_CONFIG_DIR_OVERRIDE"},
		AllowAnonymous:   true,
	})
	if err != nil {
		t.Fatalf("NewScenarioApp: %v", err)
	}

	if _, err := app.GetRoot("/health", nil); err != nil {
		t.Fatalf("GetRoot: %v", err)
	}
	if requestedPath != "/health" {
		t.Fatalf("requested path = %q, want %q", requestedPath, "/health")
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

	fixture := testkitgo.NewRepoFixture(t, testkitgo.WithScenarioDir(scenarioDir))
	fixture.WriteRepoContract(t)
	for _, scenario := range []string{"swarm-manager", "scenario-to-desktop"} {
		if err := os.MkdirAll(filepath.Join(fixture.Root, scenarioDir, scenario, "cli"), 0o755); err != nil {
			t.Fatalf("mkdir cli dir: %v", err)
		}
		fixture.WriteScenarioStub(t, scenario)
	}

	return fixture.Root
}
