package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"smoke-tier1/internal/store"
	"smoke-tier1/internal/testutil/db"
	"smoke-tier1/internal/testutil/mocks"
)

// newSchemaDB returns a sqlite handle with the production schema
// already applied. This is the canonical compose pattern for repository
// tests: db.NewSQLite for the connection, store.EnsureSchema for the
// tables. Helper exists to keep individual tests focused on the
// behavior under exercise.
func newSchemaDB(t *testing.T) *testStore {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, store.EnsureSchema(context.Background(), d))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return &testStore{
		store: store.NewSQLiteNoteStore(d, clk),
		clock: clk,
	}
}

type testStore struct {
	store store.NoteStore
	clock *mocks.FakeClock
}

// TestSQLiteNoteStore_CreateAndGetRoundTrip pins the canonical write/read
// path: a Create returns the persisted Note (ID + timestamps populated),
// and a subsequent Get returns the same row byte-identical.
func TestSQLiteNoteStore_CreateAndGetRoundTrip(t *testing.T) {
	ts := newSchemaDB(t)
	ctx := context.Background()

	created, err := ts.store.Create(ctx, store.Note{Title: "first", Body: "hello"})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID, "Create must populate ID")
	require.Equal(t, ts.clock.Now(), created.CreatedAt, "CreatedAt must come from clock")
	require.Equal(t, created.CreatedAt, created.UpdatedAt, "UpdatedAt == CreatedAt on first insert")

	got, err := ts.store.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "first", got.Title)
	require.Equal(t, "hello", got.Body)
	require.True(t, created.CreatedAt.Equal(got.CreatedAt),
		"round-trip CreatedAt: created=%v got=%v", created.CreatedAt, got.CreatedAt)
}

// TestSQLiteNoteStore_GetReturnsNotFound pins the typed-error contract
// handlers depend on: a missing ID surfaces ErrNoteNotFound, not a
// generic sql.ErrNoRows that would leak storage-layer detail through
// the boundary.
func TestSQLiteNoteStore_GetReturnsNotFound(t *testing.T) {
	ts := newSchemaDB(t)
	_, err := ts.store.Get(context.Background(), "missing-id")
	require.Error(t, err)
	var nf store.ErrNoteNotFound
	require.True(t, errors.As(err, &nf), "err must be ErrNoteNotFound, got %T: %v", err, err)
	require.Equal(t, "missing-id", nf.ID)
}

// TestSQLiteNoteStore_ListOrdersByCreatedDesc pins the wire-visible
// ordering contract. UI and CLI both render notes newest-first; if the
// SQL changes and stops respecting that, the test catches it before a
// scenario does.
func TestSQLiteNoteStore_ListOrdersByCreatedDesc(t *testing.T) {
	ts := newSchemaDB(t)
	ctx := context.Background()

	first, err := ts.store.Create(ctx, store.Note{Title: "first"})
	require.NoError(t, err)
	ts.clock.Advance(time.Second)
	second, err := ts.store.Create(ctx, store.Note{Title: "second"})
	require.NoError(t, err)
	ts.clock.Advance(time.Second)
	third, err := ts.store.Create(ctx, store.Note{Title: "third"})
	require.NoError(t, err)

	got, err := ts.store.List(ctx, 10)
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, third.ID, got[0].ID, "newest first")
	require.Equal(t, second.ID, got[1].ID)
	require.Equal(t, first.ID, got[2].ID)
}

// TestSQLiteNoteStore_ListRespectsLimit pins the limit semantics so
// future cursor pagination doesn't accidentally regress the upper-bound
// guarantee callers rely on.
func TestSQLiteNoteStore_ListRespectsLimit(t *testing.T) {
	ts := newSchemaDB(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := ts.store.Create(ctx, store.Note{Title: "n"})
		require.NoError(t, err)
		ts.clock.Advance(time.Second)
	}

	got, err := ts.store.List(ctx, 2)
	require.NoError(t, err)
	require.Len(t, got, 2)

	none, err := ts.store.List(ctx, 0)
	require.NoError(t, err)
	require.Empty(t, none, "limit <= 0 returns no rows")
}

// TestSQLiteNoteStore_CreatePopulatesTimestamps pins the timestamp
// generation contract: when callers leave CreatedAt/UpdatedAt zero, the
// store fills them from the clock seam (so tests can advance time
// deterministically).
func TestSQLiteNoteStore_CreatePopulatesTimestamps(t *testing.T) {
	ts := newSchemaDB(t)
	ctx := context.Background()

	created, err := ts.store.Create(ctx, store.Note{Title: "t"})
	require.NoError(t, err)
	require.False(t, created.CreatedAt.IsZero())
	require.False(t, created.UpdatedAt.IsZero())
	require.True(t, created.CreatedAt.Equal(created.UpdatedAt))
	require.Equal(t, ts.clock.Now(), created.CreatedAt)
}
