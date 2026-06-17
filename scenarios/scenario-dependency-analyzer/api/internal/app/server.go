package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/maturity-go/assessment"
	repocontract "github.com/vrooli/repo-contract-go"
	searchregister "github.com/vrooli/searchregister-go"

	"scenario-dependency-analyzer/internal/aisearch"
	analysisapi "scenario-dependency-analyzer/internal/analysis"
	"scenario-dependency-analyzer/internal/catalog"
	appconfig "scenario-dependency-analyzer/internal/config"
	coresetapi "scenario-dependency-analyzer/internal/coreset"
	dependenciesapi "scenario-dependency-analyzer/internal/dependencies"
	dependencygovernanceapi "scenario-dependency-analyzer/internal/dependencygovernance"
	dependencyhealthapi "scenario-dependency-analyzer/internal/dependencyhealth"
	deploymentapi "scenario-dependency-analyzer/internal/deployment"
	graphapi "scenario-dependency-analyzer/internal/graph"
	optimizationapi "scenario-dependency-analyzer/internal/optimization"
	proposalapi "scenario-dependency-analyzer/internal/proposal"
)

// Run boots the HTTP API using the provided configuration and database connection.
func Run(cfg appconfig.Config, dbConn *sql.DB) error {
	db = dbConn
	rt := ensureRuntime(cfg, dbConn)
	if rt != nil && rt.Store() != nil {
		if det := detectorInstance(); det != nil {
			if err := rt.Store().CleanupInvalidScenarioDependencies(det.ScenarioCatalog()); err != nil {
				log.Printf("Warning: failed to cleanup scenario dependencies: %v", err)
			}
		}
	}

	router := gin.Default()
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	handler := corsMiddleware.Handler(router)

	h := newHandler(rt)
	registerRoutes(router, h)
	graphapi.RegisterConnectRoutes(router, h.scenariosDir, rt.Store(), graphapi.ConnectOptions{
		CacheTTL:     cfg.InterfaceGraphCacheTTL,
		BuildTimeout: cfg.InterfaceGraphBuildTimeout,
	})
	spec, err := loadMaturitySpec()
	if err != nil {
		log.Printf("dependency-health: maturity assessment unavailable: %v", err)
	}
	dependencyhealthapi.RegisterConnectRoutes(router, h.scenariosDir, dependencyhealthapi.Options{MaturitySpec: spec})

	// AI semantic search over the governance records. The provider registry reads
	// the approved-dependencies corpus; the search service indexes it into Qdrant
	// (degrading to keyword text search when the backend is down) and is injected
	// as the SearchApprovedDependencies ranker.
	searchRegistry := dependencygovernanceapi.NewRegistry(filepath.Dir(h.scenariosDir()))
	searchService := aisearch.Start(context.Background(), searchRegistry)
	dependencygovernanceapi.RegisterConnectRoutes(router, h.scenariosDir,
		dependencygovernanceapi.WithSemanticRanker(searchService))

	// Federate the dependency-governance search into search-hub so agents reach it
	// via `search-hub query "<purpose>" --type dependency`. Best-effort: the hub
	// being down is logged, never fatal (upsert re-registers on next boot).
	searchJSONPath := filepath.Join(h.scenariosDir(), "scenario-dependency-analyzer", ".vrooli", "search.json")
	go searchregister.Register(context.Background(), searchregister.Config{
		ScenarioID:     "scenario-dependency-analyzer",
		SearchFilePath: searchJSONPath,
		Logger:         log.New(os.Stderr, "[scenario-dependency-analyzer/searchregister] ", log.LstdFlags),
	})

	log.Printf("Starting Scenario Dependency Analyzer API on port %s", cfg.Port)
	log.Printf("Scenarios directory: %s", cfg.ScenariosDir)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      max(30*time.Second, cfg.InterfaceGraphBuildTimeout+5*time.Second),
		IdleTimeout:       60 * time.Second,
	}
	return server.ListenAndServe()
}

func loadMaturitySpec() (*assessment.Spec, error) {
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("resolve repo root: %w", err)
	}
	path := filepath.Join(repoRoot, "scenarios", "scenario-dependency-analyzer", ".vrooli", "maturity.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return assessment.ParseSpec(raw)
}

func registerRoutes(router *gin.Engine, handler *handler) {
	router.GET("/health", gin.WrapF(health.New().Version("1.0.0").Check(health.DB(db), health.Critical).Handler()))
	router.GET("/api/v1/health/analysis", handler.analysisHealth)

	api := router.Group("/api/v1")
	{
		catalog.RegisterRoutes(api, handler.scenarioService())
		deploymentapi.RegisterHTTPRoutes(api, handler.deploymentService())
		analysisapi.RegisterHTTPRoutes(api, handler.analysisService(), handler.scanService())
		dependenciesapi.RegisterHTTPRoutes(api, handler.dependencyService())
		coresetapi.RegisterHTTPRoutes(api, handler.scenariosDir)
		graphapi.RegisterHTTPRoutes(api, handler.graphService(), handler.scenariosDir)
		proposalapi.RegisterHTTPRoutes(api, handler.proposalService())
		optimizationapi.RegisterHTTPRoutes(api, handler.optimizationService())
	}
}
