// Package signals collects the raw completeness signals for a target
// scenario from its cached on-disk artifacts. It is the read-model side of
// the scoring flow: collectors decode what other tools have already written
// (requirements registry, requirements-sync metadata, per-run phase results,
// service manifest, UI sources) into one Snapshot.
//
// Contract: collection is filesystem-only — no network, no subprocesses, no
// test execution. Every collector runs behind a circuit breaker; a failing
// collector contributes a Degradation instead of an error, and the score
// path never crashes on malformed input (OT-P0-006).
package signals

import (
	"time"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// Snapshot is the assembled signal set for one scenario. Zero values mean
// "not collected"; consumers must check the per-section Collected flags
// before treating zeros as observations.
type Snapshot struct {
	// Scenario is the directory name; Root the absolute scenario path.
	Scenario string
	Root     string

	// Category from .vrooli/service.json ("utility" when undeclared).
	Category string

	Requirements RequirementsSignals
	Phases       PhaseSignals
	UI           UISignals

	// Degradations lists collectors that could not contribute.
	Degradations []Degradation
}

// RequirementsSignals summarizes the requirements registry and its sync
// metadata.
type RequirementsSignals struct {
	Collected bool

	// Total and Passing count leaf requirements (grouping-only nodes with
	// children and no status are skipped).
	Total   int
	Passing int

	// TargetsTotal / TargetsPassing count operational targets. A target
	// passes when at least half of its linked requirements are complete
	// (legacy threshold, ported).
	TargetsTotal   int
	TargetsPassing int

	// WithValidation counts requirements declaring at least one validation
	// entry (the coverage signal).
	WithValidation int

	// AvgDepth is the mean nesting depth of the requirement tree (legacy
	// depth signal; 3.0+ is the "excellent" band).
	AvgDepth float64
}

// PhaseSignals carries the newest cached result per test-genie phase.
type PhaseSignals struct {
	Collected bool

	// Phases maps phase name -> newest observed result across
	// coverage/runs/<id>/phase-results/*.json (preferred) and the legacy
	// top-level coverage/phase-results/*.json (fallback).
	Phases map[string]PhaseResult
}

// PhaseResult is the newest cached outcome for one phase.
type PhaseResult struct {
	// Status: "passed", "failed", or "skipped".
	Status string

	// UpdatedAt is the result's own timestamp (zero when unparseable).
	UpdatedAt time.Time

	// Findings decoded from the result's `findings` array (the shared
	// ArchitectureFinding contract; enums marshal as proto integers).
	// Nil when the run predates findings persistence — consumers must
	// then approximate from Status and set the `approximate` flag on
	// anything derived.
	Findings []*architecturev1.ArchitectureFinding

	// HasFindings distinguishes "findings key present but empty" (clean
	// pass, zero findings) from "findings key absent" (older writer).
	HasFindings bool
}

// UISignals carries the ported UI heuristics.
type UISignals struct {
	Collected bool

	// IsTemplate is true when ui sources still look like the generated
	// starter (template signature strings / minimal App shell).
	IsTemplate bool

	FileCount      int
	ComponentCount int
	PageCount      int

	// APIEndpoints counts unique endpoint references found in UI sources;
	// APIBeyondHealth excludes /health and /status.
	APIEndpoints    int
	APIBeyondHealth int

	HasRouting bool
	RouteCount int

	TotalLOC int
}

// Degradation reports one collector that could not contribute.
type Degradation struct {
	// Collector id: "requirements", "phases", "service", "ui".
	Collector string

	// State: "failed" (this collection) or "open" (circuit breaker has
	// disabled the collector after repeated failures).
	State string

	// Reason is the human-readable cause.
	Reason string
}
