// Package persistence provides data access for session profile management.
package persistence

import (
	"errors"
	"sort"
	"sync"
)

// MockRepository implements Repository for testing.
// It stores profiles in memory and is safe for concurrent use.
type MockRepository struct {
	mu       sync.RWMutex
	profiles map[ProfileID]*SessionProfile

	// Error injection for testing error paths
	GetErr    error
	ListErr   error
	CreateErr error
	SaveErr   error
	DeleteErr error
}

// NewMockRepository creates a new mock repository for testing.
func NewMockRepository() *MockRepository {
	return &MockRepository{
		profiles: make(map[ProfileID]*SessionProfile),
	}
}

// Get retrieves a profile by ID.
func (r *MockRepository) Get(id ProfileID) (*SessionProfile, error) {
	if r.GetErr != nil {
		return nil, r.GetErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if profile, ok := r.profiles[id]; ok {
		// Return a copy to prevent mutation
		copy := *profile
		return &copy, nil
	}
	return nil, nil
}

// List returns all profiles sorted by last_used_at (desc) then created_at.
func (r *MockRepository) List() ([]SessionProfile, error) {
	if r.ListErr != nil {
		return nil, r.ListErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	profiles := make([]SessionProfile, 0, len(r.profiles))
	for _, p := range r.profiles {
		profiles = append(profiles, *p)
	}
	sort.Slice(profiles, func(i, j int) bool {
		if !profiles[i].LastUsedAt.Equal(profiles[j].LastUsedAt) {
			return profiles[i].LastUsedAt.After(profiles[j].LastUsedAt)
		}
		return profiles[i].CreatedAt.After(profiles[j].CreatedAt)
	})
	return profiles, nil
}

// Create persists a new profile.
func (r *MockRepository) Create(profile *SessionProfile) error {
	if r.CreateErr != nil {
		return r.CreateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if profile == nil {
		return errors.New("profile is nil")
	}
	if profile.ID == "" {
		return errors.New("profile id is required")
	}
	if _, exists := r.profiles[profile.ID]; exists {
		return errors.New("profile already exists")
	}
	// Store a copy to prevent mutation
	copy := *profile
	r.profiles[profile.ID] = &copy
	return nil
}

// Save atomically persists the entire profile.
func (r *MockRepository) Save(profile *SessionProfile) error {
	if r.SaveErr != nil {
		return r.SaveErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if profile == nil {
		return errors.New("profile is nil")
	}
	if profile.ID == "" {
		return errors.New("profile id is required")
	}
	// Store a copy to prevent mutation
	copy := *profile
	r.profiles[profile.ID] = &copy
	return nil
}

// Delete removes a profile by ID.
func (r *MockRepository) Delete(id ProfileID) error {
	if r.DeleteErr != nil {
		return r.DeleteErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.profiles[id]; !exists {
		return errors.New("profile not found")
	}
	delete(r.profiles, id)
	return nil
}

// GetAll returns all profiles (test helper).
func (r *MockRepository) GetAll() map[ProfileID]*SessionProfile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[ProfileID]*SessionProfile, len(r.profiles))
	for k, v := range r.profiles {
		copy := *v
		result[k] = &copy
	}
	return result
}

// Count returns the number of profiles (test helper).
func (r *MockRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.profiles)
}

// Clear removes all profiles (test helper).
func (r *MockRepository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles = make(map[ProfileID]*SessionProfile)
}

// Reset clears all data and errors (test helper).
func (r *MockRepository) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles = make(map[ProfileID]*SessionProfile)
	r.GetErr = nil
	r.ListErr = nil
	r.CreateErr = nil
	r.SaveErr = nil
	r.DeleteErr = nil
}

// Ensure MockRepository implements Repository at compile time.
var _ Repository = (*MockRepository)(nil)
