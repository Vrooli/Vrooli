package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"agent-manager/internal/adapters/database"
	capabilities "agent-manager/internal/capabilities"
	agentconfig "agent-manager/internal/config"
	"agent-manager/internal/conversationpolicy"
	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/evidenceretention"
	"agent-manager/internal/handlers"
	healthstore "agent-manager/internal/health"
	"agent-manager/internal/interventions"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/maintenance"
	"agent-manager/internal/modelpolicydrift"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/pricing"
	"agent-manager/internal/repository"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/storage"
	"agent-manager/internal/supervision"
	"agent-manager/internal/usageforecast"
	"agent-manager/internal/wiring"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/apihttp"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	corestorage "github.com/vrooli/api-core/storage"
	searchregister "github.com/vrooli/searchregister-go"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
)

type searchControlTokens struct {
	mu     sync.RWMutex
	tokens map[string]string
}

func newSearchControlTokens() *searchControlTokens {
	return &searchControlTokens{tokens: make(map[string]string)}
}

func (h *searchControlTokens) set(providerID, token string) {
	if h == nil || strings.TrimSpace(providerID) == "" || strings.TrimSpace(token) == "" {
		return
	}
	h.mu.Lock()
	h.tokens[providerID] = token
	h.mu.Unlock()
}

func (h *searchControlTokens) get(providerID string) string {
	if h == nil {
		return ""
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.tokens[providerID]
}

// Server owns lifecycle sequencing around the wiring-owned service graph.
type Server struct {
	recovery                 *maintenance.Recovery
	maintenance              *maintenance.Handler
	lifecycleService         *maintenance.LifecycleService
	capabilityRegistry       *capabilities.Registry
	db                       *database.DB
	fileRoots                *filerouting.RoutedRoots
	router                   *mux.Router
	orchestrator             *orchestration.Orchestrator
	statsRepo                repository.StatsRepository
	pricingService           pricing.Service
	pricingRepository        pricing.Repository
	wsHub                    *handlers.WebSocketHub
	reconciler               *orchestration.Reconciler
	awaitRegistry            *orchestration.AwaitRegistry
	workflowNudger           *orchestration.WorkflowNudger
	transcriptImporter       *orchestration.TranscriptIndex
	claudeQuotaFeed          *pricing.ClaudeStatuslineReader
	usageAlerts              *usageforecast.Alerter
	interventions            *interventions.Job
	modelHealthProbe         *healthstore.Probe
	modelPolicyDrift         *modelpolicydrift.Scheduler
	rolePolicyState          *rolepolicy.State
	permissionPolicyState    *permissionpolicy.State
	permissionPolicy         *permissionpolicy.Service
	storage                  storage.Service
	healthStore              *healthstore.Store
	eventRepo                eventlog.Repository
	supervisionService       *supervision.Service
	supervisionScheduler     *supervision.Scheduler
	watchActionAuthorizer    handlers.WatchActionAuthorizer
	invocationReadModel      invocationreadmodel.Store
	conversationSearch       *conversationsearch.Service
	conversationSemantic     *conversationsearch.SemanticRuntime
	conversationIndexer      *conversationsearch.Indexer
	conversationSearchFile   string
	conversationSearchPolicy *conversationpolicy.Policy
	conversationTokens       *searchControlTokens
	searchRegistrationStop   context.CancelFunc
	storageHealth            *storageHealthRuntime
	evidenceRetention        *evidenceretention.Service
	evidenceApprovals        *evidenceretention.Approvals
	workspaceSandbox         interface {
		IsAvailable(context.Context) (bool, string)
	}
}

// databaseConfigFromLevers keeps the production pool aligned with Agent
// Manager's governed storage settings. SQLite runs in WAL mode, so retaining
// more than one connection lets conversation reads proceed while incremental
// projection publication holds the writer connection.
func databaseConfigFromLevers(dsn string, storage agentconfig.StorageLevers) coredb.Config {
	// The generic storage default is sized for client/server databases. On this
	// large SQLite/WAL file, opening dozens of pragma-initialized connections and
	// allowing concurrent projection scans amplifies disk contention. Five keeps
	// one writer plus bounded readers without reverting to single-connection
	// head-of-line blocking.
	maxOpen := min(storage.MaxOpenConns, 5)
	maxIdle := min(storage.MaxIdleConns, maxOpen)
	return coredb.Config{
		Driver:          coredb.DriverSQLite,
		DSN:             dsn,
		MaxOpenConns:    maxOpen,
		MaxIdleConns:    maxIdle,
		ConnMaxLifetime: storage.ConnMaxLifetime,
	}
}

// NewServer builds the graph and routes before launching observable recovery.
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
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{AppID: "vrooli", Profile: corestorage.ProfileAuto})
	if err != nil {
		return nil, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := corestorage.ScenarioNamespace("agent-manager")
	if err != nil {
		return nil, fmt.Errorf("resolve scenario storage namespace: %w", err)
	}
	storagePaths, err := resolver.Resolve(corestorage.Options{ScenarioID: scenarioID})
	if err != nil {
		return nil, fmt.Errorf("resolve scenario storage roots: %w", err)
	}
	fileRoots := filerouting.New(storagePaths)
	dsn, err := database.SQLiteDSN(logger)
	if err != nil {
		return nil, fmt.Errorf("resolve database configuration: %w", err)
	}
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer dbCancel()
	databaseConfig := databaseConfigFromLevers(dsn, levers.Storage)
	databaseConfig.Logger = logger.Printf
	routedDB, err := coredb.Open(dbCtx, databaseConfig)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	db := database.NewRoutedDB(routedDB, logger)
	if err := db.InitializeSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize database schema: %w", err)
	}
	routedDB.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		return database.NewDB(sqlx.NewDb(pool, "sqlite"), logger).InitializeSchema()
	})
	cursorKey := make([]byte, 32)
	if _, err := rand.Read(cursorKey); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize conversation search cursor signer: %w", err)
	}
	conversationRepository := conversationsearch.NewSQLiteRepository(db)
	searchVisibilityPolicy, err := levers.ConversationPolicy.Policy()
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("compile conversation visibility policy: %w", err)
	}
	conversationSource, err := conversationsearch.NewSQLiteSource(db, conversationsearch.MustNormalizer(conversationsearch.NormalizerConfig{}), conversationsearch.WithVisibilityPolicy(searchVisibilityPolicy))
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize conversation search source: %w", err)
	}
	repoRoot := strings.TrimSpace(os.Getenv("PROJECT_ROOT"))
	if repoRoot == "" {
		repoRoot, _ = filepath.Abs(filepath.Join("..", "..", ".."))
	}
	semanticCtx, semanticCancel := context.WithTimeout(context.Background(), 10*time.Second)
	conversationSearchFile := filepath.Join(repoRoot, "scenarios", "agent-manager", ".vrooli", "search.json")
	semanticRuntime, semanticConfigErr := conversationsearch.BuildSemanticRuntime(semanticCtx, conversationsearch.SemanticRuntimeOptions{
		SearchFilePath: conversationSearchFile,
		Source:         conversationSource, Projection: conversationRepository,
		GenerationCatalog: conversationsearch.NewSQLiteQdrantCatalog(db),
	})
	semanticCancel()
	if semanticConfigErr != nil {
		logger.Printf("conversation search semantic configuration invalid; lexical API remains available: %v", semanticConfigErr)
		semanticRuntime = &conversationsearch.SemanticRuntime{}
	} else if semanticRuntime.InitializationError != nil {
		logger.Printf("conversation search semantic resources degraded; lexical API remains available: %v", semanticRuntime.InitializationError)
	}
	var searchOptions []conversationsearch.ServiceOption
	searchOptions = append(searchOptions, conversationsearch.WithNonBlockingCoverage())
	if semanticRuntime.Retriever != nil {
		searchOptions = append(searchOptions, conversationsearch.WithSemanticRetriever(semanticRuntime.Retriever))
	}
	conversationSearch, err := conversationsearch.NewService(conversationRepository, conversationRepository, conversationRepository, cursorKey, searchOptions...)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize conversation search: %w", err)
	}
	conversationIndexer, err := conversationsearch.NewIndexer(conversationsearch.IndexerOptions{Source: conversationSource, Repository: conversationRepository, Semantic: semanticRuntime})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize conversation indexer: %w", err)
	}
	wsHub := handlers.NewWebSocketHub()
	conversationTokens := newSearchControlTokens()
	conversationTokens.set(conversationsearch.ConversationSearchProviderID, strings.TrimSpace(os.Getenv("AGENT_MANAGER_SEARCH_CONTROL_TOKEN")))
	go wsHub.Run()
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "/tmp/agent-manager-uploads"
	}
	uploadStorage := storage.NewLocalService(uploadDir)
	deps, err := wiring.NewOrchestrator(db, wsHub, logger, uploadStorage, levers, fileRoots)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build orchestrator: %w", err)
	}
	deps.Orchestrator.SetConversationSearchNotifier(func(ctx context.Context, operation, runID, eventID string) error {
		return conversationIndexer.Notify(ctx, conversationsearch.ChangeOperation(operation), runID, eventID)
	})
	deps.Orchestrator.SetConversationSearchReviver(func(ctx context.Context, harness, sessionID string) error {
		_, err := db.ExecContext(ctx, `DELETE FROM conversation_search_external_tombstones WHERE source_harness = ? AND source_session_id = ?`, harness, sessionID)
		return err
	})
	// After the Transcript Indexer evicts recall it reclaims the dead FTS index
	// space through the search domain's own optimize.
	deps.TranscriptImporter.SetAfterEviction(conversationRepository.OptimizeIndex)
	srv := &Server{
		recovery:           maintenance.NewRecovery(),
		capabilityRegistry: capabilities.NewRegistry(), db: db, fileRoots: fileRoots, router: mux.NewRouter().UseEncodedPath(), orchestrator: deps.Orchestrator,
		statsRepo: deps.StatsRepository, pricingService: deps.PricingService, pricingRepository: deps.PricingRepository,
		wsHub: wsHub, reconciler: deps.Reconciler, awaitRegistry: deps.AwaitRegistry, workflowNudger: deps.WorkflowNudger, transcriptImporter: deps.TranscriptImporter, claudeQuotaFeed: deps.ClaudeQuotaFeed, usageAlerts: deps.UsageAlerts, interventions: deps.Interventions,
		modelHealthProbe: deps.ModelHealthProbe, modelPolicyDrift: deps.ModelPolicyDrift, rolePolicyState: deps.RolePolicyState, permissionPolicyState: deps.PermissionPolicyState,
		permissionPolicy: deps.PermissionPolicy, storage: uploadStorage,
		healthStore: deps.HealthStore, eventRepo: deps.EventRepository, supervisionService: deps.SupervisionService, supervisionScheduler: deps.SupervisionScheduler, watchActionAuthorizer: deps.WatchActionAuthorizer, invocationReadModel: deps.InvocationReadModel,
		conversationSearch:       conversationSearch,
		conversationSemantic:     semanticRuntime,
		conversationIndexer:      conversationIndexer,
		conversationSearchFile:   conversationSearchFile,
		conversationSearchPolicy: searchVisibilityPolicy,
		conversationTokens:       conversationTokens,
		workspaceSandbox:         deps.WorkspaceSandbox,
	}
	// One gate and routed store serve every admission path and the owner endpoint.
	// Install before recovery workers or HTTP can admit work.
	gate := maintenance.NewGate(maintenance.NewRepository(db))
	orchestration.WithMaintenanceGate(gate)(deps.Orchestrator)
	ownerHome, err := os.UserHomeDir()
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("resolve lifecycle owner home: %w", err)
	}
	interlock, err := maintenance.NewScenarioInterlock(ownerHome)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	inventory := wiring.NewMaintenanceInventory(db)
	srv.lifecycleService, err = maintenance.NewLifecycleService(gate, func(ctx context.Context) (maintenance.Inventory, error) {
		state, observeErr := inventory.Observe(ctx)
		if recovery := srv.recovery.Snapshot(); !recovery.Readiness {
			state.Remaining = nil
			state.Unknown = append(state.Unknown, "startup recovery "+recovery.Status+": "+recovery.Phase)
			return state, fmt.Errorf("startup recovery is not ready")
		}
		return state, observeErr
	}, "agent-manager", "agent-manager", uuid.NewString())
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	srv.maintenance, err = maintenance.NewHandler(gate, func(ctx context.Context) (maintenance.Inventory, error) {
		state, err := inventory.Observe(ctx)
		if recovery := srv.recovery.Snapshot(); !recovery.Readiness {
			state.Remaining = nil
			state.Unknown = append(state.Unknown, "startup recovery "+recovery.Status+": "+recovery.Phase)
			return state, fmt.Errorf("startup recovery is not ready")
		}
		return state, err
	}, deps.OwnerIdentity, interlock)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := srv.buildStorageHealth(routedDB, repoRoot); err != nil {
		_ = db.Close()
		return nil, err
	}
	if srv.evidenceRetention, srv.evidenceApprovals, err = srv.buildEvidenceRetention(repoRoot, deps.DecisionPublisher); err != nil {
		_ = db.Close()
		return nil, err
	}
	srv.setupRoutes()
	srv.startRecovery()
	return srv, nil
}

func (s *Server) startSearchRegistration(parent context.Context) {
	if s == nil || s.conversationSearchFile == "" || s.conversationTokens == nil {
		return
	}
	ctx, cancel := context.WithCancel(parent)
	s.searchRegistrationStop = cancel
	go searchregister.Register(ctx, searchregister.Config{
		ScenarioID:     "agent-manager",
		SearchFilePath: s.conversationSearchFile,
		Logger:         log.Default(),
		OnControlToken: s.conversationTokens.set,
		ControlToken:   s.conversationTokens.get,
	})
}

func envOrEmpty(key string) string { return os.Getenv(key) }

func (s *Server) startRecovery() {
	if s.recovery == nil {
		s.recovery = maintenance.NewRecovery()
	}
	// Model-policy drift is a read-only, bounded health observation. Start it
	// independently of historical recovery so a large or degraded recovery
	// queue cannot leave the live model-policy snapshot stale (and report
	// models as missing) for the entire recovery window.
	if s.modelPolicyDrift != nil {
		s.modelPolicyDrift.Start(context.Background())
	}
	s.recovery.Start(context.Background(), s.recoverySteps())
}

func (s *Server) recoverySteps() []maintenance.RecoveryStep {
	var steps []maintenance.RecoveryStep
	add := func(name string, run func(context.Context) error) {
		// Historical projections and declarations of other scenarios have their
		// own retry owners. Their findings must not globally disable legitimate
		// work; active ownership recovery is required and remains readiness-gated.
		nonCritical := name == "workflow_accounting" || name == "stats_rebuild" || name == "scenario_declarations"
		// Required steps retry: on a busy host the first attempt can lose a
		// SQLite contention race with start-up indexing and would otherwise
		// keep readiness closed until the next restart.
		attempts := 3
		if nonCritical {
			attempts = 1
		}
		steps = append(steps, maintenance.RecoveryStep{Name: name, NonCritical: nonCritical, Timeout: 30 * time.Second, Attempts: attempts, RetryDelay: 15 * time.Second, Run: func(ctx context.Context) error {
			err := run(ctx)
			if err != nil {
				obs.Logger().Warn("startup recovery failed", "phase", name, obs.KeyError, err.Error())
			}
			return err
		}})
	}
	if s.supervisionService != nil {
		add("cohort_actions", func(ctx context.Context) error { _, err := s.supervisionService.RecoverActions(ctx); return err })
	}
	if s.reconciler != nil {
		add("in_flight_runs", s.reconciler.RecoverInFlightRuns)
	}
	repoRoot := os.Getenv("PROJECT_ROOT")
	if repoRoot == "" {
		repoRoot, _ = filepath.Abs(filepath.Join("..", "..", ".."))
	}
	if s.orchestrator != nil {
		// Register this service's own declarations before scanning retained
		// historical projections. Historical work is non-critical and must not
		// consume the recovery budget needed for ownership continuity.
		add("self_declarations", func(ctx context.Context) error {
			result, err := s.orchestrator.ReconcileSelfDeclarations(ctx, repoRoot)
			if err != nil {
				return err
			}
			obs.Logger().Info("agent-manager self-declaration registration complete", "profiles_created", result.ProfilesCreated, "profiles_updated", result.ProfilesUpdated, "workflows_created", result.WorkflowsCreated, "workflows_activated", result.WorkflowsActivated, "failed", result.ProfilesFailed+result.WorkflowsFailed)
			if result.ProfilesFailed+result.WorkflowsFailed > 0 {
				return fmt.Errorf("self-declaration registration incomplete")
			}
			return nil
		})
	}
	if s.orchestrator != nil {
		add("workflow_accounting", s.orchestrator.RecoverWorkflowExecutions)
	}
	if s.orchestrator != nil {
		add("scenario_declarations", func(ctx context.Context) error {
			summary := s.orchestrator.ReconcileDeclaringScenarios(ctx, repoRoot)
			obs.Logger().Info("scenario declaration sweep complete", "scanned", summary.Scanned, "declaring", summary.Declaring, "reconciled", summary.Reconciled, "failed", summary.Failed)
			if summary.Failed > 0 {
				return fmt.Errorf("%d scenario declarations failed", summary.Failed)
			}
			return nil
		})
	}
	if s.orchestrator != nil {
		add("parked_run_compaction", s.orchestrator.RecoverParkCompactions)
	}
	if s.awaitRegistry != nil {
		add("parked_runs", func(ctx context.Context) error {
			n, err := s.awaitRegistry.RecoverParkedRuns(ctx)
			if err == nil && n > 0 {
				obs.Logger().Info("re-spawned waiters for parked runs", "count", n)
			}
			return err
		})
	}
	// Start schedulers after the initial recovery attempts so existing owners can
	// retry incomplete work. Failures remain visible in health. Their context
	// lives until Cleanup; it must not inherit a historical scan's deadline.
	steps = append(steps, maintenance.RecoveryStep{Name: "background_workers", Run: func(ctx context.Context) error {
		s.storageHealth.setWorkersContext(ctx)
		startEvidenceRetention(ctx, s.evidenceRetention)
		if s.supervisionScheduler != nil {
			s.supervisionScheduler.Start(ctx)
		}
		if s.conversationIndexer != nil {
			s.conversationIndexer.Start(ctx)
		}
		if s.reconciler != nil {
			if err := s.reconciler.Start(ctx); err != nil {
				return err
			}
		}
		if s.workflowNudger != nil {
			s.workflowNudger.Start()
		}
		if s.transcriptImporter != nil {
			s.transcriptImporter.Start(ctx)
		}
		s.claudeQuotaFeed.Start(ctx)
		s.usageAlerts.Start(ctx)
		s.interventions.Start(ctx)
		startQuotaCompaction(ctx, s.pricingRepository)
		// Scans every run's events, so it runs here, never in schema init. An
		// interrupted pass leaves no marker and repeats on the next start.
		go func() {
			if err := s.db.RepairDuplicateTranscriptEvents(ctx); err != nil && ctx.Err() == nil {
				obs.Logger().Error("duplicate transcript event repair failed; it retries on the next start", "error", err.Error())
			}
			// Deleted runs' leftover read-model rows; same contract, run after.
			if err := s.db.RepairOrphanRunProjections(ctx); err != nil && ctx.Err() == nil {
				obs.Logger().Error("orphan run projection repair failed; it retries on the next start", "error", err.Error())
			}
		}()
		if s.orchestrator != nil {
			wiring.ScheduleDeclarationReconcile(s.orchestrator, repoRoot, s.recovery.ReconcileDeclarationStep)
		}
		if s.modelHealthProbe != nil {
			s.modelHealthProbe.Start(ctx)
		}
		return nil
	}})
	return steps
}

func (s *Server) setupRoutes() {
	wiring.SetupRoutes(s.router, wiring.RouteDependencies{
		Recovery:           s.recovery,
		CapabilityRegistry: s.capabilityRegistry, DB: s.db, Orchestrator: s.orchestrator, StatsRepository: s.statsRepo,
		PricingService: s.pricingService, PricingRepository: s.pricingRepository, WebSocketHub: s.wsHub, RolePolicyState: s.rolePolicyState,
		PermissionPolicyState: s.permissionPolicyState, PermissionPolicy: s.permissionPolicy, Storage: s.storage,
		HealthStore: s.healthStore, EventRepository: s.eventRepo, SupervisionService: s.supervisionService, WatchActionAuthorizer: s.watchActionAuthorizer, InvocationReadModel: s.invocationReadModel,
		ModelPolicyDrift: s.modelPolicyDrift, TranscriptImporter: s.transcriptImporter,
		WorkspaceSandbox:         s.workspaceSandbox,
		ConversationSearch:       s.conversationSearch,
		ConversationIndexer:      s.conversationIndexer,
		ConversationSearchPolicy: s.conversationSearchPolicy,
		ConversationSemantic:     s.conversationSemantic,
		ConversationSearchFile:   s.conversationSearchFile,
		ConversationControlToken: func() string { return s.conversationTokens.get(conversationsearch.ConversationSearchProviderID) },
		StorageHealth:            s.storageHealthHandler(),
		LifecycleService:         s.lifecycleService,
		DecisionEffects:          s.decisionEffects(),
	})
	s.registerEvidenceRetentionRoutes(s.router, s.evidenceRetention, s.evidenceApprovals)
}

func (s *Server) Router() http.Handler {
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, s.db.Routed, s.fileRoots)
	if s.maintenance != nil {
		rootMux.Handle(maintenance.AdmissionPath, s.maintenance)
		rootMux.Handle(maintenance.AdmissionPath+"/", s.maintenance)
	}
	if s.recovery != nil {
		rootMux.Handle("/health", s.recovery.Health(s.router))
		rootMux.Handle("/api/v1/health", s.recovery.Readiness(s.router))
		rootMux.Handle(apiconnect.AgentManagerServiceHealthProcedure, s.recovery.ConnectHealth(s.router))
	}
	rootMux.Handle("/", gorillaHandlers.RecoveryHandler()(s.router))
	return apihttp.TestModeMiddleware(rootMux)
}

func (s *Server) Cleanup() error {
	if s.recovery != nil {
		s.recovery.Stop()
	}
	if s.searchRegistrationStop != nil {
		s.searchRegistrationStop()
	}
	if s.conversationIndexer != nil {
		s.conversationIndexer.Stop()
	}
	wiring.Shutdown(s.db, s.reconciler, s.awaitRegistry, s.workflowNudger, s.transcriptImporter, s.modelPolicyDrift)
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
	srv.startSearchRegistration(context.Background())
	stopAuthorizationBroker, err := srv.orchestrator.StartAuthorizationBroker()
	if err != nil {
		log.Fatalf("authorization broker startup failed: %v", err)
	}
	defer stopAuthorizationBroker()
	if err := server.Run(server.Config{Handler: srv.Router(), WriteTimeout: 3 * time.Minute, ReadTimeout: time.Minute, Cleanup: func(context.Context) error { return srv.Cleanup() }}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
