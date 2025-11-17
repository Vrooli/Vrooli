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
	cleanupInvalidScenarioDependencies()

	router := gin.Default()
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	handler := corsMiddleware.Handler(router)

	registerRoutes(router)

	log.Printf("Starting Scenario Dependency Analyzer API on port %s", cfg.Port)
	log.Printf("Scenarios directory: %s", cfg.ScenariosDir)

	return http.ListenAndServe(":"+cfg.Port, handler)
}

func registerRoutes(router *gin.Engine) {
	router.GET("/health", healthHandler)
	router.GET("/api/v1/health/analysis", analysisHealthHandler)

	api := router.Group("/api/v1")
	{
		api.GET("/scenarios", listScenariosHandler)
		api.GET("/scenarios/:scenario", getScenarioDetailHandler)
		api.GET("/scenarios/:scenario/deployment", getDeploymentReportHandler)
		api.GET("/analyze/:scenario", analyzeScenarioHandler)
		api.GET("/scenarios/:scenario/dependencies", getDependenciesHandler)
		api.POST("/scenarios/:scenario/scan", scanScenarioHandler)
		api.GET("/graph/:type", getGraphHandler)
		api.POST("/analyze/proposed", analyzeProposedHandler)
		api.POST("/optimize", optimizeHandler)
	}
}

func analysisHealthHandler(c *gin.Context) {
	_, err := generateDependencyGraph("combined")
	if err != nil {
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
