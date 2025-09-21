package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

// Configuration holds all app configuration
type Configuration struct {
	APIPort      string
	UIPort       string
	DatabaseURL  string
	RedisURL     string
	MinIOURL     string
	OllamaURL    string
	JWTSecret    string
	Environment  string
	Mode         string // "server" or "worker" or "both"
}

// Application holds all dependencies
type Application struct {
	Config       *Configuration
	DB           *sql.DB
	Redis        *redis.Client
	JobProcessor *JobProcessor
	PlatformMgr  *PlatformManager
	WebSocket    *WebSocketManager
	Router       *gin.Engine
}

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start social-media-scheduler

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Parse command line flags
	mode := flag.String("mode", "both", "Run mode: server, worker, or both")
	flag.Parse()

	// Load configuration
	config := loadConfiguration()
	config.Mode = *mode

	// Initialize application
	app, err := initializeApplication(config)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer app.cleanup()

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Start job processor if needed
	if config.Mode == "worker" || config.Mode == "both" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("🔄 Starting job processor...")
			app.JobProcessor.Start(ctx)
		}()
	}

	// Start HTTP server if needed
	if config.Mode == "server" || config.Mode == "both" {
		server := &http.Server{
			Addr:    ":" + config.APIPort,
			Handler: app.Router,
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("🚀 Starting API server on port %s", config.APIPort)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Server error: %v", err)
			}
		}()

		// Handle shutdown
		go func() {
			<-signalChan
			log.Println("📴 Shutting down gracefully...")
			
			cancel() // Cancel context for job processor
			
			// Shutdown HTTP server
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()
			
			if err := server.Shutdown(shutdownCtx); err != nil {
				log.Printf("Server shutdown error: %v", err)
			}
		}()
	} else {
		// Worker-only mode, just wait for signal
		go func() {
			<-signalChan
			log.Println("📴 Shutting down job processor...")
			cancel()
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()
	log.Println("✅ Shutdown complete")
}

func loadConfiguration() *Configuration {
	// All configuration must come from environment variables - NO DEFAULTS
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}
	
	uiPort := os.Getenv("UI_PORT")
	if uiPort == "" {
		log.Fatal("❌ UI_PORT environment variable is required")
	}
	
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("❌ DATABASE_URL environment variable is required")
	}
	
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("❌ REDIS_URL environment variable is required")
	}
	
	minioURL := os.Getenv("MINIO_URL")
	if minioURL == "" {
		log.Fatal("❌ MINIO_URL environment variable is required")
	}
	
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		log.Fatal("❌ OLLAMA_URL environment variable is required")
	}
	
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("❌ JWT_SECRET environment variable is required")
	}
	
	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		log.Fatal("❌ ENVIRONMENT environment variable is required")
	}

	config := &Configuration{
		APIPort:     apiPort,
		UIPort:      uiPort,
		DatabaseURL: databaseURL,
		RedisURL:    redisURL,
		MinIOURL:    minioURL,
		OllamaURL:   ollamaURL,
		JWTSecret:   jwtSecret,
		Environment: environment,
	}

	return config
}


func initializeApplication(config *Configuration) (*Application, error) {
	app := &Application{Config: config}

	// Initialize database
	var err error
	app.DB, err = sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	app.DB.SetMaxOpenConns(25)
	app.DB.SetMaxIdleConns(5)
	app.DB.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = app.DB.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
			break
		}
		
		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay) * math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		
		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter
		
		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)
		
		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		return nil, fmt.Errorf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	// Initialize Redis
	opt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	app.Redis = redis.NewClient(opt)

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = app.Redis.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis ping failed: %w", err)
	}

	// Initialize platform manager
	app.PlatformMgr = NewPlatformManager(config.OllamaURL)

	// Initialize job processor
	app.JobProcessor = NewJobProcessor(app.DB, app.Redis, app.PlatformMgr)

	// Initialize WebSocket manager
	app.WebSocket = NewWebSocketManager()

	// Set up router
	app.setupRouter()

	return app, nil
}

func (app *Application) setupRouter() {
	if app.Config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	app.Router = gin.Default()

	// CORS configuration
	app.Router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:" + app.Config.UIPort},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check endpoints
	app.Router.GET("/health", app.healthCheck)
	app.Router.GET("/health/queue", app.queueHealthCheck)

	// WebSocket endpoint for real-time updates
	app.Router.GET("/ws", app.handleWebSocket)

	// API routes
	api := app.Router.Group("/api/v1")
	{
		// Authentication routes
		auth := api.Group("/auth")
		{
			auth.POST("/login", app.login)
			auth.POST("/register", app.register)
			auth.POST("/logout", app.logout)
			auth.GET("/me", app.authenticate(), app.getCurrentUser)
			auth.GET("/platforms", app.getPlatformConfigs)
		}

		// OAuth routes for social platform connections
		oauth := api.Group("/oauth")
		{
			oauth.GET("/:platform/connect", app.authenticate(), app.initiateOAuth)
			oauth.GET("/:platform/callback", app.handleOAuthCallback)
			oauth.DELETE("/:platform/disconnect", app.authenticate(), app.disconnectPlatform)
		}

		// Protected routes (require authentication)
		protected := api.Group("")
		protected.Use(app.authenticate())
		{
			// User management
			protected.GET("/user/accounts", app.getUserSocialAccounts)
			protected.GET("/user/stats", app.getUserStats)
			protected.PUT("/user/preferences", app.updateUserPreferences)

			// Campaign management
			campaigns := protected.Group("/campaigns")
			{
				campaigns.GET("", app.getCampaigns)
				campaigns.POST("", app.createCampaign)
				campaigns.GET("/:id", app.getCampaign)
				campaigns.PUT("/:id", app.updateCampaign)
				campaigns.DELETE("/:id", app.deleteCampaign)
			}

			// Post scheduling
			posts := protected.Group("/posts")
			{
				posts.GET("/calendar", app.getCalendarPosts)
				posts.POST("/schedule", app.schedulePost)
				posts.GET("/:id", app.getPost)
				posts.PUT("/:id", app.updatePost)
				posts.DELETE("/:id", app.deletePost)
				posts.POST("/:id/optimize", app.optimizePost)
				posts.POST("/:id/duplicate", app.duplicatePost)
				posts.GET("/:id/preview", app.previewPost)
			}

			// Bulk operations
			bulk := protected.Group("/bulk")
			{
				bulk.POST("/schedule", app.bulkSchedule)
				bulk.POST("/import", app.importPosts)
				bulk.PUT("/reschedule", app.bulkReschedule)
			}

			// Analytics
			analytics := protected.Group("/analytics")
			{
				analytics.GET("/overview", app.getAnalyticsOverview)
				analytics.GET("/platforms", app.getPlatformAnalytics)
				analytics.GET("/posts/:id/metrics", app.getPostMetrics)
				analytics.GET("/optimal-times", app.getOptimalPostingTimes)
			}

			// Media management
			media := protected.Group("/media")
			{
				media.POST("/upload", app.uploadMedia)
				media.GET("", app.getMediaFiles)
				media.DELETE("/:id", app.deleteMedia)
				media.POST("/:id/optimize", app.optimizeMediaForPlatforms)
			}
		}
	}
}

func (app *Application) cleanup() {
	if app.DB != nil {
		app.DB.Close()
	}
	if app.Redis != nil {
		app.Redis.Close()
	}
	if app.WebSocket != nil {
		app.WebSocket.Close()
	}
}

// Health check handlers
func (app *Application) healthCheck(c *gin.Context) {
	services := make(map[string]string)

	// Check database
	if err := app.DB.Ping(); err != nil {
		services["database"] = "unhealthy: " + err.Error()
	} else {
		services["database"] = "healthy"
	}

	// Check Redis
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Redis.Ping(ctx).Err(); err != nil {
		services["redis"] = "unhealthy: " + err.Error()
	} else {
		services["redis"] = "healthy"
	}

	// Check if any critical service is down
	status := "healthy"
	for _, serviceStatus := range services {
		if serviceStatus != "healthy" {
			status = "unhealthy"
			break
		}
	}

	response := HealthResponse{
		Status:    status,
		Version:   "1.0.0",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  services,
	}

	if status == "healthy" {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

func (app *Application) queueHealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check queue lengths
	queueStats := make(map[string]interface{})

	// Check scheduled posts queue
	scheduledLen, err := app.Redis.LLen(ctx, "queue:scheduled_posts").Result()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Success: false,
			Error:   "Failed to check queue health: " + err.Error(),
		})
		return
	}
	queueStats["scheduled_posts"] = scheduledLen

	// Check failed posts queue
	failedLen, err := app.Redis.LLen(ctx, "queue:failed_posts").Result()
	if err == nil {
		queueStats["failed_posts"] = failedLen
	}

	// Check processing queue
	processingLen, err := app.Redis.LLen(ctx, "queue:processing").Result()
	if err == nil {
		queueStats["processing"] = processingLen
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"status":     "healthy",
			"queues":     queueStats,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
			"worker_pid": os.Getpid(),
		},
	})
}

// WebSocket handler for real-time updates
var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from UI port
		return true // TODO: Implement proper origin checking
	},
}

func (app *Application) handleWebSocket(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Extract user from JWT if available
	userID := c.Query("user_id") // TODO: Extract from JWT token

	client := &WebSocketClient{
		ID:     generateClientID(),
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	app.WebSocket.Register(client)

	// Start client handlers
	go client.writePump()
	go client.readPump(app.WebSocket)
}

func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}