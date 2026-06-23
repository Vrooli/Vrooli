package mocks

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"network-manager/internal/notes"

	"github.com/google/uuid"
)

// FakeService satisfies notes.Service for handler tests that don't
// want validation/repository plumbing in scope. Surface is deliberately
// thinner than FakeRepository: handler tests assert on routing, status,
// and envelope shape — they don't need the fake to act like a real
// store. State lives in CreateOut / GetByID / ListOut; CreateInputs
// records what the handler passed in so wire-shape tests can assert on
// it.
//
// CreateOut/CreateErr precedence: CreateErr wins. If neither is set,
// FakeService synthesises a Note with a UUID + the input fields so the
// handler's success path renders something sensible without test
// boilerplate.
type FakeService struct {
	mu sync.Mutex

	// CreateInputs records each Create call's input in order.
	CreateInputs []notes.CreateInput
	// CreateOut, when non-nil, is returned verbatim from Create on
	// success. When nil, FakeService synthesises a Note from the input.
	CreateOut *notes.Note
	CreateErr error

	// GetByID maps id → Note for the success path. Misses fall through
	// to GetErr (if set) or notes.ErrNoteNotFound{ID: id}.
	GetByID map[string]notes.Note
	GetErr  error

	// ListOut is returned verbatim from List on success.
	ListOut []notes.Note
	ListErr error

	// CountOut is returned verbatim from CountInWindow on success;
	// CountErr (if set) wins. CountWindows records each [from, to) the
	// handler resolved so measure tests can assert on the window math.
	CountOut     int
	CountErr     error
	CountWindows [][2]time.Time

	CreateCalls atomic.Int64
	GetCalls    atomic.Int64
	ListCalls   atomic.Int64
	CountCalls  atomic.Int64
}

func (f *FakeService) Create(ctx context.Context, in notes.CreateInput) (notes.Note, error) {
	f.CreateCalls.Add(1)

	f.mu.Lock()
	f.CreateInputs = append(f.CreateInputs, in)
	f.mu.Unlock()

	if f.CreateErr != nil {
		return notes.Note{}, f.CreateErr
	}
	if f.CreateOut != nil {
		return *f.CreateOut, nil
	}
	return notes.Note{
		ID:    uuid.NewString(),
		Title: in.Title,
		Body:  in.Body,
	}, nil
}

func (f *FakeService) Get(ctx context.Context, id string) (notes.Note, error) {
	f.GetCalls.Add(1)
	if f.GetErr != nil {
		return notes.Note{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if n, ok := f.GetByID[id]; ok {
		return n, nil
	}
	return notes.Note{}, notes.ErrNoteNotFound{ID: id}
}

func (f *FakeService) List(ctx context.Context, limit int) ([]notes.Note, error) {
	f.ListCalls.Add(1)
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ListOut) == 0 {
		return nil, nil
	}
	out := make([]notes.Note, len(f.ListOut))
	copy(out, f.ListOut)
	return out, nil
}

func (f *FakeService) CountInWindow(ctx context.Context, from, to time.Time) (int, error) {
	f.CountCalls.Add(1)
	f.mu.Lock()
	f.CountWindows = append(f.CountWindows, [2]time.Time{from, to})
	f.mu.Unlock()
	if f.CountErr != nil {
		return 0, f.CountErr
	}
	return f.CountOut, nil
}

// Compile-time guarantee.
var _ notes.Service = (*FakeService)(nil)
