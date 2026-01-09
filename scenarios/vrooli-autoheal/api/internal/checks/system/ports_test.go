// Package system tests for port exhaustion health check
// [REQ:SYSTEM-PORTS-001] [REQ:TEST-SEAM-001]
package system

import (
	"context"
	"testing"

	"vrooli-autoheal/internal/checks"
)

// =============================================================================
// PortCheck Unit Tests with Mock Interfaces
// =============================================================================

// TestPortCheckRunWithMock_Healthy tests healthy port usage
// [REQ:SYSTEM-PORTS-001] [REQ:TEST-SEAM-001]
func TestPortCheckRunWithMock_Healthy(t *testing.T) {
	mockReader := checks.NewMockPortReader()
	mockReader.PortInfo = &checks.PortInfo{
		UsedPorts:   1000,
		TotalPorts:  28232,
		UsedPercent: 3,
		TimeWait:    50,
	}

	check := NewPortCheck(WithPortReader(mockReader))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v (Message: %s)", result.Status, checks.StatusOK, result.Message)
	}
	if result.CheckID != check.ID() {
		t.Errorf("CheckID = %q, want %q", result.CheckID, check.ID())
	}
	// Verify details are populated
	if result.Details["usedPorts"] != 1000 {
		t.Errorf("Details[usedPorts] = %v, want 1000", result.Details["usedPorts"])
	}
	if result.Details["totalPorts"] != 28232 {
		t.Errorf("Details[totalPorts] = %v, want 28232", result.Details["totalPorts"])
	}
}

// TestPortCheckRunWithMock_Warning tests warning threshold
func TestPortCheckRunWithMock_Warning(t *testing.T) {
	mockReader := checks.NewMockPortReader()
	mockReader.PortInfo = &checks.PortInfo{
		UsedPorts:   20000,
		TotalPorts:  28232,
		UsedPercent: 71, // Above 70% warning threshold
		TimeWait:    5000,
	}

	check := NewPortCheck(WithPortReader(mockReader))
	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusWarning)
	}
}

// TestPortCheckRunWithMock_Critical tests critical threshold
func TestPortCheckRunWithMock_Critical(t *testing.T) {
	mockReader := checks.NewMockPortReader()
	mockReader.PortInfo = &checks.PortInfo{
		UsedPorts:   24000,
		TotalPorts:  28232,
		UsedPercent: 85, // At 85% critical threshold
		TimeWait:    10000,
	}

	check := NewPortCheck(WithPortReader(mockReader))
	result := check.Run(context.Background())

	if result.Status != checks.StatusCritical {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusCritical)
	}
}

// TestPortCheckRunWithMock_CustomThresholds tests custom thresholds
func TestPortCheckRunWithMock_CustomThresholds(t *testing.T) {
	mockReader := checks.NewMockPortReader()
	mockReader.PortInfo = &checks.PortInfo{
		UsedPorts:   15000,
		TotalPorts:  28232,
		UsedPercent: 53, // Above custom 50% warning
		TimeWait:    3000,
	}

	check := NewPortCheck(
		WithPortReader(mockReader),
		WithPortThresholds(50, 75), // Custom: warn at 50%, critical at 75%
	)
	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want %v (custom threshold)", result.Status, checks.StatusWarning)
	}
}

// TestPortCheckRunWithMock_ReadError tests error handling
func TestPortCheckRunWithMock_ReadError(t *testing.T) {
	mockReader := checks.NewMockPortReader()
	mockReader.Error = checks.ErrFileNotFound

	check := NewPortCheck(WithPortReader(mockReader))
	result := check.Run(context.Background())

	if result.Status != checks.StatusCritical {
		t.Errorf("Status = %v, want %v (read error)", result.Status, checks.StatusCritical)
	}
	if result.Message != "Failed to read port statistics" {
		t.Errorf("Message = %q, want %q", result.Message, "Failed to read port statistics")
	}
}

// TestPortCheckMetrics tests health metrics calculation
func TestPortCheckMetrics(t *testing.T) {
	mockReader := checks.NewMockPortReader()
	mockReader.PortInfo = &checks.PortInfo{
		UsedPorts:   5000,
		TotalPorts:  28232,
		UsedPercent: 18,
		TimeWait:    500,
	}

	check := NewPortCheck(WithPortReader(mockReader))
	result := check.Run(context.Background())

	if result.Metrics == nil {
		t.Fatal("Metrics should not be nil")
	}
	if result.Metrics.Score == nil {
		t.Fatal("Score should not be nil")
	}
	// Score should be 100 - usedPercent = 82
	expectedScore := 82
	if *result.Metrics.Score != expectedScore {
		t.Errorf("Score = %d, want %d", *result.Metrics.Score, expectedScore)
	}
	if len(result.Metrics.SubChecks) != 1 {
		t.Errorf("SubChecks count = %d, want 1", len(result.Metrics.SubChecks))
	}
}

// TestPortCheckRecoveryActions tests recovery action availability
func TestPortCheckRecoveryActions(t *testing.T) {
	tests := []struct {
		name            string
		lastResult      *checks.Result
		expectAvailable map[string]bool
	}{
		{
			name:       "nil result",
			lastResult: nil,
			expectAvailable: map[string]bool{
				"analyze":   true,
				"time-wait": false, // Not high usage
				"kill-port": true,
			},
		},
		{
			name: "high usage",
			lastResult: &checks.Result{
				Details: map[string]interface{}{
					"usedPercent": 75,
				},
			},
			expectAvailable: map[string]bool{
				"analyze":   true,
				"time-wait": true, // Available with high usage
				"kill-port": true,
			},
		},
		{
			name: "low usage",
			lastResult: &checks.Result{
				Details: map[string]interface{}{
					"usedPercent": 30,
				},
			},
			expectAvailable: map[string]bool{
				"analyze":   true,
				"time-wait": false,
				"kill-port": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := NewPortCheck()
			actions := check.RecoveryActions(tt.lastResult)

			actionMap := make(map[string]checks.RecoveryAction)
			for _, a := range actions {
				actionMap[a.ID] = a
			}

			for id, expectAvail := range tt.expectAvailable {
				action, exists := actionMap[id]
				if !exists {
					t.Errorf("Action %q not found", id)
					continue
				}
				if action.Available != expectAvail {
					t.Errorf("Action %q.Available = %v, want %v", id, action.Available, expectAvail)
				}
			}
		})
	}
}

// TestPortCheckRecoveryActionsDangerous tests dangerous action marking
func TestPortCheckRecoveryActionsDangerous(t *testing.T) {
	check := NewPortCheck()
	actions := check.RecoveryActions(nil)

	actionMap := make(map[string]checks.RecoveryAction)
	for _, a := range actions {
		actionMap[a.ID] = a
	}

	// kill-port should be dangerous
	if action, ok := actionMap["kill-port"]; ok {
		if !action.Dangerous {
			t.Error("kill-port action should be dangerous")
		}
	} else {
		t.Error("kill-port action not found")
	}

	// analyze should be safe
	if action, ok := actionMap["analyze"]; ok {
		if action.Dangerous {
			t.Error("analyze action should not be dangerous")
		}
	}
}

// TestPortCheckExecuteAction_Analyze tests analyze action
func TestPortCheckExecuteAction_Analyze(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	mockExec.Responses["ss -tunap"] = checks.MockResponse{
		Output: []byte("Netid  State  Recv-Q  Send-Q  Local Address:Port  Peer Address:Port\ntcp    ESTAB  0       0       127.0.0.1:8080      127.0.0.1:54321"),
		Error:  nil,
	}

	check := NewPortCheck(WithPortExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "analyze")

	if !result.Success {
		t.Errorf("Success = %v, want true", result.Success)
	}
	if result.ActionID != "analyze" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "analyze")
	}
	if result.Message != "Connection analysis complete" {
		t.Errorf("Message = %q, want %q", result.Message, "Connection analysis complete")
	}
}

// TestPortCheckExecuteAction_AnalyzeFallback tests analyze fallback to netstat
func TestPortCheckExecuteAction_AnalyzeFallback(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	// ss fails
	mockExec.Responses["ss -tunap"] = checks.MockResponse{
		Output: []byte(""),
		Error:  checks.ErrCommandNotFound,
	}
	// netstat succeeds
	mockExec.Responses["netstat -tunp"] = checks.MockResponse{
		Output: []byte("Active Internet connections\ntcp  0  0 127.0.0.1:8080  127.0.0.1:54321  ESTABLISHED"),
		Error:  nil,
	}

	check := NewPortCheck(WithPortExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "analyze")

	if !result.Success {
		t.Errorf("Success = %v, want true (fallback to netstat)", result.Success)
	}
}

// TestPortCheckExecuteAction_TimeWait tests time-wait action
func TestPortCheckExecuteAction_TimeWait(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	mockExec.Responses["ss -tan state time-wait"] = checks.MockResponse{
		Output: []byte("State  Recv-Q  Send-Q  Local Address:Port  Peer Address:Port\nTIME-WAIT  0  0  127.0.0.1:8080  127.0.0.1:54321\nTIME-WAIT  0  0  127.0.0.1:8081  127.0.0.1:54322"),
		Error:  nil,
	}

	check := NewPortCheck(WithPortExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "time-wait")

	if !result.Success {
		t.Errorf("Success = %v, want true", result.Success)
	}
	if result.ActionID != "time-wait" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "time-wait")
	}
}

// TestPortCheckExecuteAction_KillPort tests kill-port action
func TestPortCheckExecuteAction_KillPort(t *testing.T) {
	check := NewPortCheck()
	result := check.ExecuteAction(context.Background(), "kill-port")

	// kill-port provides instructions rather than executing
	if !result.Success {
		t.Errorf("Success = %v, want true", result.Success)
	}
	if result.ActionID != "kill-port" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "kill-port")
	}
}

// TestPortCheckExecuteAction_UnknownAction tests unknown action handling
func TestPortCheckExecuteAction_UnknownAction(t *testing.T) {
	check := NewPortCheck()
	result := check.ExecuteAction(context.Background(), "unknown-action")

	if result.Success {
		t.Error("Success should be false for unknown action")
	}
	if result.Error != "unknown action: unknown-action" {
		t.Errorf("Error = %q, want %q", result.Error, "unknown action: unknown-action")
	}
}

// TestPortCheckInterface verifies Check interface implementation
func TestPortCheckInterface(t *testing.T) {
	var _ checks.Check = (*PortCheck)(nil)
	var _ checks.HealableCheck = (*PortCheck)(nil)

	check := NewPortCheck()
	if check.ID() != "system-ports" {
		t.Errorf("ID() = %q, want %q", check.ID(), "system-ports")
	}
	if check.Title() == "" {
		t.Error("Title() is empty")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	if check.Importance() == "" {
		t.Error("Importance() is empty")
	}
	if check.IntervalSeconds() <= 0 {
		t.Error("IntervalSeconds() should be positive")
	}
	if check.Category() != checks.CategorySystem {
		t.Errorf("Category() = %v, want %v", check.Category(), checks.CategorySystem)
	}
}

// TestPortCheckReaderInjection verifies port reader is properly injected
func TestPortCheckReaderInjection(t *testing.T) {
	mockReader := checks.NewMockPortReader()
	check := NewPortCheck(WithPortReader(mockReader))

	if check.portReader != mockReader {
		t.Error("Port reader was not properly injected")
	}
}

// TestPortCheckDefaultReader verifies default port reader is used
func TestPortCheckDefaultReader(t *testing.T) {
	check := NewPortCheck()

	if check.portReader != checks.DefaultPortReader {
		t.Error("Default port reader should be used when not injected")
	}
}

// TestPortCheckMockCallsVerified verifies mock reader was called
func TestPortCheckMockCallsVerified(t *testing.T) {
	mockReader := checks.NewMockPortReader()
	mockReader.PortInfo = &checks.PortInfo{
		UsedPorts:   1000,
		TotalPorts:  28232,
		UsedPercent: 3,
		TimeWait:    50,
	}

	check := NewPortCheck(WithPortReader(mockReader))
	check.Run(context.Background())

	if mockReader.Calls != 1 {
		t.Errorf("Expected 1 ReadPortStats call, got %d", mockReader.Calls)
	}
}
