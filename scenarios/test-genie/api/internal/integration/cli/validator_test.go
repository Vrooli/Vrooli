package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidator_Success(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(h.successExecutor),
		WithCapture(h.versionCapture("demo version 1.0.0")),
	)

	result := v.Validate(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.BinaryPath == "" {
		t.Error("expected binary path to be set")
	}
	if !strings.Contains(result.VersionOutput, "version") {
		t.Errorf("expected version output, got: %s", result.VersionOutput)
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}
}

func TestValidator_NoCLIDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "scenarios", "demo")
	// Don't create cli directory

	v := New(
		Config{
			ScenarioDir:  scenarioDir,
			ScenarioName: "demo",
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(versionCapture("demo version 1.0.0")),
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when cli directory missing")
	}
	if result.FailureClass != "misconfiguration" {
		t.Errorf("expected misconfiguration, got: %s", result.FailureClass)
	}
}

func TestValidator_NoExecutableBinary(t *testing.T) {
	h := newTestHarness(t)
	// Remove executable
	os.Remove(filepath.Join(h.scenarioDir, "cli", h.scenarioName))

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(versionCapture("demo version 1.0.0")),
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when no executable found")
	}
	if !strings.Contains(result.Error.Error(), "no executable CLI binary") {
		t.Errorf("expected 'no executable' error, got: %v", result.Error)
	}
}

func TestValidator_HelpCommandFails(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(func(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
			if len(args) > 0 && args[0] == "help" {
				return errors.New("help command failed")
			}
			return nil
		}),
		WithCapture(versionCapture("demo version 1.0.0")),
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when help command fails")
	}
	if !strings.Contains(result.Error.Error(), "help command failed") {
		t.Errorf("expected help error, got: %v", result.Error)
	}
	if result.FailureClass != "system" {
		t.Errorf("expected system failure class, got: %s", result.FailureClass)
	}
}

func TestValidator_VersionCommandFails(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(func(ctx context.Context, dir string, w io.Writer, name string, args ...string) (string, error) {
			return "", errors.New("version command failed")
		}),
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when version command fails")
	}
	if !strings.Contains(result.Error.Error(), "version command failed") {
		t.Errorf("expected version error, got: %v", result.Error)
	}
}

func TestValidator_VersionOutputMalformed(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(versionCapture("1.0.0")), // Missing "version" word
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when version output malformed")
	}
	if !strings.Contains(result.Error.Error(), "malformed") {
		t.Errorf("expected malformed error, got: %v", result.Error)
	}
	if result.FailureClass != "misconfiguration" {
		t.Errorf("expected misconfiguration, got: %s", result.FailureClass)
	}
}

func TestValidator_ContextCancelled(t *testing.T) {
	h := newTestHarness(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(versionCapture("demo version 1.0.0")),
	)

	result := v.Validate(ctx)

	if result.Success {
		t.Fatal("expected failure when context cancelled")
	}
	if result.FailureClass != "system" {
		t.Errorf("expected system failure class, got: %s", result.FailureClass)
	}
}

func TestValidator_FindsScenarioNamedBinary(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(versionCapture("demo version 1.0.0")),
	)

	result := v.Validate(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	expectedPath := filepath.Join(h.scenarioDir, "cli", h.scenarioName)
	if result.BinaryPath != expectedPath {
		t.Errorf("expected binary path %s, got: %s", expectedPath, result.BinaryPath)
	}
}

func TestValidator_FindsFallbackBinary(t *testing.T) {
	h := newTestHarness(t)
	// Remove scenario-named binary, add a different one
	os.Remove(filepath.Join(h.scenarioDir, "cli", h.scenarioName))
	writeExecutable(t, filepath.Join(h.scenarioDir, "cli", "other-cli"), "#!/bin/bash\necho cli")

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(versionCapture("other version 1.0.0")),
	)

	result := v.Validate(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if !strings.Contains(result.BinaryPath, "other-cli") {
		t.Errorf("expected fallback binary, got: %s", result.BinaryPath)
	}
}

func TestValidator_NoExecutorConfigured(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		// No executor set
		WithCapture(versionCapture("demo version 1.0.0")),
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when executor not configured")
	}
	if !strings.Contains(result.Error.Error(), "executor not configured") {
		t.Errorf("expected executor error, got: %v", result.Error)
	}
}

func TestValidator_NoCaptureConfigured(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		// No capture set
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when capture not configured")
	}
	if !strings.Contains(result.Error.Error(), "capture not configured") {
		t.Errorf("expected capture error, got: %v", result.Error)
	}
}

// Test helpers

type testHarness struct {
	scenarioDir  string
	scenarioName string
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()
	tmpDir := t.TempDir()
	scenarioName := "demo"
	scenarioDir := filepath.Join(tmpDir, "scenarios", scenarioName)

	// Create cli directory
	cliDir := filepath.Join(scenarioDir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}

	// Create executable
	writeExecutable(t, filepath.Join(cliDir, scenarioName), "#!/bin/bash\necho cli")

	return &testHarness{
		scenarioDir:  scenarioDir,
		scenarioName: scenarioName,
	}
}

func (h *testHarness) successExecutor(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
	return nil
}

func (h *testHarness) versionCapture(output string) CommandCapture {
	return func(ctx context.Context, dir string, w io.Writer, name string, args ...string) (string, error) {
		return output, nil
	}
}

func successExecutor(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
	return nil
}

func versionCapture(output string) CommandCapture {
	return func(ctx context.Context, dir string, w io.Writer, name string, args ...string) (string, error) {
		return output, nil
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	perm := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		perm = 0o644
	}
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatalf("failed to write executable: %v", err)
	}
}

// Ensure validator implements interface at compile time
var _ Validator = (*validator)(nil)
