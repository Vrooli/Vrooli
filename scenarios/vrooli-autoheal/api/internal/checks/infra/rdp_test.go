// Package infra tests for RDP health check
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
package infra

import (
	"context"
	"testing"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// =============================================================================
// RDPCheck Unit Tests with Mock Interfaces
// =============================================================================

// TestRDPCheckRunWithMock_GnomeRDPRunning tests GNOME Remote Desktop detection
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
func TestRDPCheckRunWithMock_GnomeRDPRunning(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	// GNOME Remote Desktop is detected first via pgrep
	mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = checks.MockResponse{
		Output: []byte("12345"),
		Error:  nil,
	}
	mockExec.Responses["pgrep -a -f gnome-remote-desktop-daemon"] = checks.MockResponse{
		Output: []byte("12345 /usr/libexec/gnome-remote-desktop-daemon"),
		Error:  nil,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusOK)
	}
	if result.Message != "GNOME Remote Desktop is running" {
		t.Errorf("Message = %q, want %q", result.Message, "GNOME Remote Desktop is running")
	}
	if result.Details["type"] != string(RDPTypeGnome) {
		t.Errorf("Details[type] = %v, want %q", result.Details["type"], RDPTypeGnome)
	}
}

// TestRDPCheckRunWithMock_LinuxActive tests xrdp service active on Linux (no GNOME RDP)
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
func TestRDPCheckRunWithMock_LinuxActive(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	// No GNOME Remote Desktop running
	mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = checks.MockResponse{
		Output: []byte(""),
		Error:  checks.ErrConnectionRefused,
	}
	// xrdp is installed
	mockExec.Responses["systemctl list-unit-files xrdp.service"] = checks.MockResponse{
		Output: []byte("xrdp.service enabled"),
		Error:  nil,
	}
	// xrdp is active
	mockExec.Responses["systemctl is-active xrdp"] = checks.MockResponse{
		Output: []byte("active"),
		Error:  nil,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusOK)
	}
	if result.Message != "xrdp is running" {
		t.Errorf("Message = %q, want %q", result.Message, "xrdp is running")
	}
	if result.Details["service"] != "xrdp" {
		t.Errorf("Details[service] = %v, want %q", result.Details["service"], "xrdp")
	}
}

// TestRDPCheckRunWithMock_LinuxInactive tests xrdp service inactive on Linux
func TestRDPCheckRunWithMock_LinuxInactive(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	// No GNOME Remote Desktop running
	mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = checks.MockResponse{
		Output: []byte(""),
		Error:  checks.ErrConnectionRefused,
	}
	// xrdp is installed
	mockExec.Responses["systemctl list-unit-files xrdp.service"] = checks.MockResponse{
		Output: []byte("xrdp.service disabled"),
		Error:  nil,
	}
	// xrdp is inactive
	mockExec.Responses["systemctl is-active xrdp"] = checks.MockResponse{
		Output: []byte("inactive"),
		Error:  checks.ErrConnectionRefused,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusWarning)
	}
	if result.Message != "xrdp service not active" {
		t.Errorf("Message = %q, want %q", result.Message, "xrdp service not active")
	}
}

// TestRDPCheckRunWithMock_LinuxNoRDP tests Linux with no RDP service installed
func TestRDPCheckRunWithMock_LinuxNoRDP(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	// No GNOME Remote Desktop running
	mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = checks.MockResponse{
		Output: []byte(""),
		Error:  checks.ErrConnectionRefused,
	}
	// xrdp is NOT installed
	mockExec.Responses["systemctl list-unit-files xrdp.service"] = checks.MockResponse{
		Output: []byte(""),
		Error:  checks.ErrConnectionRefused,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec))
	result := check.Run(context.Background())

	// No RDP installed is OK (not a warning) - RDP is optional
	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusOK)
	}
	if result.Message != "No RDP service installed (remote desktop not configured)" {
		t.Errorf("Message = %q, want %q", result.Message, "No RDP service installed (remote desktop not configured)")
	}
}

// TestRDPCheckRunWithMock_LinuxNoSystemd tests Linux without systemd
func TestRDPCheckRunWithMock_LinuxNoSystemd(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: false,
	}

	mockExec := checks.NewMockExecutor()
	// No GNOME Remote Desktop running
	mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = checks.MockResponse{
		Output: []byte(""),
		Error:  checks.ErrConnectionRefused,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec))
	result := check.Run(context.Background())

	// No RDP service is OK - RDP is optional
	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusOK)
	}
	if result.Message != "No RDP service installed (remote desktop not configured)" {
		t.Errorf("Message = %q, want %q", result.Message, "No RDP service installed (remote desktop not configured)")
	}
}

// TestRDPCheckRunWithMock_WindowsRunning tests TermService running on Windows
func TestRDPCheckRunWithMock_WindowsRunning(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Windows,
		SupportsSystemd: false,
	}

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["sc query TermService"] = checks.MockResponse{
		Output: []byte("SERVICE_NAME: TermService\n        TYPE               : 20  WIN32_SHARE_PROCESS\n        STATE              : 4  RUNNING"),
		Error:  nil,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusOK)
	}
	if result.Message != "RDP service is running" {
		t.Errorf("Message = %q, want %q", result.Message, "RDP service is running")
	}
	if result.Details["service"] != "TermService" {
		t.Errorf("Details[service] = %v, want %q", result.Details["service"], "TermService")
	}
}

// TestRDPCheckRunWithMock_WindowsStopped tests TermService stopped on Windows
func TestRDPCheckRunWithMock_WindowsStopped(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Windows,
		SupportsSystemd: false,
	}

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["sc query TermService"] = checks.MockResponse{
		Output: []byte("SERVICE_NAME: TermService\n        TYPE               : 20  WIN32_SHARE_PROCESS\n        STATE              : 1  STOPPED"),
		Error:  nil,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusWarning)
	}
	if result.Message != "RDP service not running" {
		t.Errorf("Message = %q, want %q", result.Message, "RDP service not running")
	}
}

// TestRDPCheckRunWithMock_WindowsQueryError tests Windows query error
func TestRDPCheckRunWithMock_WindowsQueryError(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Windows,
		SupportsSystemd: false,
	}

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["sc query TermService"] = checks.MockResponse{
		Output: []byte(""),
		Error:  checks.ErrCommandNotFound,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusWarning)
	}
	if result.Message != "Unable to check RDP service" {
		t.Errorf("Message = %q, want %q", result.Message, "Unable to check RDP service")
	}
}

// TestRDPCheckRunWithMock_MacOS tests macOS (no RDP available - OK)
func TestRDPCheckRunWithMock_MacOS(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.MacOS,
		SupportsSystemd: false,
	}

	mockExec := checks.NewMockExecutor()
	check := NewRDPCheck(caps, WithRDPExecutor(mockExec))
	result := check.Run(context.Background())

	// No RDP on macOS is OK - RDP is optional
	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusOK)
	}
}

// TestSelectRDPServiceWithMock tests RDP service selection logic with various platforms
func TestSelectRDPServiceWithMock(t *testing.T) {
	tests := []struct {
		name            string
		caps            *platform.Capabilities
		expectService   string
		expectCheckable bool
	}{
		{
			name: "linux with systemd",
			caps: &platform.Capabilities{
				Platform:        platform.Linux,
				SupportsSystemd: true,
			},
			expectService:   "xrdp",
			expectCheckable: true,
		},
		{
			name: "linux without systemd",
			caps: &platform.Capabilities{
				Platform:        platform.Linux,
				SupportsSystemd: false,
			},
			expectService:   "xrdp",
			expectCheckable: false,
		},
		{
			name: "windows",
			caps: &platform.Capabilities{
				Platform:        platform.Windows,
				SupportsSystemd: false,
			},
			expectService:   "TermService",
			expectCheckable: true,
		},
		{
			name: "macos",
			caps: &platform.Capabilities{
				Platform:        platform.MacOS,
				SupportsSystemd: false,
			},
			expectService:   "",
			expectCheckable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := SelectRDPService(tt.caps)
			if info.ServiceName != tt.expectService {
				t.Errorf("ServiceName = %q, want %q", info.ServiceName, tt.expectService)
			}
			if info.Checkable != tt.expectCheckable {
				t.Errorf("Checkable = %v, want %v", info.Checkable, tt.expectCheckable)
			}
		})
	}
}

// TestRDPCheckExecutorInjection verifies executor is properly injected
func TestRDPCheckExecutorInjection(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	check := NewRDPCheck(testCaps(), WithRDPExecutor(mockExec))

	if check.executor != mockExec {
		t.Error("Executor was not properly injected")
	}
}

// TestRDPCheckDefaultExecutor verifies default executor is used
func TestRDPCheckDefaultExecutor(t *testing.T) {
	check := NewRDPCheck(testCaps())

	if check.executor != checks.DefaultExecutor {
		t.Error("Default executor should be used when not injected")
	}
}

// TestRDPCheckMockCallsVerified verifies mock was called with correct args
// The new detection order: 1) pgrep for GNOME RDP, 2) systemctl for xrdp
func TestRDPCheckMockCallsVerified(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	// No GNOME RDP running, xrdp is installed and active
	mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = checks.MockResponse{
		Output: []byte(""),
		Error:  checks.ErrConnectionRefused,
	}
	mockExec.Responses["systemctl list-unit-files xrdp.service"] = checks.MockResponse{
		Output: []byte("xrdp.service enabled"),
		Error:  nil,
	}
	mockExec.Responses["systemctl is-active xrdp"] = checks.MockResponse{
		Output: []byte("active"),
		Error:  nil,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec))
	check.Run(context.Background())

	// Verify the mock was called with expected sequence:
	// 1. pgrep -f gnome-remote-desktop-daemon (detect GNOME RDP)
	// 2. systemctl list-unit-files xrdp.service (detect xrdp installed)
	// 3. systemctl is-active xrdp (check xrdp status)
	if len(mockExec.Calls) < 1 {
		t.Errorf("Expected at least 1 call, got %d", len(mockExec.Calls))
		return
	}

	// First call should be pgrep for GNOME RDP detection
	firstCall := mockExec.Calls[0]
	if firstCall.Name != "pgrep" {
		t.Errorf("Expected first command 'pgrep', got %q", firstCall.Name)
	}
	if len(firstCall.Args) < 2 || firstCall.Args[0] != "-f" || firstCall.Args[1] != "gnome-remote-desktop-daemon" {
		t.Errorf("Expected args [-f gnome-remote-desktop-daemon], got %v", firstCall.Args)
	}
}
