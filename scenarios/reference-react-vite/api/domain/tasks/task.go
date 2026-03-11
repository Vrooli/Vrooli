// Package tasks defines the task domain model and business rules.
// Tasks are the core entity for tracking work items within projects.
//
// DOC: docs/concepts/ARCHITECTURE.md#task
// DOC: docs/reference/api-endpoints.md#tasks
// DOC: docs/reference/data-model.md#tasks
// DOC: docs/internal/SEAMS.md#domain-logic-seam
package tasks

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"reference-react-vite/api/domain"
)

// Status represents the lifecycle state of a task.
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusArchived   Status = "archived"
)

// Priority levels for task ordering.
type Priority int

const (
	PriorityLow    Priority = 1
	PriorityMedium Priority = 2
	PriorityHigh   Priority = 3
)

// Task represents a work item that can be tracked and completed.
type Task struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Status      Status    `json:"status"`
	Priority    Priority  `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateInput contains the fields needed to create a new task.
type CreateInput struct {
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	ProjectID   string    `json:"project_id,omitempty"`
	Priority    Priority  `json:"priority,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

// UpdateInput contains the fields that can be updated on a task.
type UpdateInput struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	Status      *Status    `json:"status,omitempty"`
	Priority    *Priority  `json:"priority,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

// ListFilter defines filtering options for listing tasks.
type ListFilter struct {
	ProjectID *string
	Status    *Status
	Priority  *Priority
	Limit     int
	Offset    int
}

// Validate checks if the task status is valid.
func (s Status) Validate() error {
	switch s {
	case StatusPending, StatusInProgress, StatusCompleted, StatusArchived:
		return nil
	default:
		return errors.New("invalid task status")
	}
}

// Validate checks if the priority is valid.
func (p Priority) Validate() error {
	if p < PriorityLow || p > PriorityHigh {
		return errors.New("priority must be between 1 and 3")
	}
	return nil
}

// NewTask creates a new task from input, applying business rules.
// Validation limits come from domain.DefaultValidationLimits() for consistency.
func NewTask(input CreateInput) (*Task, error) {
	limits := domain.DefaultValidationLimits()
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, errors.New("task title is required")
	}
	if len(title) > limits.TaskTitleMaxLength {
		return nil, fmt.Errorf("task title must be %d characters or less", limits.TaskTitleMaxLength)
	}

	priority := input.Priority
	if priority == 0 {
		priority = PriorityMedium
	}
	if err := priority.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &Task{
		ID:          uuid.New().String(),
		ProjectID:   input.ProjectID,
		Title:       title,
		Description: strings.TrimSpace(input.Description),
		Status:      StatusPending,
		Priority:    priority,
		DueDate:     input.DueDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// ApplyUpdate applies update input to the task, enforcing business rules.
// Validation limits come from domain.DefaultValidationLimits() for consistency.
func (t *Task) ApplyUpdate(input UpdateInput) error {
	limits := domain.DefaultValidationLimits()
	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return errors.New("task title cannot be empty")
		}
		if len(title) > limits.TaskTitleMaxLength {
			return fmt.Errorf("task title must be %d characters or less", limits.TaskTitleMaxLength)
		}
		t.Title = title
	}

	if input.Description != nil {
		t.Description = strings.TrimSpace(*input.Description)
	}

	if input.Status != nil {
		if err := input.Status.Validate(); err != nil {
			return err
		}
		t.Status = *input.Status
	}

	if input.Priority != nil {
		if err := input.Priority.Validate(); err != nil {
			return err
		}
		t.Priority = *input.Priority
	}

	if input.DueDate != nil {
		t.DueDate = input.DueDate
	}

	t.UpdatedAt = time.Now().UTC()
	return nil
}
