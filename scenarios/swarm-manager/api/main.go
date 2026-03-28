// Package main provides the Swarm Manager API server.
//
// DOC: docs/concepts/ARCHITECTURE.md#logical-architecture
// DOC: docs/internal/INTENT.md#api-components
//
// # Purpose
//
// This API serves as the backend for the Swarm Manager UI, providing endpoints
// for managing the scenario ecosystem: backlog, scenario lifecycle, and
// execution control.
//
// # Current Status: Backlog/Scenarios/Settings/Execution Implemented (File-Based)
//
// The API provides:
//   - Health check endpoints at / and /api/v1/health
//   - Backlog CRUD endpoints at /api/v1/backlog (GET, POST, PUT, DELETE)
//   - Backlog queue + research endpoints at /api/v1/backlog/{kind}/{name}/queue and /api/v1/backlog/{kind}/{name}/research
//   - Scenario catalog endpoints at /api/v1/scenarios
//   - Settings persistence at /api/v1/settings
//   - Execution control at /api/v1/execution
//   - Agent-manager status at /api/v1/agent-manager/status
//
// # Architecture
//
// The server uses:
//   - gorilla/mux for HTTP routing
//   - api-core/health for standardized health responses
//   - api-core/server for graceful shutdown
//   - File-system based storage for backlog items (git-tracked in scenarios/swarm-manager/{ideas,research,fix,execute}/)
//
// # Implemented Endpoints (P0)
//
//	GET    /api/v1/backlog                   - List all backlog items
//	POST   /api/v1/backlog                   - Create new backlog item
//	GET    /api/v1/backlog/{kind}/{name}     - Get backlog item by name
//	PUT    /api/v1/backlog/{kind}/{name}     - Update backlog item
//	DELETE /api/v1/backlog/{kind}/{name}     - Delete backlog item
//	POST   /api/v1/backlog/{kind}/{name}/queue    - Queue backlog item for processing
//	POST   /api/v1/backlog/{kind}/{name}/research - Spawn research agent
//
// # Scenario Endpoints (P0)
//
//	GET    /api/v1/scenarios      - List all scenarios (supports search, filter, sort)
//	GET    /api/v1/scenarios/{name} - Get scenario details
//	POST   /api/v1/scenarios/{name}/start - Start scenario via CLI
//	POST   /api/v1/scenarios/{name}/stop - Stop scenario via CLI
//	POST   /api/v1/scenarios/{name}/restart - Restart scenario via CLI
//
// # Settings Endpoints (P1)
//
//	GET    /api/v1/settings       - Fetch settings
//	PUT    /api/v1/settings       - Update settings (partial)
//
// # Agent-manager Endpoints
//
//	GET /api/v1/agent-manager/status - Agent-manager availability and profile status
//
// # Overview Endpoint
//
//	GET    /api/v1/overview      - Aggregated overview (items, initiatives, dep graph, summary)
//
// # Queue Endpoints (Local)
//
//	GET    /api/v1/queue          - List queue items
//	POST   /api/v1/queue          - Enqueue item
//	DELETE /api/v1/queue/{id}     - Remove item (idempotent)
//
// # Execution Endpoints (Core)
//
//	GET    /api/v1/execution                            - List execution runs
//	POST   /api/v1/execution                            - Create execution run
//	GET    /api/v1/execution/policy                     - Get execution policy defaults
//	PUT    /api/v1/execution/policy                     - Update execution policy defaults
//	GET    /api/v1/execution/{execution_id}             - Get execution run
//	POST   /api/v1/execution/{execution_id}/start       - Start run
//	POST   /api/v1/execution/{execution_id}/cancel      - Cancel run
//	POST   /api/v1/execution/{execution_id}/retry       - Retry failed run
//
// Related PRD targets: OT-P0-002, OT-P0-005, OT-P0-006
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/captures"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/overview"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/prompts"
	"swarm-manager/internal/queue"
	"swarm-manager/internal/scenarios"
	"swarm-manager/internal/settings"
)

type Server struct {
	router            *mux.Router
	agentSvc          *agentmanager.AgentService
	settingsStore     *settings.Store
	backlogHandler    *backlog.Handler
	capturesHandler   *captures.Handler
	scenariosHandler  *scenarios.Handler
	initiativeService *initiatives.Service
	executionSvc      *execution.Service
	executionHandler  *execution.Handler
	executionStopChan chan struct{}
	graphBroker       *graph.Broker
	scenarioRoot      string
}

// settingsAgentAdapter bridges settings.Store to agentmanager.SettingsReader.
type settingsAgentAdapter struct {
	store *settings.Store
}

func (a *settingsAgentAdapter) LoadAgentSettings() (maxTurns, timeoutSeconds int32, requiresApproval bool, err error) {
	s, err := a.store.Load()
	if err != nil {
		return 0, 0, false, err
	}
	return int32(s.AgentMaxTurns), int32(s.AgentTimeoutSeconds), s.AgentRequiresApproval, nil
}

// settingsPolicyAdapter bridges settings.Store to execution.PolicyProvider.
type settingsPolicyAdapter struct {
	store *settings.Store
}

func (a *settingsPolicyAdapter) LoadPolicy() (execution.Policy, error) {
	s, err := a.store.Load()
	if err != nil {
		return execution.Policy{}, err
	}
	return execution.Policy{
		DefaultMode:         execution.Mode(s.DefaultMode),
		DefaultDelaySeconds: s.DefaultDelaySeconds,
		AutoFixup:           s.AutoFixup,
		MaxFixupAttempts:    s.MaxFixupAttempts,
	}, nil
}

// settingsReviewThresholdsAdapter bridges settings.Store to execution.ReviewThresholdsProvider.
type settingsReviewThresholdsAdapter struct {
	store *settings.Store
}

func (a *settingsReviewThresholdsAdapter) LoadReviewThresholds() (*execution.ReviewThresholds, error) {
	s, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	return &execution.ReviewThresholds{
		CodeQualityMinScore:   s.ReviewCodeQualityMinScore,
		TestMinPassRate:       s.ReviewTestMinPassRate,
		MaxBlockingViolations: s.ReviewMaxBlockingViolations,
		MaxWarnings:           s.ReviewMaxWarnings,
		RequireScreenshots:    s.ReviewRequireScreenshots,
		RequireTests:          s.ReviewRequireTests,
	}, nil
}

// NewServer initializes routes using the default scenario root resolved from
// the environment. Database connection is optional.
func NewServer() *Server {
	return NewServerWithRoot(pathutil.ResolveScenarioRoot("swarm-manager"))
}

// NewServerWithRoot initializes routes using the given scenario root directory.
// Tests should use this with t.TempDir() to avoid touching production data.
func NewServerWithRoot(scenarioRoot string) *Server {
	agentEnabled := strings.ToLower(strings.TrimSpace(os.Getenv("AGENT_MANAGER_ENABLED"))) != "false"
	agentSvc := agentmanager.NewAgentService(agentmanager.AgentServiceConfig{
		ProfileName: getEnvDefault("AGENT_MANAGER_PROFILE_NAME", "swarm-manager"),
		ProfileKey:  getEnvDefault("AGENT_MANAGER_PROFILE_KEY", "swarm-manager"),
		Timeout:     30 * time.Second,
		Enabled:     agentEnabled,
	})

	srv := &Server{
		router:            mux.NewRouter(),
		agentSvc:          agentSvc,
		executionStopChan: make(chan struct{}),
		scenarioRoot:      scenarioRoot,
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	scenarioRoot := s.scenarioRoot
	scenariosDir := filepath.Dir(scenarioRoot)
	s.registerHealthRoutes()
	s.registerSettingsRoutes(scenarioRoot) // Must be before backlog/execution (they depend on settings store)
	backlogHandler := s.registerBacklogRoutes(scenarioRoot)
	initService := s.registerInitiativeRoutes(scenarioRoot, backlogHandler)
	s.registerOverviewRoutes(backlogHandler, initService)
	s.registerCapturesRoutes(scenarioRoot, backlogHandler)
	s.registerScenarioRoutes(scenariosDir)
	s.registerAgentManagerRoutes()
	s.registerQueueRoutes(scenarioRoot)
	s.registerExecutionRoutes(scenarioRoot)
	s.registerGraphRoutes(scenarioRoot)
	s.registerPromptRoutes(scenarioRoot)
}

func (s *Server) registerHealthRoutes() {
	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	// Uses api-core/health for standardized response format
	healthHandler := health.New().Version("1.0.0").
		Check(health.CheckerFunc(func(_ context.Context) health.CheckResult {
			return health.CheckResult{Name: "database", Connected: true}
		}), health.Optional).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
}

func (s *Server) registerBacklogRoutes(scenarioRoot string) *backlog.Handler {
	// Backlog endpoints
	// [REQ:REQ-P0-002] Backlog management
	backlogHandler := backlog.NewHandlerWithClients(scenarioRoot, s.agentSvc, nil)
	backlogHandler.SetPolicyProvider(&settingsPolicyAdapter{store: s.settingsStore})
	backlogHandler.RegisterRoutes(s.router)
	s.backlogHandler = backlogHandler
	return backlogHandler
}

func (s *Server) registerInitiativeRoutes(scenarioRoot string, backlogHandler *backlog.Handler) *initiatives.Service {
	// Initiative endpoints for grouping backlog items into work streams.
	initStore := initiatives.NewStore(scenarioRoot)
	initService := initiatives.NewService(initStore, backlogHandler.Store())
	initHandler := initiatives.NewHandler(initService)
	initHandler.RegisterRoutes(s.router)
	s.initiativeService = initService

	// Wire initiative assigner into backlog handler for batch operations.
	backlogHandler.SetInitiativeAssigner(&initiativeAssignerAdapter{service: initService})
	return initService
}

// initiativeAssignerAdapter bridges the initiatives.Service to the
// backlog.InitiativeAssigner interface, avoiding a direct import cycle.
type initiativeAssignerAdapter struct {
	service *initiatives.Service
}

func (a *initiativeAssignerAdapter) Get(name string) (*backlog.InitiativeSnapshot, error) {
	result, err := a.service.Get(name)
	if err != nil {
		return nil, err
	}
	return &backlog.InitiativeSnapshot{
		Name:        result.Initiative.Name,
		Title:       result.Initiative.Title,
		Description: result.Initiative.Description,
		Status:      result.Initiative.Status,
		Items:       append([]string(nil), result.Initiative.Items...),
	}, nil
}

func (a *initiativeAssignerAdapter) Create(spec backlog.InitiativeSpec) error {
	_, err := a.service.Create(initiatives.CreateRequest{
		Name:        spec.Name,
		Title:       spec.Title,
		Description: spec.Description,
		Status:      spec.Status,
	})
	return err
}

func (a *initiativeAssignerAdapter) Update(spec backlog.InitiativeSpec) error {
	title := spec.Title
	description := spec.Description
	status := spec.Status
	_, err := a.service.Update(spec.Name, initiatives.UpdateRequest{
		Title:       &title,
		Description: &description,
		Status:      &status,
	})
	return err
}

func (a *initiativeAssignerAdapter) Replace(snapshot backlog.InitiativeSnapshot) error {
	return a.service.Replace(initiatives.Initiative{
		Name:        snapshot.Name,
		Title:       snapshot.Title,
		Description: snapshot.Description,
		Status:      snapshot.Status,
		Items:       append([]string(nil), snapshot.Items...),
	})
}

func (a *initiativeAssignerAdapter) Delete(name string) error {
	return a.service.Delete(name)
}

func (a *initiativeAssignerAdapter) AddItems(name string, items []string) error {
	return a.service.AddItems(name, items)
}

func (s *Server) registerOverviewRoutes(backlogHandler *backlog.Handler, initService *initiatives.Service) {
	// Overview aggregation endpoint for situational awareness.
	overviewSvc := overview.NewService(backlogHandler.Store(), initService)
	overviewHandler := overview.NewHandler(overviewSvc)
	overviewHandler.RegisterRoutes(s.router)
}

func (s *Server) registerCapturesRoutes(scenarioRoot string, backlogHandler *backlog.Handler) {
	// Captures endpoints for quick-capture unified feed
	capturesHandler := captures.NewHandler(scenarioRoot, s.agentSvc, nil)
	capturesHandler.SetBacklogCreator(&backlogItemCreatorAdapter{store: backlogHandler.Store()})
	capturesHandler.RegisterRoutes(s.router)
	s.capturesHandler = capturesHandler
}

func (s *Server) registerScenarioRoutes(scenariosDir string) {
	// Scenarios catalog endpoints
	// [REQ:REQ-P0-006] Scenario catalog with priority, search, and filter
	s.scenariosHandler = scenarios.NewHandler(scenariosDir)
	s.scenariosHandler.RegisterRoutes(s.router)
}

func (s *Server) registerSettingsRoutes(scenarioRoot string) {
	// Settings persistence endpoints
	settingsPath := filepath.Join(scenarioRoot, ".vrooli", "settings.json")
	settingsHandler := settings.NewHandler(settingsPath)
	settingsHandler.RegisterRoutes(s.router)
	s.settingsStore = settingsHandler.GetStore()

	// Wire settings into agent service for runtime profile config resolution.
	if s.agentSvc != nil {
		s.agentSvc.SetSettingsReader(&settingsAgentAdapter{store: s.settingsStore})
	}
}

func (s *Server) registerAgentManagerRoutes() {
	// Agent-manager status endpoint
	agentManagerHandler := agentmanager.NewHandler(s.agentSvc)
	agentManagerHandler.RegisterRoutes(s.router)
}

func (s *Server) registerQueueRoutes(scenarioRoot string) {
	// Local queue endpoints (filesystem-backed)
	queueHandler := queue.NewHandler(filepath.Join(scenarioRoot, ".vrooli", "queue.json"))
	queueHandler.RegisterRoutes(s.router)
}

func (s *Server) registerExecutionRoutes(scenarioRoot string) {
	// Create archiver from scenarios handler for post-spec-sync archive
	var archiver execution.Archiver
	if s.scenariosHandler != nil {
		archiver = scenarios.NewArchiver(s.scenariosHandler)
	}

	// Execution control endpoints
	cfg := execution.ServiceConfig{
		RootDir:                  scenarioRoot,
		StorePath:                filepath.Join(scenarioRoot, ".vrooli", "execution-runs.json"),
		PolicyProvider:           &settingsPolicyAdapter{store: s.settingsStore},
		ReviewThresholdsProvider: &settingsReviewThresholdsAdapter{store: s.settingsStore},
		AgentService:             s.agentSvc,
		Archiver:                 archiver,
		ReviewClient:             execution.NewHTTPReviewClient(nil),
	}
	s.executionSvc = execution.NewService(cfg)
	s.executionHandler = execution.NewHandlerFromService(s.executionSvc)
	s.executionHandler.RegisterRoutes(s.router)

	// Wire execution queuer back into scenarios handler for spec-sync-archive
	if s.scenariosHandler != nil {
		s.scenariosHandler.SetExecutionQueuer(scenarios.NewExecutionQueuer(s.executionSvc))
	}
	if s.backlogHandler != nil {
		s.backlogHandler.SetExecutionQueuer(s.executionSvc)
	}
}

// captureAdapter bridges captures.Handler filesystem logic to graph.CaptureLister.
type captureAdapter struct {
	rootDir string
}

func (a *captureAdapter) ListCaptures() ([]graph.CaptureEntry, error) {
	capturesRoot := filepath.Join(a.rootDir, "captures")
	entries, err := os.ReadDir(capturesRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []graph.CaptureEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		capPath := filepath.Join(capturesRoot, entry.Name(), "capture.json")
		data, err := os.ReadFile(capPath)
		if err != nil {
			continue
		}
		var raw struct {
			ID     string `json:"id"`
			Text   string `json:"text"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}

		ce := graph.CaptureEntry{
			ID:     raw.ID,
			Text:   raw.Text,
			Status: raw.Status,
		}

		// Load classification if it exists.
		classPath := filepath.Join(capturesRoot, entry.Name(), "classification.json")
		classData, err := os.ReadFile(classPath)
		if err == nil {
			var cls struct {
				Items []struct {
					Kind  string `json:"kind"`
					Title string `json:"title"`
				} `json:"items"`
			}
			if json.Unmarshal(classData, &cls) == nil {
				for _, item := range cls.Items {
					ce.Items = append(ce.Items, graph.CaptureClassificationItem{
						Kind:  item.Kind,
						Title: item.Title,
					})
				}
			}
		}

		result = append(result, ce)
	}
	return result, nil
}

// graphInitiativeAdapter bridges the initiatives store to graph.InitiativeLister.
type graphInitiativeAdapter struct {
	store *initiatives.Store
}

func (a *graphInitiativeAdapter) List() ([]graph.InitiativeEntry, error) {
	items, err := a.store.LoadAll()
	if err != nil {
		return nil, err
	}

	result := make([]graph.InitiativeEntry, 0, len(items))
	for _, item := range items {
		result = append(result, graph.InitiativeEntry{
			Name:   item.Name,
			Title:  item.Title,
			Status: item.Status,
			Items:  append([]string(nil), item.Items...),
		})
	}
	return result, nil
}

// executionAdapter bridges execution.Service to graph.ExecutionLister.
type executionAdapter struct {
	svc *execution.Service
}

func (a *executionAdapter) List(ctx context.Context, filters execution.ListFilters) ([]execution.Record, error) {
	return a.svc.List(ctx, filters)
}

// runStateAdapter bridges agentmanager.AgentService to graph.RunStateGetter.
type runStateAdapter struct {
	svc *agentmanager.AgentService
}

func (a *runStateAdapter) IsAvailable(ctx context.Context) bool {
	return a.svc.IsAvailable(ctx)
}

func (a *runStateAdapter) GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error) {
	return a.svc.GetRunState(ctx, runID)
}

// backlogItemCreatorAdapter bridges backlog.Store to captures.BacklogItemCreator.
type backlogItemCreatorAdapter struct {
	store backlog.Store
}

func (a *backlogItemCreatorAdapter) ItemDir(kind, name string) string {
	return a.store.ItemDir(backlog.BacklogKind(kind), name)
}

func (a *backlogItemCreatorAdapter) SaveItem(kind, name, title, description string, tags []string) error {
	bk := backlog.BacklogKind(kind)
	itemDir := a.store.ItemDir(bk, name)
	if err := os.MkdirAll(filepath.Dir(itemDir), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	if err := os.Mkdir(itemDir, 0o755); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("already exists")
		}
		return fmt.Errorf("create item dir: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item := backlog.BacklogItem{
		Name:        name,
		Title:       title,
		Description: description,
		Status:      backlog.StatusBacklog,
		Priority:    5,
		Tags:        tags,
		Created:     now,
		Updated:     now,
		Kind:        bk,
	}
	if err := a.store.SaveItem(item); err != nil {
		_ = os.RemoveAll(itemDir)
		return fmt.Errorf("save item: %w", err)
	}
	return nil
}

func (s *Server) registerGraphRoutes(scenarioRoot string) {
	if s.backlogHandler == nil {
		return
	}

	projCfg := graph.ProjectionConfig{
		Backlog:    s.backlogHandler.Store(),
		Initiative: &graphInitiativeAdapter{store: initiatives.NewStore(scenarioRoot)},
		Capture:    &captureAdapter{rootDir: scenarioRoot},
		Scenario: graph.NewScenarioSourceAdapter(
			scenarios.NewCLIProviderWithOptions(scenarios.CLIProviderOptions{
				IncludePorts: false,
			}),
		),
	}
	if s.executionSvc != nil {
		projCfg.Execution = &executionAdapter{svc: s.executionSvc}
	}
	if s.agentSvc != nil && s.agentSvc.IsEnabled() {
		projCfg.RunState = &runStateAdapter{svc: s.agentSvc}
	}
	projSvc := graph.NewProjectionService(projCfg)
	projectionCache := graph.NewProjectionCache(graph.ProjectionCacheConfig{
		Projector: projSvc,
	})

	// HTTP handler.
	graphHandler := graph.NewHandler(projectionCache)
	graphHandler.RegisterRoutes(s.router)

	// WebSocket broker and stream handler.
	s.graphBroker = graph.NewBroker()
	streamHandler := graph.NewStreamHandler(s.graphBroker)
	streamHandler.RegisterRoutes(s.router)

	// Wire graph invalidation into mutating services and handlers.
	dispatch := graph.NewDispatch(s.graphBroker, projectionCache)
	if s.backlogHandler != nil {
		s.backlogHandler.SetEventDispatcher(dispatch)
	}
	if s.capturesHandler != nil {
		s.capturesHandler.SetEventDispatcher(dispatch)
	}
	if s.initiativeService != nil {
		s.initiativeService.SetEventDispatcher(dispatch)
	}
	if s.executionSvc != nil {
		s.executionSvc.SetEventDispatcher(dispatch)
	}
	if s.scenariosHandler != nil {
		s.scenariosHandler.SetEventDispatcher(dispatch)
	}
}

func (s *Server) registerPromptRoutes(scenarioRoot string) {
	promptHandler := prompts.NewHandler(scenarioRoot, nil)
	promptHandler.RegisterRoutes(s.router)
}

// Handler returns the HTTP handler with recovery middleware
func (s *Server) Handler() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "swarm-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	log.Printf("Running in filesystem-only mode")
	srv := NewServer()
	if srv.executionHandler != nil {
		go srv.executionHandler.StartScheduler(srv.executionStopChan)
	}

	if srv.agentSvc != nil && srv.agentSvc.IsEnabled() {
		initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := srv.agentSvc.Initialize(initCtx, nil); err != nil {
			log.Printf("[agent-manager] Warning: failed to initialize profile: %v", err)
		}
		cancel()
	}

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler: srv.Handler(),
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
	close(srv.executionStopChan)
}

func getEnvDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
