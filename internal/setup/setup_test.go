package setup

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/resources"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=3 | LAST: 2026-04-11

func TestParseOptionsAcceptsSetupFlags(t *testing.T) {
	opts, err := parseOptions("setup", []string{"--environment", "minimal", "--resources", "none", "--yes", "yes", "--sudo-mode", "skip", "--dry-run"})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opts.Environment != "minimal" || opts.Resources != "none" || opts.Yes != "yes" || opts.SudoMode != "skip" || !opts.DryRun {
		t.Fatalf("opts = %+v", opts)
	}
}

func TestApplyEnvironmentSetsDefaultsAndRestoresState(t *testing.T) {
	t.Setenv("TARGET", "")
	t.Setenv("LOCATION", "")
	root := t.TempDir()
	restore, err := applyEnvironment(root, filepath.Join(root, ".vrooli", "service.json"), options{
		Environment: "production",
		Resources:   "none",
		Yes:         "yes",
		SudoMode:    "skip",
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("applyEnvironment: %v", err)
	}

	if got := os.Getenv("APP_ROOT"); got != root {
		t.Fatalf("APP_ROOT = %q", got)
	}
	if got := os.Getenv("TARGET"); got != defaultTarget {
		t.Fatalf("TARGET = %q", got)
	}
	if got := os.Getenv("LOCATION"); got != defaultLocation {
		t.Fatalf("LOCATION = %q", got)
	}
	if got := os.Getenv("ENVIRONMENT"); got != "production" {
		t.Fatalf("ENVIRONMENT = %q", got)
	}
	if got := os.Getenv("RESOURCES"); got != "none" {
		t.Fatalf("RESOURCES = %q", got)
	}
	if got := os.Getenv("YES"); got != "yes" {
		t.Fatalf("YES = %q", got)
	}
	if got := os.Getenv("SUDO_MODE"); got != "skip" {
		t.Fatalf("SUDO_MODE = %q", got)
	}
	if got := os.Getenv("DRY_RUN"); got != "true" {
		t.Fatalf("DRY_RUN = %q", got)
	}
	if got := os.Getenv("SERVICE_JSON_PATH"); got != filepath.Join(root, ".vrooli", "service.json") {
		t.Fatalf("SERVICE_JSON_PATH = %q", got)
	}

	restore()

	if got := os.Getenv("APP_ROOT"); got != "" {
		t.Fatalf("APP_ROOT after restore = %q", got)
	}
	if got := os.Getenv("TARGET"); got != "" {
		t.Fatalf("TARGET after restore = %q", got)
	}
	if got := os.Getenv("SERVICE_JSON_PATH"); got != "" {
		t.Fatalf("SERVICE_JSON_PATH after restore = %q", got)
	}
}

func TestMarkCompleteWritesSetupAndResourceMarkers(t *testing.T) {
	root := t.TempDir()
	manifest := scenario.ServiceManifest{
		Lifecycle: scenario.Lifecycle{
			Setup: scenario.Phase{
				Steps: []scenario.PhaseStep{
					{Name: "base-setup"},
					{Name: "add-data"},
				},
			},
		},
	}

	if err := markComplete(root, manifest); err != nil {
		t.Fatalf("markComplete: %v", err)
	}

	setupMarker := filepath.Join(root, "data", ".setup-complete")
	data, err := os.ReadFile(setupMarker)
	if err != nil {
		t.Fatalf("read setup marker: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal setup marker: %v", err)
	}
	if payload["setup_version"] != "2.0.0" {
		t.Fatalf("setup_version = %v", payload["setup_version"])
	}
	if _, err := os.Stat(filepath.Join(root, "data", ".resources-populated")); err != nil {
		t.Fatalf("expected resources marker: %v", err)
	}
}

func TestRunSetupUsesNativeRuntimeAndMarksComplete(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	currentHostFn = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }

	runtimeCalls := 0
	ensureRuntimeFn = func(opts vrooliruntime.EnsureOptions) (vrooliruntime.Report, error) {
		runtimeCalls++
		return vrooliruntime.Report{}, nil
	}
	markCompleteCalled := false
	markCompleteFn = func(root string, manifest scenario.ServiceManifest) error {
		markCompleteCalled = true
		return nil
	}

	if err := RunSetup(root, home, []string{"--dry-run", "--resources", "none"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunSetup: %v", err)
	}
	if runtimeCalls != 1 {
		t.Fatalf("ensureRuntime calls = %d, want 1", runtimeCalls)
	}
	if !markCompleteCalled {
		t.Fatal("expected markCompleteFn to be called")
	}
	if _, err := os.Stat(filepath.Join(root, "data")); err != nil {
		t.Fatalf("expected data dir: %v", err)
	}
}

func TestRunSetupExportsLegacyEnvironmentContractToResourceInstall(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	currentHostFn = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	ensureRuntimeFn = func(opts vrooliruntime.EnsureOptions) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{}, nil
	}
	markCompleteFn = func(root string, manifest scenario.ServiceManifest) error { return nil }

	type installCall struct {
		name string
		args []string
		env  map[string]string
	}
	var installs []installCall
	resourcesController = func(root, home string) resourceRunner {
		return resourceRunnerFunc(func(name string, args []string, stdout, stderr io.Writer) error {
			installs = append(installs, installCall{
				name: name,
				args: append([]string(nil), args...),
				env: map[string]string{
					"APP_ROOT":           os.Getenv("APP_ROOT"),
					"SERVICE_JSON_PATH":  os.Getenv("SERVICE_JSON_PATH"),
					"ENVIRONMENT":        os.Getenv("ENVIRONMENT"),
					"RESOURCES":          os.Getenv("RESOURCES"),
					"YES":                os.Getenv("YES"),
					"SUDO_MODE":          os.Getenv("SUDO_MODE"),
					"SUDO_MODE_EXPLICIT": os.Getenv("SUDO_MODE_EXPLICIT"),
					"TARGET":             os.Getenv("TARGET"),
					"LOCATION":           os.Getenv("LOCATION"),
					"DRY_RUN":            os.Getenv("DRY_RUN"),
				},
			})
			return nil
		})
	}

	err := RunSetup(root, home, []string{
		"--environment", "minimal",
		"--resources", "redis,postgres",
		"--yes", "yes",
		"--sudo-mode", "skip",
		"--dry-run",
	}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("RunSetup: %v", err)
	}
	if len(installs) != 2 {
		t.Fatalf("install calls = %d, want 2", len(installs))
	}
	for _, call := range installs {
		if got := strings.Join(call.args, "|"); got != "install" {
			t.Fatalf("resource %s args = %q", call.name, got)
		}
		if call.env["APP_ROOT"] != root {
			t.Fatalf("resource %s APP_ROOT = %q", call.name, call.env["APP_ROOT"])
		}
		if call.env["SERVICE_JSON_PATH"] != filepath.Join(root, ".vrooli", "service.json") {
			t.Fatalf("resource %s SERVICE_JSON_PATH = %q", call.name, call.env["SERVICE_JSON_PATH"])
		}
		if call.env["ENVIRONMENT"] != "minimal" {
			t.Fatalf("resource %s ENVIRONMENT = %q", call.name, call.env["ENVIRONMENT"])
		}
		if call.env["RESOURCES"] != "redis,postgres" {
			t.Fatalf("resource %s RESOURCES = %q", call.name, call.env["RESOURCES"])
		}
		if call.env["YES"] != "yes" {
			t.Fatalf("resource %s YES = %q", call.name, call.env["YES"])
		}
		if call.env["SUDO_MODE"] != "skip" || call.env["SUDO_MODE_EXPLICIT"] != "skip" {
			t.Fatalf("resource %s sudo env = %#v", call.name, call.env)
		}
		if call.env["TARGET"] != defaultTarget {
			t.Fatalf("resource %s TARGET = %q", call.name, call.env["TARGET"])
		}
		if call.env["LOCATION"] != defaultLocation {
			t.Fatalf("resource %s LOCATION = %q", call.name, call.env["LOCATION"])
		}
		if call.env["DRY_RUN"] != "true" {
			t.Fatalf("resource %s DRY_RUN = %q", call.name, call.env["DRY_RUN"])
		}
	}
}

func TestRunDevelopRunsSetupWhenNeededAndStartsNativeServices(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)
	writePortRegistryFixture(t, root)
	writeExecutableFile(t, filepath.Join(root, ".vrooli", "build", "vrooli-api"), "#!/usr/bin/env bash\nexit 0\n")
	t.Setenv("VROOLI_API_PORT", "18096")
	t.Setenv("VROOLI_API_PORT", "18095")

	currentHostFn = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	ensureRuntimeFn = func(opts vrooliruntime.EnsureOptions) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{}, nil
	}

	setupCalls := 0
	markCompleteFn = func(root string, manifest scenario.ServiceManifest) error {
		setupCalls++
		return os.WriteFile(filepath.Join(root, "data", ".setup-complete"), []byte("ok\n"), 0o644)
	}

	apiStarted := false
	startProjectAPIFn = func(root string, spec apiLaunchSpec, stdout, stderr io.Writer) error {
		apiStarted = true
		if spec.Command == "" || spec.Port != 18095 {
			t.Fatalf("spec = %+v", spec)
		}
		return nil
	}
	healthCalls := 0
	healthCheckFn = func(port int, timeout time.Duration) error {
		healthCalls++
		if port != 18095 {
			t.Fatalf("port = %d", port)
		}
		return nil
	}
	orchestratorStarted := false
	startOrchestratorFn = func(root, home string, stdout, stderr io.Writer) error {
		orchestratorStarted = true
		return nil
	}

	stdout := &strings.Builder{}
	if err := RunDevelop(root, home, nil, stdout, io.Discard); err != nil {
		t.Fatalf("RunDevelop: %v", err)
	}
	if setupCalls != 1 {
		t.Fatalf("setup calls = %d, want 1", setupCalls)
	}
	if !apiStarted {
		t.Fatal("expected project API to start")
	}
	if healthCalls != 1 {
		t.Fatalf("health calls = %d, want 1", healthCalls)
	}
	if !orchestratorStarted {
		t.Fatal("expected orchestrator startup")
	}
	if !strings.Contains(stdout.String(), "Running setup before develop") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunDevelopExportsLegacyEnvironmentContractToAPILaunch(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)
	writePortRegistryFixture(t, root)
	writeExecutableFile(t, filepath.Join(root, ".vrooli", "build", "vrooli-api"), "#!/usr/bin/env bash\nexit 0\n")
	t.Setenv("VROOLI_API_PORT", "18095")

	currentHostFn = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	ensureRuntimeFn = func(opts vrooliruntime.EnsureOptions) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{}, nil
	}
	markCompleteFn = func(root string, manifest scenario.ServiceManifest) error {
		return os.WriteFile(filepath.Join(root, "data", ".setup-complete"), []byte("ok\n"), 0o644)
	}
	loadDotEnvFn = func(path string) (map[string]string, error) {
		return map[string]string{
			"VROOLI_API_PORT": "18095",
			"FROM_DOT_ENV":    "present",
		}, nil
	}

	var capturedSpec apiLaunchSpec
	startProjectAPIFn = func(root string, spec apiLaunchSpec, stdout, stderr io.Writer) error {
		capturedSpec = spec
		return nil
	}
	healthCheckFn = func(port int, timeout time.Duration) error { return nil }
	startOrchestratorFn = func(root, home string, stdout, stderr io.Writer) error { return nil }

	err := RunDevelop(root, home, []string{
		"--environment", "production",
		"--resources", "enabled",
		"--yes", "yes",
		"--sudo-mode", "skip",
		"--dry-run",
	}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("RunDevelop: %v", err)
	}
	if capturedSpec.Command == "" {
		t.Fatal("expected API launch spec to be populated")
	}
	env := envMapFromList(capturedSpec.Env)
	if env["APP_ROOT"] != root {
		t.Fatalf("APP_ROOT = %q", env["APP_ROOT"])
	}
	if env["SERVICE_JSON_PATH"] != filepath.Join(root, ".vrooli", "service.json") {
		t.Fatalf("SERVICE_JSON_PATH = %q", env["SERVICE_JSON_PATH"])
	}
	if env["ENVIRONMENT"] != "production" {
		t.Fatalf("ENVIRONMENT = %q", env["ENVIRONMENT"])
	}
	if env["RESOURCES"] != "enabled" {
		t.Fatalf("RESOURCES = %q", env["RESOURCES"])
	}
	if env["YES"] != "yes" {
		t.Fatalf("YES = %q", env["YES"])
	}
	if env["SUDO_MODE"] != "skip" || env["SUDO_MODE_EXPLICIT"] != "skip" {
		t.Fatalf("sudo env = %#v", env)
	}
	if env["TARGET"] != defaultTarget {
		t.Fatalf("TARGET = %q", env["TARGET"])
	}
	if env["LOCATION"] != defaultLocation {
		t.Fatalf("LOCATION = %q", env["LOCATION"])
	}
	if env["DRY_RUN"] != "true" {
		t.Fatalf("DRY_RUN = %q", env["DRY_RUN"])
	}
	if env["FROM_DOT_ENV"] != "present" {
		t.Fatalf("FROM_DOT_ENV = %q", env["FROM_DOT_ENV"])
	}
	if env["VROOLI_API_PORT"] != "18095" {
		t.Fatalf("VROOLI_API_PORT = %q", env["VROOLI_API_PORT"])
	}
	if capturedSpec.Port != 18095 {
		t.Fatalf("capturedSpec.Port = %d", capturedSpec.Port)
	}
}

func TestRunDevelopSkipsSetupWhenMarkerExists(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)
	writePortRegistryFixture(t, root)
	writeExecutableFile(t, filepath.Join(root, ".vrooli", "build", "vrooli-api"), "#!/usr/bin/env bash\nexit 0\n")
	if err := os.MkdirAll(filepath.Join(root, "data"), 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "data", ".setup-complete"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write setup marker: %v", err)
	}

	currentHostFn = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	ensureRuntimeFn = func(opts vrooliruntime.EnsureOptions) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{}, nil
	}

	setupCalls := 0
	markCompleteFn = func(root string, manifest scenario.ServiceManifest) error {
		setupCalls++
		return nil
	}
	startProjectAPIFn = func(root string, spec apiLaunchSpec, stdout, stderr io.Writer) error { return nil }
	healthCheckFn = func(port int, timeout time.Duration) error { return nil }
	startOrchestratorFn = func(root, home string, stdout, stderr io.Writer) error { return nil }

	if err := RunDevelop(root, home, nil, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunDevelop: %v", err)
	}
	if setupCalls != 0 {
		t.Fatalf("setup calls = %d, want 0", setupCalls)
	}
}

func TestRunSetupRejectsUnsupportedHost(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	currentHostFn = func() vrooliruntime.Host {
		return vrooliruntime.Host{OS: "darwin", SupportsSetup: false, Notes: []string{"unsupported in test"}}
	}

	err := RunSetup(t.TempDir(), t.TempDir(), nil, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("expected unsupported host error")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadDotEnvParsesCommonForms(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	if err := os.WriteFile(path, []byte("FOO=bar\nexport BAZ=\"two\"\n# comment\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	values, err := loadDotEnv(path)
	if err != nil {
		t.Fatalf("loadDotEnv: %v", err)
	}
	if values["FOO"] != "bar" || values["BAZ"] != "two" {
		t.Fatalf("values = %#v", values)
	}
}

type resourceRunnerFunc func(name string, args []string, stdout, stderr io.Writer) error

func (fn resourceRunnerFunc) Run(name string, args []string, stdout, stderr io.Writer) error {
	return fn(name, args, stdout, stderr)
}

func envMapFromList(env []string) map[string]string {
	values := make(map[string]string, len(env))
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	return values
}

func stubSetupDeps(t *testing.T) func() {
	t.Helper()
	originalCurrentHostFn := currentHostFn
	originalLoadProjectFn := loadProjectFn
	originalMarkCompleteFn := markCompleteFn
	originalEnsureRuntimeFn := ensureRuntimeFn
	originalStartProjectAPIFn := startProjectAPIFn
	originalStartOrchestratorFn := startOrchestratorFn
	originalHealthCheckFn := healthCheckFn
	originalLoadDotEnvFn := loadDotEnvFn
	originalResourcesController := resourcesController
	return func() {
		currentHostFn = originalCurrentHostFn
		loadProjectFn = originalLoadProjectFn
		markCompleteFn = originalMarkCompleteFn
		ensureRuntimeFn = originalEnsureRuntimeFn
		startProjectAPIFn = originalStartProjectAPIFn
		startOrchestratorFn = originalStartOrchestratorFn
		healthCheckFn = originalHealthCheckFn
		loadDotEnvFn = originalLoadDotEnvFn
		resourcesController = originalResourcesController
	}
}

func writeProjectFixture(t *testing.T, root string) scenario.Scenario {
	t.Helper()
	servicePath := filepath.Join(root, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(servicePath), 0o755); err != nil {
		t.Fatalf("mkdir service dir: %v", err)
	}
	manifest := `{
  "version": "1.0.0",
  "service": {
    "name": "project-alpha",
    "displayName": "Project Alpha",
    "description": "Project-level fixture",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "VROOLI_API_PORT",
      "port": 8092
    }
  },
  "dependencies": {
    "resources": {
      "redis": { "enabled": false }
    }
  }
}`
	if err := os.WriteFile(servicePath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write service manifest: %v", err)
	}
	parsed, err := scenario.ReadService(servicePath)
	if err != nil {
		t.Fatalf("ReadService: %v", err)
	}
	return scenario.Scenario{
		Slug:        "project-alpha",
		Path:        root,
		ServicePath: servicePath,
		Manifest:    parsed,
	}
}

func writePortRegistryFixture(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "scripts", "resources", "port_registry.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir port registry dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{\"resource_ports\":{},\"reserved_ranges\":{}}\n"), 0o644); err != nil {
		t.Fatalf("write port registry: %v", err)
	}
}

func writeExecutableFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

var _ resourceRunner = (*resources.Controller)(nil)
