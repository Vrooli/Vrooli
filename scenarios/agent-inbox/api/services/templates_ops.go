// Package services provides business logic orchestration.
// This file implements secondary template operations: update defaults, reset, import, export.
package services

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// UpdateDefaultTemplate updates the actual default template file (not a user override).
// This is for applying changes directly to the shipped defaults.
func (s *TemplatesService) UpdateDefaultTemplate(id string, updates *Template) (*TemplateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find the default template
	defaultTemplate, err := s.findTemplateByID(s.defaultsPath(), id)
	if err != nil || defaultTemplate == nil {
		return nil, fmt.Errorf("default template not found: %s", id)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Merge updates with existing template
	base := defaultTemplate.template

	// Apply updates (preserve ID)
	applyTemplateUpdates(&base, updates)

	// Write to the default template's path
	if err := s.writeTemplate(defaultTemplate.path, &base); err != nil {
		return nil, err
	}

	// Also delete any user override if it exists (since default now matches what user wanted)
	userTemplate, _ := s.findTemplateByID(s.userPath(), id)
	if userTemplate != nil {
		_ = os.Remove(userTemplate.path) // Ignore errors, not critical
	}

	return &TemplateResponse{
		Template:   base,
		Source:     SourceDefault,
		HasDefault: true,
		UpdatedAt:  now,
	}, nil
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

// applyTemplateUpdates applies non-zero update fields to a base template.
func applyTemplateUpdates(base *Template, updates *Template) {
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
	// Always apply Draft (boolean field)
	base.Draft = updates.Draft
}

// getUpdatePath returns the file path and creation time for a template update.
// If the user already has an override, it returns that path. Otherwise, it creates
// a new path based on the template's modes.
func (s *TemplatesService) getUpdatePath(id string, userTemplate *templateWithMeta, modes []string, now string) (string, string, error) {
	if userTemplate != nil {
		return userTemplate.path, userTemplate.createdAt, nil
	}

	// Creating user override of a default
	dir := s.getModePath(s.userPath(), modes)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("failed to create directory: %w", err)
	}
	return filepath.Join(dir, slugify(id)+".json"), now, nil
}
