package mocks

import (
	"context"
	"sort"
	"sync"

	skillcatalog "development-toolchain-validator/internal/skill_catalog"
)

// FakeRepository is an in-memory Repository for service-level tests.
// Mirrors the SQLite implementation's contract: Upsert reports insert
// vs change, List returns ordered, DeleteMissing removes rows not in
// the keep set.
type FakeRepository struct {
	mu     sync.Mutex
	skills map[string]skillcatalog.Skill
}

var _ skillcatalog.Repository = (*FakeRepository)(nil)

// NewFakeRepository constructs an empty FakeRepository.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{skills: map[string]skillcatalog.Skill{}}
}

func (f *FakeRepository) Upsert(_ context.Context, s skillcatalog.Skill) (bool, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	existing, ok := f.skills[s.ID]
	f.skills[s.ID] = s
	if !ok {
		return true, true, nil
	}
	changed := existing.Version != s.Version || existing.ContentHash != s.ContentHash
	return false, changed, nil
}

func (f *FakeRepository) Get(_ context.Context, id string) (skillcatalog.Skill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.skills[id]
	if !ok {
		return skillcatalog.Skill{}, skillcatalog.ErrSkillNotFound{ID: id}
	}
	return s, nil
}

func (f *FakeRepository) List(_ context.Context) ([]skillcatalog.Skill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]skillcatalog.Skill, 0, len(f.skills))
	for _, s := range f.skills {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *FakeRepository) DeleteMissing(_ context.Context, keep []string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	keepSet := make(map[string]struct{}, len(keep))
	for _, id := range keep {
		keepSet[id] = struct{}{}
	}
	removed := 0
	for id := range f.skills {
		if _, ok := keepSet[id]; !ok {
			delete(f.skills, id)
			removed++
		}
	}
	return removed, nil
}
