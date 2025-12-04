package golang

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/unit/types"
)

// mockExecutor is a test double for CommandExecutor.
type mockExecutor struct {
	lookPathFunc func(name string) (string, error)
	runFunc      func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error
	captureFunc  func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error)
}

func (m *mockExecutor) LookPath(name string) (string, error) {
	if m.lookPathFunc != nil {
		return m.lookPathFunc(name)
	}
	return "/usr/bin/" + name, nil
}

func (m *mockExecutor) Run(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
	if m.runFunc != nil {
		return m.runFunc(ctx, dir, logWriter, name, args...)
	}
	return nil
}

func (m *mockExecutor) Capture(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
	if m.captureFunc != nil {
		return m.captureFunc(ctx, dir, logWriter, name, args...)
	}
	return "", nil
}

func TestRunner_Name(t *testing.T) {
	r := New(Config{ScenarioDir: "/tmp/test"})
	if got := r.Name(); got != "go" {
		t.Errorf("Name() = %q, want %q", got, "go")
	}
}

func TestRunner_Detect_WithGoMod(t *testing.T) {
	dir := t.TempDir()
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	r := New(Config{ScenarioDir: dir})
	if !r.Detect() {
		t.Error("Detect() = false, want true when go.mod exists")
	}
}

func TestRunner_Detect_WithoutGoMod(t *testing.T) {
	dir := t.TempDir()
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	r := New(Config{ScenarioDir: dir})
	if r.Detect() {
		t.Error("Detect() = true, want false when go.mod is missing")
	}
}

func TestRunner_Detect_NoApiDir(t *testing.T) {
	dir := t.TempDir()

	r := New(Config{ScenarioDir: dir})
	if r.Detect() {
		t.Error("Detect() = true, want false when api/ dir is missing")
	}
}

func TestRunner_Run_Success(t *testing.T) {
	dir := t.TempDir()
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	executor := &mockExecutor{}
	r := New(Config{
		ScenarioDir: dir,
		Executor:    executor,
	})

	result := r.Run(context.Background())
	if !result.Success {
		t.Errorf("Run() success = false, want true; error: %v", result.Error)
	}
	if len(result.Observations) == 0 {
		t.Error("Run() returned no observations")
	}
}

func TestRunner_Run_MissingApiDir(t *testing.T) {
	dir := t.TempDir()

	executor := &mockExecutor{}
	r := New(Config{
		ScenarioDir: dir,
		Executor:    executor,
	})

	result := r.Run(context.Background())
	if result.Success {
		t.Error("Run() success = true, want false when api/ is missing")
	}
	if result.FailureClass != types.FailureClassMisconfiguration {
		t.Errorf("FailureClass = %q, want %q", result.FailureClass, types.FailureClassMisconfiguration)
	}
}

func TestRunner_Run_GoCommandMissing(t *testing.T) {
	dir := t.TempDir()
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	executor := &mockExecutor{
		lookPathFunc: func(name string) (string, error) {
			if name == "go" {
				return "", errors.New("not found")
			}
			return "/usr/bin/" + name, nil
		},
	}
	r := New(Config{
		ScenarioDir: dir,
		Executor:    executor,
	})

	result := r.Run(context.Background())
	if result.Success {
		t.Error("Run() success = true, want false when go is missing")
	}
	if result.FailureClass != types.FailureClassMissingDependency {
		t.Errorf("FailureClass = %q, want %q", result.FailureClass, types.FailureClassMissingDependency)
	}
}

func TestRunner_Run_TestFailure(t *testing.T) {
	dir := t.TempDir()
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	executor := &mockExecutor{
		runFunc: func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			if name == "go" && len(args) > 0 && args[0] == "test" {
				return errors.New("exit status 1")
			}
			return nil
		},
	}
	r := New(Config{
		ScenarioDir: dir,
		Executor:    executor,
	})

	result := r.Run(context.Background())
	if result.Success {
		t.Error("Run() success = true, want false when tests fail")
	}
	if result.FailureClass != types.FailureClassTestFailure {
		t.Errorf("FailureClass = %q, want %q", result.FailureClass, types.FailureClassTestFailure)
	}
	if !strings.Contains(result.Error.Error(), "go test") {
		t.Errorf("Error = %q, want to contain 'go test'", result.Error.Error())
	}
}

func TestRunner_Run_ContextCancelled(t *testing.T) {
	dir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := New(Config{ScenarioDir: dir})
	result := r.Run(ctx)

	if result.Success {
		t.Error("Run() success = true, want false when context is cancelled")
	}
	if result.FailureClass != types.FailureClassSystem {
		t.Errorf("FailureClass = %q, want %q", result.FailureClass, types.FailureClassSystem)
	}
}

func TestRunner_Run_CommandArgs(t *testing.T) {
	dir := t.TempDir()
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	var capturedDir string
	var capturedName string
	var capturedArgs []string

	executor := &mockExecutor{
		runFunc: func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			capturedDir = dir
			capturedName = name
			capturedArgs = args
			return nil
		},
	}
	r := New(Config{
		ScenarioDir: dir,
		Executor:    executor,
	})

	r.Run(context.Background())

	if capturedDir != apiDir {
		t.Errorf("command dir = %q, want %q", capturedDir, apiDir)
	}
	if capturedName != "go" {
		t.Errorf("command name = %q, want %q", capturedName, "go")
	}
	expectedArgs := []string{"test", "./..."}
	if len(capturedArgs) != len(expectedArgs) {
		t.Errorf("command args = %v, want %v", capturedArgs, expectedArgs)
	} else {
		for i, arg := range capturedArgs {
			if arg != expectedArgs[i] {
				t.Errorf("command args[%d] = %q, want %q", i, arg, expectedArgs[i])
			}
		}
	}
}

func TestRunner_DefaultExecutor(t *testing.T) {
	r := New(Config{ScenarioDir: "/tmp/test"})
	if r.executor == nil {
		t.Error("executor is nil, want default executor")
	}
}

func TestRunner_DefaultLogWriter(t *testing.T) {
	r := New(Config{ScenarioDir: "/tmp/test"})
	if r.logWriter == nil {
		t.Error("logWriter is nil, want io.Discard")
	}
}
