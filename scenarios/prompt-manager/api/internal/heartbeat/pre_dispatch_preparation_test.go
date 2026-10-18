package heartbeat

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"prompt-manager/internal/store"
)

type countingPreparationExecutor struct {
	calls chan context.Context
}

func (e *countingPreparationExecutor) Execute(ctx context.Context, _, _, _ string) (*ExecutionResult, error) {
	e.calls <- ctx
	return &ExecutionResult{}, nil
}

type blockingPreparationExecutor struct {
	started chan string
	release chan struct{}
}

func (e *blockingPreparationExecutor) Execute(_ context.Context, _, agentID, _ string) (*ExecutionResult, error) {
	e.started <- agentID
	<-e.release
	return &ExecutionResult{}, nil
}

func TestEnqueuePreparationSaveFailureDoesNotStartWorker(t *testing.T) {
	blockedPath := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blockedPath, []byte("block queue directory creation"), 0o600); err != nil {
		t.Fatal(err)
	}
	executor := &countingPreparationExecutor{calls: make(chan context.Context, 1)}
	queue := newTeamExecutionContext("team", executor, blockedPath, nil)

	if _, err := queue.Enqueue(context.Background(), "agent", "profile"); err == nil {
		t.Fatal("Enqueue succeeded without persisting the preparation marker")
	}
	if len(queue.running) != 0 || queue.queued["agent"] {
		t.Fatalf("failed admission left an in-memory obligation: running=%v queued=%v", queue.running, queue.queued)
	}
	select {
	case <-executor.calls:
		t.Fatal("worker started despite preparation persistence failure")
	default:
	}

	queued := newTeamExecutionContext("queued-team", executor, blockedPath, nil)
	queued.running["occupied"] = runningEntry{ProfileKey: "occupied-profile"}
	if _, err := queued.Enqueue(context.Background(), "queued-agent", "profile"); err == nil {
		t.Fatal("queued Enqueue succeeded without persisting its queue row")
	}
	if len(queued.queue) != 0 || queued.queued["queued-agent"] || len(queued.running) != 1 {
		t.Fatalf("failed queued admission left mutated state: running=%v queue=%v queued=%v", queued.running, queued.queue, queued.queued)
	}
	select {
	case <-executor.calls:
		t.Fatal("queued worker started despite queue persistence failure")
	default:
	}
}

func TestQueuePromotionSaveFailureDoesNotStartWorker(t *testing.T) {
	dir := t.TempDir()
	executor := &countingPreparationExecutor{calls: make(chan context.Context, 2)}
	queue := newTeamExecutionContext("team", executor, dir, nil)
	queue.running["occupied"] = runningEntry{ProfileKey: "occupied-profile"}
	if _, err := queue.Enqueue(context.Background(), "queued-agent", "queued-profile"); err != nil {
		t.Fatal(err)
	}
	if len(queue.queue) != 1 {
		t.Fatalf("expected one queued execution, got %d", len(queue.queue))
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir, []byte("block queue directory creation"), 0o600); err != nil {
		t.Fatal(err)
	}

	queue.OnMemberComplete("occupied")
	if len(queue.running) != 0 || len(queue.queue) != 1 || !queue.queued["queued-agent"] {
		t.Fatalf("failed promotion did not restore the queued admission: running=%v queue=%v", queue.running, queue.queue)
	}
	select {
	case <-executor.calls:
		t.Fatal("promoted worker started despite preparation persistence failure")
	default:
	}
}

func TestRecoverPreDispatchPreparationRetainsUntilFreshAdmission(t *testing.T) {
	dir := t.TempDir()
	queue := newTeamExecutionContext("team", &countingPreparationExecutor{calls: make(chan context.Context, 1)}, dir, nil)
	queue.running["agent"] = runningEntry{DispatchStage: DispatchStagePreparing, ProfileKey: "old-profile"}
	if err := queue.persistLocked(); err != nil {
		t.Fatal(err)
	}

	executor := &blockingPreparationExecutor{started: make(chan string, 1), release: make(chan struct{})}
	recovered := newTeamExecutionContext("team", executor, dir, nil)
	defer func() {
		close(executor.release)
		recovered.executionWG.Wait()
	}()
	recovered.Recover(context.Background())
	status := recovered.Status()
	if len(status.PreparationInterruptedAgentIDs) != 1 || status.PreparationInterruptedAgentIDs[0] != "agent" || len(status.RunningAgentIDs) != 0 {
		t.Fatalf("recovered preparation should be explicit and excluded from running: %#v", status)
	}
	select {
	case agentID := <-executor.started:
		t.Fatalf("recovery automatically dispatched %s without fresh eligibility/authority", agentID)
	default:
	}

	if _, err := recovered.Enqueue(context.Background(), "agent", "fresh-profile"); err != nil {
		t.Fatalf("fresh admission did not replace interrupted preparation: %v", err)
	}
	select {
	case agentID := <-executor.started:
		if agentID != "agent" {
			t.Fatalf("worker started for %q, want agent", agentID)
		}
	case <-time.After(time.Second):
		t.Fatal("fresh admission did not start the worker")
	}
	entry := recovered.running["agent"]
	if entry.State != "" || entry.DispatchStage != DispatchStagePreparing || entry.ProfileKey != "fresh-profile" {
		t.Fatalf("fresh admission did not atomically replace interrupted terms: %#v", entry)
	}
}

func TestFailedFreshAdmissionPreservesInterruptedPreparation(t *testing.T) {
	dir := t.TempDir()
	queue := newTeamExecutionContext("team", &countingPreparationExecutor{calls: make(chan context.Context, 1)}, dir, nil)
	queue.running["agent"] = runningEntry{DispatchStage: DispatchStagePreparing, ProfileKey: "old-profile"}
	if err := queue.persistLocked(); err != nil {
		t.Fatal(err)
	}
	executor := &countingPreparationExecutor{calls: make(chan context.Context, 1)}
	recovered := newTeamExecutionContext("team", executor, dir, nil)
	recovered.Recover(context.Background())
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir, []byte("block queue directory creation"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := recovered.Enqueue(context.Background(), "agent", "fresh-profile"); err == nil {
		t.Fatal("fresh admission succeeded without durably replacing the interrupted row")
	}
	entry, ok := recovered.running["agent"]
	if !ok || !isReplaceableInterruptedPreparation(entry) || entry.ProfileKey != "old-profile" {
		t.Fatalf("failed replacement did not restore the interrupted obligation: %#v exists=%v", entry, ok)
	}
	select {
	case <-executor.calls:
		t.Fatal("worker started despite replacement persistence failure")
	default:
	}
}

func TestSchedulerLeavesInterruptedPreparationWhenConfigDisabled(t *testing.T) {
	dir := t.TempDir()
	seed := newTeamExecutionContext("team", &countingPreparationExecutor{calls: make(chan context.Context, 1)}, dir, nil)
	seed.running["agent"] = runningEntry{DispatchStage: DispatchStagePreparing, ProfileKey: "old-profile"}
	if err := seed.persistLocked(); err != nil {
		t.Fatal(err)
	}
	executor := &captureExecutor{}
	teamExecStore := NewTeamExecutionStore(nil, executor, dir, nil)
	ctx := teamExecStore.GetOrCreate("team")
	ctx.Recover(context.Background())
	scheduler := NewScheduler(executor, nil, &stubConfigStore{config: &store.HeartbeatConfig{
		TeamID: "team", AgentID: "agent", Enabled: false, ProfileKey: "fresh-profile",
	}}, teamExecStore)
	scheduler.executeHeartbeat(context.Background(), "team", "agent")
	status := teamExecStore.Status("team")
	if len(status.PreparationInterruptedAgentIDs) != 1 || status.PreparationInterruptedAgentIDs[0] != "agent" || executor.callCount() != 0 {
		t.Fatalf("disabled schedule replaced interrupted work: status=%#v calls=%d", status, executor.callCount())
	}
}

func TestSchedulerReplacesInterruptedPreparationWithFreshEnabledConfig(t *testing.T) {
	dir := t.TempDir()
	seed := newTeamExecutionContext("team", &countingPreparationExecutor{calls: make(chan context.Context, 1)}, dir, nil)
	seed.running["agent"] = runningEntry{DispatchStage: DispatchStagePreparing, ProfileKey: "old-profile"}
	if err := seed.persistLocked(); err != nil {
		t.Fatal(err)
	}
	executor := &captureExecutor{}
	teamExecStore := NewTeamExecutionStore(nil, executor, dir, nil)
	teamExecStore.GetOrCreate("team").Recover(context.Background())
	scheduler := NewScheduler(executor, nil, &stubConfigStore{config: &store.HeartbeatConfig{
		TeamID: "team", AgentID: "agent", Enabled: true, ProfileKey: "current-profile",
	}}, teamExecStore)
	scheduler.executeHeartbeat(context.Background(), "team", "agent")
	deadline := time.Now().Add(time.Second)
	for executor.callCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if executor.callCount() != 1 {
		t.Fatalf("enabled current schedule did not replace the interrupted obligation: calls=%d status=%#v", executor.callCount(), teamExecStore.Status("team"))
	}
	if got := executor.callAt(0).profileKey; got != "current-profile" {
		t.Fatalf("scheduler reused stale profile %q; current config profile was not applied", got)
	}
}

func TestInterruptedPreparationReplacementRejectsOwnerTerms(t *testing.T) {
	valid := runningEntry{DispatchStage: DispatchStagePreparing, State: ObligationPreparationInterrupted}
	if !isReplaceableInterruptedPreparation(valid) {
		t.Fatal("exact pre-effect interrupted preparation should be replaceable")
	}
	for name, mutate := range map[string]func(*runningEntry){
		"task id":         func(entry *runningEntry) { entry.TaskID = "task" },
		"run id":          func(entry *runningEntry) { entry.RunID = "run" },
		"idempotency key": func(entry *runningEntry) { entry.IdempotencyKey = "key" },
		"create request":  func(entry *runningEntry) { entry.CreateRunRequestSnapshot = `{}` },
		"wrong state snapshot": func(entry *runningEntry) {
			entry.OwnerObligationID = "owner"
			entry.OwnerStateSnapshots = map[string]retainedOwnerStateSnapshot{
				ObligationNeedsReview: {OwnerObligationID: "owner", State: ObligationNeedsReview, ObservedAt: "2026-10-17T00:00:00Z"},
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			entry := valid
			mutate(&entry)
			if isReplaceableInterruptedPreparation(entry) {
				t.Fatalf("entry with %s evidence was replaceable", name)
			}
		})
	}
}

func TestRecoveredInterruptedPreparationRequiresMatchingFactSnapshot(t *testing.T) {
	item := queuedExecution{
		AgentID: "agent", ProfileKey: "profile", DispatchStage: DispatchStagePreparing,
		State: ObligationPreparationInterrupted, OwnerObligationID: "owner-obligation",
		OwnerStateSnapshots: map[string]retainedOwnerStateSnapshot{
			ObligationPreparationInterrupted: {
				OwnerObligationID: "owner-obligation", State: ObligationPreparationInterrupted,
				ObservedAt: "2026-10-17T01:02:03.123456789Z",
			},
		},
	}
	if !isProvenInterruptedPreparation(item) {
		t.Fatal("exact retained pre-effect fact should survive another restart")
	}
	item.OwnerStateSnapshots[ObligationPreparationInterrupted] = retainedOwnerStateSnapshot{
		OwnerObligationID: "different-owner", State: ObligationPreparationInterrupted,
		ObservedAt: "2026-10-17T01:02:03.123456789Z",
	}
	if isProvenInterruptedPreparation(item) {
		t.Fatal("mismatched retained-fact identity must remain held as ambiguous")
	}
}

func TestRecoverLegacyUnmarkedPreparationRemainsHeld(t *testing.T) {
	dir := t.TempDir()
	queue := newTeamExecutionContext("team", &captureExecutor{}, dir, nil)
	queue.running["legacy-agent"] = runningEntry{ProfileKey: "profile"}
	if err := queue.persistLocked(); err != nil {
		t.Fatal(err)
	}

	recovered := newTeamExecutionContext("team", &captureExecutor{}, dir, nil)
	recovered.Recover(context.Background())
	entry, ok := recovered.running["legacy-agent"]
	if !ok || entry.State != ObligationDispatchUncertain || entry.DispatchStage != "" {
		t.Fatalf("legacy row was not retained as ambiguous: %#v, exists=%v", entry, ok)
	}
}

func TestRecoverMalformedCanceledTombstoneRemainsHeld(t *testing.T) {
	dir := t.TempDir()
	queue := newTeamExecutionContext("team", &captureExecutor{}, dir, nil)
	queue.running["agent"] = runningEntry{
		DispatchStage:  DispatchStageIntent,
		State:          ObligationDispatchCanceled,
		TaskID:         "task-id",
		IdempotencyKey: "dispatch-key",
	}
	if err := queue.persistLocked(); err != nil {
		t.Fatal(err)
	}

	recovered := newTeamExecutionContext("team", &captureExecutor{}, dir, nil)
	recovered.Recover(context.Background())
	entry, ok := recovered.running["agent"]
	if !ok || entry.State != ObligationDispatchCanceled {
		t.Fatalf("incomplete cancellation tombstone must remain held, got %#v (exists=%v)", entry, ok)
	}
}
