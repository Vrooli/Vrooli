package phases

import (
	"context"
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
		filepath.Join("test", "phases"),
		filepath.Join("test", "lib"),
		".vrooli",
	}
	for _, rel := range requiredDirs {
		if err := os.MkdirAll(filepath.Join(scenarioDir, rel), 0o755); err != nil {
			t.Fatalf("failed to create %s: %v", rel, err)
		}
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "test", "run-tests.sh"), []byte("echo ok"), 0o755); err != nil {
		t.Fatalf("failed to seed run-tests.sh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "test-genie"), []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("failed to seed cli binary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "test-genie.bats"), []byte("#!/usr/bin/env bats\n"), 0o644); err != nil {
		t.Fatalf("failed to seed cli bats file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "test", "lib", "runtime.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("failed to seed runtime script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "test", "lib", "orchestrator.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("failed to seed orchestrator script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), []byte(`{"service":{"name":"`+name+`"}}`), 0o644); err != nil {
		t.Fatalf("failed to seed service.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "testing.json"), []byte(`{"structure":{}}`), 0o644); err != nil {
		t.Fatalf("failed to seed testing.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "requirements", "index.json"), []byte(`{"modules":["01-internal-orchestrator"]}`), 0o644); err != nil {
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
