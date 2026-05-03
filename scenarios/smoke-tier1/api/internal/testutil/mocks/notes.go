package mocks

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"

	"smoke-tier1/internal/store"
)

// FakeNoteStore satisfies store.NoteStore for handler tests that don't
// want the sqlite round-trip. Arrange via field mutation; failure
// injection is per-method (CreateErr, GetErr, ListErr) so a single test
// can prove handler behavior on one failure mode without poisoning the
// others. Calls counters use atomic.Int64 so go test -race stays quiet
// when handlers fan out.
type FakeNoteStore struct {
	mu sync.Mutex

	Notes []store.Note

	CreateErr error
	GetErr    error
	ListErr   error

	CreateCalls atomic.Int64
	GetCalls    atomic.Int64
	ListCalls   atomic.Int64
}

// Create appends n to the in-memory slice. If n.ID is empty the fake
// assigns a UUID so handler tests don't have to pre-populate it. Returns
// CreateErr if set, before mutating state — keeps the failure path
// observable as "didn't insert" via len(f.Notes).
func (f *FakeNoteStore) Create(ctx context.Context, n store.Note) (store.Note, error) {
	f.CreateCalls.Add(1)
	if f.CreateErr != nil {
		return store.Note{}, f.CreateErr
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
func (f *FakeNoteStore) Get(ctx context.Context, id string) (store.Note, error) {
	f.GetCalls.Add(1)
	if f.GetErr != nil {
		return store.Note{}, f.GetErr
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	for _, n := range f.Notes {
		if n.ID == id {
			return n, nil
		}
	}
	return store.Note{}, store.ErrNoteNotFound{ID: id}
}

// List returns up to `limit` notes in their insertion order — newest
// inserts at the tail, but tests that care about ordering should use
// the real sqlite store. The fake's job is to drive handler shape, not
// pin SQL ordering. Returns ListErr if set.
func (f *FakeNoteStore) List(ctx context.Context, limit int) ([]store.Note, error) {
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
	out := make([]store.Note, limit)
	copy(out, f.Notes[:limit])
	return out, nil
}

// Compile-time guarantee that *FakeNoteStore satisfies store.NoteStore.
var _ store.NoteStore = (*FakeNoteStore)(nil)
