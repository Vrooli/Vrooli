package database

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

func TestRunUpdateCannotChangeCreationWorkReferences(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	task := createQueuedTask(t, repos, "immutable run identity")
	originalRef := activeEffortOrchestratorRef("effort:original", "orchestrator")
	run := pendingTaskRun(task.ID)
	run.WorkReferences = []*eventpb.WorkReference{originalRef}
	if err := repos.Runs.CreateTaskAttempt(ctx, task, task, run, nil); err != nil {
		t.Fatalf("create run: %v", err)
	}
	stored, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || stored == nil {
		t.Fatalf("read created run: run=%+v err=%v", stored, err)
	}

	// Keep the same active status so the lifecycle version does not change.
	// Generic Update must still reject an attempt to replace creation identity.
	stored.WorkReferences = []*eventpb.WorkReference{
		activeEffortOrchestratorRef("effort:replacement", "orchestrator"),
	}
	err = repos.Runs.Update(ctx, stored)
	var stateErr *domain.StateError
	if !errors.As(err, &stateErr) || stateErr.CurrentState != "immutable_identity" {
		t.Fatalf("Update error = %v, want immutable_identity state error", err)
	}

	after, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || after == nil {
		t.Fatalf("read run after rejected update: run=%+v err=%v", after, err)
	}
	if after.Status != domain.RunStatusPending || len(after.WorkReferences) != 1 || after.WorkReferences[0].GetId() != originalRef.GetId() {
		t.Fatalf("rejected update changed persisted identity or status: status=%s refs=%+v", after.Status, after.WorkReferences)
	}
}

func TestQueuedBriefMutationAndAttemptAdmissionFenceBothOrderings(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	creator := uuid.NewString()

	makeOwnedQueued := func(title string) *domain.Task {
		t.Helper()
		task := &domain.Task{ID: uuid.New(), Title: title, Description: "Initial brief", ScopePath: ".", Status: domain.TaskStatusQueued, CreatedBy: creator}
		if err := repos.Tasks.Create(ctx, task); err != nil {
			t.Fatalf("create queued task: %v", err)
		}
		stored, err := repos.Tasks.Get(ctx, task.ID)
		if err != nil || stored == nil {
			t.Fatalf("read queued task: task=%+v err=%v", stored, err)
		}
		return stored
	}

	t.Run("brief update wins and stale admission must reload", func(t *testing.T) {
		task := makeOwnedQueued("brief-first")
		staleSnapshot := *task
		changed, err := repos.Tasks.UpdateQueuedBrief(ctx, task.ID, creator, staleSnapshot.Title, staleSnapshot.Description, "Corrected title", "Corrected brief")
		if err != nil || !changed {
			t.Fatalf("queued brief CAS changed=%v err=%v; want success", changed, err)
		}

		staleRun := pendingTaskRun(task.ID)
		if err := repos.Runs.CreateTaskAttempt(ctx, &staleSnapshot, &staleSnapshot, staleRun, nil); err == nil {
			t.Fatal("stale task snapshot admitted after brief update")
		}
		missing, err := repos.Runs.Get(ctx, staleRun.ID)
		if err != nil || missing != nil {
			t.Fatalf("stale admission wrote a run: run=%+v err=%v", missing, err)
		}

		fresh, err := repos.Tasks.Get(ctx, task.ID)
		if err != nil || fresh == nil || fresh.Title != "Corrected title" || fresh.Description != "Corrected brief" || fresh.Status != domain.TaskStatusQueued {
			t.Fatalf("read corrected queued task: task=%+v err=%v", fresh, err)
		}
		freshRun := pendingTaskRun(fresh.ID)
		if err := repos.Runs.CreateTaskAttempt(ctx, fresh, fresh, freshRun, nil); err != nil {
			t.Fatalf("admit refreshed task snapshot: %v", err)
		}
		admitted, err := repos.Runs.Get(ctx, freshRun.ID)
		if err != nil || admitted == nil || admitted.Status != domain.RunStatusPending {
			t.Fatalf("fresh admission was not preserved: run=%+v err=%v", admitted, err)
		}
	})

	t.Run("prompt context changes invalidate prepared task snapshot", func(t *testing.T) {
		task := makeOwnedQueued("context-fence")
		prepared := *task
		updated := *task
		updated.ContextAttachments = []domain.ContextAttachment{{Type: "note", Content: "new context"}}
		if err := repos.Tasks.Update(ctx, &updated); err != nil {
			t.Fatalf("update queued context: %v", err)
		}
		run := pendingTaskRun(task.ID)
		if err := repos.Runs.CreateTaskAttempt(ctx, &prepared, &prepared, run, nil); err == nil {
			t.Fatal("admitted a prompt prepared from stale context attachments")
		}
		missing, err := repos.Runs.Get(ctx, run.ID)
		if err != nil || missing != nil {
			t.Fatalf("stale context admission wrote a run: run=%+v err=%v", missing, err)
		}
	})

	t.Run("dispatch wins and brief update is refused", func(t *testing.T) {
		task := makeOwnedQueued("dispatch-first")
		run := pendingTaskRun(task.ID)
		if err := repos.Runs.CreateTaskAttempt(ctx, task, task, run, nil); err != nil {
			t.Fatalf("admit task attempt: %v", err)
		}
		changed, err := repos.Tasks.UpdateQueuedBrief(ctx, task.ID, creator, task.Title, task.Description, task.Title, "Too late")
		if err != nil || changed {
			t.Fatalf("queued brief CAS after dispatch changed=%v err=%v; want a clean refusal", changed, err)
		}
		stored, err := repos.Tasks.Get(ctx, task.ID)
		if err != nil || stored == nil {
			t.Fatalf("read dispatched task: task=%+v err=%v", stored, err)
		}
		if stored.Description != "Initial brief" || stored.Status != domain.TaskStatusRunning || stored.CurrentRunID == nil || *stored.CurrentRunID != run.ID {
			t.Fatalf("refused brief update changed admitted task: %+v", stored)
		}
	})
}

func activeEffortOrchestratorRef(id, role string) *eventpb.WorkReference {
	return &eventpb.WorkReference{
		Kind:         "effort",
		Id:           id,
		Revision:     "source-r1",
		Relationship: role,
		Verified:     true,
		Visibility:   eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC,
		State:        eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE,
	}
}

func createQueuedTask(t *testing.T, repos *Repositories, title string) *domain.Task {
	t.Helper()
	task := &domain.Task{ID: uuid.New(), Title: title, ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(context.Background(), task); err != nil {
		t.Fatalf("create task %q: %v", title, err)
	}
	return task
}

func setRunStatus(t *testing.T, repos *Repositories, run *domain.Run, status domain.RunStatus) *domain.Run {
	t.Helper()
	run.Status = status
	run.UpdatedAt = time.Now().UTC()
	if err := repos.Runs.Update(context.Background(), run); err != nil {
		t.Fatalf("update run %s to %s: %v", run.ID, status, err)
	}
	stored, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil || stored == nil {
		t.Fatalf("read run %s after status update: run=%v err=%v", run.ID, stored, err)
	}
	return stored
}

func TestEffortOrchestratorAdmissionCoordinatesCreateAndContinue(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	const effortID = "effort:shared-owner"

	// A worker for the same effort does not claim the coordinator slot.
	workerTask := createQueuedTask(t, repos, "effort worker")
	worker := pendingTaskRun(workerTask.ID)
	worker.WorkReferences = []*eventpb.WorkReference{activeEffortOrchestratorRef(effortID, "worker")}
	if err := repos.Runs.CreateTaskAttempt(ctx, workerTask, workerTask, worker, nil); err != nil {
		t.Fatalf("admit effort worker: %v", err)
	}

	ownerTask := createQueuedTask(t, repos, "original orchestrator")
	owner := pendingTaskRun(ownerTask.ID)
	owner.WorkReferences = []*eventpb.WorkReference{activeEffortOrchestratorRef(effortID, "orchestrator")}
	if err := repos.Runs.CreateTaskAttempt(ctx, ownerTask, ownerTask, owner, nil); err != nil {
		t.Fatalf("admit original orchestrator beside worker: %v", err)
	}

	newTask := createQueuedTask(t, repos, "new orchestrator")
	newOwner := pendingTaskRun(newTask.ID)
	newOwner.WorkReferences = []*eventpb.WorkReference{activeEffortOrchestratorRef(effortID, "orchestrator")}
	if err := repos.Runs.CreateTaskAttempt(ctx, newTask, newTask, newOwner, nil); err == nil {
		t.Fatal("new orchestrator was admitted while the original owner was active")
	}
	newTask, err := repos.Tasks.Get(ctx, newTask.ID)
	if err != nil || newTask.CurrentRunID != nil || newTask.Status != domain.TaskStatusQueued {
		t.Fatalf("refused create changed the candidate Task: task=%+v err=%v", newTask, err)
	}

	owner = setRunStatus(t, repos, owner, domain.RunStatusRunning)
	owner = setRunStatus(t, repos, owner, domain.RunStatusParked)
	if err := repos.Runs.CreateTaskAttempt(ctx, newTask, newTask, newOwner, nil); err == nil {
		t.Fatal("new orchestrator was admitted while the original owner was parked")
	}

	// The original run may continue itself while it owns the effort. The exact
	// run transition is checked in the same transaction as its reactivation.
	continued, err := repos.Runs.AdmitCurrentRun(ctx, owner, domain.RunStatusParked, time.Now().UTC())
	if err != nil || !continued {
		t.Fatalf("same-owner continuation admission = %v, err=%v", continued, err)
	}

	otherTask := createQueuedTask(t, repos, "different effort orchestrator")
	other := pendingTaskRun(otherTask.ID)
	other.WorkReferences = []*eventpb.WorkReference{activeEffortOrchestratorRef("effort:independent", "orchestrator")}
	if err := repos.Runs.CreateTaskAttempt(ctx, otherTask, otherTask, other, nil); err != nil {
		t.Fatalf("distinct effort admission was blocked: %v", err)
	}

	// A failed attempt does not reserve a slot or poison the Task. Once the
	// previous owner is terminal, ordinary creation may claim it.
	owner = setRunStatus(t, repos, owner, domain.RunStatusComplete)
	newOwner = pendingTaskRun(newTask.ID)
	newOwner.WorkReferences = []*eventpb.WorkReference{activeEffortOrchestratorRef(effortID, "orchestrator")}
	if err := repos.Runs.CreateTaskAttempt(ctx, newTask, newTask, newOwner, nil); err != nil {
		t.Fatalf("create after terminal owner: %v", err)
	}
}

func TestEffortOrchestratorCreateRacesTerminalOwnerUpdateAtomically(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	const effortID = "effort:admission-race"

	ownerTask := createQueuedTask(t, repos, "terminal orchestrator")
	owner := pendingTaskRun(ownerTask.ID)
	owner.WorkReferences = []*eventpb.WorkReference{activeEffortOrchestratorRef(effortID, "orchestrator")}
	if err := repos.Runs.CreateTaskAttempt(ctx, ownerTask, ownerTask, owner, nil); err != nil {
		t.Fatal(err)
	}
	owner = setRunStatus(t, repos, owner, domain.RunStatusComplete)

	newTask := createQueuedTask(t, repos, "racing orchestrator")
	newOwner := pendingTaskRun(newTask.ID)
	newOwner.WorkReferences = []*eventpb.WorkReference{activeEffortOrchestratorRef(effortID, "orchestrator")}

	start := make(chan struct{})
	created := make(chan error, 1)
	continued := make(chan struct {
		err error
	}, 1)
	go func() {
		<-start
		created <- repos.Runs.CreateTaskAttempt(ctx, newTask, newTask, newOwner, nil)
	}()
	go func() {
		<-start
		owner.Status = domain.RunStatusRunning
		err := repos.Runs.Update(ctx, owner)
		continued <- struct {
			err error
		}{err: err}
	}()
	close(start)
	createErr := <-created
	continueResult := <-continued
	createAdmitted := createErr == nil
	continueAdmitted := continueResult.err == nil
	if createAdmitted == continueAdmitted {
		t.Fatalf("same-effort admissions did not select exactly one owner: createErr=%v continueResult=%+v", createErr, continueResult)
	}

	var err error
	if createAdmitted {
		owner, err = repos.Runs.Get(ctx, owner.ID)
		if err != nil || owner.Status != domain.RunStatusComplete {
			t.Fatalf("losing continuation changed original owner: run=%+v err=%v", owner, err)
		}
		newOwner, err = repos.Runs.Get(ctx, newOwner.ID)
		if err != nil || newOwner == nil || newOwner.Status != domain.RunStatusPending {
			t.Fatalf("winning create was not durably recorded: run=%+v err=%v", newOwner, err)
		}
	} else {
		if continueResult.err != nil {
			t.Fatalf("both admissions failed: createErr=%v continuationErr=%v", createErr, continueResult.err)
		}
		owner, err = repos.Runs.Get(ctx, owner.ID)
		if err != nil || owner.Status != domain.RunStatusRunning {
			t.Fatalf("winning continuation was not durably recorded: run=%+v err=%v", owner, err)
		}
		newTask, err = repos.Tasks.Get(ctx, newTask.ID)
		if err != nil || newTask.CurrentRunID != nil || newTask.Status != domain.TaskStatusQueued {
			t.Fatalf("losing create changed candidate Task: task=%+v err=%v", newTask, err)
		}
	}
}

func TestEffortOrchestratorCreateRacesTerminalOwnerAdmissionAtomically(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	const effortID = "effort:admission-api-race"

	ownerTask := createQueuedTask(t, repos, "terminal orchestrator")
	owner := pendingTaskRun(ownerTask.ID)
	owner.WorkReferences = []*eventpb.WorkReference{activeEffortOrchestratorRef(effortID, "orchestrator")}
	if err := repos.Runs.CreateTaskAttempt(ctx, ownerTask, ownerTask, owner, nil); err != nil {
		t.Fatal(err)
	}
	owner = setRunStatus(t, repos, owner, domain.RunStatusComplete)

	newTask := createQueuedTask(t, repos, "racing orchestrator")
	newOwner := pendingTaskRun(newTask.ID)
	newOwner.WorkReferences = []*eventpb.WorkReference{activeEffortOrchestratorRef(effortID, "orchestrator")}
	admitOwner := *owner
	start := make(chan struct{})
	created := make(chan error, 1)
	continued := make(chan struct {
		admitted bool
		err      error
	}, 1)
	go func() {
		<-start
		created <- repos.Runs.CreateTaskAttempt(ctx, newTask, newTask, newOwner, nil)
	}()
	go func() {
		<-start
		admitted, err := repos.Runs.AdmitCurrentRun(ctx, &admitOwner, domain.RunStatusComplete, time.Now().UTC())
		continued <- struct {
			admitted bool
			err      error
		}{admitted: admitted, err: err}
	}()
	close(start)
	createErr := <-created
	continueResult := <-continued
	createAdmitted := createErr == nil
	continueAdmitted := continueResult.err == nil && continueResult.admitted
	if createAdmitted == continueAdmitted {
		t.Fatalf("same-effort admissions did not select exactly one owner: createErr=%v continueResult=%+v", createErr, continueResult)
	}
	if continueResult.err != nil {
		t.Fatalf("continuation admission errored: %v", continueResult.err)
	}

	if createAdmitted {
		stored, err := repos.Runs.Get(ctx, owner.ID)
		if err != nil || stored.Status != domain.RunStatusComplete {
			t.Fatalf("losing continuation changed original owner: run=%+v err=%v", stored, err)
		}
		newTask, err = repos.Tasks.Get(ctx, newTask.ID)
		if err != nil || newTask.CurrentRunID == nil || *newTask.CurrentRunID != newOwner.ID {
			t.Fatalf("winning create was not durably recorded: task=%+v err=%v", newTask, err)
		}
	} else {
		stored, err := repos.Runs.Get(ctx, owner.ID)
		if err != nil || stored.Status != domain.RunStatusRunning {
			t.Fatalf("winning continuation was not durably recorded: run=%+v err=%v", stored, err)
		}
		newTask, err = repos.Tasks.Get(ctx, newTask.ID)
		if err != nil || newTask.CurrentRunID != nil || newTask.Status != domain.TaskStatusQueued {
			t.Fatalf("losing create changed candidate Task: task=%+v err=%v", newTask, err)
		}
	}
}

func TestEffortOrchestratorContinuationWaitsForWALWriterWithoutReadUpgrade(t *testing.T) {
	tmpDir := t.TempDir()
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", filepath.Join(tmpDir, "admission.db"))
	sqlDB, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(4)
	defer sqlDB.Close()
	db := &DB{DB: sqlDB, log: logrus.New()}
	if err := db.initSchema(); err != nil {
		t.Fatal(err)
	}
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	task := createQueuedTask(t, repos, "WAL continuation")
	owner := pendingTaskRun(task.ID)
	owner.WorkReferences = []*eventpb.WorkReference{activeEffortOrchestratorRef("effort:wal-continuation", "orchestrator")}
	if err := repos.Runs.CreateTaskAttempt(ctx, task, task, owner, nil); err != nil {
		t.Fatal(err)
	}
	owner = setRunStatus(t, repos, owner, domain.RunStatusComplete)

	blocker, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := blocker.ExecContext(ctx, `UPDATE tasks SET updated_at = updated_at WHERE id = ?`, task.ID); err != nil {
		_ = blocker.Rollback()
		t.Fatal(err)
	}
	owner.Status = domain.RunStatusRunning
	continued := make(chan error, 1)
	go func() { continued <- repos.Runs.Update(ctx, owner) }()
	time.Sleep(150 * time.Millisecond)
	select {
	case err := <-continued:
		_ = blocker.Rollback()
		t.Fatalf("continuation completed while WAL writer was held: %v", err)
	default:
	}
	if err := blocker.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-continued; err != nil {
		t.Fatalf("continuation failed after WAL writer committed: %v", err)
	}
	stored, err := repos.Runs.Get(ctx, owner.ID)
	if err != nil || stored.Status != domain.RunStatusRunning {
		t.Fatalf("continued owner = %+v, err=%v", stored, err)
	}
}

func TestTaskAttemptAdmissionProjectsExactCurrentRunAndFencesOldOutcome(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "attempt lifecycle", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	staleEdit, err := repos.Tasks.Get(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	first := pendingTaskRun(task.ID)
	if err := repos.Runs.CreateTaskAttempt(ctx, task, task, first, nil); err != nil {
		t.Fatalf("admit first attempt: %v", err)
	}
	gotTask, err := repos.Tasks.Get(ctx, task.ID)
	if err != nil || gotTask.Status != domain.TaskStatusRunning || gotTask.CurrentRunID == nil || *gotTask.CurrentRunID != first.ID {
		t.Fatalf("first attempt projection = %+v, err=%v", gotTask, err)
	}
	staleEdit.Title = "edited while attempt starts"
	if err := repos.Tasks.Update(ctx, staleEdit); err != nil {
		t.Fatal(err)
	}
	gotTask, err = repos.Tasks.Get(ctx, task.ID)
	if err != nil || gotTask.Status != domain.TaskStatusRunning || gotTask.CurrentRunID == nil || *gotTask.CurrentRunID != first.ID {
		t.Fatalf("stale content edit overwrote lifecycle state: task=%+v err=%v", gotTask, err)
	}
	if err := repos.Runs.CreateTaskAttempt(ctx, task, task, pendingTaskRun(task.ID), gotTask.CurrentRunID); err == nil {
		t.Fatal("second attempt admitted while current attempt is active")
	}
	first.Status = domain.RunStatusRunning
	if err := repos.Runs.Update(ctx, first); err != nil {
		t.Fatal(err)
	}
	first.Status = domain.RunStatusNeedsReview
	if err := repos.Runs.Update(ctx, first); err != nil {
		t.Fatal(err)
	}
	gotTask, err = repos.Tasks.Get(ctx, task.ID)
	if err != nil || gotTask.Status != domain.TaskStatusNeedsReview {
		t.Fatalf("review projection = %+v, err=%v", gotTask, err)
	}
	first.Status = domain.RunStatusComplete
	first.ApprovalState = domain.ApprovalStateApproved
	first.ApprovedBy = "applyAtRunEnd"
	if err := repos.Runs.Update(ctx, first); err != nil {
		t.Fatalf("complete first attempt: %v", err)
	}
	gotTask, err = repos.Tasks.Get(ctx, task.ID)
	if err != nil || gotTask.Status != domain.TaskStatusCompleted {
		t.Fatalf("automatic apply was projected as task acceptance or not completed: task=%+v err=%v", gotTask, err)
	}
	first.Status = domain.RunStatusRunning // Same exact run can continue.
	if err := repos.Runs.Update(ctx, first); err != nil {
		t.Fatal(err)
	}
	gotTask, err = repos.Tasks.Get(ctx, task.ID)
	if err != nil || gotTask.Status != domain.TaskStatusRunning || gotTask.CurrentRunID == nil || *gotTask.CurrentRunID != first.ID {
		t.Fatalf("same-run continuation lost exact task ownership: task=%+v err=%v", gotTask, err)
	}
	first.Status = domain.RunStatusComplete
	if err := repos.Runs.Update(ctx, first); err != nil {
		t.Fatal(err)
	}
	gotTask, err = repos.Tasks.Get(ctx, task.ID)
	if err != nil || gotTask.Status != domain.TaskStatusCompleted {
		t.Fatalf("recompleted task projection = %+v, err=%v", gotTask, err)
	}
	second := pendingTaskRun(task.ID)
	if err := repos.Runs.CreateTaskAttempt(ctx, gotTask, gotTask, second, gotTask.CurrentRunID); err != nil {
		t.Fatalf("admit serial attempt: %v", err)
	}
	first.Status = domain.RunStatusFailed // A delayed prior-attempt outcome.
	if err := repos.Runs.Update(ctx, first); err != nil {
		t.Fatalf("persist old attempt outcome: %v", err)
	}
	gotTask, err = repos.Tasks.Get(ctx, task.ID)
	if err != nil || gotTask.Status != domain.TaskStatusRunning || gotTask.CurrentRunID == nil || *gotTask.CurrentRunID != second.ID {
		t.Fatalf("stale first attempt changed current projection: task=%+v err=%v", gotTask, err)
	}
}

func TestTaskAttemptAdmissionAllowsUnambiguousLegacyHistoryOnly(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	makeTask := func() *domain.Task {
		task := &domain.Task{ID: uuid.New(), Title: "legacy task", ScopePath: ".", Status: domain.TaskStatusQueued}
		if err := repos.Tasks.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
		return task
	}
	terminal := makeTask()
	old := pendingTaskRun(terminal.ID)
	if err := repos.Runs.Create(ctx, old); err != nil {
		t.Fatal(err)
	}
	old.Status = domain.RunStatusRunning
	if err := repos.Runs.Update(ctx, old); err != nil {
		t.Fatal(err)
	}
	old.Status = domain.RunStatusComplete
	if err := repos.Runs.Update(ctx, old); err != nil {
		t.Fatal(err)
	}
	if err := repos.Runs.CreateTaskAttempt(ctx, terminal, terminal, pendingTaskRun(terminal.ID), nil); err != nil {
		t.Fatalf("new explicit attempt should not need to select legacy terminal history: %v", err)
	}

	active := makeTask()
	if err := repos.Runs.Create(ctx, pendingTaskRun(active.ID)); err != nil {
		t.Fatal(err)
	}
	if err := repos.Runs.CreateTaskAttempt(ctx, active, active, pendingTaskRun(active.ID), nil); err == nil {
		t.Fatal("admitted a task with an unowned nonterminal historical run")
	}
	rows, err := repos.Runs.ListByTask(ctx, active.ID, repository.ListFilter{Limit: 10})
	if err != nil || len(rows) != 1 {
		t.Fatalf("ambiguous legacy task gained a run: count=%d err=%v", len(rows), err)
	}
}

func TestConcurrentTaskAttemptAdmissionsHaveOneExactWinner(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "concurrent admission", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	type result struct {
		runID uuid.UUID
		err   error
	}
	results := make(chan result, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			run := pendingTaskRun(task.ID)
			ready.Done()
			<-start
			err := repos.Runs.CreateTaskAttempt(ctx, task, task, run, nil)
			results <- result{runID: run.ID, err: err}
		}()
	}
	ready.Wait()
	close(start)
	first, second := <-results, <-results
	winners := 0
	var winner uuid.UUID
	for _, outcome := range []result{first, second} {
		if outcome.err == nil {
			winners++
			winner = outcome.runID
		}
	}
	if winners != 1 {
		t.Fatalf("concurrent admission winners=%d, outcomes=(%v, %v)", winners, first.err, second.err)
	}
	if task.CurrentRunID != nil || task.Status != domain.TaskStatusQueued {
		t.Fatalf("admission mutated caller's task: %+v", task)
	}
	got, err := repos.Tasks.Get(ctx, task.ID)
	if err != nil || got.CurrentRunID == nil || *got.CurrentRunID != winner || got.Status != domain.TaskStatusRunning {
		t.Fatalf("Task projection does not name the unique winner: task=%+v err=%v", got, err)
	}
	runs, err := repos.Runs.ListByTask(ctx, task.ID, repository.ListFilter{Limit: 10})
	if err != nil || len(runs) != 1 || runs[0].ID != winner {
		t.Fatalf("persisted attempts=%+v err=%v, want only winner %s", runs, err, winner)
	}
}

func TestEnsureCurrentRunBindsOnlyTheExactUnambiguousLegacyRun(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	makeTask := func(status domain.TaskStatus) *domain.Task {
		task := &domain.Task{ID: uuid.New(), Title: "legacy continuation", ScopePath: ".", Status: status}
		if err := repos.Tasks.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
		return task
	}
	legacy := makeTask(domain.TaskStatusQueued)
	terminal := pendingTaskRun(legacy.ID)
	if err := repos.Runs.Create(ctx, terminal); err != nil {
		t.Fatal(err)
	}
	terminal.Status = domain.RunStatusComplete
	if err := repos.Runs.Update(ctx, terminal); err != nil {
		t.Fatal(err)
	}
	exact := pendingTaskRun(legacy.ID)
	if err := repos.Runs.Create(ctx, exact); err != nil {
		t.Fatal(err)
	}
	claimed, err := repos.Runs.EnsureCurrentRun(ctx, legacy.ID, exact.ID, time.Now())
	if err != nil || !claimed {
		t.Fatalf("exact sole-live legacy run claim=%v err=%v", claimed, err)
	}
	claimed, err = repos.Runs.EnsureCurrentRun(ctx, legacy.ID, terminal.ID, time.Now())
	if err != nil || claimed {
		t.Fatalf("different historical run displaced exact owner: claim=%v err=%v", claimed, err)
	}

	ambiguous := makeTask(domain.TaskStatusQueued)
	one := pendingTaskRun(ambiguous.ID)
	two := pendingTaskRun(ambiguous.ID)
	if err := repos.Runs.Create(ctx, one); err != nil {
		t.Fatal(err)
	}
	if err := repos.Runs.Create(ctx, two); err != nil {
		t.Fatal(err)
	}
	claimed, err = repos.Runs.EnsureCurrentRun(ctx, ambiguous.ID, one.ID, time.Now())
	if err != nil || claimed {
		t.Fatalf("competing legacy Run was silently selected: claim=%v err=%v", claimed, err)
	}

	cancelled := makeTask(domain.TaskStatusCancelled)
	closed := pendingTaskRun(cancelled.ID)
	if err := repos.Runs.Create(ctx, closed); err != nil {
		t.Fatal(err)
	}
	claimed, err = repos.Runs.EnsureCurrentRun(ctx, cancelled.ID, closed.ID, time.Now())
	if err != nil || claimed {
		t.Fatalf("explicitly cancelled Task accepted continuation: claim=%v err=%v", claimed, err)
	}
}

func pendingTaskRun(taskID uuid.UUID) *domain.Run {
	return &domain.Run{ID: uuid.New(), TaskID: taskID, Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued, ApprovalState: domain.ApprovalStateNone}
}

func TestCancelQueuedLegacyTaskRefusesUnownedNonterminalRun(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "legacy queued task", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := pendingTaskRun(task.ID)
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	_, changed, err := repos.Runs.CancelTaskAttempt(ctx, task.ID, nil, false, time.Now())
	if err != nil || changed {
		t.Fatalf("ambiguous legacy cancellation changed=%v err=%v; want refusal", changed, err)
	}
	gotTask, err := repos.Tasks.Get(ctx, task.ID)
	if err != nil || gotTask.Status != domain.TaskStatusQueued || gotTask.CurrentRunID != nil {
		t.Fatalf("ambiguous legacy task changed after refused cancellation: task=%+v err=%v", gotTask, err)
	}
	gotRun, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || gotRun == nil || gotRun.Status != domain.RunStatusPending || gotRun.CancelRequestedAt != nil {
		t.Fatalf("unowned legacy run changed after refused cancellation: run=%+v err=%v", gotRun, err)
	}
}

func TestCancelQueuedLegacyTaskWithOnlyTerminalHistory(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "terminal legacy history", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	prior := pendingTaskRun(task.ID)
	prior.Status = domain.RunStatusComplete
	if err := repos.Runs.Create(ctx, prior); err != nil {
		t.Fatal(err)
	}
	_, changed, err := repos.Runs.CancelTaskAttempt(ctx, task.ID, nil, false, time.Now())
	if err != nil || !changed {
		t.Fatalf("terminal-only legacy cancellation changed=%v err=%v; want success", changed, err)
	}
	gotTask, err := repos.Tasks.Get(ctx, task.ID)
	if err != nil || gotTask.Status != domain.TaskStatusCancelled {
		t.Fatalf("terminal-only task status=%v err=%v; want cancelled", gotTask.Status, err)
	}
}

func TestTaskCancellationAndRunProjectionShareExactCurrentRunFence(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "cancel fence", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := pendingTaskRun(task.ID)
	if err := repos.Runs.CreateTaskAttempt(ctx, task, task, run, nil); err != nil {
		t.Fatal(err)
	}
	run.Status = domain.RunStatusRunning
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	_, changed, err := repos.Runs.CancelTaskAttempt(ctx, task.ID, nil, false, task.UpdatedAt.Add(time.Second))
	if err != nil || changed {
		t.Fatalf("nil-owner cancellation changed=%v err=%v", changed, err)
	}
	_, changed, err = repos.Runs.CancelTaskAttempt(ctx, task.ID, &run.ID, false, task.UpdatedAt.Add(time.Second))
	if err != nil || !changed {
		t.Fatalf("exact-owner cancellation changed=%v err=%v", changed, err)
	}
	run.Status = domain.RunStatusComplete
	if err := repos.Runs.Update(ctx, run); err == nil {
		t.Fatal("stale run writer overwrote explicit cancellation")
	}
	got, err := repos.Tasks.Get(ctx, task.ID)
	if err != nil || got.Status != domain.TaskStatusCancelled || got.CurrentRunID == nil || *got.CurrentRunID != run.ID {
		t.Fatalf("late run projection replaced explicit task cancellation: task=%+v err=%v", got, err)
	}
}

func TestResumeAdmissionAdvancesRunLifecycleVersionAndFencesOldFinalizer(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "resume lifecycle version", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := pendingTaskRun(task.ID)
	if err := repos.Runs.CreateTaskAttempt(ctx, task, task, run, nil); err != nil {
		t.Fatal(err)
	}
	run.Status = domain.RunStatusNeedsReview
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	staleFinalizer := *run
	admittedAt := time.Now().Add(time.Second)
	updated, err := repos.Runs.AdmitCurrentRun(ctx, run, domain.RunStatusNeedsReview, admittedAt)
	if err != nil || !updated {
		t.Fatalf("exact resumable attempt admission=%v err=%v", updated, err)
	}
	if run.Status != domain.RunStatusRunning || run.LifecycleVersion <= staleFinalizer.LifecycleVersion {
		t.Fatalf("admission did not advance lifecycle version: run=%+v stale=%d", run, staleFinalizer.LifecycleVersion)
	}
	staleFinalizer.Status = domain.RunStatusComplete
	if err := repos.Runs.Update(ctx, &staleFinalizer); err == nil {
		t.Fatal("old invocation finalized after same Run was admitted for continuation")
	}
	gotRun, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || gotRun == nil || gotRun.Status != domain.RunStatusRunning {
		t.Fatalf("current resumed invocation was overwritten: run=%+v err=%v", gotRun, err)
	}
	gotTask, err := repos.Tasks.Get(ctx, task.ID)
	if err != nil || gotTask.Status != domain.TaskStatusRunning {
		t.Fatalf("resume admission did not project Task running: task=%+v err=%v", gotTask, err)
	}

	// This is the state restoration used if dispatcher enqueue rejects the
	// resumed job. The current lifecycle version permits it when no competing
	// cancellation or finalizer has advanced the row.
	run.Status = domain.RunStatusNeedsReview
	run.UpdatedAt = time.Now().Add(2 * time.Second)
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatalf("restore prior resumable state after enqueue rejection: %v", err)
	}
	gotTask, err = repos.Tasks.Get(ctx, task.ID)
	if err != nil || gotTask.Status != domain.TaskStatusNeedsReview {
		t.Fatalf("restored run state did not restore Task projection: task=%+v err=%v", gotTask, err)
	}
}

func TestResumeAdmissionRollbackCannotOverwriteCancellation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "cancel resumed attempt", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := pendingTaskRun(task.ID)
	if err := repos.Runs.CreateTaskAttempt(ctx, task, task, run, nil); err != nil {
		t.Fatal(err)
	}
	run.Status = domain.RunStatusNeedsReview
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	if updated, err := repos.Runs.AdmitCurrentRun(ctx, run, domain.RunStatusNeedsReview, time.Now().Add(time.Second)); err != nil || !updated {
		t.Fatalf("admit current run updated=%v err=%v", updated, err)
	}
	if _, changed, err := repos.Runs.CancelTaskAttempt(ctx, task.ID, &run.ID, false, time.Now().Add(2*time.Second)); err != nil || !changed {
		t.Fatalf("cancel exact resumed attempt changed=%v err=%v", changed, err)
	}
	run.Status = domain.RunStatusNeedsReview
	if err := repos.Runs.Update(ctx, run); err == nil {
		t.Fatal("enqueue-failure rollback overwrote a newer cancellation")
	}
	gotTask, err := repos.Tasks.Get(ctx, task.ID)
	if err != nil || gotTask.Status != domain.TaskStatusCancelled {
		t.Fatalf("cancellation was not retained: task=%+v err=%v", gotTask, err)
	}
}

func TestCancelPendingAttemptIsDurableBeforeRecovery(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "pending cancel", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := pendingTaskRun(task.ID)
	if err := repos.Runs.CreateTaskAttempt(ctx, task, task, run, nil); err != nil {
		t.Fatal(err)
	}
	_, changed, err := repos.Runs.CancelTaskAttempt(ctx, task.ID, &run.ID, false, task.UpdatedAt.Add(time.Second))
	if err != nil || !changed {
		t.Fatalf("cancel pending attempt changed=%v err=%v", changed, err)
	}
	gotTask, err := repos.Tasks.Get(ctx, task.ID)
	if err != nil || gotTask.Status != domain.TaskStatusCancelled || gotTask.CurrentRunID == nil || *gotTask.CurrentRunID != run.ID {
		t.Fatalf("task cancellation not durable: task=%+v err=%v", gotTask, err)
	}
	gotRun, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || gotRun.Status != domain.RunStatusCancelled || gotRun.EndedAt == nil {
		t.Fatalf("pending Run remained recoverable: run=%+v err=%v", gotRun, err)
	}
}

func TestCancelPendingAttemptWithExecutorClaimPersistsCancellationIntent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "claimed pending cancel", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := pendingTaskRun(task.ID)
	if err := repos.Runs.CreateTaskAttempt(ctx, task, task, run, nil); err != nil {
		t.Fatal(err)
	}
	_, changed, err := repos.Runs.CancelTaskAttempt(ctx, task.ID, &run.ID, true, task.UpdatedAt.Add(time.Second))
	if err != nil || !changed {
		t.Fatalf("cancel claimed pending attempt changed=%v err=%v", changed, err)
	}
	gotRun, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || gotRun.Status != domain.RunStatusPending || gotRun.CancelRequestedAt == nil {
		t.Fatalf("claimed Run lost its cancellation intent: run=%+v err=%v", gotRun, err)
	}
}

func TestInitSchemaAddsCurrentRunIDToLegacyTasksBeforeContractValidation(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(ON)", uuid.NewString())
	sqlDB, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	_, err = sqlDB.Exec(`CREATE TABLE tasks (
		id TEXT PRIMARY KEY, title TEXT NOT NULL, description TEXT, scope_path TEXT NOT NULL,
		project_root TEXT, phase_prompt_ids TEXT DEFAULT '[]', context_attachments TEXT DEFAULT '[]',
		status TEXT DEFAULT 'queued', created_by TEXT, created_at TEXT DEFAULT (datetime('now')),
		updated_at TEXT DEFAULT (datetime('now')))`)
	if err != nil {
		t.Fatal(err)
	}
	db := NewDB(sqlDB, logrus.New())
	if err := db.InitializeSchema(); err != nil {
		t.Fatalf("InitializeSchema: %v", err)
	}
	columns, err := db.tableColumns(context.Background(), "tasks")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := columns["current_run_id"]; !ok {
		t.Fatal("legacy tasks table was not upgraded with current_run_id")
	}
}

func TestDeleteTerminalRunRequiresExactLifecycleSnapshot(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "delete fence", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := pendingTaskRun(task.ID)
	if err := repos.Runs.CreateTaskAttempt(ctx, task, task, run, nil); err != nil {
		t.Fatal(err)
	}
	run.Status = domain.RunStatusComplete
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	observed, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	deleter := repos.Runs
	if err := deleter.Delete(ctx, run.ID, observed.Status, observed.LifecycleVersion-1); err == nil {
		t.Fatal("stale lifecycle snapshot deleted a run")
	}
	stillThere, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || stillThere == nil {
		t.Fatalf("stale delete removed run: run=%+v err=%v", stillThere, err)
	}
	stillOwned, err := repos.Tasks.Get(ctx, task.ID)
	if err != nil || stillOwned.CurrentRunID == nil || *stillOwned.CurrentRunID != run.ID {
		t.Fatalf("stale delete cleared current pointer: task=%+v err=%v", stillOwned, err)
	}
	if err := deleter.Delete(ctx, run.ID, observed.Status, observed.LifecycleVersion); err != nil {
		t.Fatalf("delete exact terminal snapshot: %v", err)
	}
	deleted, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || deleted != nil {
		t.Fatalf("exact terminal run remains: run=%+v err=%v", deleted, err)
	}
	cleared, err := repos.Tasks.Get(ctx, task.ID)
	if err != nil || cleared.CurrentRunID != nil {
		t.Fatalf("exact task pointer not cleared: task=%+v err=%v", cleared, err)
	}
}
