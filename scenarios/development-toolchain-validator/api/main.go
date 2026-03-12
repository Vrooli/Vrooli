// DOC: docs/concepts/ARCHITECTURE.md#system-overview
// DOC: docs/internal/SEAMS.md#architecture-alignment-update
// DOC: docs/reference/configuration.md
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

	"development-toolchain-validator/domain/reference"
	"development-toolchain-validator/domain/skill"
	apihandlers "development-toolchain-validator/handlers"
	"development-toolchain-validator/infrastructure/postgres"
	"development-toolchain-validator/internal/config"
)

// Server wires the HTTP router, database, and domain services.
type Server struct {
	db     *sql.DB
	router *mux.Router
	config config.Config

	// Domain services
	referenceService *reference.Service
	skillService     *skill.Service
}

// NewServer initializes database connections, repositories, services, and routes.
// Configuration is loaded from environment variables with sensible defaults.
func NewServer(db *sql.DB) *Server {
	// Load configuration from environment
	cfg := config.LoadFromEnv()

	// Initialize repositories (storage layer)
	referenceRepo := postgres.NewReferenceRepository(db)
	skillRepo := postgres.NewSkillRepository(db)

	// Initialize services (business logic layer) with configuration
	serviceConfig := reference.ServiceConfig{
		Pagination: cfg.Pagination,
		Validation: cfg.Validation,
	}
	referenceService := reference.NewService(referenceRepo, reference.WithConfig(serviceConfig))
	skillService := skill.NewService(skillRepo)

	srv := &Server{
		db:               db,
		router:           mux.NewRouter(),
		config:           cfg,
		referenceService: referenceService,
		skillService:     skillService,
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	s.router.Use(s.corsMiddleware)

	// Health endpoints (infrastructure and client paths)
	healthHandler := health.New().
		Version("1.0.0").
		Check(health.DB(s.db), health.Critical).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Domain handlers
	referenceHandler := apihandlers.NewReferenceHandler(s.referenceService)
	referenceHandler.RegisterRoutes(s.router)

	skillHandler := apihandlers.NewSkillHandler(s.skillService)
	skillHandler.RegisterRoutes(s.router)

	// NOTE: Expectation handlers require postgres repositories that don't exist yet.
	// The expectation domain has service and handlers but no persistence layer.
	// See PROBLEMS.md for follow-up tasks.
}

// Handler returns the HTTP handler with recovery middleware.
func (s *Server) Handler() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}

// loggingMiddleware prints simple request logs.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// corsMiddleware adds CORS headers based on configuration.
// In production, CORS_ALLOWED_ORIGINS should be set to specific origins.
// See docs/reference/configuration.md for configuration details.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if the origin is allowed using centralized config
		if origin != "" && s.config.CORS.IsOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "development-toolchain-validator",
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
