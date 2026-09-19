// Package infra tests for systemd-resolved health check
// [REQ:INFRA-RESOLVED-001] [REQ:TEST-SEAM-001]
package infra

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// =============================================================================
// ResolvedCheck Unit Tests with Mock Interfaces
// =============================================================================

func TestResolvedCheckNonLinuxIsNotApplicable(t *testing.T) {
	originalOS := resolvedOS
	resolvedOS = "darwin"
	t.Cleanup(func() { resolvedOS = originalOS })

	result := NewResolvedCheck(&platform.Capabilities{}).Run(context.Background())
	if result.Status != checks.StatusNotApplicable {
		t.Fatalf("status = %v, want %v", result.Status, checks.StatusNotApplicable)
	}
}

// TestResolvedCheckRunWithMock_Active tests when systemd-resolved is active
// [REQ:INFRA-RESOLVED-001] [REQ:TEST-SEAM-001]
func TestResolvedCheckRunWithMock_Active(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := testutil.NewMockExecutor()
	// Service exists
	mockExec.Responses["systemctl list-unit-files systemd-resolved.service"] = testutil.MockResponse{
		Output: []byte("UNIT FILE                 STATE\nsystemd-resolved.service  enabled\n\n1 unit files listed."),
		Error:  nil,
	}
	// Service is active
	mockExec.Responses["systemctl is-active systemd-resolved"] = testutil.MockResponse{
		Output: []byte("active"),
		Error:  nil,
	}
	// Stats available
	mockExec.Responses["resolvectl statistics"] = testutil.MockResponse{
		Output: []byte("Current Transactions: 0\n  Cache size: 1234"),
		Error:  nil,
	}

	check := NewResolvedCheck(caps, WithResolvedExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v (Message: %s)", result.Status, checks.StatusOK, result.Message)
	}
	if result.Message != "systemd-resolved is running" {
		t.Errorf("Message = %q, want %q", result.Message, "systemd-resolved is running")
	}
	if result.Details["installed"] != true {
		t.Errorf("Details[installed] = %v, want true", result.Details["installed"])
	}
}

// TestResolvedCheckRunWithMock_Inactive tests when systemd-resolved is inactive
func TestResolvedCheckRunWithMock_Inactive(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := testutil.NewMockExecutor()
	mockExec.Responses["systemctl list-unit-files systemd-resolved.service"] = testutil.MockResponse{
		Output: []byte("UNIT FILE                 STATE\nsystemd-resolved.service  enabled\n"),
		Error:  nil,
	}
	mockExec.Responses["systemctl is-active systemd-resolved"] = testutil.MockResponse{
		Output: []byte("inactive"),
		Error:  nil,
	}

	check := NewResolvedCheck(caps, WithResolvedExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusCritical {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusCritical)
	}
	if result.Message != "systemd-resolved is not running" {
		t.Errorf("Message = %q, want %q", result.Message, "systemd-resolved is not running")
	}
}

// TestResolvedCheckRunWithMock_Failed tests when systemd-resolved has failed
func TestResolvedCheckRunWithMock_Failed(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := testutil.NewMockExecutor()
	mockExec.Responses["systemctl list-unit-files systemd-resolved.service"] = testutil.MockResponse{
		Output: []byte("UNIT FILE                 STATE\nsystemd-resolved.service  enabled\n"),
		Error:  nil,
	}
	mockExec.Responses["systemctl is-active systemd-resolved"] = testutil.MockResponse{
		Output: []byte("failed"),
		Error:  nil,
	}

	check := NewResolvedCheck(caps, WithResolvedExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusCritical {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusCritical)
	}
	if result.Message != "systemd-resolved has failed" {
		t.Errorf("Message = %q, want %q", result.Message, "systemd-resolved has failed")
	}
}

// TestResolvedCheckRunWithMock_Activating tests when systemd-resolved is starting
func TestResolvedCheckRunWithMock_Activating(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := testutil.NewMockExecutor()
	mockExec.Responses["systemctl list-unit-files systemd-resolved.service"] = testutil.MockResponse{
		Output: []byte("UNIT FILE                 STATE\nsystemd-resolved.service  enabled\n"),
		Error:  nil,
	}
	mockExec.Responses["systemctl is-active systemd-resolved"] = testutil.MockResponse{
		Output: []byte("activating"),
		Error:  nil,
	}

	check := NewResolvedCheck(caps, WithResolvedExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusWarning)
	}
	if result.Message != "systemd-resolved is starting" {
		t.Errorf("Message = %q, want %q", result.Message, "systemd-resolved is starting")
	}
}

// TestResolvedCheckRunWithMock_NotInstalled tests when systemd-resolved is not installed
func TestResolvedCheckRunWithMock_NotInstalled(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := testutil.NewMockExecutor()
	// Service does not exist
	mockExec.Responses["systemctl list-unit-files systemd-resolved.service"] = testutil.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}

	check := NewResolvedCheck(caps, WithResolvedExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v (alternative DNS in use)", result.Status, checks.StatusOK)
	}
	if result.Details["installed"] != false {
		t.Errorf("Details[installed] = %v, want false", result.Details["installed"])
	}
}

// TestResolvedCheckRunWithMock_NoSystemd tests when systemd is not available
func TestResolvedCheckRunWithMock_NoSystemd(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: false,
	}

	mockExec := testutil.NewMockExecutor()
	check := NewResolvedCheck(caps, WithResolvedExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v (no systemd)", result.Status, checks.StatusOK)
	}
	if result.Details["reason"] != "no systemd" {
		t.Errorf("Details[reason] = %v, want 'no systemd'", result.Details["reason"])
	}
}

// TestResolvedCheckRecoveryActions tests recovery action availability
func TestResolvedCheckRecoveryActions(t *testing.T) {
	tests := []struct {
		name            string
		lastResult      *checks.Result
		expectAvailable map[string]bool
	}{
		{
			name:       "nil result",
			lastResult: nil,
			expectAvailable: map[string]bool{
				"start":   true,
				"restart": true,
				"logs":    true,
			},
		},
		{
			name: "service running",
			lastResult: &checks.Result{
				Details: map[string]interface{}{
					"serviceStatus": "active",
				},
			},
			expectAvailable: map[string]bool{
				"start":   false, // Already running
				"restart": true,
				"logs":    true,
			},
		},
		{
			name: "service not running",
			lastResult: &checks.Result{
				Details: map[string]interface{}{
					"serviceStatus": "inactive",
				},
			},
			expectAvailable: map[string]bool{
				"start":   true,
				"restart": true,
				"logs":    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := NewResolvedCheck(testCaps())
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

// TestResolvedCheckRecoveryActionsAllSafe verifies no dangerous actions
func TestResolvedCheckRecoveryActionsAllSafe(t *testing.T) {
	check := NewResolvedCheck(testCaps())
	actions := check.RecoveryActions(nil)

	for _, a := range actions {
		if a.Dangerous {
			t.Errorf("Action %q should not be dangerous", a.ID)
		}
	}
}

// TestResolvedCheckExecuteAction_Start tests start action
func TestResolvedCheckExecuteAction_Start(t *testing.T) {
	allowRecoveryGrant(t)
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := testutil.NewMockExecutor()
	mockExec.Responses["sudo -n /usr/bin/systemctl start systemd-resolved"] = testutil.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}
	// Verification
	mockExec.Responses["systemctl list-unit-files systemd-resolved.service"] = testutil.MockResponse{
		Output: []byte("systemd-resolved.service  enabled\n"),
		Error:  nil,
	}
	mockExec.Responses["systemctl is-active systemd-resolved"] = testutil.MockResponse{
		Output: []byte("active"),
		Error:  nil,
	}
	mockExec.DefaultResponse = testutil.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}

	check := NewResolvedCheck(caps, WithResolvedExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "start")

	if result.ActionID != "start" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "start")
	}
	if result.CheckID != check.ID() {
		t.Errorf("CheckID = %q, want %q", result.CheckID, check.ID())
	}
}

// TestResolvedCheckExecuteAction_Restart tests restart action
func TestResolvedCheckExecuteAction_Restart(t *testing.T) {
	allowRecoveryGrant(t)
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := testutil.NewMockExecutor()
	mockExec.Responses["sudo -n /usr/bin/systemctl restart systemd-resolved"] = testutil.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}
	mockExec.Responses["systemctl list-unit-files systemd-resolved.service"] = testutil.MockResponse{
		Output: []byte("systemd-resolved.service  enabled\n"),
		Error:  nil,
	}
	mockExec.Responses["systemctl is-active systemd-resolved"] = testutil.MockResponse{
		Output: []byte("active"),
		Error:  nil,
	}
	mockExec.DefaultResponse = testutil.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}

	check := NewResolvedCheck(caps, WithResolvedExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "restart")

	if result.ActionID != "restart" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "restart")
	}
}

// TestResolvedCheckExecuteAction_Logs tests logs action
func TestResolvedCheckExecuteAction_Logs(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := testutil.NewMockExecutor()
	mockExec.Responses["journalctl --no-pager -o short-iso -u systemd-resolved -n 50"] = testutil.MockResponse{
		Output: []byte("systemd-resolved[123]: Listening on 127.0.0.53"),
		Error:  nil,
	}

	check := NewResolvedCheck(caps, WithResolvedExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "logs")

	if !result.Success {
		t.Errorf("Success = %v, want true", result.Success)
	}
	if result.ActionID != "logs" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "logs")
	}
}

// TestResolvedCheckExecuteAction_UnknownAction tests unknown action handling
func TestResolvedCheckExecuteAction_UnknownAction(t *testing.T) {
	check := NewResolvedCheck(testCaps())
	result := check.ExecuteAction(context.Background(), "unknown-action")

	if result.Success {
		t.Error("Success should be false for unknown action")
	}
	if result.Error != "unknown action: unknown-action" {
		t.Errorf("Error = %q, want %q", result.Error, "unknown action: unknown-action")
	}
}

// TestResolvedCheckExecuteAction_StartFailure tests start action failure
func TestResolvedCheckExecuteAction_StartFailure(t *testing.T) {
	allowRecoveryGrant(t)
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := testutil.NewMockExecutor()
	mockExec.Responses["sudo -n /usr/bin/systemctl start systemd-resolved"] = testutil.MockResponse{
		Output: []byte("Failed to start systemd-resolved.service"),
		Error:  testutil.ErrPermissionDenied,
	}

	check := NewResolvedCheck(caps, WithResolvedExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "start")

	if result.Success {
		t.Error("Success should be false when start fails")
	}
}

// TestResolvedCheckExecutorInjection verifies executor is properly injected
func TestResolvedCheckExecutorInjection(t *testing.T) {
	mockExec := testutil.NewMockExecutor()
	check := NewResolvedCheck(testCaps(), WithResolvedExecutor(mockExec))

	if check.executor != mockExec {
		t.Error("Executor was not properly injected")
	}
}

// TestResolvedCheckDefaultExecutor verifies default executor is used
func TestResolvedCheckDefaultExecutor(t *testing.T) {
	check := NewResolvedCheck(testCaps())

	if check.executor != checks.DefaultExecutor {
		t.Error("Default executor should be used when not injected")
	}
}
