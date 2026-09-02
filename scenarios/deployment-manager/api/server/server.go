package server

import (
	"context"
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
	"deployment-manager/dependencies"
	"deployment-manager/deployments"
	"deployment-manager/fitness"
	evidencehandler "deployment-manager/handlers/evidence"
	profileshandler "deployment-manager/handlers/profiles"
	"deployment-manager/health"
	internalEvidence "deployment-manager/internal/evidence"
	"deployment-manager/internal/modules"
	transport "deployment-manager/internal/transport"
	"deployment-manager/migrationtasks"
	"deployment-manager/profiles"
	"deployment-manager/releases"
	"deployment-manager/secrets"
	"deployment-manager/swaps"
	"deployment-manager/telemetry"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	evidenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence/evidencev1connect"
	profilesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/profiles/profilesv1connect"
	_ "modernc.org/sqlite"
)

// Server wires the HTTP router and database connection.
type Server struct {
	Config   *Config
	DB       interface{ Close() error }
	RoutedDB *database.RoutedDB
	Router   *mux.Router
	Logger   Logger

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
	ApprovalsHandler         *deployments.ApprovalsHandler
	PublishedVersionsHandler *deployments.PublishedVersionsHandler
	LPBSConfigHandler        *profiles.LPBSConfigHandler
	ReleasesHandler          *releases.Handler
	MigrationTasksHandler    *migrationtasks.Handler
	EvidencePath             string
	EvidenceHandler          http.Handler
	ProfilesConnectPath      string
	ProfilesConnectHandler   http.Handler
	ConnectRoutes            []transport.Route
	Orchestrator             *deployments.Orchestrator
	Handler                  http.Handler

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
	fileRoots, err := newFileRoots()
	if err != nil {
		return nil, fmt.Errorf("failed to configure file roots: %w", err)
	}
	// Connect to this scenario's own database. The path derives from the
	// scenario slug rather than from the environment, and the seam creates the
	// parent directory, so no pre-flight mkdir is needed here.
	routedDB, err := database.Open(context.Background(), database.Config{
		Driver:   database.DriverSQLite,
		Scenario: "deployment-manager",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Apply every domain schema once at boot. Schema failure is fatal: serving
	// against a partially initialized database would fabricate release state.
	if err := database.EnsureSchemas(context.Background(), routedDB.Primary(), modules.AllSchemas()...); err != nil {
		_ = routedDB.Close()
		return nil, fmt.Errorf("failed to ensure database schemas: %w", err)
	}

	// Create repositories through the RoutedDB seam.
	profilesRepo := profiles.NewSQLRepository(routedDB)

	// Signing is owned by scenario-to-desktop. deployment-manager retains only
	// the proxy repository seam and never persists signing material locally.
	signingRepo := codesigning.NewProxyRepository(profilesRepo)

	// Create domain handlers
	logFn := func(msg string, fields map[string]interface{}) {
		LogStructured(msg, fields)
	}

	// Create approvals repository and ensure schema
	approvalsRepo := deployments.NewSQLApprovalsRepository(routedDB)

	// Create published versions repository and ensure schema
	publishedVersionsRepo := deployments.NewSQLPublishedVersionsRepository(routedDB)

	// LPBS release-config repository (1:1 child of profiles).
	lpbsConfigRepo := profiles.NewSQLLPBSReleaseConfigRepository(routedDB)

	// Releases repository (canonical release records + per-platform rows).
	releasesRepo := releases.NewSQLRepository(routedDB)
	approvalsRepo.WithReadinessRepository(releasesRepo)

	// Evidence is a reference ledger owned by deployment-manager. Producers
	// retain their bytes; this service stores only target verdicts and references.
	evidenceRepo := internalEvidence.NewSQLRepository(routedDB, "sqlite")
	approvalsRepo.WithEvidenceRepository(evidenceRepo)
	evidencePath, evidenceHandler := evidenceconnect.NewEvidenceServiceHandler(evidencehandler.NewConnectHandler(evidenceRepo))
	profilesConnectPath, profilesConnectHandler := profilesconnect.NewProfilesServiceHandler(profileshandler.NewConnectHandler(profilesRepo))

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
		DB:                       routedDB.Primary(),
		RoutedDB:                 routedDB,
		Router:                   mux.NewRouter(),
		Logger:                   NewProcessLogger(),
		ProfilesRepo:             profilesRepo,
		SigningRepo:              signingRepo,
		HealthHandler:            health.NewHandler(routedDB),
		FitnessHandler:           fitness.NewHandler(logFn),
		TelemetryHandler:         telemetry.NewHandler(logFn),
		SecretsHandler:           secrets.NewHandler(profilesRepo, logFn),
		DependenciesHandler:      dependencies.NewHandler(logFn),
		SwapsHandler:             swaps.NewHandler(profilesRepo, logFn),
		DeploymentsHandler:       deployments.NewHandler(logFn),
		BundlesHandler:           bundles.NewHandlerWithSigning(secrets.NewClient(), profilesRepo, signingRepo, logFn),
		ProfilesHandler:          profiles.NewHandler(profilesRepo, logFn),
		SigningHandler:           codesigning.NewHandler(signingRepo, logFn),
		BuildHandler:             build.NewHandler(profilesRepo, logFn),
		ApprovalsHandler:         deployments.NewApprovalsHandler(approvalsRepo, logFn),
		PublishedVersionsHandler: deployments.NewPublishedVersionsHandler(publishedVersionsRepo, logFn),
		LPBSConfigHandler:        profiles.NewLPBSConfigHandler(profilesRepo, lpbsConfigRepo, logFn),
		MigrationTasksHandler:    migrationtasks.NewHandler(logFn),
		EvidencePath:             evidencePath,
		EvidenceHandler:          evidenceHandler,
		ProfilesConnectPath:      profilesConnectPath,
		ProfilesConnectHandler:   profilesConnectHandler,
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
	connectTransport := transport.NewHandler(
		srv.DependenciesHandler, srv.FitnessHandler, srv.DeploymentsHandler, srv.Orchestrator,
		srv.SwapsHandler, srv.TelemetryHandler.List, srv.TelemetryHandler.Upload, srv.MigrationTasksHandler.Report, srv.MigrationTasksHandler.Status,
		srv.ApprovalsHandler, srv.LPBSConfigHandler.Get, srv.LPBSConfigHandler.Upsert,
		srv.ReleasesHandler.ListByProfile, srv.ReleasesHandler.Get, srv.ReleasesHandler.Verify, srv.ReleasesHandler.Start,
	)
	srv.ConnectRoutes = transport.Routes(connectTransport)

	srv.setupRoutes()
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, routedDB, fileRoots)
	rootMux.Handle("/", srv.Router)
	srv.Handler = apihttp.TestModeMiddleware(SecurityHeadersMiddleware(rootMux))
	return srv, nil
}

func newFileRoots() (*filerouting.RoutedRoots, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return nil, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("deployment-manager")
	if err != nil {
		return nil, fmt.Errorf("resolve storage namespace: %w", err)
	}
	roots, err := storage.EnsureAllDirs(resolver, storage.Options{ScenarioID: scenarioID}, 0o755)
	if err != nil {
		return nil, fmt.Errorf("resolve storage roots: %w", err)
	}
	return filerouting.New(roots), nil
}

// Start launches the HTTP server with graceful shutdown.
func (s *Server) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.startReleaseFactPublisher(ctx)

	LogStructured("starting server", map[string]interface{}{
		"service": "deployment-manager-api",
		"port":    s.Config.Port,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.Config.Port),
		Handler:      handlers.RecoveryHandler()(s.Handler),
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
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
