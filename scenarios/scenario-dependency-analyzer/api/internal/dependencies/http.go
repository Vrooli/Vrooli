package dependencies

import (
	"net/http"

	"github.com/gin-gonic/gin"

	types "scenario-dependency-analyzer/internal/types"
)

// Service exposes stored dependency and impact operations needed by HTTP routes.
type Service interface {
	StoredDependencies(name string) (map[string][]types.ScenarioDependency, error)
	DependencyImpact(name string) (*types.DependencyImpactReport, error)
}

// HTTPHandler exposes dependency compatibility routes.
type HTTPHandler struct {
	dependencies Service
}

// NewHTTPHandler constructs a dependencies HTTP adapter.
func NewHTTPHandler(dependencies Service) *HTTPHandler {
	return &HTTPHandler{dependencies: dependencies}
}

// RegisterHTTPRoutes mounts dependency read routes under the API group.
func RegisterHTTPRoutes(api gin.IRoutes, dependencies Service) {
	handler := NewHTTPHandler(dependencies)
	api.GET("/scenarios/:scenario/dependencies", handler.GetDependencies)
	api.GET("/dependencies/:name/impact", handler.GetDependencyImpact)
}

// GetDependencies returns stored dependencies grouped by dependency type.
func (h *HTTPHandler) GetDependencies(c *gin.Context) {
	scenarioName := c.Param("scenario")

	stored := map[string][]types.ScenarioDependency{
		"resources":        {},
		"scenarios":        {},
		"shared_workflows": {},
	}

	if h.dependencies != nil {
		loaded, err := h.dependencies.StoredDependencies(scenarioName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		stored = loaded
	}

	c.JSON(http.StatusOK, gin.H{
		"scenario":         scenarioName,
		"resources":        stored["resources"],
		"scenarios":        stored["scenarios"],
		"shared_workflows": stored["shared_workflows"],
		"transitive_depth": 0,
	})
}

// GetDependencyImpact returns the scenarios affected by a dependency name.
func (h *HTTPHandler) GetDependencyImpact(c *gin.Context) {
	dependencyName := c.Param("name")

	if dependencyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dependency name is required"})
		return
	}

	if h.dependencies == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Dependency service unavailable"})
		return
	}

	report, err := h.dependencies.DependencyImpact(dependencyName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to analyze impact: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}
