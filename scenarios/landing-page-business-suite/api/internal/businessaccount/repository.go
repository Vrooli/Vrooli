package businessaccount

import (
	"context"
	"errors"
)

var (
	ErrInvalid       = errors.New("invalid business account")
	ErrNotFound      = errors.New("business account not found")
	ErrNotMember     = errors.New("user is not a member of business account")
	ErrAlreadyExists = errors.New("business account already exists")
)

// Repository hides the storage engine from the business-account domain.
// userEmail is used only to create the user's initial personal account and to
// retain the current commerce service's email-based entitlement projection.
type Repository interface {
	ListForUser(context.Context, string, string) ([]Account, error)
	ResolveForUser(context.Context, string, string, string) (Account, error)
	CreateForUser(context.Context, string, string, string) (Account, error)
}
