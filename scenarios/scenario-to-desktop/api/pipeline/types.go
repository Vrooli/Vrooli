package pipeline

import (
	"fmt"

	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/bundle"
	"scenario-to-desktop-api/distribution"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/preflight"
	"scenario-to-desktop-api/smoketest"
)

// Stage names as constants for consistency.
const (
	StageBundle       = "bundle"
	StagePreflight    = "preflight"
	StageGenerate     = "generate"
	StageBuild        = "build"
	StageDistribution = "distribution"
	StageSmokeTest    = "smoketest"
)

// Pipeline status values.
const (
	StatusIdle      = "idle"      // Pipeline created but not yet started
	StatusPending   = "pending"   // Pipeline queued for execution
	StatusRunning   = "running"   // Pipeline actively executing stages
	StatusCompleted = "completed" // Pipeline finished successfully
	StatusFailed    = "failed"    // Pipeline encountered an error
	StatusCancelled = "cancelled" // Pipeline was manually cancelled
	StatusSkipped   = "skipped"   // Stage/pipeline was skipped
)

// Config represents the configuration for a pipeline run.
type Config struct {
	// ScenarioName is the name of the scenario to deploy (required).
	ScenarioName string `json:"scenario_name" validate:"required"`

	// Platforms to build for. Defaults to current platform if empty.
	Platforms []string `json:"platforms,omitempty"`

	// SkipPreflight skips the preflight validation stage.
	SkipPreflight bool `json:"skip_preflight,omitempty"`

	// SkipSmokeTest skips the smoke test stage.
	SkipSmokeTest bool `json:"skip_smoke_test,omitempty"`

	// StopOnFailure stops the pipeline if any stage fails. Default: true.
	StopOnFailure *bool `json:"stop_on_failure,omitempty"`

	// DeploymentMode is "proxy" or "bundled". Default: "bundled".
	DeploymentMode string `json:"deployment_mode,omitempty"`

	// Framework is the target desktop framework. Default: "electron".
	Framework string `json:"framework,omitempty"`

	// TemplateType is the Electron template type. Default: "basic".
	TemplateType string `json:"template_type,omitempty"`

	// WebhookURL is an optional URL for webhook notifications.
	WebhookURL string `json:"webhook_url,omitempty"`

	// ProxyURL is required when DeploymentMode is "proxy".
	ProxyURL string `json:"proxy_url,omitempty"`

	// BundleManifestPath overrides the default manifest path.
	BundleManifestPath string `json:"bundle_manifest_path,omitempty"`

	// Clean forces a clean build (removes existing desktop output).
	Clean bool `json:"clean,omitempty"`

	// Sign enables code signing during the build stage.
	Sign bool `json:"sign,omitempty"`

	// Publish enables publishing after successful build.
	Publish bool `json:"publish,omitempty"`

	// Distribute enables distribution stage after successful build.
	Distribute bool `json:"distribute,omitempty"`

	// DistributionTargets specifies which distribution targets to upload to.
	// Empty means all enabled targets.
	DistributionTargets []string `json:"distribution_targets,omitempty"`

	// Version is the release version (used in distribution path).
	Version string `json:"version,omitempty"`

	// PreflightTimeoutSeconds sets the timeout for preflight validation.
	PreflightTimeoutSeconds int `json:"preflight_timeout_seconds,omitempty"`

	// PreflightSecrets provides secrets for preflight validation.
	PreflightSecrets map[string]string `json:"preflight_secrets,omitempty"`

	// StopAfterStage halts the pipeline after this stage completes.
	// Empty string means run all stages. Valid values: bundle, preflight, generate, build, smoketest, distribution.
	StopAfterStage string `json:"stop_after_stage,omitempty"`

	// ResumeFromStage starts execution from this stage, skipping all prior stages.
	// Requires that the pipeline was previously stopped with StopAfterStage.
	// The prior stages' results must be available from the parent pipeline.
	ResumeFromStage string `json:"resume_from_stage,omitempty"`

	// ParentPipelineID links this pipeline to a parent when resuming.
	// Set automatically when resuming a pipeline.
	ParentPipelineID string `json:"parent_pipeline_id,omitempty"`

	// IdempotencyKey is an optional client-provided key for request deduplication.
	// If a pipeline with the same idempotency key already exists and is running or completed,
	// the existing pipeline status will be returned instead of starting a new pipeline.
	// This enables safe retries where "running twice is no worse than running once".
	// If not provided, a new pipeline is always started (default behavior).
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// Status represents the current state of a pipeline run.
type Status struct {
	// PipelineID is the unique identifier for this pipeline run.
	PipelineID string `json:"pipeline_id"`

	// ScenarioName is the scenario being deployed.
	ScenarioName string `json:"scenario_name"`

	// Status is the overall pipeline status.
	Status string `json:"status"`

	// CurrentStage is the name of the currently executing stage.
	CurrentStage string `json:"current_stage,omitempty"`

	// ProgressPercent is the completion percentage (0-100).
	// Calculated from completed stages / total stages.
	// Always present in the response for quick status checks by agents and UIs.
	ProgressPercent int `json:"progress_percent"`

	// ProgressMessage is a human-readable summary of current progress.
	// Example: "Running bundle stage (1/6)", "Completed successfully"
	ProgressMessage string `json:"progress_message,omitempty"`

	// Stages contains the results of each completed or running stage.
	Stages map[string]*StageResult `json:"stages"`

	// StageOrder defines the execution order of stages.
	StageOrder []string `json:"stage_order"`

	// Config is the configuration used for this pipeline run.
	Config *Config `json:"config"`

	// StartedAt is the Unix timestamp when the pipeline started.
	StartedAt int64 `json:"started_at"`

	// CompletedAt is the Unix timestamp when the pipeline completed.
	CompletedAt int64 `json:"completed_at,omitempty"`

	// Error contains the error message if the pipeline failed.
	Error string `json:"error,omitempty"`

	// FinalArtifacts contains paths to final build artifacts.
	FinalArtifacts map[string]string `json:"final_artifacts,omitempty"`

	// StoppedAfterStage indicates the pipeline was intentionally stopped after this stage.
	// Empty if the pipeline completed all stages or failed.
	StoppedAfterStage string `json:"stopped_after_stage,omitempty"`

	// ParentPipelineID links this pipeline to a parent when it was resumed.
	ParentPipelineID string `json:"parent_pipeline_id,omitempty"`

	// ResumedInput contains the stage input carried forward from a parent pipeline.
	// Used to restore state when resuming. Persisted to enable resumption after server restart.
	ResumedInput *StageInput `json:"resumed_input,omitempty"`

	// IdempotencyKey is the client-provided key for request deduplication.
	// Stored on the status to enable lookup of existing pipelines by idempotency key.
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// StageInput carries data between pipeline stages.
type StageInput struct {
	// Config is the pipeline configuration.
	Config *Config `json:"config,omitempty"`

	// PipelineID is the ID of the current pipeline run.
	PipelineID string `json:"pipeline_id,omitempty"`

	// ScenarioPath is the path to the scenario directory.
	ScenarioPath string `json:"scenario_path,omitempty"`

	// DesktopPath is the path to the generated desktop wrapper.
	DesktopPath string `json:"desktop_path,omitempty"`

	// BundleResult contains the output from the bundle stage.
	BundleResult *bundle.PackageResult `json:"bundle_result,omitempty"`

	// PreflightResult contains the output from the preflight stage.
	PreflightResult *preflight.Response `json:"preflight_result,omitempty"`

	// GenerationResult contains the output from the generation stage.
	GenerationResult *generation.GenerateResponse `json:"generation_result,omitempty"`

	// BuildResult contains the output from the build stage.
	BuildResult *build.Status `json:"build_result,omitempty"`

	// SmokeTestResult contains the output from the smoke test stage.
	SmokeTestResult *smoketest.Status `json:"smoke_test_result,omitempty"`

	// DistributionResult contains the output from the distribution stage.
	DistributionResult *distribution.DistributionStatus `json:"distribution_result,omitempty"`

	// ScenarioMetadata contains analyzed scenario metadata.
	ScenarioMetadata *generation.ScenarioMetadata `json:"scenario_metadata,omitempty"`

	// Logger for stage logging. Not serialized.
	Logger Logger `json:"-"`
}

// StageResult represents the outcome of executing a pipeline stage.
type StageResult struct {
	// Stage is the name of the stage.
	Stage string `json:"stage"`

	// Status is the stage's execution status.
	Status string `json:"status"`

	// StartedAt is the Unix timestamp when the stage started.
	StartedAt int64 `json:"started_at"`

	// CompletedAt is the Unix timestamp when the stage completed.
	CompletedAt int64 `json:"completed_at,omitempty"`

	// Error contains the error message if the stage failed.
	Error string `json:"error,omitempty"`

	// Details contains stage-specific output data.
	Details interface{} `json:"details,omitempty"`

	// Logs contains log messages from stage execution.
	Logs []string `json:"logs,omitempty"`
}

// RunRequest is the HTTP request body for starting a pipeline.
type RunRequest = Config

// RunResponse is the HTTP response for starting a pipeline.
type RunResponse struct {
	PipelineID string `json:"pipeline_id"`
	StatusURL  string `json:"status_url"`
	Message    string `json:"message,omitempty"`
}

// CancelResponse is the HTTP response for cancelling a pipeline.
type CancelResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// ResumeResponse is the HTTP response for resuming a pipeline.
type ResumeResponse struct {
	PipelineID       string `json:"pipeline_id"`
	ParentPipelineID string `json:"parent_pipeline_id"`
	StatusURL        string `json:"status_url"`
	ResumeFromStage  string `json:"resume_from_stage"`
	Message          string `json:"message,omitempty"`
}

// ListResponse is the HTTP response for listing pipelines.
type ListResponse struct {
	Pipelines []*Status `json:"pipelines"`
}

// GetStopOnFailure returns the stop_on_failure setting with default true.
func (c *Config) GetStopOnFailure() bool {
	if c.StopOnFailure == nil {
		return true
	}
	return *c.StopOnFailure
}

// GetDeploymentMode returns the deployment mode with default "bundled".
func (c *Config) GetDeploymentMode() string {
	if c.DeploymentMode == "" {
		return "bundled"
	}
	return c.DeploymentMode
}

// GetTemplateType returns the template type with default "basic".
func (c *Config) GetTemplateType() string {
	if c.TemplateType == "" {
		return "basic"
	}
	return c.TemplateType
}

// GetStopAfterStage returns the stop_after_stage setting.
func (c *Config) GetStopAfterStage() string {
	return c.StopAfterStage
}

// GetResumeFromStage returns the resume_from_stage setting.
func (c *Config) GetResumeFromStage() string {
	return c.ResumeFromStage
}

// IsValidStageName checks if a stage name is valid.
func IsValidStageName(name string) bool {
	switch name {
	case StageBundle, StagePreflight, StageGenerate, StageBuild, StageSmokeTest, StageDistribution:
		return true
	default:
		return false
	}
}

// IsComplete returns true if the stage has finished executing.
func (r *StageResult) IsComplete() bool {
	return r.Status == StatusCompleted || r.Status == StatusFailed || r.Status == StatusSkipped
}

// IsSuccess returns true if the stage completed successfully.
func (r *StageResult) IsSuccess() bool {
	return r.Status == StatusCompleted || r.Status == StatusSkipped
}

// Progress returns the pipeline's progress as a fraction (0.0 to 1.0).
func (s *Status) Progress() float64 {
	if len(s.StageOrder) == 0 {
		return 0
	}
	completed := 0
	for _, stageName := range s.StageOrder {
		if result, ok := s.Stages[stageName]; ok && result.IsComplete() {
			completed++
		}
	}
	return float64(completed) / float64(len(s.StageOrder))
}

// ComputeProgressPercent calculates the integer progress percentage (0-100).
func (s *Status) ComputeProgressPercent() int {
	return int(s.Progress() * 100)
}

// ComputeProgressMessage generates a human-readable progress message.
// Returns messages like "Running bundle stage (1/6)", "Completed successfully", etc.
func (s *Status) ComputeProgressMessage() string {
	total := len(s.StageOrder)
	if total == 0 {
		return "Initializing..."
	}

	switch s.Status {
	case StatusPending:
		return "Pipeline pending"
	case StatusCompleted:
		if s.StoppedAfterStage != "" {
			return fmt.Sprintf("Stopped after %s stage", s.StoppedAfterStage)
		}
		return "Pipeline completed successfully"
	case StatusFailed:
		if s.CurrentStage != "" {
			return fmt.Sprintf("Failed at %s stage", s.CurrentStage)
		}
		return "Pipeline failed"
	case StatusCancelled:
		return "Pipeline cancelled"
	case StatusRunning:
		if s.CurrentStage == "" {
			return "Starting pipeline..."
		}
		// Count completed stages
		completed := 0
		for _, stageName := range s.StageOrder {
			if result, ok := s.Stages[stageName]; ok && result.IsComplete() {
				completed++
			}
		}
		// Find current stage index (1-based for display)
		currentIdx := completed + 1
		return fmt.Sprintf("Running %s stage (%d/%d)", s.CurrentStage, currentIdx, total)
	default:
		return "Unknown status"
	}
}

// UpdateProgress recalculates and sets the ProgressPercent and ProgressMessage fields.
// Call this after any change to the pipeline status or stages.
func (s *Status) UpdateProgress() {
	s.ProgressPercent = s.ComputeProgressPercent()
	s.ProgressMessage = s.ComputeProgressMessage()
}

// IsComplete returns true if the pipeline has finished executing.
func (s *Status) IsComplete() bool {
	return s.Status == StatusCompleted || s.Status == StatusFailed || s.Status == StatusCancelled
}

// IsIdle returns true if the pipeline is in idle state (created but not started).
func (s *Status) IsIdle() bool {
	return s.Status == StatusIdle
}

// IsStartable returns true if the pipeline can be started (is idle or pending).
func (s *Status) IsStartable() bool {
	return s.Status == StatusIdle
}

// CanResume returns true if this pipeline can be resumed from a later stage.
// A pipeline can be resumed if it completed successfully after being stopped at a stage.
func (s *Status) CanResume() bool {
	return s.Status == StatusCompleted && s.StoppedAfterStage != ""
}

// GetNextResumeStage returns the stage that should be resumed from after the stopped stage.
// Returns empty string if the pipeline cannot be resumed or was stopped at the last stage.
func (s *Status) GetNextResumeStage() string {
	if !s.CanResume() {
		return ""
	}

	// Define stage order
	stageOrder := []string{StageBundle, StagePreflight, StageGenerate, StageBuild, StageSmokeTest, StageDistribution}

	// Find the stopped stage and return the next one
	for i, stage := range stageOrder {
		if stage == s.StoppedAfterStage && i+1 < len(stageOrder) {
			return stageOrder[i+1]
		}
	}
	return ""
}
