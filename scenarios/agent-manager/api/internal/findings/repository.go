package findings

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Filter struct {
	RunID       *uuid.UUID
	Since       *time.Time
	Fingerprint string
	Severity    string
	Decision    string
	Limit       int
}

type Repository interface {
	Create(context.Context, *Finding) error
	List(context.Context, Filter) ([]Finding, error)
	SetDecision(context.Context, uuid.UUID, string) error
	RecurrenceCount(context.Context, string) (int, error)
}
