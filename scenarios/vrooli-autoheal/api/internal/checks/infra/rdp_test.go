// Package infra tests for RDP health check
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
package infra

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"

	sharedhost "github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

type blockingRDPExecutor struct{}

func (blockingRDPExecutor) Output(ctx context.Context, _ string, _ ...string) ([]byte, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestDetectRDPServiceUsesTypedRemoteDesktopFact(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:          platform.Linux,
		SupportsSystemd:   true,
		DisplayAttached:   true,
		ActiveSessionUser: "alice",
		RemoteDesktop: sharedhost.RemoteDesktopCapability{
			Supported:        true,
			Observed:         true,
			Active:           true,
			Mode:             "user-shared",
			SelectedProvider: "gnome-user-shared",
			Providers: []sharedhost.RemoteDesktopProvider{{
				Name:        "gnome-user-shared",
				Present:     true,
				Active:      true,
				UserSession: true,
			}},
		},
	}
	check := NewRDPCheck(caps, WithRDPExecutor(blockingRDPExecutor{}))
	got := check.detectRDPService(context.Background())
	if got.Type != RDPTypeGnome || got.Mode != "user-shared" || !got.IsUserSession || !got.Active {
		t.Fatalf("detectRDPService() = %+v, want active user-shared GNOME service", got)
	}
}

func (e blockingRDPExecutor) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return e.Output(ctx, name, args...)
}

func (e blockingRDPExecutor) Run(ctx context.Context, name string, args ...string) error {
	_, err := e.Output(ctx, name, args...)
	return err
}

func TestRDPCheckRunBoundsNeverReturningHostProbe(t *testing.T) {
	check := NewRDPCheck(&platform.Capabilities{Platform: platform.Linux, SupportsSystemd: true}, WithRDPExecutor(blockingRDPExecutor{}))
	start := time.Now()
	result := check.Run(context.Background())
	elapsed := time.Since(start)
	if elapsed > probeTimeout+time.Second {
		t.Fatalf("RDP check took %s with a non-returning host probe, want <= %s; result=%+v", elapsed, probeTimeout+time.Second, result)
	}
}

// =============================================================================
// RDPCheck Unit Tests with Mock Interfaces
// =============================================================================

// TestRDPCheckRunWithMock_GnomeRDPRunning tests GNOME Remote Desktop detection when running
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
func TestRDPCheckRunWithMock_GnomeRDPRunning(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := testutil.NewMockExecutor()
	// GNOME Remote Desktop is configured (detected via grdctl status)
	mockExec.Responses["grdctl status"] = testutil.MockResponse{
		Output: []byte("RDP:\n\tStatus: enabled\n\tPort: 3389"),
		Error:  nil,
	}
	// GNOME Remote Desktop daemon is running
	mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = testutil.MockResponse{
		Output: []byte("12345"),
		Error:  nil,
	}
	mockExec.Responses["pgrep -a -f gnome-remote-desktop-daemon"] = testutil.MockResponse{
		Output: []byte("12345 /usr/libexec/gnome-remote-desktop-daemon"),
		Error:  nil,
	}
	// Credentials are set, so a remote client can actually authenticate.
	mockSessionBus(mockExec, "alice", "1000")
	mockSessionGrdctl(mockExec, "1000", "RDP:\n\tStatus: enabled\n\tPort: 3389\n\tUsername: alice\n\tPassword: hunter2\n")

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" }))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusOK)
	}
	if result.Details["credentialState"] != string(CredentialStatePresent) {
		t.Errorf("credentialState = %v, want %q", result.Details["credentialState"], CredentialStatePresent)
	}
	if result.Details["type"] != string(RDPTypeGnome) {
		t.Errorf("Details[type] = %v, want %q", result.Details["type"], RDPTypeGnome)
	}
}

func TestRDPCheckDeclaredIntentVerdicts(t *testing.T) {
	tests := []struct {
		name        string
		experience  string
		managed     bool
		wantStatus  checks.Status
		wantVerdict string
	}{
		{name: "unmanaged", managed: false, wantStatus: checks.StatusOK, wantVerdict: RemoteDesktopVerdictUnmanaged},
		{name: "matching", managed: true, experience: "login-screen", wantStatus: checks.StatusOK, wantVerdict: RemoteDesktopVerdictMatching},
		{name: "drifted", managed: true, experience: "login-screen", wantStatus: checks.StatusWarning, wantVerdict: RemoteDesktopVerdictDrifted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := &platform.Capabilities{Platform: platform.Linux, SupportsSystemd: true}
			mockExec := testutil.NewMockExecutor()
			mockExec.Responses["grdctl status"] = testutil.MockResponse{Output: []byte("RDP:\n\tStatus: enabled\n")}
			provider := func(context.Context) (RemoteDesktopIntent, error) {
				return RemoteDesktopIntent{Managed: tt.managed, Experience: tt.experience, Provider: "auto"}, nil
			}
			if tt.name == "matching" {
				mockExec.Responses["systemctl is-enabled gnome-remote-desktop.service"] = testutil.MockResponse{Output: []byte("enabled\n")}
				mockExec.Responses["systemctl is-active gnome-remote-desktop.service"] = testutil.MockResponse{Output: []byte("active\n")}
				mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = testutil.MockResponse{Output: []byte("12345")}
				mockExec.Responses["pgrep -a -f gnome-remote-desktop-daemon"] = testutil.MockResponse{Output: []byte("12345 /usr/libexec/gnome-remote-desktop-daemon")}
				mockSessionBus(mockExec, "alice", "1000")
				mockSessionGrdctl(mockExec, "1000", "RDP:\n\tStatus: enabled\n\tUsername: alice\n\tPassword: hunter2\n")
			}

			check := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPDesiredStateProvider(provider), WithRDPAutoLoginUserProvider(func() string { return "" }))
			result := check.Run(context.Background())
			if result.Status != tt.wantStatus {
				t.Fatalf("status = %v, want %v; message=%q", result.Status, tt.wantStatus, result.Message)
			}
			if got := result.Details["desiredVerdict"]; got != tt.wantVerdict {
				t.Fatalf("desiredVerdict = %v, want %q", got, tt.wantVerdict)
			}
			t.Logf("declared intent verdict=%s status=%s message=%q", result.Details["desiredVerdict"], result.Status, result.Message)
			if tt.name == "drifted" && !strings.Contains(result.Message, "declared login-screen") {
				t.Fatalf("drift message = %q", result.Message)
			}
			if tt.name == "unmanaged" {
				actions := check.RecoveryActions(&result)
				if len(actions) != 0 {
					t.Fatalf("unmanaged capability offered %d recovery actions", len(actions))
				}
			}
		})
	}
}

// mockSessionBus wires the loginctl and id lookups that sessionBusEnv performs.
func mockSessionBus(m *testutil.MockExecutor, user, uid string) {
	m.Responses["loginctl show-seat seat0 -p ActiveSession --value"] = testutil.MockResponse{Output: []byte("2\n")}
	m.Responses["loginctl show-session 2 -p Name --value"] = testutil.MockResponse{Output: []byte(user + "\n")}
	m.Responses["id -u "+user] = testutil.MockResponse{Output: []byte(uid + "\n")}
	m.Responses["pgrep -u "+user+" gnome-shell"] = testutil.MockResponse{Output: []byte("4242\n")}
}

// sessionGrdctlKey builds the mock key for a grdctl probe carrying the session
// bus environment.
func sessionGrdctlKey(uid string) string {
	return "env XDG_RUNTIME_DIR=/run/user/" + uid +
		" DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/" + uid + "/bus grdctl status"
}

// mockSessionGrdctl wires the environment-carrying grdctl probe.
func mockSessionGrdctl(m *testutil.MockExecutor, uid, output string) {
	m.Responses[sessionGrdctlKey(uid)] = testutil.MockResponse{Output: []byte(output)}
}

// TestClassifyCredentialOutput covers the three credential states, including the
// verified trap where grdctl omits the credential lines entirely when it cannot
// reach the session bus while still printing "Status: enabled".
// [REQ:INFRA-RDP-001]
func TestClassifyCredentialOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   CredentialState
	}{
		{
			name:   "credentials present",
			output: "RDP:\n\tStatus: enabled\n\tPort: 3389\n\tUsername: alice\n\tPassword: hunter2\n",
			want:   CredentialStatePresent,
		},
		{
			name:   "credentials empty",
			output: "RDP:\n\tStatus: enabled\n\tPort: 3389\n\tUsername: (empty)\n\tPassword: (empty)\n",
			want:   CredentialStateEmpty,
		},
		{
			name:   "password empty but username set",
			output: "RDP:\n\tStatus: enabled\n\tUsername: alice\n\tPassword: (empty)\n",
			want:   CredentialStateEmpty,
		},
		{
			name:   "null credential value",
			output: "RDP:\n\tStatus: enabled\n\tUsername: (null)\n\tPassword: (null)\n",
			want:   CredentialStateEmpty,
		},
		{
			name:   "unreadable because lines are absent",
			output: "RDP:\n\tStatus: enabled\n\tPort: 3389\n\tView-only: no\n",
			want:   CredentialStateUnreadable,
		},
		{
			name:   "unreadable because grdctl reported a read failure",
			output: "Failed to read credentials: Cannot autolaunch D-Bus without X11 $DISPLAY.\nRDP:\n\tStatus: enabled\n\tPort: 3389\n",
			want:   CredentialStateUnreadable,
		},
		{
			name:   "empty output is unreadable",
			output: "",
			want:   CredentialStateUnreadable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyCredentialOutput(tt.output); got != tt.want {
				t.Errorf("classifyCredentialOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRDPCheckCredentialStateVerdicts proves a running, listening daemon does not
// report OK when it cannot authenticate anyone. This is the false green that
// made autoheal certify a healthy RDP service during a total outage.
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
func TestRDPCheckCredentialStateVerdicts(t *testing.T) {
	tests := []struct {
		name       string
		grdctl     string
		wantStatus checks.Status
		wantState  CredentialState
	}{
		{
			name:       "empty credentials are critical",
			grdctl:     "RDP:\n\tStatus: enabled\n\tUsername: (empty)\n\tPassword: (empty)\n",
			wantStatus: checks.StatusCritical,
			wantState:  CredentialStateEmpty,
		},
		{
			name:       "unreadable credentials are a warning",
			grdctl:     "Failed to read credentials: Cannot autolaunch D-Bus without X11 $DISPLAY.\nRDP:\n\tStatus: enabled\n",
			wantStatus: checks.StatusWarning,
			wantState:  CredentialStateUnreadable,
		},
		{
			name:       "missing credential lines are a warning, never OK",
			grdctl:     "RDP:\n\tStatus: enabled\n\tPort: 3389\n",
			wantStatus: checks.StatusWarning,
			wantState:  CredentialStateUnreadable,
		},
		{
			name:       "present credentials are OK",
			grdctl:     "RDP:\n\tStatus: enabled\n\tUsername: alice\n\tPassword: hunter2\n",
			wantStatus: checks.StatusOK,
			wantState:  CredentialStatePresent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := &platform.Capabilities{Platform: platform.Linux, SupportsSystemd: true}
			mockExec := testutil.NewMockExecutor()
			mockExec.Responses["grdctl status"] = testutil.MockResponse{
				Output: []byte("RDP:\n\tStatus: enabled\n\tPort: 3389"),
			}
			mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = testutil.MockResponse{
				Output: []byte("12345"),
			}
			mockSessionBus(mockExec, "alice", "1000")
			mockSessionGrdctl(mockExec, "1000", tt.grdctl)

			result := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" })).Run(context.Background())

			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v (message: %s)", result.Status, tt.wantStatus, result.Message)
			}
			if result.Details["credentialState"] != string(tt.wantState) {
				t.Errorf("credentialState = %v, want %q", result.Details["credentialState"], tt.wantState)
			}
			if tt.wantStatus == checks.StatusCritical && !strings.Contains(result.Message, "credentials") {
				t.Errorf("Message must name empty credentials as the cause, got: %s", result.Message)
			}
		})
	}
}

// denialJournalKey is the mock key for the denial-window journal query.
func denialJournalKey() string {
	return "journalctl --no-pager -o json --user-unit gnome-remote-desktop --since 15 minutes ago"
}

// journalEntry builds one journald JSON record.
func journalEntry(message string) string {
	return `{"__REALTIME_TIMESTAMP":"1785600000000000","PRIORITY":"6","MESSAGE":"` + message + `"}`
}

// mockDenialJournal wires the denial-window journal read.
func mockDenialJournal(m *testutil.MockExecutor, messages []string, err error) {
	lines := make([]string, 0, len(messages))
	for _, msg := range messages {
		lines = append(lines, journalEntry(msg))
	}
	m.Responses[denialJournalKey()] = testutil.MockResponse{
		Output: []byte(strings.Join(lines, "\n")),
		Error:  err,
	}
}

// TestRDPCheckDenialSignal covers the client-denial signal: denials present,
// denials absent, and a journal read that fails.
//
// The critical invariant is that neither a zero denial count nor a failed
// journal read may turn a non-OK credential verdict into OK. Both were the
// shape of the original false green.
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
func TestRDPCheckDenialSignal(t *testing.T) {
	tests := []struct {
		name         string
		grdctl       string
		journalLines []string
		journalErr   error
		wantStatus   checks.Status
		wantReadable bool
		wantDenials  int
	}{
		{
			name:         "denials observed with empty credentials",
			grdctl:       "RDP:\n\tStatus: enabled\n\tUsername: (empty)\n\tPassword: (empty)\n",
			journalLines: []string{"[RDP] Credentials are not set, denying client", "[RDP] Credentials are not set, denying client"},
			wantStatus:   checks.StatusCritical,
			wantReadable: true,
			wantDenials:  2,
		},
		{
			name:         "no denials observed still reports the credential fault",
			grdctl:       "RDP:\n\tStatus: enabled\n\tUsername: (empty)\n\tPassword: (empty)\n",
			journalLines: []string{"[RDP] Started session"},
			wantStatus:   checks.StatusCritical,
			wantReadable: true,
			wantDenials:  0,
		},
		{
			name:         "no denials with healthy credentials is OK",
			grdctl:       "RDP:\n\tStatus: enabled\n\tUsername: alice\n\tPassword: hunter2\n",
			journalLines: []string{"[RDP] Started session"},
			wantStatus:   checks.StatusOK,
			wantReadable: true,
			wantDenials:  0,
		},
		{
			name:         "denials observed outrank a healthy credential verdict",
			grdctl:       "RDP:\n\tStatus: enabled\n\tUsername: alice\n\tPassword: hunter2\n",
			journalLines: []string{"[RDP] denying client"},
			wantStatus:   checks.StatusCritical,
			wantReadable: true,
			wantDenials:  1,
		},
		{
			name:         "journal read failure never rescues a non-OK verdict",
			grdctl:       "RDP:\n\tStatus: enabled\n\tUsername: (empty)\n\tPassword: (empty)\n",
			journalErr:   testutil.ErrConnectionRefused,
			wantStatus:   checks.StatusCritical,
			wantReadable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := &platform.Capabilities{Platform: platform.Linux, SupportsSystemd: true}
			mockExec := testutil.NewMockExecutor()
			mockExec.Responses["grdctl status"] = testutil.MockResponse{
				Output: []byte("RDP:\n\tStatus: enabled\n\tPort: 3389"),
			}
			mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = testutil.MockResponse{
				Output: []byte("12345"),
			}
			mockSessionBus(mockExec, "alice", "1000")
			mockSessionGrdctl(mockExec, "1000", tt.grdctl)
			mockDenialJournal(mockExec, tt.journalLines, tt.journalErr)

			result := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" })).Run(context.Background())

			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v (message: %s)", result.Status, tt.wantStatus, result.Message)
			}
			if result.Details["journalReadable"] != tt.wantReadable {
				t.Errorf("journalReadable = %v, want %v", result.Details["journalReadable"], tt.wantReadable)
			}
			if result.Details["denialWindowMinutes"] != denialWindowMinutes {
				t.Errorf("denialWindowMinutes = %v, want %d", result.Details["denialWindowMinutes"], denialWindowMinutes)
			}
			if tt.wantReadable {
				if result.Details["recentDenials"] != tt.wantDenials {
					t.Errorf("recentDenials = %v, want %d", result.Details["recentDenials"], tt.wantDenials)
				}
			}
			if tt.wantDenials > 0 && !strings.Contains(result.Message, "denying clients") {
				t.Errorf("Message must state that clients are being denied, got: %s", result.Message)
			}
		})
	}
}

// keyringCollectionsKey is the mock key for the secret-service probe.
func keyringCollectionsKey(uid string) string {
	return "env XDG_RUNTIME_DIR=/run/user/" + uid +
		" DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/" + uid + "/bus" +
		" gdbus call --session --dest org.freedesktop.secrets" +
		" --object-path /org/freedesktop/secrets" +
		" --method org.freedesktop.DBus.Properties.Get" +
		" org.freedesktop.Secret.Service Collections"
}

// mockLoginKeyring wires the secret-service probe. When present is false the
// reply lists only the session collection, which is what an autologin host
// whose login keyring never unlocked actually returns.
func mockLoginKeyring(m *testutil.MockExecutor, uid string, present bool) {
	reply := "(<[objectpath '/org/freedesktop/secrets/collection/session']>,)\n"
	if present {
		reply = "(<[objectpath '/org/freedesktop/secrets/collection/login'," +
			" objectpath '/org/freedesktop/secrets/collection/session']>,)\n"
	}
	m.Responses[keyringCollectionsKey(uid)] = testutil.MockResponse{Output: []byte(reply)}
}

// TestRDPCheckPostureCorrelation covers the predictive signal: GDM autologin plus
// a user-session daemon plus an absent login keyring collection.
//
// The posture is recorded on every run but must never move the status on its own.
// An operator who unlocks the keyring by hand after boot still matches the
// posture while RDP works perfectly, and alarming that host would be noise.
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
func TestRDPCheckPostureCorrelation(t *testing.T) {
	tests := []struct {
		name           string
		autoLogin      string
		keyringPresent bool
		grdctl         string
		wantStatus     checks.Status
		wantPosture    bool
		wantCauseNamed bool
	}{
		{
			name:           "posture matched with credentials empty names the root cause",
			autoLogin:      "matthalloran8",
			keyringPresent: false,
			grdctl:         "RDP:\n\tStatus: enabled\n\tUsername: (empty)\n\tPassword: (empty)\n",
			wantStatus:     checks.StatusCritical,
			wantPosture:    true,
			wantCauseNamed: true,
		},
		{
			name:           "posture matched with credentials present stays OK",
			autoLogin:      "matthalloran8",
			keyringPresent: false,
			grdctl:         "RDP:\n\tStatus: enabled\n\tUsername: alice\n\tPassword: hunter2\n",
			wantStatus:     checks.StatusOK,
			wantPosture:    true,
			wantCauseNamed: false,
		},
		{
			name:           "posture unmatched with credentials empty is still critical",
			autoLogin:      "",
			keyringPresent: true,
			grdctl:         "RDP:\n\tStatus: enabled\n\tUsername: (empty)\n\tPassword: (empty)\n",
			wantStatus:     checks.StatusCritical,
			wantPosture:    false,
			wantCauseNamed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := &platform.Capabilities{Platform: platform.Linux, SupportsSystemd: true}
			mockExec := testutil.NewMockExecutor()
			mockExec.Responses["grdctl status"] = testutil.MockResponse{
				Output: []byte("RDP:\n\tStatus: enabled\n\tPort: 3389"),
			}
			mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = testutil.MockResponse{
				Output: []byte("12345"),
			}
			mockSessionBus(mockExec, "alice", "1000")
			mockSessionGrdctl(mockExec, "1000", tt.grdctl)
			mockLoginKeyring(mockExec, "1000", tt.keyringPresent)

			result := NewRDPCheck(caps,
				WithRDPExecutor(mockExec),
				WithRDPAutoLoginUserProvider(func() string { return tt.autoLogin }),
			).Run(context.Background())

			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v (message: %s)", result.Status, tt.wantStatus, result.Message)
			}
			if result.Details["lockedKeyringPosture"] != tt.wantPosture {
				t.Errorf("lockedKeyringPosture = %v, want %v", result.Details["lockedKeyringPosture"], tt.wantPosture)
			}
			if result.Details["autoLoginUser"] != tt.autoLogin {
				t.Errorf("autoLoginUser = %v, want %q", result.Details["autoLoginUser"], tt.autoLogin)
			}
			if result.Details["loginKeyringCollectionPresent"] != tt.keyringPresent {
				t.Errorf("loginKeyringCollectionPresent = %v, want %v",
					result.Details["loginKeyringCollectionPresent"], tt.keyringPresent)
			}
			if result.Details["sessionAvailable"] != true {
				t.Errorf("sessionAvailable = %v, want true", result.Details["sessionAvailable"])
			}

			causeNamed := strings.Contains(result.Message, "autologin cannot unlock the login keyring")
			if causeNamed != tt.wantCauseNamed {
				t.Errorf("root cause named = %v, want %v (message: %s)", causeNamed, tt.wantCauseNamed, result.Message)
			}
		})
	}
}

// TestRDPCheckNoGraphicalSession proves infra-rdp reports its own degradation when
// no graphical session exists, instead of restating display-manager health.
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
func TestRDPCheckNoGraphicalSession(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux, SupportsSystemd: true}
	mockExec := testutil.NewMockExecutor()
	mockExec.Responses["grdctl status"] = testutil.MockResponse{
		Output: []byte("RDP:\n\tStatus: enabled\n\tPort: 3389"),
	}
	mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = testutil.MockResponse{Output: []byte("12345")}
	mockSessionBus(mockExec, "alice", "1000")
	mockSessionGrdctl(mockExec, "1000", "RDP:\n\tStatus: enabled\n\tUsername: alice\n\tPassword: hunter2\n")
	// No gnome-shell for the session owner.
	mockExec.Responses["pgrep -u alice gnome-shell"] = testutil.MockResponse{Error: testutil.ErrConnectionRefused}

	result := NewRDPCheck(caps,
		WithRDPExecutor(mockExec),
		WithRDPAutoLoginUserProvider(func() string { return "" }),
	).Run(context.Background())

	if result.Status != checks.StatusCritical {
		t.Errorf("Status = %v, want Critical when no graphical session exists", result.Status)
	}
	if result.Details["sessionAvailable"] != false {
		t.Errorf("sessionAvailable = %v, want false", result.Details["sessionAvailable"])
	}
	if !strings.Contains(result.Message, "no graphical session") {
		t.Errorf("Message should name the missing graphical session, got: %s", result.Message)
	}
	if strings.Contains(strings.ToLower(result.Message), "display manager") {
		t.Errorf("infra-rdp must not restate display-manager health, got: %s", result.Message)
	}
}

// TestRDPCheckActionAvailability covers recovery-action availability across every
// combination of credential model and credential state.
//
// The two invariants: a restart is never offered as the remedy for a credential
// fault (it cannot repair one), and automated credential repair is offered only
// on the system-service model.
// [REQ:HEAL-ACTION-001]
func TestRDPCheckActionAvailability(t *testing.T) {
	tests := []struct {
		name            string
		model           CredentialModel
		state           CredentialState
		daemonRunning   bool
		wantRepair      bool
		wantRaise       bool
		wantRestart     bool
		wantStartOffere bool
	}{
		{
			name:  "system model with empty credentials offers automated repair",
			model: CredentialModelSystem, state: CredentialStateEmpty, daemonRunning: true,
			wantRepair: true, wantRaise: false, wantRestart: false,
		},
		{
			name:  "system model with unreadable credentials offers automated repair",
			model: CredentialModelSystem, state: CredentialStateUnreadable, daemonRunning: true,
			wantRepair: true, wantRaise: false, wantRestart: false,
		},
		{
			name:  "user-session model with empty credentials reports instead of repairing",
			model: CredentialModelUserSession, state: CredentialStateEmpty, daemonRunning: true,
			wantRepair: false, wantRaise: true, wantRestart: false,
		},
		{
			name:  "user-session model with unreadable credentials reports instead of repairing",
			model: CredentialModelUserSession, state: CredentialStateUnreadable, daemonRunning: true,
			wantRepair: false, wantRaise: true, wantRestart: false,
		},
		{
			name:  "healthy credentials offer neither repair nor report",
			model: CredentialModelUserSession, state: CredentialStatePresent, daemonRunning: true,
			wantRepair: false, wantRaise: false, wantRestart: false,
		},
		{
			name:  "a stopped daemon offers start and restart",
			model: CredentialModelUserSession, state: CredentialStatePresent, daemonRunning: false,
			wantRepair: false, wantRaise: false, wantRestart: true, wantStartOffere: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := &platform.Capabilities{Platform: platform.Linux, SupportsSystemd: true}
			mockExec := testutil.NewMockExecutor()
			mockExec.Responses["grdctl status"] = testutil.MockResponse{Output: []byte("RDP:\n\tStatus: enabled\n")}
			check := NewRDPCheck(caps, WithRDPExecutor(mockExec))

			status := "inactive"
			if tt.daemonRunning {
				status = "active"
			}
			lastResult := &checks.Result{Details: map[string]interface{}{
				"status":          status,
				"credentialModel": string(tt.model),
				"credentialState": string(tt.state),
			}}

			actions := make(map[string]checks.RecoveryAction)
			for _, a := range check.RecoveryActions(lastResult) {
				actions[a.ID] = a
				if a.Dangerous {
					t.Errorf("infra-rdp action %q must not be dangerous; dangerous actions belong to infra-display", a.ID)
				}
			}

			if got := actions["repair-credentials"].Available; got != tt.wantRepair {
				t.Errorf("repair-credentials available = %v, want %v", got, tt.wantRepair)
			}
			if got := actions["raise-incident"].Available; got != tt.wantRaise {
				t.Errorf("raise-incident available = %v, want %v", got, tt.wantRaise)
			}
			if got := actions["restart"].Available; got != tt.wantRestart {
				t.Errorf("restart available = %v, want %v", got, tt.wantRestart)
			}
			if got := actions["start"].Available; got != tt.wantStartOffere {
				t.Errorf("start available = %v, want %v", got, tt.wantStartOffere)
			}
		})
	}
}

// TestRepairCredentialsRefusesUserSessionModel proves autoheal never repairs a
// credential fault on the keyring model. Doing so would require holding a secret
// it must not hold, or minting a remote-access password on its own initiative.
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestRepairCredentialsRefusesUserSessionModel(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux, SupportsSystemd: true}
	mockExec := testutil.NewMockExecutor()
	mockExec.Responses["grdctl status"] = testutil.MockResponse{Output: []byte("RDP:\n\tStatus: enabled\n")}
	// No system-level unit: this is the user-session model.
	mockExec.Responses["systemctl is-enabled gnome-remote-desktop.service"] = testutil.MockResponse{
		Error: testutil.ErrConnectionRefused,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "repair-credentials")

	if result.Success {
		t.Error("repair-credentials must refuse the user-session model")
	}
	if result.Error == "" {
		t.Error("refusal must set an error")
	}
	for _, call := range mockExec.Calls {
		for _, arg := range call.Args {
			if strings.Contains(arg, "set-credentials") {
				t.Fatalf("must never set credentials on the user-session model, got: %s %v", call.Name, call.Args)
			}
		}
	}
	if !strings.Contains(result.Output, "autologin") {
		t.Errorf("refusal must carry the operator remedy, got: %s", result.Output)
	}
}

// TestRaiseIncidentIsNonMutating proves the report action changes nothing on the
// host. The durable incident record comes from the check's non-OK result through
// the incident pipeline, which also auto-resolves it on recovery.
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestRaiseIncidentIsNonMutating(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux, SupportsSystemd: true}
	mockExec := testutil.NewMockExecutor()
	mockExec.Responses["grdctl status"] = testutil.MockResponse{Output: []byte("RDP:\n\tStatus: enabled\n")}
	mockSessionBus(mockExec, "alice", "1000")
	mockSessionGrdctl(mockExec, "1000", "RDP:\n\tStatus: enabled\n\tUsername: (empty)\n\tPassword: (empty)\n")
	mockLoginKeyring(mockExec, "1000", false)

	check := NewRDPCheck(caps,
		WithRDPExecutor(mockExec),
		WithRDPAutoLoginUserProvider(func() string { return "matthalloran8" }),
	)
	result := check.ExecuteAction(context.Background(), "raise-incident")

	if !result.Success {
		t.Errorf("raise-incident should succeed, got error: %s", result.Error)
	}

	mutating := []string{"restart", "start", "stop", "set-credentials", "rm"}
	for _, call := range mockExec.Calls {
		joined := call.Name + " " + strings.Join(call.Args, " ")
		for _, verb := range mutating {
			if strings.Contains(joined, verb) {
				t.Errorf("raise-incident must not mutate host state, got: %s", joined)
			}
		}
		if call.Name == "systemctl" && len(call.Args) > 0 && (call.Args[0] == "enable" || call.Args[0] == "disable" || call.Args[0] == "restart" || call.Args[0] == "start" || call.Args[0] == "stop") {
			t.Errorf("raise-incident must not mutate host state, got: %s", joined)
		}
	}

	// The report must name all four items the operator needs.
	for _, want := range []string{"Credential state", "autologin", "keyring", "Operator remedies"} {
		if !strings.Contains(result.Output, want) {
			t.Errorf("report must name %q, got: %s", want, result.Output)
		}
	}
}

// TestRDPCheckNeverRequestsCredentialValues proves the check never asks grdctl to
// reveal secrets and never records one. Autoheal must not hold remote-access
// credentials.
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
func TestRDPCheckNeverRequestsCredentialValues(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux, SupportsSystemd: true}
	mockExec := testutil.NewMockExecutor()
	mockExec.Responses["grdctl status"] = testutil.MockResponse{Output: []byte("RDP:\n\tStatus: enabled\n")}
	mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = testutil.MockResponse{Output: []byte("12345")}
	mockSessionBus(mockExec, "alice", "1000")
	mockSessionGrdctl(mockExec, "1000", "RDP:\n\tStatus: enabled\n\tUsername: alice\n\tPassword: sup3rs3cret\n")

	result := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" })).Run(context.Background())

	for _, call := range mockExec.Calls {
		for _, arg := range call.Args {
			if strings.Contains(arg, "--show-credentials") {
				t.Fatalf("check must never pass --show-credentials, got: %s %v", call.Name, call.Args)
			}
		}
	}

	for key, value := range result.Details {
		if str, ok := value.(string); ok && strings.Contains(str, "sup3rs3cret") {
			t.Errorf("credential value leaked into Details[%q]", key)
		}
	}
	if strings.Contains(result.Message, "sup3rs3cret") {
		t.Error("credential value leaked into the result message")
	}
}

// TestRDPCheckNoRDPServiceStaysOK proves the credential probe never runs before
// the checkable gate. A headless host with no RDP service installed must not be
// alarmed by this work.
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
func TestRDPCheckNoRDPServiceStaysOK(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux, SupportsSystemd: true}
	mockExec := testutil.NewMockExecutor()
	// Neither GNOME RDP nor xrdp is present.
	mockExec.Responses["grdctl status"] = testutil.MockResponse{Error: testutil.ErrConnectionRefused}
	mockExec.Responses["systemctl list-unit-files xrdp.service"] = testutil.MockResponse{
		Output: []byte("0 unit files listed.\n"),
	}

	result := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" })).Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want OK for a host with no RDP service", result.Status)
	}
	if _, present := result.Details["credentialState"]; present {
		t.Error("credential probe must not run on a host with no RDP service")
	}
	for _, call := range mockExec.Calls {
		if call.Name == "env" {
			t.Errorf("session-bus probe must not run before the checkable gate, got: %s %v", call.Name, call.Args)
		}
	}
}
