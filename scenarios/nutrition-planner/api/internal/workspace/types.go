package workspace

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Workspace struct {
	ID, Name, OwnerSubject string
	Revision               int64
	CreatedAt, UpdatedAt   time.Time
	IdempotencyKey         string
	RequestHash            string
}
type (
	CreateInput struct{ Name, OwnerSubject, IdempotencyKey string }
	Repository  interface {
		Create(context.Context, Workspace) (Workspace, error)
		List(context.Context, string) ([]Workspace, error)
		Get(context.Context, string, string) (Workspace, error)
	}
)
type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("workspace %q not found", e.ID) }

type ErrForbidden struct{ ID string }

func (e ErrForbidden) Error() string { return fmt.Sprintf("workspace %q is not owned by actor", e.ID) }

type ErrInvalid struct{ Field, Reason string }

func (e ErrInvalid) Error() string { return e.Field + ": " + e.Reason }

type ErrIdempotencyConflict struct{ Key string }

func (e ErrIdempotencyConflict) Error() string {
	return fmt.Sprintf("idempotency key %q was used with a different payload", e.Key)
}

func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalid{"name", "must not be empty"}
	}
	return nil
}
