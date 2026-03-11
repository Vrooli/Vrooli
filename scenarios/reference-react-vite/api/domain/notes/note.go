// Package notes defines the note domain model and business rules.
// Notes are free-form text entries that can be attached to tasks.
//
// DOC: docs/concepts/ARCHITECTURE.md#note
// DOC: docs/reference/api-endpoints.md#notes
// DOC: docs/reference/data-model.md#notes
package notes

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
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
func NewNote(input CreateInput) (*Note, error) {
	if strings.TrimSpace(input.TaskID) == "" {
		return nil, errors.New("task_id is required")
	}

	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, errors.New("note content is required")
	}
	if len(content) > 10000 {
		return nil, errors.New("note content must be 10000 characters or less")
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
func (n *Note) ApplyUpdate(input UpdateInput) error {
	if input.Content != nil {
		content := strings.TrimSpace(*input.Content)
		if content == "" {
			return errors.New("note content cannot be empty")
		}
		if len(content) > 10000 {
			return errors.New("note content must be 10000 characters or less")
		}
		n.Content = content
	}

	n.UpdatedAt = time.Now().UTC()
	return nil
}
