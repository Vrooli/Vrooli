package mocks

import (
	"errors"
	"sync"

	"workspace-sandbox/internal/config"
)

// FakeProfileStore is a config.ProfileStore implementation backed by
// an in-memory map. Tests use it to drive ProfileResolver and other
// callers that consult the profile registry.
type FakeProfileStore struct {
	mu sync.Mutex

	// Profiles maps profile.ID to the profile. Tests seed this map
	// directly during arrange.
	Profiles map[string]config.IsolationProfile

	// Per-method error injection.
	ListErr   error
	GetErr    error
	SaveErr   error
	DeleteErr error
}

// NewFakeProfileStore returns a FakeProfileStore initialized with the
// given profiles (keyed by ID). Pass nil for an empty store.
func NewFakeProfileStore(profiles ...config.IsolationProfile) *FakeProfileStore {
	m := make(map[string]config.IsolationProfile, len(profiles))
	for _, p := range profiles {
		m[p.ID] = p
	}
	return &FakeProfileStore{Profiles: m}
}

func (f *FakeProfileStore) List() ([]config.IsolationProfile, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]config.IsolationProfile, 0, len(f.Profiles))
	for _, p := range f.Profiles {
		out = append(out, p)
	}
	return out, nil
}

func (f *FakeProfileStore) Get(id string) (*config.IsolationProfile, error) {
	if f.GetErr != nil {
		return nil, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if p, ok := f.Profiles[id]; ok {
		copy := p
		return &copy, nil
	}
	return nil, errors.New("not found")
}

func (f *FakeProfileStore) Save(profile config.IsolationProfile) error {
	if f.SaveErr != nil {
		return f.SaveErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Profiles == nil {
		f.Profiles = make(map[string]config.IsolationProfile)
	}
	f.Profiles[profile.ID] = profile
	return nil
}

func (f *FakeProfileStore) Delete(id string) error {
	if f.DeleteErr != nil {
		return f.DeleteErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.Profiles, id)
	return nil
}

var _ config.ProfileStore = (*FakeProfileStore)(nil)
