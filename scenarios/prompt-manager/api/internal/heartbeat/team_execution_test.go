package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"prompt-manager/internal/requestpolicy"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
)

func TestTeamExecutionStatusIncludesOwnerIdentifiersWithoutPayloads(t *testing.T) {
	ctx := newTeamExecutionContext("team-1", &captureExecutor{}, t.TempDir(), nil)
	ctx.running["nooch-delivery-orchestrator"] = runningEntry{
		State: ObligationDispatchUncertain, DispatchStage: DispatchStageIntent,
		TaskID: "task-123", RunID: "", AdmissionOrigin: AdmissionOriginScheduled,
		OwnerStartedAt: "2026-10-17T05:30:01Z", OwnerObligationID: "obligation-456",
		TaskRequestSnapshot: `{"private":"must not appear"}`,
	}
	ctx.queue = []queuedExecution{{AgentID: "queued-member", TaskID: "queued-task", DispatchStage: DispatchStagePreparing}}

	status := ctx.Status()
	if len(status.Owners) != 2 {
		t.Fatalf("owners = %+v, want running and queued owner records", status.Owners)
	}
	uncertain := status.Owners[0]
	if uncertain.AgentID != "nooch-delivery-orchestrator" || uncertain.Phase != "retained" || uncertain.State != ObligationDispatchUncertain || uncertain.DispatchStage != DispatchStageIntent || uncertain.TaskID != "task-123" || uncertain.RunID != "" || uncertain.ObligationID != "obligation-456" {
		t.Fatalf("uncertain owner = %+v", uncertain)
	}
	queued := status.Owners[1]
	if queued.AgentID != "queued-member" || queued.Phase != "queued" || queued.TaskID != "queued-task" || queued.DispatchStage != DispatchStagePreparing {
		t.Fatalf("queued owner = %+v", queued)
	}
	raw, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "must not appear") || strings.Contains(string(raw), "taskRequestSnapshot") {
		t.Fatalf("execution status exposed private task payload: %s", raw)
	}
}

func TestFormatExecutionFailureIncludesOnlyAllowlistedAdmissionReason(t *testing.T) {
	t.Run("known reason is a separate log field", func(t *testing.T) {
		ownerErr := &AgentManagerHTTPError{
			StatusCode:      403,
			Message:         "service authority does not include this operation",
			AdmissionReason: "grant_limit_reached",
		}
		err := fmt.Errorf("creating task: %w", NewDispatchRejectedError(ownerErr))
		got := formatExecutionFailure("team-1", "agent-1", err)
		if !strings.Contains(got, "admission_reason=grant_limit_reached") {
			t.Fatalf("failure log = %q; missing safe admission category", got)
		}
		if !strings.Contains(got, "agent-manager error: status 403: service authority does not include this operation") {
			t.Fatalf("failure log changed the owner error text: %q", got)
		}
	})

	t.Run("unknown reason is omitted", func(t *testing.T) {
		ownerErr := &AgentManagerHTTPError{
			StatusCode:      403,
			Message:         "refused",
			AdmissionReason: "internal-secret-detail",
		}
		got := formatExecutionFailure("team-1", "agent-1", ownerErr)
		if strings.Contains(got, "admission_reason=") || strings.Contains(got, "internal-secret-detail") {
			t.Fatalf("failure log exposed unrecognized admission data: %q", got)
		}
	})

	t.Run("allowlisted text on non-admission status is omitted", func(t *testing.T) {
		ownerErr := &AgentManagerHTTPError{
			StatusCode:      500,
			Message:         "temporary server failure",
			AdmissionReason: "grant_limit_reached",
		}
		got := formatExecutionFailure("team-1", "agent-1", ownerErr)
		if strings.Contains(got, "admission_reason=") {
			t.Fatalf("non-admission status emitted an admission category: %q", got)
		}
		if !strings.Contains(got, "agent-manager error: status 500: temporary server failure") {
			t.Fatalf("failure log changed existing owner error text: %q", got)
		}
	})

	t.Run("ordinary failure keeps existing format", func(t *testing.T) {
		got := formatExecutionFailure("team-1", "agent-1", fmt.Errorf("local failure"))
		want := "team_execution: execution failed for team-1/agent-1: local failure"
		if got != want {
			t.Fatalf("failure log = %q, want %q", got, want)
		}
	})
}

func TestEnqueueWhenIdle_StartsImmediately(t *testing.T) {
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, t.TempDir(), nil)

	result, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "started" {
		t.Fatalf("expected status 'started', got %q", result.Status)
	}
	if result.Position != 0 {
		t.Fatalf("expected position 0, got %d", result.Position)
	}

	status := ctx.Status()
	if status.State != "active" {
		t.Fatalf("expected state 'active', got %q", status.State)
	}
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" {
		t.Fatalf("expected running [agent-1], got %v", status.RunningAgentIDs)
	}
}

func TestEnqueueWhenActive_QueuesSuccessfully(t *testing.T) {
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, t.TempDir(), nil)

	// Start first agent
	if _, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue agent-1: %v", err)
	}

	// Queue second agent
	result, err := ctx.Enqueue(context.Background(), "agent-2", "profile-2")
	if err != nil {
		t.Fatalf("enqueue agent-2: %v", err)
	}
	if result.Status != "queued" {
		t.Fatalf("expected status 'queued', got %q", result.Status)
	}
	if result.Position != 1 {
		t.Fatalf("expected position 1, got %d", result.Position)
	}

	status := ctx.Status()
	if len(status.Queue) != 1 {
		t.Fatalf("expected 1 queued, got %d", len(status.Queue))
	}
	if status.Queue[0] != "agent-2" {
		t.Fatalf("expected agent-2 in queue, got %q", status.Queue[0])
	}
}

func TestEnqueueDuplicate_ReturnsError(t *testing.T) {
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, t.TempDir(), nil)

	// Start first agent
	if _, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue agent-1: %v", err)
	}

	// Try to enqueue same agent again (already running)
	_, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1")
	if err == nil {
		t.Fatal("expected error for duplicate enqueue")
	}
	if !IsMemberAlreadyQueued(err) {
		t.Fatalf("expected MemberAlreadyQueuedError, got %T: %v", err, err)
	}

	// Queue agent-2, then try to queue agent-2 again
	if _, err := ctx.Enqueue(context.Background(), "agent-2", "profile-2"); err != nil {
		t.Fatalf("enqueue agent-2: %v", err)
	}
	_, err = ctx.Enqueue(context.Background(), "agent-2", "profile-2")
	if err == nil {
		t.Fatal("expected error for duplicate queue entry")
	}
	if !IsMemberAlreadyQueued(err) {
		t.Fatalf("expected MemberAlreadyQueuedError, got %T: %v", err, err)
	}
}

func TestDequeueNext_StartsNextMember(t *testing.T) {
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, t.TempDir(), nil)

	// Start agent-1, queue agent-2
	if _, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue agent-1: %v", err)
	}
	if _, err := ctx.Enqueue(context.Background(), "agent-2", "profile-2"); err != nil {
		t.Fatalf("enqueue agent-2: %v", err)
	}

	// Complete agent-1 -> agent-2 should start
	ctx.OnMemberComplete("agent-1")

	// Give goroutine a moment to start
	time.Sleep(50 * time.Millisecond)

	status := ctx.Status()
	if status.State != "active" {
		t.Fatalf("expected state 'active', got %q", status.State)
	}
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-2" {
		t.Fatalf("expected running [agent-2], got %v", status.RunningAgentIDs)
	}
	if len(status.Queue) != 0 {
		t.Fatalf("expected empty queue, got %d", len(status.Queue))
	}
}

func TestDequeueNext_BecomesIdleWhenEmpty(t *testing.T) {
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, t.TempDir(), nil)

	// Start agent-1
	if _, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue agent-1: %v", err)
	}

	// Complete agent-1 with empty queue
	ctx.OnMemberComplete("agent-1")

	status := ctx.Status()
	if status.State != "idle" {
		t.Fatalf("expected state 'idle', got %q", status.State)
	}
	if len(status.RunningAgentIDs) != 0 {
		t.Fatalf("expected no running agents, got %v", status.RunningAgentIDs)
	}
}

func TestQueueOrderIsFIFO(t *testing.T) {
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, t.TempDir(), nil)

	// Start agent-1, queue agent-2, agent-3, agent-4
	if _, err := ctx.Enqueue(context.Background(), "agent-1", "p"); err != nil {
		t.Fatalf("enqueue agent-1: %v", err)
	}
	for _, id := range []string{"agent-2", "agent-3", "agent-4"} {
		if _, err := ctx.Enqueue(context.Background(), id, "p"); err != nil {
			t.Fatalf("enqueue %s: %v", id, err)
		}
	}

	status := ctx.Status()
	expected := []string{"agent-2", "agent-3", "agent-4"}
	if len(status.Queue) != len(expected) {
		t.Fatalf("expected queue length %d, got %d", len(expected), len(status.Queue))
	}
	for i, id := range expected {
		if status.Queue[i] != id {
			t.Fatalf("queue[%d]: expected %q, got %q", i, id, status.Queue[i])
		}
	}

	// Complete agent-1 -> agent-2 should be next
	ctx.OnMemberComplete("agent-1")
	time.Sleep(50 * time.Millisecond)

	status = ctx.Status()
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-2" {
		t.Fatalf("expected running [agent-2], got %v", status.RunningAgentIDs)
	}
	if len(status.Queue) != 2 {
		t.Fatalf("expected 2 remaining in queue, got %d", len(status.Queue))
	}
	if status.Queue[0] != "agent-3" || status.Queue[1] != "agent-4" {
		t.Fatalf("expected queue [agent-3, agent-4], got %v", status.Queue)
	}
}

func TestPersistAndRecover(t *testing.T) {
	dir := t.TempDir()
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, dir, nil)

	// Start agent-1, queue agent-2, agent-3
	if _, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue agent-1: %v", err)
	}
	// Simulate Executor.Execute registering the run ID after CreateRun
	// returns — otherwise Recover would drop the entry as stale.
	ctx.SetRunningRunID("agent-1", "run-active")
	if _, err := ctx.Enqueue(context.Background(), "agent-2", "profile-2"); err != nil {
		t.Fatalf("enqueue agent-2: %v", err)
	}
	if _, err := ctx.Enqueue(context.Background(), "agent-3", "profile-3"); err != nil {
		t.Fatalf("enqueue agent-3: %v", err)
	}

	// Verify queue file exists
	queueFile := filepath.Join(dir, "team-queue-team-1.json")
	if _, err := os.Stat(queueFile); os.IsNotExist(err) {
		t.Fatal("expected queue file to exist after enqueue")
	}

	// Create new context and recover with a mock client that says the run
	// is still alive (non-terminal).
	client := newMockAgentClient().
		WithGetRunResponse("run-active", &Run{ID: "run-active", Status: "RUN_STATUS_RUNNING"})
	ctx2 := newTeamExecutionContext("team-1", exec, dir, client)
	ctx2.Recover(context.Background())

	status := ctx2.Status()
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" {
		t.Fatalf("expected recovered running [agent-1], got %v", status.RunningAgentIDs)
	}
	// Stale not-yet-dispatched ticks are intentionally dropped on recovery:
	// the cron schedule is the source of truth after a restart, so restoring
	// the backlog would start a catch-up queue the acceptance forbids.
	if len(status.Queue) != 0 {
		t.Fatalf("expected stale queued ticks dropped, got %v", status.Queue)
	}
}

func TestStatus_ReflectsCurrentState(t *testing.T) {
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, t.TempDir(), nil)

	// Initially idle
	status := ctx.Status()
	if status.State != "idle" {
		t.Fatalf("expected initial state 'idle', got %q", status.State)
	}
	if len(status.RunningAgentIDs) != 0 {
		t.Fatalf("expected initial running to be empty, got %v", status.RunningAgentIDs)
	}
	if len(status.Queue) != 0 {
		t.Fatalf("expected initial empty queue, got %d", len(status.Queue))
	}
	if status.TeamID != "team-1" {
		t.Fatalf("expected teamId 'team-1', got %q", status.TeamID)
	}

	// After enqueue -> active
	if _, err := ctx.Enqueue(context.Background(), "agent-1", "p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	status = ctx.Status()
	if status.State != "active" {
		t.Fatalf("expected state 'active' after enqueue, got %q", status.State)
	}

	// After completion with empty queue -> idle
	ctx.OnMemberComplete("agent-1")
	status = ctx.Status()
	if status.State != "idle" {
		t.Fatalf("expected state 'idle' after completion, got %q", status.State)
	}
}

// TestTeamExecutionStore_Recover tests store-level recovery across teams.
func TestTeamExecutionStore_Recover(t *testing.T) {
	dir := t.TempDir()
	exec := &captureExecutor{}

	// Create some queue state manually
	store1 := NewTeamExecutionStore(nil, exec, dir, nil)
	if _, err := store1.Enqueue(context.Background(), "team-a", "agent-1", "p1"); err != nil {
		t.Fatalf("enqueue team-a/agent-1: %v", err)
	}
	store1.SetRunningRunID("team-a", "agent-1", "run-a")
	if _, err := store1.Enqueue(context.Background(), "team-b", "agent-2", "p2"); err != nil {
		t.Fatalf("enqueue team-b/agent-2: %v", err)
	}
	store1.SetRunningRunID("team-b", "agent-2", "run-b")

	// Create new store and recover with a mock that keeps both runs alive.
	client := newMockAgentClient().
		WithGetRunResponse("run-a", &Run{ID: "run-a", Status: "RUN_STATUS_RUNNING"}).
		WithGetRunResponse("run-b", &Run{ID: "run-b", Status: "RUN_STATUS_RUNNING"})
	store2 := NewTeamExecutionStore(nil, exec, dir, client)
	store2.Recover(context.Background())

	statusA := store2.Status("team-a")
	if statusA.State != "active" {
		t.Fatalf("expected team-a state 'active', got %q", statusA.State)
	}

	statusB := store2.Status("team-b")
	if statusB.State != "active" {
		t.Fatalf("expected team-b state 'active', got %q", statusB.State)
	}
}

// TestTeamExecutionStore_OnRunComplete routes completion to the exact run owner.
func TestTeamExecutionStore_OnRunComplete(t *testing.T) {
	exec := &captureExecutor{}
	store := NewTeamExecutionStore(nil, exec, t.TempDir(), nil)

	// Enqueue agent on team-1
	if _, err := store.Enqueue(context.Background(), "team-1", "agent-1", "p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	// Queue agent-2
	if _, err := store.Enqueue(context.Background(), "team-1", "agent-2", "p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// Complete agent-1's registered run.
	store.contexts["team-1"].SetRunningRunID("agent-1", "run-1")
	store.OnRunComplete("team-1", "agent-1", "run-1")
	time.Sleep(50 * time.Millisecond)

	status := store.Status("team-1")
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-2" {
		t.Fatalf("expected running [agent-2] after OnRunComplete, got %v", status.RunningAgentIDs)
	}
}

// TestTeamExecutionStore_StatusUnknownTeam returns idle for unknown teams.
func TestTeamExecutionStore_StatusUnknownTeam(t *testing.T) {
	exec := &captureExecutor{}
	store := NewTeamExecutionStore(nil, exec, t.TempDir(), nil)

	status := store.Status("nonexistent")
	if status.State != "idle" {
		t.Fatalf("expected idle for unknown team, got %q", status.State)
	}
	if status.TeamID != "nonexistent" {
		t.Fatalf("expected teamId 'nonexistent', got %q", status.TeamID)
	}
}

// TestEnqueue_ExecutorUsesDetachedContext verifies that when an agent is
// enqueued on an idle team, the executor receives a live context even if the
// caller's context is cancelled immediately after Enqueue returns (as happens
// when the HTTP handler writes 202 and the request context is done).
func TestEnqueue_ExecutorUsesDetachedContext(t *testing.T) {
	// ctxRecorder captures the context passed to Execute so we can inspect it.
	type ctxCapture struct {
		ctx context.Context
	}
	var captured ctxCapture
	var mu sync.Mutex
	done := make(chan struct{})

	exec := &captureExecutor{}
	// Wrap captureExecutor to also record the context.
	wrapper := &contextCapturingExecutor{
		inner: exec,
		onExecute: func(ctx context.Context) {
			mu.Lock()
			captured.ctx = ctx
			mu.Unlock()
			close(done)
		},
	}

	tec := newTeamExecutionContext("team-1", wrapper, t.TempDir(), nil)

	// Create a context and cancel it immediately after Enqueue.
	callerCtx, callerCancel := context.WithCancel(context.Background())
	result, err := tec.Enqueue(callerCtx, "agent-1", "profile-1")
	callerCancel() // Simulate HTTP handler returning

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "started" {
		t.Fatalf("expected started, got %q", result.Status)
	}

	// Wait for the goroutine to call Execute
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("executor was never called")
	}

	mu.Lock()
	execCtx := captured.ctx
	mu.Unlock()

	// The critical assertion: the context passed to Execute must NOT be cancelled.
	if execCtx.Err() != nil {
		t.Fatalf("executor received a cancelled context: %v — this causes agent-manager calls to fail", execCtx.Err())
	}
}

// TestEnqueue_ExecuteFailure_ClearsRunningState verifies that when Execute
// returns an error, the running state is cleared so the team doesn't get
// permanently stuck. This was a real bug: if CreateRun failed (e.g. missing
// profile defaults), OnMemberComplete was never called, leaving "running"
// set forever and blocking all future heartbeats for the team.
func TestEnqueue_ExecuteFailure_ClearsRunningState(t *testing.T) {
	exec := &failingExecutor{err: errForTest("creating run: profile not found")}
	ctx := newTeamExecutionContext("team-1", exec, t.TempDir(), nil)

	result, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1")
	if err != nil {
		t.Fatalf("unexpected error from Enqueue: %v", err)
	}
	if result.Status != "started" {
		t.Fatalf("expected status 'started', got %q", result.Status)
	}

	// Wait for the goroutine to run Execute and call OnMemberComplete
	time.Sleep(100 * time.Millisecond)

	status := ctx.Status()
	if status.State != "idle" {
		t.Fatalf("expected state 'idle' after Execute failure, got %q (running=%v) - queue stuck!", status.State, status.RunningAgentIDs)
	}
	if len(status.RunningAgentIDs) != 0 {
		t.Fatalf("expected no running agents after Execute failure, got %v", status.RunningAgentIDs)
	}

	// Verify we can enqueue again (not stuck)
	result2, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1")
	if err != nil {
		t.Fatalf("re-enqueue should succeed after failure: %v", err)
	}
	if result2.Status != "started" {
		t.Fatalf("expected re-enqueue to start, got %q", result2.Status)
	}

	// Wait for the second goroutine to complete before test cleanup
	time.Sleep(100 * time.Millisecond)
}

// failingExecutor always returns an error from Execute.
type failingExecutor struct {
	err error
}

func (f *failingExecutor) Execute(_ context.Context, teamID, agentID, _ string) (*ExecutionResult, error) {
	return &ExecutionResult{
		TeamID:  teamID,
		AgentID: agentID,
		Status:  "failed",
	}, f.err
}

// contextCapturingExecutor wraps a HeartbeatExecutor and calls onExecute with
// the context before delegating.
type contextCapturingExecutor struct {
	inner     HeartbeatExecutor
	onExecute func(ctx context.Context)
}

func (c *contextCapturingExecutor) Execute(ctx context.Context, teamID, agentID, profileKey string) (*ExecutionResult, error) {
	if c.onExecute != nil {
		c.onExecute(ctx)
	}
	return c.inner.Execute(ctx, teamID, agentID, profileKey)
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

func TestClearRunning_EmptyRunIDRequiresExplicitForce(t *testing.T) {
	dir := t.TempDir()
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, newMockAgentClient())

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	// No SetRunningRunID — entry has empty RunID.

	err := tec.ClearRunning(context.Background(), "agent-1", false)
	if !IsRunningOwnerUnverified(err) {
		t.Fatalf("ClearRunning without owner proof error=%v, want RunningOwnerUnverifiedError", err)
	}
	if len(tec.Status().RunningAgentIDs) != 1 {
		t.Fatalf("entry should remain after owner-proof refusal, got %v", tec.Status().RunningAgentIDs)
	}
	if err := tec.ClearRunning(context.Background(), "agent-1", true); err != nil {
		t.Fatalf("explicit-force ClearRunning: %v", err)
	}
	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("explicit force should clear entry")
	}
}

func TestClearCancelledTaskIntentRequiresCompleteOwnerProof(t *testing.T) {
	const taskID = "3a032333-971f-44e2-a88b-81ab16698d8f"
	for _, tc := range []struct {
		name    string
		edit    func(*mockAgentClient)
		allowed bool
	}{
		{"cancelled with no attempts", func(c *mockAgentClient) {}, true},
		{"queued task", func(c *mockAgentClient) { c.getTasks[taskID].Status = "TASK_STATUS_QUEUED" }, false},
		{"task with current run", func(c *mockAgentClient) { c.getTasks[taskID].CurrentRunID = "unobserved-run" }, false},
		{"missing task", func(c *mockAgentClient) { delete(c.getTasks, taskID) }, false},
		{"mismatched task", func(c *mockAgentClient) { c.getTasks[taskID].ID = "another-task" }, false},
		{"task owner unavailable", func(c *mockAgentClient) { c.getTaskErr = errors.New("owner down") }, false},
		{"run owner unavailable", func(c *mockAgentClient) { c.listRunsErr = errors.New("owner down") }, false},
		{"partial inventory", func(c *mockAgentClient) { c.listRunsResp.HasMore = true }, false},
		{"nonempty inventory", func(c *mockAgentClient) { c.listRunsResp.Total = 1 }, false},
		{"unaccounted run", func(c *mockAgentClient) {
			c.listRunsResp.Runs = []*Run{{ID: "run", TaskID: taskID, Status: "RUN_STATUS_RUNNING"}}
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := newMockAgentClient()
			client.getTasks = map[string]*Task{taskID: {ID: taskID, Status: "TASK_STATUS_CANCELLED"}}
			client.listRunsResp = &ListRunsResponse{}
			tc.edit(client)
			dir := t.TempDir()
			tec := newTeamExecutionContext("team", &captureExecutor{}, dir, client)
			tec.running["member"] = runningEntry{TaskID: taskID, IdempotencyKey: "old-intent", DispatchStage: DispatchStageIntent, State: ObligationDispatchUncertain}
			if err := tec.persistLocked(); err != nil {
				t.Fatal(err)
			}
			err := tec.ClearRunning(context.Background(), "member", false)
			if tc.allowed && err != nil {
				t.Fatalf("cancelled, never-run task still needs force: %v", err)
			}
			if !tc.allowed && err == nil {
				t.Fatal("incomplete owner proof released the retained intent")
			}
			if got := len(tec.Status().RunningAgentIDs); (got == 0) != tc.allowed {
				t.Fatalf("retained owners=%d allowed=%v", got, tc.allowed)
			}
			if tc.allowed {
				if len(client.listRunsCalls) != 1 || client.listRunsCalls[0].TaskID != taskID || client.listRunsCalls[0].Offset != 0 || client.listRunsCalls[0].Status != "" {
					t.Fatal("run absence was not read from the exact complete task inventory")
				}
				reloaded := newTeamExecutionContext("team", &captureExecutor{}, dir, client)
				reloaded.Recover(context.Background())
				if len(reloaded.Status().RunningAgentIDs) != 0 {
					t.Fatal("cleared intent returned after restart")
				}
			}
		})
	}
}

func TestClearCancelledTaskIntentRetainsChangedOwnerDuringRead(t *testing.T) {
	var tec *TeamExecutionContext
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/tasks/obsolete-task":
			tec.mu.Lock()
			tec.running["member"] = runningEntry{TaskID: "replacement-task", IdempotencyKey: "new-intent", DispatchStage: DispatchStageIntent, State: ObligationDispatchUncertain}
			tec.mu.Unlock()
			_, _ = w.Write([]byte(`{"task":{"id":"obsolete-task","status":"TASK_STATUS_CANCELLED"}}`))
		case "/api/v1/runs":
			if r.URL.Query().Get("task_id") != "obsolete-task" {
				t.Error("wrong task inventory")
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	tec = newTeamExecutionContext("team", &captureExecutor{}, t.TempDir(), newTestClient(t, server))
	tec.running["member"] = runningEntry{TaskID: "obsolete-task", IdempotencyKey: "old-intent", DispatchStage: DispatchStageIntent, State: ObligationDispatchUncertain}
	if err := tec.ClearRunning(context.Background(), "member", false); !IsRunningOwnerUnverified(err) {
		t.Fatalf("changed retained owner was cleared: %v", err)
	}
	if tec.running["member"].TaskID != "replacement-task" {
		t.Fatal("replacement owner was lost")
	}
}

func TestClearRunningRequiresExactTerminalOwnerProof(t *testing.T) {
	for _, tc := range []struct {
		name              string
		client            AgentClient
		wantLookupFailure bool
	}{
		{name: "agent manager unavailable", wantLookupFailure: true},
		{name: "response missing run ID", client: newMockAgentClient().WithGetRunResponse("run-expected", nil)},
		{name: "response has a different run ID", client: newMockAgentClient().WithGetRunResponse("run-expected", &Run{ID: "run-other", Status: "RUN_STATUS_COMPLETE"})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tec := newTeamExecutionContext("team-x", &captureExecutor{}, t.TempDir(), tc.client)
			if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
				t.Fatalf("enqueue: %v", err)
			}
			tec.SetRunningRunID("agent-1", "run-expected")

			err := tec.ClearRunning(context.Background(), "agent-1", false)
			if tc.wantLookupFailure {
				if !IsRunningOwnerLookupFailure(err) {
					t.Fatalf("ClearRunning error=%v, want lookup failure", err)
				}
			} else if !IsRunningOwnerUnverified(err) {
				t.Fatalf("ClearRunning error=%v, want owner-proof refusal", err)
			}
			if got := tec.Status().RunningAgentIDs; len(got) != 1 || got[0] != "agent-1" {
				t.Fatalf("owner-proof refusal changed running slot: %v", got)
			}
		})
	}
}

func TestClearRunningLookupFailureIsNotStaleOwnerEvidence(t *testing.T) {
	client := newMockAgentClient()
	client.getRunErr = errors.New("temporary lookup failure")
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, t.TempDir(), client)
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.SetRunningRunID("agent-1", "run-expected")

	err := tec.ClearRunning(context.Background(), "agent-1", false)
	if !IsRunningOwnerLookupFailure(err) || !errors.Is(err, client.getRunErr) {
		t.Fatalf("ClearRunning error=%v, want upstream lookup failure", err)
	}
	if got := tec.Status().RunningAgentIDs; len(got) != 1 || got[0] != "agent-1" {
		t.Fatalf("lookup failure changed running slot: %v", got)
	}
}

func TestClearRunning_NotFound(t *testing.T) {
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, t.TempDir(), nil)
	err := tec.ClearRunning(context.Background(), "ghost", false)
	if err != ErrRunningEntryNotFound {
		t.Fatalf("expected ErrRunningEntryNotFound, got %v", err)
	}
}

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

func TestClearTeamQueueRunning_UnverifiedOwnerRefused409(t *testing.T) {
	teamExecStore := NewTeamExecutionStore(nil, &captureExecutor{}, t.TempDir(), nil)
	tec := teamExecStore.GetOrCreate("team-x")
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	request := httptest.NewRequest(http.MethodDelete, "/teams/team-x/queue/running/agent-1", nil)
	request = mux.SetURLVars(request, map[string]string{"id": "team-x", "agentId": "agent-1"})
	response := httptest.NewRecorder()
	(&Handlers{teamExecStore: teamExecStore}).ClearTeamQueueRunning(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status=%d, want 409: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "no run ID is recorded") {
		t.Fatalf("response should explain missing owner proof, got %q", response.Body.String())
	}
	if got := teamExecStore.Status("team-x").RunningAgentIDs; len(got) != 1 || got[0] != "agent-1" {
		t.Fatalf("unverified clear changed running slot: %v", got)
	}
}

func TestClearTeamQueueRunningLookupFailureReturnsBadGateway(t *testing.T) {
	client := newMockAgentClient()
	client.getRunErr = errors.New("temporary lookup failure")
	teamExecStore := NewTeamExecutionStore(nil, &captureExecutor{}, t.TempDir(), client)
	tec := teamExecStore.GetOrCreate("team-x")
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.SetRunningRunID("agent-1", "run-expected")

	request := httptest.NewRequest(http.MethodDelete, "/teams/team-x/queue/running/agent-1", nil)
	request = mux.SetURLVars(request, map[string]string{"id": "team-x", "agentId": "agent-1"})
	response := httptest.NewRecorder()
	(&Handlers{teamExecStore: teamExecStore}).ClearTeamQueueRunning(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf("status=%d, want 502: %s", response.Code, response.Body.String())
	}
	if got := teamExecStore.Status("team-x").RunningAgentIDs; len(got) != 1 || got[0] != "agent-1" {
		t.Fatalf("lookup failure changed running slot: %v", got)
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

func TestClearTeamQueueRunningRequiresExactTeamManageWhenBounded(t *testing.T) {
	for _, tc := range []struct {
		name    string
		rules   []identity.OperationRule
		force   bool
		allowed bool
	}{
		{
			name:    "exact team target",
			rules:   []identity.OperationRule{{Policy: "team-management", Operations: []string{"team.manage"}, Targets: []string{"prompt-manager:team/team-x"}}},
			allowed: true,
		},
		{
			name:    "canonical wildcard target",
			rules:   []identity.OperationRule{{Policy: "team-management", Operations: []string{"team.manage"}, Targets: []string{"prompt-manager:*"}}},
			allowed: true,
		},
		{
			name:  "wrong team target",
			rules: []identity.OperationRule{{Policy: "team-management", Operations: []string{"team.manage"}, Targets: []string{"prompt-manager:team/other"}}},
		},
		{
			name:  "force does not bypass team authorization",
			rules: []identity.OperationRule{{Policy: "team-management", Operations: []string{"team.manage"}, Targets: []string{"prompt-manager:team/other"}}},
			force: true,
		},
		{
			name: "unrelated operation cannot pair with team target",
			rules: []identity.OperationRule{
				{Policy: "team-preparation", Operations: []string{"team.prepare"}, Targets: []string{"prompt-manager:team/team-x"}},
				{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"prompt-manager:team/team-x"}},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			client := newMockAgentClient().WithGetRunResponse("run-done", &Run{ID: "run-done", Status: "RUN_STATUS_COMPLETE"})
			teamExecStore := NewTeamExecutionStore(nil, &captureExecutor{}, dir, client)
			tec := teamExecStore.GetOrCreate("team-x")
			if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
				t.Fatalf("enqueue: %v", err)
			}
			tec.SetRunningRunID("agent-1", "run-done")

			handler := requestpolicy.PreparationBoundary(http.HandlerFunc((&Handlers{teamExecStore: teamExecStore}).ClearTeamQueueRunning))
			grant := &identity.OperationGrant{
				ID: "approved-team-management", Policy: "team-management", Approver: "owner", ExpiresAt: time.Now().Add(time.Hour), Rules: tc.rules,
			}
			principal := identity.Principal{
				Kind: identity.ActorAgent, Subject: "verified-run", Verified: true,
				Scopes: []string{"prompt-manager:write"}, Grant: grant,
			}
			url := heartbeatconnect.HeartbeatServiceClearTeamQueueRunningProcedure
			if tc.force {
				url += "?force=true"
			}
			req := httptest.NewRequest(http.MethodPost, url, nil)
			req = req.WithContext(identity.WithPrincipal(req.Context(), principal))
			req = mux.SetURLVars(req, map[string]string{"id": "team-x", "agentId": "agent-1"})
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if tc.allowed {
				if rec.Code != http.StatusOK {
					t.Fatalf("approved request status=%d body=%q", rec.Code, rec.Body.String())
				}
				if got := teamExecStore.Status("team-x").RunningAgentIDs; len(got) != 0 {
					t.Fatalf("approved request did not clear running entry: %v", got)
				}
				return
			}
			if rec.Code != http.StatusForbidden {
				t.Fatalf("unapproved request status=%d body=%q, want 403", rec.Code, rec.Body.String())
			}
			if got := teamExecStore.Status("team-x").RunningAgentIDs; len(got) != 1 || got[0] != "agent-1" {
				t.Fatalf("denied request changed running entry: %v", got)
			}
		})
	}
}

func TestClearRunningPersistenceFailureRestoresOwnerAndQueueUntilRetry(t *testing.T) {
	dir := t.TempDir()
	executor := &captureExecutor{}
	client := newMockAgentClient().WithGetRunResponse("run-done", &Run{ID: "run-done", Status: "RUN_STATUS_COMPLETE"})
	ctx := newTeamExecutionContext("team-x", executor, dir, client)
	if _, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue first owner: %v", err)
	}
	ctx.SetRunningRunID("agent-1", "run-done")
	if _, err := ctx.Enqueue(context.Background(), "agent-2", "profile-2"); err != nil {
		t.Fatalf("enqueue waiting owner: %v", err)
	}
	ctx.executionWG.Wait()
	if got := executor.callCount(); got != 1 {
		t.Fatalf("initial executions=%d, want only first owner", got)
	}

	queueFile := ctx.queueFilePath()
	before, err := os.ReadFile(queueFile)
	if err != nil {
		t.Fatalf("read durable queue before clear: %v", err)
	}
	blockedPersistDir := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blockedPersistDir, []byte("block mkdir"), 0o600); err != nil {
		t.Fatalf("create persistence blocker: %v", err)
	}
	ctx.persistDir = blockedPersistDir
	if err := ctx.ClearRunning(context.Background(), "agent-1", false); err == nil {
		t.Fatal("ClearRunning succeeded despite queue persistence failure")
	}
	if status := ctx.Status(); len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" || len(status.Queue) != 1 || status.Queue[0] != "agent-2" {
		t.Fatalf("failed clear changed in-memory owner/queue: %+v", status)
	}
	if got := executor.callCount(); got != 1 {
		t.Fatalf("failed clear dispatched waiting owner; executions=%d", got)
	}
	after, err := os.ReadFile(queueFile)
	if err != nil {
		t.Fatalf("read durable queue after failed clear: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("failed clear changed durable queue\nbefore: %s\nafter:  %s", before, after)
	}

	ctx.persistDir = dir
	if err := ctx.ClearRunning(context.Background(), "agent-1", false); err != nil {
		t.Fatalf("retry ClearRunning: %v", err)
	}
	ctx.executionWG.Wait()
	if status := ctx.Status(); len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-2" || len(status.Queue) != 0 {
		t.Fatalf("successful retry did not promote waiting owner: %+v", status)
	}
	if got := executor.callCount(); got != 2 {
		t.Fatalf("executions after retry=%d, want 2", got)
	}
}
