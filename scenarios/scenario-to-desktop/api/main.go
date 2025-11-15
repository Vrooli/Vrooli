package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Global logger for middleware and initialization code
var globalLogger *slog.Logger

func init() {
	// Initialize global structured logger with JSON output
	globalLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Server represents the API server
type Server struct {
	router         *mux.Router
	port           int
	buildStatuses  map[string]*BuildStatus
	wineInstalls   map[string]*WineInstallStatus
	buildMutex     sync.RWMutex
	wineInstallMux sync.RWMutex
	templateDir    string
	logger         *slog.Logger
}

// NewServer creates a new server instance
func NewServer(port int) *Server {
	// Initialize structured logger with JSON output
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	server := &Server{
		router:        mux.NewRouter(),
		port:          port,
		buildStatuses: make(map[string]*BuildStatus),
		wineInstalls:  make(map[string]*WineInstallStatus),
		templateDir:   "../templates", // Templates are in parent directory when running from api/
		logger:        logger,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check at root level (required for lifecycle system)
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")

	// Health check (also available under API prefix)
	s.router.HandleFunc("/api/v1/health", s.healthHandler).Methods("GET")

	// System status
	s.router.HandleFunc("/api/v1/status", s.statusHandler).Methods("GET")

	// Template management
	s.router.HandleFunc("/api/v1/templates", s.listTemplatesHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/templates/{type}", s.getTemplateHandler).Methods("GET")

	// Desktop application operations
	s.router.HandleFunc("/api/v1/desktop/generate", s.generateDesktopHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/generate/quick", s.quickGenerateDesktopHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/status/{build_id}", s.getBuildStatusHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/desktop/build", s.buildDesktopHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/test", s.testDesktopHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/package", s.packageDesktopHandler).Methods("POST")

	// Scenario discovery
	s.router.HandleFunc("/api/v1/scenarios/desktop-status", s.getScenarioDesktopStatusHandler).Methods("GET")

	// Build by scenario name (simplified endpoint)
	s.router.HandleFunc("/api/v1/desktop/build/{scenario_name}", s.buildScenarioDesktopHandler).Methods("POST")

	// Download built packages
	s.router.HandleFunc("/api/v1/desktop/download/{scenario_name}/{platform}", s.downloadDesktopHandler).Methods("GET")

	// Delete desktop application
	s.router.HandleFunc("/api/v1/desktop/delete/{scenario_name}", s.deleteDesktopHandler).Methods("DELETE")

	// Webhook endpoints
	s.router.HandleFunc("/api/v1/desktop/webhook/build-complete", s.buildCompleteWebhookHandler).Methods("POST")

	// System capabilities and dependencies
	s.router.HandleFunc("/api/v1/system/wine/check", s.checkWineHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/system/wine/install", s.installWineHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/system/wine/install/status/{install_id}", s.getWineInstallStatusHandler).Methods("GET")

	// Setup middleware - CORS must be registered before logging to handle OPTIONS requests correctly
	s.router.Use(corsMiddleware)
	s.router.Use(loggingMiddleware)
}

// Start starts the server
func (s *Server) Start() error {
	s.logger.Info("starting server",
		"service", "scenario-to-desktop-api",
		"port", s.port,
		"endpoints", []string{"/api/v1/health", "/api/v1/status", "/api/v1/desktop/generate"})

	// Setup graceful shutdown
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      handlers.RecoveryHandler()(s.router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("server startup failed", "error", err)
			log.Fatal(err)
		}
	}()

	s.logger.Info("server started successfully",
		"url", fmt.Sprintf("http://localhost:%d", s.port),
		"health_endpoint", fmt.Sprintf("http://localhost:%d/api/v1/health", s.port),
		"status_endpoint", fmt.Sprintf("http://localhost:%d/api/v1/status", s.port))

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("shutdown signal received", "timeout", "10s")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.logger.Info("server stopped successfully")
	return nil
}

// Main function
func main() {
	// SECURITY: Validate required environment variable - fail fast if not set
	lifecycleManaged := os.Getenv("VROOLI_LIFECYCLE_MANAGED")
	if lifecycleManaged != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start scenario-to-desktop

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// SECURITY: Validate port environment variables - prefer API_PORT, fallback to PORT
	port := 15200
	apiPortStr := os.Getenv("API_PORT")
	portStr := os.Getenv("PORT")

	if apiPortStr != "" {
		p, err := strconv.Atoi(apiPortStr)
		if err != nil {
			log.Fatalf("❌ Invalid API_PORT value '%s': must be a valid integer", apiPortStr)
		}
		if p < 1024 || p > 65535 {
			log.Fatalf("❌ Invalid API_PORT value %d: must be between 1024 and 65535", p)
		}
		port = p
	} else if portStr != "" {
		// Fallback to PORT for compatibility
		p, err := strconv.Atoi(portStr)
		if err != nil {
			log.Fatalf("❌ Invalid PORT value '%s': must be a valid integer", portStr)
		}
		if p < 1024 || p > 65535 {
			log.Fatalf("❌ Invalid PORT value %d: must be between 1024 and 65535", p)
		}
		port = p
	} else {
		globalLogger.Warn("no port configuration found",
			"message", "No API_PORT or PORT environment variable set",
			"action", "using default port",
			"default_port", port)
	}

	// Create and start server
	server := NewServer(port)
	if err := server.Start(); err != nil {
		globalLogger.Error("server failed", "error", err)
		log.Fatal(err)
	}
}
