// Package vrooli tests for stale lock health check
// [REQ:STALE-LOCK-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package vrooli

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"
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
			name:           "all healthy claims",
			output:         `{"success":true,"registry_claims":[{"port":17701,"scenario":"alpha","recommendation_code":"port-ok"}]}`,
			expectedStatus: checks.StatusOK,
			expectedMsg:    "No stale registry claims detected",
			expectedCount:  0,
		},
		{
			name:           "one stale claim below threshold",
			output:         `{"success":true,"registry_claims":[{"port":17701,"scenario":"alpha","recommendation_code":"port-ok"},{"port":17702,"scenario":"beta","claim_status":"bound","instance_status":"stopped","reconciliation":"stale_claim","recommendation_code":"stale-claim-expire"}]}`,
			expectedStatus: checks.StatusOK,
			expectedMsg:    "1 stale registry claims (below threshold)",
			expectedCount:  1,
		},
		{
			name:           "warning threshold",
			output:         `{"success":true,"registry_claims":[{"port":1,"reconciliation":"stale_claim","recommendation_code":"stale-claim-expire"},{"port":2,"reconciliation":"stale_instance","recommendation_code":"stale-claim-expire"},{"port":3,"reconciliation":"stale_claim","recommendation_code":"stale-claim-expire"}]}`,
			expectedStatus: checks.StatusWarning,
			expectedMsg:    "Warning: 3 stale registry claims detected",
			expectedCount:  3,
		},
		{
			name:           "command error",
			err:            testutil.ErrCommandNotFound,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "Failed to read registry claims",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := testutil.NewMockExecutor()
			executor.Responses["vrooli locks --json"] = testutil.MockResponse{
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
	executor := testutil.NewMockExecutor()
	executor.Responses["vrooli locks --json"] = testutil.MockResponse{
		Output: []byte(`{"success":true,"registry_claims":[{"port":17701,"scenario":"alpha","recommendation_code":"port-ok"},{"port":17702,"scenario":"beta","reconciliation":"stale_claim","recommendation_code":"stale-claim-expire"}]}`),
	}

	check := NewStaleLockCheck(WithStaleLockExecutor(executor))
	result := check.ExecuteAction(context.Background(), "list")
	if !result.Success {
		t.Fatalf("expected success, got error %q", result.Error)
	}
	if result.Message != "Found 1 stale registry claims out of 2 total" {
		t.Fatalf("Message = %q", result.Message)
	}
	if !containsText(result.Output, "stale-claim-expire") {
		t.Fatalf("Output = %q, want diagnostic JSON to include the stale claim recommendation", result.Output)
	}
}

func TestStaleLockCheckExecuteActionCleanDelegatesToCoreCleanup(t *testing.T) {
	executor := testutil.NewMockExecutor()
	executor.Responses["vrooli cleanup locks --json"] = testutil.MockResponse{
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
	executor := testutil.NewMockExecutor()
	executor.Responses["vrooli diagnose-port 8080 alpha --json"] = testutil.MockResponse{
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
