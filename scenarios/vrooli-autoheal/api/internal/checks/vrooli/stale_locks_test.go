// Package vrooli tests for stale lock health check
// [REQ:STALE-LOCK-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package vrooli

import (
	"context"
	"strings"
	"testing"

	"vrooli-autoheal/internal/checks"
)

// MockVrooliStateReader is a mock implementation for testing
type MockVrooliStateReader struct {
	TrackedProcesses []checks.TrackedProcess
	PortLocks        []checks.PortLock
	ListProcessErr   error
	ListLocksErr     error
	RemoveLockErr    error
	RemovedLocks     []checks.PortLock // Track which locks were removed
}

func (m *MockVrooliStateReader) ListTrackedProcesses() ([]checks.TrackedProcess, error) {
	return m.TrackedProcesses, m.ListProcessErr
}

func (m *MockVrooliStateReader) ListPortLocks() ([]checks.PortLock, error) {
	return m.PortLocks, m.ListLocksErr
}

func (m *MockVrooliStateReader) RemovePortLock(lock checks.PortLock) error {
	if m.RemoveLockErr != nil {
		return m.RemoveLockErr
	}
	m.RemovedLocks = append(m.RemovedLocks, lock)
	return nil
}

// TestStaleLockCheckInterface verifies StaleLockCheck implements Check
// [REQ:STALE-LOCK-001]
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
	// Should run on all platforms
	if check.Platforms() != nil {
		t.Error("StaleLockCheck should run on all platforms")
	}
	if check.Category() != checks.CategoryInfrastructure {
		t.Errorf("Category() = %q, want %q", check.Category(), checks.CategoryInfrastructure)
	}
}

// TestStaleLockCheckHealable verifies StaleLockCheck implements HealableCheck
// [REQ:HEAL-ACTION-001]
func TestStaleLockCheckHealable(t *testing.T) {
	var _ checks.HealableCheck = (*StaleLockCheck)(nil)

	check := NewStaleLockCheck()

	// Test recovery actions with nil result
	actions := check.RecoveryActions(nil)
	if len(actions) == 0 {
		t.Error("RecoveryActions() should return actions")
	}

	// Verify expected actions exist
	expectedActions := map[string]bool{
		"list":  false,
		"clean": false,
	}
	for _, action := range actions {
		if _, exists := expectedActions[action.ID]; exists {
			expectedActions[action.ID] = true
		}
	}
	for id, found := range expectedActions {
		if !found {
			t.Errorf("Expected action %q not found in RecoveryActions()", id)
		}
	}
}

// TestStaleLockCheckRunWithMock tests StaleLockCheck.Run() using mock state reader
// [REQ:STALE-LOCK-001] [REQ:TEST-SEAM-001]
func TestStaleLockCheckRunWithMock(t *testing.T) {
	tests := []struct {
		name           string
		portLocks      []checks.PortLock
		listLocksErr   error
		expectedStatus checks.Status
		expectMsgPart  string
	}{
		{
			name:           "no locks",
			portLocks:      nil,
			listLocksErr:   nil,
			expectedStatus: checks.StatusOK,
			expectMsgPart:  "No stale port locks",
		},
		{
			name: "all locks valid - running PIDs",
			portLocks: []checks.PortLock{
				{Port: 8080, Scenario: "test-app", PID: 1, FilePath: "/tmp/lock1"}, // PID 1 always exists
			},
			listLocksErr:   nil,
			expectedStatus: checks.StatusOK,
			expectMsgPart:  "No stale port locks",
		},
		{
			name: "one stale lock - dead PID",
			portLocks: []checks.PortLock{
				{Port: 8080, Scenario: "test-app", PID: 99999999, FilePath: "/tmp/lock1"}, // Non-existent PID
			},
			listLocksErr:   nil,
			expectedStatus: checks.StatusOK, // 1 stale lock is below warning threshold
			expectMsgPart:  "1 stale port locks",
		},
		{
			name: "multiple stale locks - warning threshold",
			portLocks: []checks.PortLock{
				{Port: 8080, Scenario: "app1", PID: 99999999, FilePath: "/tmp/lock1"},
				{Port: 8081, Scenario: "app2", PID: 99999998, FilePath: "/tmp/lock2"},
				{Port: 8082, Scenario: "app3", PID: 99999997, FilePath: "/tmp/lock3"},
			},
			listLocksErr:   nil,
			expectedStatus: checks.StatusWarning, // 3 stale locks hits warning threshold
			expectMsgPart:  "3 stale port locks",
		},
		{
			name:           "error reading locks",
			portLocks:      nil,
			listLocksErr:   checks.ErrCommandNotFound,
			expectedStatus: checks.StatusCritical,
			expectMsgPart:  "Failed to read port locks",
		},
		{
			name: "stale lock with zero PID",
			portLocks: []checks.PortLock{
				{Port: 8080, Scenario: "test-app", PID: 0, FilePath: "/tmp/lock1"}, // Invalid PID
			},
			listLocksErr:   nil,
			expectedStatus: checks.StatusOK,
			expectMsgPart:  "1 stale port locks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReader := &MockVrooliStateReader{
				PortLocks:    tt.portLocks,
				ListLocksErr: tt.listLocksErr,
			}

			check := NewStaleLockCheck(WithStaleLockStateReader(mockReader))
			result := check.Run(context.Background())

			if result.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.expectedStatus)
			}
			if !strings.Contains(result.Message, tt.expectMsgPart) {
				t.Errorf("Message = %q, want to contain %q", result.Message, tt.expectMsgPart)
			}
		})
	}
}

// TestStaleLockCheckThresholds tests custom threshold configuration
// [REQ:STALE-LOCK-001]
func TestStaleLockCheckThresholds(t *testing.T) {
	mockReader := &MockVrooliStateReader{
		PortLocks: []checks.PortLock{
			{Port: 8080, Scenario: "app1", PID: 99999999, FilePath: "/tmp/lock1"},
			{Port: 8081, Scenario: "app2", PID: 99999998, FilePath: "/tmp/lock2"},
		},
	}

	// With default thresholds (warning=3, critical=10), 2 stale locks is OK
	checkDefault := NewStaleLockCheck(WithStaleLockStateReader(mockReader))
	resultDefault := checkDefault.Run(context.Background())
	if resultDefault.Status != checks.StatusOK {
		t.Errorf("Default thresholds: Status = %v, want %v", resultDefault.Status, checks.StatusOK)
	}

	// With custom thresholds (warning=1, critical=5), 2 stale locks is Warning
	checkCustom := NewStaleLockCheck(
		WithStaleLockStateReader(mockReader),
		WithStaleLockThresholds(1, 5),
	)
	resultCustom := checkCustom.Run(context.Background())
	if resultCustom.Status != checks.StatusWarning {
		t.Errorf("Custom thresholds: Status = %v, want %v", resultCustom.Status, checks.StatusWarning)
	}
}

// TestStaleLockCheckExecuteActionList tests the list action
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestStaleLockCheckExecuteActionList(t *testing.T) {
	mockReader := &MockVrooliStateReader{
		PortLocks: []checks.PortLock{
			{Port: 8080, Scenario: "test-app", PID: 1, FilePath: "/tmp/lock1"},          // Valid (PID 1 exists)
			{Port: 9000, Scenario: "other-app", PID: 99999999, FilePath: "/tmp/lock2"}, // Stale
		},
	}

	check := NewStaleLockCheck(WithStaleLockStateReader(mockReader))
	result := check.ExecuteAction(context.Background(), "list")

	if !result.Success {
		t.Errorf("Success = false, want true")
	}
	if !strings.Contains(result.Output, "Port 8080") {
		t.Errorf("Output should contain Port 8080")
	}
	if !strings.Contains(result.Output, "VALID") {
		t.Errorf("Output should contain VALID for running process")
	}
	if !strings.Contains(result.Output, "STALE") {
		t.Errorf("Output should contain STALE for dead process")
	}
	if !strings.Contains(result.Message, "1 stale locks") {
		t.Errorf("Message = %q, want to contain '1 stale locks'", result.Message)
	}
}

// TestStaleLockCheckExecuteActionClean tests the clean action
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestStaleLockCheckExecuteActionClean(t *testing.T) {
	mockReader := &MockVrooliStateReader{
		PortLocks: []checks.PortLock{
			{Port: 8080, Scenario: "test-app", PID: 1, FilePath: "/tmp/lock1"},          // Valid (PID 1 exists)
			{Port: 9000, Scenario: "other-app", PID: 99999999, FilePath: "/tmp/lock2"}, // Stale
		},
	}

	check := NewStaleLockCheck(WithStaleLockStateReader(mockReader))
	result := check.ExecuteAction(context.Background(), "clean")

	if !result.Success {
		t.Errorf("Success = false, want true")
	}
	if !strings.Contains(result.Message, "Cleaned 1 stale port locks") {
		t.Errorf("Message = %q, want to contain 'Cleaned 1 stale port locks'", result.Message)
	}

	// Verify only stale lock was removed
	if len(mockReader.RemovedLocks) != 1 {
		t.Errorf("RemovedLocks count = %d, want 1", len(mockReader.RemovedLocks))
	}
	if len(mockReader.RemovedLocks) > 0 && mockReader.RemovedLocks[0].Port != 9000 {
		t.Errorf("Removed lock port = %d, want 9000", mockReader.RemovedLocks[0].Port)
	}
}

// TestStaleLockCheckExecuteActionClean_NoStale tests clean with no stale locks
// [REQ:HEAL-ACTION-001]
func TestStaleLockCheckExecuteActionClean_NoStale(t *testing.T) {
	mockReader := &MockVrooliStateReader{
		PortLocks: []checks.PortLock{
			{Port: 8080, Scenario: "test-app", PID: 1, FilePath: "/tmp/lock1"}, // Valid
		},
	}

	check := NewStaleLockCheck(WithStaleLockStateReader(mockReader))
	result := check.ExecuteAction(context.Background(), "clean")

	if !result.Success {
		t.Errorf("Success = false, want true")
	}
	if !strings.Contains(result.Message, "Cleaned 0 stale port locks") {
		t.Errorf("Message = %q, want to contain 'Cleaned 0 stale port locks'", result.Message)
	}
	if len(mockReader.RemovedLocks) != 0 {
		t.Errorf("RemovedLocks count = %d, want 0", len(mockReader.RemovedLocks))
	}
}

// TestStaleLockCheckExecuteActionUnknown tests unknown action handling
// [REQ:HEAL-ACTION-001]
func TestStaleLockCheckExecuteActionUnknown(t *testing.T) {
	check := NewStaleLockCheck()
	result := check.ExecuteAction(context.Background(), "invalid-action")

	if result.Success {
		t.Error("Success should be false for unknown action")
	}
	if result.Error != "unknown action: invalid-action" {
		t.Errorf("Error = %q, want %q", result.Error, "unknown action: invalid-action")
	}
}

// TestStaleLockCheckRecoveryActionsDangerous tests dangerous action marking
// [REQ:HEAL-ACTION-001]
func TestStaleLockCheckRecoveryActionsDangerous(t *testing.T) {
	check := NewStaleLockCheck()
	actions := check.RecoveryActions(nil)

	actionMap := make(map[string]checks.RecoveryAction)
	for _, a := range actions {
		actionMap[a.ID] = a
	}

	// Both actions should be safe (not dangerous)
	for id, action := range actionMap {
		if action.Dangerous {
			t.Errorf("%s action should not be dangerous", id)
		}
	}
}

// TestStaleLockCheckRecoveryActionsAvailability tests action availability based on state
// [REQ:HEAL-ACTION-001]
func TestStaleLockCheckRecoveryActionsAvailability(t *testing.T) {
	check := NewStaleLockCheck()

	// With no stale locks, clean should not be available
	noStaleResult := &checks.Result{
		Details: map[string]interface{}{"staleCount": 0},
	}
	actionsNoStale := check.RecoveryActions(noStaleResult)
	for _, action := range actionsNoStale {
		if action.ID == "clean" && action.Available {
			t.Error("clean action should not be available when no stale locks")
		}
		if action.ID == "list" && !action.Available {
			t.Error("list action should always be available")
		}
	}

	// With stale locks, clean should be available
	hasStaleResult := &checks.Result{
		Details: map[string]interface{}{"staleCount": 3},
	}
	actionsHasStale := check.RecoveryActions(hasStaleResult)
	for _, action := range actionsHasStale {
		if action.ID == "clean" && !action.Available {
			t.Error("clean action should be available when stale locks exist")
		}
	}
}

// TestStaleLockCheckHealthMetrics tests that health metrics are properly set
// [REQ:STALE-LOCK-001]
func TestStaleLockCheckHealthMetrics(t *testing.T) {
	mockReader := &MockVrooliStateReader{
		PortLocks: []checks.PortLock{
			{Port: 8080, Scenario: "app1", PID: 99999999, FilePath: "/tmp/lock1"}, // Stale
			{Port: 8081, Scenario: "app2", PID: 99999998, FilePath: "/tmp/lock2"}, // Stale
		},
	}

	check := NewStaleLockCheck(WithStaleLockStateReader(mockReader))
	result := check.Run(context.Background())

	if result.Metrics == nil {
		t.Fatal("Metrics should not be nil")
	}
	if result.Metrics.Score == nil {
		t.Fatal("Score should not be nil")
	}

	// With 2 stale locks, score should be 100 - (2 * 10) = 80
	expectedScore := 80
	if *result.Metrics.Score != expectedScore {
		t.Errorf("Score = %d, want %d", *result.Metrics.Score, expectedScore)
	}

	if len(result.Metrics.SubChecks) == 0 {
		t.Error("SubChecks should not be empty")
	}
}

// TestStaleLockCheckDetails tests that result details are properly populated
// [REQ:STALE-LOCK-001]
func TestStaleLockCheckDetails(t *testing.T) {
	mockReader := &MockVrooliStateReader{
		PortLocks: []checks.PortLock{
			{Port: 8080, Scenario: "app1", PID: 1, FilePath: "/tmp/lock1"},          // Valid
			{Port: 9000, Scenario: "app2", PID: 99999999, FilePath: "/tmp/lock2"},   // Stale
			{Port: 9001, Scenario: "app3", PID: 99999998, FilePath: "/tmp/lock3"},   // Stale
		},
	}

	check := NewStaleLockCheck(WithStaleLockStateReader(mockReader))
	result := check.Run(context.Background())

	// Verify details are populated
	if staleCount, ok := result.Details["staleCount"].(int); !ok || staleCount != 2 {
		t.Errorf("staleCount = %v, want 2", result.Details["staleCount"])
	}
	if totalLocks, ok := result.Details["totalLocks"].(int); !ok || totalLocks != 3 {
		t.Errorf("totalLocks = %v, want 3", result.Details["totalLocks"])
	}
	if _, ok := result.Details["staleLocks"]; !ok {
		t.Error("staleLocks should be in details")
	}
}
