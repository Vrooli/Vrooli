// Package mocks provides in-memory fakes for the manifest seams.
// Production code never imports this package — the no_prod_import_test
// drift gate enforces that.
package mocks

import (
	"context"
	"sort"
	"sync"
	"time"

	manifest "development-toolchain-validator/internal/manifest"
)

type key struct {
	SkillID    string
	GoldenSlug string
}

// FakeRepository is an in-memory manifest.Repository for service-level
// tests. Mirrors the SQLite implementation's contract: Upsert returns
// the stored row, Get returns ErrManifestNotFound when absent, List
// returns ordered, ClearStaleOverride / GetStaleOverride manage the
// overrides table.
type FakeRepository struct {
	mu        sync.Mutex
	rows      map[key]manifest.Manifest
	overrides map[key]time.Time
}

var _ manifest.Repository = (*FakeRepository)(nil)

// NewFakeRepository constructs an empty FakeRepository.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{
		rows:      map[key]manifest.Manifest{},
		overrides: map[key]time.Time{},
	}
}

func (f *FakeRepository) Upsert(_ context.Context, m manifest.Manifest) (manifest.Manifest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[key{m.SkillID, m.GoldenSlug}] = m
	return m, nil
}

func (f *FakeRepository) Get(_ context.Context, skillID, goldenSlug string) (manifest.Manifest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.rows[key{skillID, goldenSlug}]
	if !ok {
		return manifest.Manifest{}, manifest.ErrManifestNotFound{SkillID: skillID, GoldenSlug: goldenSlug}
	}
	return m, nil
}

func (f *FakeRepository) List(_ context.Context) ([]manifest.Manifest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]manifest.Manifest, 0, len(f.rows))
	for _, m := range f.rows {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SkillID != out[j].SkillID {
			return out[i].SkillID < out[j].SkillID
		}
		return out[i].GoldenSlug < out[j].GoldenSlug
	})
	return out, nil
}

func (f *FakeRepository) ClearStaleOverride(_ context.Context, skillID, goldenSlug string, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.overrides[key{skillID, goldenSlug}] = at
	return nil
}

func (f *FakeRepository) GetStaleOverride(_ context.Context, skillID, goldenSlug string) (time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.overrides[key{skillID, goldenSlug}], nil
}
