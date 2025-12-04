// Package infra tests for cloudflared health check
// [REQ:INFRA-CLOUDFLARED-001] [REQ:TEST-SEAM-001]
package infra

import (
	"context"
	"testing"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// =============================================================================
// CloudflaredCheck Unit Tests with Mock Interfaces
// =============================================================================

// TestCloudflaredCheckRunWithMock_ServiceActive tests when cloudflared service is active
// [REQ:INFRA-CLOUDFLARED-001] [REQ:TEST-SEAM-001]
func TestCloudflaredCheckRunWithMock_ServiceActive(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	// Service is active
	mockExec.Responses["systemctl is-active cloudflared"] = checks.MockResponse{
		Output: []byte("active"),
		Error:  nil,
	}
	// No errors in logs (use default response)
	mockExec.DefaultResponse = checks.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}

	check := NewCloudflaredCheck(caps,
		WithCloudflaredExecutor(mockExec),
		WithLocalTestPort(0), // Disable local port test
	)

	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v (Message: %s)", result.Status, checks.StatusOK, result.Message)
	}
	if result.CheckID != check.ID() {
		t.Errorf("CheckID = %q, want %q", result.CheckID, check.ID())
	}
}

// TestCloudflaredCheckRunWithMock_ServiceInactive tests when cloudflared service is not active
func TestCloudflaredCheckRunWithMock_ServiceInactive(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["systemctl is-active cloudflared"] = checks.MockResponse{
		Output: []byte("inactive"),
		Error:  checks.ErrConnectionRefused,
	}

	check := NewCloudflaredCheck(caps, WithCloudflaredExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusCritical {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusCritical)
	}
	if result.Message != "Cloudflared service not active" {
		t.Errorf("Message = %q, want %q", result.Message, "Cloudflared service not active")
	}
}

// TestCloudflaredCheckRunWithMock_NoSystemd tests when systemd is not available
func TestCloudflaredCheckRunWithMock_NoSystemd(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: false,
	}

	mockExec := checks.NewMockExecutor()
	check := NewCloudflaredCheck(caps, WithCloudflaredExecutor(mockExec))
	result := check.Run(context.Background())

	// Without systemd, we can't verify running status (returns Warning)
	validStatuses := map[checks.Status]bool{
		checks.StatusWarning: true,
		checks.StatusOK:      true,
	}
	if !validStatuses[result.Status] {
		t.Errorf("Status = %v, want Warning or OK", result.Status)
	}
}

// TestCloudflaredCheckRunWithMock_HighErrorRate tests warning on high error rate
func TestCloudflaredCheckRunWithMock_HighErrorRate(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	// Service is active
	mockExec.Responses["systemctl is-active cloudflared"] = checks.MockResponse{
		Output: []byte("active"),
		Error:  nil,
	}
	// Return logs with many ERR entries (more than threshold of 10)
	errLogs := "ERR: error1\nERR: error2\nERR: error3\nERR: error4\nERR: error5\n" +
		"ERR: error6\nERR: error7\nERR: error8\nERR: error9\nERR: error10\nERR: error11\n"
	mockExec.DefaultResponse = checks.MockResponse{
		Output: []byte(errLogs),
		Error:  nil,
	}

	check := NewCloudflaredCheck(caps,
		WithCloudflaredExecutor(mockExec),
		WithLocalTestPort(0), // Disable local port test
	)

	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want %v (high error rate)", result.Status, checks.StatusWarning)
	}
}

// TestCloudflaredCheckRecoveryActions tests recovery action availability
func TestCloudflaredCheckRecoveryActions(t *testing.T) {
	tests := []struct {
		name            string
		lastResult      *checks.Result
		expectAvailable map[string]bool
	}{
		{
			name:       "nil result",
			lastResult: nil,
			expectAvailable: map[string]bool{
				"start":       true, // Default: isInstalled=true, isRunning=false, so start is available
				"restart":     true,
				"test-tunnel": false, // Not running
				"logs":        true,
				"diagnose":    true,
			},
		},
		{
			name: "service not running",
			lastResult: &checks.Result{
				Details: map[string]interface{}{
					"installed":     true,
					"serviceStatus": "inactive",
				},
			},
			expectAvailable: map[string]bool{
				"start":       true,
				"restart":     true,
				"test-tunnel": false,
				"logs":        true,
				"diagnose":    true,
			},
		},
		{
			name: "service running",
			lastResult: &checks.Result{
				Details: map[string]interface{}{
					"installed":     true,
					"serviceStatus": "active",
				},
			},
			expectAvailable: map[string]bool{
				"start":       false, // Already running
				"restart":     true,
				"test-tunnel": true, // Available when running
				"logs":        true,
				"diagnose":    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := NewCloudflaredCheck(testCaps())
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

// TestCloudflaredCheckRecoveryActionsAllSafe verifies no dangerous actions
func TestCloudflaredCheckRecoveryActionsAllSafe(t *testing.T) {
	check := NewCloudflaredCheck(testCaps())
	actions := check.RecoveryActions(nil)

	for _, a := range actions {
		if a.Dangerous {
			t.Errorf("Action %q should not be dangerous", a.ID)
		}
	}
}

// TestCloudflaredCheckExecuteAction_Start tests the start action
func TestCloudflaredCheckExecuteAction_Start(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	// Start command succeeds
	mockExec.Responses["sudo systemctl start cloudflared"] = checks.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}
	// Verification: service is now active
	mockExec.Responses["systemctl is-active cloudflared"] = checks.MockResponse{
		Output: []byte("active"),
		Error:  nil,
	}
	// No errors in logs
	mockExec.DefaultResponse = checks.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}

	check := NewCloudflaredCheck(caps,
		WithCloudflaredExecutor(mockExec),
		WithLocalTestPort(0),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := check.ExecuteAction(ctx, "start")

	if result.ActionID != "start" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "start")
	}
	if result.CheckID != check.ID() {
		t.Errorf("CheckID = %q, want %q", result.CheckID, check.ID())
	}
}

// TestCloudflaredCheckExecuteAction_Restart tests the restart action
func TestCloudflaredCheckExecuteAction_Restart(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["sudo systemctl restart cloudflared"] = checks.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}
	mockExec.Responses["systemctl is-active cloudflared"] = checks.MockResponse{
		Output: []byte("active"),
		Error:  nil,
	}
	mockExec.DefaultResponse = checks.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}

	check := NewCloudflaredCheck(caps,
		WithCloudflaredExecutor(mockExec),
		WithLocalTestPort(0),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := check.ExecuteAction(ctx, "restart")

	if result.ActionID != "restart" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "restart")
	}
}

// TestCloudflaredCheckExecuteAction_Logs tests the logs action
func TestCloudflaredCheckExecuteAction_Logs(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["journalctl -u cloudflared -n 100 --no-pager"] = checks.MockResponse{
		Output: []byte("Jun 01 12:00:00 server cloudflared[1234]: Tunnel established\nJun 01 12:01:00 server cloudflared[1234]: Connection healthy"),
		Error:  nil,
	}

	check := NewCloudflaredCheck(caps, WithCloudflaredExecutor(mockExec))

	result := check.ExecuteAction(context.Background(), "logs")

	if !result.Success {
		t.Errorf("Success = %v, want true", result.Success)
	}
	if result.ActionID != "logs" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "logs")
	}
	if result.Message != "Retrieved cloudflared logs" {
		t.Errorf("Message = %q, want %q", result.Message, "Retrieved cloudflared logs")
	}
}

// TestCloudflaredCheckExecuteAction_Diagnose tests the diagnose action
func TestCloudflaredCheckExecuteAction_Diagnose(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["systemctl status cloudflared"] = checks.MockResponse{
		Output: []byte("● cloudflared.service - cloudflared\n   Active: active (running)"),
		Error:  nil,
	}
	mockExec.Responses["cloudflared tunnel info"] = checks.MockResponse{
		Output: []byte("Tunnel ID: abc123\nConnections: 4"),
		Error:  nil,
	}
	mockExec.DefaultResponse = checks.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}

	check := NewCloudflaredCheck(caps,
		WithCloudflaredExecutor(mockExec),
		WithLocalTestPort(0),
	)

	result := check.ExecuteAction(context.Background(), "diagnose")

	if !result.Success {
		t.Errorf("Success = %v, want true", result.Success)
	}
	if result.ActionID != "diagnose" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "diagnose")
	}
	if result.Message != "Diagnostic information gathered" {
		t.Errorf("Message = %q, want %q", result.Message, "Diagnostic information gathered")
	}
}

// TestCloudflaredCheckExecuteAction_UnknownAction tests unknown action handling
func TestCloudflaredCheckExecuteAction_UnknownAction(t *testing.T) {
	check := NewCloudflaredCheck(testCaps())

	result := check.ExecuteAction(context.Background(), "unknown-action")

	if result.Success {
		t.Error("Success should be false for unknown action")
	}
	if result.Error != "unknown action: unknown-action" {
		t.Errorf("Error = %q, want %q", result.Error, "unknown action: unknown-action")
	}
}

// TestCloudflaredCheckExecuteAction_StartFailure tests start action failure
func TestCloudflaredCheckExecuteAction_StartFailure(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["sudo systemctl start cloudflared"] = checks.MockResponse{
		Output: []byte("Failed to start cloudflared.service: Unit cloudflared.service not found."),
		Error:  checks.ErrCommandNotFound,
	}

	check := NewCloudflaredCheck(caps, WithCloudflaredExecutor(mockExec))

	result := check.ExecuteAction(context.Background(), "start")

	if result.Success {
		t.Error("Success should be false when start fails")
	}
	if result.Message != "Failed to start cloudflared service" {
		t.Errorf("Message = %q, want %q", result.Message, "Failed to start cloudflared service")
	}
}

// TestCloudflaredSelectVerifyMethod tests verify method selection
func TestCloudflaredSelectVerifyMethod(t *testing.T) {
	tests := []struct {
		name     string
		caps     *platform.Capabilities
		expected CloudflaredVerifyCapability
	}{
		{
			name: "linux with systemd",
			caps: &platform.Capabilities{
				Platform:        platform.Linux,
				SupportsSystemd: true,
			},
			expected: CanVerifyViaSystemd,
		},
		{
			name: "linux without systemd",
			caps: &platform.Capabilities{
				Platform:        platform.Linux,
				SupportsSystemd: false,
			},
			expected: CannotVerifyRunning,
		},
		{
			name: "macos",
			caps: &platform.Capabilities{
				Platform:        platform.MacOS,
				SupportsSystemd: false,
			},
			expected: CannotVerifyRunning,
		},
		{
			name: "windows",
			caps: &platform.Capabilities{
				Platform:        platform.Windows,
				SupportsSystemd: false,
			},
			expected: CannotVerifyRunning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SelectCloudflaredVerifyMethod(tt.caps)
			if result != tt.expected {
				t.Errorf("SelectCloudflaredVerifyMethod() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestCloudflaredCheckCountRecentErrors tests error counting in logs
func TestCloudflaredCheckCountRecentErrors(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	tests := []struct {
		name          string
		logOutput     string
		expectedCount int
	}{
		{
			name:          "no errors",
			logOutput:     "INFO: Connection established\nINFO: Tunnel ready",
			expectedCount: 0,
		},
		{
			name:          "some errors",
			logOutput:     "ERR: Connection failed\nINFO: Retrying\nERR: Timeout\nERR: DNS lookup failed",
			expectedCount: 3,
		},
		{
			name:          "empty logs",
			logOutput:     "",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := checks.NewMockExecutor()
			mockExec.DefaultResponse = checks.MockResponse{
				Output: []byte(tt.logOutput),
				Error:  nil,
			}

			check := NewCloudflaredCheck(caps, WithCloudflaredExecutor(mockExec))
			count := check.countRecentErrors(context.Background())

			if count != tt.expectedCount {
				t.Errorf("countRecentErrors() = %d, want %d", count, tt.expectedCount)
			}
		})
	}
}

// TestCloudflaredCheckExecutorInjection verifies executor is properly injected
func TestCloudflaredCheckExecutorInjection(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	check := NewCloudflaredCheck(testCaps(), WithCloudflaredExecutor(mockExec))

	if check.executor != mockExec {
		t.Error("Executor was not properly injected")
	}
}

// TestCloudflaredCheckDefaultExecutor verifies default executor is used
func TestCloudflaredCheckDefaultExecutor(t *testing.T) {
	check := NewCloudflaredCheck(testCaps())

	if check.executor != checks.DefaultExecutor {
		t.Error("Default executor should be used when not injected")
	}
}
