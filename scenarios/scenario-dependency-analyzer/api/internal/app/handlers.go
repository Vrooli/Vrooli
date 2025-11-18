package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"scenario-dependency-analyzer/internal/app/services"
	appconfig "scenario-dependency-analyzer/internal/config"
	types "scenario-dependency-analyzer/internal/types"
)

type handler struct {
	runtime  *Runtime
	services services.Registry
}

func newHandler(rt *Runtime) *handler {
	var reg services.Registry
	if rt != nil && rt.Analyzer() != nil {
		reg = rt.Analyzer().Services()
	}
	return &handler{runtime: rt, services: reg}
}

func (h *handler) dbConn() *sql.DB {
	if h != nil && h.runtime != nil && h.runtime.DB() != nil {
		return h.runtime.DB()
	}
	return db
}

func (h *handler) analysisService() services.AnalysisService {
	if h != nil && h.services.Analysis != nil {
		return h.services.Analysis
	}
	return analysisService{}
}

func (h *handler) scanService() services.ScanService {
	if h != nil && h.services.Scan != nil {
		return h.services.Scan
	}
	return &scanService{analysis: h.analysisService()}
}

func (h *handler) optimizationService() services.OptimizationService {
	if h != nil && h.services.Optimization != nil {
		return h.services.Optimization
	}
	return &optimizationService{analysis: h.analysisService()}
}

func (h *handler) graphService() services.GraphService {
	if h != nil && h.services.Graph != nil {
		return h.services.Graph
	}
	return &graphService{analyzer: analyzerInstance()}
}

func (h *handler) scenarioService() services.ScenarioService {
	if h != nil && h.services.Scenarios != nil {
		return h.services.Scenarios
	}
	return &scenarioService{cfg: appconfig.Load(), store: currentStore()}
}

func (h *handler) deploymentService() services.DeploymentService {
	if h != nil && h.services.Deployment != nil {
		return h.services.Deployment
	}
	return &deploymentService{cfg: appconfig.Load()}
}

func (h *handler) proposalService() services.ProposalService {
	if h != nil && h.services.Proposal != nil {
		return h.services.Proposal
	}
	return &proposalService{}
}

func (h *handler) health(c *gin.Context) {
	dbConnected := false
	var dbLatencyMs float64
	if conn := h.dbConn(); conn != nil {
		start := time.Now()
		err := conn.Ping()
		dbLatencyMs = float64(time.Since(start).Milliseconds())
		dbConnected = err == nil
	}

	status := "healthy"
	if !dbConnected {
		status = "degraded"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    status,
		"timestamp": time.Now(),
		"service":   "scenario-dependency-analyzer",
		"readiness": dbConnected,
		"dependencies": gin.H{
			"database": gin.H{
				"connected":  dbConnected,
				"latency_ms": dbLatencyMs,
				"error":      nil,
			},
		},
	})
}

func (h *handler) analysisHealth(c *gin.Context) {
	graphSvc := h.graphService()
	if _, err := graphSvc.GenerateGraph("combined"); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "Analysis capability test failed",
		})
		return
	}

	payload := gin.H{
		"status":       "healthy",
		"capabilities": []string{"dependency_analysis", "graph_generation"},
	}

	metrics, metricsErr := collectAnalysisMetrics()
	for k, v := range metrics {
		payload[k] = v
	}

	if metricsErr != nil {
		payload["status"] = "degraded"
		payload["error"] = metricsErr.Error()
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

	conn := h.dbConn()
	if conn == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database unavailable"})
		return
	}
	rows, err := conn.Query(`
        SELECT id, scenario_name, dependency_type, dependency_name, required, purpose, access_method, configuration, discovered_at, last_verified
        FROM scenario_dependencies
        WHERE scenario_name = $1
        ORDER BY dependency_type, dependency_name`, scenarioName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var dependencies []types.ScenarioDependency
	for rows.Next() {
		var dep types.ScenarioDependency
		var configJSON string

		err := rows.Scan(&dep.ID, &dep.ScenarioName, &dep.DependencyType, &dep.DependencyName,
			&dep.Required, &dep.Purpose, &dep.AccessMethod, &configJSON, &dep.DiscoveredAt, &dep.LastVerified)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(configJSON), &dep.Configuration)
		dependencies = append(dependencies, dep)
	}

	response := map[string][]types.ScenarioDependency{
		"resources":        {},
		"scenarios":        {},
		"shared_workflows": {},
	}

	for _, dep := range dependencies {
		switch dep.DependencyType {
		case "resource":
			response["resources"] = append(response["resources"], dep)
		case "scenario":
			response["scenarios"] = append(response["scenarios"], dep)
		case "shared_workflow":
			response["shared_workflows"] = append(response["shared_workflows"], dep)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"scenario":         scenarioName,
		"resources":        response["resources"],
		"scenarios":        response["scenarios"],
		"shared_workflows": response["shared_workflows"],
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
	report, err := deploymentSvc.GetDeploymentReport(c.Param("scenario"))
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
