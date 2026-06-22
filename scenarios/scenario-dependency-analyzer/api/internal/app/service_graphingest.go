package app

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"scenario-dependency-analyzer/internal/graphingest"
	"scenario-dependency-analyzer/internal/graphsweeper"
	"scenario-dependency-analyzer/internal/interfacegraph"
	"scenario-dependency-analyzer/internal/store"
)

// graphIngestService owns the unified-graph ingest pipeline and its sweeper.
// It is the single writer of the graph_edges store: a manual `graph rebuild`
// override and the default-ON freshness-gated background sweeper both flow
// through the same Ingestor.
type graphIngestService struct {
	ingestor      *graphingest.Ingestor
	sweeper       *graphsweeper.Sweeper
	store         *store.Store
	scenariosRoot string
	repoRoot      string
}

func newGraphIngestService(rt *Runtime) *graphIngestService {
	if rt == nil || rt.Store() == nil || rt.Analyzer() == nil {
		return nil
	}
	st := rt.Store()
	scenariosRoot := strings.TrimSpace(rt.Config().ScenariosDir)
	repoRoot := filepath.Dir(scenariosRoot)

	builder := interfacegraph.NewBuilder(
		interfacegraph.NewProtoHealthClient(nil, nil),
		interfacegraph.NewCodeFactsClient(nil, nil),
	)
	ingestor := graphingest.NewIngestor(builder, st, rt.Analyzer(), st)
	cfg := graphsweeper.LoadConfig(repoRoot, scenariosRoot)
	sweeper := graphsweeper.New(cfg, ingestor, st)

	return &graphIngestService{
		ingestor:      ingestor,
		sweeper:       sweeper,
		store:         st,
		scenariosRoot: scenariosRoot,
		repoRoot:      repoRoot,
	}
}

// RegisterRoutes mounts the rebuild + sweeper-status endpoints.
func (s *graphIngestService) RegisterRoutes(api gin.IRoutes) {
	if s == nil {
		return
	}
	api.POST("/graph/rebuild", s.handleRebuild)
	api.GET("/graph/sweeper/status", s.handleSweeperStatus)
}

// StartSweeper launches the background ingest loop (default-ON).
func (s *graphIngestService) StartSweeper(ctx context.Context) {
	if s == nil || s.sweeper == nil {
		return
	}
	go s.sweeper.RunLoop(ctx)
}

type rebuildRequest struct {
	Apply    bool   `json:"apply"`
	Scenario string `json:"scenario"`
}

func (s *graphIngestService) handleRebuild(c *gin.Context) {
	if s == nil || s.ingestor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "graph ingest service unavailable"})
		return
	}
	var req rebuildRequest
	_ = c.ShouldBindJSON(&req)
	// Query params override body for convenience from the CLI.
	if v := strings.TrimSpace(c.Query("scenario")); v != "" {
		req.Scenario = v
	}
	if c.Query("apply") != "" {
		req.Apply = c.Query("apply") == "true" || c.Query("apply") == "1"
	}

	ctx := c.Request.Context()
	if scenario := strings.TrimSpace(req.Scenario); scenario != "" {
		result, err := s.ingestor.IngestScenario(ctx, s.repoRoot, scenario, req.Apply)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error":    err.Error(),
				"scenario": scenario,
				"degraded": result.Degraded,
				"applied":  req.Apply,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"scope":           "scenario",
			"scenario":        scenario,
			"applied":         req.Apply,
			"edges_persisted": result.EdgesPersisted,
			"degraded":        result.Degraded,
		})
		return
	}

	report, err := s.ingestor.RebuildFleet(ctx, s.repoRoot, req.Apply)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "applied": req.Apply})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"scope":              "fleet",
		"applied":            req.Apply,
		"scenarios_analyzed": report.ScenariosAnalyzed,
		"edges_persisted":    report.EdgesPersisted,
		"scenario_edges":     report.ScenarioEdges,
		"resource_edges":     report.ResourceEdges,
		"degraded_sources":   report.DegradedSources,
		"build_stats":        report.BuildStats,
	})
}

func (s *graphIngestService) handleSweeperStatus(c *gin.Context) {
	if s == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "graph ingest service unavailable"})
		return
	}
	status := s.sweeper.Status()
	stats, statsErr := s.store.GraphEdgeStats()
	payload := gin.H{
		"enabled":       status.Enabled,
		"interval":      status.Interval.String(),
		"concurrency":   status.Concurrency,
		"cycle_budget":  status.CycleBudget.String(),
		"breaker_state": status.BreakerState,
		"last_run_at":   status.LastRunAt,
		"last_cycle":    status.LastCycle,
		"edges": gin.H{
			"total":     stats.TotalEdges,
			"scenario":  stats.ScenarioEdges,
			"resource":  stats.ResourceEdges,
			"stale":     stats.StaleEdges,
			"by_source": stats.BySource,
		},
	}
	if !stats.LastUpdated.IsZero() {
		payload["edges_last_updated"] = stats.LastUpdated
	}
	if statsErr != nil {
		payload["edges_error"] = statsErr.Error()
	}
	c.JSON(http.StatusOK, payload)
}
