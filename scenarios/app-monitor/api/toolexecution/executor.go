// Package toolexecution implements tool execution for app-monitor.
//
// This file provides the ServerExecutor which dispatches tool calls
// to the appropriate service methods.
package toolexecution

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"app-monitor-api/services"
)

// ServerExecutor dispatches tool execution to appropriate handlers.
type ServerExecutor struct {
	appService     *services.AppService
	metricsService *services.MetricsService
}

// ServerExecutorConfig holds dependencies for creating a ServerExecutor.
type ServerExecutorConfig struct {
	AppService     *services.AppService
	MetricsService *services.MetricsService
}

// NewServerExecutor creates a new ServerExecutor with the given configuration.
func NewServerExecutor(cfg ServerExecutorConfig) *ServerExecutor {
	return &ServerExecutor{
		appService:     cfg.AppService,
		metricsService: cfg.MetricsService,
	}
}

// Execute dispatches a tool call to the appropriate handler.
func (e *ServerExecutor) Execute(ctx context.Context, toolName string, args map[string]interface{}) (*ExecutionResult, error) {
	switch toolName {
	// Discovery tools
	case "list_apps":
		return e.listApps(ctx, args)
	case "get_app":
		return e.getApp(ctx, args)
	case "get_system_status":
		return e.getSystemStatus(ctx, args)
	case "get_app_summary":
		return e.getAppSummary(ctx, args)
	case "list_resources":
		return e.listResources(ctx, args)

	// Lifecycle tools
	case "start_app":
		return e.startApp(ctx, args)
	case "stop_app":
		return e.stopApp(ctx, args)
	case "restart_app":
		return e.restartApp(ctx, args)

	// Diagnostics tools
	case "get_app_diagnostics":
		return e.getAppDiagnostics(ctx, args)
	case "get_app_health":
		return e.getAppHealth(ctx, args)
	case "check_iframe_bridge":
		return e.checkIframeBridge(ctx, args)
	case "check_localhost_usage":
		return e.checkLocalhostUsage(ctx, args)
	case "get_fallback_diagnostics":
		return e.getFallbackDiagnostics(ctx, args)
	case "get_app_completeness":
		return e.getAppCompleteness(ctx, args)

	// Logs & Metrics tools
	case "get_app_logs":
		return e.getAppLogs(ctx, args)
	case "get_app_metrics":
		return e.getAppMetrics(ctx, args)
	case "get_system_metrics":
		return e.getSystemMetrics(ctx, args)
	case "search_logs":
		return e.searchLogs(ctx, args)

	// Issue tools
	case "list_app_issues":
		return e.listAppIssues(ctx, args)
	case "report_app_issue":
		return e.reportAppIssue(ctx, args)

	// Documentation tools
	case "list_app_docs":
		return e.listAppDocs(ctx, args)
	case "get_app_doc":
		return e.getAppDoc(ctx, args)
	case "search_app_docs":
		return e.searchAppDocs(ctx, args)

	// Resource tools
	case "get_resource":
		return e.getResource(ctx, args)
	case "start_resource":
		return e.startResource(ctx, args)
	case "stop_resource":
		return e.stopResource(ctx, args)

	default:
		return ErrorResult(fmt.Sprintf("unknown tool: %s", toolName), CodeUnknownTool), nil
	}
}

// -----------------------------------------------------------------------------
// Discovery Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) listApps(ctx context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	apps, err := e.appService.GetApps(ctx)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to list apps: %v", err), CodeInternalError), nil
	}
	return SuccessResult(apps), nil
}

func (e *ServerExecutor) getApp(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	app, err := e.appService.GetApp(ctx, appID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("app not found: %s", appID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to get app: %v", err), CodeInternalError), nil
	}
	return SuccessResult(app), nil
}

func (e *ServerExecutor) getSystemStatus(ctx context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	// Get system status summary
	apps, err := e.appService.GetApps(ctx)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to get system status: %v", err), CodeInternalError), nil
	}

	// Count running apps
	runningCount := 0
	for _, app := range apps {
		if app.Status == "running" {
			runningCount++
		}
	}

	result := map[string]interface{}{
		"total_apps":   len(apps),
		"running_apps": runningCount,
		"apps":         apps,
	}
	return SuccessResult(result), nil
}

func (e *ServerExecutor) getAppSummary(ctx context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	summary, err := e.appService.GetAppsSummary(ctx)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to get app summary: %v", err), CodeInternalError), nil
	}
	return SuccessResult(summary), nil
}

func (e *ServerExecutor) listResources(ctx context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	resources, err := e.appService.GetResources(ctx)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to list resources: %v", err), CodeInternalError), nil
	}
	return SuccessResult(resources), nil
}

// -----------------------------------------------------------------------------
// Lifecycle Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) startApp(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	if err := e.appService.StartApp(ctx, appID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("app not found: %s", appID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to start app: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"message": fmt.Sprintf("App %s started successfully", appID),
		"app_id":  appID,
	}), nil
}

func (e *ServerExecutor) stopApp(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	if err := e.appService.StopApp(ctx, appID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("app not found: %s", appID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to stop app: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"message": fmt.Sprintf("App %s stopped successfully", appID),
		"app_id":  appID,
	}), nil
}

func (e *ServerExecutor) restartApp(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	if err := e.appService.RestartApp(ctx, appID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("app not found: %s", appID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to restart app: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"message": fmt.Sprintf("App %s restarted successfully", appID),
		"app_id":  appID,
	}), nil
}

// -----------------------------------------------------------------------------
// Diagnostics Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) getAppDiagnostics(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	opts := services.DefaultDiagnosticOptions()
	diagnostics, err := e.appService.GetCompleteDiagnostics(ctx, appID, opts)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("app not found: %s", appID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to get diagnostics: %v", err), CodeInternalError), nil
	}
	return SuccessResult(diagnostics), nil
}

func (e *ServerExecutor) getAppHealth(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	health, err := e.appService.CheckAppHealth(ctx, appID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("app not found: %s", appID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to check health: %v", err), CodeInternalError), nil
	}
	return SuccessResult(health), nil
}

func (e *ServerExecutor) checkIframeBridge(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	result, err := e.appService.CheckIframeBridgeRule(ctx, appID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("app not found: %s", appID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to check iframe bridge: %v", err), CodeInternalError), nil
	}
	return SuccessResult(result), nil
}

func (e *ServerExecutor) checkLocalhostUsage(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	result, err := e.appService.CheckLocalhostUsage(ctx, appID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("app not found: %s", appID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to check localhost usage: %v", err), CodeInternalError), nil
	}
	return SuccessResult(result), nil
}

func (e *ServerExecutor) getFallbackDiagnostics(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	url := getOptionalStringArg(args, "url", "")

	result, err := e.appService.GetFallbackDiagnostics(ctx, appID, url)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("app not found: %s", appID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to get fallback diagnostics: %v", err), CodeInternalError), nil
	}

	// This is a potentially long-running operation
	return AsyncResult(result, appID), nil
}

func (e *ServerExecutor) getAppCompleteness(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	completeness, err := e.appService.GetAppCompleteness(ctx, appID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("app not found: %s", appID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to get completeness: %v", err), CodeInternalError), nil
	}
	return SuccessResult(completeness), nil
}

// -----------------------------------------------------------------------------
// Logs & Metrics Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) getAppLogs(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	logType := getOptionalStringArg(args, "log_type", "all")

	logs, err := e.appService.GetAppLogs(ctx, appID, logType)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to get logs: %v", err), CodeInternalError), nil
	}
	return SuccessResult(logs), nil
}

func (e *ServerExecutor) getAppMetrics(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	hours := getOptionalIntArg(args, "hours", 24)

	metrics, err := e.appService.GetAppStatusHistory(ctx, appID, hours)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to get metrics: %v", err), CodeInternalError), nil
	}
	return SuccessResult(metrics), nil
}

func (e *ServerExecutor) getSystemMetrics(ctx context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	if e.metricsService == nil {
		return SuccessResult(map[string]interface{}{
			"message": "Metrics service not available",
		}), nil
	}

	metrics, err := e.metricsService.GetSystemMetrics(ctx)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to get system metrics: %v", err), CodeInternalError), nil
	}
	return SuccessResult(metrics), nil
}

func (e *ServerExecutor) searchLogs(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appName, err := getStringArg(args, "app_name")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	logs, err := e.appService.GetAppLogs(ctx, appName, "all")
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to search logs: %v", err), CodeInternalError), nil
	}
	return SuccessResult(logs), nil
}

// -----------------------------------------------------------------------------
// Issue Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) listAppIssues(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	issues, err := e.appService.ListScenarioIssues(ctx, appID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("app not found: %s", appID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to list issues: %v", err), CodeInternalError), nil
	}
	return SuccessResult(issues), nil
}

func (e *ServerExecutor) reportAppIssue(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	message, err := getStringArg(args, "message")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	// Build issue report request
	req := &services.IssueReportRequest{
		AppID:   appID,
		Message: message,
	}

	// Add optional fields if provided
	if screenshot, ok := args["screenshot_data"].(string); ok {
		req.ScreenshotData = &screenshot
	}

	result, err := e.appService.ReportAppIssue(ctx, req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to report issue: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"message":   result.Message,
		"issue_id":  result.IssueID,
		"issue_url": result.IssueURL,
	}), nil
}

// -----------------------------------------------------------------------------
// Documentation Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) listAppDocs(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	docs, err := e.appService.ListAppDocuments(ctx, appID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("app not found: %s", appID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to list docs: %v", err), CodeInternalError), nil
	}
	return SuccessResult(docs), nil
}

func (e *ServerExecutor) getAppDoc(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	path, err := getStringArg(args, "path")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	doc, err := e.appService.GetAppDocument(ctx, appID, path, true)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("document not found: %s/%s", appID, path), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to get document: %v", err), CodeInternalError), nil
	}
	return SuccessResult(doc), nil
}

func (e *ServerExecutor) searchAppDocs(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	appID, err := getStringArg(args, "app_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	query, err := getStringArg(args, "query")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	results, err := e.appService.SearchAppDocuments(ctx, appID, query)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("app not found: %s", appID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to search docs: %v", err), CodeInternalError), nil
	}
	return SuccessResult(results), nil
}

// -----------------------------------------------------------------------------
// Resource Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) getResource(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	resourceID, err := getStringArg(args, "resource_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	resource, err := e.appService.GetResource(ctx, resourceID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("resource not found: %s", resourceID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to get resource: %v", err), CodeInternalError), nil
	}
	return SuccessResult(resource), nil
}

func (e *ServerExecutor) startResource(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	resourceID, err := getStringArg(args, "resource_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	if err := e.appService.StartResource(ctx, resourceID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("resource not found: %s", resourceID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to start resource: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"message":     fmt.Sprintf("Resource %s started successfully", resourceID),
		"resource_id": resourceID,
	}), nil
}

func (e *ServerExecutor) stopResource(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	resourceID, err := getStringArg(args, "resource_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	if err := e.appService.StopResource(ctx, resourceID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrorResult(fmt.Sprintf("resource not found: %s", resourceID), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to stop resource: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"message":     fmt.Sprintf("Resource %s stopped successfully", resourceID),
		"resource_id": resourceID,
	}), nil
}

// -----------------------------------------------------------------------------
// Argument Helpers
// -----------------------------------------------------------------------------

func getStringArg(args map[string]interface{}, key string) (string, error) {
	val, ok := args[key]
	if !ok {
		return "", fmt.Errorf("%s is required", key)
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	if strings.TrimSpace(str) == "" {
		return "", fmt.Errorf("%s cannot be empty", key)
	}
	return str, nil
}

func getOptionalStringArg(args map[string]interface{}, key, defaultValue string) string {
	val, ok := args[key]
	if !ok {
		return defaultValue
	}
	str, ok := val.(string)
	if !ok {
		return defaultValue
	}
	return str
}

func getOptionalIntArg(args map[string]interface{}, key string, defaultValue int) int {
	val, ok := args[key]
	if !ok {
		return defaultValue
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}
