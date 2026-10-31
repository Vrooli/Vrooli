package database

import (
	"agent-manager/internal/domain"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestDeleteTaskRequiresCancelledStateAndTerminalAttachedRuns(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	open := &domain.Task{ID: uuid.New(), Title: "open", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, open); err != nil {
		t.Fatal(err)
	}
	if err := repos.Tasks.Delete(ctx, open.ID); err == nil {
		t.Fatal("open task deleted")
	}

	cancelledWithActive := &domain.Task{ID: uuid.New(), Title: "cancelled but active run", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, cancelledWithActive); err != nil {
		t.Fatal(err)
	}
	active := pendingTaskRun(cancelledWithActive.ID)
	if err := repos.Runs.Create(ctx, active); err != nil {
		t.Fatal(err)
	}
	if _, err := db.DB.ExecContext(ctx, `UPDATE tasks SET status = ? WHERE id = ?`, string(domain.TaskStatusCancelled), cancelledWithActive.ID.String()); err != nil {
		t.Fatalf("arrange legacy cancelled task with active Run: %v", err)
	}
	if err := repos.Tasks.Delete(ctx, cancelledWithActive.ID); err == nil {
		t.Fatal("cancelled task cascaded a nonterminal run")
	}
	if got, err := repos.Runs.Get(ctx, active.ID); err != nil || got == nil {
		t.Fatalf("blocked delete removed attached run: run=%+v err=%v", got, err)
	}

	terminalTask := &domain.Task{ID: uuid.New(), Title: "cancelled terminal", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, terminalTask); err != nil {
		t.Fatal(err)
	}
	terminal := pendingTaskRun(terminalTask.ID)
	terminal.Status = domain.RunStatusCancelled
	if err := repos.Runs.Create(ctx, terminal); err != nil {
		t.Fatal(err)
	}
	if _, err := db.DB.ExecContext(ctx, `UPDATE tasks SET status = ? WHERE id = ?`, string(domain.TaskStatusCancelled), terminalTask.ID.String()); err != nil {
		t.Fatalf("arrange legacy cancelled task with terminal Run: %v", err)
	}
	if err := repos.Tasks.Delete(ctx, terminalTask.ID); err != nil {
		t.Fatalf("delete cancelled task with terminal run: %v", err)
	}
	if got, err := repos.Runs.Get(ctx, terminal.ID); err != nil || got != nil {
		t.Fatalf("terminal run was not cascade-deleted: run=%+v err=%v", got, err)
	}
}
