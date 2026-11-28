package commerce

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPaymentReconciliationPostgresLeaseAndCursorSurviveStoreReopen(t *testing.T) {
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine", postgres.WithDatabase("lpbs_reconcile_test"), postgres.WithUsername("lpbs"), postgres.WithPassword("test-only"))
	if err != nil {
		t.Fatalf("start isolated PostgreSQL test fixture: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	connection, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("postgres", connection)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if err := db.PingContext(ctx); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("isolated PostgreSQL fixture did not accept connections")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if _, err := db.Exec(FinancialSchema()); err != nil {
		t.Fatalf("apply isolated in-memory test schema: %v", err)
	}
	first, _ := NewSQLPaymentReconciliationStateStore(db)
	second, _ := NewSQLPaymentReconciliationStateStore(db)
	now := time.Now().UTC()
	type claimResult struct {
		store *SQLPaymentReconciliationStateStore
		lease PaymentReconciliationLease
		err   error
	}
	start := make(chan struct{})
	claims := make(chan claimResult, 2)
	for _, candidate := range []struct {
		store *SQLPaymentReconciliationStateStore
		owner string
	}{{first, "worker-one"}, {second, "worker-two"}} {
		candidate := candidate
		go func() {
			<-start
			lease, claimErr := candidate.store.Claim(ctx, candidate.owner, now, now.Add(250*time.Millisecond))
			claims <- claimResult{store: candidate.store, lease: lease, err: claimErr}
		}()
	}
	close(start)
	var lease1 PaymentReconciliationLease
	var firstStore *SQLPaymentReconciliationStateStore
	won, held := 0, 0
	for range 2 {
		claim := <-claims
		if claim.err == nil {
			won++
			lease1, firstStore = claim.lease, claim.store
		} else if errors.Is(claim.err, ErrReconciliationLeaseHeld) {
			held++
		} else {
			t.Fatalf("concurrent claim: %v", claim.err)
		}
	}
	if won != 1 || held != 1 {
		t.Fatalf("concurrent lease results: %d winner(s), %d held", won, held)
	}
	cursor := PaymentReconciliationCursor{CreatedAt: now.Add(-time.Hour), EventID: "evt_checkpointed"}
	if err := firstStore.Advance(ctx, lease1, cursor, now); err != nil {
		t.Fatal(err)
	}
	// A new store instance simulates process restart. It cannot claim until the
	// old lease expires, then must resume from the last committed cursor.
	restarted, _ := NewSQLPaymentReconciliationStateStore(db)
	time.Sleep(300 * time.Millisecond)
	claimNow := time.Now().UTC()
	lease2, err := restarted.Claim(ctx, "worker-two", claimNow, claimNow.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if lease2.Generation <= lease1.Generation || !lease2.HasCursor || lease2.Cursor != (PaymentReconciliationCursor{CreatedAt: cursor.CreatedAt.Truncate(time.Microsecond), EventID: cursor.EventID}) {
		t.Fatalf("reopened lease = %+v; want later generation and persisted cursor %+v", lease2, cursor)
	}
	if _, err := first.Renew(ctx, lease1, claimNow, claimNow.Add(time.Minute)); !errors.Is(err, ErrReconciliationLeaseLost) {
		t.Fatalf("stale owner renewed after takeover: %v", err)
	}
	if err := first.Advance(ctx, lease1, PaymentReconciliationCursor{CreatedAt: now, EventID: "evt_stale"}, claimNow); !errors.Is(err, ErrReconciliationLeaseLost) {
		t.Fatalf("stale generation advanced cursor: %v", err)
	}
	if err := restarted.Release(ctx, lease2); err != nil {
		t.Fatal(err)
	}
}
