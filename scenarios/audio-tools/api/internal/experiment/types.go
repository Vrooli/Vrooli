// Package experiment persists async STT experiment runs: reproducible recipes,
// lifecycle state, per-condition metrics, and large report references. Audio
// bytes and large reports stay in the blob store, not SQLite or git.
package experiment

import (
	"context"
	"fmt"
	"time"
)

// Status is an experiment's lifecycle state.
type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

// Terminal reports whether s is a final state.
func (s Status) Terminal() bool {
	switch s {
	case StatusSucceeded, StatusFailed, StatusCanceled:
		return true
	default:
		return false
	}
}

// Experiment is the persisted, observable record of one submitted experiment.
type Experiment struct {
	ID          string
	Name        string
	Status      Status
	RecipeJSON  []byte
	CreatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
	Error       string
	ResultRef   string
	MachineJSON []byte
}

// Run is one strategy x condition metrics cell within an experiment report.
type Run struct {
	ID            string
	ExperimentID  string
	Strategy      string
	ConditionJSON []byte
	CreatedAt     time.Time
}

// QualificationEvidence is a transcript-free proof record for one
// provider/strategy/policy promotion gate. The store keeps failures as well as
// passes so an operator cannot erase a failed qualification by recording a
// later success without the history remaining inspectable.
type QualificationEvidence struct {
	ID            string
	EngineID      string
	ModelID       string
	Strategy      string
	PolicyProfile string
	Kind          string
	FaultProfile  string
	Passed        bool
	ArtifactRef   string
	Notes         string
	MachineJSON   []byte
	ObservedAt    time.Time
}

// QualificationEvidenceFilter narrows evidence by its provider-cell identity.
type QualificationEvidenceFilter struct {
	EngineID      string
	ModelID       string
	Strategy      string
	PolicyProfile string
}

// ListFilter narrows experiment listing.
type ListFilter struct {
	Status Status
	Limit  int
	Offset int
}

// SubmitSpec describes one experiment to enqueue.
type SubmitSpec struct {
	Name             string
	RecipeJSON       []byte
	MachineJSON      []byte
	EstimatedSeconds int
	// MaxRuntime bounds server-owned execution. It is intentionally separate
	// from a caller's wait context: experiments survive a disconnected CLI, but
	// they must not consume the single worker indefinitely.
	MaxRuntime time.Duration
}

// RunResult is the artifact payload returned by a Runner after successful
// experiment compute.
type RunResult struct {
	Report     []byte
	ReportMIME string
	Runs       []Run
	RecipeJSON []byte
}

// ProgressEvent is one in-memory lifecycle update for StreamExperimentEvents.
// The durable source of truth remains the Experiment row.
type ProgressEvent struct {
	ExperimentID string
	Status       Status
	Progress     int
	Message      string
	At           time.Time
}

// Runner executes an experiment under a server-lifetime context. The context is
// canceled only by Manager.Cancel or Manager.Close, never by a waiting client
// disconnect.
type Runner func(ctx context.Context, exp Experiment, emit func(progress int, message string)) (RunResult, error)

// ErrNotFound is returned when an experiment id has no row.
type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("experiment: %q not found", e.ID) }

// Repository is the experiment metadata persistence seam. Production is the
// SQLite impl in sqlite.go; tests substitute fakes.
type Repository interface {
	CreateExperiment(ctx context.Context, exp Experiment) (Experiment, error)
	GetExperiment(ctx context.Context, id string) (Experiment, error)
	UpdateExperiment(ctx context.Context, exp Experiment) error
	CompleteSucceeded(ctx context.Context, exp Experiment, runs []Run) error
	ListExperiments(ctx context.Context, filter ListFilter) ([]Experiment, error)
	ListNonTerminal(ctx context.Context) ([]Experiment, error)
	DeleteExperiment(ctx context.Context, id string) error
	CreateRun(ctx context.Context, run Run) (Run, error)
	ListRuns(ctx context.Context, experimentID string) ([]Run, error)
	CreateQualificationEvidence(ctx context.Context, evidence QualificationEvidence) (QualificationEvidence, error)
	ListQualificationEvidence(ctx context.Context, filter QualificationEvidenceFilter) ([]QualificationEvidence, error)
}
