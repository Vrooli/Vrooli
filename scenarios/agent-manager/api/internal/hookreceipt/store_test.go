package hookreceipt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func receiptTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sqlx.Connect("sqlite", "file:"+uuid.NewString()+"?mode=memory&cache=shared&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := coredb.EnsureSchemas(context.Background(), db, coredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	return NewStore(db)
}

func TestConsumePersistsExactReceiptAndRejectsReplayOrCrossBinding(t *testing.T) {
	ctx := context.Background()
	store := receiptTestStore(t)
	now := time.Date(2026, 11, 16, 20, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	runID, invocationID := uuid.New(), uuid.New()
	credential, err := store.Issue(ctx, runID, invocationID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	receipt := Receipt{ReceiptID: uuid.NewString(), SessionID: "session-1", Runner: "codex", EvaluatorID: "policy:deny", DurationMS: 0, OccurredAt: now}
	if err := store.Consume(ctx, credential, runID, invocationID, receipt); err != nil {
		t.Fatalf("consume measured zero-duration receipt: %v", err)
	}
	if err := store.Consume(ctx, credential, runID, invocationID, receipt); !errors.Is(err, ErrCredentialUsed) {
		t.Fatalf("replay error = %v, want ErrCredentialUsed", err)
	}
	if err := store.Consume(ctx, credential, uuid.New(), invocationID, receipt); !errors.Is(err, ErrWrongBinding) {
		t.Fatalf("cross-run error = %v, want ErrWrongBinding", err)
	}
	got, err := store.ReceiptsForRuns(ctx, []uuid.UUID{runID})
	if err != nil || len(got[runID]) != 1 || got[runID][0] != receipt {
		t.Fatalf("stored receipt = %#v, err=%v", got, err)
	}

	concurrentInvocation := uuid.New()
	concurrentCredential, err := store.Issue(ctx, runID, concurrentInvocation, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	concurrentReceipt := receipt
	concurrentReceipt.ReceiptID = uuid.NewString()
	results := make(chan error, 2)
	for range 2 {
		go func() {
			results <- store.Consume(ctx, concurrentCredential, runID, concurrentInvocation, concurrentReceipt)
		}()
	}
	successes, replays := 0, 0
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrCredentialUsed):
			replays++
		default:
			t.Fatalf("concurrent consume error = %v", err)
		}
	}
	if successes != 1 || replays != 1 {
		t.Fatalf("concurrent consume successes=%d replays=%d, want 1 each", successes, replays)
	}
}

func TestConsumeRejectsExpiredCredentialAndMalformedReceipt(t *testing.T) {
	ctx := context.Background()
	store := receiptTestStore(t)
	now := time.Date(2026, 11, 16, 20, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	runID, invocationID := uuid.New(), uuid.New()
	credential, err := store.Issue(ctx, runID, invocationID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	base := Receipt{ReceiptID: uuid.NewString(), SessionID: "session-1", Runner: "codex", EvaluatorID: "policy:deny", DurationMS: 1, OccurredAt: now}
	bad := base
	bad.EvaluatorID = "policy:deny\nforged"
	if err := store.Consume(ctx, credential, runID, invocationID, bad); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("malformed receipt error = %v, want ErrInvalidInput", err)
	}
	now = now.Add(time.Minute)
	if err := store.Consume(ctx, credential, runID, invocationID, base); !errors.Is(err, ErrExpiredCredential) {
		t.Fatalf("expired credential error = %v, want ErrExpiredCredential", err)
	}
}

func TestRevokeOutstandingInvalidatesPriorInvocationAfterOwnerRestart(t *testing.T) {
	ctx := context.Background()
	store := receiptTestStore(t)
	now := time.Now().UTC()
	store.now = func() time.Time { return now }
	runID, invocationID := uuid.New(), uuid.New()
	credential, err := store.Issue(ctx, runID, invocationID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if count, err := store.RevokeOutstanding(ctx); err != nil || count != 1 {
		t.Fatalf("revoke outstanding = %d, err=%v", count, err)
	}
	receipt := Receipt{ReceiptID: uuid.NewString(), SessionID: "session-1", Runner: "codex", EvaluatorID: "policy:deny", DurationMS: 1, OccurredAt: now}
	if err := store.Consume(ctx, credential, runID, invocationID, receipt); !errors.Is(err, ErrRevokedCredential) {
		t.Fatalf("restarted invocation error = %v, want ErrRevokedCredential", err)
	}
}
