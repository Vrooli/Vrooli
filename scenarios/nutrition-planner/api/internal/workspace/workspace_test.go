package workspace

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func testRepo(t *testing.T) Repository {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	return NewSQLiteRepository(db, schedule.System())
}

func TestWorkspaceOwnershipAndIdempotency(t *testing.T) {
	r := testRepo(t)
	ctx := context.Background()
	w, err := NewService(r).Create(ctx, CreateInput{Name: " Home ", OwnerSubject: "alice", IdempotencyKey: "k1"})
	if err != nil {
		t.Fatal(err)
	}
	again, err := NewService(r).Create(ctx, CreateInput{Name: "Home", OwnerSubject: "alice", IdempotencyKey: "k1"})
	if err != nil || again.ID != w.ID {
		t.Fatalf("retry got %#v, %v", again, err)
	}
	if _, err := NewService(r).Create(ctx, CreateInput{Name: "Other", OwnerSubject: "alice", IdempotencyKey: "k1"}); err == nil {
		t.Fatal("changed payload did not conflict")
	}
	if _, err := NewService(r).Get(ctx, w.ID, "bob"); err == nil {
		t.Fatal("foreign workspace was readable")
	}
	items, err := NewService(r).List(ctx, "bob")
	if err != nil || len(items) != 0 {
		t.Fatalf("foreign list: %#v, %v", items, err)
	}
}
