package runs

import (
	"context"
	"time"

	"data-backup-manager/internal/sources"
)

// This file declares the narrow seams the runs orchestration depends on. Each
// is owned here (the consumer) and satisfied by a thin adapter over a sibling
// service, wired in main.go. Keeping them as runs-local interfaces means the
// run fan-out, partial-failure, and cap-block logic is unit-testable against
// fakes without standing up the whole domain tree, and avoids importing the
// sibling domain packages (no import cycles).

// PlanForRun is the slice of a plan the runner needs to execute it.
type PlanForRun struct {
	ID             string
	TargetIDs      []string
	DestinationIDs []string
	KeepLatest     int
}

// PlanLookup resolves a plan to its executable spec.
//
// seam: implemented by an adapter over plans.Service in main.go.
type PlanLookup interface {
	PlanForRun(ctx context.Context, planID string) (PlanForRun, error)
}

// TargetForRun is the slice of a target the runner needs to capture it. Owner
// and Name are carried (in addition to ID) so the run can stamp self-identifying
// snapshot metadata (override-source, description, tags) without a second lookup.
type TargetForRun struct {
	ID      string
	Owner   string
	Name    string
	Kind    sources.SourceKind
	Locator string
}

// TargetLookup resolves a target id to its capture spec.
//
// seam: implemented by an adapter over targets.Service in main.go.
type TargetLookup interface {
	TargetForRun(ctx context.Context, targetID string) (TargetForRun, error)
}

// ActiveTargetLookup lists target ids currently present in the live catalog.
//
// seam: implemented by an adapter over targets.Service in main.go. It lets the
// status rollup exclude orphaned/deleted historical run outcomes by default.
type ActiveTargetLookup interface {
	ActiveTargetIDs(ctx context.Context) ([]string, error)
}

// DestinationForRun is the slice of a destination the runner needs to snapshot
// into it. Name is the kopia repository name.
type DestinationForRun struct {
	ID   string
	Name string
}

// DestinationLookup resolves a destination and enforces its storage cap.
//
// seam: implemented by an adapter over destinations.Service in main.go.
type DestinationLookup interface {
	DestinationForRun(ctx context.Context, destinationID string) (DestinationForRun, error)
	// WouldBlock reports whether writing pendingBytes to the destination would
	// exceed its cap under an alert+block policy (never evicts).
	WouldBlock(ctx context.Context, destinationID string, pendingBytes int64) (blocked bool, reason string, err error)
}

// NextScheduleSource reports the next scheduled backup time per target id,
// derived from the scheduler's fire history joined with each plan's schedule.
// A target absent from the map is not on any active schedule — manual-only,
// disabled, or not yet fired since startup (the scheduler's fire history is
// in-memory and reset by a restart).
//
// seam: implemented in main.go by an adapter over the scheduler + plans; the
// runs service stays unaware of either concrete type. Optional — when nil,
// ListTargetStatus leaves next_scheduled_at unset because no scheduler seam is
// available.
type NextScheduleSource interface {
	NextScheduledByTarget(ctx context.Context) (map[string]time.Time, error)
}

// RunOutcomeEvent is the backup-outcome event emitted for platform monitoring
// (infra-health / system-monitor) after each run closes.
type RunOutcomeEvent struct {
	RunID     string
	PlanID    string
	Status    RunStatus
	Succeeded int
	Failed    int
	Blocked   int
}

// EventSink receives backup-outcome events.
//
// seam: production wires a log-backed sink in main.go; tests assert emission.
type EventSink interface {
	BackupOutcome(ctx context.Context, ev RunOutcomeEvent)
}
