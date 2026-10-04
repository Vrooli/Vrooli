// Responsibility: retain repository test declarations within their original package.
package database

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
	"testing"
	"time"
)

func TestRunCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create a task first (runs reference tasks)
	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Parent Task",
		ScopePath: "/test",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	run := &domain.Run{
		ID:              uuid.New(),
		TaskID:          task.ID,
		Tag:             "test-run",
		RunMode:         domain.RunModeSandboxed,
		Status:          domain.RunStatusPending,
		Phase:           domain.RunPhaseQueued,
		ProgressPercent: 0,
		ApprovalState:   domain.ApprovalStateNone,
	}

	// Create
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get
	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Tag != run.Tag {
		t.Errorf("expected tag %q, got %q", run.Tag, got.Tag)
	}
	if got.Status != domain.RunStatusPending {
		t.Errorf("expected status pending, got %q", got.Status)
	}
	if got.CanaryArm != "" {
		t.Errorf("expected empty canary arm to round-trip as an empty value, got %q", got.CanaryArm)
	}

	// List
	runs, err := repos.Runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: 10}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	// ListByTask
	runsByTask, err := repos.Runs.ListByTask(ctx, task.ID, repository.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListByTask: %v", err)
	}
	if len(runsByTask) != 1 {
		t.Errorf("expected 1 run for task, got %d", len(runsByTask))
	}

	// CountByStatus
	count, err := repos.Runs.CountByStatus(ctx, domain.RunStatusPending)
	if err != nil {
		t.Fatalf("CountByStatus: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Update
	startedAt := time.Now()
	run.Status = domain.RunStatusRunning
	run.StartedAt = &startedAt
	run.ProgressPercent = 50
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err = repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Status != domain.RunStatusRunning {
		t.Errorf("expected status running, got %q", got.Status)
	}
	if got.ProgressPercent != 50 {
		t.Errorf("expected progress 50, got %d", got.ProgressPercent)
	}

	// Delete
	if err := repos.Runs.Delete(ctx, run.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, err = repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

// TestRunCustomEnvRoundTrip proves Run.CustomEnv persists through the JSON
// column so the continue/wake path can re-inject it. Before Phase 0 there was
// no column at all, so a continued turn could never recover custom env.
func TestRunCustomEnvRoundTrip(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Parent Task",
		ScopePath: "/test",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	run := &domain.Run{
		ID:            uuid.New(),
		TaskID:        task.ID,
		RunMode:       domain.RunModeSandboxed,
		Status:        domain.RunStatusPending,
		Phase:         domain.RunPhaseQueued,
		ApprovalState: domain.ApprovalStateNone,
		CustomEnv: map[string]string{
			"VROOLI_SHADOW_SCENARIOS":         "agent-manager",
			"VROOLI_SWARM_MANAGER_SESSION_ID": "sess-42",
		},
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.CustomEnv) != 2 {
		t.Fatalf("expected 2 custom env entries, got %v", got.CustomEnv)
	}
	if got.CustomEnv["VROOLI_SHADOW_SCENARIOS"] != "agent-manager" ||
		got.CustomEnv["VROOLI_SWARM_MANAGER_SESSION_ID"] != "sess-42" {
		t.Errorf("custom env did not round-trip: %v", got.CustomEnv)
	}

	// Empty/nil custom env must decode as nil (not an empty map) so existing
	// rows and env-free runs behave identically.
	run.CustomEnv = nil
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.CustomEnv != nil {
		t.Errorf("expected nil custom env after clearing, got %v", got.CustomEnv)
	}
}

func TestRunAwaitHandleRoundTrip(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Await Task",
		ScopePath: "/test",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	deadline := time.Now().Add(30 * time.Minute).UTC().Truncate(time.Second)
	registered := time.Now().UTC().Truncate(time.Second)
	run := &domain.Run{
		ID:            uuid.New(),
		TaskID:        task.ID,
		RunMode:       domain.RunModeSandboxed,
		Status:        domain.RunStatusParked,
		Phase:         domain.RunPhaseExecuting,
		ApprovalState: domain.ApprovalStateNone,
		AwaitHandle: &domain.AwaitHandle{
			Producer:     "test-genie",
			Key:          "run-xyz",
			Deadline:     &deadline,
			RegisteredAt: registered,
		},
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != domain.RunStatusParked {
		t.Errorf("status did not round-trip: %s", got.Status)
	}
	if got.AwaitHandle == nil {
		t.Fatal("await handle did not round-trip (nil)")
	}
	if got.AwaitHandle.Producer != "test-genie" || got.AwaitHandle.Key != "run-xyz" {
		t.Errorf("await handle producer/key did not round-trip: %+v", got.AwaitHandle)
	}
	if got.AwaitHandle.Deadline == nil || !got.AwaitHandle.Deadline.Equal(deadline) {
		t.Errorf("await handle deadline did not round-trip: %v want %v", got.AwaitHandle.Deadline, deadline)
	}

	// Clearing the handle (wake/cancel) must persist as NULL → nil, exactly like
	// a non-parked run, so existing rows and woken runs behave identically.
	run.AwaitHandle = nil
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.AwaitHandle != nil {
		t.Errorf("expected nil await handle after clearing, got %+v", got.AwaitHandle)
	}
}

func TestTouchHeartbeat_StatusGuarded(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "HB Task",
		ScopePath: "/test",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	old := time.Now().Add(-time.Hour)
	mk := func(status domain.RunStatus) *domain.Run {
		run := &domain.Run{
			ID:            uuid.New(),
			TaskID:        task.ID,
			RunMode:       domain.RunModeInPlace,
			Status:        status,
			Phase:         domain.RunPhaseExecuting,
			ApprovalState: domain.ApprovalStateNone,
			LastHeartbeat: &old,
		}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatalf("Create run (%s): %v", status, err)
		}
		return run
	}

	// running → updated, last_heartbeat advances.
	running := mk(domain.RunStatusRunning)
	now := time.Now()
	updated, err := repos.Runs.TouchHeartbeat(ctx, running.ID, now)
	if err != nil {
		t.Fatalf("TouchHeartbeat(running): %v", err)
	}
	if !updated {
		t.Fatal("running run heartbeat should have been updated")
	}
	got, _ := repos.Runs.Get(ctx, running.ID)
	if got.LastHeartbeat == nil || got.LastHeartbeat.Before(now.Add(-2*time.Second)) {
		t.Errorf("running heartbeat not advanced: %v", got.LastHeartbeat)
	}
	if got.Status != domain.RunStatusRunning {
		t.Errorf("running status changed unexpectedly: %s", got.Status)
	}

	// starting → updated.
	starting := mk(domain.RunStatusStarting)
	if updated, err := repos.Runs.TouchHeartbeat(ctx, starting.ID, time.Now()); err != nil || !updated {
		t.Fatalf("TouchHeartbeat(starting): updated=%v err=%v (want true,nil)", updated, err)
	}

	// parked → NO-OP (the critical anti-clobber guarantee). Status stays parked,
	// heartbeat unchanged.
	parked := mk(domain.RunStatusParked)
	updated, err = repos.Runs.TouchHeartbeat(ctx, parked.ID, time.Now())
	if err != nil {
		t.Fatalf("TouchHeartbeat(parked): %v", err)
	}
	if updated {
		t.Fatal("parked run heartbeat must NOT be updated (would clobber the park)")
	}
	got, _ = repos.Runs.Get(ctx, parked.ID)
	if got.Status != domain.RunStatusParked {
		t.Errorf("parked status changed: %s", got.Status)
	}
	if got.LastHeartbeat == nil || !got.LastHeartbeat.Equal(old.UTC().Truncate(time.Nanosecond)) {
		// Heartbeat should be untouched (still ~old). Allow for storage truncation.
		if got.LastHeartbeat != nil && got.LastHeartbeat.After(old.Add(time.Minute)) {
			t.Errorf("parked heartbeat was advanced: %v", got.LastHeartbeat)
		}
	}

	// terminal (complete) → NO-OP.
	complete := mk(domain.RunStatusComplete)
	if updated, err := repos.Runs.TouchHeartbeat(ctx, complete.ID, time.Now()); err != nil || updated {
		t.Fatalf("TouchHeartbeat(complete): updated=%v err=%v (want false,nil)", updated, err)
	}
}

// TestRequestCancellation_StatusGuardedAndIdempotent pins the Phase 3 durable
// cancellation intent: a stop request stamps cancel_requested_at only while the
// run is still actively executing, a repeat request is a no-op (monotonic
// intent), and a terminal run is never stamped. The status guard is applied in
// SQL so a stale caller cannot resurrect or re-stamp a run.
func TestRequestCancellation_StatusGuardedAndIdempotent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Cancel Task",
		ScopePath: "/test",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	mk := func(status domain.RunStatus) *domain.Run {
		run := &domain.Run{
			ID:            uuid.New(),
			TaskID:        task.ID,
			RunMode:       domain.RunModeInPlace,
			Status:        status,
			Phase:         domain.RunPhaseExecuting,
			ApprovalState: domain.ApprovalStateNone,
		}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatalf("Create run (%s): %v", status, err)
		}
		return run
	}

	// running → intent recorded, status unchanged.
	running := mk(domain.RunStatusRunning)
	updated, err := repos.Runs.RequestCancellation(ctx, running.ID, time.Now())
	if err != nil {
		t.Fatalf("RequestCancellation(running): %v", err)
	}
	if !updated {
		t.Fatal("running run cancellation intent should have been recorded")
	}
	got, err := repos.Runs.Get(ctx, running.ID)
	if err != nil {
		t.Fatalf("Get(running): %v", err)
	}
	if got.CancelRequestedAt == nil {
		t.Fatal("running cancel intent was not persisted")
	}
	if got.Status != domain.RunStatusRunning {
		t.Errorf("cancellation intent changed status: %s", got.Status)
	}

	// duplicate → no-op, still stamped, status unchanged.
	if updated, err := repos.Runs.RequestCancellation(ctx, running.ID, time.Now()); err != nil || updated {
		t.Fatalf("duplicate RequestCancellation: updated=%v err=%v (want false,nil)", updated, err)
	}
	got, _ = repos.Runs.Get(ctx, running.ID)
	if got.CancelRequestedAt == nil {
		t.Fatal("duplicate request cleared the cancellation intent")
	}
	if got.Status != domain.RunStatusRunning {
		t.Errorf("duplicate request changed status: %s", got.Status)
	}

	// starting → intent recorded.
	starting := mk(domain.RunStatusStarting)
	if updated, err := repos.Runs.RequestCancellation(ctx, starting.ID, time.Now()); err != nil || !updated {
		t.Fatalf("RequestCancellation(starting): updated=%v err=%v (want true,nil)", updated, err)
	}

	// terminal (complete) → NO-OP: a terminal run is never stamped.
	complete := mk(domain.RunStatusComplete)
	if updated, err := repos.Runs.RequestCancellation(ctx, complete.ID, time.Now()); err != nil || updated {
		t.Fatalf("RequestCancellation(complete): updated=%v err=%v (want false,nil)", updated, err)
	}
	got, _ = repos.Runs.Get(ctx, complete.ID)
	if got.CancelRequestedAt != nil {
		t.Errorf("terminal run was stamped with cancellation intent: %v", got.CancelRequestedAt)
	}
}

// TestUpdateRunnerStreamState_StatusGuarded pins the park-clobber fix: the
// runner's transcript callbacks (OnAdvance/OnProcessStart/OnSessionID) persist
// streaming columns on every agent output chunk from an in-memory run whose
// Status is the stale "running". During the park turn-end grace the agent keeps
// emitting output; a full-row write would rewrite a just-persisted
// running→parked transition back to running (clobbering the park before
// detectParked reads it). UpdateRunnerStreamState must be status-guarded so a
// parked/terminal run is never resurrected. (Found by the local-inference
// park/resume e2e; the prior detectParked-only guard was structurally too late.)
func TestUpdateRunnerStreamState_StatusGuarded(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Stream Task",
		ScopePath: "/test",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	mk := func(status domain.RunStatus) *domain.Run {
		run := &domain.Run{
			ID:            uuid.New(),
			TaskID:        task.ID,
			RunMode:       domain.RunModeSandboxed,
			Status:        status,
			Phase:         domain.RunPhaseExecuting,
			ApprovalState: domain.ApprovalStateNone,
		}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatalf("Create run (%s): %v", status, err)
		}
		return run
	}

	// running → streaming write applies (cursor persists, status unchanged).
	running := mk(domain.RunStatusRunning)
	running.TranscriptCursor = 42
	running.SessionID = "sess-running"
	updated, err := repos.Runs.UpdateRunnerStreamState(ctx, running)
	if err != nil {
		t.Fatalf("UpdateRunnerStreamState(running): %v", err)
	}
	if !updated {
		t.Fatal("running run streaming state should have been updated")
	}
	got, _ := repos.Runs.Get(ctx, running.ID)
	if got.TranscriptCursor != 42 || got.SessionID != "sess-running" {
		t.Errorf("running streaming state not persisted: cursor=%d session=%q", got.TranscriptCursor, got.SessionID)
	}
	if got.Status != domain.RunStatusRunning {
		t.Errorf("running status changed unexpectedly: %s", got.Status)
	}

	// parked → NO-OP (the critical anti-clobber guarantee). An in-flight
	// transcript callback must not resurrect the park. Status stays parked and
	// the streaming columns are NOT written.
	parked := mk(domain.RunStatusParked)
	parked.TranscriptCursor = 99
	parked.SessionID = "should-not-persist"
	updated, err = repos.Runs.UpdateRunnerStreamState(ctx, parked)
	if err != nil {
		t.Fatalf("UpdateRunnerStreamState(parked): %v", err)
	}
	if updated {
		t.Fatal("parked run streaming state must NOT be updated (would clobber the park)")
	}
	got, _ = repos.Runs.Get(ctx, parked.ID)
	if got.Status != domain.RunStatusParked {
		t.Errorf("parked status was clobbered to %s (the bug)", got.Status)
	}
	if got.TranscriptCursor == 99 || got.SessionID == "should-not-persist" {
		t.Errorf("parked streaming columns were written: cursor=%d session=%q", got.TranscriptCursor, got.SessionID)
	}

	// terminal (failed) → NO-OP.
	failed := mk(domain.RunStatusFailed)
	if updated, err := repos.Runs.UpdateRunnerStreamState(ctx, failed); err != nil || updated {
		t.Fatalf("UpdateRunnerStreamState(failed): updated=%v err=%v (want false,nil)", updated, err)
	}
}

// TestRunLastAwaitFieldsRoundTrip verifies the durable re-fetch SSOT + re-park
// guard bookkeeping (last_await_key/result/resolved_at, last_wake_seq,
// same_key_park_streak) persist and reload through Create/Update/Get.
func TestRunLastAwaitFieldsRoundTrip(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{ID: uuid.New(), Title: "Await Task", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	resolved := time.Now().UTC().Truncate(time.Second)
	run := &domain.Run{
		ID:                  uuid.New(),
		TaskID:              task.ID,
		RunMode:             domain.RunModeSandboxed,
		Status:              domain.RunStatusRunning,
		Phase:               domain.RunPhaseExecuting,
		ApprovalState:       domain.ApprovalStateNone,
		LastAwaitKey:        "git-control-tower:agent-manager/am-park-resume",
		LastAwaitResult:     `{"status":"ready"}`,
		LastAwaitResolvedAt: &resolved,
		LastWakeSeq:         12,
		SameKeyParkStreak:   1,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.LastAwaitKey != run.LastAwaitKey || got.LastAwaitResult != run.LastAwaitResult {
		t.Errorf("await key/result not persisted: key=%q result=%q", got.LastAwaitKey, got.LastAwaitResult)
	}
	if got.LastWakeSeq != 12 || got.SameKeyParkStreak != 1 {
		t.Errorf("guard counters not persisted: wakeSeq=%d streak=%d", got.LastWakeSeq, got.SameKeyParkStreak)
	}
	if got.LastAwaitResolvedAt == nil || !got.LastAwaitResolvedAt.Equal(resolved) {
		t.Errorf("resolved_at not persisted: %v want %v", got.LastAwaitResolvedAt, resolved)
	}

	// Update mutates the fields (next await resolves) — reload reflects it.
	got.SameKeyParkStreak = 0
	got.LastAwaitResult = `{"status":"done"}`
	if err := repos.Runs.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	reloaded, _ := repos.Runs.Get(ctx, run.ID)
	if reloaded.SameKeyParkStreak != 0 || reloaded.LastAwaitResult != `{"status":"done"}` {
		t.Errorf("update did not persist: streak=%d result=%q", reloaded.SameKeyParkStreak, reloaded.LastAwaitResult)
	}
}

func TestRunListFilters(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create a task
	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Filter Task",
		ScopePath: "/test",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	// Create a profile
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "filter-profile",
		ProfileKey: "filter-profile", RoleRef: "code.default",
	}
	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatalf("Create profile: %v", err)
	}

	// Create runs with different tags and statuses
	sourceRunID := uuid.New()
	investigationRunID := uuid.New()
	runs := []*domain.Run{
		{ID: uuid.New(), TaskID: task.ID, AgentProfileID: &profile.ID, Tag: "batch-1", Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued, ApprovalState: domain.ApprovalStateNone, IdempotencyKey: "filter-key-1", SourceRunIDs: []uuid.UUID{sourceRunID}},
		{ID: uuid.New(), TaskID: task.ID, Tag: "batch-2", Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone, IdempotencyKey: "filter-key-2", SourceInvestigationRunID: &investigationRunID},
		{ID: uuid.New(), TaskID: task.ID, Tag: "batch-1-sub", Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, ApprovalState: domain.ApprovalStateNone, IdempotencyKey: "filter-key-3"},
	}
	for _, run := range runs {
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatalf("Create run: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Filter by status
	completedRuns, err := repos.Runs.List(ctx, repository.RunListFilter{
		Status: func() *domain.RunStatus { s := domain.RunStatusComplete; return &s }(),
	})
	if err != nil {
		t.Fatalf("List by status: %v", err)
	}
	if len(completedRuns) != 1 {
		t.Errorf("expected 1 completed run, got %d", len(completedRuns))
	}

	// Filter by tag prefix
	batchRuns, err := repos.Runs.List(ctx, repository.RunListFilter{
		TagPrefix: "batch-1",
	})
	if err != nil {
		t.Fatalf("List by tag prefix: %v", err)
	}
	if len(batchRuns) != 2 {
		t.Errorf("expected 2 runs with tag prefix 'batch-1', got %d", len(batchRuns))
	}

	// Filter by profile ID
	profileRuns, err := repos.Runs.List(ctx, repository.RunListFilter{
		AgentProfileID: &profile.ID,
	})
	if err != nil {
		t.Fatalf("List by profile: %v", err)
	}
	if len(profileRuns) != 1 {
		t.Errorf("expected 1 run with profile, got %d", len(profileRuns))
	}

	// Filter by source run ID lineage
	investigationRuns, err := repos.Runs.List(ctx, repository.RunListFilter{
		InvestigatesRunID: &sourceRunID,
	})
	if err != nil {
		t.Fatalf("List by source run ID: %v", err)
	}
	if len(investigationRuns) != 1 {
		t.Errorf("expected 1 investigation run for source run ID, got %d", len(investigationRuns))
	}

	// Filter by source investigation run ID lineage
	applyRuns, err := repos.Runs.List(ctx, repository.RunListFilter{
		AppliesInvestigationRunID: &investigationRunID,
	})
	if err != nil {
		t.Fatalf("List by source investigation run ID: %v", err)
	}
	if len(applyRuns) != 1 {
		t.Errorf("expected 1 apply run for source investigation run ID, got %d", len(applyRuns))
	}
}

// ============================================================================
// Event Repository Tests
// ============================================================================
