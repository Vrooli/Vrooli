// Package infra provides tests for display manager health checks
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
package infra

import (
	"context"
	"errors"
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

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want OK for active GDM", result.Status)
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

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "restart")

	if !result.Success {
		t.Errorf("Restart action should succeed, got error: %s", result.Error)
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

	check := NewDisplayManagerCheck(displayTestCaps(), WithDisplayExecutor(mockExec))
	result := check.Run(context.Background())

	if result.Details["displayManager"] != "lightdm" {
		t.Errorf("displayManager = %v, want lightdm", result.Details["displayManager"])
	}
}
