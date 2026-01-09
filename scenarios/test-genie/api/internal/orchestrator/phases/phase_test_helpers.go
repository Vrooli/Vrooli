package phases

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type commandExecFunc func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error
type commandCaptureFunc func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error)

func stubPhaseCommandExecutor(t *testing.T, fn commandExecFunc) {
	t.Helper()
	prev := phaseCommandExecutor
	phaseCommandExecutor = func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
		return fn(ctx, dir, logWriter, name, args...)
	}
	t.Cleanup(func() {
		phaseCommandExecutor = prev
	})
}

func stubPhaseCommandCapture(t *testing.T, fn commandCaptureFunc) {
	t.Helper()
	prev := phaseCommandCapture
	phaseCommandCapture = func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
		return fn(ctx, dir, logWriter, name, args...)
	}
	t.Cleanup(func() {
		phaseCommandCapture = prev
	})
}

func createScenarioLayout(t *testing.T, root, name string) string {
	t.Helper()
	scenarioDir := filepath.Join(root, name)
	requiredDirs := []string{
		"api",
		"cli",
		filepath.Join("cli", "test"),
		"requirements",
		"ui",
		"docs",
		"test",
		".vrooli",
	}
	for _, rel := range requiredDirs {
		if err := os.MkdirAll(filepath.Join(scenarioDir, rel), 0o755); err != nil {
			t.Fatalf("failed to create %s: %v", rel, err)
		}
	}
	cliScript := func(name string) []byte {
		return []byte(fmt.Sprintf(`#!/usr/bin/env bash
# Handle no arguments - print help
if [ -z "$1" ]; then
  echo "usage: %s <cmd>"
  exit 0
fi
# Handle known commands
case "$1" in
  version|--version|-v)
    echo "%s version 1.0.0"
    exit 0
    ;;
  help|--help|-h)
    echo "usage: %s <cmd>"
    exit 0
    ;;
  *)
    # Unknown command - return error
    echo "error: unknown command '$1'" >&2
    exit 1
    ;;
esac
`, name, name, name))
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", name), cliScript(name), 0o755); err != nil {
		t.Fatalf("failed to seed scenario cli binary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", name+".bats"), []byte("#!/usr/bin/env bats\n"), 0o644); err != nil {
		t.Fatalf("failed to seed scenario bats file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), []byte(`{"service":{"name":"`+name+`"}}`), 0o644); err != nil {
		t.Fatalf("failed to seed service.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "testing.json"), []byte(`{"structure":{"ui_smoke":{"enabled":false}}}`), 0o644); err != nil {
		t.Fatalf("failed to seed testing.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "requirements", "index.json"), []byte(`{"imports":["01-internal-orchestrator/module.json"]}`), 0o644); err != nil {
		t.Fatalf("failed to seed requirements index: %v", err)
	}
	moduleDir := filepath.Join(scenarioDir, "requirements", "01-internal-orchestrator")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}
	modulePayload := `{"requirements":[{"id":"REQ-1","title":"Seed","criticality":"p1","status":"draft","validation":[{"type":"manual","ref":"docs"}]}]}`
	if err := os.WriteFile(filepath.Join(moduleDir, "module.json"), []byte(modulePayload), 0o644); err != nil {
		t.Fatalf("failed to seed module.json: %v", err)
	}
	return scenarioDir
}
