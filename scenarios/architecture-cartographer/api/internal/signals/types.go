// Package signals is the domain-scoped home for the pluggable
// deterministic signal registry, per-signal scoring, and the aggregator.
//
// Signal invariants (enforced in code, asserted in tests):
//  1. Pure — given the same (chunk, graph, manifest), produces the same Score.
//  2. No mutation — never mutates the graph or manifest.
//  3. Self-explaining — every Score carries Evidence; aggregator refuses
//     empty Evidence and treats the signal as broken.
//  4. Bounded — Score.Value in [0, 1].
//  5. Cheap — caches (community detection, glossary lookups) live on
//     GraphContext, not on the signal.
//
// Phase 2 ships the types + registry stubs; Phase 3 fills in the
// detectors / aggregator and the six day-one signals.
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

// Evidence is one piece of justification for a Score.
type Evidence struct {
	Kind    string
	Summary string
	Locator string
	Weight  float64
}

// Score is one signal's output for one chunk. Aggregator refuses Scores
// with empty Evidence (signal is treated as broken).
type Score struct {
	Signal   string
	Domain   string
	Value    float64
	Reason   string
	Evidence []Evidence
}

// DomainValue is one row of the aggregator's per-domain summary.
type DomainValue struct {
	Domain string
	Value  float64
}

// Verdict is the aggregator's output for one chunk.
type Verdict struct {
	ChunkID        string
	ChunkPath      string
	Tier           Tier
	TopDomain      string
	TopValue       float64
	RunnerUpDomain string
	RunnerUpValue  float64
	Scores         []Score
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
