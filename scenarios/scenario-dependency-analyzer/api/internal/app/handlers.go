package app

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"scenario-dependency-analyzer/internal/app/services"
	"scenario-dependency-analyzer/internal/interfacegraph"

	"github.com/gin-gonic/gin"

	coresetpkg "scenario-dependency-analyzer/internal/coreset"
	types "scenario-dependency-analyzer/internal/types"
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

// getCoreSet returns the reflexive core set (9-seed ∪ transitive Required
// closure) and the trusted-base subset, computed fresh from disk. This path is
// database-free by design, so the core set the Baseline Modes decision tree
// consumes is available even when postgres is down.
func (h *handler) getCoreSet(c *gin.Context) {
	c.JSON(http.StatusOK, coresetpkg.Compute(h.scenariosDir()))
}

func (h *handler) analysisHealth(c *gin.Context) {
	graphSvc := h.graphService()
	timestamp := time.Now().UTC().Format(time.RFC3339)
	if _, err := graphSvc.GenerateGraph("combined"); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "unhealthy",
			"service":   "scenario-dependency-analyzer-api",
			"timestamp": timestamp,
			"readiness": false,
			"version":   "1.0.0",
			"error":     "Analysis capability test failed",
		})
		return
	}

	payload := gin.H{
		"status":       "healthy",
		"service":      "scenario-dependency-analyzer-api",
		"timestamp":    timestamp,
		"readiness":    true,
		"version":      "1.0.0",
		"capabilities": []string{"dependency_analysis", "graph_generation"},
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

func (h *handler) analyzeScenario(c *gin.Context) {
	scenarioName := c.Param("scenario")

	if scenarioName == "all" {
		analysisSvc := h.analysisService()
		results, err := analysisSvc.AnalyzeAllScenarios()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, results)
		return
	}

	analysisSvc := h.analysisService()
	result, err := analysisSvc.AnalyzeScenario(scenarioName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *handler) getDependencies(c *gin.Context) {
	scenarioName := c.Param("scenario")

	stored := map[string][]types.ScenarioDependency{
		"resources":        {},
		"scenarios":        {},
		"shared_workflows": {},
	}

	if depSvc := h.dependencyService(); depSvc != nil {
		loaded, err := depSvc.StoredDependencies(scenarioName)
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

func (h *handler) getGraph(c *gin.Context) {
	graphType := c.Param("type")

	if !contains([]string{"resource", "scenario", "combined"}, graphType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid graph type"})
		return
	}

	graphSvc := h.graphService()
	graph, err := graphSvc.GenerateGraph(graphType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, graph)
}

func (h *handler) getGraphCentrality(c *gin.Context) {
	graphSvc := h.graphService()
	if graphSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Graph service unavailable"})
		return
	}

	coreSet := coresetpkg.Compute(h.scenariosDir())
	report, err := graphSvc.GraphCentrality(coreSet.Seed, c.Query("scenario"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate graph centrality: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

func (h *handler) getActualInterfaceGraph(c *gin.Context) {
	graph, err := h.describeInterfaceGraph(c.Request.Context(), h.interfaceGraphRequest(c))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, graph)
}

func (h *handler) getDependencyDrift(c *gin.Context) {
	builder := interfacegraph.NewBuilder(
		interfacegraph.NewProtoHealthClient(nil, nil),
		interfacegraph.NewCodeFactsClient(nil, nil),
	)
	detector := interfacegraph.NewDriftDetector(builder, h.scenariosDir())
	report, err := detector.Detect(c.Request.Context(), h.interfaceGraphRequest(c))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *handler) interfaceGraphRequest(c *gin.Context) interfacegraph.BuildRequest {
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
		RepoRoot:        filepath.Dir(h.scenariosDir()),
		StabilityFilter: c.Query("stability"),
		LanguageFilter:  c.QueryArray("language"),
	}
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

func (h *handler) detectCycles(c *gin.Context) {
	graphType := c.Param("type")

	if !contains([]string{"resource", "scenario", "combined"}, graphType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid graph type. Use: resource, scenario, or combined"})
		return
	}

	graphSvc := h.graphService()
	graph, err := graphSvc.GenerateGraph(graphType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate graph: " + err.Error()})
		return
	}

	result := detectCycles(graph)
	c.JSON(http.StatusOK, result)
}

func (h *handler) getDependencyImpact(c *gin.Context) {
	dependencyName := c.Param("name")

	if dependencyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dependency name is required"})
		return
	}

	depSvc := h.dependencyService()
	if depSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Dependency service unavailable"})
		return
	}

	report, err := depSvc.DependencyImpact(dependencyName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to analyze impact: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

func (h *handler) analyzeProposed(c *gin.Context) {
	var req types.ProposedScenarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proposalSvc := h.proposalService()
	result, err := proposalSvc.AnalyzeProposedScenario(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *handler) optimize(c *gin.Context) {
	var req types.OptimizationRequest
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	optimizationSvc := h.optimizationService()
	results, err := optimizationSvc.RunOptimization(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"results":      results,
		"generated_at": time.Now().UTC(),
	})
}

func (h *handler) listScenarios(c *gin.Context) {
	svc := h.scenarioService()
	summaries, err := svc.ListScenarios()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summaries)
}

func (h *handler) getScenarioDetail(c *gin.Context) {
	svc := h.scenarioService()
	detail, err := svc.GetScenarioDetail(c.Param("scenario"))
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

func (h *handler) getDeploymentReport(c *gin.Context) {
	deploymentSvc := h.deploymentService()
	report, err := deploymentSvc.GetDeploymentReport(c.Param("scenario"), parseRefresh(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *handler) scanScenario(c *gin.Context) {
	scenarioName := c.Param("scenario")
	var req types.ScanRequest
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	scanSvc := h.scanService()
	result, err := scanSvc.ScanScenario(scenarioName, req)
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

func (h *handler) exportDAG(c *gin.Context) {
	scenarioName := c.Param("scenario")
	recursive := c.DefaultQuery("recursive", "true") == "true"
	format := c.DefaultQuery("format", "json")

	if format != "json" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only JSON format is currently supported"})
		return
	}

	deploymentSvc := h.deploymentService()
	report, err := deploymentSvc.GetDeploymentReport(scenarioName, parseRefresh(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If not recursive, flatten the tree to top-level only
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

	// Include metadata gaps if present
	if report.MetadataGaps != nil {
		response["metadata_gaps"] = report.MetadataGaps
	}

	c.JSON(http.StatusOK, response)
}

func (h *handler) getBundleManifest(c *gin.Context) {
	scenarioName := c.Param("scenario")

	deploymentSvc := h.deploymentService()
	report, err := deploymentSvc.GetDeploymentReport(scenarioName, parseRefresh(c))
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
