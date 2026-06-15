package analysis

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"scenario-dependency-analyzer/internal/app/services"
	types "scenario-dependency-analyzer/internal/types"
)

// HTTPHandler exposes scenario analysis and scan/apply compatibility routes.
type HTTPHandler struct {
	analysis services.AnalysisService
	scans    services.ScanService
}

// NewHTTPHandler constructs an analysis HTTP adapter.
func NewHTTPHandler(analysis services.AnalysisService, scans services.ScanService) *HTTPHandler {
	return &HTTPHandler{analysis: analysis, scans: scans}
}

// RegisterHTTPRoutes mounts analysis and scan routes under the API group.
func RegisterHTTPRoutes(api gin.IRoutes, analysis services.AnalysisService, scans services.ScanService) {
	handler := NewHTTPHandler(analysis, scans)
	api.GET("/analyze/:scenario", handler.AnalyzeScenario)
	api.POST("/scenarios/:scenario/scan", handler.ScanScenario)
}

// AnalyzeScenario returns dependency analysis for one scenario or all scenarios.
func (h *HTTPHandler) AnalyzeScenario(c *gin.Context) {
	scenarioName := c.Param("scenario")

	if scenarioName == "all" {
		results, err := h.analysis.AnalyzeAllScenarios()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, results)
		return
	}

	result, err := h.analysis.AnalyzeScenario(scenarioName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ScanScenario runs a dependency scan and optionally applies detected diffs.
func (h *HTTPHandler) ScanScenario(c *gin.Context) {
	scenarioName := c.Param("scenario")
	var req types.ScanRequest
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	result, err := h.scans.ScanScenario(scenarioName, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"analysis":      result.Analysis,
		"applied":       result.Applied,
		"apply_summary": result.ApplySummary,
	})
}
