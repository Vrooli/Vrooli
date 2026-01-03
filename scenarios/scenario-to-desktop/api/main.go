package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"scenario-to-desktop-api/signing"
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
	builds         *BuildStore
	records        *DesktopRecordStore
	wineInstalls   map[string]*WineInstallStatus
	wineInstallMux sync.RWMutex
	templateDir    string
	logger         *slog.Logger
	telemetryMux   sync.Mutex
}

// NewServer creates a new server instance
func NewServer(port int) *Server {
	recordStore, err := NewDesktopRecordStore(filepath.Join(
		detectVrooliRoot(),
		"scenarios",
		"scenario-to-desktop",
		"data",
		"desktop_records.json",
	))
	if err != nil {
		log.Printf("⚠️  desktop record store unavailable: %v", err)
	}

	// Initialize structured logger with JSON output
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	server := &Server{
		router:       mux.NewRouter(),
		port:         port,
		builds:       NewBuildStore(),
		records:      recordStore,
		wineInstalls: make(map[string]*WineInstallStatus),
		templateDir:  "../templates", // Templates are in parent directory when running from api/
		logger:       logger,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check - use api-core/health for standardized response
	healthHandler := health.New().
		Version("1.0.0").
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// System status
	s.router.HandleFunc("/api/v1/status", s.statusHandler).Methods("GET")

	// Template management
	s.router.HandleFunc("/api/v1/templates", s.listTemplatesHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/templates/{type}", s.getTemplateHandler).Methods("GET")

	// Desktop application operations
	s.router.HandleFunc("/api/v1/desktop/generate", s.generateDesktopHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/generate/quick", s.quickGenerateDesktopHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/probe", s.probeEndpointsHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/preflight", s.preflightBundleHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/proxy-hints/{scenario_name}", s.proxyHintsHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/desktop/status/{build_id}", s.getBuildStatusHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/desktop/build", s.buildDesktopHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/test", s.testDesktopHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/package", s.packageDesktopHandler).Methods("POST")

	// Scenario discovery
	s.router.HandleFunc("/api/v1/scenarios/desktop-status", s.getScenarioDesktopStatusHandler).Methods("GET")

	// Deployment telemetry ingestion
	s.router.HandleFunc("/api/v1/deployment/telemetry", s.telemetryIngestHandler).Methods("POST")

	// Build by scenario name (simplified endpoint)
	s.router.HandleFunc("/api/v1/desktop/build/{scenario_name}", s.buildScenarioDesktopHandler).Methods("POST")

	// Download built packages
	s.router.HandleFunc("/api/v1/desktop/download/{scenario_name}/{platform}", s.downloadDesktopHandler).Methods("GET")

	// Delete desktop application
	s.router.HandleFunc("/api/v1/desktop/delete/{scenario_name}", s.deleteDesktopHandler).Methods("DELETE")

	// Resolve scenario ports via vrooli CLI (dynamic per lifecycle)
	s.router.HandleFunc("/api/v1/ports/{scenario}/{port_name}", s.getScenarioPortHandler).Methods("GET")

	// Deployment-manager bundle exports (resolved via api-core discovery)
	s.router.HandleFunc("/api/v1/deployment-manager/bundles/export", s.exportBundleHandler).Methods("POST")

	// Desktop records
	s.router.HandleFunc("/api/v1/desktop/records", s.listDesktopRecordsHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/desktop/records/{record_id}/move", s.moveDesktopRecordHandler).Methods("POST")

	// Test artifact cleanup (legacy CI outputs in /tmp)
	s.router.HandleFunc("/api/v1/desktop/test-artifacts", s.listTestArtifactsHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/desktop/test-artifacts/cleanup", s.cleanupTestArtifactsHandler).Methods("POST")

	// Webhook endpoints
	s.router.HandleFunc("/api/v1/desktop/webhook/build-complete", s.buildCompleteWebhookHandler).Methods("POST")

	// Docs for in-app browser
	s.router.HandleFunc("/api/v1/docs/manifest", s.docsManifestHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/docs/content", s.docsContentHandler).Methods("GET")
	s.router.HandleFunc("/docs/{docPath:.*}", s.docsFileHandler).Methods("GET")

	// System capabilities and dependencies
	s.router.HandleFunc("/api/v1/system/wine/check", s.checkWineHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/system/wine/install", s.installWineHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/system/wine/install/status/{install_id}", s.getWineInstallStatusHandler).Methods("GET")

	// Code signing configuration and management
	signingHandler := signing.NewHandler()
	signingHandler.RegisterRoutes(s.router)

	// Setup middleware - CORS must be registered before logging to handle OPTIONS requests correctly
	s.router.Use(corsMiddleware)
	s.router.Use(loggingMiddleware)
}

// Router returns the HTTP handler for use with server.Run
func (s *Server) Router() http.Handler {
	s.logger.Info("initializing server",
		"service", "scenario-to-desktop-api",
		"port", s.port,
		"endpoints", []string{"/api/v1/health", "/api/v1/status", "/api/v1/desktop/generate"})
	return handlers.RecoveryHandler()(s.router)
}

// Main function
func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "scenario-to-desktop",
	}) {
		return // Process was re-exec'd after rebuild
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

	// Create server
	srv := NewServer(port)

	// Start with graceful shutdown via api-core
	if err := server.Run(server.Config{
		Handler: srv.Router(),
		Port:    strconv.Itoa(port),
	}); err != nil {
		globalLogger.Error("server failed", "error", err)
		log.Fatal(err)
	}
}
