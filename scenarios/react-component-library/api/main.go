package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/vrooli/scenarios/react-component-library/handlers"
	"github.com/vrooli/scenarios/react-component-library/middleware"
	"github.com/vrooli/scenarios/react-component-library/services"
)

func main() {
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

	// Start server
	port := getEnvOrDefault("API_PORT", "8090")
	log.Printf("🚀 React Component Library API starting on port %s", port)
	log.Printf("📚 API Documentation: http://localhost:%s/api/docs", port)
	log.Printf("🔍 Health Check: http://localhost:%s/health", port)
	
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func initDatabase() (*sql.DB, error) {
	dbHost := getEnvOrDefault("DB_HOST", "localhost")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "postgres")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "postgres")
	dbName := getEnvOrDefault("DB_NAME", "postgres")
	dbSchema := getEnvOrDefault("DB_SCHEMA", "react_component_library")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable search_path=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSchema)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("✅ Connected to PostgreSQL database: %s@%s:%s/%s", dbUser, dbHost, dbPort, dbName)
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

	// Rate limiting middleware
	rateLimitRPM, _ := strconv.Atoi(getEnvOrDefault("RATE_LIMIT_RPM", "200"))
	aiRateLimitRPM, _ := strconv.Atoi(getEnvOrDefault("AI_RATE_LIMIT_RPM", "10"))
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
					"GET /api/v1/components":              "List all components with optional filters",
					"POST /api/v1/components":             "Create a new component",
					"GET /api/v1/components/{id}":         "Get component details",
					"PUT /api/v1/components/{id}":         "Update component",
					"DELETE /api/v1/components/{id}":      "Delete component",
					"GET /api/v1/components/search":       "Search components using natural language",
					"POST /api/v1/components/generate":    "Generate component using AI",
					"POST /api/v1/components/{id}/test":   "Run accessibility/performance tests",
					"POST /api/v1/components/{id}/export": "Export component in various formats",
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

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}