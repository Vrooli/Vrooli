package recorder

import (
	"context"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
)

// Recorder normalizes and persists artifacts derived from engine output.
type Recorder interface {
	RecordStepOutcome(ctx context.Context, plan contracts.ExecutionPlan, outcome contracts.StepOutcome) (RecordResult, error)
	RecordTelemetry(ctx context.Context, plan contracts.ExecutionPlan, telemetry contracts.StepTelemetry) error
	MarkCrash(ctx context.Context, executionID uuid.UUID, failure contracts.StepFailure) error
	// UpdateCheckpoint persists the current execution progress for resumability.
	// stepIndex is the last successfully completed step (-1 for none).
	// totalSteps is the total number of steps in the workflow for progress calculation.
	UpdateCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, totalSteps int) error
}

// ExecutionRepository captures the minimal persistence surface needed by the recorder.
type ExecutionRepository interface {
	CreateExecutionStep(ctx context.Context, step *database.ExecutionStep) error
	CreateExecutionArtifact(ctx context.Context, artifact *database.ExecutionArtifact) error
	CreateExecutionLog(ctx context.Context, log *database.ExecutionLog) error
	UpdateExecutionCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, progress int) error
}

// RecordResult exposes IDs generated during persistence for downstream sinks.
type RecordResult struct {
	StepID             uuid.UUID
	ArtifactIDs        []uuid.UUID
	TimelineArtifactID *uuid.UUID
}
