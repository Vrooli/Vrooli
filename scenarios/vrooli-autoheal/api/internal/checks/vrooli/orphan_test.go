// Package vrooli tests for orphan process health check
// [REQ:ORPHAN-CHECK-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package vrooli

import (
	"context"
	"strings"
	"testing"

	"vrooli-autoheal/internal/checks"
)

func TestOrphanCheckInterface(t *testing.T) {
	var _ checks.Check = (*OrphanCheck)(nil)

	check := NewOrphanCheck()
	if check.ID() != "vrooli-orphans" {
		t.Errorf("ID() = %q, want %q", check.ID(), "vrooli-orphans")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	if check.IntervalSeconds() <= 0 {
		t.Error("IntervalSeconds() should be positive")
	}
	if check.Category() != checks.CategoryInfrastructure {
		t.Errorf("Category() = %q, want %q", check.Category(), checks.CategoryInfrastructure)
	}
	if len(check.Platforms()) == 0 {
		t.Error("OrphanCheck should have platform restrictions")
	}
}

func TestOrphanCheckHealable(t *testing.T) {
	var _ checks.HealableCheck = (*OrphanCheck)(nil)

	check := NewOrphanCheck()
	actions := check.RecoveryActions(nil)
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}

	expected := map[string]bool{
		"list": false,
		"kill": false,
	}
	for _, action := range actions {
		if _, ok := expected[action.ID]; ok {
			expected[action.ID] = true
		}
	}
	for id, found := range expected {
		if !found {
			t.Errorf("expected action %q not found", id)
		}
	}
}

func TestOrphanCheckRunWithCoreJSON(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		err            error
		expectedStatus checks.Status
		expectedMsg    string
		expectedCount  int
	}{
		{
			name:           "no orphans",
			output:         `{"success":true,"orphans":[]}`,
			expectedStatus: checks.StatusOK,
			expectedMsg:    "No orphan Vrooli processes detected",
			expectedCount:  0,
		},
		{
			name:           "warning threshold",
			output:         `{"success":true,"orphans":[{"pid":100,"ppid":1,"command":"alpha"},{"pid":101,"ppid":1,"command":"beta"},{"pid":102,"ppid":1,"command":"gamma"}]}`,
			expectedStatus: checks.StatusWarning,
			expectedMsg:    "Warning: 3 orphan Vrooli processes detected",
			expectedCount:  3,
		},
		{
			name:           "critical threshold",
			output:         `{"success":true,"orphans":[{"pid":1,"ppid":0,"command":"a"},{"pid":2,"ppid":0,"command":"b"},{"pid":3,"ppid":0,"command":"c"},{"pid":4,"ppid":0,"command":"d"},{"pid":5,"ppid":0,"command":"e"},{"pid":6,"ppid":0,"command":"f"},{"pid":7,"ppid":0,"command":"g"},{"pid":8,"ppid":0,"command":"h"},{"pid":9,"ppid":0,"command":"i"},{"pid":10,"ppid":0,"command":"j"}]}`,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "Critical: 10 orphan Vrooli processes detected",
			expectedCount:  10,
		},
		{
			name:           "command error",
			err:            checks.ErrCommandNotFound,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "Failed to read orphan process status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := checks.NewMockExecutor()
			executor.Responses["vrooli orphans --json"] = checks.MockResponse{
				Output: []byte(tt.output),
				Error:  tt.err,
			}

			check := NewOrphanCheck(WithOrphanExecutor(executor))
			result := check.Run(context.Background())
			if result.Status != tt.expectedStatus {
				t.Fatalf("Status = %v, want %v", result.Status, tt.expectedStatus)
			}
			if result.Message != tt.expectedMsg {
				t.Fatalf("Message = %q, want %q", result.Message, tt.expectedMsg)
			}
			if tt.err == nil {
				if got, _ := result.Details["orphanCount"].(int); got != tt.expectedCount {
					t.Fatalf("orphanCount = %d, want %d", got, tt.expectedCount)
				}
			}
		})
	}
}

func TestOrphanCheckExecuteActionList(t *testing.T) {
	executor := checks.NewMockExecutor()
	executor.Responses["vrooli orphans --json"] = checks.MockResponse{
		Output: []byte(`{"success":true,"orphans":[{"pid":100,"ppid":1,"command":"alpha"}]}`),
	}

	check := NewOrphanCheck(WithOrphanExecutor(executor))
	result := check.ExecuteAction(context.Background(), "list")
	if !result.Success {
		t.Fatalf("expected success, got error %q", result.Error)
	}
	if result.Message != "Found 1 orphan processes" {
		t.Fatalf("Message = %q", result.Message)
	}
	if !strings.Contains(result.Output, `"command":"alpha"`) {
		t.Fatalf("Output = %q, want raw JSON to include alpha", result.Output)
	}
}

func TestOrphanCheckExecuteActionKillDelegatesToCoreCleanup(t *testing.T) {
	executor := checks.NewMockExecutor()
	executor.Responses["vrooli cleanup orphans --json"] = checks.MockResponse{
		Output: []byte(`{"success":true,"data":{"stopped":[{"name":"100","message":"alpha"}],"failed":[],"message":"Stopped 1 processes (0 failed)"}}`),
	}

	check := NewOrphanCheck(WithOrphanExecutor(executor))
	result := check.ExecuteAction(context.Background(), "kill")
	if !result.Success {
		t.Fatalf("expected success, got error %q", result.Error)
	}
	if result.Message != "Stopped 1 processes (0 failed)" {
		t.Fatalf("Message = %q", result.Message)
	}
	if got := len(executor.Calls); got != 1 {
		t.Fatalf("expected 1 executor call, got %d", got)
	}
	call := executor.Calls[0]
	if call.Name != "vrooli" || strings.Join(call.Args, " ") != "cleanup orphans --json" {
		t.Fatalf("unexpected command: %s %s", call.Name, strings.Join(call.Args, " "))
	}
}
