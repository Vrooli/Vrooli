package recipe

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func repo(t *testing.T) Repository {
	t.Helper()
	db, e := sql.Open("sqlite", ":memory:")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	if _, e = db.Exec(Schema()); e != nil {
		t.Fatal(e)
	}
	return NewSQLiteRepository(db, schedule.System())
}

func TestRevisionHistoryAndWorkspaceIsolation(t *testing.T) {
	r := repo(t)
	s := NewService(r)
	ctx := context.Background()
	v, e := s.Create(ctx, CreateInput{WorkspaceID: "w1", Name: "Bowl"})
	if e != nil {
		t.Fatal(e)
	}
	v2, e := s.Update(ctx, UpdateInput{WorkspaceID: "w1", ID: v.ID, ExpectedRevision: 1, Name: "Better Bowl"})
	if e != nil || v2.Revision != 2 {
		t.Fatalf("update %#v %v", v2, e)
	}
	if _, e := s.Update(ctx, UpdateInput{WorkspaceID: "w1", ID: v.ID, ExpectedRevision: 1, Name: "Conflict"}); e == nil {
		t.Fatal("stale update accepted")
	}
	old, e := r.Get(ctx, v.ID, "w1")
	if e != nil || old.Name != "Better Bowl" || old.Revision != 2 {
		t.Fatalf("current %#v %v", old, e)
	}
	if _, e := s.Get(ctx, v.ID, "w2"); e == nil {
		t.Fatal("foreign recipe readable")
	}
	items, e := s.List(ctx, "w2")
	if e != nil || len(items) != 0 {
		t.Fatalf("foreign list %#v %v", items, e)
	}
}

func TestCreateIdempotencyReplaysAndRejectsChangedPayload(t *testing.T) {
	s := NewService(repo(t))
	ctx := context.Background()
	first, err := s.Create(ctx, CreateInput{WorkspaceID: "w1", Name: "Soup", IdempotencyKey: "capture-1"})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.Create(ctx, CreateInput{WorkspaceID: "w1", Name: "Soup", IdempotencyKey: "capture-1"})
	if err != nil || replay.ID != first.ID {
		t.Fatalf("replay %#v %v", replay, err)
	}
	if _, err := s.Create(ctx, CreateInput{WorkspaceID: "w1", Name: "Salad", IdempotencyKey: "capture-1"}); err == nil {
		t.Fatal("changed idempotent payload accepted")
	}
}
