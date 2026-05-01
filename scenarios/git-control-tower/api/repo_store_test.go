package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

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

func TestSQLiteRepoStore_UpsertUpdatesExistingRecord(t *testing.T) {
	store := newTestRepoStore(t)
	ctx := context.Background()
	firstNow := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	secondNow := firstNow.Add(time.Hour)
	store.now = func() time.Time { return firstNow }

	repo, err := store.Upsert(ctx, RepoRecord{
		Path:      "/work/vrooli",
		Name:      "vrooli",
		RemoteURL: "git@github.com:example/vrooli.git",
		Favorite:  true,
	})
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	store.now = func() time.Time { return secondNow }
	updated, err := store.Upsert(ctx, RepoRecord{
		Path:     "/work/vrooli",
		Name:     "Vrooli",
		Favorite: false,
	})
	if err != nil {
		t.Fatalf("update repo: %v", err)
	}
	if updated.ID != repo.ID {
		t.Fatalf("updated ID = %d, want original %d", updated.ID, repo.ID)
	}
	if updated.Name != "Vrooli" {
		t.Fatalf("Name = %q, want Vrooli", updated.Name)
	}
	if updated.RemoteURL != "git@github.com:example/vrooli.git" {
		t.Fatalf("RemoteURL = %q, want original remote preserved", updated.RemoteURL)
	}
	if updated.Favorite {
		t.Fatal("Favorite = true, want false")
	}
	if updated.LastOpenedAt != secondNow.Format(time.RFC3339) {
		t.Fatalf("LastOpenedAt = %q, want %q", updated.LastOpenedAt, secondNow.Format(time.RFC3339))
	}
}

func TestSQLiteRepoStore_GetDeleteAndClearActive(t *testing.T) {
	store := newTestRepoStore(t)
	ctx := context.Background()

	repo, err := store.Upsert(ctx, RepoRecord{Path: "/work/vrooli", Name: "vrooli"})
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	byID, err := store.GetByID(ctx, repo.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if byID.Path != repo.Path {
		t.Fatalf("GetByID path = %q, want %q", byID.Path, repo.Path)
	}
	byPath, err := store.GetByPath(ctx, repo.Path)
	if err != nil {
		t.Fatalf("get by path: %v", err)
	}
	if byPath.ID != repo.ID {
		t.Fatalf("GetByPath id = %d, want %d", byPath.ID, repo.ID)
	}

	if err := store.SetActive(ctx, repo.ID); err != nil {
		t.Fatalf("set active: %v", err)
	}
	if err := store.ClearActive(ctx); err != nil {
		t.Fatalf("clear active: %v", err)
	}
	active, err := store.GetActive(ctx)
	if err != nil {
		t.Fatalf("get active after clear: %v", err)
	}
	if active != nil {
		t.Fatalf("active repo = %#v, want nil", active)
	}

	if err := store.Delete(ctx, repo.ID); err != nil {
		t.Fatalf("delete repo: %v", err)
	}
	if _, err := store.GetByID(ctx, repo.ID); err != sql.ErrNoRows {
		t.Fatalf("GetByID after delete error = %v, want sql.ErrNoRows", err)
	}
	if err := store.Delete(ctx, repo.ID); err != sql.ErrNoRows {
		t.Fatalf("delete missing error = %v, want sql.ErrNoRows", err)
	}
}

func TestSQLiteRepoStore_ValidationAndTouch(t *testing.T) {
	store := newTestRepoStore(t)
	ctx := context.Background()

	if _, err := store.Upsert(ctx, RepoRecord{Name: "missing-path"}); err == nil {
		t.Fatal("Upsert missing path error = nil, want error")
	}
	if _, err := store.Upsert(ctx, RepoRecord{Path: "/work/missing-name"}); err == nil {
		t.Fatal("Upsert missing name error = nil, want error")
	}
	if err := store.Delete(ctx, 0); err == nil {
		t.Fatal("Delete id=0 error = nil, want error")
	}
	if err := store.SetActive(ctx, 0); err == nil {
		t.Fatal("SetActive id=0 error = nil, want error")
	}
	if err := store.TouchLastOpened(ctx, 0); err == nil {
		t.Fatal("TouchLastOpened id=0 error = nil, want error")
	}

	repo, err := store.Upsert(ctx, RepoRecord{Path: "/work/vrooli", Name: "vrooli"})
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	if err := store.TouchLastOpened(ctx, repo.ID); err != nil {
		t.Fatalf("touch repo: %v", err)
	}
	touched, err := store.GetByID(ctx, repo.ID)
	if err != nil {
		t.Fatalf("get touched repo: %v", err)
	}
	if touched.LastOpenedAt != now.Format(time.RFC3339) {
		t.Fatalf("LastOpenedAt = %q, want %q", touched.LastOpenedAt, now.Format(time.RFC3339))
	}
}
