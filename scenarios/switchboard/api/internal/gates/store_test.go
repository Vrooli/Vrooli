package gates

import (
	"context"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

// [REQ:SWBD-P1-009]
func TestStorePersistsOwnerOnlyGateAndExpiresAcrossReads(t *testing.T) {
	db := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	s := NewStore(db, func() time.Time { return now })
	g, err := s.Raise(context.Background(), "thread-1", "owner-1", "files.write", "private file", "owner approval", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	got, ok, err := s.Get(context.Background(), g.ID)
	if err != nil || !ok || got.Status != Expired {
		t.Fatalf("gate = %+v, ok=%v, err=%v", got, ok, err)
	}
	if _, err := s.Answer(context.Background(), g.ID, "viewer", true); err != ErrNotOwner {
		t.Fatalf("non-owner error = %v", err)
	}
}

func TestStoreRejectsUnknownPersistedGateStatus(t *testing.T) {
	db := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	s := NewStore(db, time.Now)
	gate, err := s.Raise(context.Background(), "thread-unknown", "owner", "files.write", "file", "approval", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(context.Background(), `UPDATE switchboard_capability_gates SET status=? WHERE id=?`, "corrupt", gate.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.Get(context.Background(), gate.ID); err == nil || ok {
		t.Fatalf("unknown persisted status was accepted: ok=%v err=%v", ok, err)
	}
}
