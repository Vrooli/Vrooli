package nodejs

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
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
	if got := r.Name(); got != "node" {
		t.Errorf("Name() = %q, want %q", got, "node")
	}
}

func TestRunner_Detect_WithUIPackageJson(t *testing.T) {
	dir := t.TempDir()
	uiDir := filepath.Join(dir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	r := New(Config{ScenarioDir: dir})
	if !r.Detect() {
		t.Error("Detect() = false, want true when ui/package.json exists")
	}
}

func TestRunner_Detect_WithRootPackageJson(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	r := New(Config{ScenarioDir: dir})
	if !r.Detect() {
		t.Error("Detect() = false, want true when package.json exists at root")
	}
}

func TestRunner_Detect_NoPackageJson(t *testing.T) {
	dir := t.TempDir()

	r := New(Config{ScenarioDir: dir})
	if r.Detect() {
		t.Error("Detect() = true, want false when no package.json exists")
	}
}

func TestRunner_Run_Success(t *testing.T) {
	dir := t.TempDir()
	uiDir := filepath.Join(dir, "ui")
	if err := os.MkdirAll(filepath.Join(uiDir, "node_modules"), 0o755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"scripts":{"test":"vitest"}}`), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	executor := &mockExecutor{
		captureFunc: func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			return "All tests passed", nil
		},
	}
	r := New(Config{
		ScenarioDir: dir,
		Executor:    executor,
	})

	result := r.Run(context.Background())
	if !result.Success {
		t.Errorf("Run() success = false, want true; error: %v", result.Error)
	}
}

func TestRunner_Run_SkipsWhenNoTestScript(t *testing.T) {
	dir := t.TempDir()
	uiDir := filepath.Join(dir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"scripts":{}}`), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	executor := &mockExecutor{}
	r := New(Config{
		ScenarioDir: dir,
		Executor:    executor,
	})

	result := r.Run(context.Background())
	if !result.Skipped {
		t.Error("Run() should skip when no test script")
	}
}

func TestRunner_Run_SkipsDefaultTestScript(t *testing.T) {
	dir := t.TempDir()
	uiDir := filepath.Join(dir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}
	defaultScript := `{"scripts":{"test":"echo \"Error: no test specified\" && exit 1"}}`
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(defaultScript), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	executor := &mockExecutor{}
	r := New(Config{
		ScenarioDir: dir,
		Executor:    executor,
	})

	result := r.Run(context.Background())
	if !result.Skipped {
		t.Error("Run() should skip when test script is the npm default")
	}
}

func TestRunner_Run_NodeMissing(t *testing.T) {
	dir := t.TempDir()
	uiDir := filepath.Join(dir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"scripts":{"test":"vitest"}}`), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	executor := &mockExecutor{
		lookPathFunc: func(name string) (string, error) {
			if name == "node" {
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
	if result.Success || result.Skipped {
		t.Error("Run() should fail when node is missing")
	}
	if result.FailureClass != types.FailureClassMissingDependency {
		t.Errorf("FailureClass = %q, want %q", result.FailureClass, types.FailureClassMissingDependency)
	}
}

func TestRunner_Run_PackageManagerMissing(t *testing.T) {
	dir := t.TempDir()
	uiDir := filepath.Join(dir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"scripts":{"test":"vitest"}}`), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	executor := &mockExecutor{
		lookPathFunc: func(name string) (string, error) {
			if name == "npm" {
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
	if result.Success || result.Skipped {
		t.Error("Run() should fail when package manager is missing")
	}
	if result.FailureClass != types.FailureClassMissingDependency {
		t.Errorf("FailureClass = %q, want %q", result.FailureClass, types.FailureClassMissingDependency)
	}
}

func TestRunner_Run_InstallsDependencies(t *testing.T) {
	dir := t.TempDir()
	uiDir := filepath.Join(dir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"scripts":{"test":"vitest"}}`), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	var installedCalled bool
	executor := &mockExecutor{
		runFunc: func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			if name == "npm" && len(args) > 0 && args[0] == "install" {
				installedCalled = true
			}
			return nil
		},
		captureFunc: func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			return "", nil
		},
	}
	r := New(Config{
		ScenarioDir: dir,
		Executor:    executor,
	})

	r.Run(context.Background())
	if !installedCalled {
		t.Error("expected npm install to be called when node_modules is missing")
	}
}

func TestRunner_Run_SkipsInstallWhenNodeModulesExists(t *testing.T) {
	dir := t.TempDir()
	uiDir := filepath.Join(dir, "ui")
	if err := os.MkdirAll(filepath.Join(uiDir, "node_modules"), 0o755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"scripts":{"test":"vitest"}}`), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	var installedCalled bool
	executor := &mockExecutor{
		runFunc: func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			if len(args) > 0 && args[0] == "install" {
				installedCalled = true
			}
			return nil
		},
		captureFunc: func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			return "", nil
		},
	}
	r := New(Config{
		ScenarioDir: dir,
		Executor:    executor,
	})

	r.Run(context.Background())
	if installedCalled {
		t.Error("install should not be called when node_modules exists")
	}
}

func TestRunner_Run_TestFailure(t *testing.T) {
	dir := t.TempDir()
	uiDir := filepath.Join(dir, "ui")
	if err := os.MkdirAll(filepath.Join(uiDir, "node_modules"), 0o755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"scripts":{"test":"vitest"}}`), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	executor := &mockExecutor{
		captureFunc: func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			return "FAIL tests", errors.New("exit status 1")
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
