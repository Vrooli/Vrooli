package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// writeQueueFile writes a persistedTeamQueue to disk for Recover tests.
func writeQueueFile(t *testing.T, dir, teamID string, q persistedTeamQueue) {
	t.Helper()
	path := filepath.Join(dir, "team-queue-"+teamID+".json")
	data, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		t.Fatalf("marshal queue: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write queue: %v", err)
	}
}

func readQueueFile(t *testing.T, dir, teamID string) persistedTeamQueue {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "team-queue-"+teamID+".json"))
	if err != nil {
		t.Fatalf("read queue: %v", err)
	}
	var q persistedTeamQueue
	if err := json.Unmarshal(data, &q); err != nil {
		t.Fatalf("unmarshal queue: %v", err)
	}
	return q
}

func TestRecover_DropsRunningWithTerminalRunID(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-terminal"},
		},
	})

	client := newMockAgentClient().
		WithGetRunResponse("run-terminal", &Run{ID: "run-terminal", Status: "RUN_STATUS_COMPLETE"})

	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.RunningAgentIDs) != 0 {
		t.Fatalf("expected terminal entry dropped, got running=%v", status.RunningAgentIDs)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 0 {
		t.Fatalf("expected disk-running cleared, got %v", persisted.Running)
	}
}

func TestRecover_RetainsUncertainRunningWithEmptyRunID(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: ""},
		},
	})

	client := newMockAgentClient()
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" {
		t.Fatalf("expected uncertain obligation retained, got running=%v", status.RunningAgentIDs)
	}
	if len(status.UncertainAgentIDs) != 1 || status.UncertainAgentIDs[0] != "agent-1" {
		t.Fatalf("expected uncertain classification, got %v", status.UncertainAgentIDs)
	}
	if len(client.getRunCalls) != 0 {
		t.Fatalf("expected no GetRun calls for empty RunID, got %v", client.getRunCalls)
	}
}

func TestRecover_RetainsUncertainRunningWhenOwnerReportsMissing(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-gone"},
		},
	})

	// (nil, nil) — owner reached, run not found. Absence of terminal evidence is
	// not terminal evidence, so the obligation is retained as uncertain.
	client := newMockAgentClient().WithGetRunResponse("run-gone", nil)
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 {
		t.Fatalf("expected missing-run obligation retained, got %v", status.RunningAgentIDs)
	}
	if len(status.UncertainAgentIDs) != 1 {
		t.Fatalf("expected uncertain classification, got %v", status.UncertainAgentIDs)
	}
}

func TestRecover_PreservesRunningWithActiveRun(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-active"},
		},
	})

	client := newMockAgentClient().
		WithGetRunResponse("run-active", &Run{ID: "run-active", Status: "RUN_STATUS_RUNNING"})

	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" {
		t.Fatalf("expected active entry preserved, got %v", status.RunningAgentIDs)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].RunID != "run-active" {
		t.Fatalf("expected RunID preserved on disk, got %+v", persisted.Running)
	}
}

func TestRecover_RetainsOwnerUnreachableOnAgentManagerError(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-err"},
		},
	})

	client := newMockAgentClient()
	client.getRunErr = errors.New("connection refused")
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 {
		t.Fatalf("expected owner-unreachable obligation retained, got %v", status.RunningAgentIDs)
	}
	if len(status.UnreachableAgentIDs) != 1 || status.UnreachableAgentIDs[0] != "agent-1" {
		t.Fatalf("expected owner_unreachable classification, got %v", status.UnreachableAgentIDs)
	}

	// The retained obligation must fence a duplicate heartbeat during the outage.
	if _, err := tec.Enqueue(context.Background(), "agent-1", "p"); err == nil || !IsMemberAlreadyQueued(err) {
		t.Fatalf("expected MemberAlreadyQueuedError while owner unreachable, got %v", err)
	}
}

func TestRecover_DropsStaleQueuedTicks(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-active"},
		},
		Queue: []queuedExecution{
			{AgentID: "agent-2", ProfileKey: "p"},
			{AgentID: "agent-3", ProfileKey: "p"},
		},
	})

	client := newMockAgentClient().
		WithGetRunResponse("run-active", &Run{ID: "run-active", Status: "RUN_STATUS_RUNNING"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.Queue) != 0 {
		t.Fatalf("expected stale queued ticks dropped, got %v", status.Queue)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Queue) != 0 {
		t.Fatalf("expected stale queue cleared on disk, got %v", persisted.Queue)
	}
	if len(status.RunningAgentIDs) != 1 {
		t.Fatalf("expected active obligation retained, got %v", status.RunningAgentIDs)
	}
}

func TestRecover_PreservesPausedParkedObligation(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-parked"},
		},
	})

	client := newMockAgentClient().
		WithGetRunResponse("run-parked", &Run{ID: "run-parked", Status: "RUN_STATUS_PARKED"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.PausedAgentIDs) != 1 || status.PausedAgentIDs[0] != "agent-1" {
		t.Fatalf("expected parked run classified paused, got %v", status.PausedAgentIDs)
	}
	if len(status.RunningAgentIDs) != 1 {
		t.Fatalf("expected parked obligation retained, got %v", status.RunningAgentIDs)
	}
}

func TestRecover_TerminalReleaseReopensCapacity(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-done"},
		},
	})

	client := newMockAgentClient().
		WithGetRunResponse("run-done", &Run{ID: "run-done", Status: "RUN_STATUS_COMPLETE"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("expected terminal obligation released, got %v", tec.Status().RunningAgentIDs)
	}
	// Proven terminal evidence releases the slot for the next scheduled tick.
	result, err := tec.Enqueue(context.Background(), "agent-2", "p")
	if err != nil {
		t.Fatalf("expected capacity reopened after terminal release, got %v", err)
	}
	if result.Status != "started" {
		t.Fatalf("expected agent-2 to start, got %q", result.Status)
	}
}

func TestRecover_PreservesRecordedObligationWithoutReprobing(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-x", State: ObligationOwnerUnreachable},
		},
	})

	client := newMockAgentClient().
		WithGetRunResponse("run-x", &Run{ID: "run-x", Status: "RUN_STATUS_RUNNING"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.UnreachableAgentIDs) != 1 {
		t.Fatalf("expected recorded obligation preserved, got %+v", status)
	}
	if len(client.getRunCalls) != 0 {
		t.Fatalf("expected no owner probe for a recorded obligation, got %v", client.getRunCalls)
	}
}

func TestSetRunningRunID_UpdatesEntry(t *testing.T) {
	dir := t.TempDir()
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, nil)
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	tec.SetRunningRunID("agent-1", "run-123")

	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].RunID != "run-123" {
		t.Fatalf("expected RunID persisted, got %+v", persisted.Running)
	}
}

// TestBeginDispatch_PersistsIntentBeforeRequest verifies the dispatch intent
// (state plus owner idempotency key) is durable before the start request, so a
// crash mid-dispatch recovers as an obligation rather than a free slot.
func TestBeginDispatch_PersistsIntentBeforeRequest(t *testing.T) {
	dir := t.TempDir()
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, nil)
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	tec.BeginDispatch("agent-1", DispatchIntent{
		IdempotencyKey: "prompt-manager-heartbeat:team-x:agent-1:attempt-1",
		TaskID:         "task-1",
		RunTag:         "heartbeat-team-x-agent-1",
	})

	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 {
		t.Fatalf("expected one persisted running entry, got %+v", persisted.Running)
	}
	entry := persisted.Running[0]
	if entry.State != ObligationDispatchUncertain {
		t.Fatalf("expected dispatch_uncertain persisted before request, got state=%q", entry.State)
	}
	if entry.IdempotencyKey != "prompt-manager-heartbeat:team-x:agent-1:attempt-1" {
		t.Fatalf("expected persisted idempotency key, got %q", entry.IdempotencyKey)
	}
	if entry.TaskID != "task-1" || entry.RunTag != "heartbeat-team-x-agent-1" {
		t.Fatalf("expected persisted owner request essentials, got taskID=%q runTag=%q", entry.TaskID, entry.RunTag)
	}
	if got := tec.Status(); len(got.UncertainAgentIDs) != 1 {
		t.Fatalf("expected the in-flight dispatch to be visible as uncertain, got %+v", got)
	}
}

// TestSetRunningRunID_ClearsDispatchIntent verifies a confirmed owner response
// resolves the pre-request uncertainty.
func TestSetRunningRunID_ClearsDispatchIntent(t *testing.T) {
	dir := t.TempDir()
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, nil)
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.BeginDispatch("agent-1", DispatchIntent{IdempotencyKey: "key-1", TaskID: "task-1"})
	tec.SetRunningRunID("agent-1", "run-confirmed")

	got := tec.Status()
	if len(got.UncertainAgentIDs) != 0 {
		t.Fatalf("expected confirmed run to clear uncertainty, got %+v", got)
	}
	if len(got.RunningAgentIDs) != 1 {
		t.Fatalf("expected run retained, got %+v", got)
	}
}

// TestRecover_PreservesDispatchIntentKey verifies a crash mid-dispatch keeps the
// recorded idempotency key across restart for later owner reconciliation.
func TestRecover_PreservesDispatchIntentKey(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", State: ObligationDispatchUncertain, IdempotencyKey: "key-1"},
		},
	})

	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, newMockAgentClient())
	tec.Recover(context.Background())

	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].IdempotencyKey != "key-1" {
		t.Fatalf("expected recorded dispatch intent key preserved, got %+v", persisted.Running)
	}
	if got := tec.Status(); len(got.UncertainAgentIDs) != 1 {
		t.Fatalf("expected uncertain obligation preserved, got %+v", got)
	}
}

// watcherExecutor records retained-run watcher calls so the Recover/Reconcile
// path can be verified to re-attach a completion waiter.
type watcherExecutor struct {
	captureExecutor
	watched []struct {
		teamID, agentID, runID, profileKey, taskID, runTag string
	}
}

func (w *watcherExecutor) WatchRecoveredRun(teamID, agentID, runID, profileKey, taskID, runTag string) {
	w.watched = append(w.watched, struct {
		teamID, agentID, runID, profileKey, taskID, runTag string
	}{teamID, agentID, runID, profileKey, taskID, runTag})
}

// TestReconcileDispatch_ReplaysRecordedIntentAndBindsRun verifies a retained
// uncertain dispatch converges on the owner's run by replaying the recorded
// request with the same idempotency key, and that a completion waiter is
// re-attached so the recovered obligation can eventually release.
func TestReconcileDispatch_ReplaysRecordedIntentAndBindsRun(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().WithCreateRunResponse(&Run{ID: "run-v1", TaskID: "task-1", Status: "RUN_STATUS_RUNNING"})
	exec := &watcherExecutor{}
	tec := newTeamExecutionContext("team-x", exec, dir, client)

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.BeginDispatch("agent-1", DispatchIntent{
		IdempotencyKey: "prompt-manager-heartbeat:team-x:agent-1:attempt-1",
		TaskID:         "task-1",
		RunTag:         "heartbeat-team-x-agent-1",
	})

	if err := tec.ReconcileDispatch(context.Background(), "agent-1"); err != nil {
		t.Fatalf("ReconcileDispatch: %v", err)
	}

	if len(client.createRunCalls) != 1 {
		t.Fatalf("expected exactly one replayed CreateRun, got %d", len(client.createRunCalls))
	}
	req := client.createRunCalls[0]
	if req.IdempotencyKey != "prompt-manager-heartbeat:team-x:agent-1:attempt-1" {
		t.Fatalf("expected replayed idempotency key preserved, got %q", req.IdempotencyKey)
	}
	if req.TaskID != "task-1" {
		t.Fatalf("expected replayed task preserved, got %q", req.TaskID)
	}

	status := tec.Status()
	if len(status.UncertainAgentIDs) != 0 {
		t.Fatalf("expected uncertainty cleared after confirmed replay, got %+v", status)
	}
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" {
		t.Fatalf("expected bound run retained, got %+v", status)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].RunID != "run-v1" {
		t.Fatalf("expected confirmed RunID persisted, got %+v", persisted.Running)
	}
	if len(exec.watched) != 1 || exec.watched[0].runID != "run-v1" || exec.watched[0].teamID != "team-x" {
		t.Fatalf("expected one recovered-run watcher for run-v1, got %+v", exec.watched)
	}
}

// TestReconcileDispatch_UncertainOwnerRetainsObligation verifies an owner that
// is still unreachable does not release the slot or start a duplicate.
func TestReconcileDispatch_UncertainOwnerRetainsObligation(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().WithCreateRunError(NewDispatchUncertainError(errors.New("connection reset")))
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.BeginDispatch("agent-1", DispatchIntent{IdempotencyKey: "key-1", TaskID: "task-1"})

	if err := tec.ReconcileDispatch(context.Background(), "agent-1"); err == nil {
		t.Fatalf("expected uncertain replay to return an error")
	}
	if len(client.createRunCalls) != 1 {
		t.Fatalf("expected one replay attempt, got %d", len(client.createRunCalls))
	}
	status := tec.Status()
	if len(status.UncertainAgentIDs) != 1 || status.UncertainAgentIDs[0] != "agent-1" {
		t.Fatalf("expected uncertain obligation retained, got %+v", status)
	}
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-p"); err == nil || !IsMemberAlreadyQueued(err) {
		t.Fatalf("expected duplicate refused while uncertain, got %v", err)
	}
}

// TestReconcileDispatch_DefinitiveRefusalReleasesSlot verifies a replay the
// owner definitively refuses releases the retained slot for a future attempt.
func TestReconcileDispatch_DefinitiveRefusalReleasesSlot(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().WithCreateRunError(NewDispatchRejectedError(errors.New("validation error")))
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.BeginDispatch("agent-1", DispatchIntent{IdempotencyKey: "key-1", TaskID: "task-1"})

	if err := tec.ReconcileDispatch(context.Background(), "agent-1"); err == nil {
		t.Fatalf("expected definitive refusal to return an error")
	}
	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("expected retained slot released, got %+v", tec.Status())
	}
	result, err := tec.Enqueue(context.Background(), "agent-2", "profile-p")
	if err != nil || result.Status != "started" {
		t.Fatalf("expected capacity reopened after refusal, got result=%+v err=%v", result, err)
	}
}

// TestReconcileDispatch_IncompleteIntentNotReplayed verifies a recorded
// obligation without enough owner request detail is never sent to the owner.
func TestReconcileDispatch_IncompleteIntentNotReplayed(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().WithCreateRunResponse(&Run{ID: "run-v1", Status: "RUN_STATUS_RUNNING"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.MarkDispatchUncertain("agent-1", "no recorded intent")

	if err := tec.ReconcileDispatch(context.Background(), "agent-1"); err == nil {
		t.Fatalf("expected incomplete intent to return an error")
	}
	if len(client.createRunCalls) != 0 {
		t.Fatalf("expected no owner replay for incomplete intent, got %d", len(client.createRunCalls))
	}
	if len(tec.Status().UncertainAgentIDs) != 1 {
		t.Fatalf("expected obligation retained, got %+v", tec.Status())
	}
}

// TestRecover_ReplaysRecordedUncertainDispatch verifies restart recovery
// resolves a persisted uncertain dispatch by replaying the recorded intent.
func TestRecover_ReplaysRecordedUncertainDispatch(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{
				AgentID:        "agent-1",
				ProfileKey:     "profile-p",
				State:          ObligationDispatchUncertain,
				IdempotencyKey: "prompt-manager-heartbeat:team-x:agent-1:attempt-1",
				TaskID:         "task-1",
				RunTag:         "heartbeat-team-x-agent-1",
			},
		},
	})
	client := newMockAgentClient().WithCreateRunResponse(&Run{ID: "run-recovered", TaskID: "task-1", Status: "RUN_STATUS_RUNNING"})
	exec := &watcherExecutor{}
	tec := newTeamExecutionContext("team-x", exec, dir, client)

	tec.Recover(context.Background())

	if len(client.createRunCalls) != 1 {
		t.Fatalf("expected one recovery replay, got %d", len(client.createRunCalls))
	}
	if client.createRunCalls[0].IdempotencyKey != "prompt-manager-heartbeat:team-x:agent-1:attempt-1" {
		t.Fatalf("expected recorded key replayed, got %q", client.createRunCalls[0].IdempotencyKey)
	}
	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 || len(status.UncertainAgentIDs) != 0 {
		t.Fatalf("expected recovered run bound, got %+v", status)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].RunID != "run-recovered" {
		t.Fatalf("expected recovered RunID persisted, got %+v", persisted.Running)
	}
	if len(exec.watched) != 1 {
		t.Fatalf("expected recovered run watcher re-attached, got %+v", exec.watched)
	}
}

func TestSetRunningRunID_NoopWhenAgentNotRunning(t *testing.T) {
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, t.TempDir(), nil)
	// Does not panic; just logs and returns.
	tec.SetRunningRunID("ghost-agent", "run-x")
	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("expected no running entries after no-op SetRunningRunID")
	}
}

func TestClearRunning_TerminalRunSucceeds(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().
		WithGetRunResponse("run-done", &Run{ID: "run-done", Status: "RUN_STATUS_COMPLETE"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.SetRunningRunID("agent-1", "run-done")

	if err := tec.ClearRunning(context.Background(), "agent-1", false); err != nil {
		t.Fatalf("ClearRunning: %v", err)
	}
	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("expected entry cleared")
	}
}

func TestClearRunning_ActiveRunRefused(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().
		WithGetRunResponse("run-live", &Run{ID: "run-live", Status: "RUN_STATUS_RUNNING"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.SetRunningRunID("agent-1", "run-live")

	err := tec.ClearRunning(context.Background(), "agent-1", false)
	if err == nil {
		t.Fatalf("expected RunningStillActiveError")
	}
	if !IsRunningStillActive(err) {
		t.Fatalf("expected RunningStillActiveError, got %T: %v", err, err)
	}
	if len(tec.Status().RunningAgentIDs) != 1 {
		t.Fatalf("entry should still be present after refusal")
	}
}

func TestClearRunning_ForceBypassesActiveCheck(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().
		WithGetRunResponse("run-live", &Run{ID: "run-live", Status: "RUN_STATUS_RUNNING"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.SetRunningRunID("agent-1", "run-live")

	if err := tec.ClearRunning(context.Background(), "agent-1", true); err != nil {
		t.Fatalf("force ClearRunning: %v", err)
	}
	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("force should clear despite active run")
	}
}

func TestClearRunning_EmptyRunIDUnconditional(t *testing.T) {
	dir := t.TempDir()
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, newMockAgentClient())

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	// No SetRunningRunID — entry has empty RunID.

	if err := tec.ClearRunning(context.Background(), "agent-1", false); err != nil {
		t.Fatalf("ClearRunning: %v", err)
	}
	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("expected entry cleared")
	}
}

func TestClearRunning_NotFound(t *testing.T) {
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, t.TempDir(), nil)
	err := tec.ClearRunning(context.Background(), "ghost", false)
	if err != ErrRunningEntryNotFound {
		t.Fatalf("expected ErrRunningEntryNotFound, got %v", err)
	}
}

// fakeTeamExecStoreRegistrar captures SetRunningRunID calls for executor tests.
type fakeTeamExecStoreRegistrar struct {
	calls          []struct{ TeamID, AgentID, RunID string }
	beginCalls     []struct{ TeamID, AgentID, Key string }
	uncertainCalls []struct{ TeamID, AgentID, Reason string }
}

func (f *fakeTeamExecStoreRegistrar) SetRunningRunID(teamID, agentID, runID string) {
	f.calls = append(f.calls, struct{ TeamID, AgentID, RunID string }{teamID, agentID, runID})
}

func (f *fakeTeamExecStoreRegistrar) BeginDispatch(teamID, agentID string, intent DispatchIntent) {
	f.beginCalls = append(f.beginCalls, struct{ TeamID, AgentID, Key string }{teamID, agentID, intent.IdempotencyKey})
}

func (f *fakeTeamExecStoreRegistrar) MarkDispatchUncertain(teamID, agentID, reason string) {
	f.uncertainCalls = append(f.uncertainCalls, struct{ TeamID, AgentID, Reason string }{teamID, agentID, reason})
}

func TestExecutor_SetTeamExecStore_RegistersRunIDAfterCreateRun(t *testing.T) {
	// This exercises the Executor.SetTeamExecStore wiring. A full Execute
	// integration test would need a configured team — keep this focused on
	// the seam: the registrar is captured and invokable.
	registrar := &fakeTeamExecStoreRegistrar{}
	e := &Executor{}
	e.SetTeamExecStore(registrar)
	if e.teamExecStore == nil {
		t.Fatalf("expected teamExecStore wired")
	}
	e.teamExecStore.SetRunningRunID("t", "a", "r")
	if len(registrar.calls) != 1 || registrar.calls[0].RunID != "r" {
		t.Fatalf("expected registrar.SetRunningRunID called once, got %+v", registrar.calls)
	}
}

// --- HTTP handler tests ---

func TestClearTeamQueueRunning_OK(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().
		WithGetRunResponse("run-done", &Run{ID: "run-done", Status: "RUN_STATUS_COMPLETE"})
	teamExecStore := NewTeamExecutionStore(nil, &captureExecutor{}, dir, client)

	// Pre-populate a running entry by enqueuing then SetRunningRunID.
	tec := teamExecStore.GetOrCreate("team-x")
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.SetRunningRunID("agent-1", "run-done")

	handlers := &Handlers{teamExecStore: teamExecStore}

	req := httptest.NewRequest(http.MethodDelete, "/teams/team-x/queue/running/agent-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-x", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.ClearTeamQueueRunning(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var status TeamExecutionStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(status.RunningAgentIDs) != 0 {
		t.Fatalf("expected cleared status, got %v", status.RunningAgentIDs)
	}
}

func TestClearTeamQueueRunning_NotFound(t *testing.T) {
	dir := t.TempDir()
	teamExecStore := NewTeamExecutionStore(nil, &captureExecutor{}, dir, newMockAgentClient())
	handlers := &Handlers{teamExecStore: teamExecStore}

	req := httptest.NewRequest(http.MethodDelete, "/teams/team-x/queue/running/ghost", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-x", "agentId": "ghost"})
	w := httptest.NewRecorder()

	handlers.ClearTeamQueueRunning(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestClearTeamQueueRunning_ActiveRefused409(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().
		WithGetRunResponse("run-live", &Run{ID: "run-live", Status: "RUN_STATUS_RUNNING"})
	teamExecStore := NewTeamExecutionStore(nil, &captureExecutor{}, dir, client)
	tec := teamExecStore.GetOrCreate("team-x")
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.SetRunningRunID("agent-1", "run-live")

	handlers := &Handlers{teamExecStore: teamExecStore}
	req := httptest.NewRequest(http.MethodDelete, "/teams/team-x/queue/running/agent-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-x", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.ClearTeamQueueRunning(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "active") {
		t.Fatalf("expected error body to mention active, got %q", w.Body.String())
	}
}

func TestClearTeamQueueRunning_ForceQueryBypassesActive(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().
		WithGetRunResponse("run-live", &Run{ID: "run-live", Status: "RUN_STATUS_RUNNING"})
	teamExecStore := NewTeamExecutionStore(nil, &captureExecutor{}, dir, client)
	tec := teamExecStore.GetOrCreate("team-x")
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.SetRunningRunID("agent-1", "run-live")

	handlers := &Handlers{teamExecStore: teamExecStore}
	req := httptest.NewRequest(http.MethodDelete, "/teams/team-x/queue/running/agent-1?force=true", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-x", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.ClearTeamQueueRunning(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with force, got %d: %s", w.Code, w.Body.String())
	}
}
