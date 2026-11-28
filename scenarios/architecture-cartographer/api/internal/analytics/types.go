// Package analytics is the domain-scoped home for cartographer's
// append-only event log, placement records, and override tracking.
//
// Analytics is the highest-leverage calibration loop — every conflict
// detection, resolution, override, and verdict produces an event here.
// The events table is append-only: corrections are new rows with
// CorrectsEventID set, never UPDATE or DELETE.
//
// Layering mirrors the canonical Vrooli pattern (handler → Service →
// Repository), with mocks/ co-located so service tests don't reach the
// sqlite repository.
package analytics

import (
	"fmt"
	"time"
)

// EventKind enumerates the closed set of analytic event types. The
// values mirror the proto's analytics_v1.EventKind enum; handlers
// translate at the boundary.
type EventKind string

const (
	EventKindConflictDetected      EventKind = "conflict_detected"
	EventKindConflictAssigned      EventKind = "conflict_assigned"
	EventKindConflictResolved      EventKind = "conflict_resolved"
	EventKindConflictReopened      EventKind = "conflict_reopened"
	EventKindConflictForceResolved EventKind = "conflict_force_resolved"
	EventKindVerdictProduced       EventKind = "verdict_produced"
	EventKindPlacementAuto         EventKind = "placement_auto"
	EventKindPlacementSuggest      EventKind = "placement_suggest"
	EventKindOverrideRecorded      EventKind = "override_recorded"
	EventKindApplyPlanned          EventKind = "apply_planned"
	EventKindApplyRan              EventKind = "apply_ran"
	EventKindApplyBuildGreen       EventKind = "apply_build_green"
	EventKindApplyBuildRed         EventKind = "apply_build_red"
	EventKindApplyReverted         EventKind = "apply_reverted"
)

// Event is one append-only record. Payload is canonical-form JSON
// bytes; consumers decode based on Kind.
type Event struct {
	ID              string
	Kind            EventKind
	Scenario        string
	Domain          string
	ConflictID      string
	ChunkID         string
	PlanID          string
	RunID           string
	CorrectsEventID string
	Payload         []byte
	Actor           string
	RecordedAt      time.Time
}

// Placement is one auto-placement outcome row. Kept separate from
// events for cheap stats queries.
type Placement struct {
	ID        string
	Scenario  string
	ChunkID   string
	ChunkPath string
	// Verdict serialized canonical-form JSON of signals.Verdict.
	VerdictJSON []byte
	Outcome     string // "auto_placed" | "suggested" | "overridden" | "rejected"
	AutoActed   bool
	RecordedAt  time.Time
}

// Override is one operator override of a verdict.
type Override struct {
	ID             string
	Scenario       string
	ChunkID        string
	VerdictDomain  string
	ChosenDomain   string
	Note           string
	VerdictEventID string
	RecordedAt     time.Time
}

// StatsSummary is the response shape for Service.GetStats. Suppresses
// success-rate fields when N<5 (the calibration threshold).
type StatsSummary struct {
	Scenario                     string
	ConflictsDetected            int64
	ConflictsResolved            int64
	ConflictsForceResolved       int64
	PlacementsAuto               int64
	PlacementsSuggest            int64
	Overrides                    int64
	VerdictSuccessRate           float64
	VerdictSuccessRateSuppressed bool
	VerdictObservationCount      int64
}

// MinVerdictObservations is the threshold below which success rate is
// suppressed. Documented at scenarios/architecture-cartographer/docs/
// concepts/DOMAINS.md::analytics::Threshold rule.
const MinVerdictObservations = 5

// ErrInvalidEvent is the typed sentinel returned by Service.RecordEvent
// when an event fails validation (e.g., empty Scenario, unknown Kind).
type ErrInvalidEvent struct {
	Field  string
	Reason string
}

func (e ErrInvalidEvent) Error() string {
	return fmt.Sprintf("invalid event: %s: %s", e.Field, e.Reason)
}

// ErrInvalidOverride is the typed sentinel returned by
// Service.RecordOverride when validation fails.
type ErrInvalidOverride struct {
	Field  string
	Reason string
}

func (e ErrInvalidOverride) Error() string {
	return fmt.Sprintf("invalid override: %s: %s", e.Field, e.Reason)
}
