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

// versionsFilePath returns the path to versions.json for a folder.
func (s *Store) versionsFilePath(folder string) string {
	return filepath.Join(s.baseDir, folder, "versions.json")
}

// LoadVersions loads all version files for a folder.
func (s *Store) LoadVersions(folder string) (map[string]*VersionFile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.versionsFilePath(folder))
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*VersionFile), nil
		}
		return nil, err
	}

	var versionFiles map[string]*VersionFile
	if err := json.Unmarshal(data, &versionFiles); err != nil {
		return nil, err
	}

	return versionFiles, nil
}

// SaveVersions saves version files for a folder.
func (s *Store) SaveVersions(folder string, versions map[string]*VersionFile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(versions, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.versionsFilePath(folder), data, 0o644)
}

// GetVersions returns version history for a skill.
func (s *Store) GetVersions(skillID string) ([]SkillVersion, error) {
	skill, folder, err := s.FindByID(skillID)
	if err != nil {
		return nil, err
	}

	versions, err := s.LoadVersions(folder)
	if err != nil {
		return nil, err
	}

	vf, ok := versions[skillID]
	if !ok || len(vf.Versions) == 0 {
		// No version history yet - return current as v1
		content, err := s.GetContent(folder, skill.File)
		if err != nil {
			return nil, err
		}
		return []SkillVersion{{
			Version:   1,
			Content:   content,
			Name:      skill.Name,
			UpdatedAt: skill.UpdatedAt,
		}}, nil
	}

	return vf.Versions, nil
}

// SaveVersion saves the current state as a new version.
func (s *Store) SaveVersion(skillID, folder string, skill *Metadata, content string) error {
	versions, err := s.LoadVersions(folder)
	if err != nil {
		return err
	}

	vf, ok := versions[skillID]
	if !ok {
		vf = &VersionFile{
			SkillID:  skillID,
			Versions: []SkillVersion{},
		}
		versions[skillID] = vf
	}

	// Determine next version number
	nextVersion := 1
	if len(vf.Versions) > 0 {
		nextVersion = vf.Versions[len(vf.Versions)-1].Version + 1
	}

	vf.Versions = append(vf.Versions, SkillVersion{
		Version:   nextVersion,
		Content:   content,
		Name:      skill.Name,
		UpdatedAt: skill.UpdatedAt,
	})

	return s.SaveVersions(folder, versions)
}

// GetVersionContent returns the content of a specific version.
func (s *Store) GetVersionContent(skillID string, version int) (*SkillVersion, error) {
	_, folder, err := s.FindByID(skillID)
	if err != nil {
		return nil, err
	}

	versions, err := s.LoadVersions(folder)
	if err != nil {
		return nil, err
	}

	vf, ok := versions[skillID]
	if !ok {
		return nil, fmt.Errorf("no version history for skill: %s", skillID)
	}

	for _, v := range vf.Versions {
		if v.Version == version {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("version %d not found for skill: %s", version, skillID)
}
