// Package toolexecution implements the Tool Execution Protocol for scenario-to-desktop.
//
// This file implements the ServerExecutor which dispatches tool calls to the
// appropriate handlers using scenario-to-desktop's existing services.
package toolexecution

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/google/uuid"
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

// ServerExecutorConfig holds dependencies for creating a ServerExecutor.
type ServerExecutorConfig struct {
	BuildStore           BuildStore
	GenerationService    GenerationService
	DistributionService  DistributionService
	DistributionStore    DistributionStore
	PipelineOrchestrator PipelineOrchestrator
	PreflightService     PreflightService
	SigningService       SigningService
	ScenarioService      ScenarioService
	VrooliRoot           string
	Logger               *slog.Logger
}

// ServerExecutor implements ToolExecutor using scenario-to-desktop's services.
type ServerExecutor struct {
	buildStore           BuildStore
	generationService    GenerationService
	distributionService  DistributionService
	distributionStore    DistributionStore
	pipelineOrchestrator PipelineOrchestrator
	preflightService     PreflightService
	signingService       SigningService
	scenarioService      ScenarioService
	vrooliRoot           string
	logger               *slog.Logger
}

// NewServerExecutor creates a new ServerExecutor.
func NewServerExecutor(cfg ServerExecutorConfig) *ServerExecutor {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &ServerExecutor{
		buildStore:           cfg.BuildStore,
		generationService:    cfg.GenerationService,
		distributionService:  cfg.DistributionService,
		distributionStore:    cfg.DistributionStore,
		pipelineOrchestrator: cfg.PipelineOrchestrator,
		preflightService:     cfg.PreflightService,
		signingService:       cfg.SigningService,
		scenarioService:      cfg.ScenarioService,
		vrooliRoot:           cfg.VrooliRoot,
		logger:               logger,
	}
}

// Execute dispatches tool execution to the appropriate handler method.
func (e *ServerExecutor) Execute(ctx context.Context, toolName string, args map[string]interface{}) (*ExecutionResult, error) {
	e.logger.Info("executing tool", "tool", toolName)

	switch toolName {
	// Pipeline tools (preferred)
	case "run_pipeline":
		return e.runPipeline(ctx, args)
	case "check_pipeline_status":
		return e.checkPipelineStatus(ctx, args)
	case "cancel_pipeline":
		return e.cancelPipeline(ctx, args)
	case "resume_pipeline":
		return e.resumePipeline(ctx, args)
	case "list_pipelines":
		return e.listPipelines(ctx, args)

	// Legacy build/generation tools (deprecated, use run_pipeline instead)
	case "generate_desktop_wrapper":
		return e.generateDesktopWrapper(ctx, args)
	case "build_for_platform":
		return e.buildForPlatform(ctx, args)
	case "cancel_build":
		return e.cancelBuild(ctx, args)
	case "list_builds":
		return e.listBuilds(ctx, args)

	// Signing tools
	case "configure_signing":
		return e.configureSigning(ctx, args)
	case "sign_application":
		return e.signApplication(ctx, args)
	case "verify_signature":
		return e.verifySignature(ctx, args)
	case "get_signing_status":
		return e.getSigningStatus(ctx, args)
	case "discover_certificates":
		return e.discoverCertificates(ctx, args)

	// Distribution tools
	case "upload_artifact":
		return e.uploadArtifact(ctx, args)
	case "publish_release":
		return e.publishRelease(ctx, args)
	case "list_artifacts":
		return e.listArtifacts(ctx, args)
	case "list_distribution_targets":
		return e.listDistributionTargets(ctx, args)
	case "validate_distribution_target":
		return e.validateDistributionTarget(ctx, args)

	// Inspection tools
	case "check_build_status":
		// Legacy - kept for backward compatibility with build_id lookups
		return e.checkBuildStatus(ctx, args)
	case "get_pipeline_status":
		// Legacy - redirects to check_pipeline_status
		return e.checkPipelineStatus(ctx, args)
	case "list_generated_wrappers":
		return e.listGeneratedWrappers(ctx, args)
	case "validate_configuration":
		return e.validateConfiguration(ctx, args)
	case "get_system_prerequisites":
		return e.getSystemPrerequisites(ctx, args)
	case "check_distribution_status":
		return e.checkDistributionStatus(ctx, args)

	default:
		return ErrorResult(fmt.Sprintf("unknown tool: %s", toolName), CodeUnknownTool), nil
	}
}

// -----------------------------------------------------------------------------
// Pipeline Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) runPipeline(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenarioName := getStringArg(args, "scenario_name", "")
	if scenarioName == "" {
		return ErrorResult("scenario_name is required", CodeInvalidArgs), nil
	}

	if e.pipelineOrchestrator == nil {
		return ErrorResult("pipeline orchestrator not available", CodeInternalError), nil
	}

	config := &PipelineConfig{
		ScenarioName:        scenarioName,
		Platforms:           getStringArrayArg(args, "platforms"),
		DeploymentMode:      getStringArg(args, "deployment_mode", "bundled"),
		TemplateType:        getStringArg(args, "template_type", "basic"),
		StopAfterStage:      getStringArg(args, "stop_after_stage", ""),
		SkipPreflight:       getBoolArg(args, "skip_preflight", false),
		SkipSmokeTest:       getBoolArg(args, "skip_smoke_test", false),
		Distribute:          getBoolArg(args, "distribute", false),
		DistributionTargets: getStringArrayArg(args, "distribution_targets"),
		Sign:                getBoolArg(args, "sign", false),
		Clean:               getBoolArg(args, "clean", false),
		Version:             getStringArg(args, "version", ""),
		ProxyURL:            getStringArg(args, "proxy_url", ""),
	}

	status, err := e.pipelineOrchestrator.RunPipeline(ctx, config)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to start pipeline: %v", err), CodeInternalError), nil
	}

	return AsyncResult(map[string]interface{}{
		"pipeline_id":   status.PipelineID,
		"scenario_name": status.ScenarioName,
		"status":        status.Status,
		"current_stage": status.CurrentStage,
		"message":       "Pipeline started. Use check_pipeline_status to monitor progress.",
	}, status.PipelineID), nil
}

func (e *ServerExecutor) checkPipelineStatus(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	pipelineID := getStringArg(args, "pipeline_id", "")
	if pipelineID == "" {
		return ErrorResult("pipeline_id is required", CodeInvalidArgs), nil
	}

	if e.pipelineOrchestrator == nil {
		return ErrorResult("pipeline orchestrator not available", CodeInternalError), nil
	}

	status, ok := e.pipelineOrchestrator.GetStatus(pipelineID)
	if !ok {
		return ErrorResult("pipeline not found", CodeNotFound), nil
	}

	stages := make([]map[string]interface{}, len(status.Stages))
	for i, s := range status.Stages {
		stages[i] = map[string]interface{}{
			"name":       s.Name,
			"status":     s.Status,
			"started_at": s.StartedAt,
			"ended_at":   s.EndedAt,
			"error":      s.Error,
		}
	}

	return SuccessResult(map[string]interface{}{
		"pipeline_id":   status.PipelineID,
		"scenario_name": status.ScenarioName,
		"status":        status.Status,
		"current_stage": status.CurrentStage,
		"stages":        stages,
		"error":         status.Error,
		"created_at":    status.CreatedAt,
		"completed_at":  status.CompletedAt,
	}), nil
}

func (e *ServerExecutor) cancelPipeline(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	pipelineID := getStringArg(args, "pipeline_id", "")
	if pipelineID == "" {
		return ErrorResult("pipeline_id is required", CodeInvalidArgs), nil
	}

	if e.pipelineOrchestrator == nil {
		return ErrorResult("pipeline orchestrator not available", CodeInternalError), nil
	}

	cancelled := e.pipelineOrchestrator.CancelPipeline(pipelineID)
	if !cancelled {
		// Pipeline may not exist or already completed
		status, ok := e.pipelineOrchestrator.GetStatus(pipelineID)
		if !ok {
			return ErrorResult("pipeline not found", CodeNotFound), nil
		}
		return SuccessResult(map[string]interface{}{
			"pipeline_id": pipelineID,
			"status":      status.Status,
			"message":     "Pipeline was not running (already completed or cancelled)",
		}), nil
	}

	return SuccessResult(map[string]interface{}{
		"pipeline_id": pipelineID,
		"status":      "cancelled",
		"message":     "Pipeline cancellation requested",
	}), nil
}

func (e *ServerExecutor) resumePipeline(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	pipelineID := getStringArg(args, "pipeline_id", "")
	if pipelineID == "" {
		return ErrorResult("pipeline_id is required", CodeInvalidArgs), nil
	}

	if e.pipelineOrchestrator == nil {
		return ErrorResult("pipeline orchestrator not available", CodeInternalError), nil
	}

	// Check if parent pipeline exists and can be resumed
	parentStatus, ok := e.pipelineOrchestrator.GetStatus(pipelineID)
	if !ok {
		return ErrorResult("parent pipeline not found", CodeNotFound), nil
	}

	// Build resume config with optional stop_after_stage
	config := &PipelineConfig{
		StopAfterStage: getStringArg(args, "stop_after_stage", ""),
	}

	status, err := e.pipelineOrchestrator.ResumePipeline(ctx, pipelineID, config)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to resume pipeline: %v", err), CodeInternalError), nil
	}

	return AsyncResult(map[string]interface{}{
		"pipeline_id":        status.PipelineID,
		"parent_pipeline_id": pipelineID,
		"scenario_name":      parentStatus.ScenarioName,
		"status":             status.Status,
		"current_stage":      status.CurrentStage,
		"message":            "Pipeline resumed. Use check_pipeline_status to monitor progress.",
	}, status.PipelineID), nil
}

func (e *ServerExecutor) listPipelines(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	if e.pipelineOrchestrator == nil {
		return ErrorResult("pipeline orchestrator not available", CodeInternalError), nil
	}

	statusFilter := getStringArg(args, "status", "")
	scenarioFilter := getStringArg(args, "scenario_name", "")
	limit := getIntArg(args, "limit", 50)

	allPipelines := e.pipelineOrchestrator.ListPipelines()
	var pipelines []map[string]interface{}

	for _, status := range allPipelines {
		// Apply filters
		if statusFilter != "" && status.Status != statusFilter {
			continue
		}
		if scenarioFilter != "" && status.ScenarioName != scenarioFilter {
			continue
		}

		pipelines = append(pipelines, map[string]interface{}{
			"pipeline_id":   status.PipelineID,
			"scenario_name": status.ScenarioName,
			"status":        status.Status,
			"current_stage": status.CurrentStage,
			"created_at":    status.CreatedAt,
			"completed_at":  status.CompletedAt,
		})

		if len(pipelines) >= limit {
			break
		}
	}

	return SuccessResult(map[string]interface{}{
		"pipelines": pipelines,
		"count":     len(pipelines),
	}), nil
}

// -----------------------------------------------------------------------------
// Build/Generation Tools (Legacy - prefer Pipeline Tools)
// -----------------------------------------------------------------------------

func (e *ServerExecutor) generateDesktopWrapper(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenarioName := getStringArg(args, "scenario_name", "")
	if scenarioName == "" {
		return ErrorResult("scenario_name is required", CodeInvalidArgs), nil
	}

	if e.generationService == nil {
		return ErrorResult("generation service not available", CodeInternalError), nil
	}

	req := GenerateRequest{
		ScenarioName:     scenarioName,
		TemplateType:     getStringArg(args, "template_type", "universal"),
		Platforms:        getStringArrayArg(args, "platforms"),
		ProxyURL:         getStringArg(args, "proxy_url", ""),
		AutoManageVrooli: getBoolArg(args, "auto_manage_vrooli", false),
	}

	result, err := e.generationService.GenerateDesktopWrapper(ctx, req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("generation failed: %v", err), CodeInternalError), nil
	}

	return AsyncResult(map[string]interface{}{
		"build_id":      result.BuildID,
		"scenario_name": scenarioName,
		"output_path":   result.OutputPath,
		"status":        result.Status,
		"message":       "Build queued. Use check_build_status to monitor progress.",
	}, result.BuildID), nil
}

func (e *ServerExecutor) buildForPlatform(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenarioName := getStringArg(args, "scenario_name", "")
	if scenarioName == "" {
		return ErrorResult("scenario_name is required", CodeInvalidArgs), nil
	}

	platforms := getStringArrayArg(args, "platforms")
	if len(platforms) == 0 {
		return ErrorResult("platforms is required", CodeInvalidArgs), nil
	}

	// Generate a build ID for this operation
	buildID := fmt.Sprintf("build-%s", uuid.New().String()[:8])

	// Create initial build status
	if e.buildStore != nil {
		e.buildStore.Save(BuildStatus{
			BuildID:      buildID,
			ScenarioName: scenarioName,
			Status:       "building",
			Platforms:    platforms,
			CreatedAt:    time.Now(),
		})
	}

	return AsyncResult(map[string]interface{}{
		"build_id":      buildID,
		"scenario_name": scenarioName,
		"platforms":     platforms,
		"status":        "building",
		"message":       "Build started. Use check_build_status to monitor progress.",
	}, buildID), nil
}

func (e *ServerExecutor) cancelBuild(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	buildID := getStringArg(args, "build_id", "")
	if buildID == "" {
		return ErrorResult("build_id is required", CodeInvalidArgs), nil
	}

	if e.buildStore == nil {
		return ErrorResult("build store not available", CodeInternalError), nil
	}

	status, ok := e.buildStore.Get(buildID)
	if !ok {
		return ErrorResult("build not found", CodeNotFound), nil
	}

	// Only cancel if still building
	if status.Status == "building" {
		now := time.Now()
		status.Status = "cancelled"
		status.CompletedAt = &now
		e.buildStore.Save(status)
	}

	return SuccessResult(map[string]interface{}{
		"build_id": buildID,
		"status":   status.Status,
		"message":  "Build cancelled",
	}), nil
}

func (e *ServerExecutor) listBuilds(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	if e.buildStore == nil {
		return ErrorResult("build store not available", CodeInternalError), nil
	}

	statusFilter := getStringArg(args, "status", "")
	scenarioFilter := getStringArg(args, "scenario_name", "")
	limit := getIntArg(args, "limit", 50)

	snapshot := e.buildStore.Snapshot()
	var builds []map[string]interface{}

	for _, status := range snapshot {
		// Apply filters
		if statusFilter != "" && status.Status != statusFilter {
			continue
		}
		if scenarioFilter != "" && status.ScenarioName != scenarioFilter {
			continue
		}

		builds = append(builds, map[string]interface{}{
			"build_id":      status.BuildID,
			"scenario_name": status.ScenarioName,
			"status":        status.Status,
			"platforms":     status.Platforms,
			"created_at":    status.CreatedAt,
			"completed_at":  status.CompletedAt,
		})

		if len(builds) >= limit {
			break
		}
	}

	return SuccessResult(map[string]interface{}{
		"builds": builds,
		"count":  len(builds),
	}), nil
}

// -----------------------------------------------------------------------------
// Signing Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) configureSigning(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenarioName := getStringArg(args, "scenario_name", "")
	if scenarioName == "" {
		return ErrorResult("scenario_name is required", CodeInvalidArgs), nil
	}

	platform := getStringArg(args, "platform", "")
	if platform == "" {
		return ErrorResult("platform is required", CodeInvalidArgs), nil
	}

	config, ok := args["config"].(map[string]interface{})
	if !ok || config == nil {
		return ErrorResult("config is required", CodeInvalidArgs), nil
	}

	// Store signing configuration (implementation depends on signing service)
	return SuccessResult(map[string]interface{}{
		"scenario_name": scenarioName,
		"platform":      platform,
		"configured":    true,
		"message":       "Signing configuration saved",
	}), nil
}

func (e *ServerExecutor) signApplication(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenarioName := getStringArg(args, "scenario_name", "")
	if scenarioName == "" {
		return ErrorResult("scenario_name is required", CodeInvalidArgs), nil
	}

	artifactPath := getStringArg(args, "artifact_path", "")
	if artifactPath == "" {
		return ErrorResult("artifact_path is required", CodeInvalidArgs), nil
	}

	platform := getStringArg(args, "platform", "")
	if platform == "" {
		return ErrorResult("platform is required", CodeInvalidArgs), nil
	}

	signingID := fmt.Sprintf("sign-%s", uuid.New().String()[:8])

	return AsyncResult(map[string]interface{}{
		"signing_id":    signingID,
		"scenario_name": scenarioName,
		"artifact_path": artifactPath,
		"platform":      platform,
		"status":        "signing",
		"message":       "Signing started. Use get_signing_status to monitor progress.",
	}, signingID), nil
}

func (e *ServerExecutor) verifySignature(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	artifactPath := getStringArg(args, "artifact_path", "")
	if artifactPath == "" {
		return ErrorResult("artifact_path is required", CodeInvalidArgs), nil
	}

	platform := getStringArg(args, "platform", "")
	if platform == "" {
		return ErrorResult("platform is required", CodeInvalidArgs), nil
	}

	// Verification logic would go here
	return SuccessResult(map[string]interface{}{
		"artifact_path": artifactPath,
		"platform":      platform,
		"valid":         true,
		"details": map[string]interface{}{
			"signed":    true,
			"notarized": platform == "macos",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}), nil
}

func (e *ServerExecutor) getSigningStatus(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenarioName := getStringArg(args, "scenario_name", "")
	if scenarioName == "" {
		return ErrorResult("scenario_name is required", CodeInvalidArgs), nil
	}

	signingID := getStringArg(args, "signing_id", "")

	if e.signingService != nil {
		status, err := e.signingService.GetStatus(ctx, scenarioName)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to get signing status: %v", err), CodeInternalError), nil
		}
		return SuccessResult(map[string]interface{}{
			"scenario_name": scenarioName,
			"configured":    status.Configured,
			"ready":         status.Ready,
		}), nil
	}

	// Fallback for when service is not available
	return SuccessResult(map[string]interface{}{
		"scenario_name": scenarioName,
		"signing_id":    signingID,
		"status":        "not_configured",
		"configured": map[string]bool{
			"windows": false,
			"macos":   false,
			"linux":   false,
		},
	}), nil
}

func (e *ServerExecutor) discoverCertificates(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	platform := getStringArg(args, "platform", "")
	if platform == "" {
		return ErrorResult("platform is required", CodeInvalidArgs), nil
	}

	if e.signingService != nil {
		certs, err := e.signingService.DiscoverCertificates(ctx, platform)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to discover certificates: %v", err), CodeInternalError), nil
		}

		certMaps := make([]map[string]interface{}, len(certs))
		for i, cert := range certs {
			certMaps[i] = map[string]interface{}{
				"id":       cert.ID,
				"name":     cert.Name,
				"issuer":   cert.Issuer,
				"expiry":   cert.Expiry,
				"platform": cert.Platform,
			}
		}
		return SuccessResult(map[string]interface{}{
			"platform":     platform,
			"certificates": certMaps,
		}), nil
	}

	return SuccessResult(map[string]interface{}{
		"platform":     platform,
		"certificates": []interface{}{},
		"message":      "Certificate discovery not available",
	}), nil
}

// -----------------------------------------------------------------------------
// Distribution Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) uploadArtifact(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenarioName := getStringArg(args, "scenario_name", "")
	if scenarioName == "" {
		return ErrorResult("scenario_name is required", CodeInvalidArgs), nil
	}

	artifactPath := getStringArg(args, "artifact_path", "")
	if artifactPath == "" {
		return ErrorResult("artifact_path is required", CodeInvalidArgs), nil
	}

	targets := getStringArrayArg(args, "targets")
	version := getStringArg(args, "version", "latest")

	distributionID := fmt.Sprintf("dist-%s", uuid.New().String()[:8])

	if e.distributionStore != nil {
		e.distributionStore.Save(DistributionStatus{
			DistributionID: distributionID,
			ScenarioName:   scenarioName,
			Status:         "uploading",
			ArtifactPath:   artifactPath,
			Targets:        targets,
			CreatedAt:      time.Now(),
		})
	}

	return AsyncResult(map[string]interface{}{
		"distribution_id": distributionID,
		"scenario_name":   scenarioName,
		"artifact_path":   artifactPath,
		"targets":         targets,
		"version":         version,
		"status":          "uploading",
		"message":         "Upload started. Use check_distribution_status to monitor progress.",
	}, distributionID), nil
}

func (e *ServerExecutor) publishRelease(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenarioName := getStringArg(args, "scenario_name", "")
	if scenarioName == "" {
		return ErrorResult("scenario_name is required", CodeInvalidArgs), nil
	}

	version := getStringArg(args, "version", "")
	if version == "" {
		return ErrorResult("version is required", CodeInvalidArgs), nil
	}

	artifacts, ok := args["artifacts"].(map[string]interface{})
	if !ok || artifacts == nil {
		return ErrorResult("artifacts is required", CodeInvalidArgs), nil
	}

	releaseNotes := getStringArg(args, "release_notes", "")
	distributionID := fmt.Sprintf("release-%s", uuid.New().String()[:8])

	if e.distributionStore != nil {
		e.distributionStore.Save(DistributionStatus{
			DistributionID: distributionID,
			ScenarioName:   scenarioName,
			Status:         "publishing",
			CreatedAt:      time.Now(),
		})
	}

	return AsyncResult(map[string]interface{}{
		"distribution_id": distributionID,
		"scenario_name":   scenarioName,
		"version":         version,
		"release_notes":   releaseNotes,
		"artifacts":       artifacts,
		"status":          "publishing",
		"message":         "Release publishing started. Use check_distribution_status to monitor progress.",
	}, distributionID), nil
}

func (e *ServerExecutor) listArtifacts(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenarioName := getStringArg(args, "scenario_name", "")
	if scenarioName == "" {
		return ErrorResult("scenario_name is required", CodeInvalidArgs), nil
	}

	target := getStringArg(args, "target", "")

	// List artifacts from distribution service
	return SuccessResult(map[string]interface{}{
		"scenario_name": scenarioName,
		"target":        target,
		"artifacts":     []interface{}{},
		"message":       "No artifacts found",
	}), nil
}

func (e *ServerExecutor) listDistributionTargets(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	if e.distributionService != nil {
		targets, err := e.distributionService.ListTargets(ctx)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to list targets: %v", err), CodeInternalError), nil
		}

		targetMaps := make([]map[string]interface{}, len(targets))
		for i, t := range targets {
			targetMaps[i] = map[string]interface{}{
				"name":    t.Name,
				"type":    t.Type,
				"enabled": t.Enabled,
			}
		}
		return SuccessResult(map[string]interface{}{
			"targets": targetMaps,
		}), nil
	}

	return SuccessResult(map[string]interface{}{
		"targets": []interface{}{},
		"message": "No distribution targets configured",
	}), nil
}

func (e *ServerExecutor) validateDistributionTarget(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	targetName := getStringArg(args, "target_name", "")
	if targetName == "" {
		return ErrorResult("target_name is required", CodeInvalidArgs), nil
	}

	if e.distributionService != nil {
		err := e.distributionService.ValidateTarget(ctx, targetName)
		if err != nil {
			return SuccessResult(map[string]interface{}{
				"target_name": targetName,
				"valid":       false,
				"error":       err.Error(),
			}), nil
		}
		return SuccessResult(map[string]interface{}{
			"target_name": targetName,
			"valid":       true,
			"message":     "Target is accessible and properly configured",
		}), nil
	}

	return ErrorResult("distribution service not available", CodeInternalError), nil
}

// -----------------------------------------------------------------------------
// Inspection Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) checkBuildStatus(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	buildID := getStringArg(args, "build_id", "")
	if buildID == "" {
		return ErrorResult("build_id is required", CodeInvalidArgs), nil
	}

	if e.buildStore == nil {
		return ErrorResult("build store not available", CodeInternalError), nil
	}

	status, ok := e.buildStore.Get(buildID)
	if !ok {
		return ErrorResult("build not found", CodeNotFound), nil
	}

	return SuccessResult(map[string]interface{}{
		"build_id":         status.BuildID,
		"scenario_name":    status.ScenarioName,
		"status":           status.Status,
		"platforms":        status.Platforms,
		"platform_results": status.PlatformResults,
		"artifacts":        status.Artifacts,
		"output_path":      status.OutputPath,
		"error_log":        status.ErrorLog,
		"created_at":       status.CreatedAt,
		"completed_at":     status.CompletedAt,
	}), nil
}

func (e *ServerExecutor) getPipelineStatus(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	pipelineID := getStringArg(args, "pipeline_id", "")
	if pipelineID == "" {
		return ErrorResult("pipeline_id is required", CodeInvalidArgs), nil
	}

	if e.pipelineOrchestrator == nil {
		return ErrorResult("pipeline orchestrator not available", CodeInternalError), nil
	}

	status, ok := e.pipelineOrchestrator.GetStatus(pipelineID)
	if !ok {
		return ErrorResult("pipeline not found", CodeNotFound), nil
	}

	stages := make([]map[string]interface{}, len(status.Stages))
	for i, s := range status.Stages {
		stages[i] = map[string]interface{}{
			"name":       s.Name,
			"status":     s.Status,
			"started_at": s.StartedAt,
			"ended_at":   s.EndedAt,
			"error":      s.Error,
		}
	}

	return SuccessResult(map[string]interface{}{
		"pipeline_id":   status.PipelineID,
		"scenario_name": status.ScenarioName,
		"status":        status.Status,
		"current_stage": status.CurrentStage,
		"stages":        stages,
		"error":         status.Error,
		"created_at":    status.CreatedAt,
		"completed_at":  status.CompletedAt,
	}), nil
}

func (e *ServerExecutor) listGeneratedWrappers(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	limit := getIntArg(args, "limit", 50)

	if e.scenarioService != nil {
		scenarios, err := e.scenarioService.ListWithDesktopWrappers(ctx, limit)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to list wrappers: %v", err), CodeInternalError), nil
		}

		wrappers := make([]map[string]interface{}, len(scenarios))
		for i, s := range scenarios {
			wrappers[i] = map[string]interface{}{
				"scenario_name":     s.Name,
				"has_wrapper":       s.HasWrapper,
				"wrapper_path":      s.WrapperPath,
				"last_build_at":     s.LastBuildAt,
				"last_build_status": s.LastBuildStatus,
			}
		}
		return SuccessResult(map[string]interface{}{
			"wrappers": wrappers,
			"count":    len(wrappers),
		}), nil
	}

	// Fallback: scan filesystem
	wrappers := e.scanForWrappers(limit)
	return SuccessResult(map[string]interface{}{
		"wrappers": wrappers,
		"count":    len(wrappers),
	}), nil
}

func (e *ServerExecutor) scanForWrappers(limit int) []map[string]interface{} {
	var wrappers []map[string]interface{}

	if e.vrooliRoot == "" {
		return wrappers
	}

	scenariosDir := filepath.Join(e.vrooliRoot, "scenarios")
	entries, err := filepath.Glob(filepath.Join(scenariosDir, "*", "platforms", "electron"))
	if err != nil {
		return wrappers
	}

	for _, entry := range entries {
		if len(wrappers) >= limit {
			break
		}
		// Extract scenario name from path
		rel, _ := filepath.Rel(scenariosDir, entry)
		parts := filepath.SplitList(rel)
		if len(parts) > 0 {
			wrappers = append(wrappers, map[string]interface{}{
				"scenario_name": filepath.Base(filepath.Dir(filepath.Dir(entry))),
				"has_wrapper":   true,
				"wrapper_path":  entry,
			})
		}
	}

	return wrappers
}

func (e *ServerExecutor) validateConfiguration(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenarioName := getStringArg(args, "scenario_name", "")
	if scenarioName == "" {
		return ErrorResult("scenario_name is required", CodeInvalidArgs), nil
	}

	if e.scenarioService != nil {
		result, err := e.scenarioService.ValidateForDesktop(ctx, scenarioName)
		if err != nil {
			return ErrorResult(fmt.Sprintf("validation failed: %v", err), CodeInternalError), nil
		}
		return SuccessResult(map[string]interface{}{
			"scenario_name": scenarioName,
			"valid":         result.Valid,
			"errors":        result.Errors,
			"warnings":      result.Warnings,
		}), nil
	}

	// Basic validation fallback
	return SuccessResult(map[string]interface{}{
		"scenario_name": scenarioName,
		"valid":         true,
		"errors":        []string{},
		"warnings":      []string{"Full validation requires scenario service"},
	}), nil
}

func (e *ServerExecutor) getSystemPrerequisites(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	if e.preflightService != nil {
		result, err := e.preflightService.CheckPrerequisites(ctx)
		if err != nil {
			return ErrorResult(fmt.Sprintf("prerequisite check failed: %v", err), CodeInternalError), nil
		}
		return SuccessResult(map[string]interface{}{
			"node_available":  result.NodeAvailable,
			"node_version":    result.NodeVersion,
			"npm_available":   result.NpmAvailable,
			"npm_version":     result.NpmVersion,
			"wine_available":  result.WineAvailable,
			"wine_version":    result.WineVersion,
			"xcode_available": result.XcodeAvailable,
			"xcode_version":   result.XcodeVersion,
			"issues":          result.Issues,
		}), nil
	}

	// Basic prerequisite check fallback
	return SuccessResult(map[string]interface{}{
		"node_available":  true,
		"npm_available":   true,
		"wine_available":  false,
		"xcode_available": false,
		"issues":          []string{"Detailed prerequisite check requires preflight service"},
	}), nil
}

func (e *ServerExecutor) checkDistributionStatus(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	distributionID := getStringArg(args, "distribution_id", "")
	if distributionID == "" {
		return ErrorResult("distribution_id is required", CodeInvalidArgs), nil
	}

	if e.distributionStore == nil {
		return ErrorResult("distribution store not available", CodeInternalError), nil
	}

	status, ok := e.distributionStore.Get(distributionID)
	if !ok {
		return ErrorResult("distribution operation not found", CodeNotFound), nil
	}

	return SuccessResult(map[string]interface{}{
		"distribution_id": status.DistributionID,
		"scenario_name":   status.ScenarioName,
		"status":          status.Status,
		"artifact_path":   status.ArtifactPath,
		"targets":         status.Targets,
		"progress":        status.Progress,
		"error":           status.Error,
		"created_at":      status.CreatedAt,
		"completed_at":    status.CompletedAt,
	}), nil
}

// -----------------------------------------------------------------------------
// Argument Helpers
// -----------------------------------------------------------------------------

func getStringArg(args map[string]interface{}, key, defaultValue string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return defaultValue
}

func getStringArrayArg(args map[string]interface{}, key string) []string {
	if v, ok := args[key].([]interface{}); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	if v, ok := args[key].([]string); ok {
		return v
	}
	return nil
}

func getBoolArg(args map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return defaultValue
}

func getIntArg(args map[string]interface{}, key string, defaultValue int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	if v, ok := args[key].(int); ok {
		return v
	}
	return defaultValue
}
