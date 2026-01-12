package pipeline

import (
	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/bundle"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/preflight"
	"scenario-to-desktop-api/smoketest"
)

// Stage names as constants for consistency.
const (
	StageBundle    = "bundle"
	StagePreflight = "preflight"
	StageGenerate  = "generate"
	StageBuild     = "build"
	StageSmokeTest = "smoketest"
)

// Pipeline status values.
const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
	StatusSkipped   = "skipped"
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

	// PreflightTimeoutSeconds sets the timeout for preflight validation.
	PreflightTimeoutSeconds int `json:"preflight_timeout_seconds,omitempty"`

	// PreflightSecrets provides secrets for preflight validation.
	PreflightSecrets map[string]string `json:"preflight_secrets,omitempty"`
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
}

// StageInput carries data between pipeline stages.
type StageInput struct {
	// Config is the pipeline configuration.
	Config *Config

	// PipelineID is the ID of the current pipeline run.
	PipelineID string

	// ScenarioPath is the path to the scenario directory.
	ScenarioPath string

	// DesktopPath is the path to the generated desktop wrapper.
	DesktopPath string

	// BundleResult contains the output from the bundle stage.
	BundleResult *bundle.PackageResult

	// PreflightResult contains the output from the preflight stage.
	PreflightResult *preflight.Response

	// GenerationResult contains the output from the generation stage.
	GenerationResult *generation.GenerateResponse

	// BuildResult contains the output from the build stage.
	BuildResult *build.Status

	// SmokeTestResult contains the output from the smoke test stage.
	SmokeTestResult *smoketest.Status

	// ScenarioMetadata contains analyzed scenario metadata.
	ScenarioMetadata *generation.ScenarioMetadata

	// Logger for stage logging.
	Logger Logger
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

// IsComplete returns true if the pipeline has finished executing.
func (s *Status) IsComplete() bool {
	return s.Status == StatusCompleted || s.Status == StatusFailed || s.Status == StatusCancelled
}
