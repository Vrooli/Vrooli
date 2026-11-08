package database

import (
	"context"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// insertLineage inserts a chain of runs where each run is the previous one's
// child, without setting Depth: the repository derives it.
func insertLineage(t *testing.T, repos *Repositories, length int) []*domain.Run {
	t.Helper()
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "lineage", ScopePath: "/lineage", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	var chain []*domain.Run
	for i := 0; i < length; i++ {
		run := &domain.Run{ID: uuid.New(), TaskID: task.ID, RunMode: domain.RunModeInPlace, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
		if i > 0 {
			parent := chain[i-1].ID
			run.ParentRunID = &parent
		}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
		chain = append(chain, run)
	}
	return chain
}

func requireDepths(t *testing.T, repos *Repositories, chain []*domain.Run, want ...int) {
	t.Helper()
	for i, run := range chain {
		saved, err := repos.Runs.Get(context.Background(), run.ID)
		if err != nil || saved == nil {
			t.Fatalf("read run %d: %v", i, err)
		}
		if saved.Depth != want[i] {
			t.Fatalf("run %d depth = %d, want %d", i, saved.Depth, want[i])
		}
	}
}

// Every insert path (create, import, attach) derives depth from the parent row,
// and a full-row update that keeps the parent keeps the depth.
func TestRunDepthIsDerivedFromTheParentRowOnInsert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	chain := insertLineage(t, repos, 4)
	requireDepths(t, repos, chain, 0, 1, 2, 3)

	ctx := context.Background()
	saved, err := repos.Runs.Get(ctx, chain[2].ID)
	if err != nil {
		t.Fatal(err)
	}
	saved.Depth = 0 // a stale in-memory value must not rewrite the lineage
	saved.Label = "updated"
	if err := repos.Runs.Update(ctx, saved); err != nil {
		t.Fatal(err)
	}
	requireDepths(t, repos, chain, 0, 1, 2, 3)

	listed, err := repos.Runs.List(ctx, runListFilterForParent(chain[1].ID))
	if err != nil || len(listed) != 1 || listed[0].Depth != 2 {
		t.Fatalf("listing children of the depth-1 run = %+v, %v; want one run at depth 2", listed, err)
	}
}

// Children created before runs.depth existed are backfilled from their lineage;
// a child whose parent row is gone starts a new chain at depth 1.
func TestRunDepthBackfillFollowsLineage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	chain := insertLineage(t, repos, 3)
	orphanChain := insertLineage(t, repos, 2)
	if _, err := db.ExecContext(ctx, `UPDATE runs SET parent_run_id = ? WHERE id = ?`, uuid.NewString(), orphanChain[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE runs SET depth = 0`); err != nil {
		t.Fatal(err)
	}
	if err := db.WithTransaction(ctx, func(tx *Tx) error { return backfillRunDepth(ctx, tx) }); err != nil {
		t.Fatal(err)
	}
	requireDepths(t, repos, chain, 0, 1, 2)
	requireDepths(t, repos, orphanChain, 1, 2)
}

func runListFilterForParent(parent uuid.UUID) repository.RunListFilter {
	return repository.RunListFilter{ParentRunID: &parent}
}
