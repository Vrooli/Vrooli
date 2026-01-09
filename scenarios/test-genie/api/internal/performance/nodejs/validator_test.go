package nodejs

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// =============================================================================
// Mock CommandExecutor and CommandLookup
// =============================================================================

func mockCommandLookupSuccess(name string) (string, error) {
	return "/usr/bin/" + name, nil
}

func mockCommandLookupFail(name string) (string, error) {
	return "", errors.New("command not found")
}

func mockCommandExecutorSuccess(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
	return nil
}

func mockCommandExecutorFail(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
	return errors.New("build failed")
}

func mockCommandExecutorSlow(delay time.Duration) CommandExecutor {
	return func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
		time.Sleep(delay)
		return nil
	}
}

// =============================================================================
// Test Helpers
// =============================================================================

func createTestScenarioDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	uiDir := filepath.Join(root, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}
	// Create package.json with build script
	pkgJSON := `{"name": "test-ui", "scripts": {"build": "echo build"}}`
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}
	// Create node_modules to skip install step
	nodeModules := filepath.Join(uiDir, "node_modules")
	if err := os.MkdirAll(nodeModules, 0o755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}
	return root
}

func createTestScenarioWithPnpm(t *testing.T) string {
	t.Helper()
	root := createTestScenarioDir(t)
	uiDir := filepath.Join(root, "ui")
	// Create pnpm-lock.yaml
	if err := os.WriteFile(filepath.Join(uiDir, "pnpm-lock.yaml"), []byte("lockfileVersion: 9"), 0o644); err != nil {
		t.Fatalf("failed to create pnpm-lock.yaml: %v", err)
	}
	return root
}

func createTestScenarioWithYarn(t *testing.T) string {
	t.Helper()
	root := createTestScenarioDir(t)
	uiDir := filepath.Join(root, "ui")
	// Create yarn.lock
	if err := os.WriteFile(filepath.Join(uiDir, "yarn.lock"), []byte("# yarn lockfile v1"), 0o644); err != nil {
		t.Fatalf("failed to create yarn.lock: %v", err)
	}
	return root
}

func createTestScenarioWithPackageManagerField(t *testing.T, mgr string) string {
	t.Helper()
	root := t.TempDir()
	uiDir := filepath.Join(root, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}
	pkgJSON := `{"name": "test-ui", "packageManager": "` + mgr + `@9.0.0", "scripts": {"build": "echo build"}}`
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}
	nodeModules := filepath.Join(uiDir, "node_modules")
	if err := os.MkdirAll(nodeModules, 0o755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}
	return root
}

// =============================================================================
// Validator Tests
// =============================================================================

func TestValidator_BenchmarkSuccess(t *testing.T) {
	scenarioDir := createTestScenarioDir(t)

	v := New(scenarioDir,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(mockCommandExecutorSuccess),
	)

	result := v.Benchmark(context.Background(), 180*time.Second)

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}
	if result.Skipped {
		t.Error("expected build not to be skipped")
	}
}

func TestValidator_BenchmarkSkipsWhenNoUIDir(t *testing.T) {
	root := t.TempDir() // No ui/ directory

	v := New(root,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(mockCommandExecutorSuccess),
	)

	result := v.Benchmark(context.Background(), 180*time.Second)

	if !result.Success {
		t.Fatalf("expected success for skip, got error: %v", result.Error)
	}
	if !result.Skipped {
		t.Error("expected build to be skipped when no UI dir")
	}
}

func TestValidator_BenchmarkSkipsWhenNoBuildScript(t *testing.T) {
	root := t.TempDir()
	uiDir := filepath.Join(root, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}
	// Create package.json WITHOUT build script
	pkgJSON := `{"name": "test-ui", "scripts": {}}`
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}

	v := New(root,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(mockCommandExecutorSuccess),
	)

	result := v.Benchmark(context.Background(), 180*time.Second)

	if !result.Success {
		t.Fatalf("expected success for skip, got error: %v", result.Error)
	}
	if !result.Skipped {
		t.Error("expected build to be skipped when no build script")
	}
}

func TestValidator_BenchmarkFailsWhenPackageManagerNotFound(t *testing.T) {
	scenarioDir := createTestScenarioDir(t)

	v := New(scenarioDir,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupFail),
		WithCommandExecutor(mockCommandExecutorSuccess),
	)

	result := v.Benchmark(context.Background(), 180*time.Second)

	if result.Success {
		t.Fatal("expected failure when package manager not found")
	}
	if result.FailureClass != FailureClassMissingDependency {
		t.Errorf("expected missing_dependency failure class, got %s", result.FailureClass)
	}
	if result.Remediation == "" {
		t.Error("expected remediation guidance")
	}
}

func TestValidator_BenchmarkFailsWhenBuildFails(t *testing.T) {
	scenarioDir := createTestScenarioDir(t)

	v := New(scenarioDir,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(mockCommandExecutorFail),
	)

	result := v.Benchmark(context.Background(), 180*time.Second)

	if result.Success {
		t.Fatal("expected failure when build fails")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure class, got %s", result.FailureClass)
	}
}

func TestValidator_BenchmarkFailsWhenDurationExceeded(t *testing.T) {
	scenarioDir := createTestScenarioDir(t)

	v := New(scenarioDir,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(mockCommandExecutorSlow(50*time.Millisecond)),
	)

	// Set a very short threshold
	result := v.Benchmark(context.Background(), 1*time.Millisecond)

	if result.Success {
		t.Fatal("expected failure when duration exceeded")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure class, got %s", result.FailureClass)
	}
	if result.Duration == 0 {
		t.Error("expected duration to be recorded")
	}
}

func TestValidator_BenchmarkFailsWhenPackageJSONInvalid(t *testing.T) {
	root := t.TempDir()
	uiDir := filepath.Join(root, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}
	// Create invalid package.json
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}

	v := New(root,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(mockCommandExecutorSuccess),
	)

	result := v.Benchmark(context.Background(), 180*time.Second)

	if result.Success {
		t.Fatal("expected failure when package.json invalid")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration failure class, got %s", result.FailureClass)
	}
}

func TestValidator_BenchmarkInstallsDependenciesWhenMissing(t *testing.T) {
	root := t.TempDir()
	uiDir := filepath.Join(root, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}
	pkgJSON := `{"name": "test-ui", "scripts": {"build": "echo build"}}`
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}
	// Note: no node_modules directory

	var installCalled bool
	v := New(root,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			if len(args) > 0 && args[0] == "install" {
				installCalled = true
			}
			return nil
		}),
	)

	result := v.Benchmark(context.Background(), 180*time.Second)

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if !installCalled {
		t.Error("expected install to be called when node_modules missing")
	}
}

// =============================================================================
// Package Manager Detection Tests
// =============================================================================

func TestValidator_DetectsPnpmFromLockfile(t *testing.T) {
	scenarioDir := createTestScenarioWithPnpm(t)

	var detectedManager string
	v := New(scenarioDir,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			if name == "pnpm" || name == "yarn" || name == "npm" {
				detectedManager = name
			}
			return nil
		}),
	)

	v.Benchmark(context.Background(), 180*time.Second)

	if detectedManager != "pnpm" {
		t.Errorf("expected pnpm, got %s", detectedManager)
	}
}

func TestValidator_DetectsYarnFromLockfile(t *testing.T) {
	scenarioDir := createTestScenarioWithYarn(t)

	var detectedManager string
	v := New(scenarioDir,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			if name == "pnpm" || name == "yarn" || name == "npm" {
				detectedManager = name
			}
			return nil
		}),
	)

	v.Benchmark(context.Background(), 180*time.Second)

	if detectedManager != "yarn" {
		t.Errorf("expected yarn, got %s", detectedManager)
	}
}

func TestValidator_DetectsPackageManagerFromField(t *testing.T) {
	tests := []struct {
		field    string
		expected string
	}{
		{"pnpm", "pnpm"},
		{"yarn", "yarn"},
		{"npm", "npm"},
		{"PNPM", "pnpm"},
		{"Yarn", "yarn"},
	}

	for _, tc := range tests {
		t.Run(tc.field, func(t *testing.T) {
			scenarioDir := createTestScenarioWithPackageManagerField(t, tc.field)

			var detectedManager string
			v := New(scenarioDir,
				WithLogger(io.Discard),
				WithCommandLookup(mockCommandLookupSuccess),
				WithCommandExecutor(func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
					if name == "pnpm" || name == "yarn" || name == "npm" {
						detectedManager = name
					}
					return nil
				}),
			)

			v.Benchmark(context.Background(), 180*time.Second)

			if detectedManager != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, detectedManager)
			}
		})
	}
}

func TestValidator_DefaultsToNpm(t *testing.T) {
	scenarioDir := createTestScenarioDir(t)

	var detectedManager string
	v := New(scenarioDir,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			if name == "pnpm" || name == "yarn" || name == "npm" {
				detectedManager = name
			}
			return nil
		}),
	)

	v.Benchmark(context.Background(), 180*time.Second)

	if detectedManager != "npm" {
		t.Errorf("expected npm as default, got %s", detectedManager)
	}
}

// =============================================================================
// ParsePackageManager Tests
// =============================================================================

func TestParsePackageManager(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"pnpm@9.0.0", "pnpm"},
		{"yarn@4.0.0", "yarn"},
		{"npm@10.0.0", "npm"},
		{"PNPM@9.0.0", "pnpm"},
		{"", ""},
		{"unknown", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := parsePackageManager(tc.input)
			if result != tc.expected {
				t.Errorf("parsePackageManager(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// =============================================================================
// Option Tests
// =============================================================================

func TestWithLogger(t *testing.T) {
	scenarioDir := createTestScenarioDir(t)
	var buf []byte
	w := &mockWriter{buf: &buf}

	v := New(scenarioDir,
		WithLogger(w),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(mockCommandExecutorSuccess),
	)

	v.Benchmark(context.Background(), 180*time.Second)

	if len(buf) == 0 {
		t.Error("expected logs to be written")
	}
}

type mockWriter struct {
	buf *[]byte
}

func (w *mockWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestFileExists(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		expected bool
	}{
		{
			name: "file exists",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				file := filepath.Join(dir, "file.txt")
				if err := os.WriteFile(file, []byte("test"), 0o644); err != nil {
					t.Fatalf("failed to create file: %v", err)
				}
				return file
			},
			expected: true,
		},
		{
			name: "file does not exist",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			expected: false,
		},
		{
			name: "path is a directory",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.setup(t)
			result := fileExists(path)
			if result != tc.expected {
				t.Errorf("fileExists() = %v, want %v", result, tc.expected)
			}
		})
	}
}

// =============================================================================
// Install Dependencies Tests
// =============================================================================

func TestValidator_InstallDependenciesWithYarn(t *testing.T) {
	scenarioDir := createTestScenarioWithYarn(t)

	// Remove node_modules to trigger install
	uiDir := filepath.Join(scenarioDir, "ui")
	os.RemoveAll(filepath.Join(uiDir, "node_modules"))

	var installedWith string
	v := New(scenarioDir,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			if len(args) > 0 && args[0] == "install" {
				installedWith = name
			}
			return nil
		}),
	)

	result := v.Benchmark(context.Background(), 180*time.Second)

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if installedWith != "yarn" {
		t.Errorf("expected yarn install, got %s", installedWith)
	}
}

func TestValidator_InstallDependenciesWithNpm(t *testing.T) {
	scenarioDir := createTestScenarioDir(t)

	// Remove node_modules to trigger install
	uiDir := filepath.Join(scenarioDir, "ui")
	os.RemoveAll(filepath.Join(uiDir, "node_modules"))

	var installedWith string
	v := New(scenarioDir,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			if len(args) > 0 && args[0] == "install" {
				installedWith = name
			}
			return nil
		}),
	)

	result := v.Benchmark(context.Background(), 180*time.Second)

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if installedWith != "npm" {
		t.Errorf("expected npm install, got %s", installedWith)
	}
}

func TestValidator_InstallDependenciesWithPnpm(t *testing.T) {
	scenarioDir := createTestScenarioWithPnpm(t)

	// Remove node_modules to trigger install
	uiDir := filepath.Join(scenarioDir, "ui")
	os.RemoveAll(filepath.Join(uiDir, "node_modules"))

	var installedWith string
	v := New(scenarioDir,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			if len(args) > 0 && args[0] == "install" {
				installedWith = name
			}
			return nil
		}),
	)

	result := v.Benchmark(context.Background(), 180*time.Second)

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if installedWith != "pnpm" {
		t.Errorf("expected pnpm install, got %s", installedWith)
	}
}

// =============================================================================
// LoadPackageManifest Tests
// =============================================================================

func TestLoadPackageManifest_EmptyScripts(t *testing.T) {
	root := t.TempDir()
	uiDir := filepath.Join(root, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	// Create package.json without scripts field
	pkgJSON := `{"name": "test-ui"}`
	pkgPath := filepath.Join(uiDir, "package.json")
	if err := os.WriteFile(pkgPath, []byte(pkgJSON), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	v := &validator{scenarioDir: root}
	manifest, err := v.loadPackageManifest(pkgPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if manifest == nil {
		t.Fatal("expected manifest, got nil")
	}
	if manifest.Scripts == nil {
		t.Error("expected Scripts to be initialized")
	}
}

func TestLoadPackageManifest_FileNotFound(t *testing.T) {
	root := t.TempDir()
	v := &validator{scenarioDir: root}

	manifest, err := v.loadPackageManifest(filepath.Join(root, "nonexistent.json"))
	if err != nil {
		t.Errorf("expected no error for missing file, got: %v", err)
	}
	if manifest != nil {
		t.Error("expected nil manifest for missing file")
	}
}

// =============================================================================
// Logging Function Tests
// =============================================================================

func TestLogInfo_NilWriter(t *testing.T) {
	// Should not panic with nil writer
	logInfo(nil, "test message %s", "arg")
}

func TestLogSuccess_NilWriter(t *testing.T) {
	// Should not panic with nil writer
	logSuccess(nil, "test message %s", "arg")
}

func TestLogError_NilWriter(t *testing.T) {
	// Should not panic with nil writer
	logError(nil, "test message %s", "arg")
}

func TestLogInfo_WritesOutput(t *testing.T) {
	var buf []byte
	w := &mockWriter{buf: &buf}
	logInfo(w, "test %s", "value")
	if len(buf) == 0 {
		t.Error("expected output to be written")
	}
	output := string(buf)
	if !containsSubstring(output, "test value") {
		t.Errorf("expected 'test value' in output, got: %s", output)
	}
}

func TestLogSuccess_WritesOutput(t *testing.T) {
	var buf []byte
	w := &mockWriter{buf: &buf}
	logSuccess(w, "build %s", "done")
	if len(buf) == 0 {
		t.Error("expected output to be written")
	}
	output := string(buf)
	if !containsSubstring(output, "SUCCESS") {
		t.Errorf("expected 'SUCCESS' in output, got: %s", output)
	}
}

func TestLogError_WritesOutput(t *testing.T) {
	var buf []byte
	w := &mockWriter{buf: &buf}
	logError(w, "failed %s", "build")
	if len(buf) == 0 {
		t.Error("expected output to be written")
	}
	output := string(buf)
	if !containsSubstring(output, "ERROR") {
		t.Errorf("expected 'ERROR' in output, got: %s", output)
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkValidatorSuccess(b *testing.B) {
	root := b.TempDir()
	uiDir := filepath.Join(root, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		b.Fatalf("failed to create ui dir: %v", err)
	}
	pkgJSON := `{"name": "test-ui", "scripts": {"build": "echo build"}}`
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		b.Fatalf("failed to create package.json: %v", err)
	}
	nodeModules := filepath.Join(uiDir, "node_modules")
	if err := os.MkdirAll(nodeModules, 0o755); err != nil {
		b.Fatalf("failed to create node_modules: %v", err)
	}

	v := New(root,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(mockCommandExecutorSuccess),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Benchmark(context.Background(), 180*time.Second)
	}
}

func BenchmarkValidatorSkipped(b *testing.B) {
	root := b.TempDir() // No ui/ directory

	v := New(root,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(mockCommandExecutorSuccess),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Benchmark(context.Background(), 180*time.Second)
	}
}

// =============================================================================
// Interface Compile-Time Checks
// =============================================================================

var _ Validator = (*validator)(nil)
