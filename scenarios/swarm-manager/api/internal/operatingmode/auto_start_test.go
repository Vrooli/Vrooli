package operatingmode

import (
	"context"
	"strings"
	"testing"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
)

// driveHolisticLoopToReviewCompletion stages an initiative through
// investigate / plan / execute as completed rounds, starts the review
// phase, and returns the live review round (status=AgentRunning, with
// run-1 wired into the fakeAgent state map ready for completion).
func driveHolisticLoopToReviewCompletion(t *testing.T, svc *Service) RoundEnvelope {
	t.Helper()
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "investigate", nil)
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "plan", nil)
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "execute", map[string]any{"replan_needed": false})
	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "review",
	})
	if err != nil {
		t.Fatalf("StartPhase(review): %v", err)
	}
	return round
}

// completeReviewRunOnAgent installs the agent run state that drives the
// fakeAgent.GetRunState path: a complete status with a structured
// operating_mode_result envelope carrying a verdict and a backlog_sync
// proposal so applyPhaseResult does not reject the round.
func completeReviewRunOnAgent(t *testing.T, agent *fakeAgent, runID string) {
	t.Helper()
	agent.states[runID] = agentmanager.RunState{
		RunID: runID, Status: "complete",
		Summary:    `{"operating_mode_result":{"verdict":"accepted","backlog_sync":{"proposal":{"form":"mutation_list","mutations":[{"id":"m1","op":"add_item","item":{"kind":"fix","name":"x","title":"X"}}]}}}}`,
		FinishedAt: "2026-05-02T12:05:00Z",
	}
}

// TestAutoStartAfter_HappyPath verifies that completing the review phase
// auto-spawns the reconcile phase. The fakeAgent records every spawn; the
// test asserts the second spawn carries the reconcile phase metadata.
func TestAutoStartAfter_HappyPath(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round := driveHolisticLoopToReviewCompletion(t, svc)
	if got := len(agent.spawned); got != 1 {
		t.Fatalf("after StartPhase(review) spawned = %d, want 1", got)
	}
	completeReviewRunOnAgent(t, agent, round.RunID)

	refreshed, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound: %v", err)
	}
	if refreshed.Status != RoundStatusCompleted {
		t.Fatalf("refreshed.Status = %q, want %q", refreshed.Status, RoundStatusCompleted)
	}
	if got := len(agent.spawned); got != 2 {
		t.Fatalf("after RefreshRound auto-dispatch spawned = %d, want 2", got)
	}
	last := agent.spawned[1]
	if !strings.Contains(last.Title, "reconcile") {
		t.Fatalf("auto-dispatched spawn title = %q, want it to mention reconcile", last.Title)
	}
	if last.Purpose != "holistic_loop_reconcile" {
		t.Fatalf("auto-dispatched spawn purpose = %q, want %q", last.Purpose, "holistic_loop_reconcile")
	}
	// Predecessor round should not carry a pending marker on the happy path.
	pred, err := svc.store.LoadRound("init-a", ModeHolisticLoop, refreshed.Round)
	if err != nil {
		t.Fatalf("LoadRound: %v", err)
	}
	if RoundPayload(pred.Payload).HasPendingAutoStart() {
		t.Fatalf("predecessor round has pending_auto_start marker on happy path: %+v", pred.Payload)
	}
}

// TestAutoStartAfter_SkipsOnFailure verifies that a failed review run does
// NOT auto-dispatch the reconcile phase. Reconcile reads round artifacts;
// a failed round produces nothing useful to reconcile against.
func TestAutoStartAfter_SkipsOnFailure(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round := driveHolisticLoopToReviewCompletion(t, svc)
	agent.states[round.RunID] = agentmanager.RunState{
		RunID: round.RunID, Status: "failed", ErrorMsg: "boom",
		FinishedAt: "2026-05-02T12:05:00Z",
	}

	refreshed, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound: %v", err)
	}
	if refreshed.Status != RoundStatusFailed {
		t.Fatalf("refreshed.Status = %q, want %q", refreshed.Status, RoundStatusFailed)
	}
	if got := len(agent.spawned); got != 1 {
		t.Fatalf("failed-round auto-dispatch spawned = %d, want 1 (no auto-start)", got)
	}
}

// TestAutoStartAfter_SkipsOnCancellation verifies that a cancelled review
// run does NOT auto-dispatch the reconcile phase. Same rationale as
// failure: an interrupted round has nothing reliable to reconcile.
func TestAutoStartAfter_SkipsOnCancellation(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round := driveHolisticLoopToReviewCompletion(t, svc)
	agent.states[round.RunID] = agentmanager.RunState{
		RunID: round.RunID, Status: "cancelled",
		FinishedAt: "2026-05-02T12:05:00Z",
	}

	refreshed, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound: %v", err)
	}
	if refreshed.Status != RoundStatusCanceled {
		t.Fatalf("refreshed.Status = %q, want %q", refreshed.Status, RoundStatusCanceled)
	}
	if got := len(agent.spawned); got != 1 {
		t.Fatalf("cancelled-round auto-dispatch spawned = %d, want 1 (no auto-start)", got)
	}
}

// TestAutoStartAfter_SkipsOnReviewChangesRequested verifies the review-reloop
// guard wins over the reconcile auto-start: when review completes with a
// changes_requested verdict, the guard routes back to execute, so reconcile
// must NOT auto-dispatch over the top of that branch. This is the auto-start
// gating fix — auto_start_after only fires when a guard actually routes there.
func TestAutoStartAfter_SkipsOnReviewChangesRequested(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round := driveHolisticLoopToReviewCompletion(t, svc)
	agent.states[round.RunID] = agentmanager.RunState{
		RunID: round.RunID, Status: "complete",
		Summary:    `{"operating_mode_result":{"verdict":"changes_requested","backlog_sync":{"proposal":{"form":"mutation_list","mutations":[]}}}}`,
		FinishedAt: "2026-05-02T12:05:00Z",
	}

	refreshed, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound: %v", err)
	}
	if refreshed.Status != RoundStatusCompleted {
		t.Fatalf("refreshed.Status = %q, want %q", refreshed.Status, RoundStatusCompleted)
	}
	// reconcile must not have spawned — the changes_requested guard routes to
	// execute, not reconcile.
	if got := len(agent.spawned); got != 1 {
		t.Fatalf("changes_requested auto-dispatch spawned = %d, want 1 (reconcile suppressed, guard routes to execute)", got)
	}
	// No pending marker: the skip is intentional routing, not a deferred spawn.
	pred, err := svc.store.LoadRound("init-a", ModeHolisticLoop, refreshed.Round)
	if err != nil {
		t.Fatalf("LoadRound: %v", err)
	}
	if RoundPayload(pred.Payload).HasPendingAutoStart() {
		t.Fatalf("predecessor carries pending_auto_start after intentional reloop skip: %+v", pred.Payload)
	}
}

// laneSaturatingActivity is a test double that satisfies the activity
// surface the operating-mode service uses for spawning. It returns
// ErrLaneSaturated for the *next* spawn after a configurable threshold;
// before that it delegates to the underlying fakeAgent so the predecessor
// review run still spawns normally.
type laneSaturatingActivity struct {
	agent     *fakeAgent
	failAfter int // spawn calls before this index succeed; at/after, ErrLaneSaturated
	calls     int
}

func (l *laneSaturatingActivity) SpawnInitiative(ctx context.Context, req agentmanager.InitiativeSpawnRequest) (agentmanager.RunResult, error) {
	l.calls++
	if l.calls > l.failAfter {
		return agentmanager.RunResult{}, agentactivity.ErrLaneSaturated
	}
	return l.agent.SpawnInitiative(ctx, req)
}

// TestAutoStartAfter_DefersOnLaneSaturation verifies the soft-failure
// contract: when the auto-dispatch hits ErrLaneSaturated, the predecessor
// round is marked with pending_auto_start, the round is still Completed,
// and the reconcile phase has NOT spawned.
func TestAutoStartAfter_DefersOnLaneSaturation(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	svc := newTestService(t, root, agent, &fakePrompts{})

	// Drive review through StartPhase normally (delegates to fakeAgent).
	round := driveHolisticLoopToReviewCompletion(t, svc)
	completeReviewRunOnAgent(t, agent, round.RunID)

	// Swap the spawning surface: from now on, every spawn returns
	// ErrLaneSaturated. The next StartPhase call (auto-dispatch) will hit it.
	svc.activity = &laneSaturatingActivity{agent: agent, failAfter: 0}

	refreshed, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound: %v", err)
	}
	if refreshed.Status != RoundStatusCompleted {
		t.Fatalf("refreshed.Status = %q, want %q (predecessor completes regardless of saturation)", refreshed.Status, RoundStatusCompleted)
	}
	if got := len(agent.spawned); got != 1 {
		t.Fatalf("lane-saturated auto-dispatch spawned = %d, want 1 (reconcile blocked)", got)
	}
	pred, err := svc.store.LoadRound("init-a", ModeHolisticLoop, refreshed.Round)
	if err != nil {
		t.Fatalf("LoadRound: %v", err)
	}
	if !RoundPayload(pred.Payload).HasPendingAutoStart() {
		t.Fatalf("predecessor missing pending_auto_start marker after saturation: %+v", pred.Payload)
	}
}

// TestAutoStartAfter_RetriesOnNextRefresh verifies the retry loop: after a
// saturation defer, the next RefreshRound on the same predecessor (now
// terminal) re-attempts the auto-dispatch. When capacity has freed, the
// retry succeeds, the marker clears, and the reconcile spawn lands.
func TestAutoStartAfter_RetriesOnNextRefresh(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round := driveHolisticLoopToReviewCompletion(t, svc)
	completeReviewRunOnAgent(t, agent, round.RunID)

	saturating := &laneSaturatingActivity{agent: agent, failAfter: 0}
	svc.activity = saturating

	refreshed, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound (defer phase): %v", err)
	}
	if !RoundPayload(refreshed.Payload).HasPendingAutoStart() {
		t.Fatalf("expected pending_auto_start after first refresh")
	}

	// Simulate capacity recovery: subsequent spawns succeed.
	saturating.failAfter = 1_000_000

	retried, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, refreshed.Round)
	if err != nil {
		t.Fatalf("RefreshRound (retry phase): %v", err)
	}
	if RoundPayload(retried.Payload).HasPendingAutoStart() {
		t.Fatalf("predecessor still has pending_auto_start after successful retry")
	}
	if got := len(agent.spawned); got != 2 {
		t.Fatalf("after retry spawned = %d, want 2 (review + reconcile)", got)
	}
	if agent.spawned[1].Purpose != "holistic_loop_reconcile" {
		t.Fatalf("retried spawn purpose = %q, want %q", agent.spawned[1].Purpose, "holistic_loop_reconcile")
	}
}

// TestAutoStartAfter_FiresAfterLockRelease pins the ordering invariant:
// the auto-dispatch starts the next phase AFTER the predecessor's lock has
// been released, so the new StartPhase doesn't deadlock against the
// initiative-exclusive lock the predecessor still holds. We assert this
// by confirming the second spawn succeeds (StartPhase would fail with a
// lock conflict if the predecessor's lock were still held).
func TestAutoStartAfter_FiresAfterLockRelease(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round := driveHolisticLoopToReviewCompletion(t, svc)
	completeReviewRunOnAgent(t, agent, round.RunID)

	if _, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round); err != nil {
		t.Fatalf("RefreshRound: %v", err)
	}
	// If the lock were still held by the review run, the auto-dispatch
	// would have surfaced the lock conflict. The fakeAgent.spawned[1]
	// existence is the proof — combined with no ErrLaneSaturated wrapping,
	// the lock release ordering is correct.
	if got := len(agent.spawned); got != 2 {
		t.Fatalf("auto-dispatch spawned = %d, want 2 (lock release ordering)", got)
	}
	// Sanity: the new round's RunID should be distinct from the predecessor.
	if agent.spawned[1].Name == "" {
		t.Fatalf("auto-dispatched spawn missing initiative name: %+v", agent.spawned[1])
	}
}
