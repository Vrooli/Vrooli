package lifecycle

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=7 | LAST: 2026-04-10

func TestRunnerStartStopRestart(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	start, err := runner.Start("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if start.Health != "healthy" {
		t.Fatalf("health = %q, want healthy", start.Health)
	}
	records, err := process.ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords: %v", err)
	}
	live := process.LiveRecords(records)
	if len(live) != 1 {
		t.Fatalf("expected 1 live record after start, got %#v", live)
	}
	firstPID := live[0].PID

	setupNeeded, _, err := runner.SetupNeeded(start.Scenario, false)
	if err != nil {
		t.Fatalf("SetupNeeded after start: %v", err)
	}
	if setupNeeded {
		t.Fatalf("expected setup to be current after start")
	}

	restarted, err := runner.Restart("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if restarted.Health != "healthy" {
		t.Fatalf("restart health = %q, want healthy", restarted.Health)
	}
	records, err = process.ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords after restart: %v", err)
	}
	live = process.LiveRecords(records)
	if len(live) != 1 {
		t.Fatalf("expected 1 live record after restart, got %#v", live)
	}
	if live[0].PID == firstPID {
		t.Fatalf("expected new PID after restart, still %d", firstPID)
	}

	if err := runner.Stop("alpha", StopOptions{}); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	records, err = process.ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords after stop: %v", err)
	}
	if len(process.LiveRecords(records)) != 0 {
		t.Fatalf("expected no live records after stop: %#v", records)
	}
}

func TestSetupNeededDetectsUpdatedSources(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	result, err := runner.Start("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	sourcePath := filepath.Join(root, "scenarios", "alpha", "api", "handler.go")
	if err := os.WriteFile(sourcePath, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", sourcePath, err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(sourcePath, future, future); err != nil {
		t.Fatalf("chtimes %s: %v", sourcePath, err)
	}

	setupNeeded, reasons, err := runner.SetupNeeded(result.Scenario, false)
	if err != nil {
		t.Fatalf("SetupNeeded: %v", err)
	}
	if !setupNeeded {
		t.Fatalf("expected setup to be needed after touching source")
	}
	if len(reasons) == 0 {
		t.Fatalf("expected setup reasons to be populated")
	}

	_ = runner.Stop("alpha", StopOptions{})
}

func TestFileDependencySpecsIgnoresNonDependencyTopLevelFields(t *testing.T) {
	packageJSON := filepath.Join(t.TempDir(), "package.json")
	data := `{
  "name": "fixture",
  "version": "1.0.0",
  "scripts": {
    "build": "vite build"
  },
  "dependencies": {
    "@local/pkg-a": "file:../pkg-a",
    "react": "^18.0.0"
  },
  "devDependencies": {
    "@local/pkg-b": "file:../pkg-b"
  },
  "optionalDependencies": {
    "@local/pkg-c": "file:../pkg-c"
  }
}`
	if err := os.WriteFile(packageJSON, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", packageJSON, err)
	}

	specs, err := fileDependencySpecs(packageJSON)
	if err != nil {
		t.Fatalf("fileDependencySpecs: %v", err)
	}

	want := []string{"file:../pkg-a", "file:../pkg-b", "file:../pkg-c"}
	if len(specs) != len(want) {
		t.Fatalf("spec count = %d, want %d (%v)", len(specs), len(want), specs)
	}
	for i, spec := range specs {
		if spec != want[i] {
			t.Fatalf("spec[%d] = %q, want %q", i, spec, want[i])
		}
	}
}

func TestEnsureScenarioDatabaseUsesPostgresResourceLibs(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "scripts", "resources"), 0o755); err != nil {
		t.Fatalf("mkdir scripts/resources: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "resources", "port_registry.sh"), []byte("#!/usr/bin/env bash\ndeclare -g -A RESOURCE_PORTS=()\n"), 0o644); err != nil {
		t.Fatalf("write port_registry.sh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "resources", "port_registry.json"), []byte("{\n  \"resource_ports\": {},\n  \"reserved_ranges\": {}\n}\n"), 0o644); err != nil {
		t.Fatalf("write port_registry.json: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(root, "resources", "postgres", "config"), 0o755); err != nil {
		t.Fatalf("mkdir postgres config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "resources", "postgres", "lib"), 0o755); err != nil {
		t.Fatalf("mkdir postgres lib: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "resources", "postgres", "config", "defaults.sh"), []byte("#!/usr/bin/env bash\n"), 0o644); err != nil {
		t.Fatalf("write defaults.sh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "resources", "postgres", "lib", "common.sh"), []byte(`#!/usr/bin/env bash
postgres::common::is_running() { return 0; }
`), 0o644); err != nil {
		t.Fatalf("write common.sh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "resources", "postgres", "lib", "database.sh"), []byte(`#!/usr/bin/env bash
postgres::database::create() { printf '%s\n' "$2" > "$APP_ROOT/create.txt"; }
postgres::database::execute_file() { printf '%s|%s\n' "$2" "$3" > "$APP_ROOT/schema.txt"; }
postgres::database::migrate() { printf '%s|%s\n' "$2" "$3" > "$APP_ROOT/migrate.txt"; }
`), 0o644); err != nil {
		t.Fatalf("write database.sh: %v", err)
	}

	scenarioPath := filepath.Join(root, "scenarios", "alpha")
	if err := os.MkdirAll(filepath.Join(scenarioPath, "initialization", "postgres"), 0o755); err != nil {
		t.Fatalf("mkdir initialization/postgres: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioPath, "initialization", "postgres", "schema.sql"), []byte("create table if not exists test();\n"), 0o644); err != nil {
		t.Fatalf("write schema.sql: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioPath, "initialization", "postgres", "migration_001.sql"), []byte("-- migration\n"), 0o644); err != nil {
		t.Fatalf("write migration_001.sql: %v", err)
	}

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	item := scenario.Scenario{
		Slug: "alpha",
		Path: scenarioPath,
	}
	if err := runner.ensureScenarioDatabase(item, map[string]string{"POSTGRES_DB": "alpha_db"}, io.Discard); err != nil {
		t.Fatalf("ensureScenarioDatabase: %v", err)
	}

	createData, err := os.ReadFile(filepath.Join(root, "create.txt"))
	if err != nil {
		t.Fatalf("read create.txt: %v", err)
	}
	if got := string(createData); got != "alpha_db\n" {
		t.Fatalf("create.txt = %q", got)
	}

	schemaData, err := os.ReadFile(filepath.Join(root, "schema.txt"))
	if err != nil {
		t.Fatalf("read schema.txt: %v", err)
	}
	if got := string(schemaData); got != filepath.Join(scenarioPath, "initialization", "postgres", "schema.sql")+"|alpha_db\n" {
		t.Fatalf("schema.txt = %q", got)
	}

	migrateData, err := os.ReadFile(filepath.Join(root, "migrate.txt"))
	if err != nil {
		t.Fatalf("read migrate.txt: %v", err)
	}
	if got := string(migrateData); got != filepath.Join(scenarioPath, "initialization", "postgres")+"|alpha_db\n" {
		t.Fatalf("migrate.txt = %q", got)
	}
}

func TestRunnerStartStartsRequiredDependencies(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()

	writeLifecycleFixture(t, root, "beta")
	alpha := lifecycleFixtureManifest("alpha")
	alpha.Dependencies.Scenarios = map[string]scenario.Dependency{
		"beta": {Required: true},
	}
	writeLifecycleFixtureManifest(t, root, alpha)

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	t.Cleanup(func() {
		_ = runner.Stop("alpha", StopOptions{})
		_ = runner.Stop("beta", StopOptions{})
	})

	result, err := runner.Start("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(result.FailedDependencies) != 0 {
		t.Fatalf("unexpected failed dependencies: %#v", result.FailedDependencies)
	}

	for _, name := range []string{"alpha", "beta"} {
		records, err := process.ReadScenarioRecords(home, name)
		if err != nil {
			t.Fatalf("ReadScenarioRecords(%s): %v", name, err)
		}
		if len(process.LiveRecords(records)) != 1 {
			t.Fatalf("expected %s to be running, records=%#v", name, records)
		}
	}

	if _, err := os.Stat(process.ScenarioDegradedPath(home, "alpha")); !os.IsNotExist(err) {
		t.Fatalf("unexpected degraded state after successful dependency start: %v", err)
	}
}

func TestRunnerStartBestEffortWritesDegradedStateForMissingDependencies(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()

	alpha := lifecycleFixtureManifest("alpha")
	alpha.Dependencies.Scenarios = map[string]scenario.Dependency{
		"missing-beta": {Required: true},
	}
	writeLifecycleFixtureManifest(t, root, alpha)

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	t.Cleanup(func() {
		_ = runner.Stop("alpha", StopOptions{})
	})

	result, err := runner.Start("alpha", StartOptions{BestEffort: true})
	if err != nil {
		t.Fatalf("Start(best-effort): %v", err)
	}
	if got := result.FailedDependencies; len(got) != 1 || got[0] != "missing-beta" {
		t.Fatalf("failed dependencies = %#v, want [missing-beta]", got)
	}

	data, err := os.ReadFile(process.ScenarioDegradedPath(home, "alpha"))
	if err != nil {
		t.Fatalf("read degraded state: %v", err)
	}
	if !strings.Contains(string(data), `"status": "degraded"`) {
		t.Fatalf("degraded payload missing status: %s", data)
	}
	if !strings.Contains(string(data), `"missing-beta"`) {
		t.Fatalf("degraded payload missing dependency name: %s", data)
	}
}

func TestRunnerStartRejectsCircularDependencies(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	alpha := lifecycleFixtureManifest("alpha")
	alpha.Dependencies.Scenarios = map[string]scenario.Dependency{
		"beta": {Required: true},
	}
	writeLifecycleFixtureManifest(t, root, alpha)

	beta := lifecycleFixtureManifest("beta")
	beta.Dependencies.Scenarios = map[string]scenario.Dependency{
		"alpha": {Required: true},
	}
	writeLifecycleFixtureManifest(t, root, beta)

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	_, err = runner.Start("alpha", StartOptions{})
	if err == nil {
		t.Fatalf("expected circular dependency to fail start")
	}
	if !strings.Contains(err.Error(), "circular scenario dependency detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStepConditionsMetSupportsFilesystemEnvAndJSONChecks(t *testing.T) {
	root := t.TempDir()

	item := scenario.Scenario{
		Slug: "alpha",
		Path: root,
		Manifest: scenario.ServiceManifest{
			Dependencies: scenario.Dependencies{
				Resources: map[string]scenario.Dependency{
					"postgres": {Enabled: true},
				},
			},
		},
	}

	if err := os.MkdirAll(filepath.Join(root, "data"), 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	configPath := filepath.Join(root, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"services":[{"name":"api"}]}`), 0o644); err != nil {
		t.Fatalf("write %s: %v", configPath, err)
	}

	t.Setenv("EXTERNAL_ONLY", "set")

	condition := &scenario.Condition{
		FileExists:      "config.json",
		DirectoryExists: "data",
		JSONPathExists:  "config.json:services.0.name",
		ResourceEnabled: "postgres",
		CommandExists:   "sh",
		BinaryExists:    "sh",
		EnvVarSet:       "EXTERNAL_ONLY",
	}

	ok, reason, err := stepConditionsMet(item, condition, nil)
	if err != nil {
		t.Fatalf("stepConditionsMet: %v", err)
	}
	if !ok || reason != "" {
		t.Fatalf("expected condition to pass, ok=%v reason=%q", ok, reason)
	}

	ok, reason, err = stepConditionsMet(item, &scenario.Condition{Always: "false"}, nil)
	if err != nil {
		t.Fatalf("stepConditionsMet(always=false): %v", err)
	}
	if ok || reason != "step disabled by always=false" {
		t.Fatalf("always=false => ok=%v reason=%q", ok, reason)
	}

	ok, reason, err = stepConditionsMet(item, &scenario.Condition{FileNotExists: "config.json"}, nil)
	if err != nil {
		t.Fatalf("stepConditionsMet(file_not_exists): %v", err)
	}
	if ok || !strings.Contains(reason, "must not exist") {
		t.Fatalf("file_not_exists => ok=%v reason=%q", ok, reason)
	}

	found, err := jsonPathExists(configPath, "config.json:services.0.name")
	if err != nil {
		t.Fatalf("jsonPathExists: %v", err)
	}
	if !found {
		t.Fatalf("expected JSON path to exist")
	}

	found, err = jsonPathExists(configPath, "config.json:services.1.name")
	if err != nil {
		t.Fatalf("jsonPathExists missing path: %v", err)
	}
	if found {
		t.Fatalf("expected missing JSON path to return false")
	}
}

func TestStepConditionsMetRejectsDisabledResourceAndInvalidJSON(t *testing.T) {
	root := t.TempDir()

	item := scenario.Scenario{
		Slug: "alpha",
		Path: root,
		Manifest: scenario.ServiceManifest{
			Dependencies: scenario.Dependencies{
				Resources: map[string]scenario.Dependency{
					"postgres": {Enabled: false},
				},
			},
		},
	}

	ok, reason, err := stepConditionsMet(item, &scenario.Condition{ResourceEnabled: "postgres"}, nil)
	if err != nil {
		t.Fatalf("stepConditionsMet(disabled resource): %v", err)
	}
	if ok || !strings.Contains(reason, `resource "postgres" is disabled`) {
		t.Fatalf("disabled resource => ok=%v reason=%q", ok, reason)
	}

	brokenJSONPath := filepath.Join(root, "broken.json")
	if err := os.WriteFile(brokenJSONPath, []byte("{broken"), 0o644); err != nil {
		t.Fatalf("write %s: %v", brokenJSONPath, err)
	}

	if _, _, err := stepConditionsMet(item, &scenario.Condition{JSONPathExists: "broken.json:services.0.name"}, nil); err == nil {
		t.Fatalf("expected invalid JSON path source to fail")
	}
}

func TestCLINeedsSetupDetectsMissingAndStaleBinary(t *testing.T) {
	appRoot := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	needed, reason, err := cliNeedsSetup(appRoot, scenario.ConditionCheck{Command: "fixture-cli"})
	if err != nil {
		t.Fatalf("cliNeedsSetup missing binary: %v", err)
	}
	if !needed || reason != "CLI not installed: fixture-cli" {
		t.Fatalf("missing binary => needed=%v reason=%q", needed, reason)
	}

	cliSourceDir := filepath.Join(appRoot, "cli")
	if err := os.MkdirAll(cliSourceDir, 0o755); err != nil {
		t.Fatalf("mkdir cli source dir: %v", err)
	}
	sourcePath := filepath.Join(cliSourceDir, "main.go")
	if err := os.WriteFile(sourcePath, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", sourcePath, err)
	}
	cliPath := filepath.Join(binDir, "fixture-cli")
	if err := os.WriteFile(cliPath, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write %s: %v", cliPath, err)
	}

	old := time.Now().Add(-2 * time.Minute)
	now := time.Now()
	if err := os.Chtimes(sourcePath, old, old); err != nil {
		t.Fatalf("chtimes source: %v", err)
	}
	if err := os.Chtimes(cliPath, now, now); err != nil {
		t.Fatalf("chtimes cli: %v", err)
	}

	needed, reason, err = cliNeedsSetup(appRoot, scenario.ConditionCheck{Command: "fixture-cli"})
	if err != nil {
		t.Fatalf("cliNeedsSetup fresh binary: %v", err)
	}
	if needed {
		t.Fatalf("expected fresh CLI binary to satisfy setup, reason=%q", reason)
	}

	future := time.Now().Add(2 * time.Minute)
	if err := os.Chtimes(sourcePath, future, future); err != nil {
		t.Fatalf("chtimes source newer: %v", err)
	}

	needed, reason, err = cliNeedsSetup(appRoot, scenario.ConditionCheck{Command: "fixture-cli"})
	if err != nil {
		t.Fatalf("cliNeedsSetup stale binary: %v", err)
	}
	if !needed || reason != "CLI not installed: fixture-cli" {
		t.Fatalf("stale binary => needed=%v reason=%q", needed, reason)
	}
}

func TestUIBundleNeedsSetupTracksBundleFreshness(t *testing.T) {
	appRoot := t.TempDir()
	sourceDir := filepath.Join(appRoot, "ui", "src")
	bundlePath := filepath.Join(appRoot, "ui", "dist", "index.html")
	packageJSON := filepath.Join(appRoot, "ui", "package.json")

	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(bundlePath), 0o755); err != nil {
		t.Fatalf("mkdir dist: %v", err)
	}
	sourcePath := filepath.Join(sourceDir, "main.tsx")
	if err := os.WriteFile(sourcePath, []byte("export default 'hi';\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", sourcePath, err)
	}
	if err := os.WriteFile(bundlePath, []byte("<html></html>\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", bundlePath, err)
	}
	if err := os.WriteFile(packageJSON, []byte(`{"name":"fixture-ui","dependencies":{}}`), 0o644); err != nil {
		t.Fatalf("write %s: %v", packageJSON, err)
	}

	older := time.Now().Add(-2 * time.Minute)
	newer := time.Now()
	if err := os.Chtimes(sourcePath, older, older); err != nil {
		t.Fatalf("chtimes source: %v", err)
	}
	if err := os.Chtimes(packageJSON, older, older); err != nil {
		t.Fatalf("chtimes package.json: %v", err)
	}
	if err := os.Chtimes(bundlePath, newer, newer); err != nil {
		t.Fatalf("chtimes bundle: %v", err)
	}

	needed, reason, err := uiBundleNeedsSetup(appRoot, scenario.ConditionCheck{})
	if err != nil {
		t.Fatalf("uiBundleNeedsSetup fresh bundle: %v", err)
	}
	if needed {
		t.Fatalf("expected fresh bundle to satisfy setup, reason=%q", reason)
	}

	future := time.Now().Add(2 * time.Minute)
	if err := os.Chtimes(sourcePath, future, future); err != nil {
		t.Fatalf("chtimes source newer: %v", err)
	}

	needed, reason, err = uiBundleNeedsSetup(appRoot, scenario.ConditionCheck{})
	if err != nil {
		t.Fatalf("uiBundleNeedsSetup stale bundle: %v", err)
	}
	if !needed || !strings.Contains(reason, "UI bundle outdated") {
		t.Fatalf("stale bundle => needed=%v reason=%q", needed, reason)
	}
}

func TestRunExternalSetupCheckerUsesScriptExitCodes(t *testing.T) {
	root := t.TempDir()
	appRoot := t.TempDir()
	checkerDir := filepath.Join(root, "scripts", "lib", "setup-conditions")
	if err := os.MkdirAll(checkerDir, 0o755); err != nil {
		t.Fatalf("mkdir checker dir: %v", err)
	}

	checkerPath := filepath.Join(checkerDir, "custom-check.sh")
	if err := os.WriteFile(checkerPath, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write %s: %v", checkerPath, err)
	}

	needed, err := runExternalSetupChecker(root, appRoot, scenario.ConditionCheck{Type: "custom", Name: "fixture"})
	if err != nil {
		t.Fatalf("runExternalSetupChecker success: %v", err)
	}
	if !needed {
		t.Fatalf("expected exit code 0 to require setup")
	}

	if err := os.WriteFile(checkerPath, []byte("#!/usr/bin/env bash\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("rewrite %s: %v", checkerPath, err)
	}

	needed, err = runExternalSetupChecker(root, appRoot, scenario.ConditionCheck{Type: "custom", Name: "fixture"})
	if err != nil {
		t.Fatalf("runExternalSetupChecker exit 1: %v", err)
	}
	if needed {
		t.Fatalf("expected exit code 1 to mean setup not required")
	}
}

func TestEvaluateSetupCheckSupportsFilesystemStateChecks(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecyclePortRegistry(t, root)

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	item := scenario.Scenario{Slug: "alpha", Path: root}

	needed, reason, err := runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "resources", Populated: true})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(resources missing): %v", err)
	}
	if !needed || reason != "Resources not populated" {
		t.Fatalf("resources missing => needed=%v reason=%q", needed, reason)
	}
	if err := os.MkdirAll(filepath.Join(root, "data"), 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "data", ".resources-populated"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write resources marker: %v", err)
	}
	needed, _, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "resources", Populated: true})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(resources ready): %v", err)
	}
	if needed {
		t.Fatalf("expected populated resources marker to satisfy setup")
	}

	uiDir := filepath.Join(root, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("mkdir ui: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"name":"fixture"}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	needed, reason, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "dependencies", Paths: []string{"ui/package.json"}})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(dependencies missing): %v", err)
	}
	if !needed || reason != "Dependencies not installed" {
		t.Fatalf("dependencies missing => needed=%v reason=%q", needed, reason)
	}
	if err := os.MkdirAll(filepath.Join(uiDir, "node_modules"), 0o755); err != nil {
		t.Fatalf("mkdir node_modules: %v", err)
	}
	needed, _, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "dependencies", Paths: []string{"ui/package.json"}})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(dependencies ready): %v", err)
	}
	if needed {
		t.Fatalf("expected node_modules to satisfy dependency check")
	}

	cacheDir := filepath.Join(root, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	needed, reason, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "data", Path: "cache"})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(data empty): %v", err)
	}
	if !needed || reason != "Data directory missing" {
		t.Fatalf("data empty => needed=%v reason=%q", needed, reason)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "seed.txt"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write cache file: %v", err)
	}
	needed, _, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "data", Path: "cache"})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(data populated): %v", err)
	}
	if needed {
		t.Fatalf("expected populated data dir to satisfy setup")
	}

	needed, reason, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "files", Paths: []string{"config/app.yaml"}})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(files missing): %v", err)
	}
	if !needed || reason != "Required files missing" {
		t.Fatalf("files missing => needed=%v reason=%q", needed, reason)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "app.yaml"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write app.yaml: %v", err)
	}
	needed, _, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "files", Paths: []string{"config/app.yaml"}})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(files ready): %v", err)
	}
	if needed {
		t.Fatalf("expected present file to satisfy setup")
	}

	needed, reason, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "directories", Targets: []string{"runtime"}})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(directories missing): %v", err)
	}
	if !needed || reason != "Missing directories" {
		t.Fatalf("directories missing => needed=%v reason=%q", needed, reason)
	}
	if err := os.MkdirAll(filepath.Join(root, "runtime"), 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	needed, _, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "directories", Targets: []string{"runtime"}})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(directories ready): %v", err)
	}
	if needed {
		t.Fatalf("expected present directory to satisfy setup")
	}
}

func TestResourceAndDependencyChecksCoverMarkersAndToolchains(t *testing.T) {
	root := t.TempDir()

	if !resourcesNeedSetup(root, scenario.ConditionCheck{Resources: []string{"postgres", "redis"}}) {
		t.Fatalf("expected missing resource markers to require setup")
	}
	if err := os.MkdirAll(filepath.Join(root, "data"), 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "data", ".postgres-populated"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write postgres marker: %v", err)
	}
	if !resourcesNeedSetup(root, scenario.ConditionCheck{Resources: []string{"postgres", "redis"}}) {
		t.Fatalf("expected missing redis marker to keep setup required")
	}
	if err := os.WriteFile(filepath.Join(root, "data", ".redis-populated"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write redis marker: %v", err)
	}
	if resourcesNeedSetup(root, scenario.ConditionCheck{Resources: []string{"postgres", "redis"}}) {
		t.Fatalf("expected all resource markers to satisfy setup")
	}

	goDir := filepath.Join(root, "go-worker")
	if err := os.MkdirAll(goDir, 0o755); err != nil {
		t.Fatalf("mkdir go worker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(goDir, "go.mod"), []byte("module fixture\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if !dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"go-worker/go.mod"}}) {
		t.Fatalf("expected missing go.sum/vendor to require setup")
	}
	if err := os.MkdirAll(filepath.Join(goDir, "vendor"), 0o755); err != nil {
		t.Fatalf("mkdir vendor: %v", err)
	}
	if dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"go-worker/go.mod"}}) {
		t.Fatalf("expected vendor fallback to satisfy Go dependency check")
	}

	pythonDir := filepath.Join(root, "python-worker")
	if err := os.MkdirAll(pythonDir, 0o755); err != nil {
		t.Fatalf("mkdir python worker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pythonDir, "requirements.txt"), []byte("pytest\n"), 0o644); err != nil {
		t.Fatalf("write requirements.txt: %v", err)
	}
	if !dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"python-worker/requirements.txt"}}) {
		t.Fatalf("expected missing Python virtualenv to require setup")
	}
	if err := os.MkdirAll(filepath.Join(pythonDir, ".venv"), 0o755); err != nil {
		t.Fatalf("mkdir .venv: %v", err)
	}
	if dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"python-worker/requirements.txt"}}) {
		t.Fatalf("expected .venv to satisfy Python dependency check")
	}

	rustDir := filepath.Join(root, "rust-worker")
	if err := os.MkdirAll(rustDir, 0o755); err != nil {
		t.Fatalf("mkdir rust worker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rustDir, "Cargo.toml"), []byte("[package]\nname=\"fixture\"\nversion=\"0.1.0\"\n"), 0o644); err != nil {
		t.Fatalf("write Cargo.toml: %v", err)
	}
	if !dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"rust-worker/Cargo.toml"}}) {
		t.Fatalf("expected missing Rust target dir to require setup")
	}
	if err := os.MkdirAll(filepath.Join(rustDir, "target"), 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"rust-worker/Cargo.toml"}}) {
		t.Fatalf("expected target dir to satisfy Rust dependency check")
	}

	if !dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"config/missing.yaml"}}) {
		t.Fatalf("expected missing generic dependency path to require setup")
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "missing.yaml"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write generic dependency marker: %v", err)
	}
	if dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"config/missing.yaml"}}) {
		t.Fatalf("expected present generic dependency path to satisfy setup")
	}
}

func TestRunnerLoadScenarioSupportsCustomPathAndMissingScenario(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecyclePortRegistry(t, root)

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	customPath := t.TempDir()
	servicePath := filepath.Join(customPath, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(servicePath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(servicePath), err)
	}
	if err := os.WriteFile(servicePath, []byte(`{"version":"1.0.0","service":{"displayName":"Custom fixture"}}`), 0o644); err != nil {
		t.Fatalf("write %s: %v", servicePath, err)
	}

	item, err := runner.loadScenario("", customPath)
	if err != nil {
		t.Fatalf("loadScenario(custom): %v", err)
	}
	if item.Slug != filepath.Base(customPath) {
		t.Fatalf("slug = %q, want %q", item.Slug, filepath.Base(customPath))
	}
	if item.Path != customPath {
		t.Fatalf("path = %q, want %q", item.Path, customPath)
	}
	if item.ServicePath != servicePath {
		t.Fatalf("service path = %q, want %q", item.ServicePath, servicePath)
	}

	if _, err := runner.loadScenario("missing", ""); err == nil || !strings.Contains(err.Error(), `scenario "missing" not found`) {
		t.Fatalf("missing scenario error = %v", err)
	}
}

func TestWaitForHealthHonorsExplicitTimeoutAndDegradedState(t *testing.T) {
	runner := &Runner{}

	status, err := runner.WaitForHealth(scenario.Scenario{Slug: "alpha"}, nil)
	if err != nil {
		t.Fatalf("WaitForHealth(no checks): %v", err)
	}
	if status != "running" {
		t.Fatalf("status = %q, want running", status)
	}

	degradedItem := scenario.Scenario{
		Slug: "beta",
		Manifest: scenario.ServiceManifest{
			Ports: map[string]scenario.Port{
				"api": {EnvVar: "API_PORT"},
			},
			Lifecycle: scenario.Lifecycle{
				Health: &scenario.HealthConfig{
					Checks: []scenario.HealthCheck{
						{
							Name:     "api",
							Type:     "http",
							Target:   "http://127.0.0.1:${API_PORT}/health",
							Critical: false,
							Timeout:  25,
						},
					},
					Timeout:  25,
					Interval: 1,
				},
			},
		},
	}

	start := time.Now()
	status, err = runner.WaitForHealth(degradedItem, map[string]string{"API_PORT": "1"})
	if err != nil {
		t.Fatalf("WaitForHealth(degraded): %v", err)
	}
	if status != "degraded" {
		t.Fatalf("status = %q, want degraded", status)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("WaitForHealth ignored explicit timeout, elapsed=%s", elapsed)
	}

	unhealthyItem := scenario.Scenario{
		Slug: "gamma",
		Manifest: scenario.ServiceManifest{
			Lifecycle: scenario.Lifecycle{
				Health: &scenario.HealthConfig{
					Checks: []scenario.HealthCheck{
						{
							Name:     "api",
							Type:     "unsupported",
							Critical: true,
						},
					},
					Timeout:  25,
					Interval: 1,
				},
			},
		},
	}

	start = time.Now()
	status, err = runner.WaitForHealth(unhealthyItem, nil)
	if err == nil {
		t.Fatalf("expected unhealthy health checks to fail")
	}
	if status != "unhealthy" {
		t.Fatalf("status = %q, want unhealthy", status)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("WaitForHealth unhealthy timeout took too long: %s", elapsed)
	}
}

func TestRuntimePortsAndStrictHealthUseRecordedMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port
	item := scenario.Scenario{
		Slug: "alpha",
		Manifest: scenario.ServiceManifest{
			Ports: map[string]scenario.Port{
				"api": {EnvVar: "API_PORT"},
			},
			Lifecycle: scenario.Lifecycle{
				Health: &scenario.HealthConfig{
					Checks: []scenario.HealthCheck{
						{
							Name:     "api",
							Type:     "http",
							Target:   "http://127.0.0.1:${API_PORT}/",
							Critical: true,
							Timeout:  1000,
						},
					},
				},
			},
		},
	}

	runner := &Runner{}
	records := []process.Record{{
		PID:  os.Getpid(),
		Step: "start-api",
		Port: port,
	}}

	ports := runner.runtimePorts(item.Manifest, records)
	if ports["API_PORT"] != port {
		t.Fatalf("API_PORT = %d, want %d", ports["API_PORT"], port)
	}
	if !runner.isScenarioHealthyStrict(item, records) {
		t.Fatalf("expected live record and healthy endpoint to pass strict health")
	}
	if runner.isScenarioHealthyStrict(item, nil) {
		t.Fatalf("expected empty runtime to fail strict health")
	}
}

func TestExecutePhaseAppendsTestArgsAndWarnsOnStopFailure(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecyclePortRegistry(t, root)

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	item := scenario.Scenario{
		Slug: "alpha",
		Path: root,
		Manifest: scenario.ServiceManifest{
			Lifecycle: scenario.Lifecycle{
				Test: scenario.Phase{
					Steps: []scenario.PhaseStep{
						{
							Name: "skip-me",
							Run:  "printf 'nope\\n' > skipped.txt",
							Condition: &scenario.Condition{
								Always: "false",
							},
						},
						{
							Name: "write-args",
							Run:  "printf '%s\\n' > args.txt",
						},
					},
				},
				Stop: scenario.Phase{
					Steps: []scenario.PhaseStep{
						{
							Name: "failing-stop",
							Run:  "exit 7",
						},
					},
				},
			},
		},
	}

	var testLog bytes.Buffer
	if err := runner.ExecutePhase(item, "test", nil, []string{"phase-a", "two words"}, &testLog); err != nil {
		t.Fatalf("ExecutePhase(test): %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "skipped.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected skipped step to avoid writing file, stat err=%v", err)
	}
	argsData, err := os.ReadFile(filepath.Join(root, "args.txt"))
	if err != nil {
		t.Fatalf("read args.txt: %v", err)
	}
	if got := string(argsData); got != "phase-a\ntwo words\n" {
		t.Fatalf("args.txt = %q", got)
	}
	if !strings.Contains(testLog.String(), "Skipping skip-me - step disabled by always=false") {
		t.Fatalf("expected skip log, got %q", testLog.String())
	}

	var stopLog bytes.Buffer
	if err := runner.ExecutePhase(item, "stop", nil, nil, &stopLog); err != nil {
		t.Fatalf("ExecutePhase(stop): %v", err)
	}
	if !strings.Contains(stopLog.String(), "[WARNING] Stop step completed with non-zero exit: failing-stop") {
		t.Fatalf("expected stop warning log, got %q", stopLog.String())
	}
}

func TestEvaluateSetupCheckUsesExternalCheckerReason(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecyclePortRegistry(t, root)

	checker := filepath.Join(root, "scripts", "lib", "setup-conditions", "custom-check.sh")
	if err := os.MkdirAll(filepath.Dir(checker), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(checker), err)
	}
	if err := os.WriteFile(checker, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write %s: %v", checker, err)
	}

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	needed, reason, err := runner.evaluateSetupCheck(scenario.Scenario{Slug: "alpha", Path: root}, scenario.ConditionCheck{Type: "custom"})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(custom): %v", err)
	}
	if !needed || reason != "Check failed: custom" {
		t.Fatalf("needed/reason = %v/%q", needed, reason)
	}
}

func TestLocalReplacePathsSupportsInlineAndBlockForms(t *testing.T) {
	goModPath := filepath.Join(t.TempDir(), "go.mod")
	data := `module fixture

go 1.23.0

replace github.com/example/alpha => ../alpha

replace (
	github.com/example/beta => ../beta
	github.com/example/gamma => ../gamma
)
`
	if err := os.WriteFile(goModPath, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", goModPath, err)
	}

	paths, err := localReplacePaths(goModPath)
	if err != nil {
		t.Fatalf("localReplacePaths: %v", err)
	}
	if got := strings.Join(paths, ","); got != "../alpha,../beta,../gamma" {
		t.Fatalf("paths = %q", got)
	}
}

func TestListeningPIDsDetectsLiveListener(t *testing.T) {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof is not installed")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	found := false
	for attempt := 0; attempt < 10; attempt++ {
		pids, err := listeningPIDs(port)
		if err != nil {
			t.Fatalf("listeningPIDs: %v", err)
		}
		for _, pid := range pids {
			if pid == os.Getpid() {
				found = true
				break
			}
		}
		if found {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !found {
		t.Fatalf("expected current pid %d to own listener on port %d", os.Getpid(), port)
	}
}

func writeLifecycleFixture(t *testing.T, root, name string) {
	t.Helper()
	writeLifecycleFixtureManifest(t, root, lifecycleFixtureManifest(name))
}

func writeLifecycleFixtureManifest(t *testing.T, root string, manifest scenario.ServiceManifest) {
	t.Helper()

	if strings.TrimSpace(manifest.Service.Name) == "" {
		t.Fatalf("fixture manifest is missing service name")
	}

	writeLifecyclePortRegistry(t, root)

	scenarioDir := filepath.Join(root, "scenarios", manifest.Service.Name)
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir scenario dir: %v", err)
	}
	servicePath := filepath.Join(scenarioDir, ".vrooli", "service.json")
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture manifest: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(servicePath, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", servicePath, err)
	}
}

func lifecycleFixtureManifest(name string) scenario.ServiceManifest {
	return scenario.ServiceManifest{
		Version: "1.0.0",
		Service: scenario.ServiceMetadata{
			Name:        name,
			DisplayName: "Lifecycle " + name,
			Description: "Lifecycle validation fixture",
			Version:     "0.1.0",
		},
		Ports: map[string]scenario.Port{
			"api": {
				EnvVar: "API_PORT",
				Range:  "22000-22010",
			},
		},
		Lifecycle: scenario.Lifecycle{
			Version: "2.0.0",
			Health: &scenario.HealthConfig{
				Checks: []scenario.HealthCheck{
					{
						Name:     "api",
						Type:     "http",
						Target:   "http://127.0.0.1:${API_PORT}/health",
						Critical: true,
						Timeout:  1000,
					},
				},
				StartupGracePeriod: 250,
				Timeout:            5000,
				Interval:           250,
			},
			Setup: scenario.Phase{
				Condition: &scenario.Condition{
					Checks: []scenario.ConditionCheck{
						{
							Type:    "binaries",
							Targets: []string{"api/mock-api"},
						},
					},
				},
				Steps: []scenario.PhaseStep{
					{
						Name: "build-api",
						Run:  "mkdir -p api public && printf 'package main\\n' > api/handler.go && printf '#!/usr/bin/env bash\\npython3 -m http.server \"$API_PORT\" --bind 127.0.0.1 --directory ../public\\n' > api/mock-api && chmod +x api/mock-api && printf 'ok\\n' > public/health",
					},
				},
			},
			Develop: scenario.Phase{
				Steps: []scenario.PhaseStep{
					{
						Name:       "start-api",
						Run:        "cd api && ./mock-api",
						Background: true,
						Condition: &scenario.Condition{
							FileExists: "api/mock-api",
						},
					},
				},
			},
		},
	}
}

func writeLifecyclePortRegistry(t *testing.T, root string) {
	t.Helper()

	portRegistry := filepath.Join(root, "scripts", "resources", "port_registry.sh")
	if err := os.MkdirAll(filepath.Dir(portRegistry), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(portRegistry), err)
	}
	if err := os.WriteFile(portRegistry, []byte("#!/usr/bin/env bash\ndeclare -g -A RESOURCE_PORTS=()\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", portRegistry, err)
	}
	portRegistryJSON := filepath.Join(root, "scripts", "resources", "port_registry.json")
	if err := os.WriteFile(portRegistryJSON, []byte("{\n  \"resource_ports\": {},\n  \"reserved_ranges\": {}\n}\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", portRegistryJSON, err)
	}
}
