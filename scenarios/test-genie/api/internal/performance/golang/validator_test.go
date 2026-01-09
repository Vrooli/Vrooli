package golang

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
	apiDir := filepath.Join(root, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}
	// Create a minimal go.mod file
	goMod := filepath.Join(apiDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
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

	result := v.Benchmark(context.Background(), 90*time.Second)

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}
}

func TestValidator_BenchmarkFailsWhenGoNotFound(t *testing.T) {
	scenarioDir := createTestScenarioDir(t)

	v := New(scenarioDir,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupFail),
		WithCommandExecutor(mockCommandExecutorSuccess),
	)

	result := v.Benchmark(context.Background(), 90*time.Second)

	if result.Success {
		t.Fatal("expected failure when go command not found")
	}
	if result.FailureClass != FailureClassMissingDependency {
		t.Errorf("expected missing_dependency failure class, got %s", result.FailureClass)
	}
	if result.Remediation == "" {
		t.Error("expected remediation guidance")
	}
}

func TestValidator_BenchmarkFailsWhenAPIDirMissing(t *testing.T) {
	// Create temp dir without api/ subdirectory
	root := t.TempDir()

	v := New(root,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(mockCommandExecutorSuccess),
	)

	result := v.Benchmark(context.Background(), 90*time.Second)

	if result.Success {
		t.Fatal("expected failure when api directory missing")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration failure class, got %s", result.FailureClass)
	}
}

func TestValidator_BenchmarkFailsWhenBuildFails(t *testing.T) {
	scenarioDir := createTestScenarioDir(t)

	v := New(scenarioDir,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(mockCommandExecutorFail),
	)

	result := v.Benchmark(context.Background(), 90*time.Second)

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

func TestValidator_BenchmarkRespectsContextCancellation(t *testing.T) {
	scenarioDir := createTestScenarioDir(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	v := New(scenarioDir,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			return ctx.Err()
		}),
	)

	result := v.Benchmark(ctx, 90*time.Second)

	if result.Success {
		t.Fatal("expected failure when context cancelled")
	}
}

func TestValidator_BenchmarkRecordsDuration(t *testing.T) {
	scenarioDir := createTestScenarioDir(t)
	sleepDuration := 10 * time.Millisecond

	v := New(scenarioDir,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(mockCommandExecutorSlow(sleepDuration)),
	)

	result := v.Benchmark(context.Background(), 5*time.Second)

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.Duration < sleepDuration {
		t.Errorf("expected duration >= %s, got %s", sleepDuration, result.Duration)
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

	v.Benchmark(context.Background(), 90*time.Second)

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

func TestEnsureDir(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string
		wantError bool
	}{
		{
			name: "directory exists",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantError: false,
		},
		{
			name: "directory does not exist",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantError: true,
		},
		{
			name: "path is a file",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				file := filepath.Join(dir, "file.txt")
				if err := os.WriteFile(file, []byte("test"), 0o644); err != nil {
					t.Fatalf("failed to create file: %v", err)
				}
				return file
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.setup(t)
			err := ensureDir(path)
			if (err != nil) != tc.wantError {
				t.Errorf("ensureDir() error = %v, wantError %v", err, tc.wantError)
			}
		})
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
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Temp File Error Test
// =============================================================================

func TestValidator_BenchmarkHandlesTempFileCreation(t *testing.T) {
	scenarioDir := createTestScenarioDir(t)

	// This test validates that temp file creation works
	// The temp file creation error path is hard to trigger without
	// modifying TMPDIR or permissions, so we just verify success works
	v := New(scenarioDir,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(mockCommandExecutorSuccess),
	)

	result := v.Benchmark(context.Background(), 90*time.Second)

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkValidatorSuccess(b *testing.B) {
	root := b.TempDir()
	apiDir := filepath.Join(root, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		b.Fatalf("failed to create api dir: %v", err)
	}

	v := New(root,
		WithLogger(io.Discard),
		WithCommandLookup(mockCommandLookupSuccess),
		WithCommandExecutor(mockCommandExecutorSuccess),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Benchmark(context.Background(), 90*time.Second)
	}
}

// =============================================================================
// Interface Compile-Time Checks
// =============================================================================

var _ Validator = (*validator)(nil)
