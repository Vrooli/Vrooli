// Package mocks holds notes-domain test fakes co-located with the
// domain they double for. Living next to internal/notes/ means deleting
// the domain folder takes its mocks with it (no central residue) and
// the package graph reflects the ownership: mocks imports notes; notes
// does not import mocks.
package mocks

import (
	"react-component-library/internal/notes"
	"react-component-library/internal/testutil/repokit"
)

// FakeRepository satisfies notes.Repository for service tests that don't
// want the sqlite round-trip. Aliased onto repokit.SliceRepo so the
// behaviour and self-tests live in the substrate package; this file only
// declares the notes-domain wiring (ID accessors + typed sentinel).
//
// Construction shape: tests use NewFakeRepository() to get a struct with
// extractors pre-wired, then mutate fields (Items, CreateErr, etc.) for
// arrangement.
type FakeRepository = repokit.SliceRepo[notes.Note]

// NewFakeRepository returns a FakeRepository with notes-domain
// extractors and the typed not-found sentinel pre-wired. Callers
// subsequently mutate fields (Items, CreateErr, GetErr, ListErr) to
// arrange the test.
func NewFakeRepository() *FakeRepository {
	return repokit.NewSliceRepo[notes.Note](
		func(n notes.Note) string { return n.ID },
		func(n *notes.Note, id string) { n.ID = id },
		func(id string) error { return notes.ErrNoteNotFound{ID: id} },
	)
}

// Compile-time guarantee that *FakeRepository satisfies notes.Repository.
var _ notes.Repository = (*FakeRepository)(nil)
