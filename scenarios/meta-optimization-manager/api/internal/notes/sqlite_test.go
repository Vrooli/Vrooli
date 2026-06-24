package notes_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"meta-optimization-manager/internal/notes"
	"meta-optimization-manager/internal/testutil/db"
	"meta-optimization-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "meta-optimization-manager/internal/database"
)

// newSchemaDB returns a sqlite handle with the production schema
// already applied. This is the canonical compose pattern for repository
// tests: db.NewSQLite for the connection + apidb.EnsureSchemas with the
// system + per-domain providers. Helper exists to keep individual tests
// focused on the behavior under exercise.
func newSchemaDB(t *testing.T) *testRepo {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(notes.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return &testRepo{
		repo:        notes.NewSQLiteRepository(d, clk),
		attachments: notes.NewSQLiteAttachmentsRepository(d, clk),
		clock:       clk,
	}
}

type testRepo struct {
	repo        notes.Repository
	attachments notes.AttachmentsRepository
	clock       *mocks.FakeClock
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

// TestSQLiteRepository_CountInWindow pins the aggregate the notes.count
// measure computes: COUNT(*) over the half-open [from, to) created_at range.
// Notes are written at distinct clock instants, then the count is asserted to
// include the lower bound and exclude the upper bound.
func TestSQLiteRepository_CountInWindow(t *testing.T) {
	tr := newSchemaDB(t)
	ctx := context.Background()

	// Three notes at 12:00, 12:01, 12:02 on 2026-05-01 (the seeded clock).
	base := tr.clock.Now()
	for i := 0; i < 3; i++ {
		_, err := tr.repo.Create(ctx, notes.Note{Title: "n"})
		require.NoError(t, err)
		tr.clock.Advance(time.Minute)
	}

	// [12:00, 12:02) includes the first two, excludes the third (12:02).
	n, err := tr.repo.Count(ctx, base, base.Add(2*time.Minute))
	require.NoError(t, err)
	require.Equal(t, 2, n)

	// A window before any note returns zero, never an error.
	n, err = tr.repo.Count(ctx, base.Add(-time.Hour), base)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	// A window covering all three.
	n, err = tr.repo.Count(ctx, base, base.Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, 3, n)
}

func TestSQLiteRepository_AttachmentMetadataRoundTrip(t *testing.T) {
	tr := newSchemaDB(t)
	ctx := context.Background()
	created, err := tr.repo.Create(ctx, notes.Note{Title: "with attachment"})
	require.NoError(t, err)

	attachment, err := tr.attachments.CreateAttachment(ctx, notes.Attachment{
		Key:       "notes/" + created.ID + "/attachments/file.txt",
		NoteID:    created.ID,
		MIMEType:  "text/plain",
		SizeBytes: 12,
	})
	require.NoError(t, err)
	require.Equal(t, tr.clock.Now(), attachment.UploadedAt)

	keys, err := tr.attachments.ListAttachmentKeys(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, []string{attachment.Key}, keys)

	got, err := tr.repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, []string{attachment.Key}, got.AttachmentKeys)
}
