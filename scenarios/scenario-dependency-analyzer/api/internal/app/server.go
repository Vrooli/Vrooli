package app

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"

	appconfig "scenario-dependency-analyzer/internal/config"
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

	log.Printf("Starting Scenario Dependency Analyzer API on port %s", cfg.Port)
	log.Printf("Scenarios directory: %s", cfg.ScenariosDir)

	return http.ListenAndServe(":"+cfg.Port, handler)
}

func registerRoutes(router *gin.Engine, handler *handler) {
	router.GET("/health", handler.health)
	router.GET("/api/v1/health/analysis", handler.analysisHealth)

	api := router.Group("/api/v1")
	{
		api.GET("/scenarios", handler.listScenarios)
		api.GET("/scenarios/:scenario", handler.getScenarioDetail)
		api.GET("/scenarios/:scenario/deployment", handler.getDeploymentReport)
		api.GET("/analyze/:scenario", handler.analyzeScenario)
		api.GET("/scenarios/:scenario/dependencies", handler.getDependencies)
		api.POST("/scenarios/:scenario/scan", handler.scanScenario)
		api.GET("/graph/:type", handler.getGraph)
		api.POST("/analyze/proposed", handler.analyzeProposed)
		api.POST("/optimize", handler.optimize)
	}
}
