package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

// Server wires the HTTP router and database connection
type Server struct {
	db            *sql.DB
	router        *mux.Router
	routeSvc      *RouteService
	portAuditor   *PortAuditor
	tunnelHealth  *TunnelHealthChecker
	probeSvc      *ProbeService
}

// NewServer initializes services and routes
func NewServer(db *sql.DB) *Server {
	routeSvc := NewRouteService(db)
	scenariosRoot := detectScenariosRoot()

	srv := &Server{
		db:           db,
		router:       mux.NewRouter(),
		routeSvc:     routeSvc,
		portAuditor:  NewPortAuditor(routeSvc, scenariosRoot),
		tunnelHealth: NewTunnelHealthChecker(),
		probeSvc:     NewProbeService(db, routeSvc),
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)

	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	healthHandler := health.New().
		Version("1.0.0").
		Check(health.DB(s.db), health.Critical).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Route manifest CRUD
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/routes", handleListRoutes(s.routeSvc)).Methods("GET")
	api.HandleFunc("/routes", handleCreateRoute(s.routeSvc)).Methods("POST")
	api.HandleFunc("/routes/{id:[0-9]+}", handleGetRoute(s.routeSvc)).Methods("GET")
	api.HandleFunc("/routes/{id:[0-9]+}", handleUpdateRoute(s.routeSvc)).Methods("PUT", "PATCH")
	api.HandleFunc("/routes/{id:[0-9]+}", handleDeleteRoute(s.routeSvc)).Methods("DELETE")

	// Port compliance audit
	api.HandleFunc("/audit/ports", handlePortAudit(s.portAuditor)).Methods("GET")

	// Tunnel health
	api.HandleFunc("/tunnel/health", handleTunnelHealth(s.tunnelHealth)).Methods("GET")

	// Liveness probes
	api.HandleFunc("/probes", handleRunProbes(s.probeSvc)).Methods("POST")
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

// detectScenariosRoot finds the scenarios directory relative to the API binary.
func detectScenariosRoot() string {
	// Try environment variable first
	if root := os.Getenv("SCENARIOS_ROOT"); root != "" {
		return root
	}
	// Default: assume we're in scenarios/<name>/api/
	wd, err := os.Getwd()
	if err != nil {
		return "/home/matthalloran8/Vrooli/scenarios"
	}
	return filepath.Join(wd, "..", "..")
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "tunnel-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Connect to database with automatic retry and backoff
	db, err := database.Connect(context.Background(), database.Config{
		Driver: database.DriverPostgres,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	// Ensure schema tables exist
	ensureSchema(db)

	srv := NewServer(db)

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
