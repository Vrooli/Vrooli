package recovery

import (
	"context"
	"time"
)

// EventRetentionWindow bounds the recovery event log. Ninety days keeps
// incident history useful for postmortems while preventing unbounded local
// growth on hosts that run unattended.
const EventRetentionWindow = 90 * 24 * time.Hour

// Repository is the persistence seam for the recovery_events log.
// Production wires the sqlite-backed implementation from sqlite.go;
// service unit tests wire a fake.
type Repository interface {
	// PersistEvent stores one attempt. The implementation populates ID
	// and CreatedAt when zero-valued and returns the stored row.
	PersistEvent(ctx context.Context, e RecoveryEvent) (RecoveryEvent, error)

	// ListEvents returns the most recent events, newest first, capped at
	// limit. A non-positive limit applies the default.
	ListEvents(ctx context.Context, limit int) ([]RecoveryEvent, error)
}

// DefaultEventLimit caps ListEvents when the caller passes 0.
const DefaultEventLimit = 50
