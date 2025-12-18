package main

import (
	"github.com/vrooli/api-core/preflight"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"landing-manager/handlers"
	"landing-manager/services"
	"landing-manager/util"
)

// Config holds minimal runtime configuration
type Config struct {
	Port        string
	DatabaseURL string
}

// Server wires the HTTP router and dependencies
type Server struct {
	config  *Config
	db      *sql.DB
	router  *mux.Router
	handler *handlers.Handler
}

// NewServer initializes configuration, database, services, and routes
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

	// Initialize services
	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)
	personaService := services.NewPersonaService(registry.GetTemplatesDir())
	previewService := services.NewPreviewService()
	analyticsService := services.NewAnalyticsService()

	// Create handler with all dependencies
	h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

	srv := &Server{
		config:  cfg,
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
	s.router.HandleFunc("/health", s.handler.HandleHealth).Methods("GET")
	s.router.HandleFunc("/api/v1/health", s.handler.HandleHealth).Methods("GET")

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

// Start launches the HTTP server with graceful shutdown
func (s *Server) Start() error {
	util.LogStructured("starting server", map[string]interface{}{
		"service": "landing-manager-api",
		"port":    s.config.Port,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.Port),
		Handler:      gorillahandlers.RecoveryHandler()(s.router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			util.LogStructuredError("server startup failed", map[string]interface{}{"error": err.Error()})
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

	util.LogStructured("server stopped", nil)
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

	return buildPostgresURL(
		strings.TrimSpace(os.Getenv("POSTGRES_HOST")),
		strings.TrimSpace(os.Getenv("POSTGRES_PORT")),
		strings.TrimSpace(os.Getenv("POSTGRES_USER")),
		strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD")),
		strings.TrimSpace(os.Getenv("POSTGRES_DB")),
	)
}

func buildPostgresURL(host, port, user, password, dbName string) (string, error) {
	if host == "" || port == "" || user == "" || password == "" || dbName == "" {
		return "", fmt.Errorf("DATABASE_URL or POSTGRES_HOST/PORT/USER/PASSWORD/DB must be set by the lifecycle system")
	}

	pgURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   dbName,
	}
	values := pgURL.Query()
	values.Set("sslmode", "disable")
	pgURL.RawQuery = values.Encode()

	return pgURL.String(), nil
}

// seedDefaultData is a no-op placeholder for database seeding
// kept for test compatibility
func seedDefaultData(db *sql.DB) error {
	// No default data to seed - this is intentionally a no-op
	return nil
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "landing-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	server, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
