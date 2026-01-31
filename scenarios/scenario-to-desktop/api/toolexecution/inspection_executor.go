package toolexecution

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"scenario-to-desktop-api/shared/args"
)

// InspectionExecutor handles inspection-related tool execution.
type InspectionExecutor struct {
	buildStore       BuildStore
	preflightService PreflightService
	scenarioService  ScenarioService
	vrooliRoot       string
	logger           *slog.Logger
}

// NewInspectionExecutor creates a new InspectionExecutor.
func NewInspectionExecutor(
	buildStore BuildStore,
	preflightService PreflightService,
	scenarioService ScenarioService,
	vrooliRoot string,
	logger *slog.Logger,
) *InspectionExecutor {
	if logger == nil {
		logger = slog.Default()
	}
	return &InspectionExecutor{
		buildStore:       buildStore,
		preflightService: preflightService,
		scenarioService:  scenarioService,
		vrooliRoot:       vrooliRoot,
		logger:           logger,
	}
}

// CheckBuildStatus checks build status.
func (e *InspectionExecutor) CheckBuildStatus(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
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

// ListGeneratedWrappers lists generated wrappers.
func (e *InspectionExecutor) ListGeneratedWrappers(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	limit := args.GetInt(toolArgs, "limit", 50)

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

func (e *InspectionExecutor) scanForWrappers(limit int) []map[string]interface{} {
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

// ValidateConfiguration validates scenario configuration.
func (e *InspectionExecutor) ValidateConfiguration(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	scenarioName, err := args.RequireString(toolArgs, "scenario_name")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
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

// GetSystemPrerequisites gets system prerequisites.
func (e *InspectionExecutor) GetSystemPrerequisites(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
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
