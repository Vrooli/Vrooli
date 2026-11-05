package agentsession

import (
	"encoding/base64"
	"errors"
	"testing"
)

// identityToken builds an Agent Manager identity token the way Agent Manager
// does: base64url(claims) "." base64url(mac). The mac is not verified here.
func identityToken(claims string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(claims)) + "." + base64.RawURLEncoding.EncodeToString([]byte("mac"))
}

func TestInferReadsEachOwnerKindFromItsOwnEnvironment(t *testing.T) {
	cases := []struct {
		name     string
		env      map[string]string
		tty      string
		wantKind string
		wantRef  string
	}{
		{
			name: "web-console pane",
			env: map[string]string{
				"WC_WEB_CONSOLE_SESSION_ID": "f69811c1-0000-4000-8000-000000000001",
				"WC_SESSION_STATE_ROOT":     "/home/op/.vrooli/state/vrooli/web-console/sessions",
				"TMUX":                      "/home/op/.vrooli/state/vrooli/web-console/sessions/tmux/wc.sock,1234,0",
			},
			tty:      "/dev/pts/7",
			wantKind: KindWebConsoleSession, wantRef: "f69811c1-0000-4000-8000-000000000001",
		},
		{
			name: "bundled desktop app helper",
			env: map[string]string{
				"VROOLI_DESKTOP_MODE":      "true",
				"VROOLI_LIFECYCLE_MANAGED": "true",
				"VROOLI_DATA":              "/home/op/.config/web-console-desktop/runtime",
				"VROOLI_AGENT_SESSION":     "vrooli-agent-opencode-1",
			},
			wantKind: KindDesktopApp, wantRef: "/home/op/.config/web-console-desktop/runtime",
		},
		{
			name:     "agent manager headless run",
			env:      map[string]string{"VROOLI_AGENT_IDENTITY_TOKEN": identityToken(`{"run_id":"7867dda8-1111-4222-8333-444455556666","profile":"x"}`)},
			wantKind: KindAgentManagerRun, wantRef: "7867dda8-1111-4222-8333-444455556666",
		},
		{
			name:     "agent manager run home",
			env:      map[string]string{"CODEX_HOME": "/home/op/.vrooli/state/vrooli/agent-manager/runs/9AC0098D-1111-4222-8333-444455556666/codex", "CODEX_AGENT_TAG": "bas/E28"},
			wantKind: KindAgentManagerRun, wantRef: "9ac0098d-1111-4222-8333-444455556666",
		},
		{
			name:     "agent manager tag without a run id",
			env:      map[string]string{"CLAUDE_CODE_AGENT_TAG": "bas/E28"},
			wantKind: KindAgentManagerRun, wantRef: "",
		},
		{
			name:     "operator terminal",
			env:      map[string]string{"TERM": "xterm-256color"},
			tty:      "/dev/pts/3",
			wantKind: KindTerminal, wantRef: "/dev/pts/3",
		},
		{
			name:     "operator tmux without a tty",
			env:      map[string]string{"TMUX": "/tmp/tmux-1000/default,999,1"},
			wantKind: KindTerminal, wantRef: "tmux:/tmp/tmux-1000/default",
		},
		{
			name:     "no evidence",
			env:      map[string]string{"PATH": "/usr/bin"},
			wantKind: KindUnknown, wantRef: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Infer(tc.env, tc.tty)
			if got.Kind != tc.wantKind || got.Ref != tc.wantRef {
				t.Fatalf("Infer = %s/%q, want %s/%q", got.Kind, got.Ref, tc.wantKind, tc.wantRef)
			}
		})
	}
}

// An operator terminal inside a desktop build of web-console carries the
// desktop markers too. It belongs to its web-console session, which outlives
// the app, so the app's exit must not orphan it.
func TestInferPrefersTheWebConsoleSessionOverDesktopMarkers(t *testing.T) {
	got := Infer(map[string]string{
		"WC_WEB_CONSOLE_SESSION_ID": "wc-1",
		"VROOLI_DESKTOP_MODE":       "true",
		"VROOLI_DATA":               "/home/op/.config/web-console-desktop/runtime",
	}, "/dev/pts/1")
	if got.Kind != KindWebConsoleSession || got.Ref != "wc-1" || got.Evidence.DesktopData == "" {
		t.Fatalf("owner = %+v, want the web-console session with the desktop data kept as evidence", got)
	}
}

// The tmux socket and state root are recorded so liveness can tell which
// web-console instance to ask.
func TestInferRecordsWebConsoleInstanceEvidence(t *testing.T) {
	got := Infer(map[string]string{
		"WC_WEB_CONSOLE_SESSION_ID": "wc-1",
		"WC_SESSION_STATE_ROOT":     "/s/root/",
		"TMUX":                      "/s/root/tmux/wc.sock,1,0",
		"CODEX_AGENT_TAG":           "plan/F1",
	}, "")
	if got.Evidence.WebConsoleStateRoot != "/s/root" || got.Evidence.TmuxSocket != "/s/root/tmux/wc.sock" || got.Evidence.AgentTag != "plan/F1" {
		t.Fatalf("evidence = %+v", got.Evidence)
	}
}

// A token whose payload is not a claims document yields no run id rather
// than a guess.
func TestInferIgnoresAMalformedIdentityToken(t *testing.T) {
	got := Infer(map[string]string{"VROOLI_AGENT_IDENTITY_TOKEN": "not-a-token"}, "")
	if got.Kind != KindUnknown {
		t.Fatalf("owner = %+v, want unknown", got)
	}
}

// fakeProcesses is a process table for ancestry walks.
type fakeProcesses struct {
	facts map[int]ProcessFact
	envs  map[int]map[string]string
}

func (f fakeProcesses) process(pid int) (ProcessFact, error) {
	fact, ok := f.facts[pid]
	if !ok {
		return ProcessFact{}, ErrProcessGone
	}
	return fact, nil
}

func (f fakeProcesses) environment(pid int) (map[string]string, error) {
	env, ok := f.envs[pid]
	if !ok {
		return nil, errors.New("unreadable")
	}
	return env, nil
}

// The app identity is the topmost ancestor still carrying the same
// VROOLI_DATA: the bundled runtime, not the service or shim between it and
// the launcher, and not Electron, which does not carry the marker itself.
func TestDesktopAppRootIsTheTopmostAncestorOfTheSameApp(t *testing.T) {
	data := "/home/op/.config/web-console-desktop/runtime"
	app := map[string]string{"VROOLI_DESKTOP_MODE": "true", "VROOLI_DATA": data}
	other := map[string]string{"VROOLI_DESKTOP_MODE": "true", "VROOLI_DATA": "/home/op/.config/other-desktop/runtime"}
	table := fakeProcesses{
		facts: map[int]ProcessFact{
			100: {PID: 100, PPID: 90, StartTime: "1000"}, // the launcher
			90:  {PID: 90, PPID: 80, StartTime: "900"},   // web-console-api in the bundle
			80:  {PID: 80, PPID: 70, StartTime: "800"},   // bundled runtime
			70:  {PID: 70, PPID: 1, StartTime: "700"},    // Electron
			60:  {PID: 60, PPID: 1, StartTime: "600"},
		},
		envs: map[int]map[string]string{100: app, 90: app, 80: app, 70: {"HOME": "/home/op"}, 60: other},
	}
	root, ok := DesktopAppRoot(100, data, table.process, table.environment)
	if !ok || root.PID != 80 || root.StartTime != "800" {
		t.Fatalf("DesktopAppRoot = %+v, %v; want the bundled runtime pid 80", root, ok)
	}
	// A launch whose parent is already init (re-parented) has no ancestry.
	table.facts[100] = ProcessFact{PID: 100, PPID: 1, StartTime: "1000"}
	if _, ok := DesktopAppRoot(100, data, table.process, table.environment); ok {
		t.Fatal("a re-parented launch must report no app ancestor")
	}
}

// The launcher records the desktop app's runtime pid and start time, which is
// what lets the session become orphaned when the app exits.
func TestOwnerAtLaunchBindsADesktopSessionToItsApp(t *testing.T) {
	data := "/home/op/.config/web-console-desktop/runtime"
	app := map[string]string{"VROOLI_DESKTOP_MODE": "true", "VROOLI_DATA": data}
	host := &fakeHost{
		facts: map[int]ProcessFact{100: {PID: 100, PPID: 80, StartTime: "1000"}, 80: {PID: 80, PPID: 70, StartTime: "800"}, 70: {PID: 70, PPID: 1}},
		envs:  map[int]map[string]string{80: app, 70: {}},
	}
	owner := OwnerAtLaunch(app, 100, host)
	if owner.Kind != KindDesktopApp || owner.Ref != data || owner.Source != SourceLaunch {
		t.Fatalf("owner = %+v", owner)
	}
	if owner.Evidence.AppPID != 80 || owner.Evidence.AppStartTime != "800" || owner.Evidence.ReadFromPID != 100 {
		t.Fatalf("app identity = %+v, want pid 80 started at 800", owner.Evidence)
	}
}
