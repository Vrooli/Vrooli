// Package infra tests for RDP health check
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
package infra

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

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

	mockExec := checks.NewMockExecutor()
	// GNOME Remote Desktop is configured (detected via grdctl status)
	mockExec.Responses["grdctl status"] = checks.MockResponse{
		Output: []byte("RDP:\n\tStatus: enabled\n\tPort: 3389"),
		Error:  nil,
	}
	// GNOME Remote Desktop daemon is running
	mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = checks.MockResponse{
		Output: []byte("12345"),
		Error:  nil,
	}
	mockExec.Responses["pgrep -a -f gnome-remote-desktop-daemon"] = checks.MockResponse{
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

// mockSessionBus wires the loginctl and id lookups that sessionBusEnv performs.
func mockSessionBus(m *checks.MockExecutor, user, uid string) {
	m.Responses["loginctl show-seat seat0 -p ActiveSession --value"] = checks.MockResponse{Output: []byte("2\n")}
	m.Responses["loginctl show-session 2 -p Name --value"] = checks.MockResponse{Output: []byte(user + "\n")}
	m.Responses["id -u "+user] = checks.MockResponse{Output: []byte(uid + "\n")}
	m.Responses["pgrep -u "+user+" gnome-shell"] = checks.MockResponse{Output: []byte("4242\n")}
}

// sessionGrdctlKey builds the mock key for a grdctl probe carrying the session
// bus environment.
func sessionGrdctlKey(uid string) string {
	return "env XDG_RUNTIME_DIR=/run/user/" + uid +
		" DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/" + uid + "/bus grdctl status"
}

// mockSessionGrdctl wires the environment-carrying grdctl probe.
func mockSessionGrdctl(m *checks.MockExecutor, uid, output string) {
	m.Responses[sessionGrdctlKey(uid)] = checks.MockResponse{Output: []byte(output)}
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
			mockExec := checks.NewMockExecutor()
			mockExec.Responses["grdctl status"] = checks.MockResponse{
				Output: []byte("RDP:\n\tStatus: enabled\n\tPort: 3389"),
			}
			mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = checks.MockResponse{
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
func mockDenialJournal(m *checks.MockExecutor, messages []string, err error) {
	lines := make([]string, 0, len(messages))
	for _, msg := range messages {
		lines = append(lines, journalEntry(msg))
	}
	m.Responses[denialJournalKey()] = checks.MockResponse{
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
			journalErr:   checks.ErrConnectionRefused,
			wantStatus:   checks.StatusCritical,
			wantReadable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := &platform.Capabilities{Platform: platform.Linux, SupportsSystemd: true}
			mockExec := checks.NewMockExecutor()
			mockExec.Responses["grdctl status"] = checks.MockResponse{
				Output: []byte("RDP:\n\tStatus: enabled\n\tPort: 3389"),
			}
			mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = checks.MockResponse{
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
func mockLoginKeyring(m *checks.MockExecutor, uid string, present bool) {
	reply := "(<[objectpath '/org/freedesktop/secrets/collection/session']>,)\n"
	if present {
		reply = "(<[objectpath '/org/freedesktop/secrets/collection/login'," +
			" objectpath '/org/freedesktop/secrets/collection/session']>,)\n"
	}
	m.Responses[keyringCollectionsKey(uid)] = checks.MockResponse{Output: []byte(reply)}
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
			mockExec := checks.NewMockExecutor()
			mockExec.Responses["grdctl status"] = checks.MockResponse{
				Output: []byte("RDP:\n\tStatus: enabled\n\tPort: 3389"),
			}
			mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = checks.MockResponse{
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
	mockExec := checks.NewMockExecutor()
	mockExec.Responses["grdctl status"] = checks.MockResponse{
		Output: []byte("RDP:\n\tStatus: enabled\n\tPort: 3389"),
	}
	mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = checks.MockResponse{Output: []byte("12345")}
	mockSessionBus(mockExec, "alice", "1000")
	mockSessionGrdctl(mockExec, "1000", "RDP:\n\tStatus: enabled\n\tUsername: alice\n\tPassword: hunter2\n")
	// No gnome-shell for the session owner.
	mockExec.Responses["pgrep -u alice gnome-shell"] = checks.MockResponse{Error: checks.ErrConnectionRefused}

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
			mockExec := checks.NewMockExecutor()
			mockExec.Responses["grdctl status"] = checks.MockResponse{Output: []byte("RDP:\n\tStatus: enabled\n")}
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
	mockExec := checks.NewMockExecutor()
	mockExec.Responses["grdctl status"] = checks.MockResponse{Output: []byte("RDP:\n\tStatus: enabled\n")}
	// No system-level unit: this is the user-session model.
	mockExec.Responses["systemctl is-enabled gnome-remote-desktop.service"] = checks.MockResponse{
		Error: checks.ErrConnectionRefused,
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
	mockExec := checks.NewMockExecutor()
	mockExec.Responses["grdctl status"] = checks.MockResponse{Output: []byte("RDP:\n\tStatus: enabled\n")}
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

	mutating := []string{"restart", "start", "stop", "set-credentials", "rm", "systemctl"}
	for _, call := range mockExec.Calls {
		joined := call.Name + " " + strings.Join(call.Args, " ")
		for _, verb := range mutating {
			if strings.Contains(joined, verb) {
				t.Errorf("raise-incident must not mutate host state, got: %s", joined)
			}
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
	mockExec := checks.NewMockExecutor()
	mockExec.Responses["grdctl status"] = checks.MockResponse{Output: []byte("RDP:\n\tStatus: enabled\n")}
	mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = checks.MockResponse{Output: []byte("12345")}
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
	mockExec := checks.NewMockExecutor()
	// Neither GNOME RDP nor xrdp is present.
	mockExec.Responses["grdctl status"] = checks.MockResponse{Error: checks.ErrConnectionRefused}
	mockExec.Responses["systemctl list-unit-files xrdp.service"] = checks.MockResponse{
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

// TestRDPCheckRunWithMock_GnomeRDPConfiguredButNotRunning tests detection when GNOME RDP is
// configured but the daemon has crashed or been stopped
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
func TestRDPCheckRunWithMock_GnomeRDPConfiguredButNotRunning(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	// GNOME Remote Desktop IS configured (grdctl shows enabled)
	mockExec.Responses["grdctl status"] = checks.MockResponse{
		Output: []byte("RDP:\n\tStatus: enabled\n\tPort: 3389"),
		Error:  nil,
	}
	// But the daemon is NOT running
	mockExec.Responses["pgrep -f gnome-remote-desktop-daemon"] = checks.MockResponse{
		Output: []byte(""),
		Error:  checks.ErrConnectionRefused,
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

	mockExec := checks.NewMockExecutor()
	// GNOME Remote Desktop is NOT configured (grdctl shows disabled or fails)
	mockExec.Responses["grdctl status"] = checks.MockResponse{
		Output: []byte("RDP:\n\tStatus: disabled"),
		Error:  nil,
	}
	// No xrdp either
	mockExec.Responses["systemctl list-unit-files xrdp.service"] = checks.MockResponse{
		Output: []byte(""),
		Error:  checks.ErrConnectionRefused,
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

	mockExec := checks.NewMockExecutor()
	// GNOME Remote Desktop is NOT configured
	mockExec.Responses["grdctl status"] = checks.MockResponse{
		Output: []byte("RDP:\n\tStatus: disabled"),
		Error:  nil,
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

	mockExec := checks.NewMockExecutor()
	// GNOME Remote Desktop is NOT configured
	mockExec.Responses["grdctl status"] = checks.MockResponse{
		Output: []byte("RDP:\n\tStatus: disabled"),
		Error:  nil,
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

	mockExec := checks.NewMockExecutor()
	// GNOME Remote Desktop is NOT configured (grdctl fails or shows disabled)
	mockExec.Responses["grdctl status"] = checks.MockResponse{
		Output: []byte(""),
		Error:  checks.ErrCommandNotFound,
	}
	// xrdp is NOT installed
	mockExec.Responses["systemctl list-unit-files xrdp.service"] = checks.MockResponse{
		Output: []byte(""),
		Error:  checks.ErrConnectionRefused,
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

	mockExec := checks.NewMockExecutor()
	// GNOME Remote Desktop is NOT configured
	mockExec.Responses["grdctl status"] = checks.MockResponse{
		Output: []byte(""),
		Error:  checks.ErrCommandNotFound,
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

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["sc query TermService"] = checks.MockResponse{
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

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["sc query TermService"] = checks.MockResponse{
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

	mockExec := checks.NewMockExecutor()
	mockExec.Responses["sc query TermService"] = checks.MockResponse{
		Output: []byte(""),
		Error:  checks.ErrCommandNotFound,
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

	mockExec := checks.NewMockExecutor()
	check := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" }))
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
// The new detection order: 1) grdctl status for GNOME RDP config, 2) systemctl for xrdp
func TestRDPCheckMockCallsVerified(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	mockExec := checks.NewMockExecutor()
	// GNOME RDP not configured, xrdp is installed and active
	mockExec.Responses["grdctl status"] = checks.MockResponse{
		Output: []byte("RDP:\n\tStatus: disabled"),
		Error:  nil,
	}
	mockExec.Responses["systemctl list-unit-files xrdp.service"] = checks.MockResponse{
		Output: []byte("xrdp.service enabled"),
		Error:  nil,
	}
	mockExec.Responses["systemctl is-active xrdp"] = checks.MockResponse{
		Output: []byte("active"),
		Error:  nil,
	}

	check := NewRDPCheck(caps, WithRDPExecutor(mockExec), WithRDPAutoLoginUserProvider(func() string { return "" }))
	check.Run(context.Background())

	// Verify the mock was called with expected sequence:
	// 1. grdctl status (detect GNOME RDP configuration)
	// 2. systemctl list-unit-files xrdp.service (detect xrdp installed)
	// 3. systemctl is-active xrdp (check xrdp status)
	if len(mockExec.Calls) < 1 {
		t.Errorf("Expected at least 1 call, got %d", len(mockExec.Calls))
		return
	}

	// First call should be grdctl for GNOME RDP configuration detection
	firstCall := mockExec.Calls[0]
	if firstCall.Name != "grdctl" {
		t.Errorf("Expected first command 'grdctl', got %q", firstCall.Name)
	}
	if len(firstCall.Args) < 1 || firstCall.Args[0] != "status" {
		t.Errorf("Expected args [status], got %v", firstCall.Args)
	}
}
