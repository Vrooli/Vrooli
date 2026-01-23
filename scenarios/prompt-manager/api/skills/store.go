// Package skills provides the core domain types and operations for skill management.
package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store handles file-based skill storage.
// Skills are stored as markdown files organized into folders (core, local, drafts).
// Each folder has a metadata.json tracking skill metadata.
//
// This is a testing seam: inject a mock Store in tests to avoid filesystem access.
type Store struct {
	baseDir string
	mu      sync.RWMutex
	cache   map[string][]Metadata // folder -> skills
}

// NewStore creates a new skill store.
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir: baseDir,
		cache:   make(map[string][]Metadata),
	}
}

// LoadMetadata loads skill metadata from a folder's metadata.json.
func (s *Store) LoadMetadata(folder string) ([]Metadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	metadataPath := filepath.Join(s.baseDir, folder, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Metadata{}, nil
		}
		return nil, err
	}

	var metadata MetadataFile
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	s.cache[folder] = metadata.Skills
	return metadata.Skills, nil
}

// SaveMetadata saves skill metadata to a folder's metadata.json.
func (s *Store) SaveMetadata(folder string, skills []Metadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metadataPath := filepath.Join(s.baseDir, folder, "metadata.json")
	metadata := MetadataFile{Skills: skills}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(metadataPath), 0o755); err != nil {
		return err
	}

	s.cache[folder] = skills
	return os.WriteFile(metadataPath, data, 0o644)
}

// GetContent reads a skill's markdown content from disk.
func (s *Store) GetContent(folder, filename string) (string, error) {
	contentPath := filepath.Join(s.baseDir, folder, filename)
	data, err := os.ReadFile(contentPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveContent writes a skill's markdown content to disk.
func (s *Store) SaveContent(folder, filename, content string) error {
	contentPath := filepath.Join(s.baseDir, folder, filename)
	if err := os.MkdirAll(filepath.Dir(contentPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(contentPath, []byte(content), 0o644)
}

// DeleteContent removes a skill's markdown file from disk.
func (s *Store) DeleteContent(folder, filename string) error {
	contentPath := filepath.Join(s.baseDir, folder, filename)
	return os.Remove(contentPath)
}

// GetAll returns all skills from all folders.
func (s *Store) GetAll() ([]Metadata, error) {
	var allSkills []Metadata

	for _, folder := range Folders {
		skills, err := s.LoadMetadata(folder)
		if err != nil {
			// Log warning but continue - don't fail if one folder is missing
			continue
		}
		for i := range skills {
			// Prefix file path with folder for disambiguation
			skills[i].File = folder + "/" + skills[i].File
		}
		allSkills = append(allSkills, skills...)
	}

	return allSkills, nil
}

// FindByID searches all folders for a skill with the given ID.
// Returns the skill metadata and the folder it was found in.
func (s *Store) FindByID(id string) (*Metadata, string, error) {
	for _, folder := range Folders {
		skills, err := s.LoadMetadata(folder)
		if err != nil {
			continue
		}
		for _, p := range skills {
			if p.ID == id {
				return &p, folder, nil
			}
		}
	}
	return nil, "", fmt.Errorf("skill not found: %s", id)
}
