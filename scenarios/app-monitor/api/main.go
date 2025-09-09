package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"app-monitor-api/config"
	"app-monitor-api/handlers"
	"app-monitor-api/middleware"
	"app-monitor-api/repository"
	"app-monitor-api/services"

	"github.com/gin-gonic/gin"
)

// Server holds all server dependencies
type Server struct {
	config   *config.Config
	router   *gin.Engine
	handlers *Handlers
}

// Handlers holds all handler instances
type Handlers struct {
	health    *handlers.HealthHandler
	app       *handlers.AppHandler
	system    *handlers.SystemHandler
	docker    *handlers.DockerHandler
	websocket *handlers.WebSocketHandler
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize server
	server, err := NewServer(cfg)
	if err != nil {
		log.Fatal("Failed to initialize server:", err)
	}

	// Start server with graceful shutdown
	if err := server.Run(); err != nil {
		log.Fatal("Server failed:", err)
	}
}

// NewServer creates and configures a new server instance
func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize dependencies
	db, err := cfg.InitializeDatabase()
	if err != nil {
		log.Printf("Warning: Database initialization failed: %v", err)
		// Continue without database
	}

	redis, err := cfg.InitializeRedis()
	if err != nil {
		log.Printf("Warning: Redis initialization failed: %v", err)
		// Continue without Redis
	}

	docker, err := cfg.InitializeDocker()
	if err != nil {
		log.Printf("Warning: Docker initialization failed: %v", err)
		// Continue without Docker
	}

	// Create repository layer
	var appRepo repository.AppRepository
	if db != nil {
		appRepo = repository.NewPostgresRepository(db)
	}

	// Create service layer
	appService := services.NewAppService(appRepo)
	metricsService := services.NewMetricsService()

	// Create handlers
	handlers := &Handlers{
		health:    handlers.NewHealthHandler(db, redis, docker),
		app:       handlers.NewAppHandler(appService),
		system:    handlers.NewSystemHandler(metricsService),
		docker:    handlers.NewDockerHandler(docker),
		websocket: handlers.NewWebSocketHandler(middleware.SecureWebSocketUpgrader(), redis),
	}

	// Setup router
	router := setupRouter(handlers, cfg)

	return &Server{
		config:   cfg,
		router:   router,
		handlers: handlers,
	}, nil
}

// setupRouter configures all routes and middleware
func setupRouter(h *Handlers, cfg *config.Config) *gin.Engine {
	// Set Gin mode based on environment
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Global middleware
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(nil))

	// Optional rate limiting
	if rateLimit := os.Getenv("RATE_LIMIT_PER_MINUTE"); rateLimit != "" {
		var limit int
		fmt.Sscanf(rateLimit, "%d", &limit)
		if limit > 0 {
			r.Use(middleware.RateLimiting(limit))
		}
	}

	// Optional API key validation
	if os.Getenv("API_KEY") != "" {
		r.Use(middleware.ValidateAPIKey())
	}

	// Health endpoints (no auth required)
	r.GET("/health", h.health.Check)
	r.GET("/api/health", h.health.APIHealth)

	// API routes
	api := r.Group("/api")
	{
		// System endpoints
		api.GET("/system/info", h.system.GetSystemInfo)
		api.GET("/system/metrics", h.system.GetSystemMetrics)

		// App management endpoints
		api.GET("/apps", h.app.GetApps)
		api.GET("/apps/:id", h.app.GetApp)
		api.POST("/apps/:id/start", h.app.StartApp)
		api.POST("/apps/:id/stop", h.app.StopApp)
		api.GET("/apps/:id/logs", h.app.GetAppLogs)
		api.GET("/apps/:id/logs/lifecycle", h.app.GetAppLifecycleLogs)
		api.GET("/apps/:id/logs/background", h.app.GetAppBackgroundLogs)
		api.GET("/apps/:id/metrics", h.app.GetAppMetrics)

		// Log endpoints for scenarios using app name
		api.GET("/logs/:appName", h.app.GetAppLogs)

		// Resource endpoints
		api.GET("/resources", h.system.GetResources)

		// Docker integration endpoints
		api.GET("/docker/info", h.docker.GetDockerInfo)
		api.GET("/docker/containers", h.docker.GetContainers)
	}

	// WebSocket endpoint
	r.GET("/ws", h.websocket.HandleWebSocket)

	return r
}

// Run starts the server with graceful shutdown support
func (s *Server) Run() error {
	srv := &http.Server{
		Addr:         ":" + s.config.API.Port,
		Handler:      s.router,
		ReadTimeout:  s.config.API.ReadTimeout,
		WriteTimeout: s.config.API.WriteTimeout,
	}

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Listen for interrupt signals
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		log.Println("Shutting down server...")

		// Shutdown signal with grace period
		shutdownCtx, cancel := context.WithTimeout(serverCtx, s.config.API.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		serverStopCtx()
	}()

	// Start server
	log.Printf("🚀 App Monitor API server starting on port %s", s.config.API.Port)
	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed to start: %w", err)
	}

	// Wait for server context to be done
	<-serverCtx.Done()
	log.Println("✅ Server shutdown complete")

	return nil
}