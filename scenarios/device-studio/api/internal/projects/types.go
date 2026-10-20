package projects

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

type Project struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	Revision  int64
}

type CreateInput struct{ Name string }
type RenameInput struct {
	ID               string
	Name             string
	ExpectedRevision int64
}

type ErrInvalidProject struct{ Field, Reason string }

func (e ErrInvalidProject) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

type ErrProjectNotFound struct{ ID string }

func (e ErrProjectNotFound) Error() string { return fmt.Sprintf("project %q not found", e.ID) }

type ErrRevisionConflict struct {
	ID                string
	Expected, Current int64
}

func (e ErrRevisionConflict) Error() string {
	return fmt.Sprintf("project %q revision conflict: expected %d, current %d", e.ID, e.Expected, e.Current)
}

func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidProject{Field: "name", Reason: "required"}
	}
	if len([]rune(name)) > 120 {
		return ErrInvalidProject{Field: "name", Reason: "must be 120 characters or fewer"}
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return ErrInvalidProject{Field: "name", Reason: "must not contain control characters"}
		}
	}
	return nil
}
