package agentsession

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

var checkedAt = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

func TestScenarioInstanceRequiresAnExactCompletedStop(t *testing.T) {
	for _, tc := range []struct {
		name, id, status string
		stopped          bool
		err              error
		want             string
	}{
		{"completed stop", "inst-1", scenarioruntime.StatusStopped, true, nil, OwnerGone},
		{"running", "inst-1", scenarioruntime.StatusRunning, false, nil, OwnerAlive},
		{"starting", "inst-1", scenarioruntime.StatusStarting, false, nil, OwnerAlive},
		{"stop in progress", "inst-1", scenarioruntime.StatusStopping, false, nil, OwnerAlive},
		{"failed is not stopped", "inst-1", scenarioruntime.StatusFailed, false, nil, OwnerUnknown},
		{"expired is not stopped", "inst-1", scenarioruntime.StatusExpired, false, nil, OwnerUnknown},
		{"missing completion evidence", "inst-1", scenarioruntime.StatusStopped, false, nil, OwnerUnknown},
		{"different instance", "inst-other", scenarioruntime.StatusStopped, true, nil, OwnerUnknown},
		{"missing record", "", "", false, scenarioruntime.ErrNotFound, OwnerUnknown},
		{"registry unavailable", "", "", false, errors.New("unavailable"), OwnerUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			probe := Probes{ScenarioInstance: func(_ context.Context, id string) (scenarioruntime.Instance, error) {
				if id != "inst-1" {
					t.Fatalf("looked up %q", id)
				}
				instance := scenarioruntime.Instance{InstanceID: tc.id, Status: tc.status}
				if tc.stopped {
					instance.StoppedAt = &checkedAt
				}
				return instance, tc.err
			}}
			owner := Owner{Kind: KindScenarioInstance, Ref: "inst-1", Evidence: Evidence{RuntimeInstanceID: "inst-1"}}
			if got := Check(context.Background(), owner, probe, checkedAt); got.State != tc.want {
				t.Fatalf("liveness=%+v, want %s", got, tc.want)
			}
		})
	}
	for _, owner := range []Owner{
		{Kind: KindScenarioInstance},
		{Kind: KindScenarioInstance, Ref: "inst-1", Evidence: Evidence{RuntimeInstanceID: "inst-other"}},
	} {
		if got := Check(context.Background(), owner, Probes{}, checkedAt); got.State != OwnerUnknown {
			t.Fatalf("unproven identity=%+v", got)
		}
	}
}

func runProbe(status string, err error, calls *probeCalls) Probes {
	return Probes{RunStatus: func(context.Context, string) (string, error) {
		calls.inc(&calls.run)
		return status, err
	}}
}

func TestAgentManagerRunLiveness(t *testing.T) {
	owner := Owner{Kind: KindAgentManagerRun, Ref: "run-1"}
	cases := []struct {
		name   string
		status string
		err    error
		want   string
	}{
		{"running run is alive", "RUN_STATUS_RUNNING", nil, OwnerAlive},
		{"parked run is alive", "RUN_STATUS_PARKED", nil, OwnerAlive},
		{"completed run is gone", "RUN_STATUS_COMPLETE", nil, OwnerGone},
		{"cancelled run is gone", "RUN_STATUS_CANCELLED", nil, OwnerGone},
		{"run unknown to agent manager is gone", "", ErrNotFound, OwnerGone},
		{"agent manager unreachable is unknown", "", errors.New("connection refused"), OwnerUnknown},
		{"status unknown confirms nothing", "RUN_STATUS_UNKNOWN", nil, OwnerUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := &probeCalls{}
			got := Check(context.Background(), owner, runProbe(tc.status, tc.err, calls), checkedAt)
			if got.State != tc.want || calls.run != 1 || !got.CheckedAt.Equal(checkedAt) {
				t.Fatalf("Check = %+v (probe calls %d), want %s", got, calls.run, tc.want)
			}
		})
	}
}

// A tag names many runs over time; a run owner without an id is never asked
// and never judged gone.
func TestAgentManagerRunWithoutAnIDIsUnknownAndNotAsked(t *testing.T) {
	calls := &probeCalls{}
	got := Check(context.Background(), Owner{Kind: KindAgentManagerRun, Evidence: Evidence{AgentTag: "bas/E28"}}, runProbe("RUN_STATUS_COMPLETE", nil, calls), checkedAt)
	if got.State != OwnerUnknown || calls.run != 0 {
		t.Fatalf("Check = %+v, probe calls %d; want unknown and no call", got, calls.run)
	}
}

func TestWebConsoleSessionLiveness(t *testing.T) {
	owner := Owner{Kind: KindWebConsoleSession, Ref: "wc-1", Evidence: Evidence{WebConsoleStateRoot: "/root"}}
	probe := func(session WebConsoleSession, err error) Probes {
		return Probes{WebConsoleSession: func(_ context.Context, root, id string) (WebConsoleSession, error) {
			if root != "/root" || id != "wc-1" {
				t.Fatalf("probe asked about %s in %s", id, root)
			}
			return session, err
		}}
	}
	live := Check(context.Background(), owner, probe(WebConsoleSession{Origin: OriginProgrammatic, Owner: "agent-manager"}, nil), checkedAt)
	if live.State != OwnerAlive || live.Origin != OriginProgrammatic || live.WebConsoleOwner != "agent-manager" {
		t.Fatalf("live session = %+v", live)
	}
	if got := Check(context.Background(), owner, probe(WebConsoleSession{}, ErrNotFound), checkedAt); got.State != OwnerGone {
		t.Fatalf("archived session = %+v, want gone", got)
	}
	if got := Check(context.Background(), owner, probe(WebConsoleSession{}, errors.New("another instance")), checkedAt); got.State != OwnerUnknown {
		t.Fatalf("unanswerable session = %+v, want unknown", got)
	}
}

func TestDesktopAppLivenessByRecordedIdentity(t *testing.T) {
	owner := Owner{Kind: KindDesktopApp, Ref: "/data", Evidence: Evidence{AppPID: 80, AppStartTime: "800"}}
	probe := func(start string, err error) Probes {
		return Probes{ProcessStartTime: func(pid int) (string, error) {
			if pid != 80 {
				t.Fatalf("probe asked about pid %d", pid)
			}
			return start, err
		}}
	}
	if got := Check(context.Background(), owner, probe("800", nil), checkedAt); got.State != OwnerAlive {
		t.Fatalf("running app = %+v, want alive", got)
	}
	if got := Check(context.Background(), owner, probe("", ErrProcessGone), checkedAt); got.State != OwnerGone {
		t.Fatalf("exited app = %+v, want gone", got)
	}
	if got := Check(context.Background(), owner, probe("9999", nil), checkedAt); got.State != OwnerGone {
		t.Fatalf("reused pid = %+v, want gone", got)
	}
	if got := Check(context.Background(), owner, probe("", errors.New("permission denied")), checkedAt); got.State != OwnerUnknown {
		t.Fatalf("unreadable pid = %+v, want unknown", got)
	}
}

// A backfilled desktop owner has no app pid; the app is alive while any
// process outside the agent slice carries its data directory.
func TestDesktopAppLivenessByDataDirectory(t *testing.T) {
	owner := Owner{Kind: KindDesktopApp, Ref: "/data"}
	probe := func(pids []int, err error) Probes {
		return Probes{DesktopAppProcesses: func(dir string) ([]int, error) {
			if dir != "/data" {
				t.Fatalf("probe asked about %s", dir)
			}
			return pids, err
		}}
	}
	if got := Check(context.Background(), owner, probe([]int{80}, nil), checkedAt); got.State != OwnerAlive {
		t.Fatalf("app with processes = %+v, want alive", got)
	}
	if got := Check(context.Background(), owner, probe(nil, nil), checkedAt); got.State != OwnerGone {
		t.Fatalf("app with no processes = %+v, want gone", got)
	}
	if got := Check(context.Background(), owner, probe(nil, errors.New("unsupported")), checkedAt); got.State != OwnerUnknown {
		t.Fatalf("unreadable table = %+v, want unknown", got)
	}
}

// A terminal or unknown owner has no authority: it is never asked and never
// orphaned.
func TestTerminalAndUnknownOwnersAreNeverJudged(t *testing.T) {
	calls := &probeCalls{}
	probes := Probes{
		RunStatus: func(context.Context, string) (string, error) { calls.inc(&calls.run); return "", ErrNotFound },
		WebConsoleSession: func(context.Context, string, string) (WebConsoleSession, error) {
			calls.inc(&calls.console)
			return WebConsoleSession{}, ErrNotFound
		},
		ProcessStartTime:    func(int) (string, error) { calls.inc(&calls.start); return "", ErrProcessGone },
		DesktopAppProcesses: func(string) ([]int, error) { calls.inc(&calls.desktop); return nil, nil },
	}
	for _, kind := range []string{KindTerminal, KindUnknown} {
		got := Check(context.Background(), Owner{Kind: kind, Ref: "/dev/pts/1"}, probes, checkedAt)
		if got.State != OwnerNotApplicable || got.Alive() != nil {
			t.Fatalf("%s owner = %+v, want not-applicable with null alive", kind, got)
		}
	}
	if calls.run+calls.console+calls.start+calls.desktop != 0 {
		t.Fatalf("probes were called: %+v", calls)
	}
}

func TestHostWebConsoleRootAcceptsOnlyTheHostInstance(t *testing.T) {
	state := "/home/op/.vrooli/state"
	cases := map[string]bool{
		"/home/op/.vrooli/state/vrooli/web-console/sessions":                    true,
		"/home/op/.vrooli/state/vrooli/web-console/sessions/variants/presenter": false,
		"/home/op/.config/web-console-desktop/runtime/web-console-sessions":     false,
		"/home/op/.vrooli/state":              false,
		"":                                    false,
		"/home/op/.vrooli/state/../elsewhere": false,
	}
	for root, want := range cases {
		if got := HostWebConsoleRoot(root, state); got != want {
			t.Errorf("HostWebConsoleRoot(%q) = %v, want %v", root, got, want)
		}
	}
}
