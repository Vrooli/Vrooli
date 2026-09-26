// Package signals is the domain-scoped home for the pluggable
// deterministic signal registry, per-signal scoring, and the aggregator.
//
// Signal invariants (enforced in code, asserted in tests):
//  1. Pure — given the same (chunk, graph, domain map), produces the same Score.
//  2. No mutation — never mutates the graph or domain map.
//  3. Self-explaining — every Score carries Reason + ≥1 Evidence; every
//     Abstention carries Reason + ≥1 Evidence. Nothing is returned without
//     explanation: a signal that has no data for a chunk MUST emit an
//     explicit Abstention (it never returns an empty ScoreResult).
//  4. Bounded — Score.Value in [0, 1].
//  5. Cheap — caches (community detection, glossary lookups) live on
//     GraphContext, not on the signal.
package signals

import (
	"fmt"
)

// Tier classifies an aggregated verdict by confidence band.
type Tier string

const (
	TierUnspecified Tier = ""
	TierAutoPlace   Tier = "auto_place"
	TierSuggest     Tier = "suggest"
	TierConflict    Tier = "conflict"
)

// Evidence is one piece of justification for a Score or an Abstention.
type Evidence struct {
	Kind    string
	Summary string
	Locator string
	Weight  float64
}

// Score is one signal's output for one (chunk, domain) pair. The
// aggregator validates that Evidence is non-empty; signals that
// produce a Score with empty Evidence violate the self-explaining
// invariant and are treated as broken (their weight counts toward
// participation confidence and the verdict carries a synthetic abstention).
type Score struct {
	Signal   string
	Domain   string
	Value    float64
	Reason   string
	Evidence []Evidence
}

// Abstention is an explicit "I have no data for this chunk" emission
// from a signal. Reason describes why; Evidence carries at least one
// concrete pointer (e.g., {Kind:"abstain", Summary:"no importers",
// Locator:chunk.Path}). The aggregator counts abstaining signals toward
// available participation weight, but not toward direction normalization.
type Abstention struct {
	Signal   string
	Reason   string
	Evidence []Evidence
}

// ScoreResult is the per-signal return shape. Exactly one of Scores or
// Abstention is populated by a well-behaved signal:
//   - Scores non-empty (each entry with ≥1 Evidence): signal contributed
//     positive evidence for the listed domains.
//   - Abstention non-nil: signal had nothing to say; the verdict still
//     accounts for its weight in participation confidence.
//
// A ScoreResult that is empty in both fields violates the self-
// explaining invariant; the aggregator records this as a synthetic
// broken-signal abstention so it surfaces in the verdict confidence.
type ScoreResult struct {
	Scores     []Score
	Abstention *Abstention
}

// DomainValue is one row of the aggregator's per-domain summary.
type DomainValue struct {
	Domain         string
	DirectionValue float64
}

// Verdict is the aggregator's output for one chunk.
type Verdict struct {
	ChunkID        string
	ChunkPath      string
	Tier           Tier
	TopDomain      string
	TopValue       float64
	DirectionValue float64
	Confidence     float64
	QuorumMet      bool
	RunnerUpDomain string
	RunnerUpValue  float64
	Scores         []Score
	Abstentions    []Abstention
	DomainValues   []DomainValue
	Tied           bool
}

// SignalDescriptor describes one registered signal.
type SignalDescriptor struct {
	Name           string
	DefaultWeight  float64
	Stability      string
	Description    string
	Disabled       bool
	DisabledReason string
}

// ErrInvalidScoreRequest is the typed sentinel returned when scoring
// input is incomplete (e.g., empty scenario, no chunk and no file id).
type ErrInvalidScoreRequest struct {
	Field  string
	Reason string
}

func (e ErrInvalidScoreRequest) Error() string {
	return fmt.Sprintf("invalid score request: %s: %s", e.Field, e.Reason)
}
