// Package workflows owns RCL's durable record of assisted catalog work.
// Agent Manager runs are execution evidence only: direct RCL ingest/apply
// remains the sole mutation authority.
package workflows

import (
	"context"
	"errors"
	"time"
)

type Kind string

const (
	KindExtract Kind = "extract"
	KindAdopt   Kind = "adopt"
)

type Status string

const (
	StatusQueued      Status = "queued"
	StatusRunning     Status = "running"
	StatusParked      Status = "parked"
	StatusSucceeded   Status = "succeeded"
	StatusFailed      Status = "failed"
	StatusStopped     Status = "stopped"
	StatusUnavailable Status = "unavailable"
)

func (s Status) Active() bool { return s == StatusQueued || s == StatusRunning || s == StatusParked }

type Workflow struct {
	ID, AssetID, SourceScenario, TargetScenario, SourcePath, RequestedVersion string
	AgentManagerTaskID, AgentManagerRunID, IdempotencyKey                     string
	Kind                                                                      Kind
	Status                                                                    Status
	LastEventSequence                                                         int64
	Summary, Error                                                            string
	CreatedAt, UpdatedAt, CompletedAt                                         time.Time
}

type StartInput struct {
	Kind                                                                  Kind
	AssetID, SourceScenario, TargetScenario, SourcePath, RequestedVersion string
	IdempotencyKey                                                        string
	ConfirmOverwrite, OverrideValidation                                  bool
}

// PromotionReadiness is a read-only evidence report. It never treats an
// assisted-workflow terminal state as proof that catalog or origin mutation
// succeeded; those facts are read from the components/adoptions domains.
type PromotionReadiness struct {
	AssetID, LibraryID, SelectedVersion, OriginScenario string
	DependencyLibraryIDs, OriginFiles, ParityFindings   []string
	RequiredExampleCount, AvailableExampleCount         int
	ParityReportPresent, ParityWaived                   bool
	OriginReplacementPresent, OriginReplacementClean    bool
	Blockers                                            []string
	Ready                                               bool
	NextValidationCommand                               string
}

type PromotionReadinessInput struct{ AssetID, OriginScenario, Version string }

type PromotionReadinessReader interface {
	PromotionReadiness(context.Context, PromotionReadinessInput) (PromotionReadiness, error)
}

type DispatchResult struct {
	TaskID, RunID string
	Status        Status
	QueueDepth    int
}

type RunSnapshot struct {
	Status            Status
	Summary, Error    string
	LastEventSequence int64
}

type Dispatcher interface {
	Dispatch(context.Context, StartInput) (DispatchResult, error)
	Snapshot(context.Context, string, int64) (RunSnapshot, error)
	Stop(context.Context, string) (RunSnapshot, error)
}

type Repository interface {
	Create(context.Context, Workflow) (Workflow, error)
	Get(context.Context, string) (Workflow, error)
	List(context.Context, string, string, bool, int) ([]Workflow, error)
	FindActiveByIdempotency(context.Context, string) (Workflow, error)
	Update(context.Context, Workflow) (Workflow, error)
}

var ErrNotFound = errors.New("workflow not found")
