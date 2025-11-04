package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"app-monitor-api/repository"
	"app-monitor-api/services"

	"github.com/gin-gonic/gin"
)

// AppHandler handles application-related endpoints
type AppHandler struct {
	appService *services.AppService
}

// NewAppHandler creates a new app handler
func NewAppHandler(appService *services.AppService) *AppHandler {
	return &AppHandler{
		appService: appService,
	}
}

// GetAppsSummary returns a fast-loading set of applications using cached CLI metadata
func (h *AppHandler) GetAppsSummary(c *gin.Context) {
	HandleServiceCallRaw(c, h.appService.GetAppsSummary, "Failed to fetch apps summary")
}

// GetApps returns all applications
func (h *AppHandler) GetApps(c *gin.Context) {
	HandleServiceCallRaw(c, h.appService.GetApps, "Failed to fetch apps")
}

// GetApp returns a single application by ID
func (h *AppHandler) GetApp(c *gin.Context) {
	id := c.Param("id")
	HandleServiceCall(c, func(ctx context.Context) (*repository.App, error) {
		return h.appService.GetApp(ctx, id)
	}, "App not found")
}

// StartApp starts an application
func (h *AppHandler) StartApp(c *gin.Context) {
	id := c.Param("id")
	HandleServiceAction(c, func(ctx context.Context) error {
		return h.appService.StartApp(ctx, id)
	}, "App started successfully", "Failed to start app")
}

// StopApp stops an application
func (h *AppHandler) StopApp(c *gin.Context) {
	id := c.Param("id")
	HandleServiceAction(c, func(ctx context.Context) error {
		return h.appService.StopApp(ctx, id)
	}, "App stopped successfully", "Failed to stop app")
}

// RestartApp restarts an application
func (h *AppHandler) RestartApp(c *gin.Context) {
	id := c.Param("id")
	HandleServiceAction(c, func(ctx context.Context) error {
		return h.appService.RestartApp(ctx, id)
	}, "App restarted successfully", "Failed to restart app")
}

// RecordAppView increments view counters for an application preview
func (h *AppHandler) RecordAppView(c *gin.Context) {
	id := c.Param("id")

	stats, err := h.appService.RecordAppView(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to record app view"))
		return
	}

	var data interface{}
	if stats != nil {
		data = gin.H{
			"scenario_name":   stats.ScenarioName,
			"view_count":      stats.ViewCount,
			"first_viewed_at": stats.FirstViewed,
			"last_viewed_at":  stats.LastViewed,
		}
	}

	c.JSON(http.StatusOK, successResponse(data))
}

// GetAppLogs returns logs for an application
func (h *AppHandler) GetAppLogs(c *gin.Context) {
	appName := c.Param("appName")
	logType := c.DefaultQuery("type", "both")

	logsResult, err := h.appService.GetAppLogs(c.Request.Context(), appName, logType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to fetch logs"))
		return
	}

	type logStreamResponse struct {
		Key     string   `json:"key"`
		Label   string   `json:"label"`
		Type    string   `json:"type"`
		Phase   string   `json:"phase,omitempty"`
		Step    string   `json:"step,omitempty"`
		Command string   `json:"command,omitempty"`
		Lines   []string `json:"lines"`
	}

	streams := make([]logStreamResponse, 0)
	aggregated := make([]string, 0)

	if logsResult != nil {
		if len(logsResult.Lifecycle) > 0 {
			streams = append(streams, logStreamResponse{
				Key:     "lifecycle",
				Label:   "Lifecycle",
				Type:    "lifecycle",
				Command: fmt.Sprintf("vrooli scenario logs %s --type lifecycle", appName),
				Lines:   logsResult.Lifecycle,
			})
			aggregated = append(aggregated, logsResult.Lifecycle...)
		}

		for _, bg := range logsResult.Background {
			label := bg.Label
			if strings.TrimSpace(label) == "" {
				label = bg.Step
				if bg.Phase != "" {
					label = fmt.Sprintf("%s (%s)", label, bg.Phase)
				}
			}

			key := fmt.Sprintf("background:%s", bg.Step)
			if bg.Phase != "" {
				key = fmt.Sprintf("%s:%s", key, bg.Phase)
			}

			streams = append(streams, logStreamResponse{
				Key:     key,
				Label:   label,
				Type:    "background",
				Phase:   bg.Phase,
				Step:    bg.Step,
				Command: bg.Command,
				Lines:   bg.Lines,
			})

			if len(bg.Lines) > 0 {
				if len(aggregated) > 0 {
					aggregated = append(aggregated, "")
				}
				aggregated = append(aggregated, fmt.Sprintf("--- Background: %s ---", label))
				aggregated = append(aggregated, bg.Lines...)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":    aggregated,
		"streams": streams,
	})
}

// GetAppMetrics returns metrics for an application
func (h *AppHandler) GetAppMetrics(c *gin.Context) {
	id := c.Param("id")
	hoursStr := c.DefaultQuery("hours", "24")
	hours, _ := strconv.Atoi(hoursStr)

	metrics, err := h.appService.GetAppStatusHistory(c.Request.Context(), id, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to fetch metrics"))
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetAppLifecycleLogs returns lifecycle logs for an application
func (h *AppHandler) GetAppLifecycleLogs(c *gin.Context) {
	appName := c.Param("id")

	logsResult, err := h.appService.GetAppLogs(c.Request.Context(), appName, "lifecycle")
	if err != nil {
		c.String(http.StatusOK, "No lifecycle logs available")
		return
	}

	if logsResult == nil || len(logsResult.Lifecycle) == 0 {
		c.String(http.StatusOK, "No lifecycle logs available")
	} else {
		// Join logs with newlines for text response
		response := ""
		for _, log := range logsResult.Lifecycle {
			response += log + "\n"
		}
		c.String(http.StatusOK, response)
	}
}

// GetAppBackgroundLogs returns background process logs for an application
func (h *AppHandler) GetAppBackgroundLogs(c *gin.Context) {
	appName := c.Param("id")

	logsResult, err := h.appService.GetAppLogs(c.Request.Context(), appName, "background")
	if err != nil {
		c.JSON(http.StatusOK, []map[string]interface{}{
			{
				"level":   "info",
				"message": "No background logs available",
				"source":  "system",
			},
		})
		return
	}

	// Convert string logs to structured format
	structuredLogs := make([]map[string]interface{}, 0)
	if logsResult != nil {
		for _, bg := range logsResult.Background {
			for _, line := range bg.Lines {
				structuredLogs = append(structuredLogs, map[string]interface{}{
					"level":   "log",
					"message": line,
					"source":  "background",
					"step":    bg.Step,
					"phase":   bg.Phase,
				})
			}
		}
	}

	c.JSON(http.StatusOK, structuredLogs)
}

// GetAppIssues returns existing issues for an application from app-issue-tracker.
func (h *AppHandler) GetAppIssues(c *gin.Context) {
	id := c.Param("id")

	issuesSummary, err := h.appService.ListScenarioIssues(c.Request.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrAppIdentifierRequired):
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrAppNotFound):
			status = http.StatusNotFound
		case errors.Is(err, services.ErrIssueTrackerUnavailable):
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, errorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    issuesSummary,
	})
}

// GetFallbackDiagnostics retrieves console logs, network requests, and page status using browserless
// This is used when the iframe bridge fails to provide diagnostics
func (h *AppHandler) GetFallbackDiagnostics(c *gin.Context) {
	appID := c.Param("id")

	var payload struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("URL is required"))
		return
	}

	result, err := h.appService.GetFallbackDiagnostics(c.Request.Context(), appID, payload.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(fmt.Sprintf("Failed to retrieve fallback diagnostics: %v", err)))
		return
	}

	c.JSON(http.StatusOK, successResponse(result))
}

// ReportAppIssue forwards an application issue report to the issue tracker scenario
func (h *AppHandler) ReportAppIssue(c *gin.Context) {
	appID := c.Param("id")

	var payload services.IssueReportRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid issue report payload"))
		return
	}

	payload.AppID = appID

	result, err := h.appService.ReportAppIssue(c.Request.Context(), &payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
		return
	}

	data := gin.H{"message": result.Message}
	if result.IssueID != "" {
		data["issue_id"] = result.IssueID
	}
	if result.IssueURL != "" {
		data["issue_url"] = result.IssueURL
	}

	c.JSON(http.StatusOK, successResponse(data))
}

// CheckAppIframeBridge evaluates iframe bridge diagnostics via scenario-auditor.
func (h *AppHandler) CheckAppIframeBridge(c *gin.Context) {
	id := c.Param("id")

	result, err := h.appService.CheckIframeBridgeRule(c.Request.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrAppIdentifierRequired):
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrAppNotFound), errors.Is(err, services.ErrScenarioBridgeScenarioMissing):
			status = http.StatusNotFound
		case errors.Is(err, services.ErrScenarioAuditorUnavailable):
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, errorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetAppScenarioStatus returns a snapshot of the scenario's orchestrator status.
func (h *AppHandler) GetAppScenarioStatus(c *gin.Context) {
	id := c.Param("id")

	result, err := h.appService.GetAppScenarioStatus(c.Request.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrAppIdentifierRequired):
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrAppNotFound):
			status = http.StatusNotFound
		}

		c.JSON(status, errorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// CheckAppHealth executes health checks against the previewed app's API and UI endpoints.
func (h *AppHandler) CheckAppHealth(c *gin.Context) {
	id := c.Param("id")

	result, err := h.appService.CheckAppHealth(c.Request.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrAppIdentifierRequired):
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrAppNotFound):
			status = http.StatusNotFound
		}

		c.JSON(status, errorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// CheckAppLocalhostUsage scans scenario files for localhost references that bypass the proxy.
func (h *AppHandler) CheckAppLocalhostUsage(c *gin.Context) {
	id := c.Param("id")

	result, err := h.appService.CheckLocalhostUsage(c.Request.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrAppIdentifierRequired):
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrAppNotFound):
			status = http.StatusNotFound
		case errors.Is(err, context.Canceled):
			status = http.StatusRequestTimeout
		}

		c.JSON(status, errorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetAppCompleteDiagnostics returns aggregated diagnostics for an application
func (h *AppHandler) GetAppCompleteDiagnostics(c *gin.Context) {
	id := c.Param("id")

	// Parse options from query parameters
	opts := services.DefaultDiagnosticOptions()

	// Allow selective fetching via query params
	if c.Query("fast") == "true" {
		opts = services.FastDiagnosticOptions()
	} else {
		if c.Query("health") == "false" {
			opts.IncludeHealth = false
		}
		if c.Query("issues") == "false" {
			opts.IncludeIssues = false
		}
		if c.Query("bridge") == "false" {
			opts.IncludeBridgeRules = false
		}
		if c.Query("localhost") == "false" {
			opts.IncludeLocalhostScan = false
		}
		if c.Query("tech_stack") == "false" {
			opts.IncludeTechStack = false
		}
		if c.Query("docs") == "false" {
			opts.IncludeDocuments = false
		}
		if c.Query("status") == "false" {
			opts.IncludeStatus = false
		}
	}

	diagnostics, err := h.appService.GetCompleteDiagnostics(c.Request.Context(), id, opts)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrAppIdentifierRequired):
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrAppNotFound):
			status = http.StatusNotFound
		}

		c.JSON(status, errorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    diagnostics,
	})
}

// GetAppDocuments lists all available documentation for an application
func (h *AppHandler) GetAppDocuments(c *gin.Context) {
	id := c.Param("id")

	docsList, err := h.appService.ListAppDocuments(c.Request.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrAppIdentifierRequired):
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrAppNotFound):
			status = http.StatusNotFound
		}

		c.JSON(status, errorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    docsList,
	})
}

// GetAppDocument retrieves a specific document for an application
func (h *AppHandler) GetAppDocument(c *gin.Context) {
	id := c.Param("id")
	docPath := c.Param("path")

	// Check if rendering is requested
	render := c.DefaultQuery("render", "true") == "true"

	doc, err := h.appService.GetAppDocument(c.Request.Context(), id, docPath, render)
	if err != nil {
		status := http.StatusInternalServerError
		errMsg := err.Error()

		switch {
		case errors.Is(err, services.ErrAppIdentifierRequired):
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrAppNotFound):
			status = http.StatusNotFound
		case strings.Contains(errMsg, "not found"):
			status = http.StatusNotFound
		case strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "traversal"):
			status = http.StatusBadRequest
		}

		c.JSON(status, errorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    doc,
	})
}

// SearchAppDocuments searches documentation content
func (h *AppHandler) SearchAppDocuments(c *gin.Context) {
	id := c.Param("id")
	query := c.Query("q")

	if strings.TrimSpace(query) == "" {
		c.JSON(http.StatusBadRequest, errorResponse("search query parameter 'q' is required"))
		return
	}

	results, err := h.appService.SearchAppDocuments(c.Request.Context(), id, query)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrAppIdentifierRequired):
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrAppNotFound):
			status = http.StatusNotFound
		}

		c.JSON(status, errorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
	})
}
