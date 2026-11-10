package destinations

import (
	"context"
	"time"

	"data-backup-manager/internal/failures"
)

// PlanRecord is the part of a backup plan the failure monitor reads.
type PlanRecord struct {
	ID             string
	Name           string
	Enabled        bool
	Schedule       string
	TargetIDs      []string
	DestinationIDs []string
	CreatedAt      time.Time
}

// DestinationFailure is one destination-scoped reason a run did not complete.
type DestinationFailure struct {
	DestinationID string
	Code          string
}

// RunRecord is the newest run of a plan, reduced to what diagnosis needs.
type RunRecord struct {
	ID         string
	Status     string
	StartedAt  time.Time
	FinishedAt time.Time
	// Failures are the destination-level incidents and blocked/failed outcomes.
	Failures []DestinationFailure
	// Terminal reports whether the run has finished.
	Terminal bool
}

// FailsDestination reports whether the run was stopped by a destination-level
// reason that this path can diagnose.
func (r RunRecord) FailsDestination(destinationID string) bool {
	for _, f := range r.Failures {
		if f.DestinationID == destinationID && diagnosableDestinationCode(f.Code) {
			return true
		}
	}
	return false
}

// diagnosableDestinationCode lists the failure codes this path diagnoses:
// reasons that live on the destination (drive or repository), not in a source,
// a credential or the storage cap.
func diagnosableDestinationCode(code string) bool {
	switch failures.Code(code) {
	case failures.RepositoryInvalid, failures.DestinationInaccessible, failures.DestinationDirty,
		failures.DestinationReadOnly, failures.DestinationUnmounted:
		return true
	default:
		return false
	}
}

// PlanSuccess is the recorded success history of one plan.
type PlanSuccess struct {
	At       time.Time
	ByTarget map[string]time.Time
}

// TargetLabel names a target the way snapshots are labelled.
type TargetLabel struct{ Owner, Name string }

func (l TargetLabel) String() string { return l.Owner + "/" + l.Name }

// BackupHistory is DBM's own backup history and the one action the monitor may
// take on it: asking for a run. Everything else here is read-only.
type BackupHistory interface {
	Plans(ctx context.Context) ([]PlanRecord, error)
	LatestRun(ctx context.Context, planID string) (RunRecord, bool, error)
	PlanSuccess(ctx context.Context, planID string) (PlanSuccess, error)
	TargetLabels(ctx context.Context, ids []string) (map[string]TargetLabel, error)
	// ActiveRun reports whether any run that writes to the destination is in
	// flight, so a repair never takes the drive away from a running backup.
	ActiveRun(ctx context.Context, destinationID string) (bool, error)
	// TriggerRun asks for a run of a plan now. A run already active is not an
	// error for the caller.
	TriggerRun(ctx context.Context, planID string) error
}
