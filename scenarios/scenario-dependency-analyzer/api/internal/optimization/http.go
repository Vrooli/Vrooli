package optimization

import (
	"errors"
	"io"
	"net/http"
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
}

// Optimize runs optimization recommendations for one scenario or all scenarios.
func (h *HTTPHandler) Optimize(c *gin.Context) {
	var req types.OptimizationRequest
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
