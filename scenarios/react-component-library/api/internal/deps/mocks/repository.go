// Package mocks holds deps-domain test fakes co-located with the
// domain they double for. mocks imports deps; deps does not import
// mocks.
package mocks

import (
	"context"
	"sort"
	"sync"

	"react-component-library/internal/deps"
)

// FakeRepository satisfies deps.Repository for service and handler
// tests. In-memory map keyed by (component_id, dep_name).
type FakeRepository struct {
	mu        sync.Mutex
	byComp    map[string][]deps.Declaration
	SyncErr   error
	ListErr   error
	DeleteErr error
}

func NewFakeRepository() *FakeRepository {
	return &FakeRepository{byComp: map[string][]deps.Declaration{}}
}

var _ deps.Repository = (*FakeRepository)(nil)

func (f *FakeRepository) SyncForComponent(_ context.Context, in deps.SyncInput) error {
	if f.SyncErr != nil {
		return f.SyncErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	rows := make([]deps.Declaration, 0, len(in.Declarations))
	for _, d := range in.Declarations {
		if d.DepName == "" {
			continue
		}
		rows = append(rows, deps.Declaration{
			ComponentID:  in.ComponentID,
			LibraryID:    in.LibraryID,
			DepName:      d.DepName,
			VersionRange: d.VersionRange,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].DepName < rows[j].DepName })
	if len(rows) == 0 {
		delete(f.byComp, in.ComponentID)
	} else {
		f.byComp[in.ComponentID] = rows
	}
	return nil
}

func (f *FakeRepository) ListForComponent(_ context.Context, componentID string) ([]deps.Declaration, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	src := f.byComp[componentID]
	if src == nil {
		return nil, nil
	}
	out := make([]deps.Declaration, len(src))
	copy(out, src)
	return out, nil
}

func (f *FakeRepository) DeleteForComponent(_ context.Context, componentID string) error {
	if f.DeleteErr != nil {
		return f.DeleteErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.byComp, componentID)
	return nil
}
