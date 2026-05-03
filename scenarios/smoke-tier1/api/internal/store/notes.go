package store

import (
	"context"
	"fmt"
	"time"
)

// Note is the internal domain shape for a note. Distinct from the
// proto wire type at packages/proto/gen/go/.../v1/notes.Note — handlers
// translate at the boundary so the domain layer never imports proto
// (api-steer §7).
type Note struct {
	ID        string
	Title     string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NoteStore is the repository seam the notes handlers depend on.
// Production wires the sqlite-backed implementation from notes_sqlite.go;
// tests wire testutil/mocks.FakeNoteStore. New methods land here when a
// handler proves it needs them — keep the surface narrow.
type NoteStore interface {
	// Create persists n. The implementation populates ID, CreatedAt,
	// and UpdatedAt; callers leaving those zero-valued is the canonical
	// shape. Returns the persisted Note (with the populated fields).
	Create(ctx context.Context, n Note) (Note, error)

	// Get returns the note with the given ID or ErrNoteNotFound{ID}
	// when no row matches. Wrapped error types other than
	// ErrNoteNotFound mean an underlying storage failure.
	Get(ctx context.Context, id string) (Note, error)

	// List returns up to `limit` notes ordered newest-first by
	// CreatedAt. limit <= 0 returns no rows; callers requesting "all"
	// pass an explicit upper bound.
	List(ctx context.Context, limit int) ([]Note, error)
}

// ErrNoteNotFound is the typed sentinel returned by NoteStore.Get when
// no row matches. Handlers translate via errors.As into a 404 response
// carrying httpx.CodeNotFound.
type ErrNoteNotFound struct {
	ID string
}

func (e ErrNoteNotFound) Error() string {
	return fmt.Sprintf("note %q not found", e.ID)
}
