package heartbeat

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"prompt-manager/internal/store"
)

type terminalReconcileConfigStore struct {
	*store.FileTeamStore
	mu               sync.Mutex
	reads            int
	disableAfterRead int
}

type terminalReconcileExecutor struct {
	projector                *Executor
	store                    *store.FileTeamStore
	mu                       sync.Mutex
	calls                    []executionCall
	projectedBeforeAdmission bool
	projectionSucceeds       bool
	watchCalls               int
	executed                 chan struct{}
}

func (e *terminalReconcileExecutor) Execute(ctx context.Context, teamID, agentID, profileKey string) (*ExecutionResult, error) {
	cfg, err := e.store.GetHeartbeatConfig(ctx, teamID, agentID)
	e.mu.Lock()
	e.calls = append(e.calls, executionCall{teamID: teamID, agentID: agentID, profileKey: profileKey})
	e.projectedBeforeAdmission = err == nil && cfg != nil && cfg.LastExecution != nil && cfg.LastExecution.RunID == "run-prior" && cfg.LastExecution.Status == store.HeartbeatStatusCompleted
	e.mu.Unlock()
	if e.executed != nil {
		e.executed <- struct{}{}
	}
	return &ExecutionResult{TeamID: teamID, AgentID: agentID, Status: store.HeartbeatStatusRunning}, nil
}

func (e *terminalReconcileExecutor) ProjectRetainedTerminal(teamID, agentID, runID, profileKey, taskID, tag, admissionOrigin string, run *Run) retainedTerminalProjection {
	if !e.projectionSucceeds {
		return retainedTerminalUnconfirmed
	}
	projector, ok := any(e.projector).(interface {
		ProjectRetainedTerminal(teamID, agentID, runID, profileKey, taskID, tag, admissionOrigin string, run *Run) retainedTerminalProjection
	})
	if !ok {
		return retainedTerminalUnconfirmed
	}
	return projector.ProjectRetainedTerminal(teamID, agentID, runID, profileKey, taskID, tag, admissionOrigin, run)
}

func (e *terminalReconcileExecutor) WatchRecoveredRunWithProjection(teamID, agentID, runID, profileKey, taskID, runTag, ownerStatus, ownerStartedAt, ownerTerminalAt, admissionOrigin string) bool {
	e.mu.Lock()
	e.watchCalls++
	e.mu.Unlock()
	return true
}

func (e *terminalReconcileExecutor) callCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.calls)
}

func (e *terminalReconcileExecutor) watchCallCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.watchCalls
}

func TestCompleteMemberKeepsExactOwnerWhenPersistenceFails(t *testing.T) {
	persistDir := t.TempDir()
	tec := newTeamExecutionContext("team-1", nil, persistDir, nil)
	tec.running["agent-1"] = runningEntry{RunID: "run-prior", State: ObligationWatcherUnavailable}
	tec.queue = []queuedExecution{{AgentID: "agent-2"}}
	tec.queued["agent-2"] = true
	if err := tec.persistLocked(); err != nil {
		t.Fatalf("persist initial owner: %v", err)
	}

	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("file"), 0o600); err != nil {
		t.Fatalf("create persistence blocker: %v", err)
	}
	tec.persistDir = filepath.Join(blocker, "child")

	tec.OnMemberCompleteRun("agent-1", "run-prior")
	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 || status.Owners[0].RunID != "run-prior" {
		t.Fatalf("failed durable release must retain exact in-memory owner: %+v", status)
	}
	if len(status.Queue) != 1 || status.Queue[0] != "agent-2" || !tec.queued["agent-2"] {
		t.Fatalf("failed queue save must restore the waiting member and index: status=%+v queued=%v", status, tec.queued)
	}
}

func (e *terminalReconcileExecutor) sawTerminalProjectionBeforeAdmission() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.projectedBeforeAdmission
}

func (s *terminalReconcileConfigStore) GetHeartbeatConfig(ctx context.Context, teamID, agentID string) (*store.HeartbeatConfig, error) {
	cfg, err := s.FileTeamStore.GetHeartbeatConfig(ctx, teamID, agentID)
	if err != nil || cfg == nil {
		return cfg, err
	}
	s.mu.Lock()
	s.reads++
	disable := s.disableAfterRead > 0 && s.reads >= s.disableAfterRead
	s.mu.Unlock()
	if disable {
		cfg.Enabled = false
	}
	return cfg, nil
}

func (s *terminalReconcileConfigStore) readCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reads
}

func seedUnavailableRun(t *testing.T, tec *TeamExecutionContext, runID string) {
	t.Helper()
	tec.mu.Lock()
	tec.running["agent-1"] = runningEntry{
		ProfileKey: "test-profile",
		TaskID:     "prior-task",
		RunID:      runID,
		State:      ObligationWatcherUnavailable,
	}
	if err := tec.persistLocked(); err != nil {
		tec.mu.Unlock()
		t.Fatalf("persist retained run: %v", err)
	}
	tec.mu.Unlock()
}

func newTerminalReconcileFixture(t *testing.T, configStore HeartbeatConfigStore) (*Scheduler, *store.FileTeamStore, *mockAgentClient, *TeamExecutionStore, *terminalReconcileExecutor) {
	t.Helper()
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	client := newMockAgentClient().
		WithGetRunResponse("run-prior", &Run{ID: "run-prior", TaskID: "prior-task", Status: "RUN_STATUS_COMPLETE", EndedAt: time.Now().UTC().Format(time.RFC3339Nano)}).
		WithCreateTaskResponse(&Task{ID: "task-next", Title: "next heartbeat"}).
		WithCreateRunResponse(&Run{ID: "run-next", TaskID: "task-next", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunError(errForTest("keep the new waiter pending"))
	projector := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	executor := &terminalReconcileExecutor{projector: projector, store: teamStore, projectionSucceeds: true, executed: make(chan struct{}, 1)}
	teamExecStore := NewTeamExecutionStore(teamStore, executor, t.TempDir(), client)
	projector.SetTeamExecStore(teamExecStore)
	tec := teamExecStore.GetOrCreate("team-1")
	seedUnavailableRun(t, tec, "run-prior")
	if configStore == nil {
		configStore = teamStore
	}
	scheduler := NewScheduler(executor, client, configStore, teamExecStore)
	return scheduler, teamStore, client, teamExecStore, executor
}

func TestSchedulerAdmitsAfterTerminalOwnerIsProjectedAndReleased(t *testing.T) {
	scheduler, teamStore, _, teamExecStore, executor := newTerminalReconcileFixture(t, nil)

	scheduler.executeHeartbeat(context.Background(), "team-1", "agent-1")
	select {
	case <-executor.executed:
	case <-time.After(time.Second):
		t.Fatal("same callback did not enqueue after terminal reconciliation")
	}

	if calls := executor.callCount(); calls != 1 {
		t.Fatalf("same callback should admit exactly one next run after terminal projection; execute calls=%d", calls)
	}
	status := teamExecStore.Status("team-1")
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" {
		t.Fatalf("next run should own the released slot: %+v", status)
	}
	cfg, err := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LastExecution == nil || cfg.LastExecution.RunID != "run-prior" || cfg.LastExecution.Status != store.HeartbeatStatusCompleted || !executor.sawTerminalProjectionBeforeAdmission() {
		t.Fatalf("next admission must follow a durable terminal projection, config=%+v beforeAdmission=%v", cfg.LastExecution, executor.sawTerminalProjectionBeforeAdmission())
	}
	attempts, _, err := teamStore.ListHeartbeatAttempts(context.Background(), "team-1", "agent-1", "", "", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	seenPrior := false
	for _, attempt := range attempts {
		if attempt.RunID == "run-prior" && attempt.Phase == "run_terminal" && attempt.Status == store.HeartbeatStatusCompleted {
			seenPrior = true
		}
	}
	if !seenPrior {
		t.Fatalf("terminal run must be durably projected before next admission; attempts=%+v", attempts)
	}
}

func TestSchedulerRechecksHeartbeatConfigAfterTerminalProjection(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	configStore := &terminalReconcileConfigStore{FileTeamStore: teamStore, disableAfterRead: 2}
	client := newMockAgentClient().
		WithGetRunResponse("run-prior", &Run{ID: "run-prior", TaskID: "prior-task", Status: "RUN_STATUS_FAILED", Error: "terminal failure", EndedAt: time.Now().UTC().Format(time.RFC3339Nano)}).
		WithCreateTaskResponse(&Task{ID: "task-next", Title: "next heartbeat"}).
		WithCreateRunResponse(&Run{ID: "run-next", TaskID: "task-next", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunError(errForTest("keep the new waiter pending"))
	projector := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	executor := &terminalReconcileExecutor{projector: projector, store: teamStore, projectionSucceeds: true}
	teamExecStore := NewTeamExecutionStore(teamStore, executor, t.TempDir(), client)
	projector.SetTeamExecStore(teamExecStore)
	seedUnavailableRun(t, teamExecStore.GetOrCreate("team-1"), "run-prior")
	scheduler := NewScheduler(executor, client, configStore, teamExecStore)
	scheduler.teamStore = teamStore

	scheduler.executeHeartbeat(context.Background(), "team-1", "agent-1")

	if got := configStore.readCount(); got != 2 {
		t.Fatalf("terminal recovery must re-read the ordinary config gate, reads=%d", got)
	}
	if calls := executor.callCount(); calls != 0 {
		t.Fatalf("disabled post-projection config must block new admission, execute calls=%d", calls)
	}
	status := teamExecStore.Status("team-1")
	if len(status.RunningAgentIDs) != 0 {
		t.Fatalf("old terminal slot should be released even though fresh admission is disabled: %+v", status)
	}
	cfg, err := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
	if err != nil || cfg.LastExecution == nil || cfg.LastExecution.RunID != "run-prior" || cfg.LastExecution.Status != store.HeartbeatStatusFailed {
		t.Fatalf("failed terminal projection should remain visible, config=%+v err=%v", cfg, err)
	}
}

func TestSchedulerRetainsOwnerWhenTerminalProjectionIsUnconfirmed(t *testing.T) {
	scheduler, _, _, teamExecStore, executor := newTerminalReconcileFixture(t, nil)
	executor.projectionSucceeds = false

	scheduler.executeHeartbeat(context.Background(), "team-1", "agent-1")

	if calls := executor.callCount(); calls != 0 {
		t.Fatalf("unconfirmed terminal projection must not admit next work, execute calls=%d", calls)
	}
	if watches := executor.watchCallCount(); watches != 0 {
		t.Fatalf("production projector refusal must not attach an asynchronous retry watcher, watch calls=%d", watches)
	}
	status := teamExecStore.Status("team-1")
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" || len(status.WatcherUnavailableAgentIDs) != 1 {
		t.Fatalf("unconfirmed projection must retain the exact owner slot: %+v", status)
	}
}

func TestSchedulerDefersAdmissionWhenTerminalProjectionIsSuperseded(t *testing.T) {
	scheduler, teamStore, _, teamExecStore, executor := newTerminalReconcileFixture(t, nil)
	cfg, err := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	cfg.LastExecution = &store.HeartbeatExecResult{
		RunID:     "run-newer",
		StartedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Status:    store.HeartbeatStatusRunning,
	}
	if err := teamStore.SetHeartbeatConfig(context.Background(), "team-1", "agent-1", cfg); err != nil {
		t.Fatalf("persist newer heartbeat projection: %v", err)
	}

	scheduler.executeHeartbeat(context.Background(), "team-1", "agent-1")

	if calls := executor.callCount(); calls != 0 {
		t.Fatalf("superseded terminal projection must not admit work in the same callback, execute calls=%d", calls)
	}
	status := teamExecStore.Status("team-1")
	if len(status.RunningAgentIDs) != 0 {
		t.Fatalf("exact terminal old owner should be released without same-callback admission: %+v", status)
	}
	cfg, err = teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
	if err != nil || cfg.LastExecution == nil || cfg.LastExecution.RunID != "run-newer" || cfg.LastExecution.Status != store.HeartbeatStatusRunning {
		t.Fatalf("newer heartbeat projection must remain unchanged, result=%+v err=%v", cfg.LastExecution, err)
	}
}

func TestSchedulerDoesNotAdmitWhileRetainedOwnerIsStillActive(t *testing.T) {
	scheduler, _, client, teamExecStore, executor := newTerminalReconcileFixture(t, nil)
	client.mu.Lock()
	client.getRuns["run-prior"] = &Run{ID: "run-prior", TaskID: "prior-task", Status: "RUN_STATUS_RUNNING"}
	client.mu.Unlock()

	scheduler.executeHeartbeat(context.Background(), "team-1", "agent-1")

	if calls := executor.callCount(); calls != 0 {
		t.Fatalf("active retained owner must not allow same-callback admission, execute calls=%d", calls)
	}
	status := teamExecStore.Status("team-1")
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" {
		t.Fatalf("active owner must retain its slot: %+v", status)
	}
}
