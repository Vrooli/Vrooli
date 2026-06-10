// Package scoring assembles the cached signals into the product answer:
// maturity rung headline (shared maturity-go ladder), 0-100 composite with
// classification, prioritized recommendations with point impact, and the
// action plan. It owns the score math; raw artifact decoding lives in
// internal/signals and digest/verdict logic in internal/freshness.
package scoring

import (
	"time"

	"scenario-completeness-scoring/internal/freshness"
	"scenario-completeness-scoring/internal/signals"
)

// Result is the assembled score payload (domain shape; the scoring handler
// converts it to the proto contract).
type Result struct {
	Scenario string
	Category string

	Maturity     Maturity
	Composite    Composite
	Freshness    freshness.Result
	Recommends   []Recommendation
	ActionPlan   []ActionPhase
	Degradations []signals.Degradation

	CalculatedAt time.Time
}

// Maturity is the rung headline computed from the cached findings snapshot.
type Maturity struct {
	// WorkingRung is the lowest unsatisfied rung label; empty when the
	// whole ladder holds (LadderClean).
	WorkingRung string
	LadderClean bool

	// SatisfiedThrough is the highest contiguously satisfied rung label
	// starting at R0; empty when even R0 is unsatisfied.
	SatisfiedThrough string

	Dimensions   []DimensionCount
	BuildPassing bool
}

// DimensionCount is the per-dimension finding evidence the gates ran on.
type DimensionCount struct {
	Dimension string
	ErrorPlus int
	Total     int

	// Approximate marks counts inferred from a phase pass/fail status
	// because the cached run predates per-finding persistence.
	Approximate bool
}

// Composite is the ported 0-100 score.
type Composite struct {
	Score               int
	Classification      string
	ClassificationLabel string
	Groups              []Group
}

// Group is one weighted signal group ("quality", "coverage", "quantity",
// "ui").
type Group struct {
	ID      string
	Label   string
	Score   float64
	Max     float64
	Metrics []Metric
}

// Metric is one scored observation inside a group.
type Metric struct {
	ID        string
	Label     string
	Observed  string
	Points    float64
	MaxPoints float64
	Threshold string // "below" | "ok" | "good" | "excellent" | ""
}

// Recommendation is a prioritized improvement with estimated composite
// point impact.
type Recommendation struct {
	Priority    string // "high" | "medium" | "low"
	Description string
	Impact      float64
}

// ActionPhase is one phase of the generated action plan.
type ActionPhase struct {
	Title           string
	Actions         []string
	EstimatedPoints float64
}
