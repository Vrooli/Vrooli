package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start graph-studio

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Check if running through lifecycle (temporarily disabled for testing)
	/*
		if os.Getenv("VROOLI_LIFECYCLE") != "active" {
			log.Println("⚠️  Not running through Vrooli lifecycle. Starting with lifecycle...")

			// Get the scenario name from environment or use default
			scenarioName := os.Getenv("SCENARIO_NAME")
			if scenarioName == "" {
				scenarioName = "graph-studio"
			}

			// Execute through lifecycle
			cmd := exec.Command("bash", "-c", fmt.Sprintf("vrooli scenario run %s", scenarioName))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				log.Fatalf("Failed to start through lifecycle: %v", err)
			}
			return
		}
	*/

	// Load environment variables
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("API_PORT environment variable is required")
	}

	// Database configuration
	dbConfig := DatabaseConfig{
		Host:       os.Getenv("POSTGRES_HOST"),
		Port:       os.Getenv("POSTGRES_PORT"),
		User:       os.Getenv("POSTGRES_USER"),
		Password:   os.Getenv("POSTGRES_PASSWORD"),
		Database:   os.Getenv("POSTGRES_DB"),
		MaxRetries: 10,
	}

	if dbConfig.Host == "" || dbConfig.Port == "" || dbConfig.User == "" || dbConfig.Password == "" || dbConfig.Database == "" {
		log.Fatal("Database configuration missing. Required: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
	}

	// Connect to database with retry logic
	db, err := ConnectWithRetry(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

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
	router.GET("/health", api.HealthCheck)

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

	// Start server
	log.Printf("🚀 Graph Studio API starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
