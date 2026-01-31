// Package toolexecution implements the Tool Execution Protocol for scenario-to-desktop.
package toolexecution

import (
	"context"
	"time"
)

// ToolExecutor executes tools from the Tool Execution Protocol.
type ToolExecutor interface {
	Execute(ctx context.Context, toolName string, args map[string]interface{}) (*ExecutionResult, error)
}

// BuildStore provides build status storage operations.
type BuildStore interface {
	Get(buildID string) (BuildStatus, bool)
	Save(status BuildStatus)
	Snapshot() map[string]BuildStatus
}

// BuildStatus represents a build's current state.
type BuildStatus struct {
	BuildID         string
	ScenarioName    string
	Status          string // building, ready, partial, failed
	Platforms       []string
	PlatformResults map[string]PlatformResult
	Artifacts       map[string]string
	OutputPath      string
	ErrorLog        []string
	BuildLog        []string
	CreatedAt       time.Time
	CompletedAt     *time.Time
	Metadata        map[string]interface{}
}

// PlatformResult holds build result for a single platform.
type PlatformResult struct {
	Status    string
	Artifact  string
	Error     string
	StartedAt time.Time
	EndedAt   *time.Time
}

// GenerationService provides desktop wrapper generation.
type GenerationService interface {
	GenerateDesktopWrapper(ctx context.Context, req GenerateRequest) (*GenerateResult, error)
}

// GenerateRequest holds generation parameters.
type GenerateRequest struct {
	ScenarioName     string
	TemplateType     string
	Platforms        []string
	ProxyURL         string
	AutoManageVrooli bool
}

// GenerateResult holds generation output.
type GenerateResult struct {
	BuildID    string
	OutputPath string
	Status     string
}

// DistributionService provides artifact distribution.
type DistributionService interface {
	Upload(ctx context.Context, req UploadRequest) (*UploadResult, error)
	ListTargets(ctx context.Context) ([]DistributionTarget, error)
	ValidateTarget(ctx context.Context, targetName string) error
}

// UploadRequest holds upload parameters.
type UploadRequest struct {
	ScenarioName string
	ArtifactPath string
	Targets      []string
	Version      string
}

// UploadResult holds upload output.
type UploadResult struct {
	DistributionID string
	Status         string
	UploadedTo     []string
}

// DistributionTarget represents a configured distribution target.
type DistributionTarget struct {
	Name    string
	Type    string // s3, r2, local
	Enabled bool
}

// DistributionStore tracks distribution operations.
type DistributionStore interface {
	Get(distributionID string) (DistributionStatus, bool)
	Save(status DistributionStatus)
}

// DistributionStatus represents a distribution operation's state.
type DistributionStatus struct {
	DistributionID string
	ScenarioName   string
	Status         string // pending, uploading, completed, failed
	ArtifactPath   string
	Targets        []string
	Progress       int
	Error          string
	CreatedAt      time.Time
	CompletedAt    *time.Time
}

// PipelineOrchestrator provides full pipeline orchestration.
type PipelineOrchestrator interface {
	// RunPipeline starts a pipeline and returns immediately with status.
	RunPipeline(ctx context.Context, config *PipelineConfig) (*PipelineStatus, error)

	// ResumePipeline resumes a stopped pipeline from its next stage.
	ResumePipeline(ctx context.Context, pipelineID string, config *PipelineConfig) (*PipelineStatus, error)

	// GetStatus retrieves current pipeline status.
	GetStatus(pipelineID string) (*PipelineStatus, bool)

	// CancelPipeline cancels a running pipeline.
	CancelPipeline(pipelineID string) bool

	// ListPipelines returns all tracked pipelines.
	ListPipelines() []*PipelineStatus
}

// PipelineConfig holds configuration for a pipeline run.
type PipelineConfig struct {
	ScenarioName        string
	Platforms           []string
	DeploymentMode      string
	TemplateType        string
	StopAfterStage      string
	SkipPreflight       bool
	SkipSmokeTest       bool
	Distribute          bool
	DistributionTargets []string
	Sign                bool
	Clean               bool
	Version             string
	ProxyURL            string
}

// PipelineStatus represents a pipeline's state.
type PipelineStatus struct {
	PipelineID   string
	ScenarioName string
	Status       string
	CurrentStage string
	Stages       []StageStatus
	Error        string
	CreatedAt    time.Time
	CompletedAt  *time.Time
}

// StageStatus represents a single pipeline stage.
type StageStatus struct {
	Name      string
	Status    string
	StartedAt *time.Time
	EndedAt   *time.Time
	Error     string
}

// PreflightService provides system prerequisite checking.
type PreflightService interface {
	CheckPrerequisites(ctx context.Context) (*PrerequisitesResult, error)
}

// PrerequisitesResult holds prerequisite check results.
type PrerequisitesResult struct {
	NodeAvailable  bool
	NodeVersion    string
	NpmAvailable   bool
	NpmVersion     string
	WineAvailable  bool
	WineVersion    string
	XcodeAvailable bool
	XcodeVersion   string
	Issues         []string
}

// SigningService provides code signing operations.
type SigningService interface {
	GetStatus(ctx context.Context, scenarioName string) (*SigningStatus, error)
	DiscoverCertificates(ctx context.Context, platform string) ([]Certificate, error)
}

// SigningStatus represents signing configuration state.
type SigningStatus struct {
	ScenarioName string
	Configured   map[string]bool // platform -> configured
	Ready        map[string]bool // platform -> ready to sign
}

// Certificate represents a discovered signing certificate.
type Certificate struct {
	ID       string
	Name     string
	Issuer   string
	Expiry   time.Time
	Platform string
}

// ScenarioService provides scenario information.
type ScenarioService interface {
	ListWithDesktopWrappers(ctx context.Context, limit int) ([]ScenarioInfo, error)
	ValidateForDesktop(ctx context.Context, scenarioName string) (*ValidationResult, error)
}

// ScenarioInfo holds scenario metadata.
type ScenarioInfo struct {
	Name            string
	HasWrapper      bool
	WrapperPath     string
	LastBuildAt     *time.Time
	LastBuildStatus string
}

// ValidationResult holds validation output.
type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}
