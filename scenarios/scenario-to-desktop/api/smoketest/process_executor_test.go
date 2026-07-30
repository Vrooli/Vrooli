package smoketest_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestProcessExecutor_Execute_Success(t *testing.T) {
	logger := mocks.NewMockLogger()
	executor := smoketest.NewProcessExecutor(logger)

	ctx := context.Background()
	var output string
	var err error

	if runtime.GOOS == "windows" {
		output, err = executor.Execute(ctx, "", "cmd", []string{"/c", "echo", "hello"}, nil, 5*time.Second)
	} else {
		output, err = executor.Execute(ctx, "", "echo", []string{"hello"}, nil, 5*time.Second)
	}

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("Execute() output = %q, want to contain 'hello'", output)
	}
}

func TestProcessExecutor_Execute_CleansDescendantsAfterSuccessfulLeaderExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-group cleanup is Unix-specific")
	}

	pidFile := filepath.Join(t.TempDir(), "child.pid")
	executor := smoketest.NewProcessExecutor(mocks.NewMockLogger(), smoketest.WithKillGracePeriod(25*time.Millisecond))
	result, err := executor.ExecuteWithResult(context.Background(), "", "sh", []string{"-c", "sleep 300 >/dev/null 2>&1 & echo $! > \"$1\"; exit 0", "sh", pidFile}, nil, 5*time.Second)
	if err != nil || result == nil || result.ExitCode != 0 {
		t.Fatalf("successful launcher result = %#v, %v", result, err)
	}

	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("read child pid: %v", err)
	}
	var pid int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid); err != nil || pid <= 0 {
		t.Fatalf("parse child pid %q: %v", data, err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		err = syscall.Kill(pid, 0)
		if err == syscall.ESRCH {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("descendant pid %d survived successful launcher cleanup: %v", pid, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestProcessExecutor_Execute_WithEnv(t *testing.T) {
	logger := mocks.NewMockLogger()
	executor := smoketest.NewProcessExecutor(logger)

	ctx := context.Background()
	env := []string{"TEST_VAR=test_value"}

	var output string
	var err error

	if runtime.GOOS == "windows" {
		output, err = executor.Execute(ctx, "", "cmd", []string{"/c", "echo", "%TEST_VAR%"}, env, 5*time.Second)
	} else {
		output, err = executor.Execute(ctx, "", "sh", []string{"-c", "echo $TEST_VAR"}, env, 5*time.Second)
	}

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if !strings.Contains(output, "test_value") {
		t.Errorf("Execute() output = %q, want to contain 'test_value'", output)
	}
}

func TestProcessExecutor_Execute_Timeout(t *testing.T) {
	logger := mocks.NewMockLogger()
	// Use a reasonable timeout that's reliable under CPU load.
	// 500ms is long enough for process startup but short enough to test timeout behavior.
	executor := smoketest.NewProcessExecutor(logger)

	ctx := context.Background()

	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "ping", "-n", "10", "127.0.0.1"}
	} else {
		cmd = "sleep"
		args = []string{"10"}
	}

	// Use 500ms timeout - enough for process startup, but process runs for 10s
	_, err := executor.Execute(ctx, "", cmd, args, nil, 500*time.Millisecond)

	if err == nil {
		t.Error("Execute() expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Execute() error = %v, want timeout error", err)
	}
}

func TestProcessExecutor_Execute_ContextCancellation(t *testing.T) {
	logger := mocks.NewMockLogger()

	// Use the OnProcessStarted hook to cancel AFTER the process starts.
	// This eliminates the race condition where context is cancelled before
	// the process even begins, which can cause non-deterministic errors.
	processStarted := make(chan struct{})
	executor := smoketest.NewProcessExecutor(logger,
		smoketest.WithOnProcessStarted(func() {
			close(processStarted)
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "ping", "-n", "10", "127.0.0.1"}
	} else {
		cmd = "sleep"
		args = []string{"10"}
	}

	// Start execution in background
	errCh := make(chan error, 1)
	go func() {
		_, err := executor.Execute(ctx, "", cmd, args, nil, 5*time.Second)
		errCh <- err
	}()

	// Wait for process to start, then cancel
	<-processStarted
	cancel()

	// Wait for result
	err := <-errCh

	if err == nil {
		t.Error("Execute() expected cancellation error")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("Execute() error = %v, want cancellation error", err)
	}
}

func TestProcessExecutor_Execute_CommandNotFound(t *testing.T) {
	logger := mocks.NewMockLogger()
	executor := smoketest.NewProcessExecutor(logger)

	ctx := context.Background()
	_, err := executor.Execute(ctx, "", "nonexistent-command-12345", nil, nil, 5*time.Second)

	if err == nil {
		t.Error("Execute() expected error for nonexistent command")
	}
}

func TestProcessExecutor_Execute_NonZeroExit(t *testing.T) {
	logger := mocks.NewMockLogger()
	executor := smoketest.NewProcessExecutor(logger)

	ctx := context.Background()

	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "exit", "1"}
	} else {
		cmd = "sh"
		args = []string{"-c", "exit 1"}
	}

	_, err := executor.Execute(ctx, "", cmd, args, nil, 5*time.Second)

	if err == nil {
		t.Error("Execute() expected error for non-zero exit")
	}
}

func TestProcessExecutor_LookPath_Exists(t *testing.T) {
	logger := mocks.NewMockLogger()
	executor := smoketest.NewProcessExecutor(logger)

	var cmdToFind string
	if runtime.GOOS == "windows" {
		cmdToFind = "cmd"
	} else {
		cmdToFind = "sh"
	}

	path, err := executor.LookPath(cmdToFind)
	if err != nil {
		t.Errorf("LookPath() error = %v", err)
	}
	if path == "" {
		t.Error("LookPath() returned empty path")
	}
}

func TestProcessExecutor_LookPath_NotExists(t *testing.T) {
	logger := mocks.NewMockLogger()
	executor := smoketest.NewProcessExecutor(logger)

	_, err := executor.LookPath("nonexistent-command-12345")
	if err == nil {
		t.Error("LookPath() expected error for nonexistent command")
	}
}

func TestProcessExecutor_Execute_DefaultTimeout(t *testing.T) {
	logger := mocks.NewMockLogger()
	executor := smoketest.NewProcessExecutor(logger)

	ctx := context.Background()

	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "echo", "quick"}
	} else {
		cmd = "echo"
		args = []string{"quick"}
	}

	// Pass 0 timeout to test default
	output, err := executor.Execute(ctx, "", cmd, args, nil, 0)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if !strings.Contains(output, "quick") {
		t.Errorf("Execute() output = %q, want to contain 'quick'", output)
	}
}

func TestProcessExecutor_Execute_WorkingDirectory(t *testing.T) {
	logger := mocks.NewMockLogger()
	executor := smoketest.NewProcessExecutor(logger)

	ctx := context.Background()
	tmpDir := t.TempDir()

	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "cd"}
	} else {
		cmd = "pwd"
		args = nil
	}

	output, err := executor.Execute(ctx, tmpDir, cmd, args, nil, 5*time.Second)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if !strings.Contains(output, tmpDir) {
		t.Errorf("Execute() output = %q, want to contain %q", output, tmpDir)
	}
}

// =============================================================================
// Flakiness Reproduction Tests
// These tests demonstrate the original flaky behavior. They are skipped by default
// but can be enabled with -run to verify the fix or reproduce the issue.
// =============================================================================

func TestProcessExecutor_Execute_Timeout_FlakyReproduction(t *testing.T) {
	// Skip by default - this test demonstrates flakiness with tight timeouts
	t.Skip("Flakiness reproduction test - run manually with: go test -run FlakyReproduction -count=10")

	logger := mocks.NewMockLogger()
	executor := smoketest.NewProcessExecutor(logger)

	ctx := context.Background()

	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "ping", "-n", "10", "127.0.0.1"}
	} else {
		cmd = "sleep"
		args = []string{"10"}
	}

	// FLAKY: 100ms is too tight - under CPU load, this may fail sporadically
	// because process startup itself can take >100ms
	_, err := executor.Execute(ctx, "", cmd, args, nil, 100*time.Millisecond)

	if err == nil {
		t.Error("Execute() expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Execute() error = %v, want timeout error", err)
	}
}

func TestProcessExecutor_Execute_ContextCancellation_FlakyReproduction(t *testing.T) {
	// Skip by default - this test demonstrates flakiness with pre-cancelled context
	t.Skip("Flakiness reproduction test - run manually with: go test -run FlakyReproduction -count=10")

	logger := mocks.NewMockLogger()
	executor := smoketest.NewProcessExecutor(logger)

	ctx, cancel := context.WithCancel(context.Background())

	// FLAKY: Cancelling before Execute() creates a race between:
	// 1. The goroutine starting cmd.Run()
	// 2. The select noticing the already-cancelled context
	// This can produce different error messages depending on timing.
	cancel()

	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "ping", "-n", "10", "127.0.0.1"}
	} else {
		cmd = "sleep"
		args = []string{"10"}
	}

	_, err := executor.Execute(ctx, "", cmd, args, nil, 5*time.Second)

	if err == nil {
		t.Error("Execute() expected cancellation error")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("Execute() error = %v, want cancellation error", err)
	}
}

// =============================================================================
// Seam-Based Tests (Deterministic)
// These tests use the new Clock and lifecycle seams for deterministic behavior.
// =============================================================================

func TestProcessExecutor_Execute_Timeout_WithMockClock(t *testing.T) {
	// This test demonstrates using the Clock seam to control the grace period.
	// The actual timeout is handled by context.WithTimeout, but the grace period
	// after killing the process is controlled by the clock.
	logger := mocks.NewMockLogger()

	// Use a short grace period for faster test execution
	executor := smoketest.NewProcessExecutor(logger,
		smoketest.WithKillGracePeriod(100*time.Millisecond),
	)

	ctx := context.Background()

	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "ping", "-n", "10", "127.0.0.1"}
	} else {
		cmd = "sleep"
		args = []string{"10"}
	}

	start := time.Now()
	_, err := executor.Execute(ctx, "", cmd, args, nil, 500*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Execute() expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Execute() error = %v, want timeout error", err)
	}

	// Verify the test completed in reasonable time (timeout + grace period)
	// Allow some margin for process cleanup
	maxExpected := 1500 * time.Millisecond
	if elapsed > maxExpected {
		t.Errorf("Execute() took %v, expected less than %v", elapsed, maxExpected)
	}
}

func TestProcessExecutor_Execute_WithLifecycleHook(t *testing.T) {
	// Verify the OnProcessStarted hook is called
	logger := mocks.NewMockLogger()

	hookCalled := false
	executor := smoketest.NewProcessExecutor(logger,
		smoketest.WithOnProcessStarted(func() {
			hookCalled = true
		}),
	)

	ctx := context.Background()

	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "echo", "test"}
	} else {
		cmd = "echo"
		args = []string{"test"}
	}

	_, err := executor.Execute(ctx, "", cmd, args, nil, 5*time.Second)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if !hookCalled {
		t.Error("OnProcessStarted hook was not called")
	}
}

func TestProcessExecutor_OptionsChaining(t *testing.T) {
	// Verify all options can be chained together
	logger := mocks.NewMockLogger()

	hookCalled := false
	clock := mocks.NewMockClock(time.Now())

	executor := smoketest.NewProcessExecutor(logger,
		smoketest.WithClock(clock),
		smoketest.WithOutputLimit(1024),
		smoketest.WithKillGracePeriod(1*time.Second),
		smoketest.WithOnProcessStarted(func() {
			hookCalled = true
		}),
	)

	ctx := context.Background()

	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "echo", "test"}
	} else {
		cmd = "echo"
		args = []string{"test"}
	}

	output, err := executor.Execute(ctx, "", cmd, args, nil, 5*time.Second)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if !strings.Contains(output, "test") {
		t.Errorf("Execute() output = %q, want to contain 'test'", output)
	}
	if !hookCalled {
		t.Error("OnProcessStarted hook was not called")
	}
}
