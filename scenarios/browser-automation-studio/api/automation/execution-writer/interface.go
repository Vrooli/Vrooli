package executionwriter

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/config"
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

	// RecordCheckpoint commits the completed cursor and actual mutable store.
	// This private recovery state is independent of artifact collection settings.
	RecordCheckpoint(ctx context.Context, checkpoint Checkpoint) error

	// RecordExecutionArtifacts persists execution-level artifacts (such as video/trace files).
	RecordExecutionArtifacts(ctx context.Context, plan contracts.ExecutionPlan, artifacts []ExternalArtifact) error

	// SetArtifactConfig updates the writer-wide artifact collection settings.
	// This is the fallback used when no per-execution config is set; prefer
	// SetArtifactConfigForExecution for concurrency-safe per-run configuration.
	SetArtifactConfig(cfg *config.ArtifactCollectionSettings)

	// GetArtifactConfig returns the writer-wide artifact collection settings.
	GetArtifactConfig() config.ArtifactCollectionSettings

	// SetArtifactConfigForExecution scopes artifact collection settings to a
	// single execution so concurrent executions sharing this recorder cannot
	// leak configuration. Pass nil to clear the override for that execution.
	SetArtifactConfigForExecution(executionID uuid.UUID, cfg *config.ArtifactCollectionSettings)

	// ForgetExecution releases outcome, timeline and configuration accumulators
	// after all execution writes finish. Persisted artifacts remain available.
	// Callers must join their writers before invoking this terminal operation.
	ForgetExecution(executionID uuid.UUID)
}

// ExternalArtifact captures an execution-level artifact to persist.
type ExternalArtifact struct {
	ArtifactType string
	Label        string
	Path         string
	ContentType  string
	Payload      map[string]any
}

// ExecutionIndexRepository captures the minimal persistence surface needed by the writer.
// This updates the database index only; detailed execution data is written to JSON files.
type ExecutionIndexRepository interface {
	UpdateExecutionResultPath(ctx context.Context, id uuid.UUID, resultPath string, updatedAt time.Time) error
}

// RecordResult exposes IDs generated during persistence for downstream sinks.
type RecordResult struct {
	StepID             uuid.UUID
	ArtifactIDs        []uuid.UUID
	TimelineArtifactID *uuid.UUID
}
