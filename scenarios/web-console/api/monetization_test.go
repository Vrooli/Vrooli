package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	monetization "github.com/vrooli/vrooli/packages/monetization-go"
	_ "modernc.org/sqlite"
)

func TestSQLMonetizationOutboxStoreIsDurableAndIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", "file:monetization-outbox-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE monetization_usage_outbox (
			operation_id TEXT PRIMARY KEY,
			user_identity TEXT NOT NULL,
			payload TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			attempts INTEGER NOT NULL DEFAULT 0,
			next_attempt_at TEXT NOT NULL DEFAULT '1970-01-01T00:00:00.000Z',
			last_error TEXT,
			delivered_at TEXT,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatal(err)
	}
	store := &sqlMonetizationOutboxStore{db: db}
	usage := monetization.Usage{
		OperationID:  "op-1",
		UserIdentity: "alice@example.com",
		BundleKey:    "business_suite",
		AppKey:       "web-console",
		MeterKey:     "voice_minutes",
		Units:        1,
		OccurredAt:   time.Now().UTC(),
		Metadata:     map[string]string{"operation": "voice_synthesis"},
	}

	inserted, err := store.Append(context.Background(), usage)
	if err != nil || !inserted {
		t.Fatalf("first append = (%v, %v), want inserted", inserted, err)
	}
	inserted, err = store.Append(context.Background(), usage)
	if err != nil || inserted {
		t.Fatalf("duplicate append = (%v, %v), want no-op", inserted, err)
	}

	rows, err := store.Pending(context.Background(), 10, time.Now().UTC().Add(time.Second))
	if err != nil || len(rows) != 1 || rows[0].Usage.OperationID != "op-1" {
		t.Fatalf("pending = (%v, %v), want op-1", rows, err)
	}
	next := time.Now().UTC().Add(time.Minute)
	if err := store.MarkRetry(context.Background(), "op-1", next, "temporary"); err != nil {
		t.Fatal(err)
	}
	rows, err = store.Pending(context.Background(), 10, time.Now().UTC())
	if err != nil || len(rows) != 0 {
		t.Fatalf("retried pending = (%v, %v), want hidden until next attempt", rows, err)
	}
	if err := store.MarkDelivered(context.Background(), "op-1", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
}
