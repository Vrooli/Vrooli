// Package mocks holds components-domain test fakes co-located with
// the domain they double for. Deleting the domain folder takes its
// mocks with it; package graph reflects ownership (mocks imports
// components; components does not import mocks).
package mocks

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"react-component-library/internal/components"
)

// FakeRepository satisfies components.Repository for service and
// indexer tests that don't want the sqlite round-trip. In-memory map
// keyed by ID with a parallel libraryID → id index.
//
// Per-method error knobs (UpsertErr, GetErr, …) let tests drive the
// failure paths without faking sqlite. Atomic call counters keep
// -race quiet under fan-out.
type FakeRepository struct {
	mu          sync.Mutex
	items       map[string]components.Component // by ID
	libToID     map[string]string                // library_id → id
	UpsertErr   error
	GetErr      error
	ListErr     error
	DeleteErr   error
	UpsertCalls atomic.Int64
	GetCalls    atomic.Int64
	ListCalls   atomic.Int64
	DeleteCalls atomic.Int64
	NowFn       func() time.Time
}

func NewFakeRepository() *FakeRepository {
	return &FakeRepository{
		items:   map[string]components.Component{},
		libToID: map[string]string{},
		NowFn:   func() time.Time { return time.Now().UTC() },
	}
}

var _ components.Repository = (*FakeRepository)(nil)

func (f *FakeRepository) Upsert(ctx context.Context, in components.UpsertInput) (components.Component, error) {
	f.UpsertCalls.Add(1)
	if f.UpsertErr != nil {
		return components.Component{}, f.UpsertErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.NowFn()
	id, existed := f.libToID[in.LibraryID]
	indexedAt := now
	if existed {
		indexedAt = f.items[id].IndexedAt
	} else {
		id = uuid.NewString()
		f.libToID[in.LibraryID] = id
	}
	c := components.Component{
		ID:          id,
		LibraryID:   in.LibraryID,
		DisplayName: in.DisplayName,
		Description: in.Description,
		SourcePath:  in.SourcePath,
		Version:     in.Version,
		Tags:        append([]string(nil), in.Tags...),
		IndexedAt:   indexedAt,
		UpdatedAt:   now,
		Headers:     copyHeaders(in.Headers),
	}
	f.items[id] = c
	return c, nil
}

func (f *FakeRepository) Get(ctx context.Context, id string) (components.Component, error) {
	f.GetCalls.Add(1)
	if f.GetErr != nil {
		return components.Component{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.items[id]
	if !ok {
		return components.Component{}, components.ErrComponentNotFound{IDOrLibraryID: id}
	}
	return c, nil
}

func (f *FakeRepository) GetByLibraryID(ctx context.Context, libraryID string) (components.Component, error) {
	f.GetCalls.Add(1)
	if f.GetErr != nil {
		return components.Component{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.libToID[libraryID]
	if !ok {
		return components.Component{}, components.ErrComponentNotFound{IDOrLibraryID: libraryID}
	}
	return f.items[id], nil
}

func (f *FakeRepository) List(ctx context.Context, q components.SearchQuery) ([]components.Component, error) {
	f.ListCalls.Add(1)
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	limit := q.Limit
	if limit <= 0 {
		return nil, nil
	}
	matchL := strings.ToLower(strings.TrimSpace(q.Match))
	tagL := strings.ToLower(strings.TrimSpace(q.Tag))
	categoryL := strings.ToLower(strings.TrimSpace(q.Category))
	multiTags := make([]string, 0, len(q.Tags))
	for _, t := range q.Tags {
		trimmed := strings.ToLower(strings.TrimSpace(t))
		if trimmed == "" {
			continue
		}
		multiTags = append(multiTags, trimmed)
	}
	var out []components.Component
	for _, c := range f.items {
		if matchL != "" {
			hay := strings.ToLower(c.LibraryID + " " + c.DisplayName + " " + c.Description + " " + c.SourcePath)
			if !strings.Contains(hay, matchL) {
				continue
			}
		}
		if tagL != "" {
			hit := false
			for _, t := range c.Tags {
				if strings.EqualFold(t, tagL) {
					hit = true
					break
				}
			}
			if !hit {
				continue
			}
		}
		if len(multiTags) > 0 {
			hit := false
			for _, want := range multiTags {
				for _, t := range c.Tags {
					if strings.EqualFold(t, want) {
						hit = true
						break
					}
				}
				if hit {
					break
				}
			}
			if !hit {
				continue
			}
		}
		if categoryL != "" {
			v, ok := c.Headers["category"]
			if !ok || !strings.EqualFold(v, categoryL) {
				continue
			}
		}
		out = append(out, c)
	}
	matchSet := matchL != ""
	// Mirrors sqlite ORDER BY: match-mode → display_name COLLATE NOCASE;
	// otherwise newest indexed first, then library_id asc.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			less := false
			if matchSet {
				li, lj := strings.ToLower(out[i].DisplayName), strings.ToLower(out[j].DisplayName)
				if lj < li || (lj == li && out[j].LibraryID < out[i].LibraryID) {
					less = true
				}
			} else if out[j].IndexedAt.After(out[i].IndexedAt) ||
				(out[j].IndexedAt.Equal(out[i].IndexedAt) && out[j].LibraryID < out[i].LibraryID) {
				less = true
			}
			if less {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *FakeRepository) DeleteMissing(ctx context.Context, keep []string) (int, error) {
	f.DeleteCalls.Add(1)
	if f.DeleteErr != nil {
		return 0, f.DeleteErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	keepSet := map[string]struct{}{}
	for _, k := range keep {
		keepSet[k] = struct{}{}
	}
	deleted := 0
	for lib, id := range f.libToID {
		if _, ok := keepSet[lib]; !ok {
			delete(f.items, id)
			delete(f.libToID, lib)
			deleted++
		}
	}
	return deleted, nil
}

func copyHeaders(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
