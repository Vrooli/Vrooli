package orchestration_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

// TestCreateRunReconcilesLostAcceptedResponse covers Phase 3's lost-response
// reconciliation contract: a creation that persisted its run but lost the caller
// response leaves a pending idempotency record with no completion. A replayed
// start with the same key must resolve the original durable run instead of
// refusing the replay or starting a duplicate.
func TestCreateRunReconcilesLostAcceptedResponse(t *testing.T) {
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithEvents(eventStore),
		orchestration.WithIdempotency(repos.Idempotency),
	)
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Lost response task",
		ScopePath: "src/",
		Status:    domain.TaskStatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if _, err := svc.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	key := "lost-response-" + uuid.NewString()

	// Simulate a dispatch that persisted its run, then lost the caller response
	// before the idempotency record was marked complete.
	if _, err := repos.Idempotency.Reserve(ctx, key, time.Hour); err != nil {
		t.Fatalf("reserve idempotency key: %v", err)
	}
	original := &domain.Run{
		ID:             uuid.New(),
		TaskID:         task.ID,
		Status:         domain.RunStatusPending,
		Phase:          domain.RunPhaseQueued,
		IdempotencyKey: key,
		ApprovalState:  domain.ApprovalStateNone,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := repos.Runs.Create(ctx, original); err != nil {
		t.Fatalf("create original run: %v", err)
	}

	replayed, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		Prompt:         "replayed start after lost response",
		IdempotencyKey: key,
	})
	if err != nil {
		t.Fatalf("replayed start should reconcile, got error: %v", err)
	}
	if replayed == nil || replayed.ID != original.ID {
		t.Fatalf("replayed start did not return the original run: got %+v want id %s", replayed, original.ID)
	}

	// Reconciliation must also close the idempotency record so later replays
	// take the completed fast path instead of reconciling again.
	record, err := repos.Idempotency.Check(ctx, key)
	if err != nil {
		t.Fatalf("check idempotency record: %v", err)
	}
	if record == nil || record.Status != domain.IdempotencyStatusComplete {
		t.Fatalf("idempotency record not completed after reconcile: %+v", record)
	}
}

func TestCreateRunAcceptedReceiptSurvivesCacheExpiryAndDeletion(t *testing.T) {
	for _, deleteRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "retained-run", true: "deleted-run"}[deleteRun], func(t *testing.T) {
			db, cleanup := testutil.SetupTestDB(t)
			t.Cleanup(cleanup)
			repos, _, _ := testutil.SetupTestReposWithDB(t, db)
			ctx := context.Background()
			o := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithIdempotency(repos.Idempotency))
			task := &domain.Task{ID: uuid.New(), Title: "accepted-once", ScopePath: ".", Status: domain.TaskStatusCancelled}
			if err := repos.Tasks.Create(ctx, task); err != nil {
				t.Fatal(err)
			}
			key := "accepted-once"
			if _, err := repos.Idempotency.Reserve(ctx, key, time.Hour); err != nil {
				t.Fatal(err)
			}
			original := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, IdempotencyKey: key}
			if err := repos.Runs.Create(ctx, original); err != nil {
				t.Fatal(err)
			}
			if err := repos.Idempotency.Complete(ctx, key, original.ID, "Run", nil); err != nil {
				t.Fatal(err)
			}
			if deleteRun {
				if err := o.DeleteRun(ctx, original.ID); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := db.ExecContext(ctx, `UPDATE idempotency_records SET expires_at = ? WHERE key = ?`, database.SQLiteTime(time.Now().Add(-time.Hour)), key); err != nil {
				t.Fatal(err)
			}
			if n, err := repos.Idempotency.CleanupExpired(ctx); err != nil || n != 1 {
				t.Fatal("ordinary creation cache was not expired/cleaned", n, err)
			}
			got, err := o.CreateRun(ctx, orchestration.CreateRunRequest{TaskID: task.ID, IdempotencyKey: key})
			if deleteRun {
				if err == nil || got != nil || !strings.Contains(err.Error(), "accepted-result-unavailable") {
					t.Fatal("deleted accepted creation reached fresh admission", err)
				}
			} else if err != nil || got == nil || got.ID != original.ID {
				t.Fatal("accepted run was not replayed after cache expiry", err)
			}
			receipt, err := repos.Runs.GetCreationReceipt(ctx, key)
			if err != nil || receipt == nil || receipt.RunID != original.ID {
				t.Fatal("durable read-only receipt lost original identity", err)
			}
		})
	}
}
