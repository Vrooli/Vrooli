package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	autorecorder "github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	aihandlers "github.com/vrooli/browser-automation-studio/handlers/ai"
	"github.com/vrooli/browser-automation-studio/internal/paths"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/recording"
	"github.com/vrooli/browser-automation-studio/services/replay"
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
	workflowCatalog   workflow.CatalogService
	executionService  workflow.ExecutionService
	exportService     workflow.ExportService
	workflowValidator *workflowvalidator.Validator
	repo              database.Repository
	wsHub             wsHub.HubInterface
	storage           storage.StorageInterface
	recordingService  recording.RecordingServiceInterface
	recordingsRoot    string
	replayRenderer    replayRenderer
	sessionProfiles   *recording.SessionProfileStore
	activeSessions    map[string]string // sessionID -> profileID
	activeSessionsMu  sync.Mutex
	log               *logrus.Logger
	upgrader          websocket.Upgrader
	wsAllowAll        bool
	wsAllowedOrigins  []string

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

type workflowResponse struct {
	*database.Workflow
	WorkflowID uuid.UUID `json:"workflow_id"`
	Nodes      []any     `json:"nodes"`
	Edges      []any     `json:"edges"`
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
	WorkflowCatalog   workflow.CatalogService
	ExecutionService  workflow.ExecutionService
	ExportService     workflow.ExportService
	WorkflowValidator *workflowvalidator.Validator
	Storage           storage.StorageInterface
	RecordingService  recording.RecordingServiceInterface
	RecordingsRoot    string
	ReplayRenderer    replayRenderer
	SessionProfiles   *recording.SessionProfileStore
}

// InitDefaultDeps initializes the standard production dependencies.
// This function is responsible for infrastructure wiring, keeping it separate
// from handler construction for clearer responsibility boundaries.
func InitDefaultDeps(repo database.Repository, wsHub *wsHub.Hub, log *logrus.Logger) HandlerDeps {
	// Initialize screenshot storage (defaults to local filesystem, optionally MinIO)
	screenshotRoot := paths.ResolveScreenshotsRoot(log)
	storageClient := storage.NewScreenshotStorage(log, screenshotRoot)

	// Initialize recordings infrastructure
	recordingsRoot := paths.ResolveRecordingsRoot(log)
	recordingService := recording.NewRecordingService(repo, storageClient, wsHub, log, recordingsRoot)
	sessionProfiles := recording.NewSessionProfileStore(paths.ResolveSessionProfilesRoot(log), log)

	// Wire automation stack
	autoExecutor := autoexecutor.NewSimpleExecutor(nil)
	autoEngineFactory, engErr := autoengine.DefaultFactory(log)
	if engErr != nil {
		log.WithError(engErr).Warn("Failed to initialize automation engine; automation executor will be disabled")
	}
	autoRecorder := autorecorder.NewDBRecorder(repo, storageClient, log)

	// Create workflow service with dependencies
	workflowSvc := workflow.NewWorkflowServiceWithDeps(repo, wsHub, log, workflow.WorkflowServiceOptions{
		Executor:         autoExecutor,
		EngineFactory:    autoEngineFactory,
		ArtifactRecorder: autoRecorder,
	})

	// Create validator
	validatorInstance, err := workflowvalidator.NewValidator()
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize workflow validator")
	}

	return HandlerDeps{
		WorkflowCatalog:   workflowSvc,
		ExecutionService:  workflowSvc,
		ExportService:     workflowSvc,
		WorkflowValidator: validatorInstance,
		Storage:           storageClient,
		RecordingService:  recordingService,
		RecordingsRoot:    recordingsRoot,
		ReplayRenderer:    replay.NewReplayRenderer(log, recordingsRoot),
		SessionProfiles:   sessionProfiles,
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

	handler := &Handler{
		workflowCatalog:   deps.WorkflowCatalog,
		executionService:  deps.ExecutionService,
		exportService:     deps.ExportService,
		workflowValidator: deps.WorkflowValidator,
		repo:              repo,
		wsHub:             wsHub,
		storage:           deps.Storage,
		recordingService:  deps.RecordingService,
		recordingsRoot:    deps.RecordingsRoot,
		replayRenderer:    deps.ReplayRenderer,
		sessionProfiles:   deps.SessionProfiles,
		activeSessions:    make(map[string]string),
		log:               log,
		wsAllowAll:        allowAllOrigins,
		wsAllowedOrigins:  allowedCopy,
		upgrader:          websocket.Upgrader{},
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
// such as the scheduler service.
func (h *Handler) GetExecutionService() workflow.ExecutionService {
	return h.executionService
}

func newWorkflowResponse(workflow *database.Workflow) workflowResponse {
	nodes, edges := normalizeWorkflowFlowDefinition(workflow)
	if workflow == nil {
		return workflowResponse{
			Workflow:   &database.Workflow{},
			WorkflowID: uuid.Nil,
			Nodes:      nodes,
			Edges:      edges,
		}
	}

	return workflowResponse{
		Workflow:   workflow,
		WorkflowID: workflow.ID,
		Nodes:      nodes,
		Edges:      edges,
	}
}

func normalizeWorkflowFlowDefinition(workflow *database.Workflow) ([]any, []any) {
	if workflow == nil {
		return []any{}, []any{}
	}

	if workflow.FlowDefinition == nil {
		workflow.FlowDefinition = database.JSONMap{}
	}

	nodes := toInterfaceSlice(workflow.FlowDefinition["nodes"])
	edges := toInterfaceSlice(workflow.FlowDefinition["edges"])

	workflow.FlowDefinition["nodes"] = nodes
	workflow.FlowDefinition["edges"] = edges

	return nodes, edges
}

func toInterfaceSlice(value any) []any {
	switch v := value.(type) {
	case nil:
		return []any{}
	case []any:
		return v
	case []map[string]any:
		result := make([]any, len(v))
		for i := range v {
			result[i] = v[i]
		}
		return result
	case []database.JSONMap:
		result := make([]any, len(v))
		for i := range v {
			result[i] = v[i]
		}
		return result
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			return []any{}
		}
		var result []any
		if err := json.Unmarshal(bytes, &result); err != nil {
			return []any{}
		}
		return result
	}
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
