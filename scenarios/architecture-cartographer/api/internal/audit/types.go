// Package audit is the CI-shaped surface over the cartographer. One
// Run call orchestrates graph extract (if no snapshot is available),
// domains derivation, and conflicts detection for a scenario, applies
// severity / type filters, and returns a deterministic summary.
//
// The audit domain owns no state of its own; it is a thin
// orchestrator over graph.Service, domains.Service, conflicts.Service.
package audit

import (
	"fmt"
	"time"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

// Outcome is the verdict shape returned to a CI caller.
type Outcome string

const (
	OutcomeUnspecified Outcome = ""
	OutcomeClean       Outcome = "clean"
	OutcomeFindings    Outcome = "findings"
	OutcomeToolError   Outcome = "tool_error"
	// OutcomePartial signals at least one analysis layer was skipped
	// (today: TS via --skip-ts or the workspace_unsupported error from
	// typescript-code-graph). The remaining layers ran cleanly. CLI maps
	// to exit 0 with a warning banner — better than tool_error when a
	// useful Go-side report still exists.
	OutcomePartial Outcome = "partial"
)

// RunInput is the explicit input DTO for Service.Run.
type RunInput struct {
	Scenario     string
	FailOn       conflicts.Severity
	IncludeTypes []string
	ExcludeTypes []string
	// When true, a LOW or MISSING authority-confidence does NOT flip
	// outcome to FINDINGS. Default false enforces "no curated DOMAINS.md
	// → fail".
	AllowLowAuthority bool
	// When true, skip the TS analysis layer entirely; remaining layers
	// still run. If TS was the only thing skipped (and no findings
	// flip the outcome), result is OutcomePartial.
	SkipTS bool
}

// SnapshotFreshness reports whether the audit reused a cached snapshot
// or extracted a fresh one. Content-hash driven, not time-based.
type SnapshotFreshness string

const (
	SnapshotFreshnessUnspecified SnapshotFreshness = ""
	SnapshotFreshnessCached      SnapshotFreshness = "cached"
	SnapshotFreshnessReExtracted SnapshotFreshness = "re-extracted"
	SnapshotFreshnessFresh       SnapshotFreshness = "fresh"
)

// Report is the orchestrated audit response.
type Report struct {
	Scenario      string
	Outcome       Outcome
	OutcomeReason string
	Error         string
	TotalFindings int
	BySeverity    map[string]int
	ByType        map[string]int
	ByDomain      map[string]int
	Findings      []conflicts.Conflict
	Domains       DomainSummary
	Graph         GraphSummary
	Coverage      CoverageSummary
	Categories    []AuditCategory
	Duration      time.Duration
	// SuppressedFindings is the count of findings whose `// arch:allow`
	// marker matched. Suppressed conflicts are reported (kept in
	// Findings) and counted here.
	SuppressedFindings int
	// SnapshotFreshness reports the snapshot reuse decision. See the
	// SnapshotFreshness* constants.
	SnapshotFreshness SnapshotFreshness
}

// RunAllInput is the explicit input DTO for Service.RunAll.
type RunAllInput struct {
	FailOn            conflicts.Severity
	IncludeTypes      []string
	ExcludeTypes      []string
	IncludeScenarios  []string
	ExcludeScenarios  []string
	AllowLowAuthority bool
	// AllowLowAuthorityScenarios is the per-scenario opt-out list. The
	// effective per-scenario AllowLowAuthority is
	// `AllowLowAuthority || contains(AllowLowAuthorityScenarios, name)` —
	// so the global bool stays a portfolio-wide override while the list
	// silences only the named scenarios. Lets a mixed portfolio strict-
	// gate the curated half without losing real findings on the rest.
	AllowLowAuthorityScenarios []string
	// Concurrency caps the worker pool; 0 → default 4.
	Concurrency int
}

// SweepReport is the aggregated output of Service.RunAll.
type SweepReport struct {
	Reports         []Report
	TotalScenarios  int
	TotalFindings   int
	TotalSuppressed int
	BySeverity      map[string]int
	ByOutcome       map[string]int
	Duration        time.Duration
}

// DomainSummary is a compact projection of the derived domain map used
// to compute the audit.
type DomainSummary struct {
	Authority   string
	Confidence  string
	DomainCount int
}

// GraphSummary is a compact projection of the graph snapshot used to
// compute the audit.
type GraphSummary struct {
	SnapshotID      string
	FileCount       int
	PackageCount    int
	ImportEdgeCount int
}

// CoverageSummary reports how completely placement signals classified
// file chunks during the audit. Buckets are mutually exclusive; a chunk
// whose signals all abstained is counted as all_abstained instead of as
// a generic conflict so low-evidence audits are visible.
type CoverageSummary struct {
	TotalFiles   int
	AutoPlace    CoverageBucket
	Suggest      CoverageBucket
	Conflict     CoverageBucket
	AllAbstained CoverageBucket
}

// CoverageBucket is one count + percentage pair in CoverageSummary.
type CoverageBucket struct {
	Count   int
	Percent float64
}

// AuditCategory is one score-matrix row. Scores are normalized advisory
// signals and never participate directly in audit outcome gating.
type AuditCategory struct {
	Key      string
	Label    string
	Score    float64
	TopItems []CategoryTopItem
}

// CategoryTopItem is the compact finding reference shown under a category.
type CategoryTopItem struct {
	ID           string
	StableID     string
	Type         string
	Subtype      string
	Severity     conflicts.Severity
	FindingClass conflicts.FindingClass
	Locations    []string
	Headline     string
}

// ErrInvalidRunRequest is the typed sentinel returned for invalid input.
type ErrInvalidRunRequest struct {
	Field  string
	Reason string
}

func (e ErrInvalidRunRequest) Error() string {
	return fmt.Sprintf("invalid audit request: %s: %s", e.Field, e.Reason)
}

// derivedSummary builds a DomainSummary from a DerivedDomainMap.
func derivedSummary(m domains.DerivedDomainMap) DomainSummary {
	return DomainSummary{
		Authority:   string(m.Authority),
		Confidence:  string(m.AuthorityConfidence),
		DomainCount: len(m.Domains),
	}
}

// graphSummary builds a GraphSummary from a snapshot.
func graphSummary(snap graph.GraphSnapshot) GraphSummary {
	return GraphSummary{
		SnapshotID:      snap.ID,
		FileCount:       len(snap.Files),
		PackageCount:    len(snap.Packages),
		ImportEdgeCount: len(snap.Imports),
	}
}
