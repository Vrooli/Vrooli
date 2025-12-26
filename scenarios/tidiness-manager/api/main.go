package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

// Config holds minimal runtime configuration
type Config struct {
	Port string
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
	// Connect to database with automatic retry and backoff.
	// Reads POSTGRES_* environment variables set by the lifecycle system.
	db, err := database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	store := NewTidinessStore(db)
	srv := &Server{
		config:          &Config{},
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

// Router returns the HTTP handler for use with server.Run
func (s *Server) Router() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}

// Cleanup releases resources when the server shuts down
func (s *Server) Cleanup() error {
	s.log("cleaning up resources", nil)
	if s.db != nil {
		return s.db.Close()
	}
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

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "tidiness-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	srv, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Run(server.Config{
		Handler: srv.Router(),
		Cleanup: func(ctx context.Context) error {
			return srv.Cleanup()
		},
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
