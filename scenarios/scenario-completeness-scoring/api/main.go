// Package main provides the entry point for the Scenario Completeness Scoring API.
// This file is intentionally minimal - it handles only server bootstrapping and
// route configuration. All business logic is delegated to domain-organized
// handler packages (handlers/scores.go, handlers/config.go, etc.)
//
// Architecture: "Screaming Architecture"
// - The top-level structure reflects the domain (scores, config, health, analysis)
// - HTTP handling is infrastructure that serves the domain
// - Each handler file groups endpoints by the domain concept they serve
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"scenario-completeness-scoring/pkg/analysis"
	"scenario-completeness-scoring/pkg/circuitbreaker"
	"scenario-completeness-scoring/pkg/collectors"
	"scenario-completeness-scoring/pkg/config"
	"scenario-completeness-scoring/pkg/handlers"
	"scenario-completeness-scoring/pkg/health"
	"scenario-completeness-scoring/pkg/history"

	gorillahndlrs "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// ServerConfig holds runtime configuration for the server
type ServerConfig struct {
	Port       string
	VrooliRoot string
}

// Server orchestrates the HTTP server and its dependencies
type Server struct {
	config    *ServerConfig
	router    *mux.Router
	handlers  *handlers.Context
	historyDB *history.DB
}

// NewServer initializes the server with all required dependencies
// [REQ:SCS-CB-001] Initialize circuit breaker with default config
// [REQ:SCS-HEALTH-001] Initialize health tracker
// [REQ:SCS-CFG-004] Initialize config loader
// [REQ:SCS-HIST-001] Initialize history database and repository
// [REQ:SCS-ANALYSIS-001] Initialize what-if analyzer
// [REQ:SCS-ANALYSIS-003] Initialize bulk refresher
func NewServer() (*Server, error) {
	cfg := &ServerConfig{
		Port:       requireEnv("API_PORT"),
		VrooliRoot: getEnvWithDefault("VROOLI_ROOT", os.Getenv("HOME")+"/Vrooli"),
	}

	// Initialize circuit breaker registry with default config
	cbRegistry := circuitbreaker.NewRegistry(circuitbreaker.DefaultConfig())

	// Initialize history database
	// [REQ:SCS-HIST-002] SQLite database for history storage
	dataDir := filepath.Join(cfg.VrooliRoot, "scenarios", "scenario-completeness-scoring", "data")
	historyDB, err := history.NewDB(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize history database: %w", err)
	}
	historyRepo := history.NewRepository(historyDB)
	trendAnalyzer := history.NewTrendAnalyzer(historyRepo, 5) // Stall after 5 unchanged

	// Initialize metrics collector with circuit breaker integration
	// [REQ:SCS-CORE-003] Graceful degradation via circuit breaker
	collector := collectors.NewMetricsCollectorWithCircuitBreaker(cfg.VrooliRoot, cbRegistry)
	configLoader := config.NewLoader(cfg.VrooliRoot)

	// Create handler context with all dependencies
	handlerCtx := handlers.NewContext(
		cfg.VrooliRoot,
		collector,
		cbRegistry,
		health.NewTracker(cbRegistry),
		configLoader,
		historyDB,
		historyRepo,
		trendAnalyzer,
		analysis.NewWhatIfAnalyzer(collector),
		analysis.NewBulkRefresher(cfg.VrooliRoot, collector, historyRepo),
	)

	srv := &Server{
		config:    cfg,
		router:    mux.NewRouter(),
		handlers:  handlerCtx,
		historyDB: historyDB,
	}

	srv.setupRoutes()
	return srv, nil
}

// setupRoutes configures all API routes, organized by domain concept
func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)

	h := s.handlers

	// ─────────────────────────────────────────────────────────────────────
	// Health & Infrastructure (basic service health)
	// ─────────────────────────────────────────────────────────────────────
	s.router.HandleFunc("/health", h.HandleHealth).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/health", h.HandleHealth).Methods("GET", "OPTIONS")

	// ─────────────────────────────────────────────────────────────────────
	// Scores Domain (core business: calculating & retrieving scores)
	// [REQ:SCS-CORE-002]
	// ─────────────────────────────────────────────────────────────────────
	s.router.HandleFunc("/api/v1/scores", h.HandleListScores).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/scores/{scenario}", h.HandleGetScore).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/scores/{scenario}/calculate", h.HandleCalculateScore).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/scores/{scenario}/validation-analysis", h.HandleValidationAnalysis).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/recommendations/{scenario}", h.HandleGetRecommendations).Methods("GET", "OPTIONS")

	// ─────────────────────────────────────────────────────────────────────
	// Configuration Domain (scoring settings & presets)
	// [REQ:SCS-CFG-001] [REQ:SCS-CFG-002] [REQ:SCS-CFG-003] [REQ:SCS-CFG-004]
	// ─────────────────────────────────────────────────────────────────────
	s.router.HandleFunc("/api/v1/config", h.HandleGetConfig).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/config", h.HandleUpdateConfig).Methods("PUT", "OPTIONS")
	s.router.HandleFunc("/api/v1/config/thresholds", h.HandleGetThresholds).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/config/thresholds/{category}", h.HandleGetCategoryThresholds).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/config/scenarios/{scenario}", h.HandleGetScenarioConfig).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/config/scenarios/{scenario}", h.HandleUpdateScenarioConfig).Methods("PUT", "OPTIONS")
	s.router.HandleFunc("/api/v1/config/scenarios/{scenario}", h.HandleDeleteScenarioConfig).Methods("DELETE", "OPTIONS")
	s.router.HandleFunc("/api/v1/config/presets", h.HandleListPresets).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/config/presets/{name}/apply", h.HandleApplyPreset).Methods("POST", "OPTIONS")

	// ─────────────────────────────────────────────────────────────────────
	// Health Monitoring Domain (collector health & circuit breakers)
	// [REQ:SCS-HEALTH-001] [REQ:SCS-CB-004]
	// ─────────────────────────────────────────────────────────────────────
	s.router.HandleFunc("/api/v1/health/collectors", h.HandleGetCollectorHealth).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/health/collectors/{name}/test", h.HandleTestCollector).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/health/circuit-breaker", h.HandleGetCircuitBreakers).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/health/circuit-breaker/reset", h.HandleResetAllCircuitBreakers).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/health/circuit-breaker/{collector}/reset", h.HandleResetCircuitBreaker).Methods("POST", "OPTIONS")

	// ─────────────────────────────────────────────────────────────────────
	// Analysis Domain (history, trends, what-if, comparisons)
	// [REQ:SCS-HIST-001] [REQ:SCS-HIST-003] [REQ:SCS-ANALYSIS-001] [REQ:SCS-ANALYSIS-003] [REQ:SCS-ANALYSIS-004]
	// ─────────────────────────────────────────────────────────────────────
	s.router.HandleFunc("/api/v1/scores/{scenario}/history", h.HandleGetHistory).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/scores/{scenario}/trends", h.HandleGetTrends).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/trends", h.HandleGetAllTrends).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/scores/{scenario}/what-if", h.HandleWhatIf).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/scores/refresh-all", h.HandleBulkRefresh).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/compare", h.HandleCompare).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/analysis/components", h.HandleListAnalysisComponents).Methods("GET", "OPTIONS")
}

// Start launches the HTTP server with graceful shutdown
func (s *Server) Start() error {
	log.Printf("starting server | service=scenario-completeness-scoring-api port=%s vrooli_root=%s",
		s.config.Port, s.config.VrooliRoot)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.Port),
		Handler:      gorillahndlrs.RecoveryHandler()(s.router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server startup failed: %v", err)
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	// Close history database
	if s.historyDB != nil {
		if err := s.historyDB.Close(); err != nil {
			log.Printf("failed to close history database: %v", err)
		}
	}

	log.Println("server stopped")
	return nil
}

// corsMiddleware adds CORS headers for cross-origin requests
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware prints request logs with timing
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// requireEnv retrieves a required environment variable or exits
func requireEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("environment variable %s is required. Run the scenario via 'vrooli scenario run <name>' so lifecycle exports it.", key)
	}
	return value
}

// getEnvWithDefault retrieves an environment variable with a fallback default
func getEnvWithDefault(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `This binary must be run through the Vrooli lifecycle system.

Instead, use:
   vrooli scenario start scenario-completeness-scoring

The lifecycle system provides environment variables, port allocation,
and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	server, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
