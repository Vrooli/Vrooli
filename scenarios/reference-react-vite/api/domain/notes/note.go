// Package notes defines the note domain model and business rules.
// Notes are free-form text entries that can be attached to tasks.
//
// DOC: docs/concepts/ARCHITECTURE.md#note
// DOC: docs/reference/api-endpoints.md#notes
// DOC: docs/reference/data-model.md#notes
// DOC: docs/internal/SEAMS.md#domain-logic-seam
package notes

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"reference-react-vite/api/domain"
)

// Note represents a text annotation attached to a task.
type Note struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	Content   string    `json:"content"`
	Author    string    `json:"author,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateInput contains the fields needed to create a new note.
type CreateInput struct {
	TaskID  string `json:"task_id"`
	Content string `json:"content"`
	Author  string `json:"author,omitempty"`
}

// UpdateInput contains the fields that can be updated on a note.
type UpdateInput struct {
	Content *string `json:"content,omitempty"`
}

// ListFilter defines filtering options for listing notes.
type ListFilter struct {
	TaskID string
	Limit  int
	Offset int
}

// NewNote creates a new note from input, applying business rules.
// Validation limits come from domain.DefaultValidationLimits() for consistency.
func NewNote(input CreateInput) (*Note, error) {
	limits := domain.DefaultValidationLimits()
	if strings.TrimSpace(input.TaskID) == "" {
		return nil, errors.New("task_id is required")
	}

	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, errors.New("note content is required")
	}
	if len(content) > limits.NoteContentMaxLength {
		return nil, fmt.Errorf("note content must be %d characters or less", limits.NoteContentMaxLength)
	}

	now := time.Now().UTC()
	return &Note{
		ID:        uuid.New().String(),
		TaskID:    input.TaskID,
		Content:   content,
		Author:    strings.TrimSpace(input.Author),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// ApplyUpdate applies update input to the note, enforcing business rules.
// Validation limits come from domain.DefaultValidationLimits() for consistency.
func (n *Note) ApplyUpdate(input UpdateInput) error {
	limits := domain.DefaultValidationLimits()
	if input.Content != nil {
		content := strings.TrimSpace(*input.Content)
		if content == "" {
			return errors.New("note content cannot be empty")
		}
		if len(content) > limits.NoteContentMaxLength {
			return fmt.Errorf("note content must be %d characters or less", limits.NoteContentMaxLength)
		}
		n.Content = content
	}

	n.UpdatedAt = time.Now().UTC()
	return nil
}
