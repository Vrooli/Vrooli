package scenario

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/testutil"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=6 | LAST: 2026-04-13

func TestSandboxEnvFromEnv(t *testing.T) {
	t.Setenv("VROOLI_SANDBOX_ID", "sandbox-123")
	t.Setenv("VROOLI_SANDBOX_MERGED", "/tmp/merged")
	t.Setenv("VROOLI_SANDBOX_SCOPE", "scenarios/alpha")

	env := SandboxEnvFromEnv()
	if env.ID != "sandbox-123" || env.Merged != "/tmp/merged" || env.Scope != "scenarios/alpha" {
		t.Fatalf("SandboxEnvFromEnv = %+v", env)
	}
}

func TestScenarioInScope(t *testing.T) {
	root := t.TempDir()
	testutil.WriteRepoContract(t, root, "scenarios")

	tests := []struct {
		name     string
		scenario string
		scope    string
		want     bool
	}{
		{name: "whole repo", scenario: "alpha", scope: "", want: true},
		{name: "scenarios root", scenario: "alpha", scope: "scenarios", want: true},
		{name: "specific scenario", scenario: "alpha", scope: "scenarios/alpha/api", want: true},
		{name: "different scenario", scenario: "alpha", scope: "scenarios/beta", want: false},
		{name: "outside scenarios", scenario: "alpha", scope: "packages/shared", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ScenarioInScope(root, tc.scenario, tc.scope); got != tc.want {
				t.Fatalf("ScenarioInScope(%q, %q) = %v, want %v", tc.scenario, tc.scope, got, tc.want)
			}
		})
	}
}

func TestResolveMergedPath(t *testing.T) {
	root := t.TempDir()
	testutil.WriteRepoContract(t, root, "scenarios")

	merged := "/tmp/sandbox/merged"
	if got := ResolveMergedPath(root, "alpha", "scenarios/alpha", merged); got != merged {
		t.Fatalf("exact scope path = %q", got)
	}
	if got := ResolveMergedPath(root, "alpha", "scenarios", merged); got != filepath.Join(merged, "alpha") {
		t.Fatalf("scenarios scope path = %q", got)
	}
	if got := ResolveMergedPath(root, "alpha", "", merged); got != filepath.Join(merged, "scenarios", "alpha") {
		t.Fatalf("root scope path = %q", got)
	}
}

func TestScenarioInScopeUsesContractDefinedScopePrefix(t *testing.T) {
	root := t.TempDir()
	testutil.WriteRepoContract(t, root, "apps")

	if !ScenarioInScope(root, "alpha", "apps/alpha/api") {
		t.Fatal("expected contract-defined app scope to match scenario")
	}
	if ScenarioInScope(root, "alpha", "scenarios/alpha") {
		t.Fatal("unexpected legacy scenarios/ scope match under apps contract")
	}
}

func TestResolveMergedPathUsesContractDefinedScenarioDir(t *testing.T) {
	root := t.TempDir()
	testutil.WriteRepoContract(t, root, "apps")

	merged := "/tmp/sandbox/merged"
	if got := ResolveMergedPath(root, "alpha", "", merged); got != filepath.Join(merged, "apps", "alpha") {
		t.Fatalf("root scope path = %q", got)
	}
	if got := ResolveMergedPath(root, "alpha", "apps", merged); got != filepath.Join(merged, "alpha") {
		t.Fatalf("apps scope path = %q", got)
	}
	if got := ResolveMergedPath(root, "alpha", "apps/alpha", merged); got != merged {
		t.Fatalf("exact app scope path = %q", got)
	}
}

func TestDiscoverIncludesSandboxOnlyScenario(t *testing.T) {
	root := t.TempDir()
	writeScenarioService(t, root, "alpha", "Canonical alpha")

	merged := t.TempDir()
	writeScenarioServiceAtPath(t, filepath.Join(merged, "alpha"), "Sandbox alpha")
	writeScenarioServiceAtPath(t, filepath.Join(merged, "beta"), "Sandbox beta")

	scenarios, err := Discover(root, SandboxEnv{Merged: merged, Scope: "scenarios"})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(scenarios) != 2 {
		t.Fatalf("scenario count = %d, want 2", len(scenarios))
	}
	if scenarios[0].Slug != "alpha" || scenarios[1].Slug != "beta" {
		t.Fatalf("scenario slugs = %q, %q", scenarios[0].Slug, scenarios[1].Slug)
	}
	if scenarios[0].Manifest.Service.Description != "Sandbox alpha" {
		t.Fatalf("alpha description = %q", scenarios[0].Manifest.Service.Description)
	}
}

func TestLoadUsesSandboxScenarioWhenInScope(t *testing.T) {
	root := t.TempDir()
	writeScenarioService(t, root, "alpha", "Canonical alpha")

	merged := t.TempDir()
	writeScenarioServiceAtPath(t, merged, "Sandbox alpha")

	loaded, err := Load(root, "alpha", SandboxEnv{Merged: merged, Scope: "scenarios/alpha"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !loaded.Redirected {
		t.Fatalf("expected sandbox redirect")
	}
	if loaded.Manifest.Service.Description != "Sandbox alpha" {
		t.Fatalf("description = %q", loaded.Manifest.Service.Description)
	}
}

func TestLoadUsesContractDefinedScenarioLayout(t *testing.T) {
	root := t.TempDir()
	testutil.WriteRepoContract(t, root, "apps")
	writeScenarioServiceUnderBase(t, filepath.Join(root, "apps"), "alpha", "Contract alpha")

	loaded, err := Load(root, "alpha", SandboxEnv{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if want := filepath.Join(root, "apps", "alpha"); loaded.Path != want {
		t.Fatalf("loaded path = %q, want %q", loaded.Path, want)
	}
	if want := filepath.Join(root, "apps", "alpha", ".vrooli", "service.json"); loaded.ServicePath != want {
		t.Fatalf("loaded service path = %q, want %q", loaded.ServicePath, want)
	}
}

func TestLoadFallsBackToCanonicalScenarioWhenSandboxPathMissing(t *testing.T) {
	root := t.TempDir()
	writeScenarioService(t, root, "alpha", "Canonical alpha")

	loaded, err := Load(root, "alpha", SandboxEnv{Merged: t.TempDir(), Scope: "scenarios/alpha"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Redirected {
		t.Fatalf("expected canonical path when sandbox scenario is missing")
	}
	if loaded.Manifest.Service.Description != "Canonical alpha" {
		t.Fatalf("description = %q", loaded.Manifest.Service.Description)
	}
}

func TestLoadMissingScenarioReturnsNotFound(t *testing.T) {
	root := t.TempDir()

	if _, err := Load(root, "missing", SandboxEnv{}); err != ErrNotFound {
		t.Fatalf("Load missing scenario error = %v, want %v", err, ErrNotFound)
	}
}

func TestReadServiceParsesHostRequirements(t *testing.T) {
	root := t.TempDir()
	servicePath := filepath.Join(root, ".vrooli", "service.json")
	testutil.WriteFile(t, servicePath, `{
  "service": {"name": "alpha"},
  "hostTools": [
    {"name": "docker", "required": true, "reason": "run containers", "when": ["setup"]}
  ],
  "hostSafeguards": [
    {"name": "remote_session_protection", "required": false, "reason": "protect sessions", "platforms": ["linux"]}
  ]
}`)

	manifest, err := ReadService(servicePath)
	if err != nil {
		t.Fatalf("ReadService: %v", err)
	}
	if len(manifest.HostTools) != 1 || manifest.HostTools[0].Name != "docker" {
		t.Fatalf("hostTools = %+v", manifest.HostTools)
	}
	if len(manifest.HostSafeguards) != 1 || manifest.HostSafeguards[0].Name != "remote_session_protection" {
		t.Fatalf("hostSafeguards = %+v", manifest.HostSafeguards)
	}
}

func TestReadServiceRejectsDuplicateHostRequirements(t *testing.T) {
	root := t.TempDir()
	servicePath := filepath.Join(root, ".vrooli", "service.json")
	testutil.WriteFile(t, servicePath, `{
  "service": {"name": "alpha"},
  "hostTools": [
    {"name": "docker", "required": true, "reason": "one"},
    {"name": "docker", "required": false, "reason": "two"}
  ]
}`)

	if _, err := ReadService(servicePath); err == nil || !strings.Contains(err.Error(), `duplicate tool declaration "docker"`) {
		t.Fatalf("ReadService error = %v", err)
	}
}

func TestResolveScenarioPathIgnoresOutOfScopeSandbox(t *testing.T) {
	root := t.TempDir()
	writeScenarioService(t, root, "alpha", "Canonical alpha")

	merged := t.TempDir()
	writeScenarioServiceAtPath(t, merged, "Sandbox alpha")

	path, redirected := ResolveScenarioPath(root, "alpha", SandboxEnv{
		Merged: merged,
		Scope:  "packages/shared",
	})
	if redirected {
		t.Fatalf("expected out-of-scope sandbox to be ignored")
	}
	if want := filepath.Join(root, "scenarios", "alpha"); path != want {
		t.Fatalf("ResolveScenarioPath = %q, want %q", path, want)
	}
}

func TestDiscoverUsesContractDefinedScenarioBase(t *testing.T) {
	root := t.TempDir()
	testutil.WriteRepoContract(t, root, "apps")
	writeScenarioServiceUnderBase(t, filepath.Join(root, "apps"), "alpha", "Contract alpha")
	writeScenarioServiceUnderBase(t, filepath.Join(root, "apps"), "beta", "Contract beta")

	scenarios, err := Discover(root, SandboxEnv{})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(scenarios) != 2 {
		t.Fatalf("scenario count = %d, want 2", len(scenarios))
	}
	if scenarios[0].Path != filepath.Join(root, "apps", "alpha") {
		t.Fatalf("alpha path = %q", scenarios[0].Path)
	}
	if scenarios[1].Path != filepath.Join(root, "apps", "beta") {
		t.Fatalf("beta path = %q", scenarios[1].Path)
	}
}

func TestScenarioPathHelpersUseInjectedContractResolver(t *testing.T) {
	original := contractPaths
	t.Cleanup(func() {
		contractPaths = original
	})

	contractPaths = fakeScenarioContractPaths{
		baseDir:     "/repo/apps",
		scenarioDir: "/repo/apps/alpha",
		servicePath: "/repo/apps/alpha/.custom/service.json",
	}

	if got := scenarioBaseDir("/repo"); got != "/repo/apps" {
		t.Fatalf("scenarioBaseDir = %q", got)
	}
	if got := scenarioRootPath("/repo", "alpha"); got != "/repo/apps/alpha" {
		t.Fatalf("scenarioRootPath = %q", got)
	}
	if got := scenarioServicePath("/repo", "alpha", "/repo/apps/alpha"); got != "/repo/apps/alpha/.custom/service.json" {
		t.Fatalf("scenarioServicePath = %q", got)
	}
}

func TestRepoContractPathsFallbacksWithoutContract(t *testing.T) {
	root := t.TempDir()
	resolver := repoContractPaths{}

	if got, want := resolver.ScenarioBaseDir(root), filepath.Join(root, "scenarios"); got != want {
		t.Fatalf("ScenarioBaseDir = %q, want %q", got, want)
	}
	if got, want := resolver.ScenarioRootPath(root, "alpha"), filepath.Join(root, "scenarios", "alpha"); got != want {
		t.Fatalf("ScenarioRootPath = %q, want %q", got, want)
	}
	if got, want := resolver.ScenarioServicePath(root, "alpha", filepath.Join(root, "scenarios", "alpha")), filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"); got != want {
		t.Fatalf("ScenarioServicePath = %q, want %q", got, want)
	}
}

func TestEvaluateHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ports := map[string]int{"API_PORT": extractPort(t, server.URL)}
	health := &HealthConfig{
		Checks: []HealthCheck{
			{
				Name:     "api",
				Type:     "http",
				Target:   "http://127.0.0.1:${API_PORT}/health",
				Critical: true,
				Timeout:  1000,
			},
		},
	}

	if got := EvaluateHealth(health, ports); got != "healthy" {
		t.Fatalf("EvaluateHealth = %q, want healthy", got)
	}
	if got := EvaluateHealth(nil, ports); got != "running" {
		t.Fatalf("EvaluateHealth(nil) = %q, want running", got)
	}
}

func TestReadServicePromotesTopLevelHealthConfig(t *testing.T) {
	root := t.TempDir()
	servicePath := filepath.Join(root, "service.json")
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "alpha"
  },
  "health": {
    "checks": [
      {
        "name": "api",
        "type": "http",
        "target": "http://127.0.0.1:${API_PORT}/health",
        "critical": true
      }
    ]
  },
  "lifecycle": {
    "version": "2.0.0"
  }
}`
	if err := os.WriteFile(servicePath, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", servicePath, err)
	}

	manifest, err := ReadService(servicePath)
	if err != nil {
		t.Fatalf("ReadService: %v", err)
	}
	if manifest.Lifecycle.Health == nil {
		t.Fatalf("expected lifecycle health to be promoted from top-level health")
	}
	if manifest.HealthConfig() == nil {
		t.Fatalf("expected HealthConfig helper to return promoted health")
	}
	if got := manifest.HealthConfig().Checks[0].Target; got != "http://127.0.0.1:${API_PORT}/health" {
		t.Fatalf("health target = %q", got)
	}
}

func TestHealthConfigFallsBackToTopLevelHealth(t *testing.T) {
	topLevel := &HealthConfig{Description: "top-level"}
	manifest := ServiceManifest{Health: topLevel}

	if got := manifest.HealthConfig(); got != topLevel {
		t.Fatalf("HealthConfig fallback = %+v, want %+v", got, topLevel)
	}
}

func TestReadServiceRejectsInvalidJSON(t *testing.T) {
	root := t.TempDir()
	servicePath := filepath.Join(root, "service.json")
	if err := os.WriteFile(servicePath, []byte(`{"service":`), 0o644); err != nil {
		t.Fatalf("write %s: %v", servicePath, err)
	}

	if _, err := ReadService(servicePath); err == nil {
		t.Fatalf("expected malformed service json to fail")
	}
}

func TestReadServiceSupportsLegacyDependencyGroups(t *testing.T) {
	root := t.TempDir()
	servicePath := filepath.Join(root, "service.json")
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "alpha"
  },
  "dependencies": {
    "resources": {
      "required": [
        {
          "name": "postgres",
          "purpose": "Store application data",
          "config": {
            "database": "alpha_db"
          }
        }
      ],
      "optional": [
        {
          "name": "redis",
          "description": "Cache responses"
        }
      ]
    },
    "scenarios": {
      "optional": [
        {
          "name": "test-genie",
          "description": "Run extended tests"
        }
      ]
    }
  }
}`
	if err := os.WriteFile(servicePath, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", servicePath, err)
	}

	manifest, err := ReadService(servicePath)
	if err != nil {
		t.Fatalf("ReadService: %v", err)
	}

	postgres, ok := manifest.Dependencies.Resources["postgres"]
	if !ok {
		t.Fatalf("expected postgres dependency to be loaded")
	}
	if !postgres.Required || !postgres.Enabled {
		t.Fatalf("postgres flags = %+v", postgres)
	}
	if postgres.Type != "resource" {
		t.Fatalf("postgres type = %q", postgres.Type)
	}
	if postgres.Database != "alpha_db" {
		t.Fatalf("postgres database = %q", postgres.Database)
	}

	redis, ok := manifest.Dependencies.Resources["redis"]
	if !ok {
		t.Fatalf("expected redis dependency to be loaded")
	}
	if redis.Required {
		t.Fatalf("redis should stay optional: %+v", redis)
	}

	testGenie, ok := manifest.Dependencies.Scenarios["test-genie"]
	if !ok {
		t.Fatalf("expected test-genie scenario dependency to be loaded")
	}
	if testGenie.Required {
		t.Fatalf("test-genie should stay optional: %+v", testGenie)
	}
	if testGenie.Type != "scenario" {
		t.Fatalf("test-genie type = %q", testGenie.Type)
	}
}

func TestEvaluateHealthDetectsDegradedAndUnhealthyStates(t *testing.T) {
	healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthyServer.Close()

	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failingServer.Close()

	degraded := &HealthConfig{
		Checks: []HealthCheck{
			{
				Name:     "api",
				Type:     "http",
				Target:   healthyServer.URL,
				Critical: true,
				Timeout:  1000,
			},
			{
				Name:     "ui",
				Type:     "http",
				Target:   failingServer.URL,
				Critical: false,
				Timeout:  1000,
			},
		},
	}
	if got := EvaluateHealth(degraded, nil); got != "degraded" {
		t.Fatalf("EvaluateHealth(degraded) = %q, want degraded", got)
	}

	unhealthy := &HealthConfig{
		Checks: []HealthCheck{
			{
				Name:     "custom",
				Type:     "unsupported",
				Critical: true,
			},
		},
	}
	if got := EvaluateHealth(unhealthy, nil); got != "unhealthy" {
		t.Fatalf("EvaluateHealth(unhealthy) = %q, want unhealthy", got)
	}
}

func TestPerformHealthCheckPostgresTarget(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer listener.Close()

	address := listener.Addr().String()
	check := HealthCheck{
		Name:    "db",
		Type:    "postgres",
		Target:  fmt.Sprintf("postgres://user:pass@%s/example", address),
		Timeout: 1000,
	}

	if err := PerformHealthCheck(check, nil); err != nil {
		t.Fatalf("PerformHealthCheck postgres: %v", err)
	}
}

func TestPerformHealthCheckRejectsInvalidHTTPURL(t *testing.T) {
	check := HealthCheck{
		Name:   "api",
		Type:   "http",
		Target: "http://[::1",
	}

	if err := PerformHealthCheck(check, nil); err == nil {
		t.Fatalf("expected invalid URL to fail health check")
	}
}

func TestScanSandboxScenarioNamesRespectsScope(t *testing.T) {
	root := t.TempDir()
	testutil.WriteRepoContract(t, root, "scenarios")

	merged := t.TempDir()
	writeScenarioServiceAtPath(t, merged, "Scoped alpha")

	names, err := scanSandboxScenarioNames(root, SandboxEnv{Merged: merged, Scope: "scenarios/alpha"})
	if err != nil {
		t.Fatalf("scanSandboxScenarioNames: %v", err)
	}
	if got := strings.Join(names, ","); got != "alpha" {
		t.Fatalf("sandbox names = %q, want alpha", got)
	}

	names, err = scanSandboxScenarioNames(root, SandboxEnv{Merged: merged, Scope: "packages/shared"})
	if err != nil {
		t.Fatalf("scanSandboxScenarioNames unrelated scope: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("expected unrelated sandbox scope to contribute no scenarios, got %v", names)
	}
}

func TestScanSandboxScenarioNamesSupportsRepoRootScope(t *testing.T) {
	root := t.TempDir()
	testutil.WriteRepoContract(t, root, "scenarios")

	merged := t.TempDir()
	writeScenarioServiceAtPath(t, filepath.Join(merged, "scenarios", "alpha"), "Sandbox alpha")

	names, err := scanSandboxScenarioNames(root, SandboxEnv{Merged: merged, Scope: ""})
	if err != nil {
		t.Fatalf("scanSandboxScenarioNames repo scope: %v", err)
	}
	if got := strings.Join(names, ","); got != "alpha" {
		t.Fatalf("sandbox names = %q, want alpha", got)
	}
}

func TestScanSandboxScenarioNamesUsesContractDefinedScenarioDir(t *testing.T) {
	root := t.TempDir()
	testutil.WriteRepoContract(t, root, "apps")

	merged := t.TempDir()
	writeScenarioServiceAtPath(t, filepath.Join(merged, "apps", "alpha"), "Sandbox alpha")

	names, err := scanSandboxScenarioNames(root, SandboxEnv{Merged: merged, Scope: ""})
	if err != nil {
		t.Fatalf("scanSandboxScenarioNames repo scope: %v", err)
	}
	if got := strings.Join(names, ","); got != "alpha" {
		t.Fatalf("sandbox names = %q, want alpha", got)
	}
}

func TestManifestHelpers(t *testing.T) {
	fixedPort := 5432
	manifest := ServiceManifest{
		Ports: map[string]Port{
			"db": {
				Description: "Database",
				Port:        &fixedPort,
			},
			"api": {
				EnvVar:      "API_PORT",
				Description: "API server",
				Range:       "15000-19999",
			},
		},
		Lifecycle: Lifecycle{
			Setup: Phase{
				Description: "Prepare the scenario",
				Steps: []PhaseStep{
					{Name: "install"},
				},
			},
		},
	}

	ports := manifest.SortedPorts()
	if len(ports) != 2 {
		t.Fatalf("port count = %d, want 2", len(ports))
	}
	if ports[0].Name != "api" || ports[1].Name != "db" {
		t.Fatalf("port order = %#v", ports)
	}
	if manifest.PortEnvVar("db") != "DB_PORT" {
		t.Fatalf("PortEnvVar fallback = %q", manifest.PortEnvVar("db"))
	}
	if got := strings.Join(manifest.PortEnvVars(), ","); got != "API_PORT,DB_PORT" {
		t.Fatalf("PortEnvVars = %q", got)
	}
	if got := manifest.PortEnvVar("missing"); got != "" {
		t.Fatalf("PortEnvVar missing = %q, want empty string", got)
	}

	phases := manifest.PhaseSummaries()
	if len(phases) != 11 {
		t.Fatalf("phase count = %d, want 11", len(phases))
	}
	if !phases[0].Defined || phases[0].Name != "setup" || phases[0].Steps != 1 {
		t.Fatalf("setup summary = %#v", phases[0])
	}
	if phases[1].Defined {
		t.Fatalf("develop phase should be undefined, got %#v", phases[1])
	}
}

func TestExpandTargetAndParsePostgresAddress(t *testing.T) {
	expanded := ExpandTarget("http://127.0.0.1:${API_PORT}/health?ui=$UI_PORT", map[string]int{
		"API_PORT": 18080,
		"UI_PORT":  38080,
	})
	if expanded != "http://127.0.0.1:18080/health?ui=38080" {
		t.Fatalf("expanded target = %q", expanded)
	}

	tests := []struct {
		name   string
		target string
		want   string
	}{
		{name: "empty", target: "", want: ""},
		{name: "dsn", target: "postgres://user:pass@db.example.com:5440/app", want: "db.example.com:5440"},
		{name: "hostport", target: "127.0.0.1:5433", want: "127.0.0.1:5433"},
		{name: "no port in dsn", target: "postgresql://user:pass@db.example.com/app", want: "db.example.com:5432"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parsePostgresAddress(tc.target)
			if err != nil {
				t.Fatalf("parsePostgresAddress(%q): %v", tc.target, err)
			}
			if got != tc.want {
				t.Fatalf("parsePostgresAddress(%q) = %q, want %q", tc.target, got, tc.want)
			}
		})
	}

	if _, err := parsePostgresAddress("[::1"); err == nil {
		t.Fatalf("expected invalid host:port input to return an error")
	}
}

func TestHealthConfigPrefersLifecycleHealth(t *testing.T) {
	topLevel := &HealthConfig{Description: "top-level"}
	lifecycle := &HealthConfig{Description: "lifecycle"}
	manifest := ServiceManifest{
		Health: topLevel,
		Lifecycle: Lifecycle{
			Health: lifecycle,
		},
	}

	if got := manifest.HealthConfig(); got != lifecycle {
		t.Fatalf("HealthConfig should prefer lifecycle health, got %+v", got)
	}
}

func writeScenarioService(t *testing.T, root, name, description string) {
	t.Helper()
	writeScenarioServiceAtPath(t, filepath.Join(root, "scenarios", name), description)
}

func writeScenarioServiceUnderBase(t *testing.T, baseDir, name, description string) {
	t.Helper()
	writeScenarioServiceAtPath(t, filepath.Join(baseDir, name), description)
}

func writeScenarioServiceAtPath(t *testing.T, scenarioPath, description string) {
	t.Helper()
	name := filepath.Base(scenarioPath)
	testutil.WriteJSON(t, filepath.Join(scenarioPath, ".vrooli", "service.json"), ServiceManifest{
		Version: "1.0.0",
		Service: ServiceMetadata{
			Name:        name,
			DisplayName: testScenarioDisplayName(name),
			Description: description,
			Version:     "0.1.0",
		},
		Ports: map[string]Port{
			"api": {EnvVar: "API_PORT", Range: "15000-19999"},
		},
		Lifecycle: Lifecycle{
			Version: "2.0.0",
			Develop: Phase{
				Description: "Run the scenario",
				Steps: []PhaseStep{{
					Name:       "start-api",
					Run:        "sleep 10",
					Background: true,
				}},
			},
		},
	})
}

func extractPort(t *testing.T, rawURL string) int {
	t.Helper()
	parts := strings.Split(rawURL, ":")
	if len(parts) == 0 {
		t.Fatalf("invalid URL %q", rawURL)
	}
	port := parts[len(parts)-1]
	port = strings.TrimSuffix(port, "/")
	value, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("parse port from %q: %v", rawURL, err)
	}
	return value
}

func testScenarioDisplayName(name string) string {
	parts := strings.FieldsFunc(strings.TrimSpace(name), func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}

type fakeScenarioContractPaths struct {
	baseDir     string
	scenarioDir string
	servicePath string
	scenarioKey string
	scopePrefix string
}

func (f fakeScenarioContractPaths) ScenarioBaseDir(root string) string {
	return f.baseDir
}

func (f fakeScenarioContractPaths) ScenarioRootPath(root, name string) string {
	return f.scenarioDir
}

func (f fakeScenarioContractPaths) ScenarioServicePath(root, name, scenarioPath string) string {
	return f.servicePath
}

func (f fakeScenarioContractPaths) ScenarioDirName(root string) string {
	return f.scenarioKey
}

func (f fakeScenarioContractPaths) ScenarioScopePrefix(root string) string {
	return f.scopePrefix
}

func (f fakeScenarioContractPaths) IsFullRepoScope(root, scope string) bool {
	scope = strings.TrimSpace(strings.TrimSuffix(filepath.ToSlash(scope), "/"))
	return scope == "" || scope == "." || scope == "/"
}
