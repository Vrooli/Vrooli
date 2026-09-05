// Package services provides business logic orchestration.
// This file defines types and interfaces for the template system.
package services

import (
	"sync"

	"agent-inbox/config"
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
	Draft             bool               `json:"draft,omitempty"` // Indicates template may not be fully working
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

// templateWithMeta is a template with file metadata.
type templateWithMeta struct {
	template  Template
	path      string
	createdAt string
	updatedAt string
}
