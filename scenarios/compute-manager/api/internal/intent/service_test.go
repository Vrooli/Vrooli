package intent

import (
	"context"
	"errors"
	"testing"
	"time"

	"compute-manager/internal/provider"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
)

func TestSQLStoreRoundTripsIntentByIdempotencyKey(t *testing.T) {
	db := databasetest.NewSQLite(t)
	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(func() string { return schema() })); err != nil {
		t.Fatal(err)
	}
	store := SQLStore{DB: db}
	want := Record{IdempotencyKey: "idem-1", Provider: "fake", Spec: map[string]any{"size": "small"}, CreatedAt: time.Now().UTC()}
	created, err := store.Create(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.GetByIDOrKey(context.Background(), "idem-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != created.ID || got.IdempotencyKey != want.IdempotencyKey || got.State != StateReserving {
		t.Fatalf("got %+v, want %+v", got, created)
	}
}

func TestServiceLeavesOpenIntentWhenCreateResponseIsLost(t *testing.T) {
	db := databasetest.NewSQLite(t)
	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(func() string { return schema() })); err != nil {
		t.Fatal(err)
	}
	fake := &provider.Fake{LoseCreateResponse: true}
	svc := Service{Store: SQLStore{DB: db}, Provider: fake}
	record, _, err := svc.Create(context.Background(), Request{IdempotencyKey: "lost-1", Provider: "fake", Spec: provider.Spec{Size: "small"}})
	if !errors.Is(err, provider.ErrCreateResponseLost) {
		t.Fatalf("Create error = %v", err)
	}
	if record.State != StateOpen || record.ID == "" {
		t.Fatalf("record = %+v, want open durable intent", record)
	}
	got, err := (SQLStore{DB: db}).GetByIDOrKey(context.Background(), record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StateOpen {
		t.Fatalf("stored state = %q, want open", got.State)
	}
}

func schema() string {
	return `CREATE TABLE instance_intents (id TEXT PRIMARY KEY,idempotency_key TEXT UNIQUE,requested_by TEXT,provider TEXT,spec_json TEXT,reservation_id TEXT,state TEXT,instance_id TEXT,created_at TEXT,resolved_at TEXT);`
}
