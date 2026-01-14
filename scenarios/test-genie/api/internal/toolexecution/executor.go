// Package toolexecution implements the Tool Execution Protocol for test-genie.
//
// This package provides the server-side implementation of tool execution,
// dispatching tool calls to the appropriate test-genie services.
package toolexecution

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"test-genie/internal/execution"
	"test-genie/internal/fix"
	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/requirementsimprove"
	"test-genie/internal/scenarios"
)

// ExecutionHistory provides access to execution history.
type ExecutionHistory interface {
	Get(ctx context.Context, id uuid.UUID) (*orchestrator.SuiteExecutionResult, error)
	List(ctx context.Context, scenario string, limit int, offset int) ([]orchestrator.SuiteExecutionResult, error)
}

// SuiteExecutor executes test suites.
type SuiteExecutor interface {
	Execute(ctx context.Context, input execution.SuiteExecutionInput) (*orchestrator.SuiteExecutionResult, error)
}

// ScenarioDirectory provides access to scenario information.
type ScenarioDirectory interface {
	ListSummaries(ctx context.Context) ([]scenarios.ScenarioSummary, error)
	GetSummary(ctx context.Context, name string) (*scenarios.ScenarioSummary, error)
	RunScenarioTests(ctx context.Context, name string, preferred string, extraArgs []string) (*scenarios.TestingCommand, *scenarios.TestingRunnerResult, error)
	RunUISmoke(ctx context.Context, name string, uiURL string, browserlessURL string, timeoutMs int64) (*scenarios.UISmokeResult, error)
	ListFiles(ctx context.Context, name string, opts scenarios.FileListOptions) ([]scenarios.FileNode, error)
	ScenarioRoot() string
}

// PhaseCatalog provides access to test phases.
type PhaseCatalog interface {
	DescribePhases() []phases.Descriptor
}

// FixService provides fix operations.
type FixService interface {
	Spawn(ctx context.Context, req fix.SpawnRequest) (*fix.SpawnResult, error)
	Get(id string) (*fix.Record, bool)
	ListByScenario(scenarioName string, limit int) []*fix.Record
	GetActiveForScenario(scenarioName string) *fix.Record
	Stop(ctx context.Context, id string) error
}

// RequirementsImproveService provides requirements improvement operations.
type RequirementsImproveService interface {
	Spawn(ctx context.Context, req requirementsimprove.SpawnRequest) (*requirementsimprove.SpawnResult, error)
	Get(id string) (*requirementsimprove.Record, bool)
	ListByScenario(scenarioName string, limit int) []*requirementsimprove.Record
	GetActiveForScenario(scenarioName string) *requirementsimprove.Record
	Stop(ctx context.Context, id string) error
}

// RequirementsSyncer syncs requirements.
type RequirementsSyncer interface {
	Sync(ctx context.Context, scenarioDir string) error
}

// ServerExecutorConfig holds dependencies for the ServerExecutor.
type ServerExecutorConfig struct {
	ExecutionHistory    ExecutionHistory
	SuiteExecutor       SuiteExecutor
	ScenarioDirectory   ScenarioDirectory
	PhaseCatalog        PhaseCatalog
	FixService          FixService
	RequirementsImprove RequirementsImproveService
	RequirementsSyncer  RequirementsSyncer
}

// ServerExecutor implements tool execution using test-genie services.
type ServerExecutor struct {
	executionHistory    ExecutionHistory
	suiteExecutor       SuiteExecutor
	scenarioDirectory   ScenarioDirectory
	phaseCatalog        PhaseCatalog
	fixService          FixService
	requirementsImprove RequirementsImproveService
	requirementsSyncer  RequirementsSyncer
}

// NewServerExecutor creates a new ServerExecutor with the given configuration.
func NewServerExecutor(cfg ServerExecutorConfig) *ServerExecutor {
	return &ServerExecutor{
		executionHistory:    cfg.ExecutionHistory,
		suiteExecutor:       cfg.SuiteExecutor,
		scenarioDirectory:   cfg.ScenarioDirectory,
		phaseCatalog:        cfg.PhaseCatalog,
		fixService:          cfg.FixService,
		requirementsImprove: cfg.RequirementsImprove,
		requirementsSyncer:  cfg.RequirementsSyncer,
	}
}

// Execute dispatches tool execution to the appropriate handler.
func (e *ServerExecutor) Execute(ctx context.Context, toolName string, args map[string]interface{}) (*ExecutionResult, error) {
	switch toolName {
	// Testing tools
	case "run_test_suite":
		return e.runTestSuite(ctx, args)
	case "run_scenario_tests":
		return e.runScenarioTests(ctx, args)
	case "run_ui_smoke":
		return e.runUISmoke(ctx, args)
	case "get_execution":
		return e.getExecution(ctx, args)
	case "list_executions":
		return e.listExecutions(ctx, args)

	// Scenario tools
	case "list_scenarios":
		return e.listScenarios(ctx, args)
	case "get_scenario":
		return e.getScenario(ctx, args)
	case "list_scenario_files":
		return e.listScenarioFiles(ctx, args)
	case "list_phases":
		return e.listPhases(ctx, args)

	// Fix tools
	case "spawn_fix":
		return e.spawnFix(ctx, args)
	case "get_fix":
		return e.getFix(ctx, args)
	case "list_fixes":
		return e.listFixes(ctx, args)
	case "get_active_fix":
		return e.getActiveFix(ctx, args)
	case "stop_fix":
		return e.stopFix(ctx, args)

	// Requirements tools
	case "improve_requirements":
		return e.improveRequirements(ctx, args)
	case "get_requirements_improve":
		return e.getRequirementsImprove(ctx, args)
	case "list_requirements_improves":
		return e.listRequirementsImproves(ctx, args)
	case "get_active_requirements_improve":
		return e.getActiveRequirementsImprove(ctx, args)
	case "stop_requirements_improve":
		return e.stopRequirementsImprove(ctx, args)
	case "sync_requirements":
		return e.syncRequirements(ctx, args)

	default:
		return ErrorResult("unknown tool: "+toolName, CodeUnknownTool), nil
	}
}

// -----------------------------------------------------------------------------
// Testing Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) runTestSuite(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenario := getStringArg(args, "scenario", "")
	if scenario == "" {
		return ErrorResult("scenario is required", CodeInvalidArgs), nil
	}

	phases := getStringSliceArg(args, "phases")

	input := execution.SuiteExecutionInput{
		Request: orchestrator.SuiteExecutionRequest{
			ScenarioName: scenario,
			Phases:       phases,
		},
	}

	result, err := e.suiteExecutor.Execute(ctx, input)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to execute test suite: %v", err), CodeInternalError), nil
	}

	status := "running"
	if result.Success {
		status = "passed"
	}

	return AsyncResult(map[string]interface{}{
		"execution_id": result.ExecutionID.String(),
		"scenario":     scenario,
		"status":       status,
		"started_at":   result.StartedAt,
	}, result.ExecutionID.String()), nil
}

func (e *ServerExecutor) runScenarioTests(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenario := getStringArg(args, "scenario", "")
	if scenario == "" {
		return ErrorResult("scenario is required", CodeInvalidArgs), nil
	}

	preferredRunner := getStringArg(args, "preferred_runner", "")
	extraArgs := getStringSliceArg(args, "extra_args")

	cmd, result, err := e.scenarioDirectory.RunScenarioTests(ctx, scenario, preferredRunner, extraArgs)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to run scenario tests: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"scenario": scenario,
		"command":  cmd,
		"result":   result,
	}), nil
}

func (e *ServerExecutor) runUISmoke(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenario := getStringArg(args, "scenario", "")
	if scenario == "" {
		return ErrorResult("scenario is required", CodeInvalidArgs), nil
	}

	uiURL := getStringArg(args, "ui_url", "")
	browserlessURL := getStringArg(args, "browserless_url", "")
	timeoutMs := int64(getIntArg(args, "timeout_ms", 30000))

	result, err := e.scenarioDirectory.RunUISmoke(ctx, scenario, uiURL, browserlessURL, timeoutMs)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to run UI smoke test: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"scenario": scenario,
		"result":   result,
	}), nil
}

func (e *ServerExecutor) getExecution(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	executionID := getStringArg(args, "execution_id", "")
	if executionID == "" {
		return ErrorResult("execution_id is required", CodeInvalidArgs), nil
	}

	id, err := uuid.Parse(executionID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid execution_id: %v", err), CodeInvalidArgs), nil
	}

	result, err := e.executionHistory.Get(ctx, id)
	if err != nil {
		return ErrorResult(fmt.Sprintf("execution not found: %s", executionID), CodeNotFound), nil
	}

	status := "failed"
	if result.Success {
		status = "passed"
	}

	return SuccessResult(map[string]interface{}{
		"id":           result.ExecutionID.String(),
		"scenario":     result.ScenarioName,
		"status":       status,
		"success":      result.Success,
		"started_at":   result.StartedAt,
		"completed_at": result.CompletedAt,
		"phases":       result.Phases,
	}), nil
}

func (e *ServerExecutor) listExecutions(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	limit := getIntArg(args, "limit", 20)
	scenario := getStringArg(args, "scenario", "")

	results, err := e.executionHistory.List(ctx, scenario, limit, 0)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to list executions: %v", err), CodeInternalError), nil
	}

	executions := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		status := "failed"
		if r.Success {
			status = "passed"
		}
		executions = append(executions, map[string]interface{}{
			"id":           r.ExecutionID.String(),
			"scenario":     r.ScenarioName,
			"status":       status,
			"success":      r.Success,
			"started_at":   r.StartedAt,
			"completed_at": r.CompletedAt,
		})
	}

	return SuccessResult(map[string]interface{}{
		"executions": executions,
		"count":      len(executions),
	}), nil
}

// -----------------------------------------------------------------------------
// Scenario Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) listScenarios(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	summaries, err := e.scenarioDirectory.ListSummaries(ctx)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to list scenarios: %v", err), CodeInternalError), nil
	}

	scenarioList := make([]map[string]interface{}, 0, len(summaries))
	for _, s := range summaries {
		scenarioList = append(scenarioList, map[string]interface{}{
			"name":        s.ScenarioName,
			"description": s.ScenarioDescription,
			"status":      s.ScenarioStatus,
			"tags":        s.ScenarioTags,
		})
	}

	return SuccessResult(map[string]interface{}{
		"scenarios": scenarioList,
		"count":     len(scenarioList),
	}), nil
}

func (e *ServerExecutor) getScenario(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	name := getStringArg(args, "name", "")
	if name == "" {
		return ErrorResult("name is required", CodeInvalidArgs), nil
	}

	summary, err := e.scenarioDirectory.GetSummary(ctx, name)
	if err != nil {
		return ErrorResult(fmt.Sprintf("scenario not found: %s", name), CodeNotFound), nil
	}

	return SuccessResult(map[string]interface{}{
		"name":        summary.ScenarioName,
		"description": summary.ScenarioDescription,
		"status":      summary.ScenarioStatus,
		"tags":        summary.ScenarioTags,
	}), nil
}

func (e *ServerExecutor) listScenarioFiles(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	name := getStringArg(args, "name", "")
	if name == "" {
		return ErrorResult("name is required", CodeInvalidArgs), nil
	}

	includeHidden := getBoolArg(args, "include_hidden", false)
	search := getStringArg(args, "search", "")
	limit := getIntArg(args, "limit", 200)

	files, err := e.scenarioDirectory.ListFiles(ctx, name, scenarios.FileListOptions{
		Search:        search,
		Limit:         limit,
		IncludeHidden: includeHidden,
	})
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to list files: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"scenario": name,
		"files":    files,
		"count":    len(files),
	}), nil
}

func (e *ServerExecutor) listPhases(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	descriptors := e.phaseCatalog.DescribePhases()

	phaseList := make([]map[string]interface{}, 0, len(descriptors))
	for _, d := range descriptors {
		phaseList = append(phaseList, map[string]interface{}{
			"name":        d.Name,
			"description": d.Description,
		})
	}

	return SuccessResult(map[string]interface{}{
		"phases": phaseList,
		"count":  len(phaseList),
	}), nil
}

// -----------------------------------------------------------------------------
// Fix Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) spawnFix(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenario := getStringArg(args, "scenario", "")
	if scenario == "" {
		return ErrorResult("scenario is required", CodeInvalidArgs), nil
	}

	message := getStringArg(args, "error_context", "")

	result, err := e.fixService.Spawn(ctx, fix.SpawnRequest{
		ScenarioName: scenario,
		Message:      message,
	})
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to spawn fix: %v", err), CodeInternalError), nil
	}

	return AsyncResult(map[string]interface{}{
		"fix_id":   result.FixID,
		"run_id":   result.RunID,
		"scenario": scenario,
		"status":   string(result.Status),
	}, result.FixID), nil
}

func (e *ServerExecutor) getFix(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	fixID := getStringArg(args, "fix_id", "")
	if fixID == "" {
		return ErrorResult("fix_id is required", CodeInvalidArgs), nil
	}

	record, found := e.fixService.Get(fixID)
	if !found {
		return ErrorResult(fmt.Sprintf("fix not found: %s", fixID), CodeNotFound), nil
	}

	return SuccessResult(map[string]interface{}{
		"fix_id":       record.ID,
		"scenario":     record.ScenarioName,
		"status":       string(record.Status),
		"run_id":       record.RunID,
		"started_at":   record.StartedAt,
		"completed_at": record.CompletedAt,
		"error":        record.Error,
	}), nil
}

func (e *ServerExecutor) listFixes(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenario := getStringArg(args, "scenario", "")
	if scenario == "" {
		return ErrorResult("scenario is required", CodeInvalidArgs), nil
	}

	limit := getIntArg(args, "limit", 10)

	records := e.fixService.ListByScenario(scenario, limit)

	fixes := make([]map[string]interface{}, 0, len(records))
	for _, r := range records {
		fixes = append(fixes, map[string]interface{}{
			"fix_id":     r.ID,
			"scenario":   r.ScenarioName,
			"status":     string(r.Status),
			"started_at": r.StartedAt,
			"completed_at": r.CompletedAt,
		})
	}

	return SuccessResult(map[string]interface{}{
		"fixes": fixes,
		"count": len(fixes),
	}), nil
}

func (e *ServerExecutor) getActiveFix(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenario := getStringArg(args, "scenario", "")
	if scenario == "" {
		return ErrorResult("scenario is required", CodeInvalidArgs), nil
	}

	record := e.fixService.GetActiveForScenario(scenario)
	if record == nil {
		return SuccessResult(map[string]interface{}{
			"active": false,
			"fix":    nil,
		}), nil
	}

	return SuccessResult(map[string]interface{}{
		"active": true,
		"fix": map[string]interface{}{
			"fix_id":     record.ID,
			"scenario":   record.ScenarioName,
			"status":     string(record.Status),
			"run_id":     record.RunID,
			"started_at": record.StartedAt,
		},
	}), nil
}

func (e *ServerExecutor) stopFix(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	fixID := getStringArg(args, "fix_id", "")
	if fixID == "" {
		return ErrorResult("fix_id is required", CodeInvalidArgs), nil
	}

	err := e.fixService.Stop(ctx, fixID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to stop fix: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Fix %s stopped", fixID),
	}), nil
}

// -----------------------------------------------------------------------------
// Requirements Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) improveRequirements(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenario := getStringArg(args, "scenario", "")
	if scenario == "" {
		return ErrorResult("scenario is required", CodeInvalidArgs), nil
	}

	result, err := e.requirementsImprove.Spawn(ctx, requirementsimprove.SpawnRequest{
		ScenarioName: scenario,
	})
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to spawn requirements improve: %v", err), CodeInternalError), nil
	}

	return AsyncResult(map[string]interface{}{
		"improve_id": result.ImproveID,
		"run_id":     result.RunID,
		"scenario":   scenario,
		"status":     string(result.Status),
	}, result.ImproveID), nil
}

func (e *ServerExecutor) getRequirementsImprove(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	improveID := getStringArg(args, "improve_id", "")
	if improveID == "" {
		return ErrorResult("improve_id is required", CodeInvalidArgs), nil
	}

	record, found := e.requirementsImprove.Get(improveID)
	if !found {
		return ErrorResult(fmt.Sprintf("improve not found: %s", improveID), CodeNotFound), nil
	}

	return SuccessResult(map[string]interface{}{
		"improve_id": record.ID,
		"scenario":   record.ScenarioName,
		"status":     string(record.Status),
		"run_id":     record.RunID,
		"started_at": record.StartedAt,
		"completed_at": record.CompletedAt,
		"error":        record.Error,
	}), nil
}

func (e *ServerExecutor) listRequirementsImproves(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenario := getStringArg(args, "scenario", "")
	if scenario == "" {
		return ErrorResult("scenario is required", CodeInvalidArgs), nil
	}

	limit := getIntArg(args, "limit", 10)

	records := e.requirementsImprove.ListByScenario(scenario, limit)

	improves := make([]map[string]interface{}, 0, len(records))
	for _, r := range records {
		improves = append(improves, map[string]interface{}{
			"improve_id": r.ID,
			"scenario":   r.ScenarioName,
			"status":     string(r.Status),
			"started_at": r.StartedAt,
			"completed_at": r.CompletedAt,
		})
	}

	return SuccessResult(map[string]interface{}{
		"improves": improves,
		"count":    len(improves),
	}), nil
}

func (e *ServerExecutor) getActiveRequirementsImprove(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenario := getStringArg(args, "scenario", "")
	if scenario == "" {
		return ErrorResult("scenario is required", CodeInvalidArgs), nil
	}

	record := e.requirementsImprove.GetActiveForScenario(scenario)
	if record == nil {
		return SuccessResult(map[string]interface{}{
			"active":  false,
			"improve": nil,
		}), nil
	}

	return SuccessResult(map[string]interface{}{
		"active": true,
		"improve": map[string]interface{}{
			"improve_id": record.ID,
			"scenario":   record.ScenarioName,
			"status":     string(record.Status),
			"run_id":     record.RunID,
			"started_at": record.StartedAt,
		},
	}), nil
}

func (e *ServerExecutor) stopRequirementsImprove(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	improveID := getStringArg(args, "improve_id", "")
	if improveID == "" {
		return ErrorResult("improve_id is required", CodeInvalidArgs), nil
	}

	err := e.requirementsImprove.Stop(ctx, improveID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to stop improve: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Improve %s stopped", improveID),
	}), nil
}

func (e *ServerExecutor) syncRequirements(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scenario := getStringArg(args, "scenario", "")
	if scenario == "" {
		return ErrorResult("scenario is required", CodeInvalidArgs), nil
	}

	scenarioDir := e.scenarioDirectory.ScenarioRoot() + "/" + scenario
	err := e.requirementsSyncer.Sync(ctx, scenarioDir)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to sync requirements: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"success":  true,
		"message":  fmt.Sprintf("Requirements synced for %s", scenario),
		"scenario": scenario,
	}), nil
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

// getStringArg extracts a string argument with a default value.
func getStringArg(args map[string]interface{}, key, defaultValue string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return defaultValue
}

// getIntArg extracts an int argument with a default value.
// Handles both int and float64 (JSON numbers decode as float64).
func getIntArg(args map[string]interface{}, key string, defaultValue int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	if v, ok := args[key].(int); ok {
		return v
	}
	return defaultValue
}

// getBoolArg extracts a bool argument with a default value.
func getBoolArg(args map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return defaultValue
}

// getStringSliceArg extracts a string slice argument.
func getStringSliceArg(args map[string]interface{}, key string) []string {
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
