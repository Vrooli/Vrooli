package mentions

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestStoreDeduplicatesEventBeforeAnyReply(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	request, err := Parse(validDelivery())
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.Record(context.Background(), request)
	if err != nil || !created {
		t.Fatalf("first event: created=%v err=%v", created, err)
	}
	created, err = store.Record(context.Background(), request)
	if err != nil || created {
		t.Fatalf("duplicate event: created=%v err=%v", created, err)
	}
	claimed, err := store.ClaimReply(context.Background(), request.Key)
	if err != nil || !claimed {
		t.Fatalf("reply claim: claimed=%v err=%v", claimed, err)
	}
	claimed, err = store.ClaimReply(context.Background(), request.Key)
	if err != nil || claimed {
		t.Fatalf("duplicate reply claim: claimed=%v err=%v", claimed, err)
	}
	if err := store.MarkReplyDelivered(context.Background(), request.Key, "receipt-1"); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkReplyDelivered(context.Background(), request.Key, "receipt-2"); err == nil {
		t.Fatal("second delivery receipt accepted")
	}
}

func TestStoreMigratesLegacySchemaForReplyOutbox(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE gct_advisory_mentions (
        event_key TEXT PRIMARY KEY, provider TEXT NOT NULL, actor_id TEXT NOT NULL,
        repository_id TEXT NOT NULL, head_revision TEXT NOT NULL, command TEXT NOT NULL,
        state TEXT NOT NULL, created_at TEXT NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}

	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	request, err := Parse(validDelivery())
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.Record(context.Background(), request)
	if err != nil || !created {
		t.Fatalf("record after migration: created=%v err=%v", created, err)
	}
	claimed, err := store.ClaimReply(context.Background(), request.Key)
	if err != nil || !claimed {
		t.Fatalf("reply claim after migration: claimed=%v err=%v", claimed, err)
	}
}

func TestStoreRefusesReplyClaimWhenHeadRevisionIsStale(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	request, err := Parse(validDelivery())
	if err != nil {
		t.Fatal(err)
	}
	if created, err := store.Record(context.Background(), request); err != nil || !created {
		t.Fatalf("record: created=%v err=%v", created, err)
	}
	claimed, err := store.ClaimReplyForRevision(context.Background(), request.Key, "new-head")
	if err != nil {
		t.Fatal(err)
	}
	if claimed {
		t.Fatal("stale reply claim was accepted")
	}
	claimed, err = store.ClaimReplyForRevision(context.Background(), request.Key, request.Delivery.HeadRevision)
	if err != nil || !claimed {
		t.Fatalf("current reply claim: claimed=%v err=%v", claimed, err)
	}
}
