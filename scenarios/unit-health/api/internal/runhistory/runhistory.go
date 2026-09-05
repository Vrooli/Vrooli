// Package runhistory persists unit-health validation runs to the scenario-local
// SQLite database and exposes cross-run timing/status history. It is the durable
// substrate behind the diagnostics analyzer's runtime-growth and flake signals,
// which need history that single-run heuristics cannot provide.
package runhistory

import (
	"context"
	_ "embed"
	"time"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the run-history DDL for EnsureSchemas (forward-only).
func Schema() string { return schemaSQL }

// CommandSample is one command's outcome in one run.
type CommandSample struct {
	RunID        string
	StartedAt    time.Time
	WorkspaceID  string
	Command      string
	DurationMS   int64
	Status       string
	FailureClass string
}

// CoverageSample is one file's coverage percent in one run.
type CoverageSample struct {
	WorkspaceID string
	File        string
	Percent     float64
}

// RunRecord is a full run to persist.
type RunRecord struct {
	RunID        string
	Scenario     string
	StartedAt    time.Time
	Status       string
	MaturityRung int
	Commands     []CommandSample
	Coverage     []CoverageSample
}

// Store persists runs and reads back command history. It is the seam the
// validation service depends on; a nil Store disables persistence (diagnostics
// then fall back to single-run heuristics).
type Store interface {
	// Record persists one run and prunes history beyond the retention window.
	Record(ctx context.Context, rec RunRecord) error
	// CommandHistory returns command samples for the scenario across the most
	// recent runs (up to runLimit runs), newest first.
	CommandHistory(ctx context.Context, scenario string, runLimit int) ([]CommandSample, error)
}
