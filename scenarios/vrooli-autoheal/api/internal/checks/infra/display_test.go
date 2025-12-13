// Package infra provides tests for display manager health checks
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
package infra

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// setMockResponse is a helper to set a response on the mock executor
func setMockResponse(m *checks.MockExecutor, key string, output []byte, err error) {
	m.Responses[key] = checks.MockResponse{Output: output, Error: err}
}

// displayTestCaps returns platform capabilities for display tests
func displayTestCaps() *platform.Capabilities {
	return &platform.Capabilities{
		Platform:         platform.Linux,
		SupportsSystemd:  true,
		IsHeadlessServer: false,
	}
}

// TestDisplayManagerCheckInterface verifies DisplayManagerCheck implements Check and HealableCheck
// [REQ:INFRA-DISPLAY-001]
func TestDisplayManagerCheckInterface(t *testing.T) {
	var _ checks.Check = (*DisplayManagerCheck)(nil)
	var _ checks.HealableCheck = (*DisplayManagerCheck)(nil)

	check := NewDisplayManagerCheck(displayTestCaps())
	if check.ID() != "infra-display" {
		t.Errorf("ID() = %q, want %q", check.ID(), "infra-display")
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
	if check.Category() != checks.CategoryInfrastructure {
		t.Error("Category should be infrastructure")
	}

	// Should be Linux-only
	platforms := check.Platforms()
	if len(platforms) == 0 {
		t.Error("DisplayManagerCheck should specify platforms")
	}
	hasLinux := false
	for _, p := range platforms {
		if p == platform.Linux {
			hasLinux = true
		}
	}
	if !hasLinux {
		t.Error("DisplayManagerCheck should include Linux platform")
	}
}

// TestDisplayManagerCheckRunHeadless verifies behavior on headless servers
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckRunHeadless(t *testing.T) {
	headlessCaps := &platform.Capabilities{
		Platform:         platform.Linux,
		SupportsSystemd:  true,
		IsHeadlessServer: true,
	}

	mockExec := checks.NewMockExecutor()
	check := NewDisplayManagerCheck(headlessCaps, WithDisplayExecutor(mockExec))

	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want OK for headless server", result.Status)
	}
	if result.Message != "Headless server - no display manager expected" {
		t.Errorf("Message = %q, unexpected", result.Message)
	}
	// Should not have made any executor calls
	if len(mockExec.Calls) != 0 {
		t.Errorf("Expected 0 executor calls for headless, got %d", len(mockExec.Calls))
	}
}

// TestDisplayManagerCheckRunNoDisplayManager verifies behavior when no DM is detected
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckRunNoDisplayManager(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	// Return multi-user.target (not graphical)
	setMockResponse(mockExec, "systemctl get-default", []byte("multi-user.target\n"), nil)
	// All DMs not enabled
	for _, dm := range supportedDisplayManagers {
		setMockResponse(mockExec, "systemctl is-enabled "+dm, []byte("disabled\n"), errors.New("not enabled"))
	}

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want OK for no display manager", result.Status)
	}
	if result.Message != "No display manager detected (headless system or custom setup)" {
		t.Errorf("Message = %q, unexpected", result.Message)
	}
}

// TestDisplayManagerCheckRunGDMActive verifies behavior when GDM is running
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckRunGDMActive(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	// Graphical target is default
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	// GDM is active
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	// Other DMs not active
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	// X11 not available (no DISPLAY)
	setMockResponse(mockExec, "printenv DISPLAY", []byte(""), errors.New("not set"))
	// No GNOME RDP configured (simple server without RDP)
	setMockResponse(mockExec, "grdctl status", []byte(""), errors.New("not found"))
	// gnome-shell check - both generic and user-specific (if auto-login is configured on test machine)
	setMockResponse(mockExec, "pgrep gnome-shell", []byte("12345\n"), nil)
	// Also mock for potential auto-login user on test machine
	setMockResponse(mockExec, "pgrep -u matthalloran8 gnome-shell", []byte("12345\n"), nil)

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want OK for active GDM. Details: %+v", result.Status, result.Details)
	}
	if result.Details == nil {
		t.Fatal("Details should not be nil")
	}
	if result.Details["displayManager"] != "gdm" {
		t.Errorf("displayManager = %v, want gdm", result.Details["displayManager"])
	}
}

// TestDisplayManagerCheckRunDMNotActive verifies behavior when DM is not running
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckRunDMNotActive(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	// Graphical target is default
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	// GDM exists but is not active
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("inactive\n"), errors.New("inactive"))
	// Check enabled status
	setMockResponse(mockExec, "systemctl is-enabled gdm", []byte("enabled\n"), nil)
	// Other DMs not found
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
			setMockResponse(mockExec, "systemctl is-enabled "+dm, []byte("disabled\n"), errors.New("disabled"))
		}
	}
	// X11 not available
	setMockResponse(mockExec, "printenv DISPLAY", []byte(""), errors.New("not set"))

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusCritical {
		t.Errorf("Status = %v, want Critical for inactive display manager", result.Status)
	}
}

// TestDisplayManagerCheckRunWithX11 verifies X11 responsiveness checking
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckRunWithX11(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	// Graphical target and GDM active
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	// X11 available and responsive
	setMockResponse(mockExec, "printenv DISPLAY", []byte(":0\n"), nil)
	setMockResponse(mockExec, "xdpyinfo", []byte("name of display: :0\n"), nil)
	// No GNOME RDP configured
	setMockResponse(mockExec, "grdctl status", []byte(""), errors.New("not found"))
	// gnome-shell running (for auto-login user on test machine)
	setMockResponse(mockExec, "pgrep gnome-shell", []byte("12345\n"), nil)
	setMockResponse(mockExec, "pgrep -u matthalloran8 gnome-shell", []byte("12345\n"), nil)

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want OK for active DM with responsive X11", result.Status)
	}
	if result.Details["x11"] == nil {
		t.Error("X11 details should be present")
	}
	x11Details := result.Details["x11"].(map[string]interface{})
	if x11Details["available"] != true {
		t.Error("X11 should be marked as available")
	}
	if x11Details["responsive"] != true {
		t.Error("X11 should be marked as responsive")
	}
}

// TestDisplayManagerCheckRunWithX11Unresponsive verifies warning when X11 is unresponsive
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckRunWithX11Unresponsive(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	// Graphical target and GDM active
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	// X11 available but unresponsive
	setMockResponse(mockExec, "printenv DISPLAY", []byte(":0\n"), nil)
	setMockResponse(mockExec, "xdpyinfo", []byte(""), errors.New("Can't open display"))
	// No GNOME RDP configured
	setMockResponse(mockExec, "grdctl status", []byte(""), errors.New("not found"))
	// gnome-shell running (for auto-login user on test machine)
	setMockResponse(mockExec, "pgrep gnome-shell", []byte("12345\n"), nil)
	setMockResponse(mockExec, "pgrep -u matthalloran8 gnome-shell", []byte("12345\n"), nil)

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want Warning for unresponsive X11", result.Status)
	}
}

// TestDisplayManagerCheckRecoveryActions verifies recovery actions are correct
// [REQ:HEAL-ACTION-001]
func TestDisplayManagerCheckRecoveryActions(t *testing.T) {
	check := NewDisplayManagerCheck(displayTestCaps())
	actions := check.RecoveryActions(nil)

	if len(actions) == 0 {
		t.Error("DisplayManagerCheck should have recovery actions")
	}

	// Should have restart, status, logs actions
	actionIDs := make(map[string]checks.RecoveryAction)
	for _, a := range actions {
		actionIDs[a.ID] = a
	}

	// Verify restart action
	if restart, ok := actionIDs["restart"]; !ok {
		t.Error("Should have restart action")
	} else {
		if !restart.Dangerous {
			t.Error("restart should be marked as dangerous")
		}
	}

	// Verify status action
	if status, ok := actionIDs["status"]; !ok {
		t.Error("Should have status action")
	} else {
		if status.Dangerous {
			t.Error("status should not be dangerous")
		}
	}

	// Verify logs action
	if logs, ok := actionIDs["logs"]; !ok {
		t.Error("Should have logs action")
	} else {
		if logs.Dangerous {
			t.Error("logs should not be dangerous")
		}
	}
}

// TestDisplayManagerCheckExecuteActionStatus verifies status action execution
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckExecuteActionStatus(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	// Set up GDM as active display manager
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	// Status command
	setMockResponse(mockExec, "systemctl status gdm", []byte("gdm.service - GNOME Display Manager\n   Active: active (running)\n"), nil)

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "status")

	if !result.Success {
		t.Errorf("Status action should succeed, got error: %s", result.Error)
	}
	if result.ActionID != "status" {
		t.Errorf("ActionID = %q, want status", result.ActionID)
	}
	if result.CheckID != check.ID() {
		t.Errorf("CheckID = %q, want %q", result.CheckID, check.ID())
	}
	if result.Output == "" {
		t.Error("Output should contain status info")
	}
}

// TestDisplayManagerCheckExecuteActionLogs verifies logs action execution
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckExecuteActionLogs(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	// Set up GDM as active display manager
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	// Logs command
	setMockResponse(mockExec, "journalctl -u gdm -n 100 --no-pager", []byte("-- Logs begin at ...\nJan 01 12:00:00 gdm[1234]: Starting...\n"), nil)

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "logs")

	if !result.Success {
		t.Errorf("Logs action should succeed, got error: %s", result.Error)
	}
	if result.ActionID != "logs" {
		t.Errorf("ActionID = %q, want logs", result.ActionID)
	}
}

// TestDisplayManagerCheckExecuteActionRestart verifies restart action execution
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckExecuteActionRestart(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	// Set up GDM as active display manager
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	// Restart command
	setMockResponse(mockExec, "sudo systemctl restart gdm", []byte(""), nil)
	// No GNOME RDP configured - skips RDP wait loop
	setMockResponse(mockExec, "grdctl status", []byte(""), errors.New("not found"))
	// gnome-shell is running after restart (for verification)
	// The test machine has auto-login for matthalloran8, so mock the user-specific check
	setMockResponse(mockExec, "pgrep gnome-shell", []byte("12345\n"), nil)
	setMockResponse(mockExec, "pgrep -u matthalloran8 gnome-shell", []byte("12345\n"), nil)

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "restart")

	if !result.Success {
		t.Errorf("Restart action should succeed, got error: %s. Output: %s", result.Error, result.Output)
	}
	if result.ActionID != "restart" {
		t.Errorf("ActionID = %q, want restart", result.ActionID)
	}
}

// TestDisplayManagerCheckExecuteActionRestartFails verifies restart failure handling
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckExecuteActionRestartFails(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	// Set up GDM as active display manager
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	// Restart command fails
	setMockResponse(mockExec, "sudo systemctl restart gdm", []byte("Failed to restart gdm.service: Access denied\n"), errors.New("exit status 1"))

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "restart")

	if result.Success {
		t.Error("Restart action should fail when sudo fails")
	}
	if result.Error == "" {
		t.Error("Error should be set when restart fails")
	}
}

// TestDisplayManagerCheckExecuteActionUnknown verifies unknown action handling
// [REQ:HEAL-ACTION-001]
func TestDisplayManagerCheckExecuteActionUnknown(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "unknown-action")

	if result.Success {
		t.Error("Unknown action should fail")
	}
	if result.Error == "" {
		t.Error("Error should describe unknown action")
	}
}

// TestDisplayManagerCheckUsesInjectedCaps verifies platform caps are used
func TestDisplayManagerCheckUsesInjectedCaps(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: false,
	}
	check := NewDisplayManagerCheck(caps)
	if check.caps != caps {
		t.Error("DisplayManagerCheck should store injected capabilities")
	}
}

// TestDisplayManagerCheckWithLightDM verifies detection of LightDM
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckWithLightDM(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	// Graphical target is default
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	// GDM is not active, but LightDM is
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("inactive\n"), errors.New("inactive"))
	setMockResponse(mockExec, "systemctl is-active gdm3", []byte("inactive\n"), errors.New("inactive"))
	setMockResponse(mockExec, "systemctl is-active lightdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" && dm != "gdm3" && dm != "lightdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	// No X11
	setMockResponse(mockExec, "printenv DISPLAY", []byte(""), errors.New("not set"))
	// No GNOME RDP configured
	setMockResponse(mockExec, "grdctl status", []byte(""), errors.New("not found"))

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Details["displayManager"] != "lightdm" {
		t.Errorf("displayManager = %v, want lightdm", result.Details["displayManager"])
	}
}

// TestDisplayManagerCheckGnomeRDPConfiguredNoSession verifies critical status when RDP configured but no session
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckGnomeRDPConfiguredNoSession(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	// GDM is active
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	// No X11
	setMockResponse(mockExec, "printenv DISPLAY", []byte(""), errors.New("not set"))
	// GNOME RDP is configured
	setMockResponse(mockExec, "grdctl status", []byte("RDP:\n\tStatus: enabled\n\tPort: 3389\n"), nil)
	// But gnome-shell is NOT running
	setMockResponse(mockExec, "pgrep gnome-shell", []byte(""), errors.New("no process"))
	// Port 3389 is NOT listening
	setMockResponse(mockExec, "ss -tln", []byte("State    Recv-Q   Send-Q     Local Address:Port      Peer Address:Port  Process\nLISTEN   0        128              0.0.0.0:22             0.0.0.0:*\n"), nil)

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.Run(context.Background())

	// Should be critical because RDP is configured but no session is available
	if result.Status != checks.StatusCritical {
		t.Errorf("Status = %v, want Critical when GNOME RDP configured but no gnome-shell", result.Status)
	}
	if result.Details["gnomeRDPConfigured"] != true {
		t.Error("gnomeRDPConfigured should be true")
	}
	if result.Details["gnomeShellRunning"] != false {
		t.Error("gnomeShellRunning should be false")
	}
}

// TestDisplayManagerCheckGnomeRDPHealthy verifies OK status when RDP is fully available
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckGnomeRDPHealthy(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	// GDM is active
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	// No X11
	setMockResponse(mockExec, "printenv DISPLAY", []byte(""), errors.New("not set"))
	// GNOME RDP is configured
	setMockResponse(mockExec, "grdctl status", []byte("RDP:\n\tStatus: enabled\n\tPort: 3389\n"), nil)
	// gnome-shell IS running (test machine has auto-login, so mock both generic and user-specific)
	setMockResponse(mockExec, "pgrep gnome-shell", []byte("12345\n"), nil)
	setMockResponse(mockExec, "pgrep -u matthalloran8 gnome-shell", []byte("12345\n"), nil)
	// Port 3389 IS listening
	setMockResponse(mockExec, "ss -tln", []byte("State    Recv-Q   Send-Q     Local Address:Port      Peer Address:Port  Process\nLISTEN   0        128              0.0.0.0:3389           0.0.0.0:*\n"), nil)

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.Run(context.Background())

	// Should be OK with RDP available message
	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want OK when GNOME RDP is fully healthy. Message: %s", result.Status, result.Message)
	}
	if result.Details["gnomeRDPConfigured"] != true {
		t.Error("gnomeRDPConfigured should be true")
	}
	if result.Details["gnomeShellRunning"] != true {
		t.Errorf("gnomeShellRunning should be true, got %v", result.Details["gnomeShellRunning"])
	}
	if result.Details["rdpPortListening"] != true {
		t.Error("rdpPortListening should be true")
	}
	if !strings.Contains(result.Message, "RDP available") {
		t.Errorf("Message should mention RDP available, got: %s", result.Message)
	}
}

// TestDisplayManagerCheckRDPPortNotListening verifies warning when RDP configured but port not listening
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckRDPPortNotListening(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	// GDM is active
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	// No X11
	setMockResponse(mockExec, "printenv DISPLAY", []byte(""), errors.New("not set"))
	// GNOME RDP is configured
	setMockResponse(mockExec, "grdctl status", []byte("RDP:\n\tStatus: enabled\n\tPort: 3389\n"), nil)
	// gnome-shell IS running (test machine has auto-login, so mock both)
	setMockResponse(mockExec, "pgrep gnome-shell", []byte("12345\n"), nil)
	setMockResponse(mockExec, "pgrep -u matthalloran8 gnome-shell", []byte("12345\n"), nil)
	// But port 3389 is NOT listening (yet)
	setMockResponse(mockExec, "ss -tln", []byte("State    Recv-Q   Send-Q     Local Address:Port      Peer Address:Port  Process\nLISTEN   0        128              0.0.0.0:22             0.0.0.0:*\n"), nil)

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.Run(context.Background())

	// Should be warning because gnome-shell is running but port not listening
	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want Warning when gnome-shell running but port not listening. Message: %s", result.Status, result.Message)
	}
}

// TestDisplayManagerCheckDiagnoseAction verifies diagnose action execution
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckDiagnoseAction(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	// Set up GDM as active display manager
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	// Diagnose commands
	setMockResponse(mockExec, "pgrep -a gnome-shell", []byte("12345 /usr/bin/gnome-shell\n"), nil)
	setMockResponse(mockExec, "loginctl list-sessions --no-legend", []byte("2 1000 matthalloran8 seat0\n"), nil)
	setMockResponse(mockExec, "grdctl status", []byte("RDP:\n\tStatus: enabled\n"), nil)
	setMockResponse(mockExec, "ss -tln", []byte("LISTEN 0 128 0.0.0.0:3389 0.0.0.0:*\n"), nil)

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "diagnose")

	if !result.Success {
		t.Errorf("Diagnose action should succeed, got error: %s", result.Error)
	}
	if result.ActionID != "diagnose" {
		t.Errorf("ActionID = %q, want diagnose", result.ActionID)
	}
	// Output should contain various diagnostic sections
	if !strings.Contains(result.Output, "Display Manager Status") {
		t.Error("Output should contain Display Manager Status section")
	}
	if !strings.Contains(result.Output, "GNOME Shell Status") {
		t.Error("Output should contain GNOME Shell Status section")
	}
	if !strings.Contains(result.Output, "GNOME RDP Status") {
		t.Error("Output should contain GNOME RDP Status section")
	}
}

// TestDisplayManagerCheckRecoverSessionAction verifies recover-session action availability
// [REQ:HEAL-ACTION-001]
func TestDisplayManagerCheckRecoverSessionAction(t *testing.T) {
	check := NewDisplayManagerCheck(displayTestCaps())

	// When gnome-shell IS running, recover-session should not be available
	runningResult := &checks.Result{
		Details: map[string]interface{}{
			"displayManager":     "gdm",
			"gnomeShellRunning":  true,
			"gnomeRDPConfigured": true,
		},
	}
	actions := check.RecoveryActions(runningResult)
	recoverFound := false
	for _, a := range actions {
		if a.ID == "recover-session" {
			if a.Available {
				t.Error("recover-session should NOT be available when gnome-shell is running")
			}
			recoverFound = true
		}
	}
	if !recoverFound {
		t.Error("recover-session action should be in the list")
	}

	// When gnome-shell is NOT running, recover-session should be available
	notRunningResult := &checks.Result{
		Details: map[string]interface{}{
			"displayManager":     "gdm",
			"gnomeShellRunning":  false,
			"gnomeRDPConfigured": true,
		},
	}
	actions = check.RecoveryActions(notRunningResult)
	for _, a := range actions {
		if a.ID == "recover-session" {
			if !a.Available {
				t.Error("recover-session should be available when gnome-shell is NOT running")
			}
		}
	}
}

