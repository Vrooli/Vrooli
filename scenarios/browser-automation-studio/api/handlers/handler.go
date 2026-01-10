package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autodriver "github.com/vrooli/browser-automation-studio/automation/driver"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	autosession "github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/domain"
	aihandlers "github.com/vrooli/browser-automation-studio/handlers/ai"
	"github.com/vrooli/browser-automation-studio/internal/paths"
	"github.com/vrooli/browser-automation-studio/performance"
	archiveingestion "github.com/vrooli/browser-automation-studio/services/archive-ingestion"
	"github.com/vrooli/browser-automation-studio/services/ai"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	"github.com/vrooli/browser-automation-studio/services/export"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
	"github.com/vrooli/browser-automation-studio/services/export/render"
	"github.com/vrooli/browser-automation-studio/services/testgenie"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics"
	uxcollector "github.com/vrooli/browser-automation-studio/services/uxmetrics/collector"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
)

// Handler contains all HTTP handlers
type replayRenderer interface {
	Render(ctx context.Context, spec *export.ReplayMovieSpec, format render.RenderFormat, filename string) (*render.RenderedMedia, error)
}

// RecordModeService defines the interface for live recording session management.
// This interface allows for testing with mock implementations.
//
// Design note: Pass-through operations (Navigate, Reload, etc.) are handled directly
// via DriverClient() to reduce unnecessary indirection. The service layer focuses on
// operations that require business logic (session lifecycle, recording callbacks, timeline).
type RecordModeService interface {
	// DriverClient returns the underlying driver client for direct pass-through operations.
	// Handlers should use this for operations that don't require service-level business logic.
	DriverClient() autodriver.ClientInterface

	// Session lifecycle
	CreateSession(ctx context.Context, cfg *livecapture.SessionConfig) (*livecapture.SessionResult, error)
	CloseSession(ctx context.Context, sessionID string) error
	GetStorageState(ctx context.Context, sessionID string) (json.RawMessage, error)

	// Recording lifecycle (has business logic for callback URL construction)
	StartRecording(ctx context.Context, sessionID string, cfg *livecapture.RecordingConfig) (*autodriver.StartRecordingResponse, error)

	// Workflow generation (has business logic for action conversion)
	GenerateWorkflow(ctx context.Context, sessionID string, cfg *livecapture.GenerateWorkflowConfig) (*livecapture.GenerateWorkflowResult, error)

	// Multi-page support (has business logic for page tracking)
	GetSession(sessionID string) (*autosession.Session, bool)
	GetPages(sessionID string) (*livecapture.PageListResult, error)
	GetOpenPages(sessionID string) ([]*domain.Page, uuid.UUID, error)
	ActivatePage(ctx context.Context, sessionID string, pageID uuid.UUID) error
	CreatePage(ctx context.Context, sessionID string, url string) (*autodriver.CreatePageResponse, error)
	RestoreTabs(ctx context.Context, sessionID string, tabs []archiveingestion.TabState) (*livecapture.TabRestorationResult, error)

	// Timeline support (has business logic for timeline management)
	AddTimelineAction(sessionID string, action *autodriver.RecordedAction, pageID uuid.UUID)
	AddTimelinePageEvent(sessionID string, event *domain.PageEvent)
	GetTimeline(sessionID string, pageID *uuid.UUID, limit int) (*domain.TimelineResponse, error)

	// Service worker management (requires session lookup)
	GetServiceWorkers(ctx context.Context, sessionID string) (*autodriver.GetServiceWorkersResponse, error)
	UnregisterAllServiceWorkers(ctx context.Context, sessionID string) (*autodriver.UnregisterServiceWorkersResponse, error)
	UnregisterServiceWorker(ctx context.Context, sessionID, scopeURL string) (*autodriver.UnregisterServiceWorkerResponse, error)
}

// Handler contains all HTTP handlers
type Handler struct {
	// Service interfaces - clearly separated responsibilities
	catalogService   workflow.CatalogService   // Workflow/project CRUD, versioning, sync
	executionService workflow.ExecutionService // Execution lifecycle, timeline, export

	workflowValidator *workflowvalidator.Validator
	repo              database.Repository
	wsHub             wsHub.HubInterface
	storage           storage.StorageInterface
	recordingService  archiveingestion.IngestionServiceInterface
	recordModeService RecordModeService // Live recording session management (interface for testability)
	recordingsRoot    string
	replayRenderer    replayRenderer
	sessionProfiles   *archiveingestion.SessionProfileStore
	log               *logrus.Logger
	upgrader          websocket.Upgrader
	wsAllowAll        bool
	wsAllowedOrigins  []string

	// Performance monitoring
	perfRegistry *performance.CollectorRegistry
	seedCleanupManager *testgenie.SeedCleanupManager

	// AI subhandlers
	screenshotHandler        *aihandlers.ScreenshotHandler
	domHandler               *aihandlers.DOMHandler
	elementAnalysisHandler   *aihandlers.ElementAnalysisHandler
	aiAnalysisHandler        *aihandlers.AIAnalysisHandler
	visionNavigationHandler  *aihandlers.VisionNavigationHandler

	// Entitlement service for tier-based feature gating
	entitlementService *entitlement.Service
}

// recordingUploadLimitBytes returns the configured maximum upload size for recording archives.
// Uses config.Load().Recording.MaxArchiveBytes - configurable via BAS_RECORDING_MAX_ARCHIVE_BYTES
func recordingUploadLimitBytes() int64 {
	return config.Load().Recording.MaxArchiveBytes
}

// recordingImportTimeout returns the configured timeout for recording imports.
// Uses config.Load().Recording.ImportTimeout - configurable via BAS_RECORDING_IMPORT_TIMEOUT_MS
func recordingImportTimeout() time.Duration {
	return config.Load().Recording.ImportTimeout
}

// eventBufferLimits returns validated event buffer limits sourced from config.
// Delegates to config.EventBufferLimitsFromConfig() for centralized configuration.
func eventBufferLimits() autocontracts.EventBufferLimits {
	return config.EventBufferLimitsFromConfig()
}

// HandlerDeps holds all dependencies for the Handler.
// This struct separates dependency wiring from handler construction.
type HandlerDeps struct {
	// Service interfaces - now using proper interface types
	CatalogService   workflow.CatalogService   // Workflow/project CRUD, versioning, sync
	ExecutionService workflow.ExecutionService // Execution lifecycle, timeline, export

	WorkflowValidator    *workflowvalidator.Validator
	Storage              storage.StorageInterface
	RecordingService     archiveingestion.IngestionServiceInterface
	RecordModeService    RecordModeService // Live recording session management (interface for testability)
	RecordingsRoot       string
	ReplayRenderer       replayRenderer
	SessionProfiles      *archiveingestion.SessionProfileStore
	UXMetricsRepo        uxmetrics.Repository          // Optional: enables UX metrics collection
	EntitlementService   *entitlement.Service          // Optional: enables tier-based feature gating
	AICreditsTracker     *entitlement.AICreditsTracker // Optional: enables AI credits tracking
}

// InitDefaultDeps initializes the standard production dependencies.
// This function is responsible for infrastructure wiring, keeping it separate
// from handler construction for clearer responsibility boundaries.
func InitDefaultDeps(repo database.Repository, wsHub *wsHub.Hub, log *logrus.Logger) HandlerDeps {
	return InitDefaultDepsWithUXMetrics(repo, wsHub, log, nil)
}

// DepsOptions holds optional dependencies for handler initialization.
type DepsOptions struct {
	UXMetricsRepo      uxmetrics.Repository
	EntitlementService *entitlement.Service
	AICreditsTracker   *entitlement.AICreditsTracker
}

// InitDefaultDepsWithUXMetrics initializes dependencies with optional UX metrics collection.
// When uxRepo is provided, the UX metrics collector wraps the event sink to passively
// capture interaction data during workflow executions.
func InitDefaultDepsWithUXMetrics(repo database.Repository, wsHub *wsHub.Hub, log *logrus.Logger, uxRepo uxmetrics.Repository) HandlerDeps {
	return InitDefaultDepsWithOptions(repo, wsHub, log, DepsOptions{UXMetricsRepo: uxRepo})
}

// InitDefaultDepsWithOptions initializes dependencies with all optional components.
// This enables proper integration of entitlement services for AI credits tracking.
func InitDefaultDepsWithOptions(repo database.Repository, wsHub *wsHub.Hub, log *logrus.Logger, opts DepsOptions) HandlerDeps {
	// Initialize recordings infrastructure
	recordingsRoot := paths.ResolveRecordingsRoot(log)
	// Store screenshots alongside other execution artifacts under recordingsRoot.
	storageClient := storage.NewScreenshotStorage(log, recordingsRoot)
	recordingService := archiveingestion.NewIngestionService(repo, storageClient, wsHub, log, recordingsRoot)
	sessionProfiles := archiveingestion.NewSessionProfileStore(paths.ResolveSessionProfilesRoot(log), log)

	// Wire automation stack
	autoExecutor := autoexecutor.NewSimpleExecutor(nil)
	autoEngineFactory, engErr := autoengine.DefaultFactoryWithRecordingsRoot(log, recordingsRoot)
	if engErr != nil {
		log.WithError(engErr).Warn("Failed to initialize automation engine; automation executor will be disabled")
	}
	// Persist execution artifacts under recordingsRoot so file-truth execution data is durable and discoverable.
	autoRecorder := executionwriter.NewFileWriter(repo, storageClient, log, recordingsRoot)

	// Configure event sink factory - optionally wrap with UX metrics collector
	var eventSinkFactory func() autoevents.Sink
	if opts.UXMetricsRepo != nil {
		// Create UX metrics collector that wraps the WebSocket sink
		// The collector passively captures interaction data while delegating events to the hub
		uxRepo := opts.UXMetricsRepo // Capture for closure
		eventSinkFactory = func() autoevents.Sink {
			baseSink := autoevents.NewWSHubSink(wsHub, log, eventBufferLimits())
			return uxcollector.NewCollector(baseSink, uxRepo)
		}
		log.Debug("UX metrics collector enabled in event pipeline")
	}

	// Create AI client - wrap with credits tracking if entitlement services are available
	var aiClient ai.AIClient = ai.NewOpenRouterClient(log)
	if opts.EntitlementService != nil && opts.AICreditsTracker != nil {
		aiClient = ai.NewCreditsClient(ai.CreditsClientOptions{
			Inner:            aiClient,
			EntitlementSvc:   opts.EntitlementService,
			AICreditsTracker: opts.AICreditsTracker,
			Logger:           log,
			UserIdentityFn:   entitlement.UserIdentityFromContext,
		})
		log.Debug("AI credits tracking enabled")
	}

	// Create workflow service with dependencies
	workflowSvc := workflow.NewWorkflowServiceWithDeps(repo, wsHub, log, workflow.WorkflowServiceOptions{
		Executor:          autoExecutor,
		EngineFactory:     autoEngineFactory,
		ArtifactRecorder:  autoRecorder,
		EventSinkFactory:  eventSinkFactory,
		ExecutionDataRoot: recordingsRoot,
		AIClient:          aiClient,
	})

	// Ensure the demo project exists so file-first operations have a stable project root.
	if workflowSvc != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if _, err := workflowSvc.EnsureSeedProject(ctx); err != nil && log != nil {
			log.WithError(err).Warn("Failed to ensure seed project")
		}
		cancel()
	}

	// Create validator
	validatorInstance, err := workflowvalidator.NewValidator()
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize workflow validator")
	}

	// Create record mode service for live recording session management
	recordModeSvc := livecapture.NewService(log)

	return HandlerDeps{
		// WorkflowService implements both CatalogService and ExecutionService interfaces
		CatalogService:     workflowSvc,
		ExecutionService:   workflowSvc,
		WorkflowValidator:  validatorInstance,
		Storage:            storageClient,
		RecordingService:   recordingService,
		RecordModeService:  recordModeSvc,
		RecordingsRoot:     recordingsRoot,
		ReplayRenderer:     render.NewReplayRenderer(log, recordingsRoot),
		SessionProfiles:    sessionProfiles,
		UXMetricsRepo:      opts.UXMetricsRepo,
		EntitlementService: opts.EntitlementService,
		AICreditsTracker:   opts.AICreditsTracker,
	}
}

// NewHandler creates a new handler instance with default dependencies.
// For testing or custom wiring, use NewHandlerWithDeps instead.
func NewHandler(repo database.Repository, wsHub *wsHub.Hub, log *logrus.Logger, allowAllOrigins bool, allowedOrigins []string) *Handler {
	deps := InitDefaultDeps(repo, wsHub, log)
	return NewHandlerWithDeps(repo, wsHub, log, allowAllOrigins, allowedOrigins, deps)
}

// NewHandlerWithDeps creates a handler with explicitly provided dependencies.
// This enables testing with mock dependencies and custom configurations.
func NewHandlerWithDeps(repo database.Repository, wsHub wsHub.HubInterface, log *logrus.Logger, allowAllOrigins bool, allowedOrigins []string, deps HandlerDeps) *Handler {
	allowedCopy := append([]string(nil), allowedOrigins...)

	// Initialize performance registry from config
	cfg := config.Load()
	perfRegistry := performance.NewCollectorRegistry(
		cfg.Performance.LogSummaryInterval, // targetFps - used as broadcast interval
		cfg.Performance.BufferSize,
	)

	seedCleanupManager := testgenie.NewSeedCleanupManager(
		deps.ExecutionService,
		nil,
		log,
		cfg.Execution.CompletionPollInterval,
		cfg.Execution.SeedCleanupTimeout,
	)

	handler := &Handler{
		catalogService:     deps.CatalogService,
		executionService:   deps.ExecutionService,
		workflowValidator:  deps.WorkflowValidator,
		repo:               repo,
		wsHub:              wsHub,
		storage:            deps.Storage,
		recordingService:   deps.RecordingService,
		recordModeService:  deps.RecordModeService,
		recordingsRoot:     deps.RecordingsRoot,
		replayRenderer:     deps.ReplayRenderer,
		sessionProfiles:    deps.SessionProfiles,
		log:                log,
		wsAllowAll:         allowAllOrigins,
		wsAllowedOrigins:   allowedCopy,
		upgrader:           websocket.Upgrader{},
		perfRegistry:       perfRegistry,
		seedCleanupManager: seedCleanupManager,
		entitlementService: deps.EntitlementService,
	}
	handler.upgrader.CheckOrigin = handler.isOriginAllowed

	// Initialize AI subhandlers with dependencies
	handler.domHandler = aihandlers.NewDOMHandler(log)
	handler.screenshotHandler = aihandlers.NewScreenshotHandler(log)
	handler.elementAnalysisHandler = aihandlers.NewElementAnalysisHandler(log)
	handler.aiAnalysisHandler = aihandlers.NewAIAnalysisHandler(log, handler.domHandler)

	// Initialize vision navigation handler with optional entitlement/credits
	visionNavOpts := []aihandlers.VisionNavigationHandlerOption{
		aihandlers.WithVisionNavigationHub(wsHub),
	}
	if deps.EntitlementService != nil {
		visionNavOpts = append(visionNavOpts, aihandlers.WithVisionNavigationEntitlement(deps.EntitlementService, deps.AICreditsTracker))
	}
	handler.visionNavigationHandler = aihandlers.NewVisionNavigationHandler(log, visionNavOpts...)

	return handler
}

func (h *Handler) isOriginAllowed(r *http.Request) bool {
	if h == nil {
		return false
	}
	if h.wsAllowAll {
		return true
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		// Non-browser clients may omit Origin; allow but log for audit trail
		if h.log != nil {
			h.log.Debug("Allowing websocket upgrade without Origin header")
		}
		return true
	}
	for _, allowed := range h.wsAllowedOrigins {
		if strings.EqualFold(allowed, origin) {
			return true
		}
	}
	if h.log != nil {
		h.log.WithField("origin", origin).Warn("Rejected websocket upgrade due to unauthorized origin")
	}
	return false
}

// GetExecutionService returns the workflow execution service for use by other components
// such as the scheduler service. Returns ExecutionService interface which satisfies
// scheduler.WorkflowExecutor.
func (h *Handler) GetExecutionService() workflow.ExecutionService {
	return h.executionService
}

// GetPerfRegistry returns the performance collector registry for use by debug endpoints.
func (h *Handler) GetPerfRegistry() *performance.CollectorRegistry {
	return h.perfRegistry
}

// AI handler delegation methods

// TakePreviewScreenshot delegates to the AI screenshot handler
func (h *Handler) TakePreviewScreenshot(w http.ResponseWriter, r *http.Request) {
	h.screenshotHandler.TakePreviewScreenshot(w, r)
}

// GetDOMTree delegates to the AI DOM handler
func (h *Handler) GetDOMTree(w http.ResponseWriter, r *http.Request) {
	h.domHandler.GetDOMTree(w, r)
}

// AnalyzeElements delegates to the AI element analysis handler
func (h *Handler) AnalyzeElements(w http.ResponseWriter, r *http.Request) {
	h.elementAnalysisHandler.AnalyzeElements(w, r)
}

// GetElementAtCoordinate delegates to the AI element analysis handler
func (h *Handler) GetElementAtCoordinate(w http.ResponseWriter, r *http.Request) {
	h.elementAnalysisHandler.GetElementAtCoordinate(w, r)
}

// AIAnalyzeElements delegates to the AI analysis handler
func (h *Handler) AIAnalyzeElements(w http.ResponseWriter, r *http.Request) {
	h.aiAnalysisHandler.AIAnalyzeElements(w, r)
}

// Vision Navigation delegation methods

// AINavigate delegates to the vision navigation handler
func (h *Handler) AINavigate(w http.ResponseWriter, r *http.Request) {
	h.visionNavigationHandler.HandleAINavigate(w, r)
}

// AINavigateCallback delegates to the vision navigation handler
func (h *Handler) AINavigateCallback(w http.ResponseWriter, r *http.Request) {
	h.visionNavigationHandler.HandleAINavigateCallback(w, r)
}

// AINavigateStatus delegates to the vision navigation handler
func (h *Handler) AINavigateStatus(w http.ResponseWriter, r *http.Request) {
	h.visionNavigationHandler.HandleAINavigateStatus(w, r)
}

// AINavigateAbort delegates to the vision navigation handler
func (h *Handler) AINavigateAbort(w http.ResponseWriter, r *http.Request) {
	h.visionNavigationHandler.HandleAINavigateAbort(w, r)
}

// AINavigateResume delegates to the vision navigation handler
func (h *Handler) AINavigateResume(w http.ResponseWriter, r *http.Request) {
	h.visionNavigationHandler.HandleAINavigateResume(w, r)
}
