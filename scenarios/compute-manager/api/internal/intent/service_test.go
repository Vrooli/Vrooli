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

func TestSQLStoreRoundTripsIntentByIdempotencyKey(t *testing.T) { // [REQ:COMPUTEM-P0-002]
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
	if got.ID != created.ID || got.IdempotencyKey != want.IdempotencyKey || got.State != StateOpen {
		t.Fatalf("got %+v, want %+v", got, created)
	}
}

func TestServiceLeavesOpenIntentWhenCreateResponseIsLost(t *testing.T) { // [REQ:COMPUTEM-P0-002] [REQ:COMPUTEM-P0-001]
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

func TestServiceRecoversLostCreateByProviderIntentTag(t *testing.T) { // [REQ:COMPUTEM-P0-002]
	db := databasetest.NewSQLite(t)
	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(func() string { return schema() })); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 4, 3, 0, 0, 0, time.UTC)
	fake := &provider.Fake{LoseCreateResponse: true, Now: func() time.Time { return now }}
	svc := Service{Store: SQLStore{DB: db}, Provider: fake, Now: func() time.Time { return now }}
	record, _, err := svc.Create(context.Background(), Request{IdempotencyKey: "recover-1", Provider: "fake", Spec: provider.Spec{Size: "small"}})
	if !errors.Is(err, provider.ErrCreateResponseLost) || record.State != StateOpen {
		t.Fatalf("create = record %+v, error %v; want open lost response", record, err)
	}
	recovered, err := svc.RecoverOpen(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered) != 1 || recovered[0].State != StateFulfilled || recovered[0].InstanceID == "" {
		t.Fatalf("recovered = %+v, want fulfilled intent with instance", recovered)
	}
	got, err := (SQLStore{DB: db}).GetByIDOrKey(context.Background(), record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StateFulfilled || got.InstanceID != recovered[0].InstanceID {
		t.Fatalf("stored recovered intent = %+v", got)
	}
}

func schema() string {
	return `CREATE TABLE instance_intents (id TEXT PRIMARY KEY,idempotency_key TEXT UNIQUE,requested_by TEXT,provider TEXT,spec_json TEXT,reservation_id TEXT,state TEXT,instance_id TEXT,created_at TEXT,resolved_at TEXT);`
}

func TestIntentStateTransitionsRejectInvalidAndTerminalReversal(t *testing.T) {
	for _, tc := range []struct{ from, to State }{
		{StateOpen, StateFulfilled}, {StateOpen, StateRefused}, {StateReserving, StateFulfilled}, {StateReserving, StateRefused}, {StateReserving, StateAbandoned},
		{StateFulfilled, StateOpen}, {StateRefused, StateFulfilled}, {StateOpen, StateOpen},
	} {
		want := tc.from == StateOpen && (tc.to == StateFulfilled || tc.to == StateRefused) ||
			(tc.from == StateReserving && (tc.to == StateFulfilled || tc.to == StateRefused || tc.to == StateAbandoned))
		if got := CanTransition(tc.from, tc.to); got != want {
			t.Errorf("CanTransition(%q, %q) = %v, want %v", tc.from, tc.to, got, want)
		}
	}
}

func TestSQLStoreReadsAndRecoversLegacyIntentStates(t *testing.T) {
	db := databasetest.NewSQLite(t)
	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(func() string { return schema() })); err != nil {
		t.Fatal(err)
	}
	_, err := db.ExecContext(context.Background(), `INSERT INTO instance_intents (id,idempotency_key,requested_by,provider,spec_json,reservation_id,state,instance_id,created_at,resolved_at) VALUES ('legacy','legacy','','fake','{}','','reserving','','2026-01-01T00:00:00Z',''),('terminal','terminal','','fake','{}','','abandoned','','2026-01-01T00:00:00Z','')`)
	if err != nil {
		t.Fatal(err)
	}
	store := SQLStore{DB: db}
	legacy, err := store.GetByIDOrKey(context.Background(), "legacy")
	if err != nil || legacy.State != StateReserving {
		t.Fatalf("legacy row = %+v, err=%v; want readable reserving row", legacy, err)
	}
	terminal, err := store.GetByIDOrKey(context.Background(), "terminal")
	if err != nil || terminal.State != StateAbandoned {
		t.Fatalf("terminal row = %+v, err=%v; want readable abandoned row", terminal, err)
	}
	open, err := store.ListOpen(context.Background())
	if err != nil || len(open) != 1 || open[0].State != StateReserving {
		t.Fatalf("ListOpen = %+v, err=%v; want legacy reserving row", open, err)
	}
	legacy.InstanceID = "provider-instance"
	if err := store.Transition(context.Background(), StateReserving, Record{ID: legacy.ID, State: StateFulfilled, InstanceID: legacy.InstanceID}); err != nil {
		t.Fatalf("transition legacy row: %v", err)
	}
}

func TestSQLStoreRejectsInvalidInitialAndPersistedStates(t *testing.T) {
	db := databasetest.NewSQLite(t)
	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(func() string { return schema() })); err != nil {
		t.Fatal(err)
	}
	store := SQLStore{DB: db}
	if _, err := store.Create(context.Background(), Record{IdempotencyKey: "bad", State: "mystery"}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Create error = %v, want invalid transition", err)
	}
	if _, err := store.Create(context.Background(), Record{IdempotencyKey: "ok", State: StateOpen}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO instance_intents (id,idempotency_key,provider,spec_json,state,created_at) VALUES ('bad-state','bad-state','fake','{}','mystery','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetByIDOrKey(context.Background(), "bad-state"); err == nil {
		t.Fatal("GetByIDOrKey accepted an invalid persisted state")
	}
	if err := store.Transition(context.Background(), StateOpen, Record{ID: "missing", State: StateFulfilled}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Transition error = %v, want not found/CAS miss", err)
	}
	if err := store.Transition(context.Background(), StateOpen, Record{ID: "ok", State: StateOpen}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Transition error = %v, want invalid transition", err)
	}
}
