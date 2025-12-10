package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"deployment-manager/bundles"
	"deployment-manager/codesigning"
	"deployment-manager/codesigning/validation"
	"deployment-manager/dependencies"
	"deployment-manager/deployments"
	"deployment-manager/fitness"
	"deployment-manager/health"
	"deployment-manager/profiles"
	"deployment-manager/secrets"
	"deployment-manager/swaps"
	"deployment-manager/telemetry"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Server wires the HTTP router and database connection.
type Server struct {
	Config *Config
	DB     *sql.DB
	Router *mux.Router

	// Domain handlers
	HealthHandler       *health.Handler
	FitnessHandler      *fitness.Handler
	TelemetryHandler    *telemetry.Handler
	SecretsHandler      *secrets.Handler
	DependenciesHandler *dependencies.Handler
	SwapsHandler        *swaps.Handler
	DeploymentsHandler  *deployments.Handler
	BundlesHandler      *bundles.Handler
	ProfilesHandler     *profiles.Handler
	SigningHandler      *codesigning.Handler

	// Repositories
	ProfilesRepo profiles.Repository
	SigningRepo  *codesigning.SQLRepository
}

// New initializes configuration, database, and routes.
func New() (*Server, error) {
	dbURL, err := ResolveDatabaseURL()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:        RequireEnv("API_PORT"),
		DatabaseURL: dbURL,
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create repositories
	profilesRepo := profiles.NewSQLRepository(db)
	signingRepo := codesigning.NewSQLRepository(db)

	// Ensure signing schema is up to date
	if err := signingRepo.EnsureSchema(context.Background()); err != nil {
		LogStructured("warning: failed to ensure signing schema", map[string]interface{}{"error": err.Error()})
		// Non-fatal - signing endpoints will fail gracefully
	}

	// Create domain handlers
	logFn := func(msg string, fields map[string]interface{}) {
		LogStructured(msg, fields)
	}

	// Create signing validators for pre-deployment checks
	signingValidator := validation.NewValidator()
	signingChecker := validation.NewPrerequisiteChecker()
	signingValidatorAdapter := deployments.NewSigningValidatorAdapter(signingRepo, signingValidator, signingChecker)

	srv := &Server{
		Config:              cfg,
		DB:                  db,
		Router:              mux.NewRouter(),
		ProfilesRepo:        profilesRepo,
		SigningRepo:         signingRepo,
		HealthHandler:       health.NewHandler(db),
		FitnessHandler:      fitness.NewHandler(logFn),
		TelemetryHandler:    telemetry.NewHandler(logFn),
		SecretsHandler:      secrets.NewHandler(profilesRepo, logFn),
		DependenciesHandler: dependencies.NewHandler(logFn),
		SwapsHandler:        swaps.NewHandler(profilesRepo, logFn),
		DeploymentsHandler:  deployments.NewHandlerWithSigning(logFn, signingValidatorAdapter),
		BundlesHandler:      bundles.NewHandlerWithSigning(secrets.NewClient(), profilesRepo, signingRepo, logFn),
		ProfilesHandler:     profiles.NewHandler(profilesRepo, logFn),
		SigningHandler:      codesigning.NewHandler(signingRepo, signingValidator, signingChecker, logFn),
	}

	srv.setupRoutes()
	return srv, nil
}

// Start launches the HTTP server with graceful shutdown.
func (s *Server) Start() error {
	LogStructured("starting server", map[string]interface{}{
		"service": "deployment-manager-api",
		"port":    s.Config.Port,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.Config.Port),
		Handler:      handlers.RecoveryHandler()(s.Router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			LogStructured("server startup failed", map[string]interface{}{"error": err.Error()})
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

	LogStructured("server stopped", nil)
	return nil
}

// WriteJSON sends a JSON response with the given status code.
func (s *Server) WriteJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
