package proposal

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"scenario-dependency-analyzer/internal/app/services"
	types "scenario-dependency-analyzer/internal/types"
)

// HTTPHandler exposes proposed scenario analysis routes.
type HTTPHandler struct {
	proposals services.ProposalService
}

// NewHTTPHandler constructs a proposal HTTP adapter.
func NewHTTPHandler(proposals services.ProposalService) *HTTPHandler {
	return &HTTPHandler{proposals: proposals}
}

// RegisterHTTPRoutes mounts proposal analysis routes under the API group.
func RegisterHTTPRoutes(api gin.IRoutes, proposals services.ProposalService) {
	handler := NewHTTPHandler(proposals)
	api.POST("/analyze/proposed", handler.AnalyzeProposed)
}

// AnalyzeProposed analyzes a proposed scenario before it exists on disk.
func (h *HTTPHandler) AnalyzeProposed(c *gin.Context) {
	var req types.ProposedScenarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.proposals.AnalyzeProposedScenario(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
