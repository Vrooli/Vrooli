package vroolicli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
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

func TestAgentReapTotalsKeepMeasurementSourcesSeparate(t *testing.T) {
	cgroupRAM, procRAM := int64(100), int64(200)
	procSwap, cgroupSwap := int64(30), int64(40)
	var totals AgentReapTotals
	totals.add(AgentReapRow{
		MemoryBytes: &cgroupRAM, MemoryStatus: agentsession.UsageMeasured, MemorySource: agentsession.UsageSourceCgroupV2,
		SwapBytes: &procSwap, SwapStatus: agentsession.UsageMeasured, SwapSource: agentsession.UsageSourceProcSum, Tasks: 2,
	})
	totals.add(AgentReapRow{
		MemoryBytes: &procRAM, MemoryStatus: agentsession.UsageMeasured, MemorySource: agentsession.UsageSourceProcSum,
		SwapBytes: &cgroupSwap, SwapStatus: agentsession.UsageMeasured, SwapSource: agentsession.UsageSourceCgroupV2, Tasks: 3,
	})
	if totals.MemoryBytes != 300 || totals.SwapBytes != 70 || totals.Tasks != 5 ||
		totals.MemoryBytesBySource[agentsession.UsageSourceCgroupV2] != cgroupRAM ||
		totals.MemoryBytesBySource[agentsession.UsageSourceProcSum] != procRAM ||
		totals.SwapBytesBySource[agentsession.UsageSourceProcSum] != procSwap ||
		totals.SwapBytesBySource[agentsession.UsageSourceCgroupV2] != cgroupSwap {
		t.Fatalf("source-separated totals = %+v", totals)
	}
}

func TestAgentReapTotalsSeparatePartialAndUnknownUsage(t *testing.T) {
	measured, partial := int64(100), int64(40)
	var totals AgentReapTotals
	totals.add(AgentReapRow{MemoryBytes: &measured, MemoryStatus: agentsession.UsageMeasured, MemorySource: agentsession.UsageSourceCgroupV2, SwapBytes: &measured, SwapStatus: agentsession.UsageMeasured})
	totals.add(AgentReapRow{MemoryBytes: &partial, MemoryStatus: agentsession.UsagePartial, MemorySource: agentsession.UsageSourceProcSum, SwapBytes: &partial, SwapStatus: agentsession.UsagePartial, SwapSource: agentsession.UsageSourceProcSum})
	totals.add(AgentReapRow{})
	if totals.MemoryBytes != measured || totals.MemoryPartial != 1 || totals.MemoryPartialBytes != partial || totals.MemoryUnknown != 1 ||
		totals.SwapBytes != measured || totals.SwapPartial != 1 || totals.SwapPartialBytes != partial || totals.SwapUnknown != 1 ||
		totals.MemoryPartialBytesBySource[agentsession.UsageSourceProcSum] != partial || totals.SwapPartialBytesBySource[agentsession.UsageSourceProcSum] != partial {
		t.Fatalf("partial/complete totals = %+v", totals)
	}
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
	if report.MemoryMeaning == "" {
		t.Fatalf("report does not explain that its memory figure is pre-action scope usage: %+v", report)
	}
}

type exactReapFactsFixture struct{}

func (exactReapFactsFixture) ScopeCgroupPath(string) (string, error) {
	return "/user.slice/user-1000.slice/vrooli.slice/vrooli-agents.slice/vrooli-agent-old.scope", nil
}
func (exactReapFactsFixture) ScopeMembers(string) ([]int, error) { return []int{19}, nil }
func (exactReapFactsFixture) Process(pid int) (agentsession.ProcessFact, error) {
	return agentsession.ProcessFact{PID: pid, StartTime: "start-19"}, nil
}
func (exactReapFactsFixture) ProcessName(int) (string, error) { return "mock-api", nil }

func TestExactAgentReapDryRunPrintsProofBoundDecisionAndNeverStops(t *testing.T) {
	now := time.Date(2026, 11, 16, 1, 0, 0, 0, time.UTC)
	leases := []scenarioruntime.EditorLease{{
		SessionID: "web-session-1", Agent: "claude", PID: 19, Scope: "vrooli-agent-old.scope", CreatedAt: now.Add(-24 * time.Hour),
		Owner: scenarioruntime.EditorLeaseOwner{Kind: agentsession.KindWebConsoleSession, Ref: "web-session-1", Source: agentsession.SourceLaunch, EvidenceJSON: `{"web_console_state_root":"/tmp/wc","web_console_session_id":"web-session-1"}`},
	}}
	reconciler := agentsession.Reconciler{
		Census: func() ([]agentscope.Entry, error) { return []agentscope.Entry{{Name: "vrooli-agent-old.scope"}}, nil },
		Alive:  alwaysAlive, Frozen: func(string) string { return "no" }, Now: fixedClock(now),
		Probes: agentsession.Probes{WebConsoleSession: func(context.Context, string, string) (agentsession.WebConsoleSession, error) {
			return agentsession.WebConsoleSession{}, agentsession.ErrNotFound
		}},
	}
	stopper := &countingStopper{}
	var output bytes.Buffer
	err := runExactAgentReapWith(context.Background(), &output,
		func() ([]scenarioruntime.EditorLease, error) { return leases, nil }, reconciler,
		"vrooli-agent-old.scope", true, true, exactReapFactsFixture{}, nil, stopper)
	if err != nil {
		t.Fatal(err)
	}
	if stopper.calls != 0 {
		t.Fatalf("dry-run called stopper %d times", stopper.calls)
	}
	var report exactAgentReapReport
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatalf("decode %s: %v", output.String(), err)
	}
	if !report.DryRun || report.Outcome != "approval-required" || report.Scope != "vrooli-agent-old.scope" || report.ProofDigest == "" ||
		report.AuthorizationTarget != "agent-session:reap/vrooli-agent-old.scope/proof/"+report.ProofDigest ||
		!strings.Contains(report.AuthorizationCommand, "--policy agent-session-reaping --target "+report.AuthorizationTarget) || report.ReclaimedBytes != 0 {
		t.Fatalf("exact dry-run did not produce an exact, non-reclaiming decision: %+v", report)
	}
}
