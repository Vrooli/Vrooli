package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/vrooli/scenarios/react-component-library/handlers"
	"github.com/vrooli/scenarios/react-component-library/middleware"
	"github.com/vrooli/scenarios/react-component-library/services"
)

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start react-component-library

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Initialize database
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize services
	componentService := services.NewComponentService(db)
	testingService := services.NewTestingService()
	aiService := services.NewAIService()
	searchService := services.NewSearchService()

	// Initialize handlers
	componentHandler := handlers.NewComponentHandler(componentService, testingService, aiService, searchService)
	healthHandler := handlers.NewHealthHandler(db)

	// Setup router
	router := setupRouter(componentHandler, healthHandler)

	// Start server - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}
	log.Printf("🚀 React Component Library API starting on port %s", port)
	log.Printf("📚 API Documentation: http://localhost:%s/api/docs", port)
	log.Printf("🔍 Health Check: http://localhost:%s/health", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func initDatabase() (*sql.DB, error) {
	// Database configuration - support both POSTGRES_URL and individual components
	connStr := os.Getenv("POSTGRES_URL")
	if connStr == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		dbSchema := os.Getenv("DB_SCHEMA")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		if dbSchema == "" {
			dbSchema = "react_component_library" // This is application-specific, not a credential
		}

		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable search_path=%s",
			dbHost, dbPort, dbUser, dbPassword, dbName, dbSchema)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📊 Database URL configured")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * rand.Float64())
		actualDelay := delay + jitter

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt+1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		return nil, fmt.Errorf("❌ Database connection failed after %d attempts: %w", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")
	return db, nil
}

func setupRouter(componentHandler *handlers.ComponentHandler, healthHandler *handlers.HealthHandler) *gin.Engine {
	// Set gin mode based on environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RequestID())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:*", "https://*.vrooli.com"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
	}))

	// Rate limiting middleware - with sensible defaults for non-sensitive config
	rateLimitRPM, _ := strconv.Atoi(getEnvWithDefault("RATE_LIMIT_RPM", "200"))
	aiRateLimitRPM, _ := strconv.Atoi(getEnvWithDefault("AI_RATE_LIMIT_RPM", "10"))
	router.Use(middleware.RateLimit(rateLimitRPM, aiRateLimitRPM))

	// Health endpoints
	router.GET("/health", healthHandler.HealthCheck)
	router.GET("/api/metrics", healthHandler.Metrics)

	// API documentation
	router.GET("/api/docs", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":        "React Component Library API",
			"version":     "1.0.0",
			"description": "AI-powered React component library with accessibility testing and performance benchmarking",
			"endpoints": gin.H{
				"components": gin.H{
					"GET /api/v1/components":               "List all components with optional filters",
					"POST /api/v1/components":              "Create a new component",
					"GET /api/v1/components/{id}":          "Get component details",
					"PUT /api/v1/components/{id}":          "Update component",
					"DELETE /api/v1/components/{id}":       "Delete component",
					"GET /api/v1/components/search":        "Search components using natural language",
					"POST /api/v1/components/generate":     "Generate component using AI",
					"POST /api/v1/components/{id}/test":    "Run accessibility/performance tests",
					"POST /api/v1/components/{id}/export":  "Export component in various formats",
					"POST /api/v1/components/{id}/improve": "Get AI improvement suggestions",
				},
			},
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Component CRUD operations
		v1.GET("/components", componentHandler.ListComponents)
		v1.POST("/components", componentHandler.CreateComponent)
		v1.GET("/components/:id", componentHandler.GetComponent)
		v1.PUT("/components/:id", componentHandler.UpdateComponent)
		v1.DELETE("/components/:id", componentHandler.DeleteComponent)

		// Component search and discovery
		v1.GET("/components/search", componentHandler.SearchComponents)

		// AI-powered features
		v1.POST("/components/generate", middleware.AIRateLimit(), componentHandler.GenerateComponent)
		v1.POST("/components/:id/improve", middleware.AIRateLimit(), componentHandler.ImproveComponent)

		// Testing and analysis
		v1.POST("/components/:id/test", componentHandler.TestComponent)
		v1.GET("/components/:id/benchmark", componentHandler.BenchmarkComponent)

		// Export and sharing
		v1.POST("/components/:id/export", componentHandler.ExportComponent)
		v1.GET("/components/:id/versions", componentHandler.GetComponentVersions)

		// Analytics
		v1.GET("/analytics/usage", componentHandler.GetUsageAnalytics)
		v1.GET("/analytics/popular", componentHandler.GetPopularComponents)
	}

	// Serve static files for UI (in production)
	if gin.Mode() == gin.ReleaseMode {
		router.Static("/static", "./ui/build/static")
		router.StaticFile("/", "./ui/build/index.html")
		router.NoRoute(func(c *gin.Context) {
			c.File("./ui/build/index.html")
		})
	}

	return router
}

// getEnvWithDefault - ONLY for non-sensitive configuration values like rate limits, UI settings
// NEVER use for credentials, database passwords, API keys, or security-sensitive values
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
