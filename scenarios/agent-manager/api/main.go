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

	"agent-manager/internal/adapters/database"
	capabilities "agent-manager/internal/capabilities"
	agentconfig "agent-manager/internal/config"
	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/handlers"
	healthstore "agent-manager/internal/health"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/modelpolicydrift"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/pricing"
	"agent-manager/internal/repository"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/stats"
	"agent-manager/internal/storage"
	"agent-manager/internal/supervision"
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
	capabilityRegistry     *capabilities.Registry
	db                     *database.DB
	fileRoots              *filerouting.RoutedRoots
	router                 *mux.Router
	orchestrator           *orchestration.Orchestrator
	statsService           orchestration.StatsService
	statsRepo              repository.StatsRepository
	pricingService         pricing.Service
	pricingRepository      pricing.Repository
	wsHub                  *handlers.WebSocketHub
	reconciler             *orchestration.Reconciler
	awaitRegistry          *orchestration.AwaitRegistry
	workflowNudger         *orchestration.WorkflowNudger
	transcriptImporter     *orchestration.TranscriptImportScheduler
	frictionPublisher      *orchestration.FrictionPublishScheduler
	modelHealthProbe       *healthstore.Probe
	modelPolicyDrift       *modelpolicydrift.Scheduler
	rolePolicyState        *rolepolicy.State
	permissionPolicyState  *permissionpolicy.State
	permissionPolicy       *permissionpolicy.Service
	storage                storage.Service
	statsEngine            *stats.Engine
	healthStore            *healthstore.Store
	eventRepo              eventlog.Repository
	supervisionService     *supervision.Service
	supervisionScheduler   *supervision.Scheduler
	watchActionAuthorizer  handlers.WatchActionAuthorizer
	invocationReadModel    invocationreadmodel.Store
	conversationSearch     *conversationsearch.Service
	conversationSemantic   *conversationsearch.SemanticRuntime
	conversationIndexer    *conversationsearch.Indexer
	conversationSearchFile string
	conversationTokens     *searchControlTokens
	searchRegistrationStop context.CancelFunc
	workspaceSandbox       interface {
		IsAvailable(context.Context) (bool, string)
	}
}

// databaseConfigFromLevers keeps the production pool aligned with Agent
// Manager's governed storage settings. SQLite runs in WAL mode, so retaining
// more than one connection lets conversation reads proceed while incremental
// projection publication holds the writer connection.
func databaseConfigFromLevers(dsn string, storage agentconfig.StorageLevers) coredb.Config {
	return coredb.Config{
		Driver:          coredb.DriverSQLite,
		DSN:             dsn,
		MaxOpenConns:    storage.MaxOpenConns,
		MaxIdleConns:    storage.MaxIdleConns,
		ConnMaxLifetime: storage.ConnMaxLifetime,
	}
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
	conversationSource, err := conversationsearch.NewSQLiteSource(db, conversationsearch.MustNormalizer(conversationsearch.NormalizerConfig{}))
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
	})
	semanticCancel()
	if semanticConfigErr != nil {
		logger.Printf("conversation search semantic configuration invalid; lexical API remains available: %v", semanticConfigErr)
	} else if semanticRuntime.InitializationError != nil {
		logger.Printf("conversation search semantic resources degraded; lexical API remains available: %v", semanticRuntime.InitializationError)
	}
	var searchOptions []conversationsearch.ServiceOption
	if semanticRuntime.Retriever != nil {
		searchOptions = append(searchOptions, conversationsearch.WithSemanticRetriever(semanticRuntime.Retriever))
	}
	conversationSearch, err := conversationsearch.NewService(conversationRepository, conversationRepository, conversationRepository, cursorKey, searchOptions...)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize conversation search: %w", err)
	}
	conversationIndexer, err := conversationsearch.NewIndexer(conversationsearch.IndexerOptions{Source: conversationSource, Repository: conversationRepository, Semantic: &semanticRuntime})
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
	srv := &Server{
		capabilityRegistry: capabilities.NewRegistry(), db: db, fileRoots: fileRoots, router: mux.NewRouter().UseEncodedPath(), orchestrator: deps.Orchestrator,
		statsService: deps.StatsService, statsRepo: deps.StatsRepository, pricingService: deps.PricingService, pricingRepository: deps.PricingRepository,
		wsHub: wsHub, reconciler: deps.Reconciler, awaitRegistry: deps.AwaitRegistry, workflowNudger: deps.WorkflowNudger, transcriptImporter: deps.TranscriptImporter, frictionPublisher: deps.FrictionPublisher,
		modelHealthProbe: deps.ModelHealthProbe, modelPolicyDrift: deps.ModelPolicyDrift, rolePolicyState: deps.RolePolicyState, permissionPolicyState: deps.PermissionPolicyState,
		permissionPolicy: deps.PermissionPolicy, storage: uploadStorage, statsEngine: deps.StatsEngine,
		healthStore: deps.HealthStore, eventRepo: deps.EventRepository, supervisionService: deps.SupervisionService, supervisionScheduler: deps.SupervisionScheduler, watchActionAuthorizer: deps.WatchActionAuthorizer, invocationReadModel: deps.InvocationReadModel,
		conversationSearch:     conversationSearch,
		conversationSemantic:   &semanticRuntime,
		conversationIndexer:    conversationIndexer,
		conversationSearchFile: conversationSearchFile,
		conversationTokens:     conversationTokens,
		workspaceSandbox:       deps.WorkspaceSandbox,
	}
	srv.startRecovery()
	srv.setupRoutes()
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
	ctx := context.Background()
	if s.supervisionScheduler != nil {
		if _, err := s.supervisionService.RecoverActions(ctx); err != nil {
			obs.Logger().Warn("cohort action recovery failed", obs.KeyError, err.Error())
		}
		s.supervisionScheduler.Start(ctx)
	}
	if s.conversationIndexer != nil {
		s.conversationIndexer.Start(ctx)
	}
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
	if s.transcriptImporter != nil {
		s.transcriptImporter.Start(ctx)
	}
	if s.frictionPublisher != nil {
		s.frictionPublisher.Start(ctx)
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
	if s.modelPolicyDrift != nil {
		s.modelPolicyDrift.Start(ctx)
	}
	if s.statsEngine != nil {
		if err := s.statsEngine.Rebuild(ctx); err != nil {
			obs.Logger().Warn("stats engine rebuild failed", obs.KeyError, err.Error())
		}
	}
}

func (s *Server) setupRoutes() {
	wiring.SetupRoutes(s.router, wiring.RouteDependencies{
		CapabilityRegistry: s.capabilityRegistry, DB: s.db, Orchestrator: s.orchestrator, StatsService: s.statsService, StatsRepository: s.statsRepo,
		PricingService: s.pricingService, PricingRepository: s.pricingRepository, WebSocketHub: s.wsHub, RolePolicyState: s.rolePolicyState,
		PermissionPolicyState: s.permissionPolicyState, PermissionPolicy: s.permissionPolicy, Storage: s.storage,
		StatsEngine: s.statsEngine, HealthStore: s.healthStore, EventRepository: s.eventRepo, SupervisionService: s.supervisionService, WatchActionAuthorizer: s.watchActionAuthorizer, InvocationReadModel: s.invocationReadModel,
		ModelPolicyDrift: s.modelPolicyDrift, TranscriptImporter: s.transcriptImporter,
		WorkspaceSandbox:         s.workspaceSandbox,
		ConversationSearch:       s.conversationSearch,
		ConversationIndexer:      s.conversationIndexer,
		ConversationSearchFile:   s.conversationSearchFile,
		ConversationControlToken: func() string { return s.conversationTokens.get(conversationsearch.ConversationSearchProviderID) },
	})
}

func (s *Server) Router() http.Handler {
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, s.db.Routed, s.fileRoots)
	rootMux.Handle("/", gorillaHandlers.RecoveryHandler()(s.router))
	return apihttp.TestModeMiddleware(rootMux)
}

func (s *Server) Cleanup() error {
	if s.searchRegistrationStop != nil {
		s.searchRegistrationStop()
	}
	if s.conversationIndexer != nil {
		s.conversationIndexer.Stop()
	}
	wiring.Shutdown(s.db, s.reconciler, s.awaitRegistry, s.workflowNudger, s.transcriptImporter, s.frictionPublisher, s.modelPolicyDrift)
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
	if err := server.Run(server.Config{Handler: srv.Router(), WriteTimeout: 3 * time.Minute, ReadTimeout: time.Minute, Cleanup: func(context.Context) error { return srv.Cleanup() }}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
