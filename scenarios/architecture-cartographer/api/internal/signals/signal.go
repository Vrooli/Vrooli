package signals

import (
	"context"

	"architecture-cartographer/internal/graph"
)

// Signal is one pluggable, deterministic scorer. Each registered signal
// returns a ScoreResult per (chunk, ctx) call — either a non-empty
// Scores slice (each Score carrying ≥1 Evidence) OR a non-nil
// Abstention (carrying Reason + ≥1 Evidence). The aggregator treats
// an empty ScoreResult as a broken-signal contract violation and
// synthesizes a diagnostic abstention so the failure is visible.
type Signal interface {
	// Name is the stable identifier (e.g., "path-token").
	Name() string

	// DefaultWeight is the day-one weight applied before any configured
	// weight overlay (see SIGNAL_LADDER.md::Default Weights Summary).
	DefaultWeight() float64

	// Score returns either Scores (positive evidence for domains) or an
	// Abstention (explicit "no data"). Implementations must obey the
	// five invariants documented at package level.
	Score(ctx context.Context, gctx GraphContext, chunk graph.Chunk) ScoreResult

	// IsAvailable returns false when the signal can't run in the
	// current environment (e.g., git-co-edit when `git` is missing).
	// Aggregator treats unavailable signals as silently skipped; their
	// weight is NOT added to the verdict denominator (an unavailable
	// signal is not "abstaining" — it simply isn't running).
	IsAvailable(ctx context.Context) (bool, string)
}

// abstain is a small helper for signals to produce a typed Abstention
// with a single-evidence entry pointing at the chunk under test.
func abstain(name, reason, locator string) ScoreResult {
	return ScoreResult{
		Abstention: &Abstention{
			Signal: name,
			Reason: reason,
			Evidence: []Evidence{{
				Kind:    "abstain",
				Summary: reason,
				Locator: locator,
			}},
		},
	}
}

// Abstain is the exported helper so signals in other packages can
// construct uniform abstentions without re-deriving the boilerplate.
func Abstain(name, reason, locator string) ScoreResult {
	return abstain(name, reason, locator)
}
