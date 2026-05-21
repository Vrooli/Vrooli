package signals

import (
	"context"

	"architecture-cartographer/internal/graph"
)

// Signal is one pluggable, deterministic scorer. Each registered signal
// produces zero or more Score entries per (chunk, ctx) call. Empty
// evidence on a returned Score is treated by the aggregator as a broken
// signal — the score is dropped and the signal flagged.
type Signal interface {
	// Name is the stable identifier (e.g., "path-token").
	Name() string

	// DefaultWeight is the day-one weight applied before any manifest
	// overlay (see SIGNAL_LADDER.md::Default Weights Summary).
	DefaultWeight() float64

	// Score the chunk against every plausible domain. Implementations
	// must obey the five invariants documented at package level.
	Score(ctx context.Context, gctx GraphContext, chunk graph.Chunk) []Score

	// IsAvailable returns false when the signal can't run in the
	// current environment (e.g., git-co-edit when `git` is missing).
	// Aggregator treats unavailable signals as silently skipped.
	IsAvailable(ctx context.Context) (bool, string)
}
