package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"landing-manager/handlers"
	"landing-manager/services"
)

// Server wires the HTTP router and dependencies
type Server struct {
	db      *sql.DB
	router  *mux.Router
	handler *handlers.Handler
}

// NewServer initializes database, services, and routes
func NewServer() (*Server, error) {
	// Connect to database with automatic retry and backoff
	db, err := database.Connect(context.Background(), database.Config{
		Driver: database.DriverPostgres,
	})
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	// Initialize services
	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)
	personaService := services.NewPersonaService(registry.GetTemplatesDir())
	previewService := services.NewPreviewService()
	analyticsService := services.NewAnalyticsService()

	// Create handler with all dependencies
	h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

	srv := &Server{
		db:      db,
		router:  mux.NewRouter(),
		handler: h,
	}

	srv.setupRoutes()
	return srv, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(handlers.LoggingMiddleware)
	s.router.Use(handlers.RequestSizeLimitMiddleware)

	// Health endpoints
	healthHandler := health.New().Version("1.0.0").Check(health.DB(s.db), health.Critical).Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Template management
	s.router.HandleFunc("/api/v1/templates", s.handler.HandleTemplateList).Methods("GET")
	s.router.HandleFunc("/api/v1/templates/{id}", s.handler.HandleTemplateShow).Methods("GET")
	s.router.HandleFunc("/api/v1/generate", s.handler.HandleGenerate).Methods("POST")
	s.router.HandleFunc("/api/v1/customize", s.handler.HandleCustomize).Methods("POST")
	s.router.HandleFunc("/api/v1/generated", s.handler.HandleGeneratedList).Methods("GET")
	s.router.HandleFunc("/api/v1/preview/{scenario_id}", s.handler.HandlePreviewLinks).Methods("GET")

	// Personas
	s.router.HandleFunc("/api/v1/personas", s.handler.HandlePersonaList).Methods("GET")
	s.router.HandleFunc("/api/v1/personas/{id}", s.handler.HandlePersonaShow).Methods("GET")

	// Analytics
	s.router.HandleFunc("/api/v1/analytics/summary", s.handler.HandleAnalyticsSummary).Methods("GET")
	s.router.HandleFunc("/api/v1/analytics/events", s.handler.HandleAnalyticsEvents).Methods("GET")

	// Template-only placeholders (features that belong to generated scenarios)
	s.router.HandleFunc("/api/v1/admin/login", handlers.HandleTemplateOnly("admin authentication")).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/logout", handlers.HandleTemplateOnly("admin authentication")).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/session", handlers.HandleTemplateOnly("admin authentication")).Methods("GET")

	s.router.HandleFunc("/api/v1/variants/select", handlers.HandleTemplateOnly("variant selection")).Methods("GET")
	s.router.HandleFunc("/api/v1/variants", handlers.HandleTemplateOnly("variant management")).Methods("GET")
	s.router.HandleFunc("/api/v1/variants", handlers.HandleTemplateOnly("variant management")).Methods("POST")
	s.router.HandleFunc("/api/v1/variants/{slug}", handlers.HandleTemplateOnly("variant management")).Methods("GET")
	s.router.HandleFunc("/api/v1/variants/{slug}", handlers.HandleTemplateOnly("variant management")).Methods("PATCH")
	s.router.HandleFunc("/api/v1/variants/{slug}/archive", handlers.HandleTemplateOnly("variant management")).Methods("POST")
	s.router.HandleFunc("/api/v1/variants/{slug}", handlers.HandleTemplateOnly("variant management")).Methods("DELETE")

	s.router.HandleFunc("/api/v1/metrics/track", handlers.HandleTemplateOnly("metrics tracking")).Methods("POST")
	s.router.HandleFunc("/api/v1/metrics/summary", handlers.HandleTemplateOnly("metrics summary")).Methods("GET")
	s.router.HandleFunc("/api/v1/metrics/variants", handlers.HandleTemplateOnly("metrics variant stats")).Methods("GET")

	s.router.HandleFunc("/api/v1/checkout/create", handlers.HandleTemplateOnly("checkout")).Methods("POST")
	s.router.HandleFunc("/api/v1/webhooks/stripe", handlers.HandleTemplateOnly("Stripe webhooks")).Methods("POST")
	s.router.HandleFunc("/api/v1/subscription/verify", handlers.HandleTemplateOnly("subscription verification")).Methods("GET")
	s.router.HandleFunc("/api/v1/subscription/cancel", handlers.HandleTemplateOnly("subscription cancel")).Methods("POST")

	s.router.HandleFunc("/api/v1/variants/{variant_id}/sections", handlers.HandleTemplateOnly("section listing")).Methods("GET")
	s.router.HandleFunc("/api/v1/sections/{id}", handlers.HandleTemplateOnly("section detail")).Methods("GET")
	s.router.HandleFunc("/api/v1/sections/{id}", handlers.HandleTemplateOnly("section update")).Methods("PATCH")
	s.router.HandleFunc("/api/v1/sections", handlers.HandleTemplateOnly("section creation")).Methods("POST")
	s.router.HandleFunc("/api/v1/sections/{id}", handlers.HandleTemplateOnly("section delete")).Methods("DELETE")

	// Lifecycle management for generated scenarios
	s.router.HandleFunc("/api/v1/lifecycle/{scenario_id}/start", s.handler.HandleScenarioStart).Methods("POST")
	s.router.HandleFunc("/api/v1/lifecycle/{scenario_id}/stop", s.handler.HandleScenarioStop).Methods("POST")
	s.router.HandleFunc("/api/v1/lifecycle/{scenario_id}/restart", s.handler.HandleScenarioRestart).Methods("POST")
	s.router.HandleFunc("/api/v1/lifecycle/{scenario_id}/status", s.handler.HandleScenarioStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/lifecycle/{scenario_id}/logs", s.handler.HandleScenarioLogs).Methods("GET")
	s.router.HandleFunc("/api/v1/lifecycle/{scenario_id}/promote", s.handler.HandleScenarioPromote).Methods("POST")
	s.router.HandleFunc("/api/v1/lifecycle/{scenario_id}", s.handler.HandleScenarioDelete).Methods("DELETE")
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "landing-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	srv, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	// Start server with graceful shutdown
	if err := server.Run(server.Config{
		Handler: gorillahandlers.RecoveryHandler()(srv.router),
		Cleanup: func(ctx context.Context) error { return srv.db.Close() },
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
