package runs

import (
	"context"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/runmanager"
	sharedruns "test-genie/internal/shared/runs"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// rpcFakeExecutor drives a controllable run for the lifecycle RPC tests.
type rpcFakeExecutor struct {
	scenarioDir string
	blockOnCtx  bool
	release     chan struct{}
	startedOnce sync.Once
	started     chan struct{}
	result      *orchestrator.SuiteExecutionResult
}

func newRPCFake(scenarioDir string) *rpcFakeExecutor {
	return &rpcFakeExecutor{
		scenarioDir: scenarioDir,
		release:     make(chan struct{}),
		started:     make(chan struct{}),
		result:      &orchestrator.SuiteExecutionResult{ScenarioName: "demo", Success: true, Verdict: "PASS", CompletedAt: time.Now().UTC()},
	}
}

func (f *rpcFakeExecutor) ExecuteWithEvents(ctx context.Context, input execution.SuiteExecutionInput, emit orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error) {
	_ = sharedruns.NewIndex(f.scenarioDir).Append(sharedruns.RunRecord{
		RunID: input.Request.RunID, Scenario: input.Request.ScenarioName, StartedAt: time.Now().UTC(), Status: sharedruns.StatusInProgress,
	})
	f.startedOnce.Do(func() { close(f.started) })
	if f.blockOnCtx {
		<-ctx.Done()
	} else {
		<-f.release
	}
	_ = sharedruns.NewIndex(f.scenarioDir).Update(input.Request.RunID, func(r *sharedruns.RunRecord) error {
		r.Status = sharedruns.StatusPassed
		r.CompletedAt = time.Now().UTC()
		return nil
	})
	return f.result, nil
}

func TestLifecycleRPC_StartWaitStatus(t *testing.T) {
	root := t.TempDir()
	fake := newRPCFake(root + "/demo")
	fake.result.Phases = []phases.ExecutionResult{{
		Name: "architecture",
		MaturityStanding: &runspb.PhaseMaturityStanding{
			Provider:             "architecture-health",
			Phase:                "architecture",
			CurrentLevel:         "L2",
			NextLevel:            "L3",
			BlockingFindingCodes: []string{"arch.primitive_unverified"},
			NextMove:             "Prove each command primitive.",
		},
		FindingsSummary: &runspb.PhaseFindingsSummary{Errors: 1, Total: 1},
	}}
	svc := NewService(root, runmanager.New(fake, root), nil, nil)
	ctx := context.Background()

	start, err := svc.StartRun(ctx, connect.NewRequest(&runspb.StartRunRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	runID := start.Msg.GetRunId()
	if runID == "" {
		t.Fatal("StartRun returned empty run id")
	}
	<-fake.started

	st, err := svc.GetRunStatus(ctx, connect.NewRequest(&runspb.GetRunStatusRequest{Scenario: "demo", RunId: runID}))
	if err != nil {
		t.Fatalf("GetRunStatus: %v", err)
	}
	if st.Msg.GetStatus() != sharedruns.StatusInProgress {
		t.Fatalf("status = %q, want in_progress", st.Msg.GetStatus())
	}
	if !st.Msg.GetActive() {
		t.Fatal("expected active=true for a live run")
	}

	// Wait with a short timeout returns a non-terminal snapshot (run continues).
	wr, err := svc.WaitRun(ctx, connect.NewRequest(&runspb.WaitRunRequest{Scenario: "demo", RunId: runID, TimeoutSeconds: 1}))
	if err != nil {
		t.Fatalf("WaitRun(timeout): %v", err)
	}
	if !wr.Msg.GetTimedOut() {
		t.Fatal("expected timed_out=true on short wait")
	}
	if wr.Msg.GetStatus().GetStatus() != sharedruns.StatusInProgress {
		t.Fatalf("timed-out wait status = %q, want in_progress", wr.Msg.GetStatus().GetStatus())
	}

	close(fake.release)

	wr2, err := svc.WaitRun(ctx, connect.NewRequest(&runspb.WaitRunRequest{Scenario: "demo", RunId: runID}))
	if err != nil {
		t.Fatalf("WaitRun(terminal): %v", err)
	}
	if wr2.Msg.GetTimedOut() {
		t.Fatal("expected timed_out=false on terminal wait")
	}
	if wr2.Msg.GetStatus().GetStatus() != sharedruns.StatusPassed {
		t.Fatalf("terminal wait status = %q, want passed", wr2.Msg.GetStatus().GetStatus())
	}
	standings := wr2.Msg.GetStatus().GetTerminalStandings()
	if len(standings) != 1 {
		t.Fatalf("terminal standings = %d, want 1", len(standings))
	}
	if got := standings[0].GetBlockingFindingCodes(); len(got) != 1 || got[0] != "arch.primitive_unverified" {
		t.Fatalf("terminal standing blocking codes = %v", got)
	}
	summaries := wr2.Msg.GetStatus().GetTerminalFindingsSummaries()
	if len(summaries) != 1 || summaries[0].GetErrors() != 1 {
		t.Fatalf("terminal findings summaries = %+v", summaries)
	}
}

func TestLifecycleRPC_Abort(t *testing.T) {
	root := t.TempDir()
	fake := newRPCFake(root + "/demo")
	fake.blockOnCtx = true
	fake.result = &orchestrator.SuiteExecutionResult{ScenarioName: "demo", Success: false, Verdict: "FAIL", CompletedAt: time.Now().UTC()}
	svc := NewService(root, runmanager.New(fake, root), nil, nil)
	ctx := context.Background()

	start, err := svc.StartRun(ctx, connect.NewRequest(&runspb.StartRunRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	runID := start.Msg.GetRunId()
	<-fake.started

	ab, err := svc.AbortRun(ctx, connect.NewRequest(&runspb.AbortRunRequest{Scenario: "demo", RunId: runID}))
	if err != nil {
		t.Fatalf("AbortRun: %v", err)
	}
	if ab.Msg.GetStatus().GetStatus() != sharedruns.StatusAborted {
		t.Fatalf("abort status = %q, want aborted", ab.Msg.GetStatus().GetStatus())
	}
}

// TestKeepFollowEvent proves the per-follower heartbeat filter drops ONLY
// phase_heartbeat events (and only when suppression is on), never phase
// transitions or the terminal run_completed.
func TestKeepFollowEvent(t *testing.T) {
	cases := []struct {
		kind     string
		suppress bool
		want     bool
	}{
		{runmanager.EventPhaseHeartbeat, true, false}, // dropped when suppressing
		{runmanager.EventPhaseHeartbeat, false, true}, // kept for an interactive follower
		{runmanager.EventPhaseStarted, true, true},    // phase transitions always survive
		{runmanager.EventPhaseCompleted, true, true},  // ...
		{runmanager.EventPhaseFailed, true, true},     // ...
		{runmanager.EventRunCompleted, true, true},    // the verdict always survives
		{runmanager.EventPhaseProgress, true, true},   // progress always survives
	}
	for _, tc := range cases {
		if got := keepFollowEvent(runmanager.Event{Kind: tc.kind}, tc.suppress); got != tc.want {
			t.Errorf("keepFollowEvent(%q, suppress=%v) = %v, want %v", tc.kind, tc.suppress, got, tc.want)
		}
	}
}

func TestLifecycleRPC_StartRejectsEmptyScenario(t *testing.T) {
	svc := NewService(t.TempDir(), runmanager.New(newRPCFake(t.TempDir()), t.TempDir()), nil, nil)
	if _, err := svc.StartRun(context.Background(), connect.NewRequest(&runspb.StartRunRequest{})); err == nil {
		t.Fatal("expected error for empty scenario")
	}
}
