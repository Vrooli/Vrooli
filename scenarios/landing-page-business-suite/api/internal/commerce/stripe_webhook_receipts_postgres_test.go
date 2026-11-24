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

func TestStripeWebhookReceiptStaleProcessingReclaimUsesDatabaseClock(t *testing.T) {
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine", postgres.WithDatabase("lpbs_webhook_receipts_test"), postgres.WithUsername("lpbs"), postgres.WithPassword("test-only"))
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
	for db.PingContext(ctx) != nil {
		if time.Now().After(deadline) {
			t.Fatal("isolated PostgreSQL fixture did not accept connections")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if _, err := db.Exec(OperationsSchema()); err != nil {
		t.Fatalf("apply isolated webhook schema: %v", err)
	}
	service := &StripeWebhookService{db: db}
	body := []byte(`{"id":"evt_stale","type":"future.event","data":{"object":{"id":"obj_stale"}}}`)
	firstClaim, claimed, err := service.beginWebhookReceipt(ctx, "evt_stale", "future.event", "obj_stale", false, body, nil)
	if err != nil || !claimed {
		t.Fatalf("initial claim = %t, %v", claimed, err)
	}
	if _, err := db.Exec(`UPDATE stripe_webhook_events SET last_seen_at='2000-01-01 00:00:00' WHERE stripe_event_id='evt_stale'`); err != nil {
		t.Fatal(err)
	}
	secondClaim, claimed, err := service.beginWebhookReceipt(ctx, "evt_stale", "future.event", "obj_stale", false, body, nil)
	if err != nil || !claimed || secondClaim == firstClaim {
		t.Fatalf("stale processing reclaim = %t, token_changed=%t, err=%v", claimed, secondClaim != firstClaim, err)
	}
	if err := service.finishWebhookReceipt(ctx, "evt_stale", firstClaim, nil); !errors.Is(err, ErrStripeWebhookReceiptClaimLost) {
		t.Fatalf("old claim finish = %v; want claim lost", err)
	}
	if err := service.finishWebhookReceipt(ctx, "evt_stale", secondClaim, nil); err != nil {
		t.Fatalf("new claim finish = %v", err)
	}
}
