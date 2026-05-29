// Package conflicts is the DETECTION-ONLY home for drift detection, the
// pluggable Detector/Resolver registries, and conflict persistence. It is
// a stateless photograph of what is wrong now; it owns no lifecycle. The
// stateful per-finding lifecycle (assign/resolve/validate/regress) lives in
// the migration domain, which ingests these findings and tracks them over
// time.
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
//
// ID, StableID, InstanceID:
//   - StableID (v0.2+) is the deterministic content-hash primary key
//     produced by StableID(). It is stable across runs for the same
//     underlying drift.
//   - InstanceID is the per-run UUID preserved from v0.1 so external
//     systems can dedupe on the prior key during the transition.
//   - ID is the user-facing alias and equals StableID after the
//     DetectConflicts pipeline runs.
type Conflict struct {
	ID             string
	StableID       string
	InstanceID     string
	Scenario       string
	Detector       string
	Type           string
	Subtype        string
	Severity       Severity
	Locations      []string
	Domains        []string
	Evidence       []Evidence
	SuggestedFixes []Fix
	SnapshotID     string
	Verdict        *signals.Verdict
	// Suppressed is true when an active in-repo `// arch:allow` marker
	// sanctions this finding. A suppressed conflict is reported (not
	// dropped) so the operator sees what is being excused and why.
	Suppressed bool
	// SuppressionReason is the marker's reason, when Suppressed.
	SuppressionReason string
	DetectedAt        time.Time
	UpdatedAt         time.Time
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

// ErrInvalidInput signals a malformed request to a detection operation
// (e.g., a missing scenario). Mapped to CodeInvalidArgument.
type ErrInvalidInput struct {
	Reason string
}

func (e ErrInvalidInput) Error() string {
	return fmt.Sprintf("invalid input: %s", e.Reason)
}
