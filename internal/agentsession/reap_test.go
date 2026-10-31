package agentsession

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func orphan(id string, owner Owner, age time.Duration) Session {
	return Session{
		SessionID: id, Scope: "vrooli-agent-" + id, Leased: true, KernelState: StateLive, ScopeHeld: true,
		Owner: owner, Liveness: Liveness{State: OwnerGone, Detail: "gone"}, StartedAt: now.Add(-age), State: StateOrphaned, Frozen: "no",
	}
}

func decisions(plan []Candidate) map[string]string {
	out := map[string]string{}
	for _, c := range plan {
		out[c.Session.SessionID] = c.Decision
	}
	return out
}

func TestPlanStopsOnlyOrphanedSessionsTheOwnerPolicyReleases(t *testing.T) {
	day := 24 * time.Hour
	desktop := Owner{Kind: KindDesktopApp, Ref: "/data"}
	run := Owner{Kind: KindAgentManagerRun, Ref: "run-1"}
	frozen := orphan("frozen", desktop, day)
	frozen.Frozen = "yes"
	live := orphan("live", desktop, day)
	live.State, live.Liveness = StateLive, Liveness{State: OwnerAlive}
	idle := orphan("idle", Owner{Kind: KindTerminal}, 30*day)
	idle.State, idle.Liveness = StateIdle, Liveness{State: OwnerNotApplicable}
	agentless := orphan("agentless", Owner{Kind: KindUnknown}, day)
	agentless.State, agentless.Liveness = StateAgentless, Liveness{State: OwnerNotApplicable}
	noScope := orphan("noscope", desktop, day)
	noScope.Scope = "none"
	sessions := []Session{
		orphan("desktop", desktop, day),
		orphan("run", run, day),
		orphan("young", run, 10*time.Minute),
		orphan("wc-programmatic", Owner{Kind: KindWebConsoleSession, Ref: "a", Evidence: Evidence{WebConsoleOrigin: OriginProgrammatic}}, day),
		orphan("wc-am-tagged", Owner{Kind: KindWebConsoleSession, Ref: "b", Evidence: Evidence{AgentTag: "bas/E28"}}, day),
		orphan("wc-ui", Owner{Kind: KindWebConsoleSession, Ref: "c", Evidence: Evidence{WebConsoleOrigin: OriginUI, AgentTag: "bas/E28"}}, day),
		orphan("wc-unspecified", Owner{Kind: KindWebConsoleSession, Ref: "d", Evidence: Evidence{WebConsoleOrigin: OriginUnspecified}}, day),
		orphan("wc-unobserved", Owner{Kind: KindWebConsoleSession, Ref: "e"}, day),
		frozen, live, idle, agentless, noScope,
	}
	got := decisions(Plan(sessions, ReapPolicy{OlderThan: time.Hour}, now))
	want := map[string]string{
		"desktop": DecisionStop, "run": DecisionStop, "wc-programmatic": DecisionStop, "wc-am-tagged": DecisionStop,
		"young": DecisionKeep, "wc-ui": DecisionKeep, "wc-unspecified": DecisionKeep, "wc-unobserved": DecisionKeep,
		"frozen": DecisionKeep, "noscope": DecisionKeep,
	}
	if len(got) != len(want) {
		t.Fatalf("plan = %v, want exactly %v (live, idle and agentless are never candidates)", got, want)
	}
	for id, decision := range want {
		if got[id] != decision {
			t.Errorf("%s: decision %q, want %q", id, got[id], decision)
		}
	}
}

// The promised default: a session younger than an hour is never reaped.
func TestDefaultReapOlderThanIsOneHour(t *testing.T) {
	if DefaultReapOlderThan != time.Hour {
		t.Fatalf("DefaultReapOlderThan = %s, want 1h", DefaultReapOlderThan)
	}
	got := decisions(Plan([]Session{orphan("a", Owner{Kind: KindDesktopApp, Ref: "/d"}, 59*time.Minute)}, ReapPolicy{}, now))
	if got["a"] != DecisionKeep {
		t.Fatalf("a 59-minute-old orphan under the default policy = %q, want keep", got["a"])
	}
}

type recordingStopper struct {
	stopped []string
	err     error
}

func (s *recordingStopper) Stop(_ context.Context, session Session) error {
	s.stopped = append(s.stopped, session.ScopeUnit())
	return s.err
}

type recordingAudit struct {
	reaps []scenarioruntime.AgentSessionReap
}

func (a *recordingAudit) RecordAgentSessionReap(_ context.Context, reap scenarioruntime.AgentSessionReap) (scenarioruntime.AgentSessionReap, error) {
	reap.ReapID = fmt.Sprintf("reap-%d", len(a.reaps)+1)
	a.reaps = append(a.reaps, reap)
	return reap, nil
}

func goneAgain(context.Context, Session) Liveness {
	return Liveness{State: OwnerGone, Detail: "still gone"}
}

// measured is a session's usage as the reconciler measured it.
func measured(session Session) Session {
	rss, swap := int64(512<<20), int64(1<<30)
	session.Usage = ScopeUsage{MemoryBytes: &rss, MemoryStatus: UsageMeasured, SwapBytes: &swap, SwapStatus: UsageMeasured}
	session.Tasks = 9
	return session
}

func TestExecuteDryRunNeverCallsTheAdapterOrTheAudit(t *testing.T) {
	stopper, audit := &recordingStopper{}, &recordingAudit{}
	rechecks := 0
	executor := Executor{
		Stopper: stopper, Audit: audit,
		Recheck: func(context.Context, Session) Liveness { rechecks++; return Liveness{State: OwnerGone} },
	}
	plan := Plan([]Session{measured(orphan("a", Owner{Kind: KindDesktopApp, Ref: "/d"}, 48*time.Hour))}, ReapPolicy{}, now)
	results := executor.Execute(context.Background(), plan, true)
	if len(results) != 1 || results[0].Outcome != OutcomeWouldStop || results[0].Usage.MemoryBytes == nil || *results[0].Usage.MemoryBytes != 512<<20 {
		t.Fatalf("results = %+v", results)
	}
	if len(stopper.stopped) != 0 || len(audit.reaps) != 0 || rechecks != 0 {
		t.Fatalf("dry run stopped %v, audited %d, rechecked %d", stopper.stopped, len(audit.reaps), rechecks)
	}
}

func TestExecuteStopsEligibleSessionsAndAuditsEach(t *testing.T) {
	stopper, audit := &recordingStopper{}, &recordingAudit{}
	executor := Executor{Stopper: stopper, Audit: audit, Recheck: goneAgain, Initiator: "test", Now: func() time.Time { return now }}
	owner := Owner{Kind: KindDesktopApp, Ref: "/d", Evidence: Evidence{DesktopData: "/d", AppPID: 80, AppStartTime: "800"}}
	sessions := []Session{
		measured(orphan("eligible", owner, 48*time.Hour)),
		orphan("young", owner, time.Minute),
	}
	results := executor.Execute(context.Background(), Plan(sessions, ReapPolicy{}, now), false)
	if len(stopper.stopped) != 1 || stopper.stopped[0] != "vrooli-agent-eligible.scope" {
		t.Fatalf("stopped %v, want only the eligible session's scope", stopper.stopped)
	}
	if len(audit.reaps) != 1 {
		t.Fatalf("audits = %+v, want one", audit.reaps)
	}
	reap := audit.reaps[0]
	if reap.SessionID != "eligible" || reap.OwnerKind != KindDesktopApp || reap.OwnerRef != "/d" || reap.Outcome != OutcomeStopped ||
		reap.MemoryBytes != 512<<20 || reap.SwapBytes != 1<<30 || reap.Tasks != 9 || reap.LivenessDetail != "still gone" || reap.Initiator != "test" {
		t.Fatalf("audit = %+v", reap)
	}
	evidence, _ := OwnerFromLease(scenarioruntime.EditorLeaseOwner{Kind: reap.OwnerKind, EvidenceJSON: reap.OwnerEvidenceJSON})
	if evidence.Evidence.AppPID != 80 {
		t.Fatalf("audit evidence = %s, want the app identity", reap.OwnerEvidenceJSON)
	}
	outcomes := map[string]string{}
	for _, r := range results {
		outcomes[r.Candidate.Session.SessionID] = r.Outcome
	}
	if outcomes["eligible"] != OutcomeStopped || outcomes["young"] != OutcomeKept {
		t.Fatalf("outcomes = %v", outcomes)
	}
}

// An owner that comes back between the plan and the stop wins.
func TestExecuteSkipsASessionWhoseOwnerReturned(t *testing.T) {
	stopper, audit := &recordingStopper{}, &recordingAudit{}
	executor := Executor{Stopper: stopper, Audit: audit, Recheck: func(context.Context, Session) Liveness { return Liveness{State: OwnerAlive} }}
	plan := Plan([]Session{orphan("a", Owner{Kind: KindAgentManagerRun, Ref: "r"}, 48*time.Hour)}, ReapPolicy{}, now)
	results := executor.Execute(context.Background(), plan, false)
	if results[0].Outcome != OutcomeOwnerReturns || len(stopper.stopped) != 0 || len(audit.reaps) != 0 {
		t.Fatalf("result %+v, stopped %v, audits %d", results[0], stopper.stopped, len(audit.reaps))
	}
}

// A platform without a stop primitive says so, and the audit records it.
func TestExecuteReportsAnUnsupportedPlatform(t *testing.T) {
	stopper, audit := &recordingStopper{err: fmt.Errorf("stop: %w", platformgo.ErrUnsupported)}, &recordingAudit{}
	executor := Executor{Stopper: stopper, Audit: audit, Recheck: goneAgain}
	plan := Plan([]Session{orphan("a", Owner{Kind: KindAgentManagerRun, Ref: "r"}, 48*time.Hour)}, ReapPolicy{}, now)
	results := executor.Execute(context.Background(), plan, false)
	if results[0].Outcome != OutcomeUnsupported || len(audit.reaps) != 1 || audit.reaps[0].Outcome != OutcomeUnsupported {
		t.Fatalf("result %+v, audits %+v", results[0], audit.reaps)
	}
	if !errors.Is(stopper.err, platformgo.ErrUnsupported) {
		t.Fatal("fixture error lost its unsupported identity")
	}
}

// An audit row never records an unknown figure as 0.
func TestExecuteAuditsUnknownUsageAsMinusOne(t *testing.T) {
	audit := &recordingAudit{}
	executor := Executor{Stopper: &recordingStopper{}, Audit: audit, Recheck: goneAgain}
	session := orphan("a", Owner{Kind: KindAgentManagerRun, Ref: "r"}, 48*time.Hour)
	session.Usage = ScopeUsage{MemoryStatus: UsageUnsupported, SwapStatus: UsageUnsupported}
	executor.Execute(context.Background(), Plan([]Session{session}, ReapPolicy{}, now), false)
	if len(audit.reaps) != 1 || audit.reaps[0].MemoryBytes != -1 || audit.reaps[0].SwapBytes != -1 {
		t.Fatalf("audits = %+v, want memory and swap -1 (unknown)", audit.reaps)
	}
}

// The list's per-row verdict is the planner's: a non-orphaned session says
// why it is not reapable, and an orphaned one gets the same reason Plan gives.
func TestReapDecisionExplainsEveryState(t *testing.T) {
	live := orphan("live", Owner{Kind: KindDesktopApp, Ref: "/d"}, 48*time.Hour)
	live.State = StateLive
	if ok, reason := ReapDecision(live, ReapPolicy{}, now); ok || reason != "state live: only orphaned sessions are reaped" {
		t.Fatalf("live = %v %q", ok, reason)
	}
	ui := orphan("ui", Owner{Kind: KindWebConsoleSession, Ref: "w", Evidence: Evidence{WebConsoleOrigin: OriginUI}}, 48*time.Hour)
	ok, reason := ReapDecision(ui, ReapPolicy{}, now)
	plan := Plan([]Session{ui}, ReapPolicy{}, now)
	if ok || plan[0].Decision != DecisionKeep || plan[0].Reason != reason || reason != "web-console session origin ui: an operator terminal is never auto-reaped" {
		t.Fatalf("ui orphan = %v %q, plan %+v", ok, reason, plan[0])
	}
}
