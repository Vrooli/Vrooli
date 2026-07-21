package pipeline

import (
	"fmt"
	"time"

	"scenario-to-desktop-api/generation"
)

// Stage names as constants for consistency.
const (
	StageResolveDeployment = "resolve-deployment"
	StageBundle            = "bundle"
	StagePreflight         = "preflight"
	StageGenerate          = "generate"
	StageBuild             = "build"
	StageSmokeTest         = "smoketest"
	StageDeploy            = "deploy"
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

// PipelineState represents the current fine-grained phase of pipeline execution.
// This provides more granular observability than the high-level Status field.
type PipelineState string

const (
	// PipelineStateCreated indicates the pipeline has been created but not started.
	PipelineStateCreated PipelineState = "created"
	// PipelineStateInitializing indicates the pipeline is initializing resources.
	PipelineStateInitializing PipelineState = "initializing"
	// PipelineStateQueueingStage indicates the pipeline is preparing to execute a stage.
	PipelineStateQueueingStage PipelineState = "queueing_stage"
	// PipelineStateExecutingStage indicates a stage is actively executing.
	PipelineStateExecutingStage PipelineState = "executing_stage"
	// PipelineStateGateBlocked indicates the pipeline is waiting for an approval gate to clear.
	PipelineStateGateBlocked PipelineState = "gate_blocked"
	// PipelineStatePollingCompletion indicates the pipeline is polling for async completion.
	PipelineStatePollingCompletion PipelineState = "polling_completion"
	// PipelineStateProcessingResult indicates the pipeline is processing stage results.
	PipelineStateProcessingResult PipelineState = "processing_result"
	// PipelineStateCompleted indicates the pipeline finished successfully.
	PipelineStateCompleted PipelineState = "completed"
	// PipelineStateFailed indicates the pipeline encountered an error.
	PipelineStateFailed PipelineState = "failed"
	// PipelineStateCancelled indicates the pipeline was cancelled.
	PipelineStateCancelled PipelineState = "cancelled"
)

// ValidPipelineStateTransitions defines all valid state transitions in the pipeline state machine.
var ValidPipelineStateTransitions = map[PipelineState][]PipelineState{
	"": { // Initial empty state
		PipelineStateCreated,
	},
	PipelineStateCreated: {
		PipelineStateInitializing,
		PipelineStateCancelled,
	},
	PipelineStateInitializing: {
		PipelineStateQueueingStage,
		PipelineStateFailed,
		PipelineStateCancelled,
	},
	PipelineStateQueueingStage: {
		PipelineStateExecutingStage,
		PipelineStateCompleted, // If no more stages
		PipelineStateFailed,
		PipelineStateCancelled,
	},
	PipelineStateExecutingStage: {
		PipelineStateGateBlocked,
		PipelineStatePollingCompletion,
		PipelineStateProcessingResult,
		PipelineStateFailed,
		PipelineStateCancelled,
	},
	PipelineStateGateBlocked: {
		PipelineStateExecutingStage,
		PipelineStateFailed,
		PipelineStateCancelled,
	},
	PipelineStatePollingCompletion: {
		PipelineStateProcessingResult,
		PipelineStateFailed,
		PipelineStateCancelled,
	},
	PipelineStateProcessingResult: {
		PipelineStateQueueingStage, // Next stage
		PipelineStateCompleted,
		PipelineStateFailed,
		PipelineStateCancelled,
	},
	PipelineStateCompleted: {}, // Terminal state
	PipelineStateFailed:    {}, // Terminal state
	PipelineStateCancelled: {}, // Terminal state
}

// CanTransitionTo checks if transitioning from this state to the target is valid.
func (s PipelineState) CanTransitionTo(target PipelineState) bool {
	validTargets, ok := ValidPipelineStateTransitions[s]
	if !ok {
		return false
	}
	for _, valid := range validTargets {
		if valid == target {
			return true
		}
	}
	return false
}

// IsTerminal returns true if this is a terminal state (no valid outgoing transitions).
func (s PipelineState) IsTerminal() bool {
	transitions, ok := ValidPipelineStateTransitions[s]
	return ok && len(transitions) == 0
}

// PipelineStateTransition records a state change for debugging and observability.
type PipelineStateTransition struct {
	From       PipelineState `json:"from"`
	To         PipelineState `json:"to"`
	Timestamp  time.Time     `json:"timestamp"`
	Stage      string        `json:"stage,omitempty"`       // Current stage when transition occurred
	Message    string        `json:"message,omitempty"`     // Optional context message
	DurationMs int64         `json:"duration_ms,omitempty"` // Time spent in From state (ms)
}

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

	// LocationMode controls where the desktop output is written.
	// Options: proper (default), staging/temp (write to the scenario-to-desktop cache staging root), custom (requires output_path).
	LocationMode string `json:"location_mode,omitempty"`

	// WebhookURL is an optional URL for webhook notifications.
	WebhookURL string `json:"webhook_url,omitempty"`

	// ProxyURL is required when DeploymentMode is "proxy".
	ProxyURL string `json:"proxy_url,omitempty"`

	// BundleManifestPath overrides the default manifest path.
	BundleManifestPath string `json:"bundle_manifest_path,omitempty"`

	// ResourceArtifactRoot is a verified Vrooli release directory containing
	// resource artifacts plus SHA256SUMS. Bundled resource modes refuse to
	// package from source when this is absent.
	ResourceArtifactRoot string `json:"resource_artifact_root,omitempty"`

	// Clean forces a clean build (removes existing desktop output).
	Clean bool `json:"clean,omitempty"`

	// Sign enables code signing during the build stage.
	Sign bool `json:"sign,omitempty"`

	// Publish enables publishing after successful build.
	Publish bool `json:"publish,omitempty"`

	// DeployConfig configures the deploy stage (LPBS deployment).
	// If nil, the deploy stage is skipped.
	DeployConfig *DeployConfig `json:"deploy,omitempty"`

	// Version is the release version (used in deploy path).
	Version string `json:"version,omitempty"`

	// versionRollback stores persisted version changes for rollback on failure.
	versionRollback *versionRollback `json:"-"`

	// PreflightTimeoutSeconds sets the timeout for preflight validation.
	PreflightTimeoutSeconds int `json:"preflight_timeout_seconds,omitempty"`

	// PreflightSecrets provides secrets for preflight validation.
	PreflightSecrets map[string]string `json:"preflight_secrets,omitempty"`

	// StopAfterStage halts the pipeline after this stage completes.
	// Empty string means run all stages. Valid values: bundle, preflight, generate, build, smoketest, deploy.
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

	// Stages specifies which stages to run. Empty means all stages.
	// Valid values: bundle, preflight, generate, build, smoketest, deploy.
	// Stages are executed in pipeline order, regardless of the order specified here.
	Stages []string `json:"stages,omitempty"`

	// UpdateConfig configures auto-update settings for the desktop application.
	// If nil, the default provider (generic) is used but auto-updates are disabled
	// until generic.url is configured.
	UpdateConfig *generation.UpdateConfig `json:"update_config,omitempty"`
}

// DeployConfig configures the deploy stage for LPBS deployment.
type DeployConfig struct {
	// TargetName is the saved deploy target key from deploy-targets.json.
	TargetName string `json:"target_name,omitempty"`

	// ScenarioName is the LPBS scenario name (for inline config).
	ScenarioName string `json:"scenario_name,omitempty"`

	// RemoteProfile is the remote profile tag (for inline config).
	RemoteProfile string `json:"remote_profile,omitempty"`

	// AppKey is the download app key on the remote LPBS (always required).
	AppKey string `json:"app_key"`

	// UpdateURL is auto-derived from the remote profile if empty.
	UpdateURL string `json:"update_url,omitempty"`

	// ReleaseID is the deployment-manager release UUID for traceability.
	// When set, stored on the LPBS artifact for correlation.
	ReleaseID string `json:"release_id,omitempty"`

	// Channel is the update channel (e.g. "stable", "beta", "nightly").
	// Maps to variant_key on the LPBS asset. Defaults to "stable" if empty.
	Channel string `json:"channel,omitempty"`

	// DeploymentManagerProfileID is the deployment-manager profile to check for approval gates.
	// If empty, the deploy stage skips gate checks.
	DeploymentManagerProfileID string `json:"deployment_manager_profile_id,omitempty"`

	// GateTimeout overrides DefaultGateTimeout for how long to wait for gates to clear.
	GateTimeout string `json:"gate_timeout,omitempty"`

	// GatePollInterval overrides DefaultGatePollInterval for initial poll spacing.
	GatePollInterval string `json:"gate_poll_interval,omitempty"`
}

// DeployResult contains the outcome of the deploy stage.
type DeployResult struct {
	// Artifacts lists the uploaded artifacts.
	Artifacts []DeployArtifactResult `json:"artifacts,omitempty"`

	// UpdateURL is the derived update endpoint for electron-updater.
	UpdateURL string `json:"update_url,omitempty"`
}

// DeployArtifactResult tracks a single artifact upload.
type DeployArtifactResult struct {
	ArtifactID int64  `json:"artifact_id"`
	Platform   string `json:"platform"`
}

// VersionUpdateRequest controls how a scenario version is resolved for a pipeline run.
// When provided, the API can validate, optionally persist, and apply a version update
// before the pipeline starts.
type VersionUpdateRequest struct {
	// Mode controls how the version is determined: "set" or "bump".
	Mode string `json:"mode,omitempty"`

	// Version is the explicit version to set when Mode is "set".
	Version string `json:"version,omitempty"`

	// Bump is the semver component to increment when Mode is "bump": patch, minor, medium, major.
	// "auto" is accepted as a patch bump alias.
	Bump string `json:"bump,omitempty"`

	// Persist controls whether the version is written to scenario files.
	Persist bool `json:"persist,omitempty"`

	// AllowDowngrade allows setting a version lower than the current scenario version.
	AllowDowngrade bool `json:"allow_downgrade,omitempty"`

	// Source controls which files are updated when Persist is true: both, service, ui.
	Source string `json:"source,omitempty"`
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

	// CurrentState is the fine-grained pipeline state for observability.
	CurrentState PipelineState `json:"current_state,omitempty"`

	// Transitions records the state change history for debugging.
	Transitions []PipelineStateTransition `json:"transitions,omitempty"`

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

	// Provenance captures the git state at pipeline start time.
	// Used by deployment-manager for approval gating and build-to-commit traceability.
	Provenance *BuildProvenance `json:"provenance,omitempty"`

	// lastStateChange tracks when the current state was entered (for duration calculation).
	lastStateChange time.Time
}

// TransitionTo transitions the pipeline to a new state with validation.
// Returns true if the transition was valid and applied, false otherwise.
// If the transition is invalid, the state remains unchanged.
func (s *Status) TransitionTo(target PipelineState, message string) bool {
	if !s.CurrentState.CanTransitionTo(target) && s.CurrentState != "" {
		// Invalid transition - log but don't panic
		return false
	}

	now := time.Now()
	var durationMs int64
	if !s.lastStateChange.IsZero() {
		durationMs = now.Sub(s.lastStateChange).Milliseconds()
	}

	transition := PipelineStateTransition{
		From:       s.CurrentState,
		To:         target,
		Timestamp:  now,
		Stage:      s.CurrentStage,
		Message:    message,
		DurationMs: durationMs,
	}

	s.Transitions = append(s.Transitions, transition)
	s.CurrentState = target
	s.lastStateChange = now

	return true
}

// GetStopOnFailure returns the stop_on_failure setting with default true.
func (c *Config) GetStopOnFailure() bool {
	if c.StopOnFailure == nil {
		return true
	}
	return *c.StopOnFailure
}

// GetDeploymentMode returns the deployment mode with default "bundled".
//
// IMPORTANT: "bundled" is the default and MUST remain so. Bundled mode creates
// a fully self-contained desktop application that:
//   - Works offline without any external dependencies
//   - Includes all UI assets, API binaries, and runtime
//   - Is the most common deployment mode for production desktop apps
//
// Other modes (external-server, cloud-api, proxy) are thin-client modes that
// require a running server. These should be explicitly requested when needed.
func (c *Config) GetDeploymentMode() string {
	if c.DeploymentMode == "" {
		return DeploymentModeBundled
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

// GetStages returns the stages to run, or nil for all stages.
func (c *Config) GetStages() []string {
	if c == nil || len(c.Stages) == 0 {
		return nil
	}
	return c.Stages
}

func (c *Config) setVersionRollback(rollback *versionRollback) {
	if c == nil {
		return
	}
	c.versionRollback = rollback
}

func (c *Config) takeVersionRollback() *versionRollback {
	if c == nil {
		return nil
	}
	rollback := c.versionRollback
	c.versionRollback = nil
	return rollback
}

// IsValidStageName checks if a stage name is valid.
func IsValidStageName(name string) bool {
	switch name {
	case StageResolveDeployment, StageBundle, StagePreflight, StageGenerate, StageBuild, StageSmokeTest, StageDeploy:
		return true
	default:
		return false
	}
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
		if s.CurrentState == PipelineStateGateBlocked {
			return fmt.Sprintf("Waiting for approval gate (%s stage)", s.CurrentStage)
		}
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
	stageOrder := []string{StageResolveDeployment, StageBundle, StagePreflight, StageGenerate, StageBuild, StageSmokeTest, StageDeploy}

	// Find the stopped stage and return the next one
	for i, stage := range stageOrder {
		if stage == s.StoppedAfterStage && i+1 < len(stageOrder) {
			return stageOrder[i+1]
		}
	}
	return ""
}
