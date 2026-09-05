package catalog

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

// ScenarioService exposes the catalog read operations needed by the HTTP adapter.
type ScenarioService interface {
	ListScenarios() ([]types.ScenarioSummary, error)
	GetScenarioDetail(name string) (*types.ScenarioDetailResponse, error)
}

// Handler exposes scenario catalog HTTP routes.
type Handler struct {
	scenarios ScenarioService
}

// NewHandler constructs a catalog HTTP adapter.
func NewHandler(scenarios ScenarioService) *Handler {
	return &Handler{scenarios: scenarios}
}

// RegisterRoutes mounts catalog routes under the provided API group.
func RegisterRoutes(api gin.IRoutes, scenarios ScenarioService) {
	handler := NewHandler(scenarios)
	api.GET("/scenarios", handler.ListScenarios)
	api.GET("/scenarios/:scenario", handler.GetScenarioDetail)
}

// ListScenarios returns the scenario catalog summaries used by the UI and CLI.
func (h *Handler) ListScenarios(c *gin.Context) {
	summaries, err := h.scenarios.ListScenarios()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summaries)
}

// GetScenarioDetail returns the detail model for one scenario catalog entry.
func (h *Handler) GetScenarioDetail(c *gin.Context) {
	detail, err := h.scenarios.GetScenarioDetail(c.Param("scenario"))
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}
