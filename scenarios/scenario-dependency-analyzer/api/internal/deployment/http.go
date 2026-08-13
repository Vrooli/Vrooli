package deployment

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	types "scenario-dependency-analyzer/internal/types"
)

// Reporter exposes computed deployment reports to presentation adapters.
type Reporter interface {
	GetDeploymentReport(name string, refresh bool) (*types.DeploymentAnalysisReport, error)
}

// HTTPHandler exposes deployment report routes.
type HTTPHandler struct {
	reporter Reporter
}

// NewHTTPHandler constructs a deployment HTTP adapter.
func NewHTTPHandler(reporter Reporter) *HTTPHandler {
	return &HTTPHandler{reporter: reporter}
}

// RegisterHTTPRoutes mounts deployment reporting routes under the API group.
func RegisterHTTPRoutes(api gin.IRoutes, reporter Reporter) {
	handler := NewHTTPHandler(reporter)
	api.GET("/scenarios/:scenario/deployment", handler.GetDeploymentReport)
	api.GET("/scenarios/:scenario/bundle/manifest", handler.GetBundleManifest)
	api.GET("/scenarios/:scenario/dag/export", handler.ExportDAG)
	api.GET("/targets/:target/dag/export", handler.ExportTargetDAG)
}

// GetDeploymentReport returns the computed deployment readiness report.
func (h *HTTPHandler) GetDeploymentReport(c *gin.Context) {
	report, err := h.reporter.GetDeploymentReport(c.Param("scenario"), parseRefresh(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

// ExportDAG returns a JSON dependency DAG derived from the deployment report.
func (h *HTTPHandler) ExportDAG(c *gin.Context) {
	scenarioName := c.Param("scenario")
	if strings.Contains(scenarioName, ":") {
		h.exportTargetDAG(c, scenarioName)
		return
	}
	recursive := c.DefaultQuery("recursive", "true") == "true"
	format := c.DefaultQuery("format", "json")

	if format != "json" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only JSON format is currently supported"})
		return
	}

	report, err := h.reporter.GetDeploymentReport(scenarioName, parseRefresh(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !recursive {
		for i := range report.Dependencies {
			report.Dependencies[i].Children = nil
		}
	}

	response := gin.H{
		"scenario":     scenarioName,
		"recursive":    recursive,
		"generated_at": report.GeneratedAt,
		"dag":          report.Dependencies,
	}
	if report.MetadataGaps != nil {
		response["metadata_gaps"] = report.MetadataGaps
	}

	c.JSON(http.StatusOK, response)
}

// ExportTargetDAG is the target-aware form of the DAG export. The legacy
// /scenarios/:scenario route remains supported for bare scenario slugs.
func (h *HTTPHandler) ExportTargetDAG(c *gin.Context) {
	h.exportTargetDAG(c, c.Param("target"))
}

func (h *HTTPHandler) exportTargetDAG(c *gin.Context, expression string) {
	reporter, ok := h.reporter.(interface {
		ExportTargetDAG(string, bool, bool) (*types.TargetDAGResponse, error)
	})
	if !ok {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "target DAG export is unavailable"})
		return
	}
	if c.DefaultQuery("format", "json") != "json" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only JSON format is currently supported"})
		return
	}
	response, err := reporter.ExportTargetDAG(expression, c.DefaultQuery("recursive", "true") == "true", parseRefresh(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
}

// GetBundleManifest returns the bundle manifest embedded in the deployment report.
func (h *HTTPHandler) GetBundleManifest(c *gin.Context) {
	scenarioName := c.Param("scenario")

	report, err := h.reporter.GetDeploymentReport(scenarioName, parseRefresh(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"scenario":  scenarioName,
		"generated": report.GeneratedAt,
		"manifest":  report.BundleManifest,
	})
}

func parseRefresh(c *gin.Context) bool {
	raw := strings.TrimSpace(c.Query("refresh"))
	if raw == "" {
		return false
	}
	parsed, err := strconv.ParseBool(raw)
	return err == nil && parsed
}
