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
	"time"
	"scenario-to-ios/internal/notes"

	"github.com/google/uuid"
)

// FakeRepository satisfies notes.Repository for service tests that don't
// want the sqlite round-trip. It intentionally keeps its in-memory CRUD
// behavior local to this domain package so production-shaped files do not
// import internal/testutil packages.
//
// Construction shape: tests use NewFakeRepository() to get a struct with
// extractors pre-wired, then mutate fields (Items, CreateErr, CountOut,
// etc.) for arrangement.
type FakeRepository struct {
	mu sync.Mutex

	// Items is the in-memory store. Tests arrange existing rows here;
	// Create appends; Get scans linearly; List returns insertion order.
	Items []notes.Note

	CreateErr error
	GetErr    error
	ListErr   error

	CreateCalls atomic.Int64
	GetCalls    atomic.Int64
	ListCalls   atomic.Int64

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
	return &FakeRepository{}
}

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
	f.Items = append(f.Items, n)
	return n, nil
}

func (f *FakeRepository) Get(ctx context.Context, id string) (notes.Note, error) {
	f.GetCalls.Add(1)
	if f.GetErr != nil {
		return notes.Note{}, f.GetErr
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	for _, n := range f.Items {
		if n.ID == id {
			return n, nil
		}
	}
	return notes.Note{}, notes.ErrNoteNotFound{ID: id}
}

func (f *FakeRepository) List(ctx context.Context, limit int) ([]notes.Note, error) {
	f.ListCalls.Add(1)
	if f.ListErr != nil {
		return nil, f.ListErr
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if limit <= 0 || len(f.Items) == 0 {
		return nil, nil
	}
	if limit > len(f.Items) {
		limit = len(f.Items)
	}
	out := make([]notes.Note, limit)
	copy(out, f.Items[:limit])
	return out, nil
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
