package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func TestTaskRepositoryRejectsInvalidInitialAndUnknownPersistedStatus(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	if err := repos.Tasks.Create(ctx, &domain.Task{ID: uuid.New(), Title: "invalid", ScopePath: ".", Status: domain.TaskStatusApproved}); err == nil {
		t.Fatal("new Task with a terminal status was accepted")
	}
	task := &domain.Task{ID: uuid.New(), Title: "valid", ScopePath: "."}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	if task.Status != domain.TaskStatusQueued {
		t.Fatalf("empty initial status normalized to %q, want queued", task.Status)
	}
	if _, err := db.DB.ExecContext(ctx, `UPDATE tasks SET status = 'future' WHERE id = ?`, task.ID.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := repos.Tasks.Get(ctx, task.ID); err == nil {
		t.Fatal("unknown persisted Task status was accepted")
	}
}

func TestRunProjectionUsesTaskTransitionOwnerAndExactObservedState(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	taskID, runID := uuid.New(), uuid.New()
	task := &domain.Task{ID: taskID, Title: "projection", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	if _, err := db.DB.ExecContext(ctx, `UPDATE tasks SET current_run_id = ? WHERE id = ?`, runID.String(), taskID.String()); err != nil {
		t.Fatal(err)
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectTaskStatusTx(ctx, tx, taskID, runID, domain.TaskStatusNeedsReview, time.Now()); err == nil {
		_ = tx.Rollback()
		t.Fatal("queued Task projected directly to needs_review")
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	tx, err = db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if applied, err := projectTaskStatusTx(ctx, tx, taskID, runID, domain.TaskStatusRunning, time.Now()); err != nil || !applied {
		_ = tx.Rollback()
		t.Fatalf("queued-to-running projection applied=%v err=%v", applied, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	got, err := repos.Tasks.Get(ctx, taskID)
	if err != nil || got.Status != domain.TaskStatusRunning {
		t.Fatalf("projected Task=%+v err=%v", got, err)
	}
}
