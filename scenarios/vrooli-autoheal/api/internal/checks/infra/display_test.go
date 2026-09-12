// Package infra provides tests for display manager health checks
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
package infra

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// setMockResponse is a helper to set a response on the mock executor
func setMockResponse(m *testutil.MockExecutor, key string, output []byte, err error) {
	m.Responses[key] = testutil.MockResponse{Output: output, Error: err}
}

// displayTestCaps returns platform capabilities for display tests
func displayTestCaps() *platform.Capabilities {
	return &platform.Capabilities{
		Platform:         platform.Linux,
		SupportsSystemd:  true,
		IsHeadlessServer: false,
	}
}

func newDisplayManagerCheckForTest(mockExec *testutil.MockExecutor) *DisplayManagerCheck {
	return NewDisplayManagerCheck(
		displayTestCaps(),
		WithDisplayExecutor(mockExec),
		WithDisplayAutoLoginUserProvider(func() string { return "" }),
	)
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

	mockExec := testutil.NewMockExecutor()
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
	mockExec := testutil.NewMockExecutor()
	// Return multi-user.target (not graphical)
	setMockResponse(mockExec, "systemctl get-default", []byte("multi-user.target\n"), nil)
	// All DMs not enabled
	for _, dm := range supportedDisplayManagers {
		setMockResponse(mockExec, "systemctl is-enabled "+dm, []byte("disabled\n"), errors.New("not enabled"))
	}

	check := newDisplayManagerCheckForTest(mockExec)
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
	mockExec := testutil.NewMockExecutor()
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
	// gnome-shell check - both generic and user-specific (if auto-login is configured on test machine)
	setMockResponse(mockExec, "pgrep gnome-shell", []byte("12345\n"), nil)
	// Also mock for potential auto-login user on test machine
	setMockResponse(mockExec, "pgrep -u alice gnome-shell", []byte("12345\n"), nil)

	check := newDisplayManagerCheckForTest(mockExec)
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
	mockExec := testutil.NewMockExecutor()
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

	check := newDisplayManagerCheckForTest(mockExec)
	result := check.Run(context.Background())

	if result.Status != checks.StatusCritical {
		t.Errorf("Status = %v, want Critical for inactive display manager", result.Status)
	}
}

// TestDisplayManagerCheckRunWithX11 verifies X11 responsiveness checking
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckRunWithX11(t *testing.T) {
	mockExec := testutil.NewMockExecutor()
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
	// gnome-shell running (for auto-login user on test machine)
	setMockResponse(mockExec, "pgrep gnome-shell", []byte("12345\n"), nil)
	setMockResponse(mockExec, "pgrep -u alice gnome-shell", []byte("12345\n"), nil)

	check := newDisplayManagerCheckForTest(mockExec)
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
	mockExec := testutil.NewMockExecutor()
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
	// gnome-shell running (for auto-login user on test machine)
	setMockResponse(mockExec, "pgrep gnome-shell", []byte("12345\n"), nil)
	setMockResponse(mockExec, "pgrep -u alice gnome-shell", []byte("12345\n"), nil)

	check := newDisplayManagerCheckForTest(mockExec)
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
	mockExec := testutil.NewMockExecutor()
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

	check := newDisplayManagerCheckForTest(mockExec)
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
	mockExec := testutil.NewMockExecutor()
	// Set up GDM as active display manager
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	// Logs command
	setMockResponse(mockExec, "journalctl --no-pager -o short-iso -u gdm -n 100", []byte("-- Logs begin at ...\nJan 01 12:00:00 gdm[1234]: Starting...\n"), nil)

	check := newDisplayManagerCheckForTest(mockExec)
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
	allowRecoveryGrant(t)
	mockExec := testutil.NewMockExecutor()
	// Set up GDM as active display manager
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	// Restart command
	setMockResponse(mockExec, "sudo -n /usr/bin/systemctl restart gdm", []byte(""), nil)
	// gnome-shell is running after restart (for verification)
	// The test machine has auto-login for alice, so mock the user-specific check
	setMockResponse(mockExec, "pgrep gnome-shell", []byte("12345\n"), nil)
	setMockResponse(mockExec, "pgrep -u alice gnome-shell", []byte("12345\n"), nil)

	check := newDisplayManagerCheckForTest(mockExec)
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
	allowRecoveryGrant(t)
	mockExec := testutil.NewMockExecutor()
	// Set up GDM as active display manager
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	// Restart command fails
	setMockResponse(mockExec, "sudo -n /usr/bin/systemctl restart gdm", []byte("Failed to restart gdm.service: Access denied\n"), errors.New("exit status 1"))

	check := newDisplayManagerCheckForTest(mockExec)
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
	mockExec := testutil.NewMockExecutor()
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}

	check := newDisplayManagerCheckForTest(mockExec)
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
	mockExec := testutil.NewMockExecutor()
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

	check := newDisplayManagerCheckForTest(mockExec)
	result := check.Run(context.Background())

	if result.Details["displayManager"] != "lightdm" {
		t.Errorf("displayManager = %v, want lightdm", result.Details["displayManager"])
	}
}

// TestDisplayManagerCheckMakesNoRDPClaims verifies that the display check owns the
// graphical-session dependency layer only. Even on a host where GNOME Remote Desktop
// is enabled and port 3389 is listening, infra-display must not report on RDP: that
// is infra-rdp's boundary, and the old "RDP available on port 3389" message was the
// direct source of a false green during a total RDP outage.
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckMakesNoRDPClaims(t *testing.T) {
	mockExec := testutil.NewMockExecutor()
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
	// gnome-shell IS running
	setMockResponse(mockExec, "loginctl show-seat seat0 -p ActiveSession --value", []byte("2\n"), nil)
	setMockResponse(mockExec, "loginctl show-session 2 -p Name --value", []byte("alice\n"), nil)
	setMockResponse(mockExec, "pgrep gnome-shell", []byte("12345\n"), nil)
	setMockResponse(mockExec, "pgrep -u alice gnome-shell", []byte("12345\n"), nil)

	check := newDisplayManagerCheckForTest(mockExec)
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want OK for healthy graphical session. Message: %s", result.Status, result.Message)
	}
	for _, field := range []string{"gnomeRDPConfigured", "rdpPortListening"} {
		if _, present := result.Details[field]; present {
			t.Errorf("Details must not contain RDP field %q; infra-rdp owns the RDP service layer", field)
		}
	}
	if strings.Contains(strings.ToLower(result.Message), "rdp") {
		t.Errorf("Message must make no statement about RDP, got: %s", result.Message)
	}
	if result.Metrics != nil {
		for _, sc := range result.Metrics.SubChecks {
			if strings.Contains(strings.ToLower(sc.Name), "rdp") {
				t.Errorf("SubCheck %q must not exist on infra-display", sc.Name)
			}
		}
	}
	// The check never shells out to grdctl any more.
	for _, call := range mockExec.Calls {
		if call.Name == "grdctl" {
			t.Errorf("infra-display must not call grdctl, got call: %s %v", call.Name, call.Args)
		}
	}
}

// TestDisplayManagerCheckNoSessionForAutoLoginUser verifies the check reports a
// missing desktop session for a configured auto-login user, without reference to
// any consumer of that session.
// [REQ:INFRA-DISPLAY-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckNoSessionForAutoLoginUser(t *testing.T) {
	mockExec := testutil.NewMockExecutor()
	setMockResponse(mockExec, "systemctl get-default", []byte("graphical.target\n"), nil)
	setMockResponse(mockExec, "systemctl is-active gdm", []byte("active\n"), nil)
	for _, dm := range supportedDisplayManagers {
		if dm != "gdm" {
			setMockResponse(mockExec, "systemctl is-active "+dm, []byte("inactive\n"), errors.New("inactive"))
		}
	}
	setMockResponse(mockExec, "printenv DISPLAY", []byte(""), errors.New("not set"))
	// gnome-shell is NOT running for the auto-login user
	setMockResponse(mockExec, "pgrep -u alice gnome-shell", []byte(""), errors.New("no process"))

	check := NewDisplayManagerCheck(
		displayTestCaps(),
		WithDisplayExecutor(mockExec),
		WithDisplayAutoLoginUserProvider(func() string { return "alice" }),
	)
	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want Warning when auto-login user has no session", result.Status)
	}
	if result.Details["gnomeShellRunning"] != false {
		t.Errorf("gnomeShellRunning should be false, got %v", result.Details["gnomeShellRunning"])
	}
	if strings.Contains(strings.ToLower(result.Message), "rdp") {
		t.Errorf("Message must make no statement about RDP, got: %s", result.Message)
	}
}

// TestDisplayManagerCheckDiagnoseAction verifies diagnose action execution
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestDisplayManagerCheckDiagnoseAction(t *testing.T) {
	mockExec := testutil.NewMockExecutor()
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
	setMockResponse(mockExec, "loginctl list-sessions --no-legend", []byte("2 1000 alice seat0\n"), nil)
	setMockResponse(mockExec, "loginctl show-seat seat0 -p ActiveSession --value", []byte("2\n"), nil)
	setMockResponse(mockExec, "loginctl show-session 2 -p Name --value", []byte("alice\n"), nil)

	check := newDisplayManagerCheckForTest(mockExec)
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
	if !strings.Contains(result.Output, "Seat Assignment") {
		t.Error("Output should contain Seat Assignment section")
	}
	if strings.Contains(result.Output, "RDP") {
		t.Errorf("Diagnose output must make no statement about RDP, got: %s", result.Output)
	}
}

// TestDisplayManagerCheckRecoverSessionAction verifies recover-session action availability
// [REQ:HEAL-ACTION-001]
func TestDisplayManagerCheckRecoverSessionAction(t *testing.T) {
	check := NewDisplayManagerCheck(displayTestCaps())

	// When gnome-shell IS running, recover-session should not be available
	runningResult := &checks.Result{
		Details: map[string]interface{}{
			"displayManager":    "gdm",
			"gnomeShellRunning": true,
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
			"displayManager":    "gdm",
			"gnomeShellRunning": false,
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
