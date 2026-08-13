package app

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"scenario-dependency-analyzer/internal/app/services"

	"github.com/gin-gonic/gin"
)

// handler routes HTTP requests to the appropriate service implementations.
// Services are resolved once at construction to avoid repeated nil checks.
type handler struct {
	runtime  *Runtime
	services services.Registry
	dbHandle *sql.DB
}

// newHandler constructs a handler with services resolved from the runtime.
// If no runtime is available, fallback services are created automatically.
func newHandler(rt *Runtime) *handler {
	h := &handler{runtime: rt}

	// Resolve database handle
	if rt != nil && rt.DB() != nil {
		h.dbHandle = rt.DB()
	} else {
		h.dbHandle = db
	}

	// Resolve services from runtime or create fallbacks
	if rt != nil && rt.Analyzer() != nil {
		h.services = rt.Analyzer().Services()
	} else {
		h.services = newFallbackServices()
	}

	return h
}

// newFallbackServices creates a services registry using global state.
// Used when no runtime is configured (e.g., testing, legacy code paths).
func newFallbackServices() services.Registry {
	analyzer := analyzerInstance()
	workspace := newScenarioWorkspace(loadConfig())
	analysis := &analysisService{analyzer: analyzer}
	dependencies := defaultDependencyService()
	opt := &optimizationService{
		analysis:     analysis,
		workspace:    workspace,
		detector:     dependencies.detector,
		dependencies: dependencies,
		store:        dependencies.store,
	}
	return services.Registry{
		Analysis:     analysis,
		Scan:         &scanService{analysis: analysis, dependencies: dependencies},
		Graph:        &graphService{analyzer: analyzer},
		Optimization: opt,
		Scenarios:    &scenarioService{workspace: workspace, store: currentStore()},
		Dependencies: dependencies,
		Deployment:   &deploymentService{workspace: workspace},
		Proposal:     &proposalService{},
	}
}

func (h *handler) analysisService() services.AnalysisService         { return h.services.Analysis }
func (h *handler) scanService() services.ScanService                 { return h.services.Scan }
func (h *handler) optimizationService() services.OptimizationService { return h.services.Optimization }
func (h *handler) graphService() services.GraphService               { return h.services.Graph }
func (h *handler) scenarioService() services.ScenarioService         { return h.services.Scenarios }
func (h *handler) dependencyService() services.DependencyService     { return h.services.Dependencies }
func (h *handler) deploymentService() services.DeploymentService     { return h.services.Deployment }
func (h *handler) proposalService() services.ProposalService         { return h.services.Proposal }

// scenariosDir resolves the scenarios root for fresh-from-disk computation,
// preferring the active runtime config and falling back to a fresh load.
func (h *handler) scenariosDir() string {
	if h.runtime != nil {
		if dir := strings.TrimSpace(h.runtime.Config().ScenariosDir); dir != "" {
			return dir
		}
	}
	return loadConfig().ScenariosDir
}

func (h *handler) analysisHealth(c *gin.Context) {
	graphSvc := h.graphService()
	timestamp := time.Now().UTC().Format(time.RFC3339)
	if _, err := graphSvc.GenerateGraph("combined"); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "unhealthy",
			"service":   "scenario-dependency-analyzer-api",
			"timestamp": timestamp,
			// The combined graph is generated as part of this health probe, so
			// this is the materialization timestamp for the three federated
			// graph/resource search leaves, not merely the HTTP check time.
			"last_indexed_at": timestamp,
			"readiness":       false,
			"version":         "1.0.0",
			"error":           "Analysis capability test failed",
		})
		return
	}

	payload := gin.H{
		"status":          "healthy",
		"service":         "scenario-dependency-analyzer-api",
		"timestamp":       timestamp,
		"last_indexed_at": timestamp,
		"readiness":       true,
		"version":         "1.0.0",
		"capabilities":    []string{"dependency_analysis", "graph_generation"},
	}

	if depSvc := h.dependencyService(); depSvc != nil {
		metrics, metricsErr := depSvc.AnalysisMetrics()
		for k, v := range metrics {
			payload[k] = v
		}

		if metricsErr != nil {
			payload["status"] = "degraded"
			payload["error"] = metricsErr.Error()
		}
	} else {
		payload["status"] = "degraded"
		payload["error"] = "dependency store unavailable"
	}

	c.JSON(http.StatusOK, payload)
}
