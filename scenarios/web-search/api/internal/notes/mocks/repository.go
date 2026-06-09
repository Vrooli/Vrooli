// Package mocks holds notes-domain test fakes co-located with the
// domain they double for. Living next to internal/notes/ means deleting
// the domain folder takes its mocks with it (no central residue) and
// the package graph reflects the ownership: mocks imports notes; notes
// does not import mocks.
package mocks

import (
	"context"
	"time"

	"web-search/internal/notes"
	"web-search/internal/testutil/repokit"
)

// FakeRepository satisfies notes.Repository for service tests that don't
// want the sqlite round-trip. It embeds repokit.SliceRepo so the CRUD
// behaviour and self-tests live in the substrate package; this file only
// declares the notes-domain wiring (ID accessors + typed sentinel) plus
// the domain-specific Count seam the generic substrate can't express
// (it has no notion of CreatedAt).
//
// Construction shape: tests use NewFakeRepository() to get a struct with
// extractors pre-wired, then mutate fields (Items, CreateErr, CountOut,
// etc.) for arrangement.
type FakeRepository struct {
	*repokit.SliceRepo[notes.Note]

	// CountOut is returned verbatim from Count on success; CountErr (if
	// set) wins. CountWindows records each [from, to) the caller resolved.
	CountOut     int
	CountErr     error
	CountWindows [][2]time.Time
}

// NewFakeRepository returns a FakeRepository with notes-domain
// extractors and the typed not-found sentinel pre-wired. Callers
// subsequently mutate fields (Items, CreateErr, GetErr, ListErr,
// CountOut) to arrange the test.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{
		SliceRepo: repokit.NewSliceRepo[notes.Note](
			func(n notes.Note) string { return n.ID },
			func(n *notes.Note, id string) { n.ID = id },
			func(id string) error { return notes.ErrNoteNotFound{ID: id} },
		),
	}
}

// Count records the window and returns the arranged CountOut/CountErr. It
// does not filter Items by time — service/measure tests assert on the
// resolved window, and the real sqlite repository owns the range query.
func (f *FakeRepository) Count(ctx context.Context, from, to time.Time) (int, error) {
	f.CountWindows = append(f.CountWindows, [2]time.Time{from, to})
	if f.CountErr != nil {
		return 0, f.CountErr
	}
	return f.CountOut, nil
}

// Compile-time guarantee that *FakeRepository satisfies notes.Repository.
var _ notes.Repository = (*FakeRepository)(nil)
