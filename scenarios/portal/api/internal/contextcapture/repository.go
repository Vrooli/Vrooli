package contextcapture

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUnavailable  = errors.New("context unavailable")
	ErrIntentExists = errors.New("context import intent already reserved")
	ErrConflict     = errors.New("context import intent changed")
	ErrQuota        = errors.New("context storage quota exceeded")
)

type State string

const (
	Staging State = "staging"
	Ready   State = "ready"
)

type (
	Record struct {
		Document Document
		Size     int64
		State    State
	}
	Action int
)

const (
	Keep Action = iota
	Publish
	Remove
)

type OwnedID struct {
	Owner string
	ID    string
}

// Repository owns metadata and serialization across blob lifecycle operations.
// WithRecord holds write exclusion until its callback and state change finish.
// A committed reservation survives a process crash during a later blob write.
type ImportIntent struct {
	DocumentID string
	Digest     string
	CreatedAt  time.Time
}

type Repository interface {
	CancelIntent(context.Context, string, string, time.Time, func(Record) error) error
	LookupIntent(context.Context, string, string) (ImportIntent, error)
	PruneIntents(context.Context, time.Time) error
	Reserve(context.Context, Document, int64) error
	WithRecord(context.Context, string, string, func(Record) (Action, error)) error
	Expired(context.Context, time.Time, int) ([]OwnedID, error)
}
