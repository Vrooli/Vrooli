package recorder

import (
	"context"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

// Recorder normalizes and persists artifacts derived from engine output.
type Recorder interface {
	RecordStepOutcome(ctx context.Context, plan contracts.ExecutionPlan, outcome contracts.StepOutcome) (RecordResult, error)
	RecordTelemetry(ctx context.Context, plan contracts.ExecutionPlan, telemetry contracts.StepTelemetry) error
	MarkCrash(ctx context.Context, executionID uuid.UUID, failure contracts.StepFailure) error
}

// RecordResult exposes IDs generated during persistence for downstream sinks.
type RecordResult struct {
	StepID             uuid.UUID
	ArtifactIDs        []uuid.UUID
	TimelineArtifactID *uuid.UUID
}
