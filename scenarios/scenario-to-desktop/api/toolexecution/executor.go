// DOC: docs/reference/api-architecture.md#tool-execution-system

// Package toolexecution implements the Tool Execution Protocol for scenario-to-desktop.
//
// This file implements the ServerExecutor which dispatches tool calls to the
// appropriate domain executors using scenario-to-desktop's existing services.
package toolexecution

import (
	"context"
	"fmt"
	"log/slog"
)

// ServerExecutorConfig holds dependencies for creating a ServerExecutor.
type ServerExecutorConfig struct {
	BuildStore           BuildStore
	GenerationService    GenerationService
	PipelineOrchestrator PipelineOrchestrator
	PreflightService     PreflightService
	SigningService       SigningService
	ScenarioService      ScenarioService
	VrooliRoot           string
	Logger               *slog.Logger
}

// ServerExecutor implements ToolExecutor using scenario-to-desktop's services.
// It dispatches to domain-specific executors for each tool category.
type ServerExecutor struct {
	pipeline   *PipelineExecutor
	signing    *SigningExecutor
	inspection *InspectionExecutor
	legacy     *LegacyExecutor
	logger     *slog.Logger
}

// NewServerExecutor creates a new ServerExecutor.
func NewServerExecutor(cfg ServerExecutorConfig) *ServerExecutor {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &ServerExecutor{
		pipeline:   NewPipelineExecutor(cfg.PipelineOrchestrator, logger),
		signing:    NewSigningExecutor(cfg.SigningService, logger),
		inspection: NewInspectionExecutor(cfg.BuildStore, cfg.PreflightService, cfg.ScenarioService, cfg.VrooliRoot, logger),
		legacy:     NewLegacyExecutor(cfg.BuildStore, cfg.GenerationService, logger),
		logger:     logger,
	}
}

// Execute dispatches tool execution to the appropriate domain executor.
func (e *ServerExecutor) Execute(ctx context.Context, toolName string, args map[string]interface{}) (*ExecutionResult, error) {
	e.logger.Info("executing tool", "tool", toolName)

	switch toolName {
	// Pipeline tools (preferred)
	case "run_pipeline":
		return e.pipeline.RunPipeline(ctx, args)
	case "check_pipeline_status":
		return e.pipeline.CheckPipelineStatus(ctx, args)
	case "cancel_pipeline":
		return e.pipeline.CancelPipeline(ctx, args)
	case "resume_pipeline":
		return e.pipeline.ResumePipeline(ctx, args)
	case "list_pipelines":
		return e.pipeline.ListPipelines(ctx, args)

	// Legacy build/generation tools (deprecated, use run_pipeline instead)
	case "generate_desktop_wrapper":
		return e.legacy.GenerateDesktopWrapper(ctx, args)
	case "build_for_platform":
		return e.legacy.BuildForPlatform(ctx, args)
	case "cancel_build":
		return e.legacy.CancelBuild(ctx, args)
	case "list_builds":
		return e.legacy.ListBuilds(ctx, args)

	// Signing tools
	case "configure_signing":
		return e.signing.ConfigureSigning(ctx, args)
	case "sign_application":
		return e.signing.SignApplication(ctx, args)
	case "verify_signature":
		return e.signing.VerifySignature(ctx, args)
	case "get_signing_status":
		return e.signing.GetSigningStatus(ctx, args)
	case "discover_certificates":
		return e.signing.DiscoverCertificates(ctx, args)

	// Inspection tools
	case "check_build_status":
		return e.inspection.CheckBuildStatus(ctx, args)
	case "get_pipeline_status":
		// Legacy - redirects to check_pipeline_status
		return e.pipeline.CheckPipelineStatus(ctx, args)
	case "list_generated_wrappers":
		return e.inspection.ListGeneratedWrappers(ctx, args)
	case "validate_configuration":
		return e.inspection.ValidateConfiguration(ctx, args)
	case "get_system_prerequisites":
		return e.inspection.GetSystemPrerequisites(ctx, args)

	default:
		return ErrorResult(fmt.Sprintf("unknown tool: %s", toolName), CodeUnknownTool), nil
	}
}
