package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	agentconfig "agent-manager/internal/config"
	"agent-manager/internal/database"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/handlers"
	healthstore "agent-manager/internal/health"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/pricing"
	"agent-manager/internal/repository"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/stats"
	"agent-manager/internal/storage"
	"agent-manager/internal/wiring"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

// Server owns lifecycle sequencing around the wiring-owned service graph.
type Server struct {
	db                    *database.DB
	router                *mux.Router
	orchestrator          *orchestration.Orchestrator
	statsService          orchestration.StatsService
	statsRepo             repository.StatsRepository
	pricingService        pricing.Service
	wsHub                 *handlers.WebSocketHub
	reconciler            *orchestration.Reconciler
	awaitRegistry         *orchestration.AwaitRegistry
	workflowNudger        *orchestration.WorkflowNudger
	modelHealthProbe      *healthstore.Probe
	rolePolicyState       *rolepolicy.State
	permissionPolicyState *permissionpolicy.State
	permissionPolicy      *permissionpolicy.Service
	storage               storage.Service
	statsEngine           *stats.Engine
	healthStore           *healthstore.Store
	eventRepo             eventlog.Repository
}

// NewServer builds the graph, then starts durable recovery in dependency order.
func NewServer() (*Server, error) {
	levers, leversErr := agentconfig.LoadLevers()
	if levers == nil {
		defaults := agentconfig.DefaultLevers()
		levers = &defaults
	}
	obs.Init(levers.Observability.LogFormat, levers.Observability.LogLevel)
	if leversErr != nil {
		obs.Logger().Warn("config levers failed to load; using defaults", obs.KeyError, leversErr.Error())
	}
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	db, err := database.NewConnection(logger)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	wsHub := handlers.NewWebSocketHub()
	go wsHub.Run()
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "/tmp/agent-manager-uploads"
	}
	uploadStorage := storage.NewLocalService(uploadDir)
	deps, err := wiring.NewOrchestrator(db, wsHub, logger, uploadStorage, levers)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build orchestrator: %w", err)
	}
	srv := &Server{
		db: db, router: mux.NewRouter().UseEncodedPath(), orchestrator: deps.Orchestrator,
		statsService: deps.StatsService, statsRepo: deps.StatsRepository, pricingService: deps.PricingService,
		wsHub: wsHub, reconciler: deps.Reconciler, awaitRegistry: deps.AwaitRegistry, workflowNudger: deps.WorkflowNudger,
		modelHealthProbe: deps.ModelHealthProbe, rolePolicyState: deps.RolePolicyState, permissionPolicyState: deps.PermissionPolicyState,
		permissionPolicy: deps.PermissionPolicy, storage: uploadStorage, statsEngine: deps.StatsEngine,
		healthStore: deps.HealthStore, eventRepo: deps.EventRepository,
	}
	srv.startRecovery()
	srv.setupRoutes()
	return srv, nil
}

func envOrEmpty(key string) string { return os.Getenv(key) }

func (s *Server) startRecovery() {
	ctx := context.Background()
	if s.reconciler != nil {
		if err := s.reconciler.RecoverInFlightRuns(ctx); err != nil {
			obs.Logger().Warn("initial run recovery failed", obs.KeyError, err.Error())
		}
		if err := s.reconciler.Start(ctx); err != nil {
			obs.Logger().Warn("reconciler start failed", obs.KeyError, err.Error())
		}
	}
	if err := s.orchestrator.RecoverWorkflowExecutions(ctx); err != nil {
		obs.Logger().Warn("initial workflow recovery failed", obs.KeyError, err.Error())
	}
	if s.workflowNudger != nil {
		s.workflowNudger.Start()
	}
	repoRoot := os.Getenv("PROJECT_ROOT")
	if repoRoot == "" {
		repoRoot, _ = filepath.Abs(filepath.Join("..", "..", ".."))
	}
	summary := s.orchestrator.ReconcileDeclaringScenarios(ctx, repoRoot)
	obs.Logger().Info("scenario declaration sweep complete", "scanned", summary.Scanned, "declaring", summary.Declaring, "reconciled", summary.Reconciled, "failed", summary.Failed)
	if result, err := s.orchestrator.ReconcileSelfDeclarations(ctx, repoRoot); err != nil {
		obs.Logger().Warn("agent-manager self-declaration registration failed", obs.KeyError, err.Error())
	} else {
		obs.Logger().Info("agent-manager self-declaration registration complete", "profiles_created", result.ProfilesCreated, "profiles_updated", result.ProfilesUpdated, "workflows_created", result.WorkflowsCreated, "workflows_activated", result.WorkflowsActivated, "failed", result.ProfilesFailed+result.WorkflowsFailed)
	}
	wiring.ScheduleDeclarationReconcile(s.orchestrator, repoRoot)
	if s.awaitRegistry != nil {
		if n, err := s.awaitRegistry.RecoverParkedRuns(ctx); err != nil {
			obs.Logger().Warn("parked-run waiter recovery failed", obs.KeyError, err.Error())
		} else if n > 0 {
			obs.Logger().Info("re-spawned waiters for parked runs", "count", n)
		}
	}
	if s.modelHealthProbe != nil {
		s.modelHealthProbe.Start(ctx)
	}
	if s.statsEngine != nil {
		if err := s.statsEngine.Rebuild(ctx); err != nil {
			obs.Logger().Warn("stats engine rebuild failed", obs.KeyError, err.Error())
		}
	}
}

func (s *Server) setupRoutes() {
	wiring.SetupRoutes(s.router, wiring.RouteDependencies{
		DB: s.db, Orchestrator: s.orchestrator, StatsService: s.statsService, StatsRepository: s.statsRepo,
		PricingService: s.pricingService, WebSocketHub: s.wsHub, RolePolicyState: s.rolePolicyState,
		PermissionPolicyState: s.permissionPolicyState, PermissionPolicy: s.permissionPolicy, Storage: s.storage,
		StatsEngine: s.statsEngine, HealthStore: s.healthStore, EventRepository: s.eventRepo,
	})
}

func (s *Server) Router() http.Handler { return gorillaHandlers.RecoveryHandler()(s.router) }

func (s *Server) Cleanup() error {
	wiring.Shutdown(s.db, s.reconciler, s.awaitRegistry, s.workflowNudger)
	return nil
}

func main() {
	if preflight.Run(preflight.Config{ScenarioName: "agent-manager"}) {
		return
	}
	srv, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}
	if err := server.Run(server.Config{Handler: srv.Router(), WriteTimeout: 3 * time.Minute, ReadTimeout: time.Minute, Cleanup: func(context.Context) error { return srv.Cleanup() }}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
