// Package infra tests for NTP health check
// [REQ:INFRA-NTP-001] [REQ:TEST-SEAM-001]
package infra

import (
	"context"
	"testing"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// =============================================================================
// NTPCheck Unit Tests with Mock Interfaces
// =============================================================================

// TestNTPCheckRunWithMock_Synchronized tests when NTP is synchronized
// [REQ:INFRA-NTP-001] [REQ:TEST-SEAM-001]
func TestNTPCheckRunWithMock_Synchronized(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	// timedatectl is available
	mockExec.Responses["which timedatectl"] = checks.MockResponse{
		Output: []byte("/usr/bin/timedatectl"),
		Error:  nil,
	}
	// NTP is synchronized
	mockExec.Responses["timedatectl show -p NTPSynchronized --value"] = checks.MockResponse{
		Output: []byte("yes"),
		Error:  nil,
	}
	// NTP is enabled
	mockExec.Responses["timedatectl show -p NTP --value"] = checks.MockResponse{
		Output: []byte("yes"),
		Error:  nil,
	}
	// Timezone info
	mockExec.Responses["timedatectl show -p Timezone --value"] = checks.MockResponse{
		Output: []byte("America/New_York"),
		Error:  nil,
	}
	mockExec.DefaultResponse = checks.MockResponse{
		Output: []byte("2024-01-15T12:00:00-0500"),
		Error:  nil,
	}

	check := NewNTPCheck(caps, WithNTPExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v (Message: %s)", result.Status, checks.StatusOK, result.Message)
	}
	if result.Message != "System clock is synchronized via NTP" {
		t.Errorf("Message = %q, want %q", result.Message, "System clock is synchronized via NTP")
	}
}

// TestNTPCheckRunWithMock_NotSynchronized tests when NTP is not synchronized
func TestNTPCheckRunWithMock_NotSynchronized(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["which timedatectl"] = checks.MockResponse{
		Output: []byte("/usr/bin/timedatectl"),
		Error:  nil,
	}
	mockExec.Responses["timedatectl show -p NTPSynchronized --value"] = checks.MockResponse{
		Output: []byte("no"),
		Error:  nil,
	}
	mockExec.Responses["timedatectl show -p NTP --value"] = checks.MockResponse{
		Output: []byte("yes"),
		Error:  nil,
	}
	mockExec.DefaultResponse = checks.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}

	check := NewNTPCheck(caps, WithNTPExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusWarning)
	}
	if result.Message != "NTP enabled but not yet synchronized" {
		t.Errorf("Message = %q, want %q", result.Message, "NTP enabled but not yet synchronized")
	}
}

// TestNTPCheckRunWithMock_NTPDisabled tests when NTP is disabled
func TestNTPCheckRunWithMock_NTPDisabled(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["which timedatectl"] = checks.MockResponse{
		Output: []byte("/usr/bin/timedatectl"),
		Error:  nil,
	}
	mockExec.Responses["timedatectl show -p NTPSynchronized --value"] = checks.MockResponse{
		Output: []byte("no"),
		Error:  nil,
	}
	mockExec.Responses["timedatectl show -p NTP --value"] = checks.MockResponse{
		Output: []byte("no"),
		Error:  nil,
	}
	mockExec.DefaultResponse = checks.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}

	check := NewNTPCheck(caps, WithNTPExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusWarning)
	}
	if result.Message != "NTP is disabled - time may drift" {
		t.Errorf("Message = %q, want %q", result.Message, "NTP is disabled - time may drift")
	}
	if result.Details["recommendation"] != "Run: sudo timedatectl set-ntp true" {
		t.Errorf("Missing or incorrect recommendation")
	}
}

// TestNTPCheckRunWithMock_TimedatectlNotAvailable tests when timedatectl is not available
func TestNTPCheckRunWithMock_TimedatectlNotAvailable(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["which timedatectl"] = checks.MockResponse{
		Output: []byte(""),
		Error:  checks.ErrCommandNotFound,
	}

	check := NewNTPCheck(caps, WithNTPExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusWarning)
	}
	if result.Message != "timedatectl not available - cannot verify time sync" {
		t.Errorf("Message = %q, want %q", result.Message, "timedatectl not available - cannot verify time sync")
	}
}

// TestNTPCheckRecoveryActions tests recovery action availability
func TestNTPCheckRecoveryActions(t *testing.T) {
	tests := []struct {
		name            string
		lastResult      *checks.Result
		expectAvailable map[string]bool
	}{
		{
			name:       "nil result",
			lastResult: nil,
			expectAvailable: map[string]bool{
				"enable-ntp": false, // Default: NTP assumed enabled
				"force-sync": true,
			},
		},
		{
			name: "ntp disabled",
			lastResult: &checks.Result{
				Details: map[string]interface{}{
					"ntpEnabled": false,
				},
			},
			expectAvailable: map[string]bool{
				"enable-ntp": true,
				"force-sync": true,
			},
		},
		{
			name: "ntp enabled",
			lastResult: &checks.Result{
				Details: map[string]interface{}{
					"ntpEnabled": true,
				},
			},
			expectAvailable: map[string]bool{
				"enable-ntp": false,
				"force-sync": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := NewNTPCheck(testCaps())
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

// TestNTPCheckRecoveryActionsAllSafe verifies no dangerous actions
func TestNTPCheckRecoveryActionsAllSafe(t *testing.T) {
	check := NewNTPCheck(testCaps())
	actions := check.RecoveryActions(nil)

	for _, a := range actions {
		if a.Dangerous {
			t.Errorf("Action %q should not be dangerous", a.ID)
		}
	}
}

// TestNTPCheckExecuteAction_EnableNTP tests enable-ntp action
func TestNTPCheckExecuteAction_EnableNTP(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["sudo timedatectl set-ntp true"] = checks.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}
	// After enabling, verification shows NTP is now synced
	mockExec.Responses["which timedatectl"] = checks.MockResponse{
		Output: []byte("/usr/bin/timedatectl"),
		Error:  nil,
	}
	mockExec.Responses["timedatectl show -p NTPSynchronized --value"] = checks.MockResponse{
		Output: []byte("yes"),
		Error:  nil,
	}
	mockExec.DefaultResponse = checks.MockResponse{
		Output: []byte("yes"),
		Error:  nil,
	}

	check := NewNTPCheck(caps, WithNTPExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "enable-ntp")

	if result.ActionID != "enable-ntp" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "enable-ntp")
	}
	if result.CheckID != check.ID() {
		t.Errorf("CheckID = %q, want %q", result.CheckID, check.ID())
	}
}

// TestNTPCheckExecuteAction_ForceSync tests force-sync action
func TestNTPCheckExecuteAction_ForceSync(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["sudo systemctl restart systemd-timesyncd"] = checks.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}
	mockExec.Responses["which timedatectl"] = checks.MockResponse{
		Output: []byte("/usr/bin/timedatectl"),
		Error:  nil,
	}
	mockExec.Responses["timedatectl show -p NTPSynchronized --value"] = checks.MockResponse{
		Output: []byte("yes"),
		Error:  nil,
	}
	mockExec.DefaultResponse = checks.MockResponse{
		Output: []byte("yes"),
		Error:  nil,
	}

	check := NewNTPCheck(caps, WithNTPExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "force-sync")

	if result.ActionID != "force-sync" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "force-sync")
	}
}

// TestNTPCheckExecuteAction_UnknownAction tests unknown action handling
func TestNTPCheckExecuteAction_UnknownAction(t *testing.T) {
	check := NewNTPCheck(testCaps())
	result := check.ExecuteAction(context.Background(), "unknown-action")

	if result.Success {
		t.Error("Success should be false for unknown action")
	}
	if result.Error != "unknown action: unknown-action" {
		t.Errorf("Error = %q, want %q", result.Error, "unknown action: unknown-action")
	}
}

// TestNTPCheckExecuteAction_EnableNTPFailure tests enable-ntp action failure
func TestNTPCheckExecuteAction_EnableNTPFailure(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["sudo timedatectl set-ntp true"] = checks.MockResponse{
		Output: []byte("Failed to set NTP"),
		Error:  checks.ErrPermissionDenied,
	}

	check := NewNTPCheck(caps, WithNTPExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "enable-ntp")

	if result.Success {
		t.Error("Success should be false when enable-ntp fails")
	}
}

// TestNTPCheckExecutorInjection verifies executor is properly injected
func TestNTPCheckExecutorInjection(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	check := NewNTPCheck(testCaps(), WithNTPExecutor(mockExec))

	if check.executor != mockExec {
		t.Error("Executor was not properly injected")
	}
}

// TestNTPCheckDefaultExecutor verifies default executor is used
func TestNTPCheckDefaultExecutor(t *testing.T) {
	check := NewNTPCheck(testCaps())

	if check.executor != checks.DefaultExecutor {
		t.Error("Default executor should be used when not injected")
	}
}
