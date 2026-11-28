// Package projectid owns normalization and validation of external project IDs.
package projectid

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrRequired = errors.New("project_id is required")
	ErrInvalid  = errors.New("invalid project_id")
)

// Parse trims raw and distinguishes an omitted ID from a malformed one.
func Parse(raw string) (uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.Nil, ErrRequired
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, ErrInvalid
	}
	return id, nil
}
