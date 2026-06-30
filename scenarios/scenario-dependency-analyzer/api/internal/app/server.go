package app

import (
	"context"
	"database/sql"
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
	vroolicli "github.com/vrooli/vrooli-cli-go"

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
	resourceusageapi "scenario-dependency-analyzer/internal/resourceusage"
	searchcontrolapi "scenario-dependency-analyzer/internal/searchcontrol"
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
	graphIngest := newGraphIngestService(rt)
	registerRoutes(router, h, graphIngest)

	// AI semantic search over SDA's corpora. ONE multi-corpus service backs every
	// federated leaf: the dependency-governance corpus (RankIDs ranker), the
	// scenario-connection corpus (graph SearchInterfaceGraph), and the
	// resource-usage corpus (ResourceUsageService). Built before the leaf handlers
	// so each can query it; a down Qdrant/Ollama degrades to keyword text search.
	graphOpts := graphapi.ConnectOptions{
		CacheTTL:     cfg.InterfaceGraphCacheTTL,
		BuildTimeout: cfg.InterfaceGraphBuildTimeout,
	}
	searchJSONPath := filepath.Join(h.scenariosDir(), "scenario-dependency-analyzer", ".vrooli", "search.json")
	searchRegistry := dependencygovernanceapi.NewRegistry(filepath.Dir(h.scenariosDir()))
	connectionsProvider := graphapi.NewConnectionsProvider(h.scenariosDir, rt.Store(), graphOpts)
	resourceProvider := resourceusageapi.NewUsageProvider(h.scenariosDir)
	searchService := aisearch.Start(context.Background(), aisearch.Sources{
		Dependencies:   searchRegistry,
		Scenarios:      connectionsProvider,
		Resources:      resourceProvider,
		SearchJSONPath: searchJSONPath,
	})

	// The control token gates the shared, token-gated reindex/config-write plane.
	// search-hub mints it at registration and is its only holder, so the token
	// alone gates the mutating verbs for every SDA leaf (provider_id selects).
	controlTokens := searchcontrolapi.NewTokenStore()
	searchcontrolapi.RegisterConnectRoutes(router, searchcontrolapi.Deps{
		Logger:       log.New(os.Stderr, "[scenario-dependency-analyzer/searchcontrol] ", log.LstdFlags),
		Reindexer:    searchcontrolapi.ServiceAdapter{Service: searchService},
		ConfigWriter: searchcontrolapi.FileConfigWriter{Path: searchJSONPath},
		CorpusWriter: searchcontrolapi.FileCorpusWriter{Path: searchJSONPath},
		Gate:         &searchcontrolapi.Gate{Tokens: controlTokens},
	})

	graphapi.RegisterConnectRoutes(router, h.scenariosDir, rt.Store(), graphOpts, searchService)
	resourceusageapi.RegisterConnectRoutes(router, searchService)
	if graphIngest != nil {
		graphIngest.StartSweeper(context.Background())
	}
	repoRoot, repoRootErr := repocontract.ResolveRepoRoot()
	if repoRootErr != nil {
		log.Printf("dependency-health: could not resolve repo root for maturity spec: %v", repoRootErr)
	}
	spec, specErr := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "scenario-dependency-analyzer"))
	if specErr != nil {
		log.Printf("dependency-health: maturity assessment unavailable: %v", specErr)
	}
	// Capture host facts once; they do not change during the process lifetime.
	// A failure (CLI unavailable) is non-fatal — the metrics collector backfills
	// os/arch/num_cpu from the stdlib, leaving richer facts unset.
	environment, envErr := vroolicli.New().HostCaptureEnvironment(context.Background())
	if envErr != nil {
		log.Printf("dependency-health: host inventory unavailable, metrics environment limited to stdlib baseline: %v", envErr)
		environment = nil
	}
	dependencyhealthapi.RegisterConnectRoutes(router, h.scenariosDir, dependencyhealthapi.Options{MaturitySpec: spec, Environment: environment})

	// The dependency-governance corpus is injected as the SearchApprovedDependencies
	// ranker (searchService is the same multi-corpus service built above).
	dependencygovernanceapi.RegisterConnectRoutes(router, h.scenariosDir,
		dependencygovernanceapi.WithSemanticRanker(searchService))

	// Federate every SDA leaf into search-hub from the same .vrooli/search.json
	// SSOT (dependencies, scenarios, resources). Best-effort: the hub being down is
	// logged, never fatal (upsert re-registers on next boot). The control-token
	// callbacks cache the secret search-hub mints so the searchcontrol plane can
	// authorize the sweep's reindex/config-write verbs.
	go searchregister.Register(context.Background(), searchregister.Config{
		ScenarioID:     "scenario-dependency-analyzer",
		SearchFilePath: searchJSONPath,
		Logger:         log.New(os.Stderr, "[scenario-dependency-analyzer/searchregister] ", log.LstdFlags),
		OnControlToken: func(providerID string, token string) { controlTokens.Set(providerID, token) },
		ControlToken:   func(providerID string) string { return controlTokens.Get(providerID) },
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

func registerRoutes(router *gin.Engine, handler *handler, graphIngest *graphIngestService) {
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
		graphIngest.RegisterRoutes(api)
		proposalapi.RegisterHTTPRoutes(api, handler.proposalService())
		optimizationapi.RegisterHTTPRoutes(api, handler.optimizationService())
	}
}
