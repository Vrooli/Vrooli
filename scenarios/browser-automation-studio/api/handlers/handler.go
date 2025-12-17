package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	aihandlers "github.com/vrooli/browser-automation-studio/handlers/ai"
	"github.com/vrooli/browser-automation-studio/internal/paths"
	"github.com/vrooli/browser-automation-studio/performance"
	archiveingestion "github.com/vrooli/browser-automation-studio/services/archive-ingestion"
	"github.com/vrooli/browser-automation-studio/services/export"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
	"github.com/vrooli/browser-automation-studio/services/replay"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics"
	uxcollector "github.com/vrooli/browser-automation-studio/services/uxmetrics/collector"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
)

// Handler contains all HTTP handlers
type replayRenderer interface {
	Render(ctx context.Context, spec *export.ReplayMovieSpec, format replay.RenderFormat, filename string) (*replay.RenderedMedia, error)
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
	recordModeService *livecapture.Service // Live recording session management
	recordingsRoot    string
	replayRenderer  replayRenderer
	sessionProfiles *archiveingestion.SessionProfileStore
	log             *logrus.Logger
	upgrader          websocket.Upgrader
	wsAllowAll        bool
	wsAllowedOrigins  []string

	// Performance monitoring
	perfRegistry *performance.CollectorRegistry

	// AI subhandlers
	screenshotHandler      *aihandlers.ScreenshotHandler
	domHandler             *aihandlers.DOMHandler
	elementAnalysisHandler *aihandlers.ElementAnalysisHandler
	aiAnalysisHandler      *aihandlers.AIAnalysisHandler
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

// HealthResponse represents the health check response following Vrooli standards
type HealthResponse struct {
	Status       string         `json:"status"`
	Service      string         `json:"service"`
	Timestamp    string         `json:"timestamp"`
	Readiness    bool           `json:"readiness"`
	Version      string         `json:"version,omitempty"`
	Dependencies map[string]any `json:"dependencies,omitempty"`
	Metrics      map[string]any `json:"metrics,omitempty"`
}

// HandlerDeps holds all dependencies for the Handler.
// This struct separates dependency wiring from handler construction.
type HandlerDeps struct {
	// Service interfaces - now using proper interface types
	CatalogService   workflow.CatalogService   // Workflow/project CRUD, versioning, sync
	ExecutionService workflow.ExecutionService // Execution lifecycle, timeline, export

	WorkflowValidator *workflowvalidator.Validator
	Storage           storage.StorageInterface
	RecordingService  archiveingestion.IngestionServiceInterface
	RecordModeService *livecapture.Service // Live recording session management
	RecordingsRoot    string
	ReplayRenderer    replayRenderer
	SessionProfiles   *archiveingestion.SessionProfileStore
	UXMetricsRepo     uxmetrics.Repository // Optional: enables UX metrics collection
}

// InitDefaultDeps initializes the standard production dependencies.
// This function is responsible for infrastructure wiring, keeping it separate
// from handler construction for clearer responsibility boundaries.
func InitDefaultDeps(repo database.Repository, wsHub *wsHub.Hub, log *logrus.Logger) HandlerDeps {
	return InitDefaultDepsWithUXMetrics(repo, wsHub, log, nil)
}

// InitDefaultDepsWithUXMetrics initializes dependencies with optional UX metrics collection.
// When uxRepo is provided, the UX metrics collector wraps the event sink to passively
// capture interaction data during workflow executions.
func InitDefaultDepsWithUXMetrics(repo database.Repository, wsHub *wsHub.Hub, log *logrus.Logger, uxRepo uxmetrics.Repository) HandlerDeps {
	// Initialize screenshot storage (defaults to local filesystem, optionally MinIO)
	screenshotRoot := paths.ResolveScreenshotsRoot(log)
	storageClient := storage.NewScreenshotStorage(log, screenshotRoot)

	// Initialize recordings infrastructure
	recordingsRoot := paths.ResolveRecordingsRoot(log)
	recordingService := archiveingestion.NewIngestionService(repo, storageClient, wsHub, log, recordingsRoot)
	sessionProfiles := archiveingestion.NewSessionProfileStore(paths.ResolveSessionProfilesRoot(log), log)

	// Wire automation stack
	autoExecutor := autoexecutor.NewSimpleExecutor(nil)
	autoEngineFactory, engErr := autoengine.DefaultFactory(log)
	if engErr != nil {
		log.WithError(engErr).Warn("Failed to initialize automation engine; automation executor will be disabled")
	}
	// Persist execution artifacts under recordingsRoot so file-truth execution data is durable and discoverable.
	autoRecorder := executionwriter.NewFileRecorder(repo, storageClient, log, recordingsRoot)

	// Configure event sink factory - optionally wrap with UX metrics collector
	var eventSinkFactory func() autoevents.Sink
	if uxRepo != nil {
		// Create UX metrics collector that wraps the WebSocket sink
		// The collector passively captures interaction data while delegating events to the hub
		eventSinkFactory = func() autoevents.Sink {
			baseSink := autoevents.NewWSHubSink(wsHub, log, eventBufferLimits())
			return uxcollector.NewCollector(baseSink, uxRepo)
		}
		log.Debug("UX metrics collector enabled in event pipeline")
	}

	// Create workflow service with dependencies
	workflowSvc := workflow.NewWorkflowServiceWithDeps(repo, wsHub, log, workflow.WorkflowServiceOptions{
		Executor:         autoExecutor,
		EngineFactory:    autoEngineFactory,
		ArtifactRecorder: autoRecorder,
		EventSinkFactory: eventSinkFactory,
		ExecutionDataRoot: recordingsRoot,
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
		CatalogService:    workflowSvc,
		ExecutionService:  workflowSvc,
		WorkflowValidator: validatorInstance,
		Storage:           storageClient,
		RecordingService:  recordingService,
		RecordModeService: recordModeSvc,
		RecordingsRoot:    recordingsRoot,
		ReplayRenderer:    replay.NewReplayRenderer(log, recordingsRoot),
		SessionProfiles:   sessionProfiles,
		UXMetricsRepo:     uxRepo,
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

	handler := &Handler{
		catalogService:    deps.CatalogService,
		executionService:  deps.ExecutionService,
		workflowValidator: deps.WorkflowValidator,
		repo:              repo,
		wsHub:             wsHub,
		storage:           deps.Storage,
		recordingService:  deps.RecordingService,
		recordModeService: deps.RecordModeService,
		recordingsRoot:  deps.RecordingsRoot,
		replayRenderer:  deps.ReplayRenderer,
		sessionProfiles: deps.SessionProfiles,
		log:             log,
		wsAllowAll:        allowAllOrigins,
		wsAllowedOrigins:  allowedCopy,
		upgrader:          websocket.Upgrader{},
		perfRegistry:      perfRegistry,
	}
	handler.upgrader.CheckOrigin = handler.isOriginAllowed

	// Initialize AI subhandlers with dependencies
	handler.domHandler = aihandlers.NewDOMHandler(log)
	handler.screenshotHandler = aihandlers.NewScreenshotHandler(log)
	handler.elementAnalysisHandler = aihandlers.NewElementAnalysisHandler(log)
	handler.aiAnalysisHandler = aihandlers.NewAIAnalysisHandler(log, handler.domHandler)

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
