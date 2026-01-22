// Package prompts provides the core domain types and operations for prompt management.
package prompts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store handles file-based prompt storage.
// Prompts are stored as markdown files organized into folders (core, local, drafts).
// Each folder has a metadata.json tracking prompt metadata.
//
// This is a testing seam: inject a mock Store in tests to avoid filesystem access.
type Store struct {
	baseDir string
	mu      sync.RWMutex
	cache   map[string][]Metadata // folder -> prompts
}

// NewStore creates a new prompt store.
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir: baseDir,
		cache:   make(map[string][]Metadata),
	}
}

// LoadMetadata loads prompt metadata from a folder's metadata.json.
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

	s.cache[folder] = metadata.Prompts
	return metadata.Prompts, nil
}

// SaveMetadata saves prompt metadata to a folder's metadata.json.
func (s *Store) SaveMetadata(folder string, prompts []Metadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metadataPath := filepath.Join(s.baseDir, folder, "metadata.json")
	metadata := MetadataFile{Prompts: prompts}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(metadataPath), 0755); err != nil {
		return err
	}

	s.cache[folder] = prompts
	return os.WriteFile(metadataPath, data, 0644)
}

// GetContent reads a prompt's markdown content from disk.
func (s *Store) GetContent(folder, filename string) (string, error) {
	contentPath := filepath.Join(s.baseDir, folder, filename)
	data, err := os.ReadFile(contentPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveContent writes a prompt's markdown content to disk.
func (s *Store) SaveContent(folder, filename, content string) error {
	contentPath := filepath.Join(s.baseDir, folder, filename)
	if err := os.MkdirAll(filepath.Dir(contentPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(contentPath, []byte(content), 0644)
}

// DeleteContent removes a prompt's markdown file from disk.
func (s *Store) DeleteContent(folder, filename string) error {
	contentPath := filepath.Join(s.baseDir, folder, filename)
	return os.Remove(contentPath)
}

// GetAll returns all prompts from all folders.
func (s *Store) GetAll() ([]Metadata, error) {
	var allPrompts []Metadata

	for _, folder := range Folders {
		prompts, err := s.LoadMetadata(folder)
		if err != nil {
			// Log warning but continue - don't fail if one folder is missing
			continue
		}
		for i := range prompts {
			// Prefix file path with folder for disambiguation
			prompts[i].File = folder + "/" + prompts[i].File
		}
		allPrompts = append(allPrompts, prompts...)
	}

	return allPrompts, nil
}

// FindByID searches all folders for a prompt with the given ID.
// Returns the prompt metadata and the folder it was found in.
func (s *Store) FindByID(id string) (*Metadata, string, error) {
	for _, folder := range Folders {
		prompts, err := s.LoadMetadata(folder)
		if err != nil {
			continue
		}
		for _, p := range prompts {
			if p.ID == id {
				return &p, folder, nil
			}
		}
	}
	return nil, "", fmt.Errorf("prompt not found: %s", id)
}
