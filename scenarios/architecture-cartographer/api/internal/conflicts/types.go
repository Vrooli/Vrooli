// Package conflicts is the domain-scoped home for drift detection, the
// pluggable Detector/Resolver registries, conflict persistence, and the
// per-conflict lifecycle state machine.
//
// Conflict envelope shape is locked in v0.1: adding a new Detector does
// not add fields to Conflict; only optional fields on Fix may be added.
package conflicts

import (
	"fmt"
	"time"

	"architecture-cartographer/internal/signals"
)

// Severity classifies how blocking a conflict is.
type Severity string

const (
	SeverityUnspecified Severity = ""
	SeverityInfo        Severity = "info"
	SeverityWarn        Severity = "warn"
	SeverityError       Severity = "error"
	SeverityBlocker     Severity = "blocker"
)

// ResolutionStatus is the lifecycle state.
type ResolutionStatus string

const (
	ResolutionStatusDetected      ResolutionStatus = "detected"
	ResolutionStatusAssigned      ResolutionStatus = "assigned"
	ResolutionStatusSplit         ResolutionStatus = "split"
	ResolutionStatusResolved      ResolutionStatus = "resolved"
	ResolutionStatusValidated     ResolutionStatus = "validated"
	ResolutionStatusCommitted     ResolutionStatus = "committed"
	ResolutionStatusForceResolved ResolutionStatus = "force_resolved"
)

// FixKind enumerates the operator-facing fix categories.
type FixKind string

const (
	FixKindUnspecified     FixKind = ""
	FixKindMoveFile        FixKind = "move_file"
	FixKindReassignDomain  FixKind = "reassign_domain"
	FixKindBreakCycle      FixKind = "break_cycle"
	FixKindAddDependency   FixKind = "add_dependency"
	FixKindAddTransitional FixKind = "add_transitional"
)

// Evidence is one detector-supplied justification.
type Evidence struct {
	Kind    string
	Summary string
	Locator string
	Payload []byte
}

// Fix is one suggested resolution.
type Fix struct {
	ID         string
	Kind       FixKind
	Resolver   string
	Summary    string
	Payload    []byte
	Confidence float64
}

// Conflict is the canonical envelope a Detector emits.
type Conflict struct {
	ID             string
	Scenario       string
	Detector       string
	Type           string
	Subtype        string
	Severity       Severity
	Locations      []string
	Domains        []string
	Evidence       []Evidence
	SuggestedFixes []Fix
	Status         ResolutionStatus
	AssignedDomain string
	ResolutionNote string
	SnapshotID     string
	Verdict        *signals.Verdict
	DetectedAt     time.Time
	UpdatedAt      time.Time
}

// DetectorDescriptor describes one registered detector.
type DetectorDescriptor struct {
	Name        string
	Description string
	Stability   string
	EmitsTypes  []string
}

// ResolverDescriptor describes one registered resolver.
type ResolverDescriptor struct {
	Name          string
	Description   string
	Stability     string
	HandlesKinds  []FixKind
	RequiresApply bool
}

// ErrConflictNotFound is the typed sentinel returned by GetConflict
// when no row matches.
type ErrConflictNotFound struct {
	ID string
}

func (e ErrConflictNotFound) Error() string {
	return fmt.Sprintf("conflict %q not found", e.ID)
}

// ErrInvalidTransition signals that the operator-requested status
// change is not allowed from the current status.
type ErrInvalidTransition struct {
	From ResolutionStatus
	To   ResolutionStatus
}

func (e ErrInvalidTransition) Error() string {
	return fmt.Sprintf("invalid conflict transition %s -> %s", e.From, e.To)
}

// ErrInvalidAssignment signals that the assigned domain is not declared
// in the manifest.
type ErrInvalidAssignment struct {
	Domain string
	Reason string
}

func (e ErrInvalidAssignment) Error() string {
	return fmt.Sprintf("invalid assignment to domain %q: %s", e.Domain, e.Reason)
}
