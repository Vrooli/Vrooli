// Package main provides the Swarm Manager API server.
//
// DOC: docs/concepts/ARCHITECTURE.md#logical-architecture
// DOC: docs/internal/INTENT.md#api-components
//
// The API serves as the backend for the Swarm Manager UI, providing endpoints
// for backlog management, scenario lifecycle, execution control, settings,
// agent coordination, and real-time graph streaming.
//
// Related PRD targets: OT-P0-002, OT-P0-005, OT-P0-006
package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/handlers/audio_admin"
	"swarm-manager/handlers/audio_runtime"
	"swarm-manager/handlers/discovery"
	"swarm-manager/integrations/audiotools"
	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/aisearch"
	"swarm-manager/internal/audioports"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/captures"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/initiativereview"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/overview"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/prompts"
	"swarm-manager/internal/queue"
	"swarm-manager/internal/records"
	"swarm-manager/internal/review"
	"swarm-manager/internal/runtimepaths"
	"swarm-manager/internal/scenarios"
	"swarm-manager/internal/sessioncontext"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/stats"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"
)

type Server struct {
	router              *mux.Router
	agentSvc            *agentmanager.AgentService
	agentActivitySvc    *agentactivity.Service
	agentSessionSvc     *agentsessions.Service
	agentSessionStore   agentsessions.Store
	settingsStore       *settings.Store
	backlogHandler      *backlog.Handler
	capturesHandler     *captures.Handler
	recordsService      *records.Service
	recordsHandler      *records.Handler
	recordsStore        records.Store
	scenariosHandler    *scenarios.Handler
	initStore           *initiatives.Store
	initiativeService   *initiatives.Service
	overviewSvc         *overview.Service
	executionSvc        *execution.Service
	executionHandler    *execution.Handler
	operatingModeSvc    *operatingmode.Service
	reviewSvc           *review.Service
	reviewHandler       *review.Handler
	initiativeReviewSvc *initiativereview.Service
	executionStopChan   chan struct{}
	reviewStopChan      chan struct{}
	initReviewStopChan  chan struct{}
	graphBroker         *graph.Broker
	graphDispatch       *graph.Dispatch
	graphProjection     *graph.ProjectionService
	queueHandler        *queue.Handler
	scenarioRoot        string
	dataRoot            string
	cacheRoot           string
	promptClient        promptmanager.Client
	eventDB             *sql.DB
	emitter             *eventlog.Emitter
	statsEngine         *stats.Engine
	aiSearchSvc         *aisearch.Service
	aiSearchReconciler  *aisearch.Reconciler
	aiSearchSyncLoop    *aisearch.SyncLoop
	aiSearchStopChan    chan struct{}
	feedbackSweeperStop chan struct{}
	reviewSweeperStop   chan struct{}
	audioToolsResolver  audiotools.URLResolver

	// Audio ports — all backed by audio-tools. Mirrors web-console's
	// audio integration: the UI talks same-origin to swarm-manager's own
	// AudioAdminService + AudioRuntimeService, and these ports proxy to
	// audio-tools server-side.
	sttPort              audioports.SpeechToText
	ttsPort              audioports.TextToSpeech
	speechProcessor      audioports.SpeechTextProcessor
	summarizer           audioports.Summarizer
	streamConfigAdmin    audioports.StreamConfigAdmin
	wakeWordAdmin        audioports.WakeWordAdmin
	speakerAdmin         audioports.SpeakerAdmin
	ttsConfigAdmin       audioports.TTSConfigAdmin
	summarizeConfigAdmin audioports.SummarizeConfigAdmin
	playbackRecorder     audioports.PlaybackEventRecorder
}

type executionSnapshotLister struct {
	svc *execution.Service
}

func (l executionSnapshotLister) List(ctx context.Context, filters execution.ListFilters) ([]execution.Record, error) {
	return l.svc.ListSnapshot(ctx, filters)
}

// NewServer initializes routes using the default scenario root resolved from
// the environment.
func NewServer() *Server {
	return NewServerWithRoot(pathutil.ResolveScenarioRoot("swarm-manager"))
}

// NewServerWithRoot initializes routes using the given scenario root directory.
// Tests should use this with t.TempDir() to avoid touching production data.
func NewServerWithRoot(scenarioRoot string) *Server {
	return newServerWithRoot(scenarioRoot, nil)
}

func newServerWithRoot(scenarioRoot string, promptClient promptmanager.Client) *Server {
	dataRoot, err := runtimepaths.DataPath("")
	if err != nil {
		log.Fatalf("resolve runtime data root: %v", err)
	}
	cacheRoot, err := runtimepaths.CachePath("")
	if err != nil {
		log.Fatalf("resolve runtime cache root: %v", err)
	}
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		log.Fatalf("create runtime data root %q: %v", dataRoot, err)
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		log.Fatalf("create runtime cache root %q: %v", cacheRoot, err)
	}

	agentEnabled := strings.ToLower(strings.TrimSpace(os.Getenv("AGENT_MANAGER_ENABLED"))) != "false"
	if err := operatingmode.ValidateRegistry(); err != nil {
		log.Fatalf("invalid operating-mode registry: %v", err)
	}
	requiredProfileKeys, err := operatingmode.RequiredProfileKeys()
	if err != nil {
		log.Fatalf("invalid operating-mode profile policy: %v", err)
	}
	agentSvc := agentmanager.NewAgentService(agentmanager.AgentServiceConfig{
		ProfileName:  getEnvDefault("AGENT_MANAGER_PROFILE_NAME", "swarm-manager"),
		ProfileKey:   getEnvDefault("AGENT_MANAGER_PROFILE_KEY", "swarm-manager/default"),
		RequiredKeys: requiredProfileKeys,
		Timeout:      30 * time.Second,
		Enabled:      agentEnabled,
	})

	srv := &Server{
		router:              mux.NewRouter(),
		agentSvc:            agentSvc,
		executionStopChan:   make(chan struct{}),
		reviewStopChan:      make(chan struct{}),
		initReviewStopChan:  make(chan struct{}),
		aiSearchStopChan:    make(chan struct{}),
		feedbackSweeperStop: make(chan struct{}),
		reviewSweeperStop:   make(chan struct{}),
		scenarioRoot:        scenarioRoot,
		dataRoot:            dataRoot,
		cacheRoot:           cacheRoot,
		promptClient:        promptClient,
		audioToolsResolver:  resolveAudioToolsResolver(),
	}

	// Wire audioports against an audio-tools client. Mirrors web-console:
	// UI calls our own AudioAdminService / AudioRuntimeService same-origin;
	// these ports proxy to audio-tools server-to-server. We keep going if
	// audio-tools is not reachable at startup — Required=false lets the
	// ports surface a typed error per call instead of fail-fast at boot.
	if srv.audioToolsResolver != nil {
		atClient, err := audiotools.New(srv.audioToolsResolver, audiotools.Policy{
			Required:       false,
			PerCallTimeout: 150 * time.Second,
		})
		if err != nil {
			log.Printf("audio-tools client: not reachable at startup: %v (ports will retry per call)", err)
		}
		if atClient != nil {
			srv.sttPort = &audioports.RemoteSpeechToText{Client: atClient}
			srv.ttsPort = &audioports.RemoteTextToSpeech{Client: atClient}
			srv.speechProcessor = &audioports.RemoteSpeechTextProcessor{Client: atClient}
			srv.summarizer = &audioports.RemoteSummarizer{Client: atClient}
			srv.streamConfigAdmin = &audioports.RemoteStreamConfigAdmin{Client: atClient}
			srv.wakeWordAdmin = &audioports.RemoteWakeWordAdmin{Client: atClient}
			srv.speakerAdmin = &audioports.RemoteSpeakerAdmin{Client: atClient}
			srv.ttsConfigAdmin = &audioports.RemoteTTSConfigAdmin{Client: atClient}
			srv.summarizeConfigAdmin = &audioports.RemoteSummarizeConfigAdmin{Client: atClient}
			srv.playbackRecorder = &audioports.RemotePlaybackEventRecorder{Client: atClient}
		}
	}

	// initEventLog must run before setupRoutes so that route registration
	// captures a non-nil s.emitter. Constructors like registerFeedbackRoutes
	// build internal services (backlog.Service for proposal apply) that take
	// the emitter at construction time and have no SetEventLogger backstop;
	// if s.emitter were still nil here, those services would hold a typed-nil
	// *eventlog.Emitter behind a non-nil CreationEventEmitter interface and
	// panic on first emit.
	srv.initEventLog()
	srv.setupRoutes()
	srv.wireEventLoggers()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	s.router.Use(identity.Middleware(identity.CLIUtilVerifier{}))
	scenarioRoot := s.scenarioRoot
	scenariosDir := filepath.Dir(scenarioRoot)

	// --- Infrastructure ---
	s.registerHealthRoutes()
	s.registerDiscoveryRoutes()
	s.registerAudioRoutes()
	s.registerSettingsRoutes(scenarioRoot)      // Must be before backlog/execution (they depend on settings store)
	s.registerAgentActivityRoutes(scenarioRoot) // Must be before backlog/execution (they depend on agent activity)
	s.registerScenarioRoutes(scenariosDir)
	s.registerAgentSessionRoutes(scenarioRoot)
	s.router.Use(identity.SessionMiddleware(s.agentSessionSvc))

	// --- Core domain ---
	backlogHandler := s.registerBacklogRoutes(s.dataRoot, scenarioRoot)
	initService := s.registerInitiativeRoutes(s.dataRoot, backlogHandler)
	s.registerCapturesRoutes(s.cacheRoot, backlogHandler)
	s.registerRecordsRoutes(s.dataRoot)

	// --- Execution & review ---
	execSvc := s.registerExecutionRoutes(s.dataRoot, scenarioRoot)
	s.registerReviewRoutes(scenarioRoot, execSvc)
	s.registerQueueRoutes(scenarioRoot)

	// --- Cross-domain wiring ---
	s.scenariosHandler.SetBacklogLister(backlogHandler.Store())
	s.scenariosHandler.SetInitiativesLister(initService)
	if execSvc != nil {
		s.scenariosHandler.SetExecutionLister(executionSnapshotLister{svc: execSvc})
	}

	// --- Read-only surfaces ---
	overviewSvc := s.registerOverviewRoutes(backlogHandler, initService)
	if execSvc != nil {
		overviewSvc.SetGovernanceProvider(execSvc)
	}
	materializer := s.registerGraphRoutes(scenarioRoot)
	s.registerFeedbackRoutes(materializer)
	s.registerInitiativeReviewRoutes(materializer)
	s.registerOperatingModeRoutes(scenarioRoot, materializer)
	s.registerOperationsRoutes()
	s.registerPromptRoutes(scenarioRoot)
	s.registerAgentManagerRoutes()

	// --- AI search (must come last so readers see fully-wired stores) ---
	s.registerAISearchRoutes(backlogHandler, initService)
}

// registerAISearchRoutes constructs the aisearch service from environment
// configuration, wires index-on-write hooks into the backlog and initiative
// stores, and registers HTTP routes under /api/v1/search/ai. If required
// resources (Ollama, Qdrant) are not configured, the service is still created
// so /status can explain why AI search is unavailable; write hooks are still
// attached so index operations queue correctly once resources come online.
func (s *Server) registerAISearchRoutes(backlogHandler *backlog.Handler, initService *initiatives.Service) {
	cfg := aisearch.LoadConfigFromEnv()

	embedder := aisearch.NewEmbedder(cfg.EmbeddingModel)
	backlogVS := aisearch.NewVectorStore(cfg.QdrantURL, cfg.QdrantAPIKey, cfg.BacklogCollection, cfg.VectorDimensions)
	initVS := aisearch.NewVectorStore(cfg.QdrantURL, cfg.QdrantAPIKey, cfg.InitiativeCollection, cfg.VectorDimensions)
	recordVS := aisearch.NewVectorStore(cfg.QdrantURL, cfg.QdrantAPIKey, cfg.RecordCollection, cfg.VectorDimensions)

	backlogReader := aisearch.NewBacklogStoreAdapter(backlogHandler.Store())
	initReader := aisearch.NewInitiativeStoreAdapter(s.initStore)

	svc := aisearch.NewService(embedder, backlogVS, initVS, backlogReader, initReader, cfg.Threshold)
	svc.SetRecordStore(recordVS)

	// The Reconciler is the single owner of the "make qdrant match disk"
	// decision. The Service handles search + status; the Reconciler handles
	// reconcile + reconcile-status + reconcile-cancel. They share embedder
	// and stores but don't share state.
	reconciler := aisearch.NewReconciler(
		embedder, backlogVS, initVS,
		backlogReader, initReader,
		aisearch.ResolveReconcileParallelism(),
	)

	// Only attach indexer hooks if both subsystems are configured; otherwise
	// write-path goroutines would bang on unreachable URLs on every mutation.
	if cfg.OllamaURL != "" && cfg.QdrantURL != "" {
		backlogHandler.SetAIIndexer(svc)
		initService.SetAIIndexer(svc)
		if s.recordsService != nil {
			s.recordsService.SetIndexer(aisearch.NewRecordIndexerAdapter(svc))
		}
		if s.recordsHandler != nil && s.recordsStore != nil {
			s.recordsHandler.SetSearcher(aisearch.NewRecordSearcherAdapter(svc, s.recordsStore))
		}
	}

	handler := aisearch.NewHandler(svc, reconciler)
	handler.RegisterRoutes(s.router)
	s.aiSearchSvc = svc
	s.aiSearchReconciler = reconciler
}

// registerDiscoveryRoutes mounts the Connect-RPC DiscoveryService. The
// service is currently the only Connect-RPC surface in swarm-manager;
// see docs/internal/PROBLEMS.md for the residual REST drift.
func (s *Server) registerDiscoveryRoutes() {
	discovery.RegisterRoutes(s.router, newAudioToolsResolverAdapter(s), nil)
}

// registerAudioRoutes mounts the swarm-manager-owned AudioAdminService +
// AudioRuntimeService Connect handlers (delegating to internal/audioports
// which proxy to audio-tools), plus the raw WebSocket proxy for streaming
// STT at /api/v1/voice/stream.
func (s *Server) registerAudioRoutes() {
	audio_admin.RegisterRoutes(s.router, audio_admin.Deps{
		StreamConfig:    s.streamConfigAdmin,
		WakeWord:        s.wakeWordAdmin,
		Speaker:         s.speakerAdmin,
		TTSConfig:       s.ttsConfigAdmin,
		SummarizeConfig: s.summarizeConfigAdmin,
	})
	audio_runtime.RegisterRoutes(s.router, audio_runtime.Deps{
		STT:      s.sttPort,
		TTS:      s.ttsPort,
		Playback: s.playbackRecorder,
		Summ:     s.summarizer,
	})
	if s.audioToolsResolver != nil {
		s.router.Handle("/api/v1/voice/stream", newVoiceStreamProxy(s.audioToolsResolver)).Methods(http.MethodGet)
	}
}

func (s *Server) registerHealthRoutes() {
	healthHandler := health.New().Version("1.0.0").
		Check(health.CheckerFunc(func(_ context.Context) health.CheckResult {
			return health.CheckResult{Name: "database", Connected: true}
		}), health.Optional).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
}

func (s *Server) registerBacklogRoutes(dataRoot, scenarioRoot string) *backlog.Handler {
	backlogHandler := backlog.NewHandlerWithClients(dataRoot, scenarioRoot, s.requireTrackedAgentService(), nil)
	backlogHandler.SetPolicyProvider(settings.NewPolicyAdapter(s.settingsStore))
	backlogHandler.SetGovernanceProvider(settings.NewGovernanceAdapter(s.settingsStore))
	backlogHandler.SetAgentSessionArtifactRecorder(s.agentSessionSvc)
	s.agentSessionSvc.SetBacklogBatchApplier(backlogHandler)
	backlogHandler.RegisterRoutes(s.router)
	backlogHandler.StartWorkshopTicker()
	s.backlogHandler = backlogHandler
	return backlogHandler
}

func (s *Server) registerInitiativeRoutes(dataRoot string, backlogHandler *backlog.Handler) *initiatives.Service {
	initStore := initiatives.NewStore(dataRoot)
	s.initStore = initStore
	initService := initiatives.NewService(initStore, backlogHandler.Store())
	initHandler := initiatives.NewHandler(initService)
	initHandler.SetAgentSessionArtifactRecorder(s.agentSessionSvc)
	initHandler.RegisterRoutes(s.router)
	s.initiativeService = initService

	// Wire initiative assigner into backlog handler for batch operations.
	backlogHandler.SetInitiativeAssigner(initiatives.NewBacklogAssignerAdapter(initService))
	return initService
}

func (s *Server) registerOverviewRoutes(backlogHandler *backlog.Handler, initService *initiatives.Service) *overview.Service {
	overviewSvc := overview.NewService(backlogHandler.Store(), initService)
	overviewHandler := overview.NewHandler(overviewSvc)
	overviewHandler.RegisterRoutes(s.router)
	s.overviewSvc = overviewSvc
	return overviewSvc
}

func (s *Server) registerCapturesRoutes(cacheRoot string, backlogHandler *backlog.Handler) {
	capturesHandler := captures.NewHandler(cacheRoot, s.requireTrackedAgentService(), nil)
	capturesHandler.SetBacklogCreator(captures.NewBacklogItemCreatorAdapter(backlogHandler.Store()))
	capturesHandler.RegisterRoutes(s.router)
	s.capturesHandler = capturesHandler
}

func (s *Server) registerRecordsRoutes(dataRoot string) {
	store := records.NewFileStore(dataRoot)
	svc := records.NewService(store, nil, nil)
	handler := records.NewHandler(svc, nil)
	handler.RegisterRoutes(s.router)
	s.recordsService = svc
	s.recordsHandler = handler
	s.recordsStore = store
	// Soft-prompt hook: auto-create stub records on backlog terminal transitions.
	// Backlog must already be registered (it is — registerBacklogRoutes runs
	// before registerRecordsRoutes); guard defensively anyway.
	if s.backlogHandler != nil {
		s.backlogHandler.SetRecordStubCreator(newRecordStubAdapter(svc))
	}
}

func (s *Server) registerScenarioRoutes(scenariosDir string) {
	s.scenariosHandler = scenarios.NewHandler(scenariosDir)
	s.scenariosHandler.RegisterRoutes(s.router)
}

func (s *Server) registerSettingsRoutes(scenarioRoot string) {
	settingsPath := filepath.Join(scenarioRoot, "config", "settings.json")
	settingsHandler := settings.NewHandler(settingsPath)
	settingsHandler.RegisterRoutes(s.router)
	s.settingsStore = settingsHandler.GetStore()

	// Wire settings into agent service for runtime profile config resolution.
	if s.agentSvc != nil {
		s.agentSvc.SetSettingsReader(settings.NewAgentAdapter(s.settingsStore))
	}
}

func (s *Server) registerAgentActivityRoutes(_ string) {
	storePath, err := runtimepaths.StatePath("agent-activities.json")
	if err != nil {
		panic(err)
	}
	cfg := agentactivity.ServiceConfig{
		StorePath:    storePath,
		AgentService: s.agentSvc,
	}
	if s.settingsStore != nil {
		cfg.LanePolicy = settings.NewLanePolicyAdapter(s.settingsStore)
	}
	s.agentActivitySvc = agentactivity.NewService(cfg)
	agentActivityHandler := agentactivity.NewHandler(s.agentActivitySvc)
	agentActivityHandler.RegisterRoutes(s.router)
}

func (s *Server) registerAgentManagerRoutes() {
	agentManagerHandler := agentmanager.NewHandler(s.agentSvc)
	agentManagerHandler.RegisterRoutes(s.router)
}

func (s *Server) registerAgentSessionRoutes(scenarioRoot string) {
	sessionStore := agentsessions.NewFileStore(scenarioRoot)
	svc, err := agentsessions.NewService(agentsessions.ServiceConfig{
		Store:       sessionStore,
		Spawner:     s.requireTrackedAgentService(),
		EventLogger: s.emitter,
		ProjectRoot: filepath.Dir(filepath.Dir(scenarioRoot)),
		ProfileKey:  getEnvDefault("AGENT_MANAGER_PROFILE_KEY", "swarm-manager/default"),
	})
	if err != nil {
		panic(err)
	}
	svc.SetContextResolver(sessioncontext.NewResolver(scenarioRoot, filepath.Dir(scenarioRoot), sessionStore))
	s.agentSessionSvc = svc
	s.agentSessionStore = sessionStore
	agentsessions.NewHandler(svc).RegisterRoutes(s.router)
}

func (s *Server) registerQueueRoutes(_ string) {
	storePath, err := runtimepaths.StatePath("queue.json")
	if err != nil {
		panic(err)
	}
	s.queueHandler = queue.NewHandler(storePath)
	s.queueHandler.RegisterRoutes(s.router)
}

func (s *Server) requireTrackedAgentService() *agentactivity.Service {
	if s.agentActivitySvc == nil {
		panic("agent activity service must be initialized before agent-dependent routes")
	}
	return s.agentActivitySvc
}

func (s *Server) registerPromptRoutes(scenarioRoot string) {
	promptHandler := prompts.NewHandler(scenarioRoot, nil)
	promptHandler.RegisterRoutes(s.router)
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
		slog.Info("request", "method", r.Method, "uri", r.RequestURI, "duration", time.Since(start))
	})
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "swarm-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	slog.Info("running in filesystem-only mode")

	srv := NewServer()
	srv.runMigrationsOnce()

	// Register stats endpoint (requires event log).
	if srv.statsEngine != nil {
		statsHandler := stats.NewHandler(srv.statsEngine)
		statsHandler.RegisterRoutes(srv.router)
	}

	if srv.executionHandler != nil {
		go srv.executionHandler.StartBackgroundWorker(srv.executionStopChan)
	}

	if srv.reviewSvc != nil {
		srv.reviewSvc.RecoverActiveRounds()
		go srv.reviewSvc.StartBackgroundWorker(srv.reviewStopChan)
	}

	if srv.initiativeReviewSvc != nil {
		srv.initiativeReviewSvc.RecoverActiveRounds()
		go srv.initiativeReviewSvc.StartBackgroundWorker(srv.initReviewStopChan)
	}

	if srv.agentSvc != nil && srv.agentSvc.IsEnabled() {
		initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := srv.agentSvc.Initialize(initCtx, nil); err != nil {
			cancel()
			log.Fatalf("failed to initialize agent-manager profiles: %v", err)
		}
		cancel()
	}

	srv.startAISearchBackground()

	if err := server.Run(server.Config{
		Handler:      srv.Handler(),
		WriteTimeout: 180 * time.Second,
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
	close(srv.executionStopChan)
	close(srv.reviewStopChan)
	close(srv.initReviewStopChan)
	close(srv.aiSearchStopChan)
	close(srv.feedbackSweeperStop)
	close(srv.reviewSweeperStop)
}

// startAISearchBackground kicks off two background tasks for aisearch:
// a one-shot boot-time reconcile (drains any post-deploy drift via the
// reconciler's content-hash compare) and a periodic SyncLoop. Both are no-ops
// when the reconciler is nil (no Ollama/Qdrant configured) or when the
// AI_SEARCH_SYNC_DISABLED kill-switch is on.
//
// Boot-time reconcile uses a 5-minute timeout — the legacy-hash drain on
// first deploy can re-embed every item once, and we'd rather log a timeout
// than block API readiness on Ollama latency. Periodic ticks pick up where
// boot left off.
func (s *Server) startAISearchBackground() {
	if s.aiSearchReconciler == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		plan, result, err := s.aiSearchReconciler.RunOnce(ctx)
		switch {
		case err == nil:
			// Surface partial-Apply failures (per-item embed/upsert/delete
			// errors) that RunOnce does not bubble up. Silent on the fully
			// clean happy path; SyncLoop logs subsequent ticks.
			if result != nil && len(result.Errors) > 0 {
				upserts, deletes := 0, 0
				if plan != nil {
					upserts = plan.UpsertCount()
					deletes = plan.DeleteCount()
				}
				slog.Warn("aisearch boot-time reconcile completed with errors",
					"errorCount", len(result.Errors),
					"plannedUpserts", upserts,
					"plannedDeletes", deletes,
					"firstErr", result.Errors[0].Err,
				)
			}
		case errors.Is(err, aisearch.ErrReconcileBusy):
			// Another caller (rare at boot) acquired the singleton; no-op.
		default:
			slog.Warn("aisearch boot-time reconcile failed", "err", err)
		}
	}()

	if s.aiSearchSyncLoop == nil {
		s.aiSearchSyncLoop = aisearch.NewSyncLoop(s.aiSearchReconciler)
	}
	syncCtx, cancel := context.WithCancel(context.Background())
	go func() {
		<-s.aiSearchStopChan
		cancel()
	}()
	go s.aiSearchSyncLoop.Start(syncCtx)
}

func getEnvDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
