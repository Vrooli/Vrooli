package graph

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/coreset"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/interfacegraph"
	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

// Service exposes graph operations needed by the HTTP adapter.
type Service interface {
	GenerateGraph(graphType string) (*types.DependencyGraph, error)
	GraphCentrality(coreSeeds []string, scenario string) (*types.GraphCentralityReport, error)
}

// HTTPHandler exposes dependency graph and drift REST compatibility routes.
type HTTPHandler struct {
	graphs       Service
	scenariosDir func() string
}

// NewHTTPHandler constructs a graph HTTP adapter.
func NewHTTPHandler(graphs Service, scenariosDir func() string) *HTTPHandler {
	return &HTTPHandler{
		graphs:       graphs,
		scenariosDir: scenariosDir,
	}
}

// RegisterHTTPRoutes mounts graph reporting routes under the API group.
func RegisterHTTPRoutes(api gin.IRoutes, graphs Service, scenariosDir func() string) {
	handler := NewHTTPHandler(graphs, scenariosDir)
	api.GET("/graph/centrality", handler.GetGraphCentrality)
	api.GET("/graph/actual", handler.GetActualInterfaceGraph)
	api.GET("/drift", handler.GetDependencyDrift)
	api.GET("/graph/:type", handler.GetGraph)
	api.GET("/graph/:type/cycles", handler.DetectCycles)
}

// GetGraph returns the requested dependency graph type.
func (h *HTTPHandler) GetGraph(c *gin.Context) {
	graphType := c.Param("type")

	if !validGraphType(graphType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid graph type"})
		return
	}

	graph, err := h.graphs.GenerateGraph(graphType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, graph)
}

// GetGraphCentrality returns reverse-dependency centrality metrics.
func (h *HTTPHandler) GetGraphCentrality(c *gin.Context) {
	if h.graphs == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Graph service unavailable"})
		return
	}

	coreSet := coreset.Compute(h.resolveScenariosDir())
	report, err := h.graphs.GraphCentrality(coreSet.Seed, c.Query("scenario"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate graph centrality: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetActualInterfaceGraph keeps the REST compatibility route for the Connect-backed graph.
func (h *HTTPHandler) GetActualInterfaceGraph(c *gin.Context) {
	graph, err := h.describeInterfaceGraph(c)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, graph)
}

// GetDependencyDrift returns interface graph drift against declared dependencies.
func (h *HTTPHandler) GetDependencyDrift(c *gin.Context) {
	builder := interfacegraph.NewBuilder(
		interfacegraph.NewProtoHealthClient(nil, nil),
		interfacegraph.NewCodeFactsClient(nil, nil),
	)
	detector := interfacegraph.NewDriftDetector(builder, h.resolveScenariosDir())
	report, err := detector.Detect(c.Request.Context(), h.interfaceGraphRequest(c))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

// DetectCycles performs circular dependency detection on a generated graph.
func (h *HTTPHandler) DetectCycles(c *gin.Context) {
	graphType := c.Param("type")

	if !validGraphType(graphType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid graph type. Use: resource, scenario, or combined"})
		return
	}

	graph, err := h.graphs.GenerateGraph(graphType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate graph: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, DetectCycles(graph))
}

func (h *HTTPHandler) describeInterfaceGraph(c *gin.Context) (interfacegraph.Graph, error) {
	builder := interfacegraph.NewBuilder(
		interfacegraph.NewProtoHealthClient(nil, nil),
		interfacegraph.NewCodeFactsClient(nil, nil),
	)
	return builder.Build(c.Request.Context(), h.interfaceGraphRequest(c))
}

func (h *HTTPHandler) interfaceGraphRequest(c *gin.Context) interfacegraph.BuildRequest {
	scenarios := c.QueryArray("scenario")
	if csv := strings.TrimSpace(c.Query("scenarios")); csv != "" {
		for _, part := range strings.Split(csv, ",") {
			if part = strings.TrimSpace(part); part != "" {
				scenarios = append(scenarios, part)
			}
		}
	}
	return interfacegraph.BuildRequest{
		Scenarios:       scenarios,
		Limit:           parseNonNegativeInt32(c.Query("limit")),
		RepoRoot:        filepath.Dir(h.resolveScenariosDir()),
		StabilityFilter: c.Query("stability"),
		LanguageFilter:  c.QueryArray("language"),
	}
}

func (h *HTTPHandler) resolveScenariosDir() string {
	if h.scenariosDir == nil {
		return ""
	}
	return strings.TrimSpace(h.scenariosDir())
}

func parseNonNegativeInt32(value string) int32 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil || parsed < 0 {
		return 0
	}
	return int32(parsed)
}

func validGraphType(graphType string) bool {
	switch graphType {
	case "resource", "scenario", "combined":
		return true
	default:
		return false
	}
}
