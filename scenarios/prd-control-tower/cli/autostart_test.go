package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestEnsureScenarioAPIReadyNoopWhenExplicitAPIBaseOverride(t *testing.T) {
	app := newTestApp(t)
	startScenarioFunc = func(ctx context.Context, scenarioName string) error {
		t.Fatalf("startScenarioFunc should not be called")
		return nil
	}
	healthCheckFunc = func(ctx context.Context, base string) error {
		t.Fatalf("healthCheckFunc should not be called")
		return nil
	}
	t.Cleanup(resetAutoStartHooks)

	err := ensureScenarioAPIReady(app.core, cliapp.GlobalOptions{APIBaseOverride: "http://example.com"}, appName)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestEnsureScenarioAPIReadyStartsWhenHealthFails(t *testing.T) {
	app := newTestApp(t)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", "http://127.0.0.1:1")

	started := false
	startScenarioFunc = func(ctx context.Context, scenarioName string) error {
		started = true
		return nil
	}
	canAutostartFunc = func(scenarioName string) bool {
		return true
	}
	attempt := 0
	healthCheckFunc = func(ctx context.Context, base string) error {
		attempt++
		if attempt == 1 {
			return errors.New("down")
		}
		return nil
	}
	t.Cleanup(resetAutoStartHooks)

	err := ensureScenarioAPIReady(app.core, cliapp.GlobalOptions{}, appName)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !started {
		t.Fatalf("expected scenario to be started")
	}
}

func TestCanAutostartScenarioUsesContractScenarioPath(t *testing.T) {
	root := newCLIContractFixtureRepo(t)
	scenarioPath := filepath.Join(root, "scenarios", appName)
	if err := os.MkdirAll(filepath.Join(scenarioPath, "api"), 0o755); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioPath, "ui", "dist"), 0o755); err != nil {
		t.Fatalf("mkdir ui dist: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioPath, "api", appName+"-api"), []byte("binary"), 0o755); err != nil {
		t.Fatalf("write api binary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioPath, "ui", "dist", "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("write ui bundle: %v", err)
	}

	t.Setenv("VROOLI_ROOT", filepath.Join(root, "scenarios", appName, "cli"))

	if !canAutostartScenario(appName) {
		t.Fatalf("expected canAutostartScenario(%q) to succeed", appName)
	}
}

func TestDetectEntityTypeFromRepoUsesContractRoot(t *testing.T) {
	root := newCLIContractFixtureRepo(t)
	if err := os.MkdirAll(filepath.Join(root, "resources", "redis"), 0o755); err != nil {
		t.Fatalf("mkdir resource: %v", err)
	}
	t.Setenv("VROOLI_ROOT", filepath.Join(root, "scenarios", appName, "cli"))

	if got := detectEntityTypeFromRepo("redis"); got != "resource" {
		t.Fatalf("detectEntityTypeFromRepo() = %q, want resource", got)
	}
}

func resetAutoStartHooks() {
	startScenarioFunc = startScenarioViaVrooli
	healthCheckFunc = checkHealth
	canAutostartFunc = canAutostartScenario
}

func newCLIContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoRoot := cliRepoRoot(t)

	contractData, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo contract: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contractData, 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/prd-cli-test\n\ngo 1.22.3\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, dir := range []string{"templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return root
}

func cliRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}
