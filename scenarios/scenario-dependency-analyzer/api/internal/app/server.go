package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/maturity-go/assessment"
	repocontract "github.com/vrooli/repo-contract-go"
	searchregister "github.com/vrooli/searchregister-go"
	vroolicli "github.com/vrooli/vrooli-cli-go"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/aisearch"
	analysisapi "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/analysis"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/catalog"
	appconfig "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/config"
	coresetapi "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/coreset"
	dependenciesapi "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/dependencies"
	dependencygovernanceapi "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/dependencygovernance"
	dependencyhealthapi "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/dependencyhealth"
	deploymentapi "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/deployment"
	graphapi "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/graph"
	optimizationapi "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/optimization"
	platformverdictapi "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/platformverdict"
	proposalapi "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/proposal"
	resourceusageapi "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/resourceusage"
	searchcontrolapi "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/searchcontrol"
)

// Run boots the HTTP API using the provided configuration and database connection.
func Run(cfg appconfig.Config, dbConn *database.RoutedDB) error {
	primaryDB := dbConn.Primary()
	db = primaryDB
	rt := ensureRuntime(cfg, primaryDB)
	log.Println("Scenario Dependency Analyzer runtime initialized")
	// Catalog discovery touches every scenario directory and the cleanup shares
	// the API's SQLite connection. It is maintenance, not a readiness
	// dependency, so leave a startup window for health checks before running it.
	time.AfterFunc(time.Minute, func() { cleanupInvalidScenarioDependencies(rt) })

	router := gin.Default()
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	handler := corsMiddleware.Handler(securityHeaders(router))

	// Route test-genie requests into an installed test pool without restarting
	// the scenario. The production path remains the primary database when the
	// test-mode header is absent or the scenario is not in development mode.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, dbConn)
	rootMux.Handle("/", handler)
	handler = apihttp.TestModeMiddleware(rootMux)

	h := newHandler(rt)
	graphIngest := newGraphIngestService(rt)
	registerRoutes(router, h, graphIngest)
	log.Println("Scenario Dependency Analyzer HTTP routes registered")

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
	log.Println("Scenario Dependency Analyzer semantic search initialized")

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
	platformverdictapi.RegisterConnectRoutes(router, h.scenariosDir)

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

// securityHeaders protects every REST and Connect response at the router
// boundary so newly registered routes inherit the same policy.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("X-XSS-Protection", "0")
		header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

func cleanupInvalidScenarioDependencies(rt *Runtime) {
	if rt == nil || rt.Store() == nil {
		return
	}
	det := detectorInstance()
	if det == nil {
		return
	}
	if err := rt.Store().CleanupInvalidScenarioDependencies(det.ScenarioCatalog()); err != nil {
		log.Printf("Warning: failed to cleanup scenario dependencies: %v", err)
		return
	}
	log.Println("Scenario Dependency Analyzer dependency cleanup complete")
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
