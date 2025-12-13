// Package vrooli tests for orphan process health check
// [REQ:ORPHAN-CHECK-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package vrooli

import (
	"context"
	"strings"
	"testing"

	"vrooli-autoheal/internal/checks"
)

// MockProcReader is a mock implementation of ProcReader for testing
type MockProcReader struct {
	Processes []checks.ProcessInfo
	Error     error
}

func (m *MockProcReader) ReadMeminfo() (*checks.MemInfo, error) {
	return &checks.MemInfo{}, nil
}

func (m *MockProcReader) ListProcesses() ([]checks.ProcessInfo, error) {
	return m.Processes, m.Error
}

// TestOrphanCheckInterface verifies OrphanCheck implements Check
// [REQ:ORPHAN-CHECK-001]
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
	// Should only run on Linux
	platforms := check.Platforms()
	if len(platforms) == 0 {
		t.Error("OrphanCheck should have platform restrictions")
	}
}

// TestOrphanCheckHealable verifies OrphanCheck implements HealableCheck
// [REQ:HEAL-ACTION-001]
func TestOrphanCheckHealable(t *testing.T) {
	var _ checks.HealableCheck = (*OrphanCheck)(nil)

	check := NewOrphanCheck()

	// Test recovery actions with nil result
	actions := check.RecoveryActions(nil)
	if len(actions) == 0 {
		t.Error("RecoveryActions() should return actions")
	}

	// Verify expected actions exist
	expectedActions := map[string]bool{
		"list": false,
		"kill": false,
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

// TestOrphanCheckRunWithMock tests OrphanCheck.Run() using mock readers
// [REQ:ORPHAN-CHECK-001] [REQ:TEST-SEAM-001]
func TestOrphanCheckRunWithMock(t *testing.T) {
	tests := []struct {
		name             string
		trackedProcesses []checks.TrackedProcess
		runningProcesses []checks.ProcessInfo
		stateReaderErr   error
		procReaderErr    error
		expectedStatus   checks.Status
		expectMsgPart    string
	}{
		{
			name:             "no processes",
			trackedProcesses: nil,
			runningProcesses: nil,
			expectedStatus:   checks.StatusOK,
			expectMsgPart:    "No orphan Vrooli processes",
		},
		{
			name: "all processes tracked",
			trackedProcesses: []checks.TrackedProcess{
				{PID: 100, PGID: 100, Scenario: "app-monitor"},
			},
			runningProcesses: []checks.ProcessInfo{
				{PID: 100, PPid: 1, Comm: "vrooli-app", State: "S"},
			},
			expectedStatus: checks.StatusOK,
			expectMsgPart:  "No orphan Vrooli processes",
		},
		{
			name: "one orphan process",
			trackedProcesses: []checks.TrackedProcess{
				{PID: 100, PGID: 100, Scenario: "app-monitor"},
			},
			runningProcesses: []checks.ProcessInfo{
				{PID: 100, PPid: 1, Comm: "vrooli-app", State: "S"},
				{PID: 200, PPid: 1, Comm: "vrooli-orphan", State: "S"}, // Orphan - not tracked
			},
			expectedStatus: checks.StatusOK, // 1 orphan below warning threshold
			expectMsgPart:  "1 orphan Vrooli processes",
		},
		{
			name: "multiple orphans - warning threshold",
			trackedProcesses: []checks.TrackedProcess{
				{PID: 100, PGID: 100, Scenario: "app-monitor"},
			},
			runningProcesses: []checks.ProcessInfo{
				{PID: 100, PPid: 1, Comm: "vrooli-app", State: "S"},
				{PID: 200, PPid: 1, Comm: "vrooli-orphan1", State: "S"},
				{PID: 201, PPid: 1, Comm: "vrooli-orphan2", State: "S"},
				{PID: 202, PPid: 1, Comm: "vrooli-orphan3", State: "S"},
			},
			expectedStatus: checks.StatusWarning, // 3 orphans hits warning threshold
			expectMsgPart:  "3 orphan Vrooli processes",
		},
		{
			name:             "error reading state",
			trackedProcesses: nil,
			runningProcesses: nil,
			stateReaderErr:   checks.ErrCommandNotFound,
			expectedStatus:   checks.StatusCritical,
			expectMsgPart:    "Failed to read tracked processes",
		},
		{
			name:             "error reading processes",
			trackedProcesses: []checks.TrackedProcess{},
			runningProcesses: nil,
			procReaderErr:    checks.ErrCommandNotFound,
			expectedStatus:   checks.StatusCritical,
			expectMsgPart:    "Failed to list running processes",
		},
		{
			name: "child of tracked process is not orphan",
			trackedProcesses: []checks.TrackedProcess{
				{PID: 100, PGID: 100, Scenario: "app-monitor"},
			},
			runningProcesses: []checks.ProcessInfo{
				{PID: 100, PPid: 1, Comm: "vrooli-parent", State: "S"},
				{PID: 200, PPid: 100, Comm: "vrooli-child", State: "S"}, // Child of tracked - not orphan
			},
			expectedStatus: checks.StatusOK,
			expectMsgPart:  "No orphan Vrooli processes",
		},
		{
			name: "non-vrooli process ignored",
			trackedProcesses: []checks.TrackedProcess{},
			runningProcesses: []checks.ProcessInfo{
				{PID: 100, PPid: 1, Comm: "nginx", State: "S"},       // Not a Vrooli process
				{PID: 200, PPid: 1, Comm: "postgresql", State: "S"}, // Not a Vrooli process
			},
			expectedStatus: checks.StatusOK,
			expectMsgPart:  "No orphan Vrooli processes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStateReader := &MockVrooliStateReader{
				TrackedProcesses: tt.trackedProcesses,
				ListProcessErr:   tt.stateReaderErr,
			}
			mockProcReader := &MockProcReader{
				Processes: tt.runningProcesses,
				Error:     tt.procReaderErr,
			}

			check := NewOrphanCheck(
				WithOrphanStateReader(mockStateReader),
				WithOrphanProcReader(mockProcReader),
			)
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

// TestOrphanCheckThresholds tests custom threshold configuration
// [REQ:ORPHAN-CHECK-001]
func TestOrphanCheckThresholds(t *testing.T) {
	mockStateReader := &MockVrooliStateReader{
		TrackedProcesses: []checks.TrackedProcess{},
	}
	mockProcReader := &MockProcReader{
		Processes: []checks.ProcessInfo{
			{PID: 100, PPid: 1, Comm: "vrooli-orphan1", State: "S"},
			{PID: 101, PPid: 1, Comm: "vrooli-orphan2", State: "S"},
		},
	}

	// With default thresholds (warning=3, critical=10), 2 orphans is OK
	checkDefault := NewOrphanCheck(
		WithOrphanStateReader(mockStateReader),
		WithOrphanProcReader(mockProcReader),
	)
	resultDefault := checkDefault.Run(context.Background())
	if resultDefault.Status != checks.StatusOK {
		t.Errorf("Default thresholds: Status = %v, want %v", resultDefault.Status, checks.StatusOK)
	}

	// With custom thresholds (warning=1, critical=5), 2 orphans is Warning
	checkCustom := NewOrphanCheck(
		WithOrphanStateReader(mockStateReader),
		WithOrphanProcReader(mockProcReader),
		WithOrphanThresholds(1, 5),
	)
	resultCustom := checkCustom.Run(context.Background())
	if resultCustom.Status != checks.StatusWarning {
		t.Errorf("Custom thresholds: Status = %v, want %v", resultCustom.Status, checks.StatusWarning)
	}
}

// TestOrphanCheckExecuteActionList tests the list action
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestOrphanCheckExecuteActionList(t *testing.T) {
	mockStateReader := &MockVrooliStateReader{
		TrackedProcesses: []checks.TrackedProcess{
			{PID: 100, PGID: 100, Scenario: "app-monitor"},
		},
	}
	mockProcReader := &MockProcReader{
		Processes: []checks.ProcessInfo{
			{PID: 100, PPid: 1, Comm: "vrooli-tracked", State: "S"},
			{PID: 200, PPid: 1, Comm: "vrooli-orphan", State: "S"},
		},
	}

	check := NewOrphanCheck(
		WithOrphanStateReader(mockStateReader),
		WithOrphanProcReader(mockProcReader),
	)
	result := check.ExecuteAction(context.Background(), "list")

	if !result.Success {
		t.Errorf("Success = false, want true")
	}
	if !strings.Contains(result.Output, "TRACKED") {
		t.Errorf("Output should contain TRACKED")
	}
	if !strings.Contains(result.Output, "ORPHAN") {
		t.Errorf("Output should contain ORPHAN")
	}
	if !strings.Contains(result.Message, "1 orphan") {
		t.Errorf("Message = %q, want to contain '1 orphan'", result.Message)
	}
}

// TestOrphanCheckExecuteActionUnknown tests unknown action handling
// [REQ:HEAL-ACTION-001]
func TestOrphanCheckExecuteActionUnknown(t *testing.T) {
	check := NewOrphanCheck()
	result := check.ExecuteAction(context.Background(), "invalid-action")

	if result.Success {
		t.Error("Success should be false for unknown action")
	}
	if result.Error != "unknown action: invalid-action" {
		t.Errorf("Error = %q, want %q", result.Error, "unknown action: invalid-action")
	}
}

// TestOrphanCheckRecoveryActionsDangerous tests dangerous action marking
// [REQ:HEAL-ACTION-001]
func TestOrphanCheckRecoveryActionsDangerous(t *testing.T) {
	check := NewOrphanCheck()
	actions := check.RecoveryActions(nil)

	actionMap := make(map[string]checks.RecoveryAction)
	for _, a := range actions {
		actionMap[a.ID] = a
	}

	// list should be safe
	if action, ok := actionMap["list"]; ok {
		if action.Dangerous {
			t.Error("list action should not be dangerous")
		}
	}

	// kill should be dangerous
	if action, ok := actionMap["kill"]; ok {
		if !action.Dangerous {
			t.Error("kill action should be dangerous")
		}
	}
}

// TestOrphanCheckRecoveryActionsAvailability tests action availability based on state
// [REQ:HEAL-ACTION-001]
func TestOrphanCheckRecoveryActionsAvailability(t *testing.T) {
	check := NewOrphanCheck()

	// With no orphans, kill should not be available
	noOrphansResult := &checks.Result{
		Details: map[string]interface{}{"orphanCount": 0},
	}
	actionsNoOrphans := check.RecoveryActions(noOrphansResult)
	for _, action := range actionsNoOrphans {
		if action.ID == "kill" && action.Available {
			t.Error("kill action should not be available when no orphans")
		}
		if action.ID == "list" && !action.Available {
			t.Error("list action should always be available")
		}
	}

	// With orphans, kill should be available
	hasOrphansResult := &checks.Result{
		Details: map[string]interface{}{"orphanCount": 3},
	}
	actionsHasOrphans := check.RecoveryActions(hasOrphansResult)
	for _, action := range actionsHasOrphans {
		if action.ID == "kill" && !action.Available {
			t.Error("kill action should be available when orphans exist")
		}
	}
}

// TestOrphanCheckHealthMetrics tests that health metrics are properly set
// [REQ:ORPHAN-CHECK-001]
func TestOrphanCheckHealthMetrics(t *testing.T) {
	mockStateReader := &MockVrooliStateReader{
		TrackedProcesses: []checks.TrackedProcess{},
	}
	mockProcReader := &MockProcReader{
		Processes: []checks.ProcessInfo{
			{PID: 100, PPid: 1, Comm: "vrooli-orphan1", State: "S"},
			{PID: 101, PPid: 1, Comm: "vrooli-orphan2", State: "S"},
		},
	}

	check := NewOrphanCheck(
		WithOrphanStateReader(mockStateReader),
		WithOrphanProcReader(mockProcReader),
	)
	result := check.Run(context.Background())

	if result.Metrics == nil {
		t.Fatal("Metrics should not be nil")
	}
	if result.Metrics.Score == nil {
		t.Fatal("Score should not be nil")
	}

	// With 2 orphans, score should be 100 - (2 * 10) = 80
	expectedScore := 80
	if *result.Metrics.Score != expectedScore {
		t.Errorf("Score = %d, want %d", *result.Metrics.Score, expectedScore)
	}

	if len(result.Metrics.SubChecks) == 0 {
		t.Error("SubChecks should not be empty")
	}
}

// TestOrphanCheckDetails tests that result details are properly populated
// [REQ:ORPHAN-CHECK-001]
func TestOrphanCheckDetails(t *testing.T) {
	mockStateReader := &MockVrooliStateReader{
		TrackedProcesses: []checks.TrackedProcess{
			{PID: 100, PGID: 100, Scenario: "app-monitor"},
		},
	}
	mockProcReader := &MockProcReader{
		Processes: []checks.ProcessInfo{
			{PID: 100, PPid: 1, Comm: "vrooli-tracked", State: "S"},
			{PID: 200, PPid: 1, Comm: "vrooli-orphan1", State: "S"},
			{PID: 201, PPid: 1, Comm: "vrooli-orphan2", State: "S"},
		},
	}

	check := NewOrphanCheck(
		WithOrphanStateReader(mockStateReader),
		WithOrphanProcReader(mockProcReader),
	)
	result := check.Run(context.Background())

	// Verify details are populated
	if orphanCount, ok := result.Details["orphanCount"].(int); !ok || orphanCount != 2 {
		t.Errorf("orphanCount = %v, want 2", result.Details["orphanCount"])
	}
	if trackedCount, ok := result.Details["trackedCount"].(int); !ok || trackedCount != 1 {
		t.Errorf("trackedCount = %v, want 1", result.Details["trackedCount"])
	}
	if _, ok := result.Details["orphans"]; !ok {
		t.Error("orphans should be in details")
	}
}

// TestOrphanCheckAncestryTracking tests that children of tracked processes are not marked as orphans
// [REQ:ORPHAN-CHECK-001]
func TestOrphanCheckAncestryTracking(t *testing.T) {
	mockStateReader := &MockVrooliStateReader{
		TrackedProcesses: []checks.TrackedProcess{
			{PID: 100, PGID: 100, Scenario: "app-monitor"},
		},
	}
	mockProcReader := &MockProcReader{
		Processes: []checks.ProcessInfo{
			{PID: 100, PPid: 1, Comm: "vrooli-parent", State: "S"},
			{PID: 200, PPid: 100, Comm: "vrooli-child", State: "S"},   // Child of 100
			{PID: 300, PPid: 200, Comm: "vrooli-grandchild", State: "S"}, // Grandchild of 100
		},
	}

	check := NewOrphanCheck(
		WithOrphanStateReader(mockStateReader),
		WithOrphanProcReader(mockProcReader),
	)
	result := check.Run(context.Background())

	// All processes should be tracked (parent or descendants of parent)
	if orphanCount, ok := result.Details["orphanCount"].(int); !ok || orphanCount != 0 {
		t.Errorf("orphanCount = %v, want 0 (all are tracked or descendants)", result.Details["orphanCount"])
	}
}

// TestVrooliProcessPatterns tests the process pattern matching
// [REQ:ORPHAN-CHECK-001]
func TestVrooliProcessPatterns(t *testing.T) {
	tests := []struct {
		command  string
		expected bool
	}{
		{"vrooli", true},
		{"vrooli-api", true},
		{"/scenarios/app-monitor/api", true},
		{"/scenarios/test-app/ui", true},
		{"node_modules/.bin/vite", true},
		{"ecosystem-manager", true},
		{"picker-wheel", true},
		{"nginx", false},
		{"postgresql", false},
		{"bash", false},
		{"/usr/bin/node", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := VrooliProcessPatterns.MatchString(tt.command)
			if result != tt.expected {
				t.Errorf("VrooliProcessPatterns.MatchString(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}

// TestGetOrphanPIDs tests the GetOrphanPIDs helper function
// [REQ:ORPHAN-CHECK-001]
func TestGetOrphanPIDs(t *testing.T) {
	mockStateReader := &MockVrooliStateReader{
		TrackedProcesses: []checks.TrackedProcess{
			{PID: 100, PGID: 100, Scenario: "app-monitor"},
		},
	}
	mockProcReader := &MockProcReader{
		Processes: []checks.ProcessInfo{
			{PID: 100, PPid: 1, Comm: "vrooli-tracked", State: "S"},
			{PID: 200, PPid: 1, Comm: "vrooli-orphan1", State: "S"},
			{PID: 201, PPid: 1, Comm: "vrooli-orphan2", State: "S"},
		},
	}

	check := NewOrphanCheck(
		WithOrphanStateReader(mockStateReader),
		WithOrphanProcReader(mockProcReader),
	)

	pids, err := check.GetOrphanPIDs()
	if err != nil {
		t.Fatalf("GetOrphanPIDs() error = %v", err)
	}

	if len(pids) != 2 {
		t.Errorf("GetOrphanPIDs() returned %d PIDs, want 2", len(pids))
	}

	// Verify the correct PIDs are returned
	pidSet := make(map[int]bool)
	for _, pid := range pids {
		pidSet[pid] = true
	}
	if !pidSet[200] || !pidSet[201] {
		t.Errorf("GetOrphanPIDs() = %v, want [200, 201]", pids)
	}
}

// TestOrphanCheckGracePeriod tests that young processes are skipped
// [REQ:ORPHAN-CHECK-001]
func TestOrphanCheckGracePeriod(t *testing.T) {
	// Create a mock state reader with no tracked processes
	mockStateReader := &MockVrooliStateReader{
		TrackedProcesses: []checks.TrackedProcess{},
	}

	// Create processes with different ages using StartTime
	// StartTime is in clock ticks since boot - we simulate this by using
	// different values. The actual age calculation depends on boot time,
	// but for testing we can verify the logic is applied correctly.
	mockProcReader := &MockProcReader{
		Processes: []checks.ProcessInfo{
			// Old process (StartTime = 0 means age calculation returns 0, treated as "unknown age" -> not skipped)
			{PID: 100, PPid: 1, Comm: "vrooli-old", State: "S", StartTime: 0},
			// We can't easily mock the boot time, so we test the configuration instead
		},
	}

	// Test that grace period can be configured
	check := NewOrphanCheck(
		WithOrphanStateReader(mockStateReader),
		WithOrphanProcReader(mockProcReader),
		WithGracePeriod(60), // 60 second grace period
	)

	// Verify grace period is set
	if check.gracePeriodSeconds != 60 {
		t.Errorf("gracePeriodSeconds = %v, want 60", check.gracePeriodSeconds)
	}

	// Test default grace period
	defaultCheck := NewOrphanCheck()
	if defaultCheck.gracePeriodSeconds != DefaultGracePeriodSeconds {
		t.Errorf("default gracePeriodSeconds = %v, want %v", defaultCheck.gracePeriodSeconds, DefaultGracePeriodSeconds)
	}
}

// TestOrphanCheckGracePeriodInDetails tests that grace period info appears in result details
func TestOrphanCheckGracePeriodInDetails(t *testing.T) {
	mockStateReader := &MockVrooliStateReader{
		TrackedProcesses: []checks.TrackedProcess{},
	}
	mockProcReader := &MockProcReader{
		Processes: []checks.ProcessInfo{},
	}

	check := NewOrphanCheck(
		WithOrphanStateReader(mockStateReader),
		WithOrphanProcReader(mockProcReader),
		WithGracePeriod(45),
	)

	result := check.Run(context.Background())

	// Verify grace period is in details
	gracePeriod, ok := result.Details["gracePeriodSeconds"]
	if !ok {
		t.Error("gracePeriodSeconds not found in result details")
	}
	if gracePeriod != float64(45) {
		t.Errorf("gracePeriodSeconds = %v, want 45", gracePeriod)
	}

	// Verify skippedYoung is in details
	skippedYoung, ok := result.Details["skippedYoung"]
	if !ok {
		t.Error("skippedYoung not found in result details")
	}
	if skippedYoung != 0 {
		t.Errorf("skippedYoung = %v, want 0 (no processes)", skippedYoung)
	}
}
