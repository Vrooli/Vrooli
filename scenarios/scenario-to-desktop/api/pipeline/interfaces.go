// Package pipeline provides a unified orchestrator for running the complete
// scenario-to-desktop deployment pipeline. It coordinates bundle packaging,
// preflight validation, wrapper generation, electron building, and smoke testing.
package pipeline

import (
	"context"
)

// Stage represents a single phase of the pipeline.
// Each stage is independently executable and can be skipped based on configuration.
type Stage interface {
	// Name returns the unique identifier for this stage.
	Name() string

	// Execute runs the stage with the given input and returns a result.
	// The context can be used for cancellation.
	Execute(ctx context.Context, input *StageInput) *StageResult

	// CanSkip checks if this stage can be skipped given the current input.
	// Some stages (like preflight) are skippable, others (like generate) are not.
	CanSkip(input *StageInput) bool

	// Dependencies returns the names of stages that must complete before this one.
	Dependencies() []string
}

// Orchestrator coordinates the execution of pipeline stages.
type Orchestrator interface {
	// RunPipeline starts a complete pipeline execution.
	// Returns immediately with a pipeline ID; poll GetStatus for progress.
	RunPipeline(ctx context.Context, config *Config) (*Status, error)

	// ResumePipeline resumes a stopped pipeline from its next stage.
	// The parent pipeline must have been stopped with StopAfterStage.
	// Returns a new pipeline that continues from where the parent stopped.
	ResumePipeline(ctx context.Context, pipelineID string, config *Config) (*Status, error)

	// GetStatus retrieves the current status of a pipeline run.
	GetStatus(pipelineID string) (*Status, bool)

	// CancelPipeline cancels a running pipeline.
	// Returns true if the pipeline was found and cancellation was signaled.
	CancelPipeline(pipelineID string) bool

	// ListPipelines returns all tracked pipeline runs.
	ListPipelines() []*Status
}

// Store persists pipeline run states.
type Store interface {
	// Save creates or updates a pipeline status.
	Save(status *Status)

	// Get retrieves a pipeline status by ID.
	Get(pipelineID string) (*Status, bool)

	// Update updates a pipeline status using a modifier function.
	Update(pipelineID string, fn func(status *Status)) bool

	// UpdateStage updates a specific stage's result within a pipeline.
	UpdateStage(pipelineID, stageName string, result *StageResult) bool

	// Delete removes a pipeline status.
	Delete(pipelineID string) bool

	// List returns all pipeline statuses.
	List() []*Status

	// Cleanup removes completed pipelines older than the given duration.
	Cleanup(olderThan int64)
}

// CancelManager manages cancellation functions for running pipelines.
type CancelManager interface {
	// Set registers a cancellation function for a pipeline.
	Set(pipelineID string, cancel context.CancelFunc)

	// Take retrieves and removes the cancellation function.
	// Returns nil if no cancellation function is registered.
	Take(pipelineID string) context.CancelFunc

	// Clear removes a cancellation function without calling it.
	Clear(pipelineID string)
}

// IDGenerator generates unique identifiers for pipelines.
type IDGenerator interface {
	// Generate returns a new unique pipeline ID.
	Generate() string
}

// Logger provides structured logging for the pipeline.
type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// TimeProvider abstracts time for deterministic testing.
type TimeProvider interface {
	// Now returns the current Unix timestamp in seconds.
	Now() int64
}

// WebhookNotifier sends webhook notifications about pipeline events.
type WebhookNotifier interface {
	// Notify sends a webhook notification about a pipeline event.
	// The event can be "started", "stage_completed", "completed", "failed", or "cancelled".
	Notify(ctx context.Context, webhookURL string, event string, status *Status) error
}

// ManifestGenerator creates bundle manifests via deployment-manager integration.
// This allows the BundleStage to generate manifests on-demand if they don't exist.
type ManifestGenerator interface {
	// GenerateManifest creates a bundle manifest for the given scenario.
	// Returns the path to the generated manifest, or an error if generation failed.
	GenerateManifest(ctx context.Context, scenarioName, outputDir string) (string, error)
}
