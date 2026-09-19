package docaccess

import (
	"context"
	"time"
)

// Access is one logged document operation.
type Access struct {
	ID           string
	ScenarioName string
	DocType      string
	Operation    string // "read", "write", "reset"
	CreatedAt    time.Time
}

// Filter narrows a stats query. Empty fields match everything.
type Filter struct {
	ScenarioName string
	DocType      string
}

// Stat is the per-scenario, per-doc-type operation tally.
type Stat struct {
	ScenarioName string
	DocType      string
	ReadCount    int
	WriteCount   int
	ResetCount   int
}

// Repository is the docaccess domain's storage surface.
type Repository interface {
	LogAccess(ctx context.Context, a Access) error
	QueryStats(ctx context.Context, filter Filter) ([]Stat, error)
	Recent(ctx context.Context, limit int) ([]Access, error)
}
