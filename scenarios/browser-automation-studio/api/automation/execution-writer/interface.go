package executionwriter

import (
	"context"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
)

// ExecutionWriter normalizes and persists artifacts derived from engine output.
// Execution details are written to JSON files on disk; the database stores only index data.
//
// Artifact collection is controlled by ArtifactCollectionSettings which can be configured:
//   - Per-execution via ExecutionParameters.ArtifactConfig
//   - Globally via environment variables (BAS_ARTIFACT_*)
//   - Defaults to "full" profile (all artifacts collected)
//
// See: config/artifact_profiles.go for available profiles and settings.
type ExecutionWriter interface {
	// RecordStepOutcome persists step execution results and artifacts.
	// Respects the artifact collection settings configured for this writer.
	RecordStepOutcome(ctx context.Context, plan contracts.ExecutionPlan, outcome contracts.StepOutcome) (RecordResult, error)

	// RecordTelemetry persists real-time telemetry events (heartbeats, console logs, etc.).
	// Respects CollectTelemetry setting in artifact config.
	RecordTelemetry(ctx context.Context, plan contracts.ExecutionPlan, telemetry contracts.StepTelemetry) error

	// MarkCrash records a crash event and updates the execution index to failed status.
	MarkCrash(ctx context.Context, executionID uuid.UUID, failure contracts.StepFailure) error

	// UpdateCheckpoint persists the current execution progress for resumability.
	// stepIndex is the last successfully completed step (-1 for none).
	// totalSteps is the total number of steps in the workflow for progress calculation.
	UpdateCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, totalSteps int) error

	// SetArtifactConfig updates the artifact collection settings for this writer.
	// Call this before starting execution to configure what artifacts are collected.
	// If not called, defaults to "full" profile (all artifacts collected).
	SetArtifactConfig(cfg *config.ArtifactCollectionSettings)

	// GetArtifactConfig returns the current artifact collection settings.
	GetArtifactConfig() config.ArtifactCollectionSettings
}

// ExecutionIndexRepository captures the minimal persistence surface needed by the writer.
// This updates the database index only; detailed execution data is written to JSON files.
type ExecutionIndexRepository interface {
	GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error)
	UpdateExecution(ctx context.Context, execution *database.ExecutionIndex) error
}

// RecordResult exposes IDs generated during persistence for downstream sinks.
type RecordResult struct {
	StepID             uuid.UUID
	ArtifactIDs        []uuid.UUID
	TimelineArtifactID *uuid.UUID
}
