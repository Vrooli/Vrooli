package notes_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"{{SCENARIO_ID}}/internal/notes"
	"{{SCENARIO_ID}}/internal/store"
	"{{SCENARIO_ID}}/internal/testutil/db"
	"{{SCENARIO_ID}}/internal/testutil/mocks"
)

// newSchemaDB returns a sqlite handle with the production schema
// already applied. This is the canonical compose pattern for repository
// tests: db.NewSQLite for the connection, store.EnsureSchema for the
// tables. Helper exists to keep individual tests focused on the
// behavior under exercise.
func newSchemaDB(t *testing.T) *testRepo {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, store.EnsureSchema(context.Background(), d))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return &testRepo{
		repo:  notes.NewSQLiteRepository(d, clk),
		clock: clk,
	}
}

type testRepo struct {
	repo  notes.Repository
	clock *mocks.FakeClock
}

// TestSQLiteRepository_CreateAndGetRoundTrip pins the canonical
// write/read path: a Create returns the persisted Note (ID + timestamps
// populated), and a subsequent Get returns the same row byte-identical.
func TestSQLiteRepository_CreateAndGetRoundTrip(t *testing.T) {
	tr := newSchemaDB(t)
	ctx := context.Background()

	created, err := tr.repo.Create(ctx, notes.Note{Title: "first", Body: "hello"})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID, "Create must populate ID")
	require.Equal(t, tr.clock.Now(), created.CreatedAt, "CreatedAt must come from clock")
	require.Equal(t, created.CreatedAt, created.UpdatedAt, "UpdatedAt == CreatedAt on first insert")

	got, err := tr.repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "first", got.Title)
	require.Equal(t, "hello", got.Body)
	require.True(t, created.CreatedAt.Equal(got.CreatedAt),
		"round-trip CreatedAt: created=%v got=%v", created.CreatedAt, got.CreatedAt)
}

// TestSQLiteRepository_GetReturnsNotFound pins the typed-error contract
// the service depends on: a missing ID surfaces ErrNoteNotFound, not a
// generic sql.ErrNoRows that would leak storage-layer detail through
// the boundary.
func TestSQLiteRepository_GetReturnsNotFound(t *testing.T) {
	tr := newSchemaDB(t)
	_, err := tr.repo.Get(context.Background(), "missing-id")
	require.Error(t, err)
	var nf notes.ErrNoteNotFound
	require.True(t, errors.As(err, &nf), "err must be ErrNoteNotFound, got %T: %v", err, err)
	require.Equal(t, "missing-id", nf.ID)
}

// TestSQLiteRepository_ListOrdersByCreatedDesc pins the wire-visible
// ordering contract. UI and CLI both render notes newest-first; if the
// SQL changes and stops respecting that, the test catches it before a
// scenario does.
func TestSQLiteRepository_ListOrdersByCreatedDesc(t *testing.T) {
	tr := newSchemaDB(t)
	ctx := context.Background()

	first, err := tr.repo.Create(ctx, notes.Note{Title: "first"})
	require.NoError(t, err)
	tr.clock.Advance(time.Second)
	second, err := tr.repo.Create(ctx, notes.Note{Title: "second"})
	require.NoError(t, err)
	tr.clock.Advance(time.Second)
	third, err := tr.repo.Create(ctx, notes.Note{Title: "third"})
	require.NoError(t, err)

	got, err := tr.repo.List(ctx, 10)
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, third.ID, got[0].ID, "newest first")
	require.Equal(t, second.ID, got[1].ID)
	require.Equal(t, first.ID, got[2].ID)
}

// TestSQLiteRepository_ListRespectsLimit pins the limit semantics so
// future cursor pagination doesn't accidentally regress the upper-bound
// guarantee callers rely on.
func TestSQLiteRepository_ListRespectsLimit(t *testing.T) {
	tr := newSchemaDB(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := tr.repo.Create(ctx, notes.Note{Title: "n"})
		require.NoError(t, err)
		tr.clock.Advance(time.Second)
	}

	got, err := tr.repo.List(ctx, 2)
	require.NoError(t, err)
	require.Len(t, got, 2)

	none, err := tr.repo.List(ctx, 0)
	require.NoError(t, err)
	require.Empty(t, none, "limit <= 0 returns no rows")
}

// TestSQLiteRepository_CreatePopulatesTimestamps pins the timestamp
// generation contract: when callers leave CreatedAt/UpdatedAt zero, the
// repository fills them from the clock seam (so tests can advance time
// deterministically).
func TestSQLiteRepository_CreatePopulatesTimestamps(t *testing.T) {
	tr := newSchemaDB(t)
	ctx := context.Background()

	created, err := tr.repo.Create(ctx, notes.Note{Title: "t"})
	require.NoError(t, err)
	require.False(t, created.CreatedAt.IsZero())
	require.False(t, created.UpdatedAt.IsZero())
	require.True(t, created.CreatedAt.Equal(created.UpdatedAt))
	require.Equal(t, tr.clock.Now(), created.CreatedAt)
}
