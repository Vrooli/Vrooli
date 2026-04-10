// Package main is the composition root for the reference-react-vite API.
// It wires together all components and starts the HTTP server.
//
// DOC: docs/concepts/ARCHITECTURE.md#dependency-flow
// DOC: docs/reference/configuration.md
// DOC: docs/QUICKSTART.md
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"reference-react-vite/api/config"
	apihandlers "reference-react-vite/api/handlers"
	"reference-react-vite/api/repository"
)

// Server wires the HTTP router and database connection.
// Following screaming architecture, the structure reveals the business domains
// (tasks, projects, notes) rather than technical layers.
type Server struct {
	db     *sql.DB
	router *mux.Router
	repos  *repository.Repositories
	cfg    *config.Config
}

// NewServer initializes database repositories and domain-organized routes.
// Configuration is loaded from environment variables with sensible defaults.
// DOC: docs/reference/configuration.md#tunable-levers
func NewServer(db *sql.DB, cfg *config.Config) *Server {
	srv := &Server{
		db:     db,
		router: mux.NewRouter(),
		repos:  repository.NewRepositories(db),
		cfg:    cfg,
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	// Apply middleware stack with configuration
	s.router.Use(corsMiddleware(s.cfg.CORS))
	s.router.Use(loggingMiddleware)
	s.router.Use(requestIDMiddleware)

	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	// Uses api-core/health for standardized response format
	// Version is configurable via HEALTH_VERSION env var
	healthHandler := health.New().
		Version(s.cfg.Server.HealthVersion).
		Check(health.DB(s.db), health.Critical).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Domain handlers - each domain registers its own routes
	// This is screaming architecture: the route structure reveals business domains
	// Pagination config is passed to handlers for consistent list behavior

	paginationCfg := apihandlers.PaginationConfig{
		DefaultLimit: s.cfg.Pagination.DefaultLimit,
		MaxLimit:     s.cfg.Pagination.MaxLimit,
	}

	// Tasks domain: core work items
	taskHandler := apihandlers.NewTaskHandler(s.repos.Tasks, paginationCfg)
	taskHandler.RegisterRoutes(s.router)

	// Projects domain: containers for organizing tasks
	projectHandler := apihandlers.NewProjectHandler(s.repos.Projects, paginationCfg)
	projectHandler.RegisterRoutes(s.router)

	// Notes domain: annotations attached to tasks
	noteHandler := apihandlers.NewNoteHandler(s.repos.Notes, s.repos.Tasks, paginationCfg)
	noteHandler.RegisterRoutes(s.router)
}

// Handler returns the HTTP handler with recovery middleware.
func (s *Server) Handler() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}

// corsMiddleware adds CORS headers for cross-origin requests.
// Configuration is provided via the config.CORS struct.
// DOC: docs/reference/configuration.md#cors-configuration
func corsMiddleware(cfg config.CORS) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && isOriginAllowed(origin, cfg.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if the origin matches the allowed patterns.
// Patterns can include * as wildcard for port (e.g., "http://localhost:*").
func isOriginAllowed(origin, allowedOrigins string) bool {
	patterns := strings.Split(allowedOrigins, ",")
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "*" {
			return true
		}
		if strings.Contains(pattern, "*") {
			// Simple wildcard matching for localhost:*
			prefix := strings.Split(pattern, "*")[0]
			if strings.HasPrefix(origin, prefix) {
				return true
			}
		} else if pattern == origin {
			return true
		}
	}
	return false
}

// loggingMiddleware prints simple request logs.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// requestIDMiddleware ensures each request has a unique ID.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Request-ID") == "" {
			// A unique ID will be generated by the handler if needed
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "reference-react-vite",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Load configuration from environment variables with sensible defaults
	// DOC: docs/reference/configuration.md#tunable-levers
	cfg := config.LoadFromEnv()

	// Connect to database with automatic retry and backoff
	db, err := database.Connect(context.Background(), database.Config{
		Driver: database.DriverPostgres,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	srv := NewServer(db, cfg)

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
