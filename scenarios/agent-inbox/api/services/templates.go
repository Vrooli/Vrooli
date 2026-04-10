// Package services provides business logic orchestration.
// This file implements core CRUD operations for the template system.
package services

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

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
	applyTemplateUpdates(&base, updates)

	// Determine path - use existing user path or create new override
	filePath, createdAt, err := s.getUpdatePath(id, userTemplate, base.Modes, now)
	if err != nil {
		return nil, err
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
