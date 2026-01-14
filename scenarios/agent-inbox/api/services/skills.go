// Package services provides business logic orchestration.
// This file implements file-based skill storage for knowledge modules.
package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"agent-inbox/config"

	"github.com/google/uuid"
)

// Skill represents a knowledge module that provides methodology and expertise.
// Skills are injected into the agent's context to enhance specific tasks.
type Skill struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Icon         string   `json:"icon,omitempty"`
	Modes        []string `json:"modes,omitempty"` // Hierarchical path like ["Architecture", "Audits"]
	Content      string   `json:"content"`
	Tags         []string `json:"tags,omitempty"`
	TargetToolID string   `json:"targetToolId,omitempty"` // Optional tool this skill targets
	Draft        bool     `json:"draft,omitempty"`        // Indicates skill may not be fully working
}

// SkillSource indicates where a skill came from.
type SkillSource string

const (
	SkillSourceDefault  SkillSource = "default"
	SkillSourceUser     SkillSource = "user"
	SkillSourceModified SkillSource = "modified" // User modified a default
)

// SkillResponse is a skill with additional metadata for API responses.
type SkillResponse struct {
	Skill
	Source     SkillSource `json:"source"`
	HasDefault bool        `json:"hasDefault"`
	CreatedAt  string      `json:"createdAt,omitempty"`
	UpdatedAt  string      `json:"updatedAt,omitempty"`
}

// SkillListResponse is the response for listing skills.
type SkillListResponse struct {
	Skills                []SkillResponse `json:"skills"`
	DefaultsCount         int             `json:"defaults_count"`
	UserCount             int             `json:"user_count"`
	ModifiedDefaultsCount int             `json:"modified_defaults_count"`
}

// SkillsService provides CRUD operations for skills stored as files.
type SkillsService struct {
	cfg   *config.SkillsConfig
	mu    sync.RWMutex
	cache map[string]*SkillResponse
}

// NewSkillsService creates a new skill service with the given configuration.
func NewSkillsService(cfg *config.SkillsConfig) *SkillsService {
	return &SkillsService{
		cfg:   cfg,
		cache: make(map[string]*SkillResponse),
	}
}

// defaultsPath returns the full path to the defaults directory.
func (s *SkillsService) defaultsPath() string {
	return filepath.Join(s.cfg.BasePath, s.cfg.DefaultsDir)
}

// userPath returns the full path to the user directory.
func (s *SkillsService) userPath() string {
	return filepath.Join(s.cfg.BasePath, s.cfg.UserDir)
}

// ListSkills returns all skills, merging defaults with user overrides.
func (s *SkillsService) ListSkills() (*SkillListResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Load all default skills
	defaults, err := s.loadSkillsFromDir(s.defaultsPath())
	if err != nil {
		return nil, fmt.Errorf("failed to load default skills: %w", err)
	}

	// Load all user skills
	userSkills, err := s.loadSkillsFromDir(s.userPath())
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load user skills: %w", err)
	}

	// Build a map of user skills by ID for quick lookup
	userByID := make(map[string]*skillWithMeta)
	for _, us := range userSkills {
		userByID[us.skill.ID] = us
	}

	// Build default IDs set
	defaultIDs := make(map[string]bool)
	for _, ds := range defaults {
		defaultIDs[ds.skill.ID] = true
	}

	result := make([]SkillResponse, 0) // Initialize as empty slice, not nil (JSON: [] not null)
	modifiedCount := 0
	userCount := 0

	// Process defaults - check if there's a user override
	for _, ds := range defaults {
		if us, hasOverride := userByID[ds.skill.ID]; hasOverride {
			// User has modified this default
			result = append(result, SkillResponse{
				Skill:      us.skill,
				Source:     SkillSourceModified,
				HasDefault: true,
				CreatedAt:  us.createdAt,
				UpdatedAt:  us.updatedAt,
			})
			modifiedCount++
		} else {
			// Pure default, no override
			result = append(result, SkillResponse{
				Skill:      ds.skill,
				Source:     SkillSourceDefault,
				HasDefault: true,
			})
		}
	}

	// Add user-only skills (those not overriding defaults)
	for _, us := range userSkills {
		if !defaultIDs[us.skill.ID] {
			result = append(result, SkillResponse{
				Skill:      us.skill,
				Source:     SkillSourceUser,
				HasDefault: false,
				CreatedAt:  us.createdAt,
				UpdatedAt:  us.updatedAt,
			})
			userCount++
		}
	}

	return &SkillListResponse{
		Skills:                result,
		DefaultsCount:         len(defaults),
		UserCount:             userCount,
		ModifiedDefaultsCount: modifiedCount,
	}, nil
}

// GetSkill returns a single skill by ID.
func (s *SkillsService) GetSkill(id string) (*SkillResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check user directory first (user skills and overrides take precedence)
	userSkill, userErr := s.findSkillByID(s.userPath(), id)
	defaultSkill, defaultErr := s.findSkillByID(s.defaultsPath(), id)

	if userErr != nil && defaultErr != nil {
		return nil, fmt.Errorf("skill not found: %s", id)
	}

	if userSkill != nil {
		source := SkillSourceUser
		hasDefault := defaultSkill != nil
		if hasDefault {
			source = SkillSourceModified
		}
		return &SkillResponse{
			Skill:      userSkill.skill,
			Source:     source,
			HasDefault: hasDefault,
			CreatedAt:  userSkill.createdAt,
			UpdatedAt:  userSkill.updatedAt,
		}, nil
	}

	return &SkillResponse{
		Skill:      defaultSkill.skill,
		Source:     SkillSourceDefault,
		HasDefault: true,
	}, nil
}

// CreateSkill creates a new user skill.
func (s *SkillsService) CreateSkill(sk *Skill) (*SkillResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate ID if not provided
	if sk.ID == "" {
		sk.ID = fmt.Sprintf("user-%d-%s", time.Now().UnixMilli(), uuid.New().String()[:8])
	}

	// Validate ID doesn't conflict with existing
	existing, _ := s.findSkillByID(s.userPath(), sk.ID)
	if existing != nil {
		return nil, fmt.Errorf("skill with ID %s already exists", sk.ID)
	}
	existingDefault, _ := s.findSkillByID(s.defaultsPath(), sk.ID)
	if existingDefault != nil {
		return nil, fmt.Errorf("skill with ID %s already exists as a default", sk.ID)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Determine the path: user/Custom/{id}.json for new skills
	dir := filepath.Join(s.userPath(), "Custom")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(dir, slugify(sk.ID)+".json")

	if err := s.writeSkill(filePath, sk); err != nil {
		return nil, err
	}

	return &SkillResponse{
		Skill:      *sk,
		Source:     SkillSourceUser,
		HasDefault: false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// UpdateSkill updates an existing skill. If it's a default, creates a user override.
func (s *SkillsService) UpdateSkill(id string, updates *Skill) (*SkillResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find existing skill
	userSkill, _ := s.findSkillByID(s.userPath(), id)
	defaultSkill, _ := s.findSkillByID(s.defaultsPath(), id)

	if userSkill == nil && defaultSkill == nil {
		return nil, fmt.Errorf("skill not found: %s", id)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	hasDefault := defaultSkill != nil

	// Merge updates with existing skill
	var base Skill
	if userSkill != nil {
		base = userSkill.skill
	} else {
		base = defaultSkill.skill
	}

	// Apply updates (preserve ID)
	if updates.Name != "" {
		base.Name = updates.Name
	}
	if updates.Description != "" {
		base.Description = updates.Description
	}
	if updates.Icon != "" {
		base.Icon = updates.Icon
	}
	if updates.Content != "" {
		base.Content = updates.Content
	}
	if updates.Modes != nil {
		base.Modes = updates.Modes
	}
	if updates.Tags != nil {
		base.Tags = updates.Tags
	}
	if updates.TargetToolID != "" {
		base.TargetToolID = updates.TargetToolID
	}
	// Always apply Draft (boolean field)
	base.Draft = updates.Draft

	// Determine path - use existing user path or create new override
	var filePath string
	var createdAt string

	if userSkill != nil {
		filePath = userSkill.path
		createdAt = userSkill.createdAt
	} else {
		// Creating user override of a default
		dir := s.getModePath(s.userPath(), base.Modes)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
		filePath = filepath.Join(dir, slugify(id)+".json")
		createdAt = now
	}

	if err := s.writeSkill(filePath, &base); err != nil {
		return nil, err
	}

	source := SkillSourceUser
	if hasDefault {
		source = SkillSourceModified
	}

	return &SkillResponse{
		Skill:      base,
		Source:     source,
		HasDefault: hasDefault,
		CreatedAt:  createdAt,
		UpdatedAt:  now,
	}, nil
}

// DeleteSkill deletes a user skill or user override.
func (s *SkillsService) DeleteSkill(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userSkill, err := s.findSkillByID(s.userPath(), id)
	if err != nil || userSkill == nil {
		return fmt.Errorf("skill not found in user skills: %s", id)
	}

	if err := os.Remove(userSkill.path); err != nil {
		return fmt.Errorf("failed to delete skill: %w", err)
	}

	return nil
}

// UpdateDefaultSkill updates the actual default skill file (not a user override).
// This is for applying changes directly to the shipped defaults.
func (s *SkillsService) UpdateDefaultSkill(id string, updates *Skill) (*SkillResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find the default skill
	defaultSkill, err := s.findSkillByID(s.defaultsPath(), id)
	if err != nil || defaultSkill == nil {
		return nil, fmt.Errorf("default skill not found: %s", id)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Merge updates with existing skill
	base := defaultSkill.skill

	// Apply updates (preserve ID)
	if updates.Name != "" {
		base.Name = updates.Name
	}
	if updates.Description != "" {
		base.Description = updates.Description
	}
	if updates.Icon != "" {
		base.Icon = updates.Icon
	}
	if updates.Content != "" {
		base.Content = updates.Content
	}
	if updates.Modes != nil {
		base.Modes = updates.Modes
	}
	if updates.Tags != nil {
		base.Tags = updates.Tags
	}
	if updates.TargetToolID != "" {
		base.TargetToolID = updates.TargetToolID
	}
	// Always apply Draft (boolean field)
	base.Draft = updates.Draft

	// Write to the default skill's path
	if err := s.writeSkill(defaultSkill.path, &base); err != nil {
		return nil, err
	}

	// Also delete any user override if it exists (since default now matches what user wanted)
	userSkill, _ := s.findSkillByID(s.userPath(), id)
	if userSkill != nil {
		_ = os.Remove(userSkill.path) // Ignore errors, not critical
	}

	return &SkillResponse{
		Skill:      base,
		Source:     SkillSourceDefault,
		HasDefault: true,
		UpdatedAt:  now,
	}, nil
}

// ResetSkill resets a modified default skill by deleting the user override.
func (s *SkillsService) ResetSkill(id string) (*SkillResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if there's a default to reset to
	defaultSkill, err := s.findSkillByID(s.defaultsPath(), id)
	if err != nil || defaultSkill == nil {
		return nil, fmt.Errorf("no default skill to reset to: %s", id)
	}

	// Check if there's a user override to delete
	userSkill, _ := s.findSkillByID(s.userPath(), id)
	if userSkill != nil {
		if err := os.Remove(userSkill.path); err != nil {
			return nil, fmt.Errorf("failed to delete user override: %w", err)
		}
	}

	return &SkillResponse{
		Skill:      defaultSkill.skill,
		Source:     SkillSourceDefault,
		HasDefault: true,
	}, nil
}

// ImportSkills imports multiple skills from a JSON array.
func (s *SkillsService) ImportSkills(skills []Skill) (int, error) {
	imported := 0
	for _, sk := range skills {
		skCopy := sk
		_, err := s.CreateSkill(&skCopy)
		if err != nil {
			// Skip duplicates, continue with others
			continue
		}
		imported++
	}
	return imported, nil
}

// ExportSkills exports all user skills.
func (s *SkillsService) ExportSkills() ([]Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userSkills, err := s.loadSkillsFromDir(s.userPath())
	if err != nil {
		return nil, err
	}

	result := make([]Skill, 0, len(userSkills))
	for _, us := range userSkills {
		result = append(result, us.skill)
	}
	return result, nil
}

// skillWithMeta is a skill with file metadata.
type skillWithMeta struct {
	skill     Skill
	path      string
	createdAt string
	updatedAt string
}

// loadSkillsFromDir recursively loads all skills from a directory.
func (s *SkillsService) loadSkillsFromDir(dir string) ([]*skillWithMeta, error) {
	var result []*skillWithMeta

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		sk, err := s.readSkill(path)
		if err != nil {
			// Log and skip invalid skills
			return nil
		}

		result = append(result, &skillWithMeta{
			skill:     *sk,
			path:      path,
			createdAt: info.ModTime().UTC().Format(time.RFC3339),
			updatedAt: info.ModTime().UTC().Format(time.RFC3339),
		})

		return nil
	})

	return result, err
}

// findSkillByID finds a skill by ID in a directory.
func (s *SkillsService) findSkillByID(dir, id string) (*skillWithMeta, error) {
	var result *skillWithMeta

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		sk, err := s.readSkill(path)
		if err != nil {
			return nil
		}

		if sk.ID == id {
			result = &skillWithMeta{
				skill:     *sk,
				path:      path,
				createdAt: info.ModTime().UTC().Format(time.RFC3339),
				updatedAt: info.ModTime().UTC().Format(time.RFC3339),
			}
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// readSkill reads a skill from a JSON file.
func (s *SkillsService) readSkill(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var sk Skill
	if err := json.Unmarshal(data, &sk); err != nil {
		return nil, err
	}

	return &sk, nil
}

// writeSkill writes a skill to a JSON file.
func (s *SkillsService) writeSkill(path string, sk *Skill) error {
	data, err := json.MarshalIndent(sk, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal skill: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write skill: %w", err)
	}

	return nil
}

// getModePath constructs a directory path from modes.
func (s *SkillsService) getModePath(basePath string, modes []string) string {
	if len(modes) == 0 {
		return filepath.Join(basePath, "Custom")
	}

	parts := make([]string, len(modes))
	for i, mode := range modes {
		parts[i] = slugify(mode)
	}

	return filepath.Join(basePath, filepath.Join(parts...))
}

// EnsureDirectories creates the skill directories if they don't exist.
func (s *SkillsService) EnsureDirectories() error {
	dirs := []string{
		s.defaultsPath(),
		s.userPath(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
