package settlement

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("settlement not found")

// ClaimResult identifies whether this caller durably acquired the only right
// to invoke the rail for an idempotency key.
type ClaimResult struct {
	Record  Record
	Claimed bool
}

type Repository interface {
	Get(context.Context, string) (Record, error)
	GetByIdempotencyKey(context.Context, string) (Record, error)
	Claim(context.Context, Record) (ClaimResult, error)
	Complete(context.Context, string, Outcome, RailResult, string, string) (Record, error)
}
