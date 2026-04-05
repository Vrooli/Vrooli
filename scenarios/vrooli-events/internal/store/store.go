package store

import (
	"context"
	"time"
)

// Event is the internal representation of an event in the store.
type Event struct {
	ID             int64
	EventID        string
	SourceScenario string
	TargetScenario string
	EventType      string
	CorrelationID  string
	Payload        []byte
	Metadata       map[string]string
	CreatedAt      time.Time
}

// QueryFilters defines filters for querying events.
type QueryFilters struct {
	EventType     string // glob pattern
	Source        string // exact match
	CorrelationID string // exact match
	Since         int64  // return events with ID > Since
	Limit         int    // max results (default 100)
}

// PruneResult reports what the pruning operation removed.
type PruneResult struct {
	TimeDeletedCount int64
	SizeDeletedCount int64
}

// Stats holds event store statistics.
type Stats struct {
	TotalEvents       int64
	TotalPayloadBytes int64
	OldestEvent       *time.Time
	NewestEvent       *time.Time
}

// Store defines the event storage interface.
type Store interface {
	Insert(ctx context.Context, e Event) (int64, error)
	Query(ctx context.Context, f QueryFilters) ([]Event, error)
	GetSince(ctx context.Context, lastID int64, limit int) ([]Event, error)
	Prune(ctx context.Context) (PruneResult, error)
	Stats(ctx context.Context) (Stats, error)
	Close() error
}
