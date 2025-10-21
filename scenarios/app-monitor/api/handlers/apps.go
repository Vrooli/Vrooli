package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
	apps, err := h.appService.GetAppsSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch apps summary",
		})
		return
	}

	c.JSON(http.StatusOK, apps)
}

// GetApps returns all applications
func (h *AppHandler) GetApps(c *gin.Context) {
	apps, err := h.appService.GetApps(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch apps",
		})
		return
	}

	c.JSON(http.StatusOK, apps)
}

// GetApp returns a single application by ID
func (h *AppHandler) GetApp(c *gin.Context) {
	id := c.Param("id")

	app, err := h.appService.GetApp(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "App not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    app,
	})
}

// StartApp starts an application
func (h *AppHandler) StartApp(c *gin.Context) {
	id := c.Param("id")

	if err := h.appService.StartApp(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start app",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "App started successfully",
	})
}

// StopApp stops an application
func (h *AppHandler) StopApp(c *gin.Context) {
	id := c.Param("id")

	if err := h.appService.StopApp(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to stop app",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "App stopped successfully",
	})
}

// RestartApp restarts an application
func (h *AppHandler) RestartApp(c *gin.Context) {
	id := c.Param("id")

	if err := h.appService.RestartApp(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to restart app",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "App restarted successfully",
	})
}

// RecordAppView increments view counters for an application preview
func (h *AppHandler) RecordAppView(c *gin.Context) {
	id := c.Param("id")

	stats, err := h.appService.RecordAppView(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to record app view",
		})
		return
	}

	response := gin.H{
		"success": true,
	}

	if stats != nil {
		response["data"] = gin.H{
			"scenario_name":   stats.ScenarioName,
			"view_count":      stats.ViewCount,
			"first_viewed_at": stats.FirstViewed,
			"last_viewed_at":  stats.LastViewed,
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetAppLogs returns logs for an application
func (h *AppHandler) GetAppLogs(c *gin.Context) {
	appName := c.Param("appName")
	logType := c.DefaultQuery("type", "both")

	logsResult, err := h.appService.GetAppLogs(c.Request.Context(), appName, logType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"logs":  []string{},
			"error": "Failed to fetch logs",
		})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch metrics",
		})
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

		c.JSON(status, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    issuesSummary,
	})
}

// ReportAppIssue forwards an application issue report to the issue tracker scenario
func (h *AppHandler) ReportAppIssue(c *gin.Context) {
	appID := c.Param("id")

	var payload services.IssueReportRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid issue report payload",
		})
		return
	}

	payload.AppID = appID

	result, err := h.appService.ReportAppIssue(c.Request.Context(), &payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	response := gin.H{
		"success": true,
		"message": result.Message,
	}

	data := gin.H{}
	if result.IssueID != "" {
		data["issue_id"] = result.IssueID
	}
	if result.IssueURL != "" {
		data["issue_url"] = result.IssueURL
	}
	if len(data) > 0 {
		response["data"] = data
	}

	c.JSON(http.StatusOK, response)
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

		c.JSON(status, gin.H{
			"success": false,
			"error":   err.Error(),
		})
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

		c.JSON(status, gin.H{
			"success": false,
			"error":   err.Error(),
		})
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

		c.JSON(status, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
