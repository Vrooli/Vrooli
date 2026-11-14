package optimization

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/app/services"
	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

// HTTPHandler exposes optimization recommendation routes.
type HTTPHandler struct {
	optimization services.OptimizationService
}

// NewHTTPHandler constructs an optimization HTTP adapter.
func NewHTTPHandler(optimization services.OptimizationService) *HTTPHandler {
	return &HTTPHandler{optimization: optimization}
}

// RegisterHTTPRoutes mounts optimization routes under the API group.
func RegisterHTTPRoutes(api gin.IRoutes, optimization services.OptimizationService) {
	handler := NewHTTPHandler(optimization)
	api.POST("/optimize", handler.Optimize)
	// A distinct route prevents an older server from ignoring a selector and
	// applying every recommendation.
	api.POST("/optimize-scenarios", handler.OptimizeScenarios)
}

func (h *HTTPHandler) OptimizeScenarios(c *gin.Context) { h.optimize(c, true) }

// Optimize runs optimization recommendations for one scenario or all scenarios.
func (h *HTTPHandler) Optimize(c *gin.Context) {
	h.optimize(c, false)
}

func (h *HTTPHandler) optimize(c *gin.Context, requireSelection bool) {
	var req types.OptimizationRequest
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if requireSelection && (len(req.OnlyScenarios) == 0 || strings.TrimSpace(req.Scenario) == "" || req.Scenario == "all") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scenario selection requires one target scenario and at least one dependency"})
		return
	}
	for _, name := range req.OnlyScenarios {
		if strings.TrimSpace(name) == "" || strings.ContainsAny(name, "/\\") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "selected dependency names must be nonempty scenario names"})
			return
		}
	}
	results, err := h.optimization.RunOptimization(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"results":      results,
		"generated_at": time.Now().UTC(),
	})
}
