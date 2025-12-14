package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Config holds minimal runtime configuration
type Config struct {
	Port        string
	DatabaseURL string
}

// Server wires the HTTP router and database connection
type Server struct {
	config               *Config
	db                   *sql.DB
	store                *TidinessStore
	router               *mux.Router
	campaignMgr          *CampaignManager
	campaignOrchestrator *AutoCampaignOrchestrator
	scanCoordinator      *ScanCoordinator
	scenarioLocator      *ScenarioLocator
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	dbURL, err := resolveDatabaseURL()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:        requireEnv("API_PORT"),
		DatabaseURL: dbURL,
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := NewTidinessStore(db)
	srv := &Server{
		config:          cfg,
		db:              db,
		store:           store,
		router:          mux.NewRouter(),
		campaignMgr:     NewCampaignManager(),
		scenarioLocator: NewScenarioLocator(5 * time.Minute),
	}

	srv.scanCoordinator = NewScanCoordinator(
		db,
		srv.scenarioLocator,
		srv.log,
		srv.persistDetailedFileMetrics,
		srv.persistFileMetrics,
		store.StoreLintTypeIssues,
		srv.storeAIIssue,
		srv.recordScanHistory,
	)

	// Best-effort creation; handlers will guard against nil orchestrator
	if orchestrator, err := NewAutoCampaignOrchestrator(db, srv.campaignMgr); err == nil {
		srv.campaignOrchestrator = orchestrator
	} else {
		srv.log("failed to initialize campaign orchestrator", map[string]interface{}{"error": err.Error()})
	}

	srv.setupRoutes()
	return srv, nil
}

func (s *Server) setupRoutes() {
	// Add CORS middleware for all routes
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)

	// Expose health at both root (for infrastructure) and /api/v1 (for clients)
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/api/v1/health", s.handleHealth).Methods("GET")

	// Light scanning endpoints
	s.router.HandleFunc("/api/v1/scan/light", s.handleLightScan).Methods("POST")
	s.router.HandleFunc("/api/v1/scan/light/parse-lint", s.handleParseLint).Methods("POST")
	s.router.HandleFunc("/api/v1/scan/light/parse-type", s.handleParseType).Methods("POST")

	// Smart scanning endpoints (TM-SS-001, TM-SS-002)
	s.router.HandleFunc("/api/v1/scan/smart", s.handleSmartScan).Methods("POST", "OPTIONS")

	// Agent API endpoints (TM-API-001 through TM-API-007)
	s.router.HandleFunc("/api/v1/agent/issues", s.handleAgentGetIssues).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/agent/issues", s.handleAgentStoreIssue).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/agent/issues/{id}", s.handleAgentUpdateIssue).Methods("PATCH", "OPTIONS")
	s.router.HandleFunc("/api/v1/agent/issues/generate-from-metrics", s.handleGenerateIssuesFromMetrics).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/agent/staleness", s.handleGetStalenessInfo).Methods("GET", "OPTIONS")

	// Scenario detail endpoint (OT-P0-010) - must be registered before generic list endpoint
	s.router.HandleFunc("/api/v1/agent/scenarios/{name}", s.handleAgentGetScenarioDetail).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/agent/scenarios", s.handleAgentGetScenarios).Methods("GET", "OPTIONS")

	// Refactor recommendations endpoint - combines visited-tracker + file metrics
	s.router.HandleFunc("/api/v1/agent/refactor-recommendations", s.handleRefactorRecommendations).Methods("GET", "OPTIONS")

	// Tidiness score endpoint - provides aggregate tidiness metrics for ecosystem-manager
	// Supports two URL patterns for compatibility:
	// - /api/v1/scenarios/{scenario}/tidiness (preferred, RESTful)
	// - /api/v1/scan/{scenario} (legacy, for ecosystem-manager compatibility)
	s.router.HandleFunc("/api/v1/scenarios/{scenario}/tidiness", s.handleGetTidinessScore).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/scan/{scenario}", s.handleGetTidinessScore).Methods("GET", "OPTIONS")

	// Auto-campaign endpoints (OT-P1-001, OT-P1-002)
	s.router.HandleFunc("/api/v1/campaigns", s.handleCreateCampaign).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/campaigns", s.handleListCampaigns).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/campaigns/{id}", s.handleGetCampaign).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/campaigns/{id}/action", s.handleCampaignAction).Methods("POST", "OPTIONS")
}

// Start launches the HTTP server with graceful shutdown
func (s *Server) Start() error {
	s.log("starting server", map[string]interface{}{
		"service": "tidiness-manager-api",
		"port":    s.config.Port,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.Port),
		Handler:      handlers.RecoveryHandler()(s.router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log("server startup failed", map[string]interface{}{"error": err.Error()})
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

	s.log("server stopped", nil)
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	dbStatus := "connected"

	if s.db == nil {
		status = "unhealthy"
		dbStatus = "not initialized"
	} else if err := s.db.PingContext(r.Context()); err != nil {
		status = "unhealthy"
		dbStatus = "disconnected"
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "Tidiness Manager API",
		"version":   "1.0.0",
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]string{
			"database": dbStatus,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// corsMiddleware adds CORS headers to allow cross-origin requests from the UI
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from localhost origins only
		origin := r.Header.Get("Origin")
		if origin != "" && (strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:")) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware prints structured request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		// Structured logging for better observability
		logJSON(map[string]interface{}{
			"method":   r.Method,
			"uri":      r.RequestURI,
			"duration": time.Since(start).String(),
			"type":     "request",
		})
	})
}

func (s *Server) log(msg string, fields map[string]interface{}) {
	if len(fields) == 0 {
		logJSON(map[string]interface{}{
			"message": msg,
		})
		return
	}
	fields["message"] = msg
	logJSON(fields)
}

// logJSON outputs structured JSON logs for better observability
func logJSON(fields map[string]interface{}) {
	fields["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	if data, err := json.Marshal(fields); err == nil {
		log.Println(string(data))
	} else {
		log.Printf("ERROR: failed to marshal log: %v", err)
	}
}

// getCampaignOrchestrator returns a ready orchestrator, initializing it if needed.
func (s *Server) getCampaignOrchestrator() (*AutoCampaignOrchestrator, error) {
	if s.campaignOrchestrator != nil {
		return s.campaignOrchestrator, nil
	}

	if s.campaignMgr == nil {
		s.campaignMgr = NewCampaignManager()
	}

	if s.db == nil {
		return nil, fmt.Errorf("campaign orchestrator dependencies missing")
	}

	orchestrator, err := NewAutoCampaignOrchestrator(s.db, s.campaignMgr)
	if err != nil {
		return nil, err
	}

	s.campaignOrchestrator = orchestrator
	return orchestrator, nil
}

func (s *Server) ensureScanCoordinator() error {
	if s.scanCoordinator != nil {
		return nil
	}

	if s.scenarioLocator == nil {
		s.scenarioLocator = NewScenarioLocator(5 * time.Minute)
	}

	if s.db == nil {
		return fmt.Errorf("scan coordinator dependencies missing")
	}

	s.scanCoordinator = NewScanCoordinator(
		s.db,
		s.scenarioLocator,
		s.log,
		s.persistDetailedFileMetrics,
		s.persistFileMetrics,
		s.store.StoreLintTypeIssues,
		s.storeAIIssue,
		s.recordScanHistory,
	)
	return nil
}

func requireEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("environment variable %s is required. Run the scenario via 'vrooli scenario run <name>' so lifecycle exports it.", key)
	}
	return value
}

func resolveDatabaseURL() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		return raw, nil
	}

	host := strings.TrimSpace(os.Getenv("POSTGRES_HOST"))
	port := strings.TrimSpace(os.Getenv("POSTGRES_PORT"))
	user := strings.TrimSpace(os.Getenv("POSTGRES_USER"))
	password := strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD"))

	// Use SCENARIO_NAME for the database name when running in scenario mode
	// This allows each scenario to have its own isolated database
	name := strings.TrimSpace(os.Getenv("SCENARIO_NAME"))
	if name == "" {
		name = strings.TrimSpace(os.Getenv("POSTGRES_DB"))
	}

	if host == "" || port == "" || user == "" || password == "" || name == "" {
		// Security: Don't leak which specific variables are missing
		return "", fmt.Errorf("database configuration incomplete - required environment variables must be set by lifecycle system")
	}

	pgURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   name,
	}
	values := pgURL.Query()
	// Security Note: SSL disabled for local development. In production deployments,
	// use sslmode=require or sslmode=verify-full with appropriate certificates.
	values.Set("sslmode", "disable")
	pgURL.RawQuery = values.Encode()

	return pgURL.String(), nil
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start tidiness-manager

💡 The lifecycle system provides environment variables, port allocation,
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
