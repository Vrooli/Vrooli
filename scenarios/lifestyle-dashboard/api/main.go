// DOC: docs/concepts/ARCHITECTURE.md
// DOC: docs/internal/STORAGE_AUDIT.md#ADR-003
// DOC: PRD.md#OT-P0-001
//
// Package main is the entry point for the Lifestyle Dashboard API.
// It wires HTTP handlers, repositories, and the SQLite database.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/retry"
	"github.com/vrooli/api-core/server"

	"lifestyle-dashboard/domain"
	"lifestyle-dashboard/handlers"
	"lifestyle-dashboard/repository"
)

// Server wires the HTTP router and repositories.
// It delegates request handling to the handlers package,
// which uses repository interfaces for storage abstraction.
type Server struct {
	db      *sql.DB
	router  *mux.Router
	handler *handlers.Handler
}

// NewServer initializes repositories and routes.
// This implements the Storage Architecture skill's abstraction pattern:
// - Handlers use repository interfaces (not direct DB)
// - Repositories encapsulate SQLite-specific queries
// - Business logic is testable without a database
func NewServer(db *sql.DB) *Server {
	// Create repository implementations
	eventRepo := repository.NewSQLiteEventRepository(db)
	domainRepo := repository.NewSQLiteDomainRepository(db)
	statsRepo := repository.NewSQLiteStatsRepository(db)
	storageRepo := repository.NewSQLiteStorageRepository(db)
	briefsRepo := repository.NewSQLiteBriefRepository(db)
	scoreConfigRepo := repository.NewSQLiteScoreConfigRepository(db)

	srv := &Server{
		db:      db,
		router:  mux.NewRouter(),
		handler: handlers.New(eventRepo, domainRepo, statsRepo, storageRepo, briefsRepo, scoreConfigRepo),
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)

	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	healthHandler := health.New().
		Version("1.0.0").
		Check(&sqliteChecker{db: s.db}, health.Critical).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Events API - P0-001, P0-003
	s.router.HandleFunc("/api/v1/events", s.handler.CreateEvent).Methods("POST")
	s.router.HandleFunc("/api/v1/events", s.handler.QueryEvents).Methods("GET")
	s.router.HandleFunc("/api/v1/events/{id}", s.handler.GetEvent).Methods("GET")

	// Domains API - P0-002
	s.router.HandleFunc("/api/v1/domains", s.handler.RegisterDomain).Methods("POST")
	s.router.HandleFunc("/api/v1/domains", s.handler.ListDomains).Methods("GET")
	s.router.HandleFunc("/api/v1/domains/{name}", s.handler.GetDomain).Methods("GET")
	s.router.HandleFunc("/api/v1/domains/{name}", s.handler.UpdateDomain).Methods("PATCH")
	s.router.HandleFunc("/api/v1/domains/{name}/health", s.handler.GetDomainHealth).Methods("GET")

	// Statistics API - P0-003, P0-004
	s.router.HandleFunc("/api/v1/stats/timeline", s.handler.GetTimeline).Methods("GET")
	s.router.HandleFunc("/api/v1/stats/summary", s.handler.GetSummary).Methods("GET")
	s.router.HandleFunc("/api/v1/stats/score", s.handler.GetScore).Methods("GET")

	// Score Configuration API - P1-003
	// [REQ:LD-SCORE-CALC] Configurable domain weights for lifestyle score
	s.router.HandleFunc("/api/v1/score/config", s.handler.GetScoreConfig).Methods("GET")
	s.router.HandleFunc("/api/v1/score/config/{domain}", s.handler.GetDomainWeight).Methods("GET")
	s.router.HandleFunc("/api/v1/score/config/{domain}", s.handler.UpdateDomainWeight).Methods("PUT")

	// Storage API - P0-006
	s.router.HandleFunc("/api/v1/storage", s.handler.GetStorageInfo).Methods("GET")
	s.router.HandleFunc("/api/v1/storage/events", s.handler.CleanupEvents).Methods("DELETE")

	// Briefs API - P0-005
	s.router.HandleFunc("/api/v1/briefs/current", s.handler.GetCurrentBrief).Methods("GET")
	s.router.HandleFunc("/api/v1/briefs/morning", s.handler.GetMorningBrief).Methods("GET")
	s.router.HandleFunc("/api/v1/briefs/evening", s.handler.GetEveningBrief).Methods("GET")
}

// Handler returns the HTTP handler with recovery middleware.
func (s *Server) Handler() http.Handler {
	return gorillaHandlers.RecoveryHandler()(s.router)
}

// =============================================================================
// SQLite Health Checker
// =============================================================================

type sqliteChecker struct {
	db *sql.DB
}

func (c *sqliteChecker) Check(ctx context.Context) health.CheckResult {
	if c.db == nil {
		return health.CheckResult{
			Name:      "database",
			Connected: false,
			Error:     fmt.Errorf("not configured"),
		}
	}
	start := time.Now()
	err := c.db.PingContext(ctx)
	latency := time.Since(start)

	if err != nil {
		return health.CheckResult{
			Name:      "database",
			Connected: false,
			Latency:   latency,
			Error:     err,
		}
	}

	return health.CheckResult{
		Name:      "database",
		Connected: true,
		Latency:   latency,
		Database:  "lifestyle.db",
	}
}

// =============================================================================
// Middleware
// =============================================================================

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow CORS from the UI server (local development and production)
		origin := r.Header.Get("Origin")
		if origin != "" {
			// Accept requests from localhost and common development origins
			// In production, this would be restricted to the specific UI domain
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// =============================================================================
// Main
// =============================================================================

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "lifestyle-dashboard",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Ensure SQLITE_PATH is set with WAL mode and busy timeout
	// The api-core/database package will read this automatically
	dbPath := os.Getenv("SQLITE_PATH")
	if dbPath == "" {
		dbPath = os.Getenv("SQLITE_DB")
	}
	if dbPath == "" {
		// Default to data directory
		dataDir := os.Getenv("SCENARIO_DATA_DIR")
		if dataDir == "" {
			dataDir = "."
		}
		dbPath = filepath.Join(dataDir, "lifestyle.db")
	}

	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Add SQLite-specific options to path for api-core
	dbPathWithOptions := dbPath + "?_journal_mode=WAL&_busy_timeout=5000"
	os.Setenv("SQLITE_PATH", dbPathWithOptions)

	log.Printf("Opening SQLite database at: %s", dbPath)

	// Connect using api-core/database package with automatic retry and jitter.
	// SQLite single-writer constraint is enforced via MaxOpenConns=1.
	ctx := context.Background()
	db, err := database.Connect(ctx, database.Config{
		Driver:       database.DriverSQLite,
		MaxOpenConns: 1, // SQLite single-writer constraint
		MaxIdleConns: 1,
		// Use minimal retry for SQLite (local file, no network issues)
		Retry: &retry.Config{
			MaxAttempts: 3,
			BaseDelay:   100 * time.Millisecond,
			MaxDelay:    500 * time.Millisecond,
		},
		Logger: func(format string, args ...interface{}) {
			log.Printf(format, args...)
		},
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize schema
	if err := domain.InitSchema(db); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	srv := NewServer(db)

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
