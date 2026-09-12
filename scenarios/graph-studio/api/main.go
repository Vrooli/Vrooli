package main

import (
	"context"
	"github.com/vrooli/api-core/database"
	schema "graph-studio/internal/graph"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "graph-studio",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Load environment variables
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("API_PORT environment variable is required")
	}

	// Database configuration is resolved by the shared Postgres seam.
	postgresURL, dsnErr := database.ResolvePostgresDSN(os.Getenv)
	if dsnErr != nil {
		log.Fatalf("Database configuration missing: %v", dsnErr)
	}
	dbConfig := DatabaseConfig{
		URL:        postgresURL,
		MaxRetries: 10,
	}

	// Connect to database with retry logic
	db, err := ConnectWithRetry(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Start connection monitor in background
	go MonitorConnection(db, dbConfig)

	// Initialize API
	api := NewAPI()

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Add middleware
	router.Use(ErrorHandlerMiddleware())
	router.Use(SecurityHeadersMiddleware())
	router.Use(PreviewAccessMiddleware())
	router.Use(RateLimitMiddleware(50, 100))                 // 50 req/sec, burst of 100
	router.Use(RequestSizeLimitMiddleware(10 * 1024 * 1024)) // 10 MB limit
	router.Use(RequestIDMiddleware())
	router.Use(LoggingMiddleware())
	router.Use(DatabaseMiddleware(db))
	router.Use(UserContextMiddleware())
	router.Use(TimeoutMiddleware(30 * time.Second))
	router.Use(api.metrics.PerformanceMiddleware())
	router.Use(api.metrics.ErrorTrackingMiddleware())

	// Configure CORS
	corsConfig := cors.DefaultConfig()

	// Get allowed origins from environment or use sensible defaults
	allowedOrigins := os.Getenv("CORS_ORIGINS")
	if allowedOrigins == "" || allowedOrigins == "*" {
		// Default to localhost for development
		corsConfig.AllowOrigins = []string{
			"http://localhost:" + os.Getenv("UI_PORT"),
			"http://127.0.0.1:" + os.Getenv("UI_PORT"),
		}
	} else {
		// Parse comma-separated origins from environment
		corsConfig.AllowOrigins = strings.Split(allowedOrigins, ",")
	}

	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "Authorization", "X-User-ID", "X-Request-ID", "X-Preview-Token")
	router.Use(cors.New(corsConfig))

	// Health check
	healthHandler := health.New().Version("1.0.0").Check(health.DB(db), health.Critical).Handler()
	router.GET("/health", gin.WrapF(healthHandler))

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Dashboard stats
		v1.GET("/stats", api.GetStats)

		// Plugin routes
		v1.GET("/plugins", api.ListPlugins)

		// Graph routes
		v1.GET("/graphs", api.ListGraphs)
		v1.POST("/graphs", api.CreateGraph)
		v1.GET("/graphs/:id", api.GetGraph)
		v1.PUT("/graphs/:id", api.UpdateGraph)
		v1.DELETE("/graphs/:id", api.DeleteGraph)

		// Graph operations
		v1.POST("/graphs/:id/validate", api.ValidateGraph)
		v1.POST("/graphs/:id/convert", api.ConvertGraph)
		v1.POST("/graphs/:id/render", api.RenderGraph)
		v1.POST("/graphs/:id/export", api.ExportGraph)

		// Conversion capabilities
		v1.GET("/conversions", api.ListConversions)
		v1.GET("/conversions/:from/:to", api.GetConversionMetadata)

		// Monitoring and metrics
		v1.GET("/metrics", api.GetSystemMetrics)
		v1.GET("/health/detailed", api.GetDetailedHealth)
	}

	// Start server with graceful shutdown
	log.Printf("🚀 Graph Studio API starting on port %s", port)

	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(schema.Schema)); err != nil {
		log.Fatalf("database schema initialization failed: %v", err)
	}
	if err := server.Run(server.Config{
		Handler: router,
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
