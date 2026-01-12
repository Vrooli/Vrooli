// Package services provides business logic orchestration.
// This file implements file-based template storage for the Suggestions system.
package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"agent-inbox/config"

	"github.com/google/uuid"
)

// TemplateVariable defines a form field for template customization.
type TemplateVariable struct {
	Name         string   `json:"name"`
	Label        string   `json:"label"`
	Type         string   `json:"type"` // "text", "textarea", "select"
	Placeholder  string   `json:"placeholder,omitempty"`
	Options      []string `json:"options,omitempty"` // For select type
	Required     bool     `json:"required,omitempty"`
	DefaultValue string   `json:"defaultValue,omitempty"`
}

// Template represents a suggestion template for the message composer.
type Template struct {
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	Description       string             `json:"description"`
	Icon              string             `json:"icon,omitempty"`
	Modes             []string           `json:"modes,omitempty"` // Hierarchical path like ["Research", "Codebase Structure"]
	Content           string             `json:"content"`
	Variables         []TemplateVariable `json:"variables"`
	SuggestedSkillIDs []string           `json:"suggestedSkillIds,omitempty"`
	SuggestedToolIDs  []string           `json:"suggestedToolIds,omitempty"`
}

// TemplateSource indicates where a template came from.
type TemplateSource string

const (
	SourceDefault  TemplateSource = "default"
	SourceUser     TemplateSource = "user"
	SourceModified TemplateSource = "modified" // User modified a default
)

// TemplateResponse is a template with additional metadata for API responses.
type TemplateResponse struct {
	Template
	Source     TemplateSource `json:"source"`
	HasDefault bool           `json:"hasDefault"`
	CreatedAt  string         `json:"createdAt,omitempty"`
	UpdatedAt  string         `json:"updatedAt,omitempty"`
}

// TemplateListResponse is the response for listing templates.
type TemplateListResponse struct {
	Templates             []TemplateResponse `json:"templates"`
	DefaultsCount         int                `json:"defaults_count"`
	UserCount             int                `json:"user_count"`
	ModifiedDefaultsCount int                `json:"modified_defaults_count"`
}

// TemplatesService provides CRUD operations for templates stored as files.
type TemplatesService struct {
	cfg   *config.TemplatesConfig
	mu    sync.RWMutex
	cache map[string]*TemplateResponse
}

// NewTemplatesService creates a new template service with the given configuration.
func NewTemplatesService(cfg *config.TemplatesConfig) *TemplatesService {
	return &TemplatesService{
		cfg:   cfg,
		cache: make(map[string]*TemplateResponse),
	}
}

// defaultsPath returns the full path to the defaults directory.
func (s *TemplatesService) defaultsPath() string {
	return filepath.Join(s.cfg.BasePath, s.cfg.DefaultsDir)
}

// userPath returns the full path to the user directory.
func (s *TemplatesService) userPath() string {
	return filepath.Join(s.cfg.BasePath, s.cfg.UserDir)
}

// ListTemplates returns all templates, merging defaults with user overrides.
func (s *TemplatesService) ListTemplates() (*TemplateListResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Load all default templates
	defaults, err := s.loadTemplatesFromDir(s.defaultsPath())
	if err != nil {
		return nil, fmt.Errorf("failed to load default templates: %w", err)
	}

	// Load all user templates
	userTemplates, err := s.loadTemplatesFromDir(s.userPath())
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load user templates: %w", err)
	}

	// Build a map of user templates by ID for quick lookup
	userByID := make(map[string]*templateWithMeta)
	for _, ut := range userTemplates {
		userByID[ut.template.ID] = ut
	}

	// Build default IDs set
	defaultIDs := make(map[string]bool)
	for _, dt := range defaults {
		defaultIDs[dt.template.ID] = true
	}

	result := make([]TemplateResponse, 0) // Initialize as empty slice, not nil (JSON: [] not null)
	modifiedCount := 0
	userCount := 0

	// Process defaults - check if there's a user override
	for _, dt := range defaults {
		if ut, hasOverride := userByID[dt.template.ID]; hasOverride {
			// User has modified this default
			result = append(result, TemplateResponse{
				Template:   ut.template,
				Source:     SourceModified,
				HasDefault: true,
				CreatedAt:  ut.createdAt,
				UpdatedAt:  ut.updatedAt,
			})
			modifiedCount++
		} else {
			// Pure default, no override
			result = append(result, TemplateResponse{
				Template:   dt.template,
				Source:     SourceDefault,
				HasDefault: true,
			})
		}
	}

	// Add user-only templates (those not overriding defaults)
	for _, ut := range userTemplates {
		if !defaultIDs[ut.template.ID] {
			result = append(result, TemplateResponse{
				Template:   ut.template,
				Source:     SourceUser,
				HasDefault: false,
				CreatedAt:  ut.createdAt,
				UpdatedAt:  ut.updatedAt,
			})
			userCount++
		}
	}

	return &TemplateListResponse{
		Templates:             result,
		DefaultsCount:         len(defaults),
		UserCount:             userCount,
		ModifiedDefaultsCount: modifiedCount,
	}, nil
}

// GetTemplate returns a single template by ID.
func (s *TemplatesService) GetTemplate(id string) (*TemplateResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check user directory first (user templates and overrides take precedence)
	userTemplate, userErr := s.findTemplateByID(s.userPath(), id)
	defaultTemplate, defaultErr := s.findTemplateByID(s.defaultsPath(), id)

	if userErr != nil && defaultErr != nil {
		return nil, fmt.Errorf("template not found: %s", id)
	}

	if userTemplate != nil {
		source := SourceUser
		hasDefault := defaultTemplate != nil
		if hasDefault {
			source = SourceModified
		}
		return &TemplateResponse{
			Template:   userTemplate.template,
			Source:     source,
			HasDefault: hasDefault,
			CreatedAt:  userTemplate.createdAt,
			UpdatedAt:  userTemplate.updatedAt,
		}, nil
	}

	return &TemplateResponse{
		Template:   defaultTemplate.template,
		Source:     SourceDefault,
		HasDefault: true,
	}, nil
}

// CreateTemplate creates a new user template.
func (s *TemplatesService) CreateTemplate(t *Template) (*TemplateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate ID if not provided
	if t.ID == "" {
		t.ID = fmt.Sprintf("user-%d-%s", time.Now().UnixMilli(), uuid.New().String()[:8])
	}

	// Validate ID doesn't conflict with existing
	existing, _ := s.findTemplateByID(s.userPath(), t.ID)
	if existing != nil {
		return nil, fmt.Errorf("template with ID %s already exists", t.ID)
	}
	existingDefault, _ := s.findTemplateByID(s.defaultsPath(), t.ID)
	if existingDefault != nil {
		return nil, fmt.Errorf("template with ID %s already exists as a default", t.ID)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Determine the path: user/Custom/{id}.json for new templates
	dir := filepath.Join(s.userPath(), "Custom")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(dir, slugify(t.ID)+".json")

	if err := s.writeTemplate(filePath, t); err != nil {
		return nil, err
	}

	return &TemplateResponse{
		Template:   *t,
		Source:     SourceUser,
		HasDefault: false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// UpdateTemplate updates an existing template. If it's a default, creates a user override.
func (s *TemplatesService) UpdateTemplate(id string, updates *Template) (*TemplateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find existing template
	userTemplate, _ := s.findTemplateByID(s.userPath(), id)
	defaultTemplate, _ := s.findTemplateByID(s.defaultsPath(), id)

	if userTemplate == nil && defaultTemplate == nil {
		return nil, fmt.Errorf("template not found: %s", id)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	hasDefault := defaultTemplate != nil

	// Merge updates with existing template
	var base Template
	if userTemplate != nil {
		base = userTemplate.template
	} else {
		base = defaultTemplate.template
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
	if updates.Variables != nil {
		base.Variables = updates.Variables
	}
	if updates.SuggestedSkillIDs != nil {
		base.SuggestedSkillIDs = updates.SuggestedSkillIDs
	}
	if updates.SuggestedToolIDs != nil {
		base.SuggestedToolIDs = updates.SuggestedToolIDs
	}

	// Determine path - use existing user path or create new override
	var filePath string
	var createdAt string

	if userTemplate != nil {
		filePath = userTemplate.path
		createdAt = userTemplate.createdAt
	} else {
		// Creating user override of a default
		dir := s.getModePath(s.userPath(), base.Modes)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
		filePath = filepath.Join(dir, slugify(id)+".json")
		createdAt = now
	}

	if err := s.writeTemplate(filePath, &base); err != nil {
		return nil, err
	}

	source := SourceUser
	if hasDefault {
		source = SourceModified
	}

	return &TemplateResponse{
		Template:   base,
		Source:     source,
		HasDefault: hasDefault,
		CreatedAt:  createdAt,
		UpdatedAt:  now,
	}, nil
}

// DeleteTemplate deletes a user template or user override.
func (s *TemplatesService) DeleteTemplate(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userTemplate, err := s.findTemplateByID(s.userPath(), id)
	if err != nil || userTemplate == nil {
		return fmt.Errorf("template not found in user templates: %s", id)
	}

	if err := os.Remove(userTemplate.path); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

// ResetTemplate resets a modified default template by deleting the user override.
func (s *TemplatesService) ResetTemplate(id string) (*TemplateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if there's a default to reset to
	defaultTemplate, err := s.findTemplateByID(s.defaultsPath(), id)
	if err != nil || defaultTemplate == nil {
		return nil, fmt.Errorf("no default template to reset to: %s", id)
	}

	// Check if there's a user override to delete
	userTemplate, _ := s.findTemplateByID(s.userPath(), id)
	if userTemplate != nil {
		if err := os.Remove(userTemplate.path); err != nil {
			return nil, fmt.Errorf("failed to delete user override: %w", err)
		}
	}

	return &TemplateResponse{
		Template:   defaultTemplate.template,
		Source:     SourceDefault,
		HasDefault: true,
	}, nil
}

// ImportTemplates imports multiple templates from a JSON array.
func (s *TemplatesService) ImportTemplates(templates []Template) (int, error) {
	imported := 0
	for _, t := range templates {
		tCopy := t
		_, err := s.CreateTemplate(&tCopy)
		if err != nil {
			// Skip duplicates, continue with others
			continue
		}
		imported++
	}
	return imported, nil
}

// ExportTemplates exports all user templates.
func (s *TemplatesService) ExportTemplates() ([]Template, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userTemplates, err := s.loadTemplatesFromDir(s.userPath())
	if err != nil {
		return nil, err
	}

	result := make([]Template, 0, len(userTemplates))
	for _, ut := range userTemplates {
		result = append(result, ut.template)
	}
	return result, nil
}

// templateWithMeta is a template with file metadata.
type templateWithMeta struct {
	template  Template
	path      string
	createdAt string
	updatedAt string
}

// loadTemplatesFromDir recursively loads all templates from a directory.
func (s *TemplatesService) loadTemplatesFromDir(dir string) ([]*templateWithMeta, error) {
	var result []*templateWithMeta

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

		t, err := s.readTemplate(path)
		if err != nil {
			// Log and skip invalid templates
			return nil
		}

		result = append(result, &templateWithMeta{
			template:  *t,
			path:      path,
			createdAt: info.ModTime().UTC().Format(time.RFC3339),
			updatedAt: info.ModTime().UTC().Format(time.RFC3339),
		})

		return nil
	})

	return result, err
}

// findTemplateByID finds a template by ID in a directory.
func (s *TemplatesService) findTemplateByID(dir, id string) (*templateWithMeta, error) {
	var result *templateWithMeta

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

		t, err := s.readTemplate(path)
		if err != nil {
			return nil
		}

		if t.ID == id {
			result = &templateWithMeta{
				template:  *t,
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

// readTemplate reads a template from a JSON file.
func (s *TemplatesService) readTemplate(path string) (*Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var t Template
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}

	return &t, nil
}

// writeTemplate writes a template to a JSON file.
func (s *TemplatesService) writeTemplate(path string, t *Template) error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	return nil
}

// getModePath constructs a directory path from modes.
func (s *TemplatesService) getModePath(basePath string, modes []string) string {
	if len(modes) == 0 {
		return filepath.Join(basePath, "Custom")
	}

	parts := make([]string, len(modes))
	for i, mode := range modes {
		parts[i] = slugify(mode)
	}

	return filepath.Join(basePath, filepath.Join(parts...))
}

// slugify converts a string to a filesystem-safe slug.
func slugify(s string) string {
	// Replace spaces and special chars with hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9-]`)
	result := reg.ReplaceAllString(s, "-")
	// Remove consecutive hyphens
	result = regexp.MustCompile(`-+`).ReplaceAllString(result, "-")
	// Trim leading/trailing hyphens
	result = strings.Trim(result, "-")
	// Lowercase
	return strings.ToLower(result)
}

// EnsureDirectories creates the template directories if they don't exist.
func (s *TemplatesService) EnsureDirectories() error {
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
