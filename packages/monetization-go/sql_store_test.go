package monetization

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSQLStorePersistsAndCountsSharedUsage(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE monetization_usage_outbox (
		operation_id TEXT PRIMARY KEY,
		user_identity TEXT NOT NULL,
		payload TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		attempts INTEGER NOT NULL DEFAULT 0,
		next_attempt_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_error TEXT,
		delivered_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatal(err)
	}

	store := NewSQLStore(db, SQLDialectSQLite)
	usage := testUsage()
	inserted, err := store.Append(context.Background(), usage)
	if err != nil || !inserted {
		t.Fatalf("append: inserted=%v err=%v", inserted, err)
	}
	inserted, err = store.Append(context.Background(), usage)
	if err != nil || inserted {
		t.Fatalf("duplicate append: inserted=%v err=%v", inserted, err)
	}

	count, err := store.PendingCount(context.Background(), usage.UserIdentity)
	if err != nil || count != 1 {
		t.Fatalf("pending count: count=%d err=%v", count, err)
	}
	records, err := store.Pending(context.Background(), 10, time.Now().Add(time.Minute))
	if err != nil || len(records) != 1 {
		t.Fatalf("pending records: len=%d err=%v", len(records), err)
	}
	if records[0].Usage.OperationID != usage.OperationID {
		t.Fatalf("operation id changed: got %q", records[0].Usage.OperationID)
	}
	if err := store.MarkDelivered(context.Background(), usage.OperationID, time.Now()); err != nil {
		t.Fatal(err)
	}
	count, err = store.PendingCount(context.Background(), usage.UserIdentity)
	if err != nil || count != 0 {
		t.Fatalf("delivered pending count: count=%d err=%v", count, err)
	}
}
