package heartbeat

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// Crash-window harness for heartbeat dispatch (runtime Phase 4, R28/R29).
//
// The phase validation names two process-death windows that must not allow a
// duplicate heartbeat:
//
//  1. crash before dispatch: Execute created the owner task and persisted the
//     dispatch intent (BeginDispatch), then died before the start request left
//     the process. The owner has no run.
//  2. crash after accepted start, before run-ID persistence: the owner accepted
//     the start request, but the process died before SetRunningRunID recorded
//     the run identity.
//
// Both windows leave the identical durable state (a dispatch_uncertain running
// entry carrying the owner idempotency key and task). They differ only in
// whether the owner already holds a run for that key. This harness runs a fresh
// Recover over that durable state against an idempotency-aware fake owner and
// proves the recovery converges on exactly one accepted live obligation.

// idempotentRunOwner is a fake owner that deduplicates run creation by the
// request idempotency key, mirroring agent-manager's contract: a replayed start
// whose key was already accepted returns the same run instead of starting a
// second one.
type idempotentRunOwner struct {
	*mockAgentClient
	idemMu    sync.Mutex
	runsByKey map[string]*Run
	created   int
}

func newIdempotentRunOwner() *idempotentRunOwner {
	return &idempotentRunOwner{
		mockAgentClient: newMockAgentClient(),
		runsByKey:       make(map[string]*Run),
	}
}

// seedAccepted models an owner that accepted a start request before the caller
// learned the run identity: the keyed run already exists and a replay must
// return it rather than create another.
func (o *idempotentRunOwner) seedAccepted(key, runID string) {
	o.idemMu.Lock()
	defer o.idemMu.Unlock()
	o.runsByKey[key] = &Run{ID: runID, TaskID: "task-1", Status: "RUN_STATUS_RUNNING"}
}

func (o *idempotentRunOwner) CreateRun(_ context.Context, req *CreateRunRequest) (*Run, error) {
	o.idemMu.Lock()
	defer o.idemMu.Unlock()
	o.createRunCalls = append(o.createRunCalls, req)
	if run, ok := o.runsByKey[req.IdempotencyKey]; ok {
		return run, nil
	}
	o.created++
	run := &Run{
		ID:     fmt.Sprintf("run-%d", o.created),
		TaskID: req.TaskID,
		Status: "RUN_STATUS_RUNNING",
	}
	o.runsByKey[req.IdempotencyKey] = run
	return run, nil
}

func (o *idempotentRunOwner) acceptedRuns() int {
	o.idemMu.Lock()
	defer o.idemMu.Unlock()
	return len(o.runsByKey)
}

func (o *idempotentRunOwner) newRunsCreated() int {
	o.idemMu.Lock()
	defer o.idemMu.Unlock()
	return o.created
}

// seedCrashDurableState writes the durable queue file exactly as Execute leaves
// it when the process dies mid-dispatch: one dispatch_uncertain running entry
// carrying the owner idempotency key, task and tag, with no RunID.
func seedCrashDurableState(t *testing.T, dir, key string) {
	t.Helper()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{
				AgentID:        "agent-1",
				ProfileKey:     "profile-p",
				State:          ObligationDispatchUncertain,
				IdempotencyKey: key,
				TaskID:         "task-1",
				RunTag:         "heartbeat-team-x-agent-1",
			},
		},
	})
}

// TestCrashWindow_BeforeDispatchReplaysToExactlyOneObligation covers window 1:
// the intent is durable but no request reached the owner. Recovery replays the
// idempotent start, creates the intended run, refuses a duplicate heartbeat
// while the obligation holds the slot, and reopens capacity once the owner
// reports terminal state.
func TestCrashWindow_BeforeDispatchReplaysToExactlyOneObligation(t *testing.T) {
	dir := t.TempDir()
	key := "prompt-manager-heartbeat:team-x:agent-1:attempt-1"
	seedCrashDurableState(t, dir, key)

	owner := newIdempotentRunOwner()
	exec := &watcherExecutor{}
	tec := newTeamExecutionContext("team-x", exec, dir, owner)

	tec.Recover(context.Background())

	if got := len(owner.createRunCalls); got != 1 {
		t.Fatalf("expected exactly one recovery replay, got %d", got)
	}
	if got := owner.createRunCalls[0].IdempotencyKey; got != key {
		t.Fatalf("expected recorded idempotency key replayed, got %q", got)
	}
	if got := owner.acceptedRuns(); got != 1 {
		t.Fatalf("expected exactly one accepted live obligation, got %d", got)
	}

	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" {
		t.Fatalf("expected recovered obligation retained, got %+v", status)
	}
	if len(status.UncertainAgentIDs) != 0 {
		t.Fatalf("expected uncertainty cleared after replay, got %+v", status)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].RunID != "run-1" {
		t.Fatalf("expected recovered RunID persisted, got %+v", persisted.Running)
	}
	if len(exec.watched) != 1 || exec.watched[0].runID != "run-1" {
		t.Fatalf("expected completion waiter re-attached to the recovered run, got %+v", exec.watched)
	}

	// The bound obligation holds its slot: the next tick cannot admit a
	// duplicate heartbeat for the same member.
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-p"); err == nil || !IsMemberAlreadyQueued(err) {
		t.Fatalf("expected duplicate heartbeat refused while obligation is live, got %v", err)
	}

	// Definitive owner terminal state releases the slot. This is the path the
	// re-attached waiter drives via OnComplete.
	tec.OnMemberComplete("agent-1")
	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("expected terminal release to clear the obligation, got %+v", tec.Status())
	}
	result, err := tec.Enqueue(context.Background(), "agent-2", "profile-p")
	if err != nil || result.Status != "started" {
		t.Fatalf("expected capacity reopened after terminal release, got result=%+v err=%v", result, err)
	}
}

// TestCrashWindow_AfterAcceptedStartConvergesOnAcceptedRun covers window 2: the
// owner already holds the accepted run, but the caller never recorded its ID.
// Recovery replays the same key and must converge on the existing run rather
// than start a second, so there is exactly one accepted live obligation.
func TestCrashWindow_AfterAcceptedStartConvergesOnAcceptedRun(t *testing.T) {
	dir := t.TempDir()
	key := "prompt-manager-heartbeat:team-x:agent-1:attempt-1"
	seedCrashDurableState(t, dir, key)

	owner := newIdempotentRunOwner()
	owner.seedAccepted(key, "run-accepted")
	exec := &watcherExecutor{}
	tec := newTeamExecutionContext("team-x", exec, dir, owner)

	tec.Recover(context.Background())

	if got := len(owner.createRunCalls); got != 1 {
		t.Fatalf("expected exactly one recovery replay, got %d", got)
	}
	if got := owner.createRunCalls[0].IdempotencyKey; got != key {
		t.Fatalf("expected recorded idempotency key replayed, got %q", got)
	}
	if got := owner.newRunsCreated(); got != 0 {
		t.Fatalf("expected the replay to deduplicate, not create a second run, got %d new runs", got)
	}
	if got := owner.acceptedRuns(); got != 1 {
		t.Fatalf("expected exactly one accepted live obligation, got %d", got)
	}

	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" {
		t.Fatalf("expected recovered obligation bound to the accepted run, got %+v", status)
	}
	if len(status.UncertainAgentIDs) != 0 {
		t.Fatalf("expected uncertainty cleared after converging replay, got %+v", status)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].RunID != "run-accepted" {
		t.Fatalf("expected the accepted RunID bound, got %+v", persisted.Running)
	}
	if len(exec.watched) != 1 || exec.watched[0].runID != "run-accepted" {
		t.Fatalf("expected completion waiter re-attached to the accepted run, got %+v", exec.watched)
	}
}
