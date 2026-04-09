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

// toolHandler is a function that handles a tool execution request.
type toolHandler func(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error)

// buildToolRegistry constructs the tool name -> handler dispatch table.
func (e *ServerExecutor) buildToolRegistry() map[string]toolHandler {
	return map[string]toolHandler{
		// Pipeline tools (preferred)
		"run_pipeline":          e.pipeline.RunPipeline,
		"check_pipeline_status": e.pipeline.CheckPipelineStatus,
		"cancel_pipeline":       e.pipeline.CancelPipeline,
		"resume_pipeline":       e.pipeline.ResumePipeline,
		"list_pipelines":        e.pipeline.ListPipelines,

		// Legacy build/generation tools (deprecated, use run_pipeline instead)
		"generate_desktop_wrapper": e.legacy.GenerateDesktopWrapper,
		"build_for_platform":       e.legacy.BuildForPlatform,
		"cancel_build":             e.legacy.CancelBuild,
		"list_builds":              e.legacy.ListBuilds,

		// Signing tools
		"configure_signing":     e.signing.ConfigureSigning,
		"sign_application":      e.signing.SignApplication,
		"verify_signature":      e.signing.VerifySignature,
		"get_signing_status":    e.signing.GetSigningStatus,
		"discover_certificates": e.signing.DiscoverCertificates,

		// Inspection tools
		"check_build_status":       e.inspection.CheckBuildStatus,
		"get_pipeline_status":      e.pipeline.CheckPipelineStatus, // Legacy redirect
		"list_generated_wrappers":  e.inspection.ListGeneratedWrappers,
		"validate_configuration":   e.inspection.ValidateConfiguration,
		"get_system_prerequisites": e.inspection.GetSystemPrerequisites,
	}
}

// Execute dispatches tool execution to the appropriate domain executor.
func (e *ServerExecutor) Execute(ctx context.Context, toolName string, args map[string]interface{}) (*ExecutionResult, error) {
	e.logger.Info("executing tool", "tool", toolName)

	handler, ok := e.buildToolRegistry()[toolName]
	if !ok {
		return ErrorResult(fmt.Sprintf("unknown tool: %s", toolName), CodeUnknownTool), nil
	}
	return handler(ctx, args)
}
