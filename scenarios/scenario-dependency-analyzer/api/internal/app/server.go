package app

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
	"github.com/vrooli/api-core/health"

	analysisapi "scenario-dependency-analyzer/internal/analysis"
	"scenario-dependency-analyzer/internal/catalog"
	appconfig "scenario-dependency-analyzer/internal/config"
	coresetapi "scenario-dependency-analyzer/internal/coreset"
	dependenciesapi "scenario-dependency-analyzer/internal/dependencies"
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
	graphapi.RegisterConnectRoutes(router, h.scenariosDir)

	log.Printf("Starting Scenario Dependency Analyzer API on port %s", cfg.Port)
	log.Printf("Scenarios directory: %s", cfg.ScenariosDir)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return server.ListenAndServe()
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
