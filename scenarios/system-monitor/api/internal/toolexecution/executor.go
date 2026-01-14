// Package toolexecution provides the tool execution service for system-monitor.
package toolexecution

import (
	"context"
	"fmt"
	"log/slog"

	"system-monitor-api/internal/services"
)

// ServerExecutorConfig holds the configuration for creating a ServerExecutor.
type ServerExecutorConfig struct {
	MonitorSvc       *services.MonitorService
	InvestigationSvc *services.InvestigationService
	ReportSvc        *services.ReportService
	SettingsMgr      *services.SettingsManager
	Logger           *slog.Logger
}

// ServerExecutor executes tools using the scenario's services.
type ServerExecutor struct {
	monitorSvc       *services.MonitorService
	investigationSvc *services.InvestigationService
	reportSvc        *services.ReportService
	settingsMgr      *services.SettingsManager
	log              *slog.Logger
}

// NewServerExecutor creates a new ServerExecutor.
func NewServerExecutor(cfg ServerExecutorConfig) *ServerExecutor {
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}
	return &ServerExecutor{
		monitorSvc:       cfg.MonitorSvc,
		investigationSvc: cfg.InvestigationSvc,
		reportSvc:        cfg.ReportSvc,
		settingsMgr:      cfg.SettingsMgr,
		log:              log,
	}
}

// Execute dispatches a tool execution request to the appropriate handler.
func (e *ServerExecutor) Execute(ctx context.Context, toolName string, args map[string]interface{}) (*ExecutionResult, error) {
	e.log.Debug("executing tool", "tool", toolName, "args", args)

	switch toolName {
	// Metrics tools
	case "get_metrics":
		return e.executeGetMetrics(ctx, args)
	case "get_detailed_metrics":
		return e.executeGetDetailedMetrics(ctx, args)
	case "get_processes":
		return e.executeGetProcesses(ctx, args)
	case "get_system_health":
		return e.executeGetSystemHealth(ctx, args)

	// Investigation tools
	case "trigger_investigation":
		return e.executeTriggerInvestigation(ctx, args)
	case "check_investigation_status":
		return e.executeCheckInvestigationStatus(ctx, args)
	case "get_latest_investigation":
		return e.executeGetLatestInvestigation(ctx, args)
	case "stop_investigation":
		return e.executeStopInvestigation(ctx, args)
	case "generate_report":
		return e.executeGenerateReport(ctx, args)

	// Configuration tools
	case "get_triggers":
		return e.executeGetTriggers(ctx, args)
	case "update_trigger":
		return e.executeUpdateTrigger(ctx, args)
	case "get_cooldown_status":
		return e.executeGetCooldownStatus(ctx, args)
	case "reset_cooldown":
		return e.executeResetCooldown(ctx, args)

	default:
		return NewErrorResult(ErrorCodeUnknownTool, fmt.Sprintf("unknown tool: %s", toolName)), nil
	}
}

// =============================================================================
// Metrics Tool Implementations
// =============================================================================

func (e *ServerExecutor) executeGetMetrics(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	if e.monitorSvc == nil {
		return NewErrorResult(ErrorCodeUnavailable, "monitor service not available"), nil
	}

	// Check if fresh=true (default)
	fresh := true
	if freshArg, ok := args["fresh"].(bool); ok {
		fresh = freshArg
	}

	var metrics interface{}
	var err error
	if fresh {
		metrics, err = e.monitorSvc.GetCurrentMetricsFresh(ctx)
	} else {
		metrics, err = e.monitorSvc.GetCurrentMetrics(ctx)
	}

	if err != nil {
		return NewErrorResult(ErrorCodeInternalError, fmt.Sprintf("failed to get metrics: %v", err)), nil
	}

	return NewSuccessResult(metrics), nil
}

func (e *ServerExecutor) executeGetDetailedMetrics(ctx context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	if e.monitorSvc == nil {
		return NewErrorResult(ErrorCodeUnavailable, "monitor service not available"), nil
	}

	metrics, err := e.monitorSvc.GetDetailedMetrics(ctx)
	if err != nil {
		return NewErrorResult(ErrorCodeInternalError, fmt.Sprintf("failed to get detailed metrics: %v", err)), nil
	}

	return NewSuccessResult(metrics), nil
}

func (e *ServerExecutor) executeGetProcesses(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	if e.monitorSvc == nil {
		return NewErrorResult(ErrorCodeUnavailable, "monitor service not available"), nil
	}

	// Process arguments are handled internally by the service
	data, err := e.monitorSvc.GetProcessMonitorData(ctx)
	if err != nil {
		return NewErrorResult(ErrorCodeInternalError, fmt.Sprintf("failed to get process data: %v", err)), nil
	}

	return NewSuccessResult(data), nil
}

func (e *ServerExecutor) executeGetSystemHealth(ctx context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	// Build health response similar to health handler
	healthData := map[string]interface{}{
		"status":    "healthy",
		"readiness": true,
	}

	// Add metrics if available
	if e.monitorSvc != nil {
		if metrics, err := e.monitorSvc.GetCurrentMetrics(ctx); err == nil && metrics != nil {
			healthData["cpu_usage_percent"] = metrics.CPUUsage
			healthData["memory_usage_percent"] = metrics.MemoryUsage
		}
	}

	// Add settings manager info
	if e.settingsMgr != nil {
		healthData["processor_active"] = e.settingsMgr.IsActive()
		healthData["maintenance_state"] = e.settingsMgr.GetMaintenanceState()
	}

	return NewSuccessResult(healthData), nil
}

// =============================================================================
// Investigation Tool Implementations
// =============================================================================

func (e *ServerExecutor) executeTriggerInvestigation(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	if e.investigationSvc == nil {
		return NewErrorResult(ErrorCodeUnavailable, "investigation service not available"), nil
	}

	// Parse arguments
	autoFix := false
	if autoFixArg, ok := args["auto_fix"].(bool); ok {
		autoFix = autoFixArg
	}

	note := ""
	if noteArg, ok := args["note"].(string); ok {
		note = noteArg
	}

	// Trigger investigation
	investigation, err := e.investigationSvc.TriggerInvestigation(ctx, autoFix, note)
	if err != nil {
		// Check if it's a cooldown error
		errMsg := err.Error()
		if errMsg != "" && (errMsg[:11] == "cooldown" || errMsg[:len("investigation is in cooldown")] == "investigation is in cooldown") {
			return NewErrorResult(ErrorCodeCooldown, err.Error()), nil
		}
		return NewErrorResult(ErrorCodeInternalError, fmt.Sprintf("failed to trigger investigation: %v", err)), nil
	}

	// Return async result
	result := map[string]interface{}{
		"investigation_id": investigation.ID,
		"message":          "Investigation started",
		"auto_fix":         autoFix,
	}
	if note != "" {
		result["note"] = note
	}

	return NewAsyncResult(investigation.ID, investigation.Status, result), nil
}

func (e *ServerExecutor) executeCheckInvestigationStatus(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	if e.investigationSvc == nil {
		return NewErrorResult(ErrorCodeUnavailable, "investigation service not available"), nil
	}

	// Get investigation ID
	investigationID, ok := args["investigation_id"].(string)
	if !ok || investigationID == "" {
		return NewErrorResult(ErrorCodeInvalidArgs, "investigation_id is required"), nil
	}

	// Get investigation status
	investigation, err := e.investigationSvc.GetInvestigation(ctx, investigationID)
	if err != nil {
		return NewErrorResult(ErrorCodeNotFound, fmt.Sprintf("investigation not found: %s", investigationID)), nil
	}

	// Return investigation data
	result := map[string]interface{}{
		"investigation_id": investigation.ID,
		"status":           investigation.Status,
		"progress":         investigation.Progress,
		"findings":         investigation.Findings,
		"start_time":       investigation.StartTime,
		"details":          investigation.Details,
	}

	if investigation.EndTime != nil {
		result["end_time"] = investigation.EndTime
	}

	return NewSuccessResult(result), nil
}

func (e *ServerExecutor) executeGetLatestInvestigation(ctx context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	if e.investigationSvc == nil {
		return NewErrorResult(ErrorCodeUnavailable, "investigation service not available"), nil
	}

	investigation, err := e.investigationSvc.GetLatestInvestigation(ctx)
	if err != nil {
		return NewErrorResult(ErrorCodeInternalError, fmt.Sprintf("failed to get latest investigation: %v", err)), nil
	}

	return NewSuccessResult(investigation), nil
}

func (e *ServerExecutor) executeStopInvestigation(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	if e.investigationSvc == nil {
		return NewErrorResult(ErrorCodeUnavailable, "investigation service not available"), nil
	}

	// Get investigation ID
	investigationID, ok := args["investigation_id"].(string)
	if !ok || investigationID == "" {
		return NewErrorResult(ErrorCodeInvalidArgs, "investigation_id is required"), nil
	}

	// Stop the investigation
	err := e.investigationSvc.StopInvestigationAgent(ctx, investigationID)
	if err != nil {
		return NewErrorResult(ErrorCodeInternalError, fmt.Sprintf("failed to stop investigation: %v", err)), nil
	}

	return NewSuccessResult(map[string]interface{}{
		"investigation_id": investigationID,
		"status":           "stopped",
		"message":          "Investigation stopped successfully",
	}), nil
}

func (e *ServerExecutor) executeGenerateReport(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	if e.reportSvc == nil {
		return NewErrorResult(ErrorCodeUnavailable, "report service not available"), nil
	}

	// Get report type (default: daily)
	reportType := "daily"
	if typeArg, ok := args["type"].(string); ok && typeArg != "" {
		reportType = typeArg
	}

	// Validate report type
	if reportType != "daily" && reportType != "weekly" {
		return NewErrorResult(ErrorCodeInvalidArgs, "type must be 'daily' or 'weekly'"), nil
	}

	// Generate report
	report, err := e.reportSvc.GenerateReport(ctx, reportType)
	if err != nil {
		return NewErrorResult(ErrorCodeInternalError, fmt.Sprintf("failed to generate report: %v", err)), nil
	}

	return NewSuccessResult(report), nil
}

// =============================================================================
// Configuration Tool Implementations
// =============================================================================

func (e *ServerExecutor) executeGetTriggers(ctx context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	if e.investigationSvc == nil {
		return NewErrorResult(ErrorCodeUnavailable, "investigation service not available"), nil
	}

	triggers, err := e.investigationSvc.GetTriggers(ctx)
	if err != nil {
		return NewErrorResult(ErrorCodeInternalError, fmt.Sprintf("failed to get triggers: %v", err)), nil
	}

	return NewSuccessResult(triggers), nil
}

func (e *ServerExecutor) executeUpdateTrigger(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	if e.investigationSvc == nil {
		return NewErrorResult(ErrorCodeUnavailable, "investigation service not available"), nil
	}

	// Get trigger ID
	triggerID, ok := args["trigger_id"].(string)
	if !ok || triggerID == "" {
		return NewErrorResult(ErrorCodeInvalidArgs, "trigger_id is required"), nil
	}

	// Parse optional parameters
	var enabled *bool
	if enabledArg, ok := args["enabled"].(bool); ok {
		enabled = &enabledArg
	}

	var autoFix *bool
	if autoFixArg, ok := args["auto_fix"].(bool); ok {
		autoFix = &autoFixArg
	}

	var threshold *float64
	if thresholdArg, ok := args["threshold"].(float64); ok {
		threshold = &thresholdArg
	}

	// Update trigger
	err := e.investigationSvc.UpdateTrigger(ctx, triggerID, enabled, autoFix, threshold)
	if err != nil {
		// Check if trigger not found
		if err.Error()[:len("trigger not found")] == "trigger not found" {
			return NewErrorResult(ErrorCodeNotFound, err.Error()), nil
		}
		return NewErrorResult(ErrorCodeInternalError, fmt.Sprintf("failed to update trigger: %v", err)), nil
	}

	return NewSuccessResult(map[string]interface{}{
		"trigger_id": triggerID,
		"status":     "updated",
		"message":    "Trigger updated successfully",
	}), nil
}

func (e *ServerExecutor) executeGetCooldownStatus(ctx context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	if e.investigationSvc == nil {
		return NewErrorResult(ErrorCodeUnavailable, "investigation service not available"), nil
	}

	status, err := e.investigationSvc.GetCooldownStatus(ctx)
	if err != nil {
		return NewErrorResult(ErrorCodeInternalError, fmt.Sprintf("failed to get cooldown status: %v", err)), nil
	}

	return NewSuccessResult(status), nil
}

func (e *ServerExecutor) executeResetCooldown(ctx context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	if e.investigationSvc == nil {
		return NewErrorResult(ErrorCodeUnavailable, "investigation service not available"), nil
	}

	err := e.investigationSvc.ResetCooldown(ctx)
	if err != nil {
		return NewErrorResult(ErrorCodeInternalError, fmt.Sprintf("failed to reset cooldown: %v", err)), nil
	}

	return NewSuccessResult(map[string]interface{}{
		"status":  "reset",
		"message": "Cooldown reset successfully. Investigation can be triggered immediately.",
	}), nil
}
