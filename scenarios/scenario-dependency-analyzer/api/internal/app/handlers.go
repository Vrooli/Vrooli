package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	appconfig "scenario-dependency-analyzer/internal/config"
	types "scenario-dependency-analyzer/internal/types"
)

func healthHandler(c *gin.Context) {
	dbConnected := false
	var dbLatencyMs float64
	if db != nil {
		start := time.Now()
		err := db.Ping()
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

func analyzeScenarioHandler(c *gin.Context) {
	scenarioName := c.Param("scenario")

	if scenarioName == "all" {
		results, err := analyzeAllScenarios()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, results)
		return
	}

	result, err := analyzeScenario(scenarioName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func getDependenciesHandler(c *gin.Context) {
	scenarioName := c.Param("scenario")

	rows, err := db.Query(`
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

func getGraphHandler(c *gin.Context) {
	graphType := c.Param("type")

	if !contains([]string{"resource", "scenario", "combined"}, graphType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid graph type"})
		return
	}

	graph, err := generateDependencyGraph(graphType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, graph)
}

func analyzeProposedHandler(c *gin.Context) {
	var req types.ProposedScenarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := analyzeProposedScenario(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func optimizeHandler(c *gin.Context) {
	var req types.OptimizationRequest
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	scenario := strings.TrimSpace(req.Scenario)
	if scenario == "" {
		scenario = "all"
	}
	var targets []string
	if scenario == "all" {
		names, err := listScenarioNames()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		targets = names
	} else {
		if !isKnownScenario(scenario) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("scenario %s not found", scenario)})
			return
		}
		targets = []string{scenario}
	}
	results := make(map[string]*types.OptimizationResult, len(targets))
	for _, target := range targets {
		result, err := runScenarioOptimization(target, req)
		if err != nil {
			results[target] = &types.OptimizationResult{
				Scenario:          target,
				Recommendations:   nil,
				Summary:           types.OptimizationSummary{},
				Applied:           false,
				AnalysisTimestamp: time.Now().UTC(),
				Error:             err.Error(),
			}
			continue
		}
		results[target] = result
	}
	c.JSON(http.StatusOK, gin.H{
		"results":      results,
		"generated_at": time.Now().UTC(),
	})
}

func listScenariosHandler(c *gin.Context) {
	metadata, err := loadScenarioMetadataMap()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	scenariosDir := appconfig.Load().ScenariosDir
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	summaries := []types.ScenarioSummary{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		summary, ok := metadata[name]
		if !ok {
			scenarioPath := filepath.Join(scenariosDir, name)
			cfg, err := loadServiceConfigFromFile(scenarioPath)
			if err != nil {
				continue
			}
			summary = types.ScenarioSummary{
				Name:        name,
				DisplayName: cfg.Service.DisplayName,
				Description: cfg.Service.Description,
				Tags:        cfg.Service.Tags,
			}
		}
		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i, j int) bool { return summaries[i].Name < summaries[j].Name })
	c.JSON(http.StatusOK, summaries)
}

func getScenarioDetailHandler(c *gin.Context) {
	scenarioName := c.Param("scenario")
	envCfg := appconfig.Load()
	scenarioPath := filepath.Join(envCfg.ScenariosDir, scenarioName)
	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	stored, err := loadStoredDependencies(scenarioName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	optRecs, err := loadOptimizationRecommendations(scenarioName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	declaredResources := resolvedResourceMap(cfg)
	declaredScenarios := cfg.Dependencies.Scenarios
	if declaredScenarios == nil {
		declaredScenarios = map[string]types.ScenarioDependencySpec{}
	}

	resourceDiff := buildResourceDiff(declaredResources, filterDetectedDependencies(stored["resources"]))
	scenarioDiff := buildScenarioDiff(declaredScenarios, filterDetectedDependencies(stored["scenarios"]))

	var lastScanned *time.Time
	if db != nil {
		row := db.QueryRow("SELECT last_scanned FROM scenario_metadata WHERE scenario_name = $1", scenarioName)
		var ts sql.NullTime
		if err := row.Scan(&ts); err == nil && ts.Valid {
			lastScanned = &ts.Time
		}
	}

	detail := types.ScenarioDetailResponse{
		Scenario:                    scenarioName,
		DisplayName:                 cfg.Service.DisplayName,
		Description:                 cfg.Service.Description,
		LastScanned:                 lastScanned,
		DeclaredResources:           declaredResources,
		DeclaredScenarios:           declaredScenarios,
		StoredDependencies:          stored,
		ResourceDiff:                resourceDiff,
		ScenarioDiff:                scenarioDiff,
		OptimizationRecommendations: optRecs,
	}

	if report := buildDeploymentReport(scenarioName, scenarioPath, envCfg.ScenariosDir, cfg); report != nil {
		detail.DeploymentReport = report
	}

	c.JSON(http.StatusOK, detail)
}

func getDeploymentReportHandler(c *gin.Context) {
	scenarioName := c.Param("scenario")
	envCfg := appconfig.Load()
	scenarioPath := filepath.Join(envCfg.ScenariosDir, scenarioName)
	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	report, err := loadPersistedDeploymentReport(scenarioPath)
	if err != nil {
		report = buildDeploymentReport(scenarioName, scenarioPath, envCfg.ScenariosDir, cfg)
		if report != nil {
			if persistErr := persistDeploymentReport(scenarioPath, report); persistErr != nil {
				log.Printf("Warning: failed to persist deployment report for %s: %v", scenarioName, persistErr)
			}
		}
	}

	if report == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to build deployment report"})
		return
	}

	c.JSON(http.StatusOK, report)
}

func scanScenarioHandler(c *gin.Context) {
	scenarioName := c.Param("scenario")
	var req types.ScanRequest
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	analysis, err := analyzeScenario(scenarioName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	applyResources := req.ApplyResources || req.Apply
	applyScenarios := req.ApplyScenarios || req.Apply
	var applySummary map[string]interface{}
	applied := false
	if applyResources || applyScenarios {
		applySummary, err = applyDetectedDiffs(scenarioName, analysis, applyResources, applyScenarios)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "analysis": analysis})
			return
		}
		if changed, ok := applySummary["changed"].(bool); ok && changed {
			applied = true
			analysis, err = analyzeScenario(scenarioName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis":      analysis,
		"applied":       applied,
		"apply_summary": applySummary,
	})
}
