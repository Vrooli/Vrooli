package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
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
	db     *sql.DB
	router *mux.Router
}

// NewServer initializes database and routes
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
	healthHandler := health.New().
		Version("1.0.0").
		Check(health.DB(s.db), health.Critical).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Resource endpoints (health must be before {name} to avoid capture)
	s.router.HandleFunc("/api/v1/resources/health", s.handleResourceHealth).Methods("GET")
	s.router.HandleFunc("/api/v1/resources", s.handleListResources).Methods("GET")
	s.router.HandleFunc("/api/v1/resources/{name}", s.handleGetResource).Methods("GET")

	// Onboarding progress endpoints
	s.router.HandleFunc("/api/v1/progress", s.handleGetProgress).Methods("GET")
	s.router.HandleFunc("/api/v1/progress", s.handleUpdateProgress).Methods("PUT")

	// Config endpoints
	s.router.HandleFunc("/api/v1/config/generate", s.handleConfigGenerate).Methods("POST")
	s.router.HandleFunc("/api/v1/config/validate", s.handleConfigValidate).Methods("POST")
	s.router.HandleFunc("/api/v1/config/export", s.handleConfigExport).Methods("POST")

	// Setup order endpoint
	s.router.HandleFunc("/api/v1/setup-order", s.handleSetupOrder).Methods("GET")

	// Glossary endpoint
	s.router.HandleFunc("/api/v1/glossary", s.handleGlossary).Methods("GET")
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
		ScenarioName: "vrooli-onboarding",
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

	srv := NewServer(db)

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
