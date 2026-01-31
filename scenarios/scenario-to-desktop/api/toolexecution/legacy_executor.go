package toolexecution

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"scenario-to-desktop-api/shared/args"
)

// LegacyExecutor handles legacy/deprecated tool execution.
// These tools are kept for backward compatibility but should be migrated to pipeline tools.
type LegacyExecutor struct {
	buildStore        BuildStore
	generationService GenerationService
	logger            *slog.Logger
}

// NewLegacyExecutor creates a new LegacyExecutor.
func NewLegacyExecutor(buildStore BuildStore, generationService GenerationService, logger *slog.Logger) *LegacyExecutor {
	if logger == nil {
		logger = slog.Default()
	}
	return &LegacyExecutor{
		buildStore:        buildStore,
		generationService: generationService,
		logger:            logger,
	}
}

// GenerateDesktopWrapper generates a desktop wrapper (deprecated - use run_pipeline).
func (e *LegacyExecutor) GenerateDesktopWrapper(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	scenarioName, err := args.RequireString(toolArgs, "scenario_name")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	if e.generationService == nil {
		return ErrorResult("generation service not available - server may not be fully initialized; wait and retry or check server logs", CodeInternalError), nil
	}

	req := GenerateRequest{
		ScenarioName:     scenarioName,
		TemplateType:     args.GetString(toolArgs, "template_type", "universal"),
		Platforms:        args.GetStringArray(toolArgs, "platforms"),
		ProxyURL:         args.GetString(toolArgs, "proxy_url", ""),
		AutoManageVrooli: args.GetBool(toolArgs, "auto_manage_vrooli", false),
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

// BuildForPlatform builds for a platform (deprecated - use run_pipeline).
func (e *LegacyExecutor) BuildForPlatform(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	scenarioName, err := args.RequireString(toolArgs, "scenario_name")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	platforms, err := args.RequireStringArray(toolArgs, "platforms")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
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

// CancelBuild cancels a build (deprecated - use cancel_pipeline).
func (e *LegacyExecutor) CancelBuild(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	buildID, err := args.RequireString(toolArgs, "build_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	if e.buildStore == nil {
		return ErrorResult("build store not available - server may not be fully initialized; wait and retry or check server logs", CodeInternalError), nil
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

// ListBuilds lists builds (deprecated - use list_pipelines).
func (e *LegacyExecutor) ListBuilds(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	if e.buildStore == nil {
		return ErrorResult("build store not available - server may not be fully initialized; wait and retry or check server logs", CodeInternalError), nil
	}

	statusFilter := args.GetString(toolArgs, "status", "")
	scenarioFilter := args.GetString(toolArgs, "scenario_name", "")
	limit := args.GetInt(toolArgs, "limit", 50)

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
