// Package mocks provides in-memory fakes for versions tests.
package mocks

import (
	"context"
	"sort"
	"sync"

	"react-component-library/internal/versions"

	"github.com/google/uuid"
)

// FakeRepository is an in-memory versions repository keyed by id, with
// helpers for tests. Safe for parallel callers; tests rarely need that
// but the mutex is cheap.
type FakeRepository struct {
	mu   sync.Mutex
	rows []versions.Version
}

func NewFakeRepository() *FakeRepository { return &FakeRepository{} }

var _ versions.Repository = (*FakeRepository)(nil)

func (f *FakeRepository) Insert(_ context.Context, v versions.Version) (versions.Version, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	f.rows = append(f.rows, v)
	return v, nil
}

func (f *FakeRepository) Latest(_ context.Context, componentID string) (versions.Version, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var latest *versions.Version
	for i := range f.rows {
		r := f.rows[i]
		if r.ComponentID != componentID {
			continue
		}
		if latest == nil || r.RecordedAt.After(latest.RecordedAt) {
			latest = &f.rows[i]
		}
	}
	if latest == nil {
		return versions.Version{}, nil
	}
	return *latest, nil
}

func (f *FakeRepository) List(_ context.Context, q versions.ListQuery) ([]versions.Version, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if q.Limit <= 0 {
		return nil, nil
	}
	var out []versions.Version
	for _, r := range f.rows {
		if r.ComponentID == q.ComponentID {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].RecordedAt.After(out[j].RecordedAt)
	})
	if len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out, nil
}

func (f *FakeRepository) Get(_ context.Context, componentID, version string) (versions.Version, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var match *versions.Version
	for i := range f.rows {
		r := f.rows[i]
		if r.ComponentID == componentID && r.Version == version {
			if match == nil || r.RecordedAt.After(match.RecordedAt) {
				match = &f.rows[i]
			}
		}
	}
	if match == nil {
		return versions.Version{}, versions.ErrVersionNotFound{ComponentID: componentID, Version: version}
	}
	return *match, nil
}

// Seed inserts a row verbatim, bypassing Insert. Use for setting up
// deterministic state in Diff / List tests.
func (f *FakeRepository) Seed(v versions.Version) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	f.rows = append(f.rows, v)
}
