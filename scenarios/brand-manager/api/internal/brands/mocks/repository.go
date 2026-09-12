// Package mocks provides in-memory fakes for the brands domain seams so service
// unit tests run without the sqlite round-trip. Mirrors internal/notes/mocks.
package mocks

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"brand-manager/internal/brands"

	"github.com/google/uuid"
)

// FakeRepository satisfies brands.Repository with an in-memory map keyed by ID.
// Per-method error knobs drive failure paths; atomic counters keep `go test
// -race` quiet under fan-out.
type FakeRepository struct {
	mu     sync.Mutex
	brands map[string]brands.Brand

	CreateErr error
	GetErr    error
	ListErr   error
	UpdateErr error
	DeleteErr error

	CreateCalls atomic.Int64
	UpdateCalls atomic.Int64
	DeleteCalls atomic.Int64
}

func (f *FakeRepository) Create(_ context.Context, b brands.Brand) (brands.Brand, error) {
	f.CreateCalls.Add(1)
	if f.CreateErr != nil {
		return brands.Brand{}, f.CreateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.brands == nil {
		f.brands = map[string]brands.Brand{}
	}
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	if b.Version == 0 {
		b.Version = 1
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now().UTC()
	}
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = b.CreatedAt
	}
	f.brands[b.ID] = b
	return b, nil
}

func (f *FakeRepository) Get(_ context.Context, id string) (brands.Brand, error) {
	if f.GetErr != nil {
		return brands.Brand{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.brands[id]
	if !ok {
		return brands.Brand{}, brands.ErrBrandNotFound{ID: id}
	}
	return b, nil
}

func (f *FakeRepository) List(_ context.Context, filter brands.ListFilter) ([]brands.Brand, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]brands.Brand, 0, len(f.brands))
	for _, b := range f.brands {
		out = append(out, b)
	}
	return out, nil
}

func (f *FakeRepository) Update(_ context.Context, b brands.Brand) (brands.Brand, error) {
	f.UpdateCalls.Add(1)
	if f.UpdateErr != nil {
		return brands.Brand{}, f.UpdateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.brands[b.ID]; !ok {
		return brands.Brand{}, brands.ErrBrandNotFound{ID: b.ID}
	}
	b.Version++
	b.UpdatedAt = time.Now().UTC()
	f.brands[b.ID] = b
	return b, nil
}

func (f *FakeRepository) Delete(_ context.Context, id string) error {
	f.DeleteCalls.Add(1)
	if f.DeleteErr != nil {
		return f.DeleteErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.brands[id]; !ok {
		return brands.ErrBrandNotFound{ID: id}
	}
	delete(f.brands, id)
	return nil
}

// Seed inserts b directly, bypassing Create, for arranging test state.
func (f *FakeRepository) Seed(b brands.Brand) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.brands == nil {
		f.brands = map[string]brands.Brand{}
	}
	f.brands[b.ID] = b
}

var _ brands.Repository = (*FakeRepository)(nil)

// FakeVersionRepository satisfies brands.VersionRepository in memory.
type FakeVersionRepository struct {
	mu       sync.Mutex
	versions []brands.BrandVersion

	CreateErr error
	ListErr   error

	CreateCalls atomic.Int64
}

func (f *FakeVersionRepository) CreateVersion(_ context.Context, v brands.BrandVersion) (brands.BrandVersion, error) {
	f.CreateCalls.Add(1)
	if f.CreateErr != nil {
		return brands.BrandVersion{}, f.CreateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now().UTC()
	}
	f.versions = append(f.versions, v)
	return v, nil
}

func (f *FakeVersionRepository) ListVersions(_ context.Context, brandID string) ([]brands.BrandVersion, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []brands.BrandVersion
	for i := len(f.versions) - 1; i >= 0; i-- {
		if f.versions[i].BrandID == brandID {
			out = append(out, f.versions[i])
		}
	}
	return out, nil
}

// Recorded returns a copy of every snapshot created, in arrival order.
func (f *FakeVersionRepository) Recorded() []brands.BrandVersion {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]brands.BrandVersion(nil), f.versions...)
}

var _ brands.VersionRepository = (*FakeVersionRepository)(nil)
