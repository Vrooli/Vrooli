package python

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
	if got := r.Name(); got != "python" {
		t.Errorf("Name() = %q, want %q", got, "python")
	}
}

func TestRunner_Detect_WithRequirementsTxt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("pytest\n"), 0o644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	r := New(Config{ScenarioDir: dir})
	if !r.Detect() {
		t.Error("Detect() = false, want true when requirements.txt exists")
	}
}

func TestRunner_Detect_WithPyprojectToml(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[tool.poetry]\n"), 0o644); err != nil {
		t.Fatalf("failed to write pyproject.toml: %v", err)
	}

	r := New(Config{ScenarioDir: dir})
	if !r.Detect() {
		t.Error("Detect() = false, want true when pyproject.toml exists")
	}
}

func TestRunner_Detect_WithSetupPy(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "setup.py"), []byte("from setuptools import setup\n"), 0o644); err != nil {
		t.Fatalf("failed to write setup.py: %v", err)
	}

	r := New(Config{ScenarioDir: dir})
	if !r.Detect() {
		t.Error("Detect() = false, want true when setup.py exists")
	}
}

func TestRunner_Detect_WithTestsDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "tests"), 0o755); err != nil {
		t.Fatalf("failed to create tests dir: %v", err)
	}

	r := New(Config{ScenarioDir: dir})
	if !r.Detect() {
		t.Error("Detect() = false, want true when tests/ directory exists")
	}
}

func TestRunner_Detect_InPythonSubdir(t *testing.T) {
	dir := t.TempDir()
	pythonDir := filepath.Join(dir, "python")
	if err := os.MkdirAll(pythonDir, 0o755); err != nil {
		t.Fatalf("failed to create python dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pythonDir, "requirements.txt"), []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	r := New(Config{ScenarioDir: dir})
	if !r.Detect() {
		t.Error("Detect() = false, want true when python/ subdirectory has indicators")
	}
}

func TestRunner_Detect_NoIndicators(t *testing.T) {
	dir := t.TempDir()

	r := New(Config{ScenarioDir: dir})
	if r.Detect() {
		t.Error("Detect() = true, want false when no Python indicators exist")
	}
}

func TestRunner_Run_WithPytest(t *testing.T) {
	dir := t.TempDir()
	testsDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("failed to create tests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "test_example.py"), []byte("def test_pass(): pass\n"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var usedPytest bool
	executor := &mockExecutor{
		runFunc: func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			// Check for pytest support
			if len(args) >= 1 && args[0] == "-c" {
				return nil // pytest is available
			}
			// Check for actual pytest run
			if len(args) >= 2 && args[1] == "pytest" {
				usedPytest = true
			}
			return nil
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
	if !usedPytest {
		t.Error("expected pytest to be used when available")
	}
}

func TestRunner_Run_FallsBackToUnittest(t *testing.T) {
	dir := t.TempDir()
	testsDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("failed to create tests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "test_example.py"), []byte("def test_pass(): pass\n"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var usedUnittest bool
	executor := &mockExecutor{
		runFunc: func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			// Pytest not available
			if len(args) >= 1 && args[0] == "-c" {
				return errors.New("pytest not found")
			}
			// Check for unittest run
			if len(args) >= 2 && args[1] == "unittest" {
				usedUnittest = true
			}
			return nil
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
	if !usedUnittest {
		t.Error("expected unittest to be used when pytest unavailable")
	}
}

func TestRunner_Run_SkipsWhenNoTestFiles(t *testing.T) {
	dir := t.TempDir()
	// Has indicator but no test files
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	executor := &mockExecutor{}
	r := New(Config{
		ScenarioDir: dir,
		Executor:    executor,
	})

	result := r.Run(context.Background())
	if !result.Skipped {
		t.Error("Run() should skip when no test files exist")
	}
}

func TestRunner_Run_PythonMissing(t *testing.T) {
	dir := t.TempDir()
	testsDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("failed to create tests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "test_example.py"), []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	executor := &mockExecutor{
		lookPathFunc: func(name string) (string, error) {
			if name == "python3" || name == "python" {
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
		t.Error("Run() should fail when python is missing")
	}
	if result.FailureClass != types.FailureClassMissingDependency {
		t.Errorf("FailureClass = %q, want %q", result.FailureClass, types.FailureClassMissingDependency)
	}
}

func TestRunner_Run_TestFailure(t *testing.T) {
	dir := t.TempDir()
	testsDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("failed to create tests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "test_example.py"), []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	executor := &mockExecutor{
		runFunc: func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			// Pytest available
			if len(args) >= 1 && args[0] == "-c" {
				return nil
			}
			// Pytest fails
			if len(args) >= 2 && args[1] == "pytest" {
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

func TestHasTestFiles(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected bool
	}{
		{
			name:     "test_example.py",
			files:    []string{"test_example.py"},
			expected: true,
		},
		{
			name:     "example_test.py",
			files:    []string{"example_test.py"},
			expected: true,
		},
		{
			name:     "nested test file",
			files:    []string{"tests/test_example.py"},
			expected: true,
		},
		{
			name:     "no test files",
			files:    []string{"example.py", "main.py"},
			expected: false,
		},
		{
			name:     "empty directory",
			files:    []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, file := range tt.files {
				path := filepath.Join(dir, file)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
				if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
			}

			if got := HasTestFiles(dir); got != tt.expected {
				t.Errorf("HasTestFiles() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHasTestFiles_SkipsVirtualEnv(t *testing.T) {
	dir := t.TempDir()
	// Create test file in venv - should be skipped
	venvDir := filepath.Join(dir, "venv", "lib")
	if err := os.MkdirAll(venvDir, 0o755); err != nil {
		t.Fatalf("failed to create venv dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(venvDir, "test_example.py"), []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if HasTestFiles(dir) {
		t.Error("HasTestFiles() = true, want false (should skip venv)")
	}
}

func TestHasIndicators(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		dirs     []string
		expected bool
	}{
		{
			name:     "requirements.txt",
			files:    []string{"requirements.txt"},
			expected: true,
		},
		{
			name:     "pyproject.toml",
			files:    []string{"pyproject.toml"},
			expected: true,
		},
		{
			name:     "setup.py",
			files:    []string{"setup.py"},
			expected: true,
		},
		{
			name:     "tests directory",
			dirs:     []string{"tests"},
			expected: true,
		},
		{
			name:     "tests/__init__.py",
			files:    []string{"tests/__init__.py"},
			dirs:     []string{"tests"},
			expected: true,
		},
		{
			name:     "no indicators",
			files:    []string{"main.py"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, d := range tt.dirs {
				if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
			}
			for _, file := range tt.files {
				path := filepath.Join(dir, file)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
				if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
			}

			if got := hasIndicators(dir); got != tt.expected {
				t.Errorf("hasIndicators() = %v, want %v", got, tt.expected)
			}
		})
	}
}
