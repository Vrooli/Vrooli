package vroolicli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/agentscope"
	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

type countingStopper struct{ calls int }

func (s *countingStopper) Stop(context.Context, agentsession.Session) error {
	s.calls++
	return nil
}

// A dry run lists exactly the sessions whose owner is confirmed gone, with
// evidence, keeps the operator's web-console terminal and the live Agent
// Manager run, and never reaches the platform adapter.
func TestAgentReapDryRunListsOnlyOrphansAndStopsNothing(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	old := now.Add(-21 * 24 * time.Hour)
	leases := []scenarioruntime.EditorLease{
		{
			SessionID: "desktop", Agent: "opencode", PID: 1, Scope: "vrooli-agent-desktop", CreatedAt: old, LastHeartbeatAt: old,
			Owner: scenarioruntime.EditorLeaseOwner{Kind: agentsession.KindDesktopApp, Ref: "/data", Source: "launch"},
		},
		{
			SessionID: "operator-ui", Agent: "claude", PID: 2, Scope: "vrooli-agent-ui", CreatedAt: old, LastHeartbeatAt: old,
			Owner: scenarioruntime.EditorLeaseOwner{Kind: agentsession.KindWebConsoleSession, Ref: "wc-ui", Source: "launch", EvidenceJSON: `{"web_console_origin":"ui"}`},
		},
		{
			SessionID: "am-live", Agent: "codex", PID: 3, Scope: "vrooli-agent-am", CreatedAt: old, LastHeartbeatAt: old,
			Owner: scenarioruntime.EditorLeaseOwner{Kind: agentsession.KindAgentManagerRun, Ref: "run-live", Source: "launch"},
		},
		{
			SessionID: "shell", Agent: "claude", PID: 4, Scope: "vrooli-agent-shell", CreatedAt: old, LastHeartbeatAt: old,
			Owner: scenarioruntime.EditorLeaseOwner{Kind: agentsession.KindTerminal, Ref: "/dev/pts/2", Source: "launch"},
		},
	}
	census := func() ([]agentscope.Entry, error) {
		return []agentscope.Entry{{Name: "vrooli-agent-desktop.scope"}, {Name: "vrooli-agent-ui.scope"}, {Name: "vrooli-agent-am.scope"}, {Name: "vrooli-agent-shell.scope"}}, nil
	}
	reconciler := agentsession.Reconciler{
		Census: census, Alive: alwaysAlive, Frozen: frozenUnknown, Now: fixedClock(now),
		Probes: agentsession.Probes{
			DesktopAppProcesses: func(string) ([]int, error) { return nil, nil },
			WebConsoleSession: func(context.Context, string, string) (agentsession.WebConsoleSession, error) {
				return agentsession.WebConsoleSession{}, agentsession.ErrNotFound
			},
			RunStatus: func(context.Context, string) (string, error) { return "RUN_STATUS_RUNNING", nil },
		},
	}
	stopper := &countingStopper{}
	var out bytes.Buffer
	err := runAgentReapWith(context.Background(), &out, func() ([]scenarioruntime.EditorLease, error) { return leases, nil },
		reconciler, agentsession.Executor{Stopper: stopper}, agentsession.ReapPolicy{}, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if stopper.calls != 0 {
		t.Fatalf("a dry run called the stopper %d times", stopper.calls)
	}
	var report AgentReapReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("decode %s: %v", out.String(), err)
	}
	outcomes := map[string]string{}
	for _, row := range report.Rows {
		outcomes[row.Session] = row.Outcome
	}
	want := map[string]string{"desktop": agentsession.OutcomeWouldStop, "operator-ui": agentsession.OutcomeKept}
	if len(outcomes) != len(want) || outcomes["desktop"] != want["desktop"] || outcomes["operator-ui"] != want["operator-ui"] {
		t.Fatalf("orphaned rows = %v, want %v (the live run and the terminal are not orphaned)", outcomes, want)
	}
	if !report.DryRun || report.Stop != 1 || report.Kept != 1 || report.Sessions != 4 {
		t.Fatalf("report = %+v", report)
	}
}
