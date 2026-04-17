// Package vrooli tests for stale lock health check
// [REQ:STALE-LOCK-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package vrooli

import (
	"context"
	"strings"
	"testing"

	"vrooli-autoheal/internal/checks"
)

func TestStaleLockCheckInterface(t *testing.T) {
	var _ checks.Check = (*StaleLockCheck)(nil)

	check := NewStaleLockCheck()
	if check.ID() != "vrooli-stale-locks" {
		t.Errorf("ID() = %q, want %q", check.ID(), "vrooli-stale-locks")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	if check.IntervalSeconds() <= 0 {
		t.Error("IntervalSeconds() should be positive")
	}
	if check.Platforms() != nil {
		t.Error("StaleLockCheck should run on all platforms")
	}
	if check.Category() != checks.CategoryInfrastructure {
		t.Errorf("Category() = %q, want %q", check.Category(), checks.CategoryInfrastructure)
	}
}

func TestStaleLockCheckHealable(t *testing.T) {
	var _ checks.HealableCheck = (*StaleLockCheck)(nil)

	check := NewStaleLockCheck()
	actions := check.RecoveryActions(nil)
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
}

func TestStaleLockCheckRunWithCoreJSON(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		err            error
		expectedStatus checks.Status
		expectedMsg    string
		expectedCount  int
	}{
		{
			name:           "no locks",
			output:         `{"success":true,"locks":[]}`,
			expectedStatus: checks.StatusOK,
			expectedMsg:    "No stale port locks detected",
			expectedCount:  0,
		},
		{
			name:           "mixed locks",
			output:         `{"success":true,"locks":[{"port":8080,"scenario":"alpha","pid":100,"path":"/tmp/a","owner_running":true,"stale":false},{"port":8081,"scenario":"beta","pid":0,"path":"/tmp/b","owner_running":false,"stale":true}]}`,
			expectedStatus: checks.StatusOK,
			expectedMsg:    "1 stale port locks (below threshold)",
			expectedCount:  1,
		},
		{
			name:           "warning threshold",
			output:         `{"success":true,"locks":[{"port":1,"path":"/tmp/1","stale":true},{"port":2,"path":"/tmp/2","stale":true},{"port":3,"path":"/tmp/3","stale":true}]}`,
			expectedStatus: checks.StatusWarning,
			expectedMsg:    "Warning: 3 stale port locks detected",
			expectedCount:  3,
		},
		{
			name:           "command error",
			err:            checks.ErrCommandNotFound,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "Failed to read port locks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := checks.NewMockExecutor()
			executor.Responses["vrooli locks --json"] = checks.MockResponse{
				Output: []byte(tt.output),
				Error:  tt.err,
			}

			check := NewStaleLockCheck(WithStaleLockExecutor(executor))
			result := check.Run(context.Background())
			if result.Status != tt.expectedStatus {
				t.Fatalf("Status = %v, want %v", result.Status, tt.expectedStatus)
			}
			if result.Message != tt.expectedMsg {
				t.Fatalf("Message = %q, want %q", result.Message, tt.expectedMsg)
			}
			if tt.err == nil {
				if got, _ := result.Details["staleCount"].(int); got != tt.expectedCount {
					t.Fatalf("staleCount = %d, want %d", got, tt.expectedCount)
				}
			}
		})
	}
}

func TestStaleLockCheckExecuteActionList(t *testing.T) {
	executor := checks.NewMockExecutor()
	executor.Responses["vrooli locks --json"] = checks.MockResponse{
		Output: []byte(`{"success":true,"locks":[{"port":8080,"path":"/tmp/a","stale":false},{"port":8081,"path":"/tmp/b","stale":true}]}`),
	}

	check := NewStaleLockCheck(WithStaleLockExecutor(executor))
	result := check.ExecuteAction(context.Background(), "list")
	if !result.Success {
		t.Fatalf("expected success, got error %q", result.Error)
	}
	if result.Message != "Found 1 stale locks out of 2 total" {
		t.Fatalf("Message = %q", result.Message)
	}
	if !strings.Contains(result.Output, `"port":8081`) {
		t.Fatalf("Output = %q", result.Output)
	}
}

func TestStaleLockCheckExecuteActionCleanDelegatesToCoreCleanup(t *testing.T) {
	executor := checks.NewMockExecutor()
	executor.Responses["vrooli cleanup locks --json"] = checks.MockResponse{
		Output: []byte(`{"success":true,"data":{"stopped":[{"name":"8081","message":"Removed stale lock"}],"failed":[],"message":"Stopped 1 processes (0 failed)"}}`),
	}

	check := NewStaleLockCheck(WithStaleLockExecutor(executor))
	result := check.ExecuteAction(context.Background(), "clean")
	if !result.Success {
		t.Fatalf("expected success, got error %q", result.Error)
	}
	if result.Message != "Stopped 1 processes (0 failed)" {
		t.Fatalf("Message = %q", result.Message)
	}
	call := executor.Calls[0]
	if call.Name != "vrooli" || strings.Join(call.Args, " ") != "cleanup locks --json" {
		t.Fatalf("unexpected command: %s %s", call.Name, strings.Join(call.Args, " "))
	}
}

func TestDiagnosePortDelegatesToCoreCommand(t *testing.T) {
	executor := checks.NewMockExecutor()
	executor.Responses["vrooli diagnose-port 8080 alpha --json"] = checks.MockResponse{
		Output: []byte(`{"success":true,"diagnostic":{"port":8080,"scenario":"alpha","in_use":true,"listener_inspection":{"available":true},"host_orphan_count":2,"recommendations":["Inspect listener"]}}`),
	}

	diagnostic, err := DiagnosePort(8080, "alpha", executor)
	if err != nil {
		t.Fatalf("DiagnosePort returned error: %v", err)
	}
	if diagnostic.Port != 8080 || diagnostic.Scenario != "alpha" {
		t.Fatalf("unexpected diagnostic: %+v", diagnostic)
	}
}
