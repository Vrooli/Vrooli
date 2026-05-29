// Package migration is the stateful migration TRACKER — the project-plan
// half of Vrooli's two-layer architecture-validation model.
//
// Detection has no memory; tracking has all of it. test-genie (the camera)
// produces a normalized findings photograph; this domain INGESTS that
// photograph, tracks every finding through a lifecycle, hands the agent an
// ordered worklist, and on each re-audit RECONCILES by stable ID — findings
// that vanished are validated, findings that persist stay open, and
// findings that (re)appear are flagged as regressions. This is the
// substrate that handholds an agent through a large screaming-architecture
// refactor (the workflow that failed on swarm-manager because the surface
// area was too large to track by hand).
//
// The domain is source-agnostic: it tracks the shared ArchitectureFinding
// contract, so a CLI, UI, docs, standards, or architecture finding are all
// tracked the same way. It NEVER runs detection and never calls test-genie
// or the health CLIs — findings arrive only by ingest (push).
package migration

import (
	"time"
)

// FindingStatus is the lifecycle state of one tracked finding. The states
// mirror the conflict lifecycle verbatim; the migration domain is now the
// canonical owner of this state machine.
//
// Transitions:
//
//	detected ── resolve ──> resolved ── reaudit(absent) ──> validated ── close ──> committed
//	    ^                       │
//	    └── reaudit(present) ───┘   (a resolved finding that reappears is a regression)
type FindingStatus string

const (
	StatusDetected      FindingStatus = "detected"
	StatusAssigned      FindingStatus = "assigned"
	StatusSplit         FindingStatus = "split"
	StatusResolved      FindingStatus = "resolved"
	StatusValidated     FindingStatus = "validated"
	StatusCommitted     FindingStatus = "committed"
	StatusForceResolved FindingStatus = "force_resolved"
)

// IsTerminal reports whether a finding is in a state that no longer needs
// the agent's attention (resolved/validated/committed/force_resolved).
func (s FindingStatus) IsTerminal() bool {
	switch s {
	case StatusResolved, StatusValidated, StatusCommitted, StatusForceResolved:
		return true
	default:
		return false
	}
}

// IsOpen reports whether a finding still needs work (the inverse of
// terminal). Used to build the Next worklist.
func (s FindingStatus) IsOpen() bool { return !s.IsTerminal() }

// MigrationStatus is the lifecycle of a whole migration.
type MigrationStatus string

const (
	MigrationOpen   MigrationStatus = "open"
	MigrationClosed MigrationStatus = "closed"
)

// Finding is one tracked finding inside a migration. It is the durable,
// source-agnostic projection of an ingested ArchitectureFinding plus the
// lifecycle state the tracker layers on top.
type Finding struct {
	// StableID is the afid reconciliation key (the migration's primary
	// identity for the finding). Re-audits match purely on this.
	StableID string
	// Scenario the finding belongs to.
	Scenario string
	// Source token (cli, ui, docs, standards, architecture, structure,
	// tidiness) — the surface that produced the finding.
	Source string
	// Code is the producer-stable finding code.
	Code string
	// Severity token (blocker, error, warn, info).
	Severity string
	// Locations the finding touches.
	Locations []string
	// Domains the finding touches (display only).
	Domains []string
	// Message is the human-readable description.
	Message string
	// Suggestion is the optional remediation hint.
	Suggestion string
	// Status is the lifecycle state.
	Status FindingStatus
	// ResolutionNote is the operator note attached on resolve.
	ResolutionNote string
	// Regressed is true when this finding (re)appeared after the migration
	// began — either a previously-terminal finding that came back, or a
	// brand-new finding surfaced by a re-audit (introduced by the
	// in-progress refactor). It is a loud signal, not a status.
	Regressed bool
	// FirstSeenAt is when the finding was first ingested into the migration.
	FirstSeenAt time.Time
	// UpdatedAt is the last lifecycle/ingest update.
	UpdatedAt time.Time
}

// Migration is a tracked refactor of one scenario toward well-architected.
type Migration struct {
	ID        string
	Scenario  string
	Name      string
	Status    MigrationStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Status is the full status projection returned by the service: the
// migration plus every tracked finding and rollup counts.
type Status struct {
	Migration   Migration
	Findings    []Finding
	Total       int
	Open        int
	Resolved    int
	Validated   int
	Regressions int
	BySeverity  map[string]int
	ByStatus    map[string]int
}

// ReauditResult summarizes what a re-audit reconciled.
type ReauditResult struct {
	Validated   []Finding // were tracked, gone from fresh → validated
	StillOpen   []Finding // tracked and still present
	Regressions []Finding // reappeared-after-terminal or brand-new
	Status      Status
}
