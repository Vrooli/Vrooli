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
}

// ExecutionRepository captures the minimal persistence surface needed by the recorder.
type ExecutionRepository interface {
	CreateExecutionStep(ctx context.Context, step *database.ExecutionStep) error
	CreateExecutionArtifact(ctx context.Context, artifact *database.ExecutionArtifact) error
	CreateExecutionLog(ctx context.Context, log *database.ExecutionLog) error
}

// RecordResult exposes IDs generated during persistence for downstream sinks.
type RecordResult struct {
	StepID             uuid.UUID
	ArtifactIDs        []uuid.UUID
	TimelineArtifactID *uuid.UUID
}
