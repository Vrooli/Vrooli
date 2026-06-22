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

	"deployment-manager/build"
	"deployment-manager/bundles"
	"deployment-manager/codesigning"
	"deployment-manager/codesigning/validation"
	"deployment-manager/dependencies"
	"deployment-manager/deployments"
	"deployment-manager/fitness"
	"deployment-manager/health"
	"deployment-manager/migrationtasks"
	"deployment-manager/profiles"
	"deployment-manager/releases"
	"deployment-manager/secrets"
	"deployment-manager/swaps"
	"deployment-manager/telemetry"
	visualvalidation "deployment-manager/validation"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
)

// Server wires the HTTP router and database connection.
type Server struct {
	Config *Config
	DB     *sql.DB
	Router *mux.Router

	// Domain handlers
	HealthHandler            *health.Handler
	FitnessHandler           *fitness.Handler
	TelemetryHandler         *telemetry.Handler
	SecretsHandler           *secrets.Handler
	DependenciesHandler      *dependencies.Handler
	SwapsHandler             *swaps.Handler
	DeploymentsHandler       *deployments.Handler
	BundlesHandler           *bundles.Handler
	ProfilesHandler          *profiles.Handler
	SigningHandler           *codesigning.Handler
	BuildHandler             *build.Handler
	ValidationHandler        *visualvalidation.Handler
	ApprovalsHandler         *deployments.ApprovalsHandler
	PublishedVersionsHandler *deployments.PublishedVersionsHandler
	LPBSConfigHandler        *profiles.LPBSConfigHandler
	ReleasesHandler          *releases.Handler
	MigrationTasksHandler    *migrationtasks.Handler
	Orchestrator             *deployments.Orchestrator

	// Repositories
	ProfilesRepo   profiles.Repository
	SigningRepo    codesigning.Repository // Interface to allow SQL or Proxy implementation
	LPBSConfigRepo profiles.LPBSReleaseConfigRepository
	ReleasesRepo   releases.Repository
}

// New initializes configuration, database, and routes.
func New() (*Server, error) {
	cfg := &Config{
		Port: RequireEnv("API_PORT"),
	}

	// Connect to database with automatic retry and backoff.
	// Reads POSTGRES_* environment variables set by the lifecycle system.
	db, err := database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Create repositories
	profilesRepo := profiles.NewSQLRepository(db)

	// Determine signing repository based on configuration
	// SIGNING_PROXY_ENABLED=true routes signing to scenario-to-desktop service
	var signingRepo codesigning.Repository
	if os.Getenv("SIGNING_PROXY_ENABLED") == "true" {
		LogStructured("using proxy signing repository", map[string]interface{}{
			"target": os.Getenv("SCENARIO_TO_DESKTOP_URL"),
		})
		signingRepo = codesigning.NewProxyRepository(profilesRepo)
	} else {
		sqlSigningRepo := codesigning.NewSQLRepository(db)
		// Ensure signing schema is up to date (SQL mode only)
		if err := sqlSigningRepo.EnsureSchema(context.Background()); err != nil {
			LogStructured("warning: failed to ensure signing schema", map[string]interface{}{"error": err.Error()})
			// Non-fatal - signing endpoints will fail gracefully
		}
		signingRepo = sqlSigningRepo
	}

	// Create domain handlers
	logFn := func(msg string, fields map[string]interface{}) {
		LogStructured(msg, fields)
	}

	// Create signing validators for pre-deployment checks
	signingValidator := validation.NewValidator()
	signingChecker := validation.NewPrerequisiteChecker()
	signingValidatorAdapter := deployments.NewSigningValidatorAdapter(signingRepo, signingValidator, signingChecker)

	// Create approvals repository and ensure schema
	approvalsRepo := deployments.NewSQLApprovalsRepository(db)
	if err := approvalsRepo.EnsureSchema(context.Background()); err != nil {
		LogStructured("warning: failed to ensure approvals schema", map[string]interface{}{"error": err.Error()})
	}

	// Create published versions repository and ensure schema
	publishedVersionsRepo := deployments.NewSQLPublishedVersionsRepository(db)
	if err := publishedVersionsRepo.EnsureSchema(context.Background()); err != nil {
		LogStructured("warning: failed to ensure published versions schema", map[string]interface{}{"error": err.Error()})
	}

	// LPBS release-config repository (1:1 child of profiles).
	lpbsConfigRepo := profiles.NewSQLLPBSReleaseConfigRepository(db)
	if err := lpbsConfigRepo.EnsureSchema(context.Background()); err != nil {
		LogStructured("warning: failed to ensure lpbs config schema", map[string]interface{}{"error": err.Error()})
	}

	// Releases repository (canonical release records + per-platform rows).
	releasesRepo := releases.NewSQLRepository(db)
	if err := releasesRepo.EnsureSchema(context.Background()); err != nil {
		LogStructured("warning: failed to ensure releases schema", map[string]interface{}{"error": err.Error()})
	}

	// Best-effort inter-scenario clients; the orchestrator skips the matching
	// step if a client is nil, and logs a warning on construction failure.
	var cloudClient deployments.CloudHealthClient
	if c, err := deployments.NewHTTPCloudHealthClient(logFn); err == nil {
		cloudClient = c
	} else {
		LogStructured("cloud health client unavailable", map[string]interface{}{"error": err.Error()})
	}
	var lpbsClient deployments.LPBSReleaseClient
	if c, err := deployments.NewHTTPLPBSReleaseClient(deployments.LPBSClientConfig{Log: logFn}); err == nil {
		lpbsClient = c
	} else {
		LogStructured("lpbs release client unavailable", map[string]interface{}{"error": err.Error()})
	}

	srv := &Server{
		Config:                   cfg,
		DB:                       db,
		Router:                   mux.NewRouter(),
		ProfilesRepo:             profilesRepo,
		SigningRepo:              signingRepo,
		HealthHandler:            health.NewHandler(db),
		FitnessHandler:           fitness.NewHandler(logFn),
		TelemetryHandler:         telemetry.NewHandler(logFn),
		SecretsHandler:           secrets.NewHandler(profilesRepo, logFn),
		DependenciesHandler:      dependencies.NewHandler(logFn),
		SwapsHandler:             swaps.NewHandler(profilesRepo, logFn),
		DeploymentsHandler:       deployments.NewHandlerWithSigning(logFn, signingValidatorAdapter),
		BundlesHandler:           bundles.NewHandlerWithSigning(secrets.NewClient(), profilesRepo, signingRepo, logFn),
		ProfilesHandler:          profiles.NewHandler(profilesRepo, logFn),
		SigningHandler:           codesigning.NewHandler(signingRepo, signingValidator, signingChecker, logFn),
		BuildHandler:             build.NewHandler(profilesRepo, logFn),
		ValidationHandler:        visualvalidation.NewHandler(visualvalidation.NewSQLRepository(db), approvalsRepo, db, validationVideoDir(), logFn),
		ApprovalsHandler:         deployments.NewApprovalsHandler(approvalsRepo, logFn),
		PublishedVersionsHandler: deployments.NewPublishedVersionsHandler(publishedVersionsRepo, logFn),
		LPBSConfigHandler:        profiles.NewLPBSConfigHandler(profilesRepo, lpbsConfigRepo, logFn),
		MigrationTasksHandler:    migrationtasks.NewHandler(logFn),
		LPBSConfigRepo:           lpbsConfigRepo,
		ReleasesRepo:             releasesRepo,
		Orchestrator: deployments.NewOrchestratorFull(
			profilesRepo, approvalsRepo, publishedVersionsRepo,
			releasesRepo, lpbsConfigRepo, cloudClient, lpbsClient, logFn,
		),
	}
	srv.ReleasesHandler = releases.NewHandler(
		releasesRepo, lpbsConfigRepo, releasesVerifierAdapter{inner: lpbsClient}, srv.Orchestrator, logFn,
	)

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
