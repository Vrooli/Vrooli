// Package effectiveness is the controller's runtime ledger of how well each
// skill closes each dimension, per token spent. The selection bandit
// (pkg/autosteer.Selector) reads it to rank eligible skills by expected
// reduction-per-token; credit assignment (after every iteration) writes to it.
//
// The math is kept here, out of pkg/autosteer, so it can be unit-tested in
// isolation and reused by both selection and the Layer-2 thrashing defense.
package effectiveness

import (
	"time"

	"github.com/vrooli/maturity-go/dimensions"
)

// Stat is one (skill, dimension) row of the lifetime effectiveness ledger. The
// raw counts are authoritative; the per-token efficacy is always derived on read
// (never stored) so the prior/shrinkage policy can evolve without a migration.
type Stat struct {
	SkillID         string               `json:"skill_id"`
	Dimension       dimensions.Dimension `json:"dimension"`
	ClosedCount     int64                `json:"closed_count"`
	IntroducedCount int64                `json:"introduced_count"`
	TotalRuns       int64                `json:"total_runs"`
	TotalTokens     int64                `json:"total_tokens"`
	LastRunAt       time.Time            `json:"last_run_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

// NetClosed is the lifetime net findings this skill closed in this dimension
// (closed minus self-inflicted). It can be negative when a skill introduces more
// than it closes — exactly the signal the regression defenses key on.
func (s Stat) NetClosed() int64 { return s.ClosedCount - s.IntroducedCount }

// CreditEvent is the per-iteration outcome attributed to the skill that ran. It
// is recorded after MEASURE: the skill targeted TargetDimension, the run cost
// Tokens, and the re-audit observed these per-dimension findings deltas
// (collateral dimensions included so cross-dimension debt is earned).
type CreditEvent struct {
	SkillID               string
	TargetDimension       dimensions.Dimension
	ClosedByDimension     map[dimensions.Dimension]int
	IntroducedByDimension map[dimensions.Dimension]int
	// Tokens is the run's token cost (0 = unknown; recorded faithfully, never
	// treated as a free run by the derived efficacy).
	Tokens int64
}

// Store is the effectiveness ledger's read/write boundary.
//
// seam: Store is the controller's effectiveness ledger. Production wires
// PostgresStore (backed by skill_dimension_effectiveness); tests wire MemoryStore.
type Store interface {
	// Get returns the (skill, dimension) stat and whether a row exists (no row =
	// never observed = cold start).
	Get(skillID string, dim dimensions.Dimension) (Stat, bool, error)
	// Bulk returns every skill's stat for one dimension, keyed by skill ID — the
	// selection hot path (rank eligible skills within the heaviest dimension).
	Bulk(dim dimensions.Dimension) (map[string]Stat, error)
	// Record applies one credit event (atomic, commutative increments).
	Record(ev CreditEvent) error
	// List returns ledger rows, optionally filtered by skill and/or dimension
	// (empty filter = whole table). Used by the operator effectiveness view.
	List(skillID string, dim dimensions.Dimension) ([]Stat, error)
}

// efficacyEpsilon avoids division-by-zero and damps the per-token rate when a
// run reports very few tokens.
const efficacyEpsilon = 1.0

// DefaultShrinkageK is the shrinkage constant: the estimate equals the prior
// until roughly K runs of evidence accumulate. Small (3) because there are few
// skills per dimension and evidence is precious.
const DefaultShrinkageK = 3.0

// ObservedEfficacyPerToken is the raw, un-shrunk efficacy: net findings closed
// per 1000 tokens. When the token cost is unknown (no positive total) it falls
// back to net findings per run — degraded but valid, and never treats the run as
// free (see the cold-start note in the P1 plan).
func (s Stat) ObservedEfficacyPerToken() float64 {
	net := float64(s.NetClosed())
	if s.TotalTokens > 0 {
		return net / (float64(s.TotalTokens)/1000.0 + efficacyEpsilon)
	}
	if s.TotalRuns > 0 {
		return net / float64(s.TotalRuns)
	}
	return 0
}

// ExpectedEfficacyPerToken blends the observed efficacy with a cold-start prior
// via Bayesian shrinkage: (n/(n+k))·observed + (k/(n+k))·prior, n = total_runs.
// With n = 0 the result is exactly the prior, so a fresh target reproduces v0
// greedy ordering (uniform prior ⇒ all eligible skills tie). k must be > 0.
func (s Stat) ExpectedEfficacyPerToken(prior, k float64) float64 {
	if k <= 0 {
		k = DefaultShrinkageK
	}
	n := float64(s.TotalRuns)
	if n <= 0 {
		return prior
	}
	w := n / (n + k)
	return w*s.ObservedEfficacyPerToken() + (1-w)*prior
}

// SampleSize reports how many runs of evidence back this stat (the bandit's n).
func (s Stat) SampleSize() int64 { return s.TotalRuns }
