package pipeline

import (
	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/bundle"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/preflight"
	"scenario-to-desktop-api/smoketest"
)

// StageInput carries data between pipeline stages.
type StageInput struct {
	// Config is the pipeline configuration.
	Config *Config `json:"config,omitempty"`

	// PipelineID is the ID of the current pipeline run.
	PipelineID string `json:"pipeline_id,omitempty"`

	// ScenarioPath is the path to the scenario directory.
	ScenarioPath string `json:"scenario_path,omitempty"`

	// ResourceDeploymentPlan is the immutable target/resource selection made
	// before packaging. Bundle staging and the runtime consume this exact plan.
	ResourceDeploymentPlan *ResourceDeploymentPlan `json:"resource_deployment_plan,omitempty"`

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

	// DeployResult contains the output from the deploy stage.
	DeployResult *DeployResult `json:"deploy_result,omitempty"`

	// ScenarioMetadata contains analyzed scenario metadata.
	ScenarioMetadata *generation.ScenarioMetadata `json:"scenario_metadata,omitempty"`

	// Provenance captures the git state at pipeline start time.
	// Carried through stages so build metadata can include commit info.
	Provenance *BuildProvenance `json:"provenance,omitempty"`

	// Logger for stage logging. Not serialized.
	Logger Logger `json:"-"`

	// GateStateReporter is called by stages to signal approval-gate state transitions.
	// The orchestrator injects this callback so stages can update the pipeline state
	// machine (e.g. GateBlocked ↔ ExecutingStage) while blocking synchronously.
	GateStateReporter func(blocked bool) `json:"-"`
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

	// ErrorInfo contains structured error information with recovery guidance.
	// Populated when the stage fails with a DomainError.
	ErrorInfo *StageErrorInfo `json:"error_info,omitempty"`

	// Details contains stage-specific output data.
	Details interface{} `json:"details,omitempty"`

	// Logs contains log messages from stage execution.
	Logs []string `json:"logs,omitempty"`
}

// IsComplete returns true if the stage has finished executing.
func (r *StageResult) IsComplete() bool {
	return r.Status == StatusCompleted || r.Status == StatusFailed || r.Status == StatusSkipped
}

// IsSuccess returns true if the stage completed successfully.
func (r *StageResult) IsSuccess() bool {
	return r.Status == StatusCompleted || r.Status == StatusSkipped
}

// StageErrorInfo contains structured error information for stage failures.
// This mirrors DomainError fields relevant for stage-level error reporting.
type StageErrorInfo struct {
	Code          string                 `json:"code"`
	Message       string                 `json:"message"`
	Domain        string                 `json:"domain,omitempty"`
	Details       map[string]interface{} `json:"details,omitempty"`
	Recovery      string                 `json:"recovery,omitempty"`
	RecoveryHint  string                 `json:"recovery_hint,omitempty"`
	RetryStrategy *RetryStrategyInfo     `json:"retry_strategy,omitempty"`
	AutoFix       *AutoFixInfo           `json:"auto_fix,omitempty"`
	ManualSteps   []string               `json:"manual_steps,omitempty"`
	Diagnostic    *DiagnosticInfo        `json:"diagnostic,omitempty"`
}

// RetryStrategyInfo mirrors errors.RetryStrategy for JSON serialization.
type RetryStrategyInfo struct {
	MaxAttempts       int     `json:"max_attempts"`
	BackoffMs         int     `json:"backoff_ms"`
	BackoffMultiplier float64 `json:"backoff_multiplier"`
}

// AutoFixInfo mirrors errors.AutoFix for JSON serialization.
type AutoFixInfo struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Safe        bool   `json:"safe"`
}

// DiagnosticInfo mirrors errors.DiagnosticContext for JSON serialization.
type DiagnosticInfo struct {
	Process *ProcessDiagnosticInfo `json:"process,omitempty"`
	System  map[string]string      `json:"system,omitempty"`
}

// ProcessDiagnosticInfo mirrors errors.ProcessDiagnostic for JSON serialization.
type ProcessDiagnosticInfo struct {
	PID        int    `json:"pid,omitempty"`
	ExitCode   int    `json:"exit_code,omitempty"`
	RuntimeMs  int64  `json:"runtime_ms,omitempty"`
	LastOutput string `json:"last_output,omitempty"`
}

// RunRequest is the HTTP request body for starting a pipeline.
type RunRequest struct {
	Config
	VersionUpdate *VersionUpdateRequest `json:"version_update,omitempty"`
}

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
