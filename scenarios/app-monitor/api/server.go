package main

import (
	"context"
	"database/sql"
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
	"app-monitor-api/toolexecution"
	"app-monitor-api/toolregistry"

	"github.com/gin-gonic/gin"
	"github.com/vrooli/api-core/health"
)

// Server holds all server dependencies
type Server struct {
	config   *config.Config
	router   *gin.Engine
	handlers *Handlers
}

// Handlers holds all handler instances
type Handlers struct {
	app           *handlers.AppHandler
	system        *handlers.SystemHandler
	docker        *handlers.DockerHandler
	websocket     *handlers.WebSocketHandler
	lighthouse    *handlers.LighthouseHandler
	tools         *handlers.ToolsHandler
	toolExecution *toolexecution.Handler
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

	// Initialize Tool Discovery Protocol registry
	toolReg := toolregistry.NewRegistry(toolregistry.RegistryConfig{
		ScenarioName:        "app-monitor",
		ScenarioVersion:     "1.0.0",
		ScenarioDescription: "Centralized monitoring and control dashboard for the Vrooli ecosystem",
	})

	// Register all tool providers
	toolReg.RegisterProvider(toolregistry.NewDiscoveryToolProvider())
	toolReg.RegisterProvider(toolregistry.NewLifecycleToolProvider())
	toolReg.RegisterProvider(toolregistry.NewDiagnosticsToolProvider())
	toolReg.RegisterProvider(toolregistry.NewLogsMetricsToolProvider())
	toolReg.RegisterProvider(toolregistry.NewIssueToolProvider())
	toolReg.RegisterProvider(toolregistry.NewDocsToolProvider())
	toolReg.RegisterProvider(toolregistry.NewResourceToolProvider())

	// Create tool executor
	toolExecutor := toolexecution.NewServerExecutor(toolexecution.ServerExecutorConfig{
		AppService:     appService,
		MetricsService: metricsService,
	})

	// Create handlers
	handlers := &Handlers{
		app:           handlers.NewAppHandler(appService),
		system:        handlers.NewSystemHandler(metricsService),
		docker:        handlers.NewDockerHandler(docker),
		websocket:     handlers.NewWebSocketHandler(middleware.SecureWebSocketUpgrader(), redis),
		lighthouse:    handlers.NewLighthouseHandler(),
		tools:         handlers.NewToolsHandler(toolReg),
		toolExecution: toolexecution.NewHandler(toolExecutor),
	}

	// Setup router
	router := setupRouter(handlers, cfg, db)

	return &Server{
		config:   cfg,
		router:   router,
		handlers: handlers,
	}, nil
}

// setupRouter configures all routes and middleware
func setupRouter(h *Handlers, cfg *config.Config, db *sql.DB) *gin.Engine {
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
	healthHandler := health.New().Version("1.0.0").Check(health.DB(db), health.Optional).Handler()
	r.GET("/health", gin.WrapF(healthHandler))
	r.GET("/api/health", gin.WrapF(healthHandler))

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// System endpoints
		v1.GET("/system/status", h.system.GetSystemStatus)
		v1.GET("/system/metrics", h.system.GetSystemMetrics)

		// App management endpoints
		v1.GET("/apps/summary", h.app.GetAppsSummary)
		v1.GET("/apps", h.app.GetApps)
		v1.GET("/apps/:id", h.app.GetApp)
		v1.POST("/apps/:id/start", h.app.StartApp)
		v1.POST("/apps/:id/stop", h.app.StopApp)
		v1.POST("/apps/:id/restart", h.app.RestartApp)
		v1.POST("/apps/:id/view", h.app.RecordAppView)
		v1.GET("/apps/:id/issues", h.app.GetAppIssues)
		v1.POST("/apps/:id/report", h.app.ReportAppIssue)
		v1.POST("/apps/:id/fallback-diagnostics", h.app.GetFallbackDiagnostics)
		v1.GET("/apps/:id/diagnostics", h.app.GetAppCompleteDiagnostics)
		v1.GET("/apps/:id/diagnostics/iframe-bridge", h.app.CheckAppIframeBridge)
		v1.GET("/apps/:id/diagnostics/status", h.app.GetAppScenarioStatus)
		v1.GET("/apps/:id/diagnostics/health", h.app.CheckAppHealth)
		v1.GET("/apps/:id/diagnostics/localhost", h.app.CheckAppLocalhostUsage)
		v1.GET("/apps/:id/completeness", h.app.GetAppCompleteness)
		v1.GET("/apps/:id/docs", h.app.GetAppDocuments)
		v1.GET("/apps/:id/docs/*path", h.app.GetAppDocument)
		v1.GET("/apps/:id/docs-search", h.app.SearchAppDocuments)
		v1.GET("/apps/:id/logs", h.app.GetAppLogs)
		v1.GET("/apps/:id/logs/lifecycle", h.app.GetAppLifecycleLogs)
		v1.GET("/apps/:id/logs/background", h.app.GetAppBackgroundLogs)
		v1.GET("/apps/:id/metrics", h.app.GetAppMetrics)

		// Log endpoints for scenarios using app name
		v1.GET("/logs/:appName", h.app.GetAppLogs)

		// Resource endpoints
		v1.GET("/resources", h.system.GetResources)
		v1.GET("/resources/:id", h.system.GetResourceDetails)
		v1.POST("/resources/:id/start", h.system.StartResource)
		v1.POST("/resources/:id/stop", h.system.StopResource)
		v1.GET("/resources/:id/status", h.system.GetResourceStatus)

		// Docker integration endpoints
		v1.GET("/docker/info", h.docker.GetDockerInfo)
		v1.GET("/docker/containers", h.docker.GetContainers)

		// Lighthouse testing endpoints
		v1.GET("/lighthouse/missing-configs", h.lighthouse.ListMissingConfigs)
		v1.POST("/scenarios/:scenario/lighthouse/run", h.lighthouse.RunLighthouse)
		v1.GET("/scenarios/:scenario/lighthouse/history", h.lighthouse.GetLighthouseHistory)
		v1.GET("/scenarios/:scenario/lighthouse/report/:reportId", h.lighthouse.GetLighthouseReport)

		// Tool Discovery Protocol endpoints
		v1.GET("/tools", h.tools.GetManifest)
		v1.GET("/tools/:name", h.tools.GetTool)
		v1.POST("/tools/execute", h.toolExecution.Execute)
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
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

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
	log.Printf("📊 API endpoints available at http://localhost:%s/api/v1", s.config.API.Port)
	log.Printf("🔧 Tool Discovery Protocol available at http://localhost:%s/api/v1/tools", s.config.API.Port)

	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed to start: %w", err)
	}

	// Wait for server context to be done
	<-serverCtx.Done()
	log.Println("✅ Server shutdown complete")

	return nil
}
