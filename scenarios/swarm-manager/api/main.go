// Package main provides the Swarm Manager API server.
//
// DOC: docs/concepts/ARCHITECTURE.md#logical-architecture
// DOC: docs/internal/SEAMS.md#api-to-database-seam
// DOC: docs/internal/INTENT.md#api-components
//
// # Purpose
//
// This API serves as the backend for the Swarm Manager UI, providing endpoints
// for managing the scenario ecosystem: ideas backlog, scenario lifecycle, and
// recommendations.
//
// # Current Status: Ideas CRUD Implemented (File-Based)
//
// The API provides:
//   - Health check endpoints at / and /api/v1/health
//   - Ideas CRUD endpoints at /api/v1/ideas (GET, POST, PUT, DELETE)
//
// # Architecture
//
// The server uses:
//   - gorilla/mux for HTTP routing
//   - api-core/health for standardized health responses
//   - api-core/server for graceful shutdown
//   - File-system based storage for ideas (git-tracked in scenarios/swarm-manager/ideas/)
//   - PostgreSQL connection is OPTIONAL (enabled when POSTGRES_DB is set)
//
// # Implemented Endpoints (P0)
//
//	GET    /api/v1/ideas          - List all ideas
//	POST   /api/v1/ideas          - Create new idea
//	GET    /api/v1/ideas/{name}   - Get idea by name
//	PUT    /api/v1/ideas/{name}   - Update idea
//	DELETE /api/v1/ideas/{name}   - Delete idea
//
// # Scenario Endpoints (P0)
//
//	GET    /api/v1/scenarios      - List all scenarios (supports search, filter, sort)
//	GET    /api/v1/scenarios/{name} - Get scenario details
//
// Related PRD targets: OT-P0-002, OT-P0-005, OT-P0-006
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/retry"
	"github.com/vrooli/api-core/server"

	"swarm-manager/internal/ideas"
	"swarm-manager/internal/scenarios"
)

// Server wires the HTTP router and optional database connection.
// Database is only connected when POSTGRES_DB environment variable is set.
type Server struct {
	db     *sql.DB // May be nil when running in file-only mode
	router *mux.Router
}

// NewServer initializes routes. Database connection is optional.
func NewServer(db *sql.DB) *Server {
	srv := &Server{
		db:     db,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)

	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	// Uses api-core/health for standardized response format
	healthBuilder := health.New().Version("1.0.0")
	if s.db != nil {
		// Only add DB health check when database is connected
		healthBuilder = healthBuilder.Check(health.DB(s.db), health.Critical)
	}
	healthHandler := healthBuilder.Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Ideas CRUD endpoints
	// [REQ:REQ-P0-002] Ideas backlog management
	ideasHandler := ideas.NewHandler("")
	ideasHandler.RegisterRoutes(s.router)

	// Scenarios catalog endpoints
	// [REQ:REQ-P0-006] Scenario catalog with priority, search, and filter
	scenariosHandler := scenarios.NewHandler("")
	scenariosHandler.RegisterRoutes(s.router)
}

// Handler returns the HTTP handler with recovery middleware
func (s *Server) Handler() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "swarm-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Database connection is optional. Only connect when POSTGRES_DB is set.
	// The ideas CRUD is file-based, so the API can run without a database.
	var db *sql.DB
	var cleanup func(ctx context.Context) error

	if os.Getenv("POSTGRES_DB") != "" {
		var err error
		// Use short retry config since database is optional for the ideas API.
		// If postgres isn't available, fail fast and continue in file-only mode.
		db, err = database.Connect(context.Background(), database.Config{
			Driver: database.DriverPostgres,
			Retry: &retry.Config{
				MaxAttempts: 3, // Only 3 attempts instead of 10
				BaseDelay:   500 * time.Millisecond,
				MaxDelay:    2 * time.Second, // Cap at 2s instead of 30s
			},
		})
		if err != nil {
			log.Printf("Warning: Database connection failed (running in file-only mode): %v", err)
			db = nil
		} else {
			cleanup = func(ctx context.Context) error { return db.Close() }
			log.Printf("Database connected successfully")
		}
	} else {
		log.Printf("Running in file-only mode (POSTGRES_DB not set)")
	}

	srv := NewServer(db)

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler: srv.Handler(),
		Cleanup: cleanup,
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
