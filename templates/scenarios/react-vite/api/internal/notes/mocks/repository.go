// Package mocks holds notes-domain test fakes co-located with the
// domain they double for. Living next to internal/notes/ means deleting
// the domain folder takes its mocks with it (no central residue) and
// the package graph reflects the ownership: mocks imports notes; notes
// does not import mocks.
package mocks

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"

	"{{SCENARIO_ID}}/internal/notes"
)

// FakeRepository satisfies notes.Repository for service tests that
// don't want the sqlite round-trip. Arrange via field mutation; failure
// injection is per-method (CreateErr, GetErr, ListErr) so a single test
// can prove service behavior on one failure mode without poisoning the
// others. Calls counters use atomic.Int64 so go test -race stays quiet
// when callers fan out.
type FakeRepository struct {
	mu sync.Mutex

	Notes []notes.Note

	CreateErr error
	GetErr    error
	ListErr   error

	CreateCalls atomic.Int64
	GetCalls    atomic.Int64
	ListCalls   atomic.Int64
}

// Create appends n to the in-memory slice. If n.ID is empty the fake
// assigns a UUID so service tests don't have to pre-populate it.
// Returns CreateErr if set, before mutating state — keeps the failure
// path observable as "didn't insert" via len(f.Notes).
func (f *FakeRepository) Create(ctx context.Context, n notes.Note) (notes.Note, error) {
	f.CreateCalls.Add(1)
	if f.CreateErr != nil {
		return notes.Note{}, f.CreateErr
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	f.Notes = append(f.Notes, n)
	return n, nil
}

// Get linear-scans Notes for an ID match. Returns GetErr if set
// (overrides any in-memory match) so tests can drive the not-found and
// internal-error paths independently.
func (f *FakeRepository) Get(ctx context.Context, id string) (notes.Note, error) {
	f.GetCalls.Add(1)
	if f.GetErr != nil {
		return notes.Note{}, f.GetErr
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	for _, n := range f.Notes {
		if n.ID == id {
			return n, nil
		}
	}
	return notes.Note{}, notes.ErrNoteNotFound{ID: id}
}

// List returns up to `limit` notes in their insertion order — newest
// inserts at the tail, but tests that care about ordering should use
// the real sqlite repository. The fake's job is to drive service shape,
// not pin SQL ordering. Returns ListErr if set.
func (f *FakeRepository) List(ctx context.Context, limit int) ([]notes.Note, error) {
	f.ListCalls.Add(1)
	if f.ListErr != nil {
		return nil, f.ListErr
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if limit <= 0 || len(f.Notes) == 0 {
		return nil, nil
	}
	if limit > len(f.Notes) {
		limit = len(f.Notes)
	}
	out := make([]notes.Note, limit)
	copy(out, f.Notes[:limit])
	return out, nil
}

// Compile-time guarantee that *FakeRepository satisfies notes.Repository.
var _ notes.Repository = (*FakeRepository)(nil)
