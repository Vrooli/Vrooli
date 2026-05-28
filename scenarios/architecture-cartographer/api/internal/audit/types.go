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
)

// RunInput is the explicit input DTO for Service.Run.
type RunInput struct {
	Scenario     string
	FailOn       conflicts.Severity
	IncludeTypes []string
	ExcludeTypes []string
}

// Report is the orchestrated audit response.
type Report struct {
	Scenario      string
	Outcome       Outcome
	Error         string
	TotalFindings int
	BySeverity    map[string]int
	ByType        map[string]int
	Findings      []conflicts.Conflict
	Domains       DomainSummary
	Graph         GraphSummary
	Duration      time.Duration
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
