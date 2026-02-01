package smoketest_test

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"
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

	_, err := executor.Execute(ctx, "", cmd, args, nil, 100*time.Millisecond)

	if err == nil {
		t.Error("Execute() expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Execute() error = %v, want timeout error", err)
	}
}

func TestProcessExecutor_Execute_ContextCancellation(t *testing.T) {
	logger := mocks.NewMockLogger()
	executor := smoketest.NewProcessExecutor(logger)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
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
