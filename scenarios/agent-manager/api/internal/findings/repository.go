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
	Limit       int
}

type Repository interface {
	Create(context.Context, *Finding) error
	List(context.Context, Filter) ([]Finding, error)
	SetTopic(context.Context, uuid.UUID, string) error
}
