// Package avatars provides types and operations for avatar management.
package avatars

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store handles JSON file-based avatar storage.
// Avatars are stored in a single avatars.json file in the data directory.
type Store struct {
	dataDir string
	mu      sync.RWMutex
	cache   []Avatar
}

// NewStore creates a new avatar store.
func NewStore(dataDir string) *Store {
	return &Store{
		dataDir: dataDir,
		cache:   nil,
	}
}

// avatarsFilePath returns the path to avatars.json.
func (s *Store) avatarsFilePath() string {
	return filepath.Join(s.dataDir, "avatars.json")
}

// Load loads avatars from avatars.json.
func (s *Store) Load() ([]Avatar, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.avatarsFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return []Avatar{}, nil
		}
		return nil, err
	}

	var file AvatarsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}

	s.cache = file.Avatars
	return file.Avatars, nil
}

// Save saves avatars to avatars.json.
func (s *Store) Save(avatars []Avatar) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file := AvatarsFile{Avatars: avatars}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return err
	}

	s.cache = avatars
	return os.WriteFile(s.avatarsFilePath(), data, 0644)
}

// GetAll returns all avatars.
func (s *Store) GetAll() ([]Avatar, error) {
	return s.Load()
}

// FindByID finds an avatar by ID.
func (s *Store) FindByID(id string) (*Avatar, error) {
	avatars, err := s.Load()
	if err != nil {
		return nil, err
	}

	for i := range avatars {
		if avatars[i].ID == id {
			return &avatars[i], nil
		}
	}
	return nil, fmt.Errorf("avatar not found: %s", id)
}

// Create adds a new avatar.
func (s *Store) Create(avatar Avatar) error {
	avatars, err := s.Load()
	if err != nil {
		return err
	}

	// Check for duplicate ID
	for _, a := range avatars {
		if a.ID == avatar.ID {
			return fmt.Errorf("avatar with ID %s already exists", avatar.ID)
		}
	}

	avatars = append(avatars, avatar)
	return s.Save(avatars)
}

// Update modifies an existing avatar.
func (s *Store) Update(avatar Avatar) error {
	avatars, err := s.Load()
	if err != nil {
		return err
	}

	found := false
	for i := range avatars {
		if avatars[i].ID == avatar.ID {
			avatars[i] = avatar
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("avatar not found: %s", avatar.ID)
	}

	return s.Save(avatars)
}

// Delete removes an avatar by ID.
func (s *Store) Delete(id string) error {
	avatars, err := s.Load()
	if err != nil {
		return err
	}

	var filtered []Avatar
	found := false
	for _, a := range avatars {
		if a.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, a)
	}

	if !found {
		return fmt.Errorf("avatar not found: %s", id)
	}

	return s.Save(filtered)
}
