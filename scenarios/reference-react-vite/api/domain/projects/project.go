// Package projects defines the project domain model and business rules.
// Projects are containers that group related tasks together.
//
// DOC: docs/concepts/ARCHITECTURE.md#project
// DOC: docs/reference/api-endpoints.md#projects
// DOC: docs/reference/data-model.md#projects
// DOC: docs/internal/SEAMS.md#domain-logic-seam
package projects

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"reference-react-vite/api/domain"
)

// Status represents the lifecycle state of a project.
type Status string

const (
	StatusActive   Status = "active"
	StatusPaused   Status = "paused"
	StatusComplete Status = "complete"
	StatusArchived Status = "archived"
)

// Project represents a container for organizing tasks.
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Status      Status    `json:"status"`
	Color       string    `json:"color,omitempty"`
	TaskCount   int       `json:"task_count,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateInput contains the fields needed to create a new project.
type CreateInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
}

// UpdateInput contains the fields that can be updated on a project.
type UpdateInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *Status `json:"status,omitempty"`
	Color       *string `json:"color,omitempty"`
}

// ListFilter defines filtering options for listing projects.
type ListFilter struct {
	Status *Status
	Limit  int
	Offset int
}

// Validate checks if the project status is valid.
func (s Status) Validate() error {
	switch s {
	case StatusActive, StatusPaused, StatusComplete, StatusArchived:
		return nil
	default:
		return errors.New("invalid project status")
	}
}

// ValidateColor checks if a color is a valid hex color code.
// Uses domain.IsValidHexColor() for consistent validation across packages.
func ValidateColor(color string) error {
	if !domain.IsValidHexColor(color) {
		return errors.New("color must be a valid hex code (e.g., #FF5733)")
	}
	return nil
}

// NewProject creates a new project from input, applying business rules.
// Validation limits come from domain.DefaultValidationLimits() for consistency.
func NewProject(input CreateInput) (*Project, error) {
	limits := domain.DefaultValidationLimits()
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("project name is required")
	}
	if len(name) > limits.ProjectNameMaxLength {
		return nil, fmt.Errorf("project name must be %d characters or less", limits.ProjectNameMaxLength)
	}

	if err := ValidateColor(input.Color); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &Project{
		ID:          uuid.New().String(),
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Status:      StatusActive,
		Color:       input.Color,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// ApplyUpdate applies update input to the project, enforcing business rules.
// Validation limits come from domain.DefaultValidationLimits() for consistency.
func (p *Project) ApplyUpdate(input UpdateInput) error {
	limits := domain.DefaultValidationLimits()
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return errors.New("project name cannot be empty")
		}
		if len(name) > limits.ProjectNameMaxLength {
			return fmt.Errorf("project name must be %d characters or less", limits.ProjectNameMaxLength)
		}
		p.Name = name
	}

	if input.Description != nil {
		p.Description = strings.TrimSpace(*input.Description)
	}

	if input.Status != nil {
		if err := input.Status.Validate(); err != nil {
			return err
		}
		p.Status = *input.Status
	}

	if input.Color != nil {
		if err := ValidateColor(*input.Color); err != nil {
			return err
		}
		p.Color = *input.Color
	}

	p.UpdatedAt = time.Now().UTC()
	return nil
}
