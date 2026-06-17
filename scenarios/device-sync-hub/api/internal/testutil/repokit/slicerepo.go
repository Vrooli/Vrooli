package repokit

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

// SliceRepo is a generic in-memory CRUD fake satisfying any per-domain
// Repository interface with the shape:
//
//	Create(ctx, T) (T, error)
//	Get(ctx, id string) (T, error)
//	List(ctx, limit int) ([]T, error)
//
// The domain plugs in three things via fields on the struct:
//
//   - GetID, SetID: ID-field accessors. SliceRepo assigns a UUID on Create
//     when the input's ID is empty, mirroring real-store behaviour.
//   - NotFound: typed-sentinel constructor; Get returns this on miss so
//     handler-side errors.As routing keeps working.
//
// Failure injection is per-method (CreateErr, GetErr, ListErr) so a single
// test can prove behaviour on one failure mode without poisoning the
// others. The mu/atomic separation is preserved so go test -race stays
// quiet when callers fan out.
//
// Construction: most callers want NewSliceRepo (or a per-domain helper
// like devices/mocks::NewFakeRepository) which returns a struct with the
// extractors pre-wired. Bare zero-value SliceRepo{} is also valid for
// tests that override defaults (e.g. injecting a custom NotFound), but
// callers MUST set GetID, SetID, and NotFound before exercising the
// methods — the helpers below panic with a clear diagnostic if any are
// nil at the point of use.
type SliceRepo[T any] struct {
	mu sync.Mutex

	// Items is the in-memory store. Tests that arrange existing rows
	// assign here; Create appends; Get scans linearly.
	Items []T

	// GetID returns the ID field of an item. Required.
	GetID func(T) string
	// SetID sets the ID field of an item. Required.
	SetID func(*T, string)
	// NotFound constructs the typed sentinel returned by Get on miss.
	// Required.
	NotFound func(id string) error

	// Per-method failure injection. Each takes precedence over normal
	// behaviour and prevents state mutation.
	CreateErr error
	GetErr    error
	ListErr   error

	CreateCalls atomic.Int64
	GetCalls    atomic.Int64
	ListCalls   atomic.Int64
}

// NewSliceRepo returns a SliceRepo wired with the supplied extractors and
// not-found constructor. Per-method error fields and Items default to
// zero; tests override them after construction.
func NewSliceRepo[T any](
	getID func(T) string,
	setID func(*T, string),
	notFound func(id string) error,
) *SliceRepo[T] {
	return &SliceRepo[T]{
		GetID:    getID,
		SetID:    setID,
		NotFound: notFound,
	}
}

// Create appends item to Items. If item's ID is empty, SliceRepo assigns
// a UUID first. Returns CreateErr if set, before mutating state — keeps
// the failure path observable as "didn't insert" via len(Items).
func (r *SliceRepo[T]) Create(ctx context.Context, item T) (T, error) {
	r.CreateCalls.Add(1)
	if r.CreateErr != nil {
		var zero T
		return zero, r.CreateErr
	}
	r.requireExtractors("Create")

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.GetID(item) == "" {
		r.SetID(&item, uuid.NewString())
	}
	r.Items = append(r.Items, item)
	return item, nil
}

// Get linear-scans Items for an ID match. Returns GetErr if set
// (overrides any in-memory match) so tests can drive the not-found and
// internal-error paths independently.
func (r *SliceRepo[T]) Get(ctx context.Context, id string) (T, error) {
	r.GetCalls.Add(1)
	if r.GetErr != nil {
		var zero T
		return zero, r.GetErr
	}
	r.requireExtractors("Get")
	if r.NotFound == nil {
		panic("repokit.SliceRepo.Get: NotFound is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, item := range r.Items {
		if r.GetID(item) == id {
			return item, nil
		}
	}
	var zero T
	return zero, r.NotFound(id)
}

// List returns up to `limit` items in their insertion order — newest
// inserts at the tail, but tests that care about ordering should use the
// real repository. The fake's job is to drive service shape, not pin SQL
// ordering. Returns ListErr if set.
//
// limit <= 0 returns no rows; that mirrors the production repository
// contract (callers requesting "all" pass an explicit upper bound).
func (r *SliceRepo[T]) List(ctx context.Context, limit int) ([]T, error) {
	r.ListCalls.Add(1)
	if r.ListErr != nil {
		return nil, r.ListErr
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if limit <= 0 || len(r.Items) == 0 {
		return nil, nil
	}
	if limit > len(r.Items) {
		limit = len(r.Items)
	}
	out := make([]T, limit)
	copy(out, r.Items[:limit])
	return out, nil
}

func (r *SliceRepo[T]) requireExtractors(op string) {
	if r.GetID == nil {
		panic(fmt.Sprintf("repokit.SliceRepo.%s: GetID is required", op))
	}
	if op == "Create" && r.SetID == nil {
		panic(fmt.Sprintf("repokit.SliceRepo.%s: SetID is required", op))
	}
}
