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

func TestRDPCheckRunWithMock_GnomeRDPConfiguredButNotRunning(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := testutil.NewMockExecutor()
	// GNOME Remote Desktop IS configured (grdctl shows enabled)
	mockExec.Responses["grdctl status"] = testutil.MockResponse{
		Output: []byte("RDP:\n\tStatus: enabled\n\tPort: 3389"),
		Error:  nil,
	}
	// But the daemon is NOT running
	mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = testutil.MockResponse{
		Output: []byte(""),
		Error:  testutil.ErrConnectionRefused,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" }))
	result := check.Run(context.Background())

	// Should be WARNING (configured but not running is a problem)
	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusWarning)
	}
	if result.Message != "GNOME Remote Desktop is configured but not running" {
		t.Errorf("Message = %q, want %q", result.Message, "GNOME Remote Desktop is configured but not running")
	}
	if result.Details["configured"] != true {
		t.Errorf("Details[configured] = %v, want true", result.Details["configured"])
	}
	if result.Details["running"] != false {
		t.Errorf("Details[running] = %v, want false", result.Details["running"])
	}
}

// TestRDPCheckRunWithMock_GnomeRDPNotConfigured tests when GNOME RDP is not set up at all
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
func TestRDPCheckRunWithMock_GnomeRDPNotConfigured(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := testutil.NewMockExecutor()
	// GNOME Remote Desktop is NOT configured (grdctl shows disabled or fails)
	mockExec.Responses["grdctl status"] = testutil.MockResponse{
		Output: []byte("RDP:\n\tStatus: disabled"),
		Error:  nil,
	}
	// No xrdp either
	mockExec.Responses["systemctl list-unit-files xrdp.service"] = testutil.MockResponse{
		Output: []byte(""),
		Error:  testutil.ErrConnectionRefused,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" }))
	result := check.Run(context.Background())

	// Should be OK (no RDP configured is fine - it's optional)
	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusOK)
	}
	if result.Message != "No RDP service installed (remote desktop not configured)" {
		t.Errorf("Message = %q, want %q", result.Message, "No RDP service installed (remote desktop not configured)")
	}
}

// TestRDPCheckRunWithMock_LinuxActive tests xrdp service active on Linux (no GNOME RDP)
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
func TestRDPCheckRunWithMock_LinuxActive(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := testutil.NewMockExecutor()
	// GNOME Remote Desktop is NOT configured
	mockExec.Responses["grdctl status"] = testutil.MockResponse{
		Output: []byte("RDP:\n\tStatus: disabled"),
		Error:  nil,
	}
	// xrdp is installed
	mockExec.Responses["systemctl list-unit-files xrdp.service"] = testutil.MockResponse{
		Output: []byte("xrdp.service enabled"),
		Error:  nil,
	}
	// xrdp is active
	mockExec.Responses["systemctl is-active xrdp"] = testutil.MockResponse{
		Output: []byte("active"),
		Error:  nil,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" }))
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

	mockExec := testutil.NewMockExecutor()
	// GNOME Remote Desktop is NOT configured
	mockExec.Responses["grdctl status"] = testutil.MockResponse{
		Output: []byte("RDP:\n\tStatus: disabled"),
		Error:  nil,
	}
	// xrdp is installed
	mockExec.Responses["systemctl list-unit-files xrdp.service"] = testutil.MockResponse{
		Output: []byte("xrdp.service disabled"),
		Error:  nil,
	}
	// xrdp is inactive
	mockExec.Responses["systemctl is-active xrdp"] = testutil.MockResponse{
		Output: []byte("inactive"),
		Error:  testutil.ErrConnectionRefused,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" }))
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

	mockExec := testutil.NewMockExecutor()
	// GNOME Remote Desktop is NOT configured (grdctl fails or shows disabled)
	mockExec.Responses["grdctl status"] = testutil.MockResponse{
		Output: []byte(""),
		Error:  testutil.ErrCommandNotFound,
	}
	// xrdp is NOT installed
	mockExec.Responses["systemctl list-unit-files xrdp.service"] = testutil.MockResponse{
		Output: []byte(""),
		Error:  testutil.ErrConnectionRefused,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" }))
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

	mockExec := testutil.NewMockExecutor()
	// GNOME Remote Desktop is NOT configured
	mockExec.Responses["grdctl status"] = testutil.MockResponse{
		Output: []byte(""),
		Error:  testutil.ErrCommandNotFound,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" }))
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

	mockExec := testutil.NewMockExecutor()
	mockExec.Responses["sc.exe query TermService"] = testutil.MockResponse{
		Output: []byte("SERVICE_NAME: TermService\n        TYPE               : 20  WIN32_SHARE_PROCESS\n        STATE              : 4  RUNNING"),
		Error:  nil,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" }))
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

	mockExec := testutil.NewMockExecutor()
	mockExec.Responses["sc.exe query TermService"] = testutil.MockResponse{
		Output: []byte("SERVICE_NAME: TermService\n        TYPE               : 20  WIN32_SHARE_PROCESS\n        STATE              : 1  STOPPED"),
		Error:  nil,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" }))
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

	mockExec := testutil.NewMockExecutor()
	mockExec.Responses["sc.exe query TermService"] = testutil.MockResponse{
		Output: []byte(""),
		Error:  testutil.ErrCommandNotFound,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" }))
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

	mockExec := testutil.NewMockExecutor()
	check := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" }))
	result := check.Run(context.Background())

	// No RDP on macOS is OK - RDP is optional
	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusOK)
	}
}

// TestRDPCheckExecutorInjection verifies executor is properly injected
func TestRDPCheckExecutorInjection(t *testing.T) {
	mockExec := testutil.NewMockExecutor()
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
// Detection is delegated to shared hostinventory; this test verifies that the
// classifier still receives both provider observations.
func TestRDPCheckMockCallsVerified(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := testutil.NewMockExecutor()
	// GNOME RDP not configured, xrdp is installed and active
	mockExec.Responses["grdctl status"] = testutil.MockResponse{
		Output: []byte("RDP:\n\tStatus: disabled"),
		Error:  nil,
	}
	mockExec.Responses["systemctl list-unit-files xrdp.service"] = testutil.MockResponse{
		Output: []byte("xrdp.service enabled"),
		Error:  nil,
	}
	mockExec.Responses["systemctl is-active xrdp"] = testutil.MockResponse{
		Output: []byte("active"),
		Error:  nil,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" }))
	check.Run(context.Background())

	seen := map[string]bool{}
	for _, call := range mockExec.Calls {
		seen[call.Name+" "+strings.Join(call.Args, " ")] = true
	}
	for _, command := range []string{"grdctl status", "systemctl list-unit-files xrdp.service"} {
		if !seen[command] {
			t.Errorf("expected shared classifier to observe %q; calls=%v", command, mockExec.Calls)
		}
	}
}

// keyringJournalKey is the mock key for the boot-scoped keyring-load query.
func keyringJournalKey() string {
	return "journalctl --no-pager -o json -b 0 -g keyring"
}

// mockKeyringJournal wires the keyring-load journal read.
func mockKeyringJournal(m *testutil.MockExecutor, messages []string, err error) {
	lines := make([]string, 0, len(messages))
	for _, msg := range messages {
		lines = append(lines, journalEntry(msg))
	}
	m.Responses[keyringJournalKey()] = testutil.MockResponse{
		Output: []byte(strings.Join(lines, "\n")),
		Error:  err,
	}
}

// rdpKeyringHarness builds a check whose every probe is mocked, so a test can
// vary one signal at a time.
func rdpKeyringHarness(t *testing.T, grdctl string, autoLogin string, keyringPresent bool, keyringJournal []string, journalErr error) checks.Result {
	t.Helper()
	caps := &platform.Capabilities{Platform: platform.Linux, SupportsSystemd: true}
	mockExec := testutil.NewMockExecutor()
	mockExec.Responses["grdctl status"] = testutil.MockResponse{
		Output: []byte("RDP:\n\tStatus: enabled\n\tPort: 3389"),
	}
	mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = testutil.MockResponse{
		Output: []byte("12345"),
	}
	mockSessionBus(mockExec, "alice", "1000")
	mockSessionGrdctl(mockExec, "1000", grdctl)
	mockLoginKeyring(mockExec, "1000", keyringPresent)
	mockKeyringJournal(mockExec, keyringJournal, journalErr)

	return NewRDPCheck(caps,
		WithRDPExecutor(mockExec),
		WithRDPAutoLoginUserProvider(func() string { return autoLogin }),
	).Run(context.Background())
}

const emptyCredentialsOutput = "RDP:\n\tStatus: enabled\n\tUsername: (empty)\n\tPassword: (empty)\n"

// TestRDPCheckDistinguishesCorruptKeyringFromLockedKeyring is the regression
// test for a real misdiagnosis. The check reported "GDM autologin cannot unlock
// the login keyring" on a host whose keyring daemon had already logged that it
// threw the file away as malformed. Both remedies it offered required root, one
// destroyed the operator's session, and neither would have fixed the fault.
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
func TestRDPCheckDistinguishesCorruptKeyringFromLockedKeyring(t *testing.T) {
	const rejection = "keyring was in an invalid or unrecognized format: /home/alice/.local/share/keyrings/login.keyring"

	t.Run("rejected keyring file is named as the cause", func(t *testing.T) {
		result := rdpKeyringHarness(t, emptyCredentialsOutput, "alice", false,
			[]string{rejection, "gkr-pam: couldn't unlock the login keyring."}, nil)

		if result.Details["keyringCorrupt"] != true {
			t.Fatalf("keyringCorrupt = %v, want true", result.Details["keyringCorrupt"])
		}
		if result.Details["keyringFileRejected"] != true {
			t.Errorf("keyringFileRejected = %v, want true", result.Details["keyringFileRejected"])
		}
		if got := result.Details["keyringFilePath"]; got != "/home/alice/.local/share/keyrings/login.keyring" {
			t.Errorf("keyringFilePath = %v, want the path the daemon named", got)
		}
		// The posture claim must be withdrawn: it is the wrong diagnosis for a
		// file that never loaded, and acting on it costs root and a session.
		if !strings.Contains(result.Message, "malformed") {
			t.Errorf("Message must name the file fault, got: %s", result.Message)
		}
		if strings.Contains(result.Message, "autologin cannot unlock") {
			t.Errorf("Message must not blame autologin for a rejected file, got: %s", result.Message)
		}

		// The remedy must be the non-root repair, not the two root-requiring ones.
		actions, ok := result.Details["operatorActions"].([]string)
		if !ok || len(actions) == 0 {
			t.Fatalf("operatorActions missing: %#v", result.Details["operatorActions"])
		}
		joined := strings.Join(actions, " ")
		if !strings.Contains(joined, "keyring repair") {
			t.Errorf("remedies must name the repair command, got: %s", joined)
		}
		if strings.Contains(joined, "/etc/gdm3/custom.conf") {
			t.Errorf("remedies must not tell the operator to disable autologin, got: %s", joined)
		}
	})

	t.Run("locked-keyring posture is not reported when no rejection was logged", func(t *testing.T) {
		result := rdpKeyringHarness(t, emptyCredentialsOutput, "alice", false,
			[]string{"gnome-keyring-daemon: some unrelated keyring message"}, nil)

		if result.Details["keyringCorrupt"] != false {
			t.Errorf("keyringCorrupt = %v, want false", result.Details["keyringCorrupt"])
		}
		if _, present := result.Details["lockedKeyringPosture"]; present {
			t.Errorf("lockedKeyringPosture must be retired, got: %v", result.Details["lockedKeyringPosture"])
		}
		if strings.Contains(result.Message, "autologin cannot unlock") {
			t.Errorf("Message must not advise changing autologin, got: %s", result.Message)
		}
	})

	// An unreadable journal must never be read as "the keyring loaded fine",
	// which is the same false-green shape the denial probe already guards.
	t.Run("unreadable journal never asserts corruption", func(t *testing.T) {
		result := rdpKeyringHarness(t, emptyCredentialsOutput, "alice", false, nil, errors.New("journalctl unavailable"))

		if result.Details["keyringJournalReadable"] != false {
			t.Errorf("keyringJournalReadable = %v, want false", result.Details["keyringJournalReadable"])
		}
		if result.Details["keyringCorrupt"] != false {
			t.Errorf("keyringCorrupt = %v, want false when the evidence could not be read", result.Details["keyringCorrupt"])
		}
		if result.Status == checks.StatusOK {
			t.Errorf("a failed journal read must not produce OK, got %v", result.Status)
		}
	})

	// Credentials present means no fault to explain, so no root cause is claimed
	// even though the rejection line is in the journal from an earlier boot state.
	t.Run("healthy credentials are not annotated with a keyring cause", func(t *testing.T) {
		result := rdpKeyringHarness(t, "RDP:\n\tStatus: enabled\n\tUsername: alice\n\tPassword: hunter2\n",
			"alice", false, []string{rejection}, nil)

		if result.Details["keyringCorrupt"] != false {
			t.Errorf("keyringCorrupt = %v, want false when credentials are present", result.Details["keyringCorrupt"])
		}
		if strings.Contains(result.Message, "malformed") {
			t.Errorf("Message must not claim a fault when credentials are present, got: %s", result.Message)
		}
	})
}

// TestRDPRepairKeyringActionAvailability covers the one automated repair
// autoheal will perform on the user-session credential model.
// [REQ:INFRA-RDP-001] [REQ:HEAL-ACTION-001]
func TestRDPRepairKeyringActionAvailability(t *testing.T) {
	find := func(actions []checks.RecoveryAction, id string) (checks.RecoveryAction, bool) {
		for _, action := range actions {
			if action.ID == id {
				return action, true
			}
		}
		return checks.RecoveryAction{}, false
	}

	caps := &platform.Capabilities{Platform: platform.Linux, SupportsSystemd: true}
	check := NewRDPCheck(caps, WithRDPExecutor(testutil.NewMockExecutor()))
	check.cachedServiceInfo = &RDPServiceInfo{Type: RDPTypeGnome, ServiceName: "gnome-remote-desktop", IsUserSession: true}

	t.Run("offered when the keyring is corrupt", func(t *testing.T) {
		result := &checks.Result{Details: map[string]interface{}{
			"status":          "active",
			"credentialState": string(CredentialStateEmpty),
			"credentialModel": string(CredentialModelUserSession),
			"keyringCorrupt":  true,
		}}
		actions := check.RecoveryActions(result)

		repair, ok := find(actions, "repair-keyring")
		if !ok || !repair.Available {
			t.Fatalf("repair-keyring must be available for a corrupt keyring: %+v", repair)
		}
		if repair.Dangerous {
			t.Error("repair-keyring must not be marked dangerous: it needs no root and destroys no session")
		}
		// Offering "report it" beside "fix it" invites the operator to pick the
		// useless one.
		if incident, ok := find(actions, "raise-incident"); ok && incident.Available {
			t.Error("raise-incident must be withdrawn when a repair is available")
		}
	})

	t.Run("withheld when the keyring is merely locked", func(t *testing.T) {
		result := &checks.Result{Details: map[string]interface{}{
			"status":          "active",
			"credentialState": string(CredentialStateEmpty),
			"credentialModel": string(CredentialModelUserSession),
			"keyringCorrupt":  false,
		}}
		actions := check.RecoveryActions(result)

		if repair, ok := find(actions, "repair-keyring"); ok && repair.Available {
			t.Error("repair-keyring must not be offered for a locked keyring: there is nothing malformed to repair")
		}
		if incident, ok := find(actions, "raise-incident"); !ok || !incident.Available {
			t.Error("raise-incident must remain available when no repair applies")
		}
	})
}

// TestKeyringPathFromMessage covers the path extraction an operator needs on a
// host with more than one keyring file.
func TestKeyringPathFromMessage(t *testing.T) {
	cases := []struct {
		message string
		want    string
	}{
		{"keyring was in an invalid or unrecognized format: /home/a/.local/share/keyrings/login.keyring", "/home/a/.local/share/keyrings/login.keyring"},
		{"gnome-keyring-daemon[1]: keyring was in an invalid or unrecognized format: /x/y.keyring", "/x/y.keyring"},
		{"unrelated message", ""},
	}
	for _, testCase := range cases {
		if got := keyringPathFromMessage(testCase.message); got != testCase.want {
			t.Errorf("keyringPathFromMessage(%q) = %q, want %q", testCase.message, got, testCase.want)
		}
	}
}

// TestRDPCheckReportsRepairPending covers the state after a successful repair
// but before the next login.
//
// Without it the check could never leave "malformed": the rejection is
// boot-scoped evidence that stays in the journal until reboot, so an operator
// following the check's advice would re-run a repair that is already done.
// [REQ:INFRA-RDP-003] [REQ:TEST-SEAM-001]
func TestRDPCheckReportsRepairPending(t *testing.T) {
	const rejection = "keyring was in an invalid or unrecognized format: /home/alice/.local/share/keyrings/login.keyring"
	const inspectKey = "vrooli credentials keyring inspect --path /home/alice/.local/share/keyrings/login.keyring --format json"

	build := func(inspectOutput string, inspectErr error) checks.Result {
		caps := &platform.Capabilities{Platform: platform.Linux, SupportsSystemd: true}
		mockExec := testutil.NewMockExecutor()
		mockExec.Responses["grdctl status"] = testutil.MockResponse{Output: []byte("RDP:\n\tStatus: enabled\n\tPort: 3389")}
		mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = testutil.MockResponse{Output: []byte("12345")}
		mockSessionBus(mockExec, "alice", "1000")
		mockSessionGrdctl(mockExec, "1000", emptyCredentialsOutput)
		mockLoginKeyring(mockExec, "1000", false)
		mockKeyringJournal(mockExec, []string{rejection}, nil)
		mockExec.Responses[inspectKey] = testutil.MockResponse{Output: []byte(inspectOutput), Error: inspectErr}
		previousInspect := keyringInspectOutput
		keyringInspectOutput = func(ctx context.Context, path string) ([]byte, error) {
			return mockExec.Output(ctx, "vrooli", "credentials", "keyring", "inspect", "--path", path, "--format", "json")
		}
		t.Cleanup(func() { keyringInspectOutput = previousInspect })
		return NewRDPCheck(caps, WithRDPExecutor(mockExec),
			WithRDPAutoLoginUserProvider(func() string { return "alice" })).Run(context.Background())
	}

	t.Run("file parses again so the repair is pending a login", func(t *testing.T) {
		result := build(`{"reports":[{"path":"/home/alice/.local/share/keyrings/login.keyring","loadable":true,"repaired":0}]}`, nil)

		if result.Details["keyringRepairPending"] != true {
			t.Fatalf("keyringRepairPending = %v, want true", result.Details["keyringRepairPending"])
		}
		if !strings.Contains(result.Message, "repaired") {
			t.Errorf("Message must say the file is repaired, got: %s", result.Message)
		}
		actions, _ := result.Details["operatorActions"].([]string)
		joined := strings.Join(actions, " ")
		if strings.Contains(joined, "keyring repair") {
			t.Errorf("must not tell the operator to re-run a repair that is already done, got: %s", joined)
		}
		if !strings.Contains(joined, "Log out and back in") {
			t.Errorf("remedies must name the re-login, got: %s", joined)
		}
	})

	t.Run("file still malformed keeps the repair remedy", func(t *testing.T) {
		result := build(`{"reports":[{"path":"/home/alice/.local/share/keyrings/login.keyring","loadable":false,"repaired":0}]}`, nil)

		if result.Details["keyringRepairPending"] != false {
			t.Fatalf("keyringRepairPending = %v, want false", result.Details["keyringRepairPending"])
		}
		actions, _ := result.Details["operatorActions"].([]string)
		if !strings.Contains(strings.Join(actions, " "), "keyring repair") {
			t.Errorf("remedies must still offer the repair, got: %v", actions)
		}
	})

	// An inspect that cannot run must not be read as "already repaired": that
	// would drop the repair remedy on a host that still needs it.
	t.Run("unreadable inspect never claims a pending repair", func(t *testing.T) {
		result := build("", errors.New("vrooli not on PATH"))

		if result.Details["keyringRepairPending"] != false {
			t.Fatalf("keyringRepairPending = %v, want false when inspect failed", result.Details["keyringRepairPending"])
		}
		actions, _ := result.Details["operatorActions"].([]string)
		if !strings.Contains(strings.Join(actions, " "), "keyring repair") {
			t.Errorf("remedies must still offer the repair, got: %v", actions)
		}
	})
}
