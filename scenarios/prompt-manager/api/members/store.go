// Package members provides types and operations for member management.
package members

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store handles JSON file-based member storage.
// Members are stored in a single members.json file in the data directory.
type Store struct {
	dataDir string
	mu      sync.RWMutex
	cache   []Member
}

// NewStore creates a new member store.
func NewStore(dataDir string) *Store {
	return &Store{
		dataDir: dataDir,
		cache:   nil,
	}
}

// membersFilePath returns the path to members.json.
func (s *Store) membersFilePath() string {
	return filepath.Join(s.dataDir, "members.json")
}

// Load loads members from members.json.
func (s *Store) Load() ([]Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.membersFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return []Member{}, nil
		}
		return nil, err
	}

	var file MembersFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}

	s.cache = file.Members
	return file.Members, nil
}

// Save saves members to members.json.
func (s *Store) Save(members []Member) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file := MembersFile{Members: members}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return err
	}

	s.cache = members
	return os.WriteFile(s.membersFilePath(), data, 0644)
}

// GetAll returns all members.
func (s *Store) GetAll() ([]Member, error) {
	return s.Load()
}

// FindByID finds a member by ID.
func (s *Store) FindByID(id string) (*Member, error) {
	members, err := s.Load()
	if err != nil {
		return nil, err
	}

	for i := range members {
		if members[i].ID == id {
			return &members[i], nil
		}
	}
	return nil, fmt.Errorf("member not found: %s", id)
}

// Create adds a new member.
func (s *Store) Create(member Member) error {
	members, err := s.Load()
	if err != nil {
		return err
	}

	// Check for duplicate ID
	for _, m := range members {
		if m.ID == member.ID {
			return fmt.Errorf("member with ID %s already exists", member.ID)
		}
	}

	members = append(members, member)
	return s.Save(members)
}

// Update modifies an existing member.
func (s *Store) Update(member Member) error {
	members, err := s.Load()
	if err != nil {
		return err
	}

	found := false
	for i := range members {
		if members[i].ID == member.ID {
			members[i] = member
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("member not found: %s", member.ID)
	}

	return s.Save(members)
}

// Delete removes a member by ID.
func (s *Store) Delete(id string) error {
	members, err := s.Load()
	if err != nil {
		return err
	}

	var filtered []Member
	found := false
	for _, m := range members {
		if m.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, m)
	}

	if !found {
		return fmt.Errorf("member not found: %s", id)
	}

	return s.Save(filtered)
}
