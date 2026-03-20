package toolexecution

import (
	"context"
	"fmt"
	"log/slog"

	"scenario-to-desktop-api/shared/args"
)

// PipelineExecutor handles pipeline-related tool execution.
type PipelineExecutor struct {
	orchestrator PipelineOrchestrator
	logger       *slog.Logger
}

// NewPipelineExecutor creates a new PipelineExecutor.
func NewPipelineExecutor(orchestrator PipelineOrchestrator, logger *slog.Logger) *PipelineExecutor {
	if logger == nil {
		logger = slog.Default()
	}
	return &PipelineExecutor{
		orchestrator: orchestrator,
		logger:       logger,
	}
}

// RunPipeline starts a new pipeline.
func (e *PipelineExecutor) RunPipeline(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	scenarioName, err := args.RequireString(toolArgs, "scenario_name")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	if e.orchestrator == nil {
		return ErrorResult("pipeline orchestrator not available - server may not be fully initialized; wait and retry or check server logs", CodeInternalError), nil
	}

	config := &PipelineConfig{
		ScenarioName:   scenarioName,
		Platforms:      args.GetStringArray(toolArgs, "platforms"),
		DeploymentMode: args.GetString(toolArgs, "deployment_mode", "bundled"),
		TemplateType:   args.GetString(toolArgs, "template_type", "basic"),
		LocationMode:   args.GetString(toolArgs, "location_mode", ""),
		StopAfterStage: args.GetString(toolArgs, "stop_after_stage", ""),
		SkipPreflight:  args.GetBool(toolArgs, "skip_preflight", false),
		SkipSmokeTest:  args.GetBool(toolArgs, "skip_smoke_test", false),
		DeployTarget:   args.GetString(toolArgs, "deploy_target", ""),
		DeployTo:       args.GetString(toolArgs, "deploy_to", ""),
		RemoteProfile:  args.GetString(toolArgs, "remote_profile", ""),
		AppKey:         args.GetString(toolArgs, "app_key", ""),
		Sign:           args.GetBool(toolArgs, "sign", false),
		Clean:          args.GetBool(toolArgs, "clean", false),
		Version:        args.GetString(toolArgs, "version", ""),
		ProxyURL:       args.GetString(toolArgs, "proxy_url", ""),
	}

	status, err := e.orchestrator.RunPipeline(ctx, config)
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

// CheckPipelineStatus gets pipeline status.
func (e *PipelineExecutor) CheckPipelineStatus(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	pipelineID, err := args.RequireString(toolArgs, "pipeline_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	if e.orchestrator == nil {
		return ErrorResult("pipeline orchestrator not available - server may not be fully initialized; wait and retry or check server logs", CodeInternalError), nil
	}

	status, ok := e.orchestrator.GetStatus(pipelineID)
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

// CancelPipeline cancels a running pipeline.
func (e *PipelineExecutor) CancelPipeline(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	pipelineID, err := args.RequireString(toolArgs, "pipeline_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	if e.orchestrator == nil {
		return ErrorResult("pipeline orchestrator not available - server may not be fully initialized; wait and retry or check server logs", CodeInternalError), nil
	}

	cancelled := e.orchestrator.CancelPipeline(pipelineID)
	if !cancelled {
		// Pipeline may not exist or already completed
		status, ok := e.orchestrator.GetStatus(pipelineID)
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

// ResumePipeline resumes a stopped pipeline.
func (e *PipelineExecutor) ResumePipeline(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	pipelineID, err := args.RequireString(toolArgs, "pipeline_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	if e.orchestrator == nil {
		return ErrorResult("pipeline orchestrator not available - server may not be fully initialized; wait and retry or check server logs", CodeInternalError), nil
	}

	// Check if parent pipeline exists and can be resumed
	parentStatus, ok := e.orchestrator.GetStatus(pipelineID)
	if !ok {
		return ErrorResult("parent pipeline not found", CodeNotFound), nil
	}

	// Build resume config with optional stop_after_stage
	config := &PipelineConfig{
		StopAfterStage: args.GetString(toolArgs, "stop_after_stage", ""),
	}

	status, err := e.orchestrator.ResumePipeline(ctx, pipelineID, config)
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

// ListPipelines lists all pipelines.
func (e *PipelineExecutor) ListPipelines(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	if e.orchestrator == nil {
		return ErrorResult("pipeline orchestrator not available - server may not be fully initialized; wait and retry or check server logs", CodeInternalError), nil
	}

	statusFilter := args.GetString(toolArgs, "status", "")
	scenarioFilter := args.GetString(toolArgs, "scenario_name", "")
	limit := args.GetInt(toolArgs, "limit", 50)

	allPipelines := e.orchestrator.ListPipelines()
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
