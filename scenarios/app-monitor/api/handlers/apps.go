package handlers

import (
	"net/http"
	"strconv"

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

	logs, err := h.appService.GetAppLogs(c.Request.Context(), appName, logType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"logs":  []string{},
			"error": "Failed to fetch logs",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs": logs,
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

	logs, err := h.appService.GetAppLogs(c.Request.Context(), appName, "lifecycle")
	if err != nil {
		c.String(http.StatusOK, "No lifecycle logs available")
		return
	}

	if len(logs) == 0 {
		c.String(http.StatusOK, "No lifecycle logs available")
	} else {
		// Join logs with newlines for text response
		response := ""
		for _, log := range logs {
			response += log + "\n"
		}
		c.String(http.StatusOK, response)
	}
}

// GetAppBackgroundLogs returns background process logs for an application
func (h *AppHandler) GetAppBackgroundLogs(c *gin.Context) {
	appName := c.Param("id")

	logs, err := h.appService.GetAppLogs(c.Request.Context(), appName, "background")
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
	structuredLogs := make([]map[string]interface{}, 0, len(logs))
	for _, log := range logs {
		structuredLogs = append(structuredLogs, map[string]interface{}{
			"level":   "log",
			"message": log,
			"source":  "background",
		})
	}

	c.JSON(http.StatusOK, structuredLogs)
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

	if result.IssueID != "" {
		response["data"] = gin.H{
			"issue_id": result.IssueID,
		}
	}

	c.JSON(http.StatusOK, response)
}
