package orchestration

import (
	"testing"
	"time"

	"agent-manager/internal/orchestration/testutil"
	"github.com/google/uuid"
)

func TestContinuationAcceptedRequiresExactDurableOwnerReceipt(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithIdempotency(repos.Idempotency))
	ctx, id, key := t.Context(), uuid.New(), "effort-directive:receipt-fixture"
	message := "verify the assigned regression against retained baseline"
	if ok, err := o.ContinuationAccepted(ctx, id, message, key); err != nil || ok {
		t.Fatal("missing receipt inferred acceptance", ok, err)
	}
	if _, err := repos.Idempotency.Reserve(ctx, key, time.Hour); err != nil {
		t.Fatal(err)
	}
	if ok, err := o.ContinuationAccepted(ctx, id, message, key); err != nil || ok {
		t.Fatal("pending reservation inferred acceptance", ok, err)
	}
	// Exercise the actual continuation emitter, not a hand-written receipt shape.
	if err := o.completeContinuationReceipt(ctx, ContinueRunRequest{RunID: id, Message: message, IdempotencyKey: key}); err != nil {
		t.Fatal(err)
	}
	if ok, err := o.ContinuationAccepted(ctx, id, message, key); err != nil || !ok {
		t.Fatal("durable admission lost", ok, err)
	}
	if ok, err := o.ContinuationAccepted(ctx, uuid.New(), message, key); err == nil || ok {
		t.Fatal("another run's receipt accepted", ok, err)
	}
	if ok, err := o.ContinuationAccepted(ctx, id, message+"changed", key); err == nil || ok {
		t.Fatal("another payload's receipt accepted", ok, err)
	}
}
