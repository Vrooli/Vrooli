package main

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestRepoStore(t *testing.T) *SQLiteRepoStore {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := ensureRepoSchema(db); err != nil {
		db.Close()
		t.Fatalf("ensure repo schema: %v", err)
	}
	store := NewSQLiteRepoStore(db)
	t.Cleanup(func() {
		_ = db.Close()
	})
	return store
}

func TestSQLiteRepoStore_UpsertAndList(t *testing.T) {
	store := newTestRepoStore(t)
	ctx := context.Background()

	repo, err := store.Upsert(ctx, RepoRecord{Path: "/work/vrooli", Name: "vrooli"})
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}
	if repo.ID == 0 {
		t.Fatalf("expected repo id to be set")
	}

	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list repos: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(list))
	}
	if list[0].Path != "/work/vrooli" {
		t.Fatalf("expected path %q, got %q", "/work/vrooli", list[0].Path)
	}
}

func TestSQLiteRepoStore_ActiveRepo(t *testing.T) {
	store := newTestRepoStore(t)
	ctx := context.Background()

	repo, err := store.Upsert(ctx, RepoRecord{Path: "/work/vrooli", Name: "vrooli"})
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	if err := store.SetActive(ctx, repo.ID); err != nil {
		t.Fatalf("set active: %v", err)
	}
	active, err := store.GetActive(ctx)
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if active == nil || active.ID != repo.ID {
		t.Fatalf("expected active repo id %d, got %#v", repo.ID, active)
	}
}
