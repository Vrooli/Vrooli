package convergence

import (
	"context"
	"time"
)

// Repository is the owned-state seam for the fitness-audit index: dated
// per-template fitness records (the trend) and the latest per-reference health.
// Production wires the SQLite implementation; tests use a fake. A nil Repository
// disables persistence — the service still computes live, but no trend
// accumulates and GetConvergenceTrend returns empty.
type Repository interface {
	// SaveFitness appends one dated fitness record per template (the trend grows).
	SaveFitness(ctx context.Context, fitness []TemplateFitness, at time.Time) error
	// SaveReferences records the latest reference-health verdicts (dated).
	SaveReferences(ctx context.Context, refs []ReferenceHealth, at time.Time) error
	// Trend returns dated trend points (optionally for one template), oldest first.
	Trend(ctx context.Context, template string) ([]FitnessTrendPoint, error)
}
