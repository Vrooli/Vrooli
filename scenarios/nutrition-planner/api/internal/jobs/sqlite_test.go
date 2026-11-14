package jobs

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func jobRepo(t *testing.T) Repository {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	return NewSQLiteRepository(db, schedule.System())
}

func TestSQLiteJobsPersistDedupAndReviewGate(t *testing.T) {
	repo := jobRepo(t)
	ctx := context.Background()
	first, err := repo.Create(ctx, Job{ID: "j1", WorkspaceID: "w1", Type: "recipe_extract", DedupKey: "period-1", RequestHash: "hash-1", InputRevisions: []string{"recipe:1"}, BudgetUnits: 3})
	if err != nil || first.State != Queued {
		t.Fatalf("first=%#v err=%v", first, err)
	}
	replay, err := repo.Create(ctx, Job{ID: "different", WorkspaceID: "w1", Type: "recipe_extract", DedupKey: "period-1", RequestHash: "hash-1"})
	if err != nil || replay.ID != "j1" {
		t.Fatalf("replay=%#v err=%v", replay, err)
	}
	if _, err := repo.Create(ctx, Job{WorkspaceID: "w1", Type: "recipe_extract", DedupKey: "period-1", RequestHash: "changed"}); err == nil {
		t.Fatal("changed dedup input accepted")
	}
	if _, err := repo.Transition(ctx, TransitionInput{WorkspaceID: "w1", ID: "j1", NextState: Succeeded}); err == nil {
		t.Fatal("job skipped running/review state")
	}
	if _, err := repo.Transition(ctx, TransitionInput{WorkspaceID: "w1", ID: "j1", NextState: Running, Attempts: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Transition(ctx, TransitionInput{WorkspaceID: "w1", ID: "j1", NextState: WaitingForReview, ResultReference: "proposal-1"}); err != nil {
		t.Fatal(err)
	}
	job, err := repo.Get(ctx, "j1", "w1")
	if err != nil || job.State != WaitingForReview || job.ResultReference != "proposal-1" || job.Attempts != 1 {
		t.Fatalf("job=%#v err=%v", job, err)
	}
}

func TestSQLiteJobsRejectForeignAccessAndCancelExactlyOnce(t *testing.T) {
	repo := jobRepo(t)
	ctx := context.Background()
	if _, err := repo.Create(ctx, Job{ID: "j1", WorkspaceID: "w1", Type: "draft", DedupKey: "period-1", RequestHash: "hash"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(ctx, "j1", "w2"); err == nil {
		t.Fatal("foreign job was readable")
	}
	if _, err := repo.Transition(ctx, TransitionInput{WorkspaceID: "w1", ID: "j1", NextState: Canceled}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Transition(ctx, TransitionInput{WorkspaceID: "w1", ID: "j1", NextState: Canceled}); err == nil {
		t.Fatal("terminal job transitioned twice")
	}
}
