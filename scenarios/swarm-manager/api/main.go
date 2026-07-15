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
	"encoding/json"
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
	measureshandler "swarm-manager/handlers/measures"
	"swarm-manager/integrations/audiotools"
	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/aisearch"
	"swarm-manager/internal/audioports"
	"swarm-manager/internal/autodrain"
	"swarm-manager/internal/autofiler"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/captures"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/evidence"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/goals"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/initiativereview"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/operations"
	"swarm-manager/internal/opsbridge"
	"swarm-manager/internal/overview"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/planclient"
	"swarm-manager/internal/planimport"
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
	aisearchpkg "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	searchregister "github.com/vrooli/searchregister-go"
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
	goalService         *goals.Service
	overviewSvc         *overview.Service
	executionSvc        *execution.Service
	executionHandler    *execution.Handler
	operatingModeSvc    *operatingmode.Service
	reviewSvc           *review.Service
	reviewHandler       *review.Handler
	initiativeReviewSvc *initiativereview.Service
	backlogOpsRunner    *opsbridge.BacklogRunner
	executionStopChan   chan struct{}
	reviewStopChan      chan struct{}
	initReviewStopChan  chan struct{}
	opsRunnerStopChan   chan struct{}
	graphBroker         *graph.Broker
	graphDispatch       *graph.Dispatch
	graphProjection     *graph.ProjectionService
	queueHandler        *queue.Handler
	scenarioRoot        string
	dataRoot            string
	cacheRoot           string
	promptClient        promptmanager.Client
	eventDB             *database.RoutedDB
	emitter             *eventlog.Emitter
	statsEngine         *stats.Engine
	eventRepo           *eventlog.SQLiteRepository
	evidenceStore       *evidence.Store
	evidenceSvc         *evidence.Service
	aiSearchSvc         *aisearch.Service
	aiSearchReconciler  *aisearch.Reconciler
	aiSearchSyncLoop    *aisearch.SyncLoop
	aiSearchStopChan    chan struct{}
	reviewSweeperStop   chan struct{}
	autoFilerSweeper    *autofiler.Sweeper
	autoFilerStopChan   chan struct{}
	audioToolsResolver  audiotools.URLResolver
	opsAggregator       *operations.Aggregator

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
	if err := os.MkdirAll(dataRoot, 0o750); err != nil {
		log.Fatalf("create runtime data root %q: %v", dataRoot, err)
	}
	if err := os.MkdirAll(cacheRoot, 0o750); err != nil {
		log.Fatalf("create runtime cache root %q: %v", cacheRoot, err)
	}

	agentEnabled := strings.ToLower(strings.TrimSpace(os.Getenv("AGENT_MANAGER_ENABLED"))) != "false"
	// Loading the operating-mode registry from the resolved scenario modes dir
	// is the validation step (LoadModesFromDir runs ValidateLoadedModes); a
	// malformed or missing mode folder is a fatal, actionable startup error.
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
		router:             mux.NewRouter(),
		agentSvc:           agentSvc,
		executionStopChan:  make(chan struct{}),
		reviewStopChan:     make(chan struct{}),
		initReviewStopChan: make(chan struct{}),
		opsRunnerStopChan:  make(chan struct{}),
		aiSearchStopChan:   make(chan struct{}),
		reviewSweeperStop:  make(chan struct{}),
		autoFilerStopChan:  make(chan struct{}),
		scenarioRoot:       scenarioRoot,
		dataRoot:           dataRoot,
		cacheRoot:          cacheRoot,
		promptClient:       promptClient,
		audioToolsResolver: resolveAudioToolsResolver(),
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
			srv.streamConfigAdmin = audioports.NewRemoteStreamConfigAdmin(atClient)
			srv.wakeWordAdmin = audioports.NewRemoteWakeWordAdmin(atClient)
			srv.speakerAdmin = audioports.NewRemoteSpeakerAdmin(atClient)
			srv.ttsConfigAdmin = audioports.NewRemoteTTSConfigAdmin(atClient)
			srv.summarizeConfigAdmin = audioports.NewRemoteSummarizeConfigAdmin(atClient)
			srv.playbackRecorder = &audioports.RemotePlaybackEventRecorder{Client: atClient}
		}
	}

	// initEventLog must run before setupRoutes so that route registration
	// captures a non-nil s.emitter. Constructors such as proposal application
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
	s.registerGoalsRoutes(s.dataRoot, backlogHandler)
	s.registerCapturesRoutes(s.cacheRoot, backlogHandler)
	s.registerRecordsRoutes(s.dataRoot, scenariosDir)

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

	// Goal-directed execution (D4): the drain prefers higher-priority goals
	// (FIFO fallback for ungoaled work), and a default-OFF continuous mode
	// auto-enqueues ready goal items through the governed QueueBacklog path.
	if execSvc != nil && s.goalService != nil {
		autoDrain := autodrain.NewStore(s.dataRoot)
		autodrain.NewHandler(autoDrain).RegisterRoutes(s.router)
		execSvc.SetGoalDirectedProviders(s.goalService, goalReadyAdapter{svc: s.goalService}, autoDrain)
	}

	// --- Read-only surfaces ---
	overviewSvc := s.registerOverviewRoutes(backlogHandler, initService)
	if execSvc != nil {
		overviewSvc.SetGovernanceProvider(execSvc)
	}
	materializer := s.registerGraphRoutes(scenarioRoot)
	s.wireSessionMutationProposals(materializer)
	s.registerInitiativeReviewRoutes(materializer)
	s.registerOperatingModeRoutes(scenarioRoot, materializer)
	s.registerAgentOperationsDiagnostics(scenarioRoot)
	s.registerBacklogOperationsRunner(scenarioRoot)
	s.wireEvidence()
	s.registerOperationsRoutes()
	s.registerPlanRoutes(scenarioRoot)
	s.registerPlanImportRoutes(backlogHandler, initService)
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
	policyCtx, cancelPolicy := context.WithTimeout(context.Background(), 5*time.Second)
	embeddingPolicy, err := aisearchpkg.ResolveEmbeddingPolicy(policyCtx, "embedding.default")
	cancelPolicy()
	if err != nil {
		log.Fatalf("resolve swarm-manager embedding policy: %v", err)
	}

	embedder := aisearch.NewEmbedder(embeddingPolicy.Role)
	backlogVS := aisearch.NewVectorStoreForPolicy(cfg.QdrantURL, cfg.QdrantAPIKey, cfg.BacklogCollection, embeddingPolicy)
	initVS := aisearch.NewVectorStoreForPolicy(cfg.QdrantURL, cfg.QdrantAPIKey, cfg.InitiativeCollection, embeddingPolicy)
	recordVS := aisearch.NewVectorStoreForPolicy(cfg.QdrantURL, cfg.QdrantAPIKey, cfg.RecordCollection, embeddingPolicy)

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
	backlogHandler.SetAgentSessionArtifactRecorder(s.agentSessionSvc)
	s.agentSessionSvc.SetBacklogBatchApplier(backlogHandler)
	backlogHandler.RegisterRoutes(s.router)
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

// registerGoalsRoutes wires the goals domain (store, service, HTTP routes).
// Depends on the initiative store (already set by registerInitiativeRoutes) and
// the backlog store for scope computation and seeding.
func (s *Server) registerGoalsRoutes(dataRoot string, backlogHandler *backlog.Handler) *goals.Service {
	goalStore := goals.NewStore(dataRoot)
	goalService := goals.NewService(goalStore, backlogHandler.Store(), s.initStore)
	// Wire the ETA estimator factory. It reads s.eventRepo / s.executionSvc at
	// call time, so it is safe to set here even though executionSvc is created
	// later in setupRoutes — the closure resolves them lazily per request.
	goalService.SetEstimatorFactory(s.newETAEstimator)
	goalHandler := goals.NewHandler(goalService)
	goalHandler.RegisterRoutes(s.router)
	s.goalService = goalService
	return goalService
}

// registerPlanImportRoutes wires the read-only plan-manager import bridge:
// POST /api/v1/plan-import fetches an authored plan over Connect and lands its
// phases as a provenance-stamped linear chain via the atomic batch-create.
func (s *Server) registerPlanImportRoutes(backlogHandler *backlog.Handler, initService *initiatives.Service) {
	if backlogHandler == nil {
		return
	}
	svc := planimport.NewService(
		planclient.NewConnectClient(nil, nil),
		planImportLander{handler: backlogHandler},
		planImportInitiativeLander{svc: initService},
	)
	planimport.NewHandler(svc).RegisterRoutes(s.router)
}

// planImportLander adapts *backlog.Handler to planimport.BatchLander, converting
// the created backlog items to the import-package ref shape so planimport need
// not import backlog.
type planImportLander struct{ handler *backlog.Handler }

func (l planImportLander) ImportBatch(ctx context.Context, payloadJSON string, prov identity.Provenance) ([]planimport.ImportedRef, error) {
	items, err := l.handler.ImportBatchItems(ctx, payloadJSON, prov)
	if err != nil {
		return nil, err
	}
	out := make([]planimport.ImportedRef, 0, len(items))
	for _, it := range items {
		out = append(out, planimport.ImportedRef{Kind: string(it.Kind), Name: it.Name, Title: it.Title})
	}
	return out, nil
}

func (l planImportLander) LandBatch(ctx context.Context, payload planimport.BatchPayload, prov identity.Provenance) ([]planimport.ImportedRef, error) {
	store := l.handler.Store()
	refs := make([]planimport.ImportedRef, 0, len(payload.Items))
	missing := make([]planimport.BatchItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		kind := backlog.BacklogKind(item.Kind)
		existing, err := store.LoadItem(kind, item.Name)
		if err != nil {
			if errors.Is(err, backlog.ErrNotFound) {
				missing = append(missing, item)
				continue
			}
			return nil, err
		}
		action := "linked"
		updated := existing
		updated.Title = item.Title
		updated.Description = item.Description
		updated.DependsOn = append([]string(nil), item.DependsOn...)
		updated.Effort = item.Effort
		updated.AcceptanceAllow = append([]string(nil), item.AcceptanceAllow...)
		updated.AcceptanceDeny = append([]string(nil), item.AcceptanceDeny...)
		if strings.TrimSpace(item.Initiative) != "" {
			if strings.TrimSpace(existing.Initiative) != "" && existing.Initiative != item.Initiative {
				return nil, errors.New("plan-import item already belongs to another initiative")
			}
			updated.Initiative = item.Initiative
		}
		updated.SpawnedFrom = item.SpawnedFrom
		updated.PlanRef = backlogPlanRef(item.PlanRef)
		if !backlogItemsSame(existing, updated) {
			updated.Updated = time.Now().UTC().Format(time.RFC3339)
			if err := store.SaveItem(updated); err != nil {
				return nil, err
			}
			action = "updated"
		}
		refs = append(refs, planimport.ImportedRef{Kind: item.Kind, Name: item.Name, Title: item.Title, Action: action})
	}
	if len(missing) > 0 {
		data, err := json.Marshal(planimport.BatchPayload{Items: missing})
		if err != nil {
			return nil, err
		}
		created, err := l.ImportBatch(ctx, string(data), prov)
		if err != nil {
			return nil, err
		}
		for _, ref := range created {
			ref.Action = "created"
			refs = append(refs, ref)
		}
	}
	return refs, nil
}

func backlogPlanRef(ref *planimport.PlanRef) *backlog.PlanRef {
	if ref == nil {
		return nil
	}
	return &backlog.PlanRef{
		Provider: ref.Provider,
		PlanID:   ref.PlanID,
		Slug:     ref.Slug,
		Role:     ref.Role,
	}
}

func backlogItemsSame(a, b backlog.BacklogItem) bool {
	return a.Title == b.Title &&
		a.Description == b.Description &&
		a.Effort == b.Effort &&
		a.Initiative == b.Initiative &&
		a.SpawnedFrom == b.SpawnedFrom &&
		stringSlicesEqual(a.DependsOn, b.DependsOn) &&
		stringSlicesEqual(a.AcceptanceAllow, b.AcceptanceAllow) &&
		stringSlicesEqual(a.AcceptanceDeny, b.AcceptanceDeny) &&
		planImportRefsEqual(a.PlanRef, b.PlanRef)
}

type planImportInitiativeLander struct{ svc *initiatives.Service }

func (l planImportInitiativeLander) LandInitiative(
	_ context.Context,
	spec planimport.InitiativeSpec,
	itemRefs []planimport.ImportedRef,
	prov identity.Provenance,
) (planimport.ImportedInitiative, error) {
	if l.svc == nil {
		return planimport.ImportedInitiative{}, errors.New("initiative service is not configured")
	}
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return planimport.ImportedInitiative{}, errors.New("initiative name is required")
	}
	title := strings.TrimSpace(spec.Title)
	if title == "" {
		title = name
	}
	mode := initiatives.NormalizeMode(spec.Mode)
	if mode == "" {
		mode = initiatives.NormalizeMode("")
	}
	planRef := initiativePlanRef(spec.PlanRef)
	refs := importedItemRefs(itemRefs)

	action := "linked"
	existing, err := l.svc.Get(name)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return planimport.ImportedInitiative{}, err
		}
		createdBy := prov
		if _, err := l.svc.Create(initiatives.CreateRequest{
			Name:        name,
			Title:       title,
			Description: strings.TrimSpace(spec.Description),
			Items:       refs,
			CreatedBy:   &createdBy,
			PlanRef:     planRef,
		}); err != nil {
			return planimport.ImportedInitiative{}, err
		}
		if mode != initiatives.NormalizeMode("") {
			if _, err := l.svc.SetModeLifecycle(name, mode); err != nil {
				return planimport.ImportedInitiative{}, err
			}
		}
		return planimport.ImportedInitiative{Name: name, Title: title, Mode: mode, Action: "created"}, nil
	}

	init := existing.Initiative
	needsMetaUpdate := init.Title != title ||
		init.Description != strings.TrimSpace(spec.Description) ||
		!planImportInitiativeRefsEqual(init.PlanRef, planRef)
	if needsMetaUpdate {
		description := strings.TrimSpace(spec.Description)
		if _, err := l.svc.Update(name, initiatives.UpdateRequest{
			Title:       &title,
			Description: &description,
			PlanRef:     planRef,
			PlanRefSet:  true,
		}); err != nil {
			return planimport.ImportedInitiative{}, err
		}
		action = "updated"
	}
	if initiatives.NormalizeMode(init.Mode) != mode {
		if _, err := l.svc.SetModeLifecycle(name, mode); err != nil {
			return planimport.ImportedInitiative{}, err
		}
		action = "updated"
	}
	missingRefs := missingInitiativeRefs(init.Items, refs)
	if len(missingRefs) > 0 {
		if err := l.svc.AddItems(name, missingRefs); err != nil {
			return planimport.ImportedInitiative{}, err
		}
		action = "updated"
	}
	return planimport.ImportedInitiative{Name: name, Title: title, Mode: mode, Action: action}, nil
}

func initiativePlanRef(ref planimport.PlanRef) *initiatives.PlanRef {
	if strings.TrimSpace(ref.Provider) == "" &&
		strings.TrimSpace(ref.PlanID) == "" &&
		strings.TrimSpace(ref.Slug) == "" &&
		strings.TrimSpace(ref.Role) == "" {
		return nil
	}
	return &initiatives.PlanRef{
		Provider: strings.TrimSpace(ref.Provider),
		PlanID:   strings.TrimSpace(ref.PlanID),
		Slug:     strings.TrimSpace(ref.Slug),
		Role:     strings.TrimSpace(ref.Role),
	}
}

func importedItemRefs(items []planimport.ImportedRef) []string {
	refs := make([]string, 0, len(items))
	for _, item := range items {
		kind := strings.TrimSpace(item.Kind)
		name := strings.TrimSpace(item.Name)
		if kind == "" || name == "" {
			continue
		}
		refs = append(refs, kind+"/"+name)
	}
	return refs
}

func missingInitiativeRefs(existing, refs []string) []string {
	seen := make(map[string]bool, len(existing))
	for _, ref := range existing {
		seen[ref] = true
	}
	missing := make([]string, 0, len(refs))
	for _, ref := range refs {
		if seen[ref] {
			continue
		}
		seen[ref] = true
		missing = append(missing, ref)
	}
	return missing
}

func planImportInitiativeRefsEqual(a, b *initiatives.PlanRef) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Provider == b.Provider && a.PlanID == b.PlanID && a.Slug == b.Slug && a.Role == b.Role
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func planImportRefsEqual(a, b *backlog.PlanRef) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Provider == b.Provider && a.PlanID == b.PlanID && a.Slug == b.Slug && a.Role == b.Role
}

// goalReadyAdapter adapts *goals.Service to execution.GoalReadyProvider,
// converting goals.ReadyGoalItem to the execution-package shape so the
// execution service need not import goals.
type goalReadyAdapter struct{ svc *goals.Service }

func (a goalReadyAdapter) ReadyGoalItems() ([]execution.GoalReadyItem, error) {
	items, err := a.svc.ReadyGoalItems()
	if err != nil {
		return nil, err
	}
	out := make([]execution.GoalReadyItem, 0, len(items))
	for _, it := range items {
		out = append(out, execution.GoalReadyItem{Kind: it.Kind, Name: it.Name, GoalPriority: it.GoalPriority})
	}
	return out, nil
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

func (s *Server) registerRecordsRoutes(dataRoot, scenariosDir string) {
	store := records.NewFileStore(dataRoot)
	svc := records.NewService(store, nil, nil)
	handler := records.NewHandler(svc, nil)
	// Warn (never block) when a record targets an off-registry slug, so typos
	// stop fragmenting the learning corpus. Known slugs = scenario, package,
	// and resource directories, plus "vrooli" for repo-level work.
	repoRoot := filepath.Dir(scenariosDir)
	handler.SetScenarioChecker(records.NewDirectoryScenarioChecker([]string{
		scenariosDir,
		filepath.Join(repoRoot, "packages"),
		filepath.Join(repoRoot, "resources"),
	}, "vrooli"))
	handler.RegisterRoutes(s.router)
	s.recordsService = svc
	s.recordsHandler = handler
	s.recordsStore = store
	// Capture hook: auto-write a filled, indexed record on backlog terminal
	// transitions (the recursive-learning write-side). Backlog must already be
	// registered (it is — registerBacklogRoutes runs before
	// registerRecordsRoutes); guard defensively anyway.
	if s.backlogHandler != nil {
		s.backlogHandler.SetRecordCreator(newRecordCaptureAdapter(svc))
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

	// Register the granular measures surface (requires the event log): the
	// measures-go serve registry at /measures (GET /declarations, POST /execute)
	// — consumed by the measures-health behavioral probe and the search-hub
	// central index — plus the typed Connect MeasuresService. Both share one
	// compute path so a measure and its RPC can never report different numbers.
	if srv.eventRepo != nil {
		measuresHandler, err := measureshandler.MeasuresHandler(srv.eventRepo, nil)
		if err != nil {
			slog.Error("measures registry init error", "error", err)
		} else {
			srv.router.PathPrefix("/measures/").Handler(http.StripPrefix("/measures", measuresHandler))
			measureshandler.RegisterRoutes(srv.router, srv.eventRepo, nil)
		}
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

	// Declarative operation runner: poll active target rounds toward completion
	// (nothing else drives a non-initiative target round) and fire due scheduled
	// intents. Both reload durable workflow state each tick, so a restart resumes
	// cleanly. The stop chan cancels the shared context on shutdown.
	if srv.backlogOpsRunner != nil {
		go func(built *opsbridge.BacklogRunner, stop <-chan struct{}) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() { <-stop; cancel() }()
			go func() {
				if err := built.RefreshDriver.Run(ctx, 5*time.Second); err != nil && ctx.Err() == nil {
					slog.Warn("backlog-ops refresh driver stopped", "err", err)
				}
			}()
			if err := built.Scheduler.Run(ctx, 5*time.Second); err != nil && ctx.Err() == nil {
				slog.Warn("backlog-ops scheduler stopped", "err", err)
			}
		}(srv.backlogOpsRunner, srv.opsRunnerStopChan)
	}

	if srv.autoFilerSweeper != nil {
		go func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() {
				<-srv.autoFilerStopChan
				cancel()
			}()
			srv.autoFilerSweeper.Start(ctx)
		}()
	}

	if srv.agentSvc != nil && srv.agentSvc.IsEnabled() {
		initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := srv.agentSvc.Initialize(initCtx); err != nil {
			cancel()
			log.Fatalf("failed to initialize agent-manager profiles: %v", err)
		}
		cancel()
	}

	srv.startAISearchBackground()
	srv.startSearchRegistration()

	// Top-level mux mounts the API handler plus, in development mode, the
	// dev-only RoutingService test-genie calls to install a runtime test DB pool
	// without restarting this scenario. The RoutingService's own path is more
	// specific than "/", so http.ServeMux routes it ahead of the API handler.
	rootMux := http.NewServeMux()
	if srv.eventDB != nil {
		devrouting.Register(rootMux, srv.eventDB)
	}
	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the installed test
	// pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := server.Run(server.Config{
		Handler:      handler,
		WriteTimeout: 180 * time.Second,
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
	close(srv.executionStopChan)
	close(srv.reviewStopChan)
	close(srv.initReviewStopChan)
	close(srv.opsRunnerStopChan)
	close(srv.aiSearchStopChan)
	close(srv.reviewSweeperStop)
	close(srv.autoFilerStopChan)
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

// startSearchRegistration self-registers swarm-manager's search provider(s)
// with search-hub from the `.vrooli/search.json` SSOT. Today that is the
// `records` leaf, mapped to the rich POST /api/v1/records/search endpoint so a
// federated hit carries the record's trigger/approach lesson (not just an id —
// the thin /search/ai entity payload it used to be reachable through).
//
// search-hub is an OPTIONAL dependency, so this runs in a background goroutine
// with bounded retry and degrades gracefully: the scenario keeps serving
// records search whether or not the hub is up, and the registry upsert is
// idempotent, so re-registering on every boot is safe. The bounded retry means
// the goroutine terminates on its own, so a plain Background context is fine.
func (s *Server) startSearchRegistration() {
	searchJSONPath := filepath.Join(s.scenarioRoot, ".vrooli", "search.json")
	if _, err := os.Stat(searchJSONPath); err != nil {
		// No search.json on this root (e.g. a test temp dir) — nothing to register.
		return
	}
	go searchregister.Register(context.Background(), searchregister.Config{
		ScenarioID:     "swarm-manager",
		SearchFilePath: searchJSONPath,
	})
}

func getEnvDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
