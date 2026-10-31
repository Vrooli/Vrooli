// Package mocks provides in-memory fakes for the design domain seams so service
// and handler unit tests run without touching the brands domain. Mirrors
// internal/discovery/mocks.
package mocks

import (
	"context"
	"sync"

	"brand-manager/internal/design"
)

// FakeBrandStore satisfies design.BrandStore from an in-memory map keyed by id.
type FakeBrandStore struct {
	mu      sync.Mutex
	brands  map[string]design.Brand
	GetErr  error
	GetCall []string
}

// Seed registers a brand to be returned by Get.
func (f *FakeBrandStore) Seed(b design.Brand) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.brands == nil {
		f.brands = map[string]design.Brand{}
	}
	f.brands[b.ID] = b
}

func (f *FakeBrandStore) Get(_ context.Context, brandID string) (design.Brand, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.GetCall = append(f.GetCall, brandID)
	if f.GetErr != nil {
		return design.Brand{}, f.GetErr
	}
	b, ok := f.brands[brandID]
	if !ok {
		return design.Brand{}, design.ErrBrandNotFound{ID: brandID}
	}
	return b, nil
}

var _ design.BrandStore = (*FakeBrandStore)(nil)

// FakeService satisfies design.Service for handler tests that drive the
// transport edge without the rendering logic. GenerateFunc overrides behaviour;
// a nil field returns zero values.
type FakeService struct {
	GenerateFunc func(ctx context.Context, brandID string) (design.Design, error)
}

func (f FakeService) GenerateDesignLanguage(ctx context.Context, brandID string) (design.Design, error) {
	if f.GenerateFunc != nil {
		return f.GenerateFunc(ctx, brandID)
	}
	return design.Design{}, nil
}

var _ design.Service = (*FakeService)(nil)
