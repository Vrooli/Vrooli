package shell

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
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

func TestLinter_Name(t *testing.T) {
	l := New(Config{ScenarioDir: "/tmp/test", ScenarioName: "test"})
	if got := l.Name(); got != "shell" {
		t.Errorf("Name() = %q, want %q", got, "shell")
	}
}

func TestLinter_Detect_WithCLIBinary(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "demo"), []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatalf("failed to write cli binary: %v", err)
	}

	l := New(Config{ScenarioDir: dir, ScenarioName: "demo"})
	if !l.Detect() {
		t.Error("Detect() = false, want true when CLI binary exists")
	}
}

func TestLinter_Detect_NoCLIDir(t *testing.T) {
	dir := t.TempDir()

	l := New(Config{ScenarioDir: dir, ScenarioName: "demo"})
	if l.Detect() {
		t.Error("Detect() = true, want false when cli/ dir is missing")
	}
}

func TestLinter_Detect_NoCLIBinary(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}

	l := New(Config{ScenarioDir: dir, ScenarioName: "demo"})
	if l.Detect() {
		t.Error("Detect() = true, want false when no executable exists")
	}
}

func TestLinter_Run_Success(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "demo"), []byte("#!/bin/bash\necho hello\n"), 0o755); err != nil {
		t.Fatalf("failed to write cli binary: %v", err)
	}

	executor := &mockExecutor{}
	l := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Executor:     executor,
	})

	result := l.Run(context.Background())
	if !result.Success {
		t.Errorf("Run() success = false, want true; error: %v", result.Error)
	}
	if len(result.Observations) == 0 {
		t.Error("Run() returned no observations")
	}
}

func TestLinter_Run_SkipsWhenNoCLI(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}

	executor := &mockExecutor{}
	l := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Executor:     executor,
	})

	result := l.Run(context.Background())
	if !result.Skipped {
		t.Error("Run() should skip when no CLI binary exists")
	}
}

func TestLinter_Run_BashMissing(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "demo"), []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatalf("failed to write cli binary: %v", err)
	}

	executor := &mockExecutor{
		lookPathFunc: func(name string) (string, error) {
			if name == "bash" {
				return "", errors.New("not found")
			}
			return "/usr/bin/" + name, nil
		},
	}
	l := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Executor:     executor,
	})

	result := l.Run(context.Background())
	if result.Success || result.Skipped {
		t.Error("Run() should fail when bash is missing")
	}
	if result.FailureClass != types.FailureClassMissingDependency {
		t.Errorf("FailureClass = %q, want %q", result.FailureClass, types.FailureClassMissingDependency)
	}
}

func TestLinter_Run_SyntaxError(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "demo"), []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatalf("failed to write cli binary: %v", err)
	}

	executor := &mockExecutor{
		runFunc: func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			if name == "bash" && len(args) > 0 && args[0] == "-n" {
				return errors.New("line 5: syntax error")
			}
			return nil
		},
	}
	l := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Executor:     executor,
	})

	result := l.Run(context.Background())
	if result.Success {
		t.Error("Run() success = true, want false when syntax error")
	}
	if result.FailureClass != types.FailureClassTestFailure {
		t.Errorf("FailureClass = %q, want %q", result.FailureClass, types.FailureClassTestFailure)
	}
}

func TestLinter_Run_ContextCancelled(t *testing.T) {
	dir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	l := New(Config{ScenarioDir: dir, ScenarioName: "demo"})
	result := l.Run(ctx)

	if result.Success {
		t.Error("Run() success = true, want false when context is cancelled")
	}
	if result.FailureClass != types.FailureClassSystem {
		t.Errorf("FailureClass = %q, want %q", result.FailureClass, types.FailureClassSystem)
	}
}

func TestLinter_Run_CommandArgs(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}
	cliPath := filepath.Join(cliDir, "demo")
	if err := os.WriteFile(cliPath, []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatalf("failed to write cli binary: %v", err)
	}

	var capturedName string
	var capturedArgs []string

	executor := &mockExecutor{
		runFunc: func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			capturedName = name
			capturedArgs = args
			return nil
		},
	}
	l := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Executor:     executor,
	})

	l.Run(context.Background())

	if capturedName != "bash" {
		t.Errorf("command name = %q, want %q", capturedName, "bash")
	}
	if len(capturedArgs) != 2 || capturedArgs[0] != "-n" {
		t.Errorf("command args = %v, want [-n <path>]", capturedArgs)
	}
	if !strings.HasSuffix(capturedArgs[1], "demo") {
		t.Errorf("command args[1] = %q, want path ending in 'demo'", capturedArgs[1])
	}
}

func TestDiscoverCLIBinary_ScenarioName(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "my-app"), []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatalf("failed to write cli binary: %v", err)
	}

	l := New(Config{ScenarioDir: dir, ScenarioName: "my-app"})
	path, err := l.discoverCLIBinary()
	if err != nil {
		t.Fatalf("discoverCLIBinary() error = %v", err)
	}
	if !strings.HasSuffix(path, "my-app") {
		t.Errorf("discoverCLIBinary() = %q, want path ending in 'my-app'", path)
	}
}

func TestDiscoverCLIBinary_ShExtension(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "my-app.sh"), []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatalf("failed to write cli binary: %v", err)
	}

	l := New(Config{ScenarioDir: dir, ScenarioName: "my-app"})
	path, err := l.discoverCLIBinary()
	if err != nil {
		t.Fatalf("discoverCLIBinary() error = %v", err)
	}
	if !strings.HasSuffix(path, "my-app.sh") {
		t.Errorf("discoverCLIBinary() = %q, want path ending in 'my-app.sh'", path)
	}
}

func TestDiscoverCLIBinary_Fallback(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "other-script"), []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatalf("failed to write cli binary: %v", err)
	}

	l := New(Config{ScenarioDir: dir, ScenarioName: "my-app"})
	path, err := l.discoverCLIBinary()
	if err != nil {
		t.Fatalf("discoverCLIBinary() error = %v", err)
	}
	if !strings.HasSuffix(path, "other-script") {
		t.Errorf("discoverCLIBinary() = %q, want path ending in 'other-script'", path)
	}
}

func TestEnsureExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows - executable bit not used")
	}

	dir := t.TempDir()

	// Non-executable file
	nonExec := filepath.Join(dir, "non-exec")
	if err := os.WriteFile(nonExec, []byte("#!/bin/bash\n"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := ensureExecutable(nonExec); err == nil {
		t.Error("ensureExecutable() should fail for non-executable file")
	}

	// Executable file
	exec := filepath.Join(dir, "exec")
	if err := os.WriteFile(exec, []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := ensureExecutable(exec); err != nil {
		t.Errorf("ensureExecutable() error = %v for executable file", err)
	}

	// Missing file
	if err := ensureExecutable(filepath.Join(dir, "missing")); err == nil {
		t.Error("ensureExecutable() should fail for missing file")
	}

	// Directory
	subdir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	if err := ensureExecutable(subdir); err == nil {
		t.Error("ensureExecutable() should fail for directory")
	}
}

func TestLinter_IsExcluded(t *testing.T) {
	tests := []struct {
		name     string
		exclude  []string
		path     string
		expected bool
	}{
		{
			name:     "exact match",
			exclude:  []string{"cli/my-binary"},
			path:     "cli/my-binary",
			expected: true,
		},
		{
			name:     "no match",
			exclude:  []string{"cli/other-binary"},
			path:     "cli/my-binary",
			expected: false,
		},
		{
			name:     "prefix match with separator",
			exclude:  []string{"cli"},
			path:     "cli/my-binary",
			expected: true,
		},
		{
			name:     "prefix match without separator - should not match",
			exclude:  []string{"cli"},
			path:     "cli-tools/my-binary",
			expected: false,
		},
		{
			name:     "multiple excludes",
			exclude:  []string{"api/server", "cli/my-binary"},
			path:     "cli/my-binary",
			expected: true,
		},
		{
			name:     "path normalization",
			exclude:  []string{"cli//my-binary"},
			path:     "cli/my-binary",
			expected: true,
		},
		{
			name:     "empty exclude list",
			exclude:  nil,
			path:     "cli/my-binary",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Linter{
				scenarioDir: "/tmp/test",
				exclude:     tt.exclude,
			}
			if got := l.isExcluded(tt.path); got != tt.expected {
				t.Errorf("isExcluded(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestIsShellScript(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name        string
		content     []byte
		expectShell bool
		reasonPart  string
	}{
		{
			name:        "shell script with shebang",
			content:     []byte("#!/bin/bash\necho hello\n"),
			expectShell: true,
			reasonPart:  "shebang",
		},
		{
			name:        "shell script with sh shebang",
			content:     []byte("#!/bin/sh\necho hello\n"),
			expectShell: true,
			reasonPart:  "shebang",
		},
		{
			name:        "ELF binary",
			content:     []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expectShell: false,
			reasonPart:  "ELF",
		},
		{
			name:        "Windows PE binary",
			content:     []byte{'M', 'Z', 0x90, 0x00, 0x03, 0x00, 0x00, 0x00, 0x04, 0x00},
			expectShell: false,
			reasonPart:  "PE",
		},
		{
			name:        "binary with null bytes",
			content:     []byte{'s', 'o', 'm', 'e', 0x00, 'd', 'a', 't', 'a'},
			expectShell: false,
			reasonPart:  "null bytes",
		},
		{
			name:        "plain text assumed shell",
			content:     []byte("# some comment\necho hello\n"),
			expectShell: true,
			reasonPart:  "assumed",
		},
		{
			name:        "shell pattern set -e",
			content:     []byte("set -e\nexport FOO=bar\n"),
			expectShell: true,
			reasonPart:  "shell pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, strings.ReplaceAll(tt.name, " ", "_"))
			if err := os.WriteFile(path, tt.content, 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			isShell, reason := isShellScript(path)
			if isShell != tt.expectShell {
				t.Errorf("isShellScript() = %v, want %v; reason: %s", isShell, tt.expectShell, reason)
			}
			if tt.reasonPart != "" && !strings.Contains(strings.ToLower(reason), strings.ToLower(tt.reasonPart)) {
				t.Errorf("isShellScript() reason = %q, want to contain %q", reason, tt.reasonPart)
			}
		})
	}
}

func TestDiscoverCLIBinary_SkipsExcluded(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}

	// Create excluded shell script
	if err := os.WriteFile(filepath.Join(cliDir, "excluded-cli"), []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatalf("failed to write excluded cli: %v", err)
	}
	// Create non-excluded shell script
	if err := os.WriteFile(filepath.Join(cliDir, "my-cli"), []byte("#!/bin/bash\necho allowed\n"), 0o755); err != nil {
		t.Fatalf("failed to write allowed cli: %v", err)
	}

	l := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "excluded-cli",
		Exclude:      []string{"cli/excluded-cli"},
	})

	path, err := l.discoverCLIBinary()
	if err != nil {
		t.Fatalf("discoverCLIBinary() error = %v", err)
	}
	if strings.Contains(path, "excluded-cli") {
		t.Errorf("discoverCLIBinary() = %q, should not return excluded path", path)
	}
	if !strings.HasSuffix(path, "my-cli") {
		t.Errorf("discoverCLIBinary() = %q, want path ending in 'my-cli'", path)
	}
}

func TestDiscoverCLIBinary_SkipsCompiledBinaries(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}

	// Create ELF binary (simulated)
	elfContent := []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x01, 0x00}
	elfContent = append(elfContent, make([]byte, 56)...) // Pad to make it look more like a real ELF
	if err := os.WriteFile(filepath.Join(cliDir, "my-app"), elfContent, 0o755); err != nil {
		t.Fatalf("failed to write ELF binary: %v", err)
	}

	// Create shell script
	if err := os.WriteFile(filepath.Join(cliDir, "my-app.sh"), []byte("#!/bin/bash\necho hello\n"), 0o755); err != nil {
		t.Fatalf("failed to write shell script: %v", err)
	}

	l := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "my-app",
	})

	path, err := l.discoverCLIBinary()
	if err != nil {
		t.Fatalf("discoverCLIBinary() error = %v", err)
	}
	if !strings.HasSuffix(path, "my-app.sh") {
		t.Errorf("discoverCLIBinary() = %q, want .sh file (not ELF binary)", path)
	}
}

func TestDiscoverCLIBinary_OnlyBinaryNoShellScript(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}

	// Create only ELF binary (simulated)
	elfContent := []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x01, 0x00}
	elfContent = append(elfContent, make([]byte, 56)...)
	if err := os.WriteFile(filepath.Join(cliDir, "my-app"), elfContent, 0o755); err != nil {
		t.Fatalf("failed to write ELF binary: %v", err)
	}

	l := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "my-app",
	})

	_, err := l.discoverCLIBinary()
	if err == nil {
		t.Error("discoverCLIBinary() should fail when only compiled binary exists")
	}
	if !strings.Contains(err.Error(), "compiled binaries") && !strings.Contains(err.Error(), "skipped") {
		t.Errorf("error = %q, want to mention compiled binaries or skipped", err.Error())
	}
}
