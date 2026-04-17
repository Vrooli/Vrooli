// Package services provides business logic orchestration.
// This file implements file I/O operations for the template system.
package services

import (
	"agent-inbox/config"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

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
