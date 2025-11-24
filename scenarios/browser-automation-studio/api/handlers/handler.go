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
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	autorecorder "github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/handlers/ai"
	"github.com/vrooli/browser-automation-studio/internal/paths"
	"github.com/vrooli/browser-automation-studio/services"
	"github.com/vrooli/browser-automation-studio/storage"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
)

type WorkflowService interface {
	CreateWorkflowWithProject(ctx context.Context, projectID *uuid.UUID, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error)
	ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error)
	GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error)
	UpdateWorkflow(ctx context.Context, workflowID uuid.UUID, input services.WorkflowUpdateInput) (*database.Workflow, error)
	ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*services.WorkflowVersionSummary, error)
	GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*services.WorkflowVersionSummary, error)
	RestoreWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int, changeDescription string) (*database.Workflow, error)
	ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error)
	ExecuteAdhocWorkflow(ctx context.Context, flowDefinition map[string]any, parameters map[string]any, name string) (*database.Execution, error)
	ModifyWorkflow(ctx context.Context, workflowID uuid.UUID, prompt string, currentFlow map[string]any) (*database.Workflow, error)
	GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error)
	GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*services.ExecutionTimeline, error)
	DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*services.ExecutionExportPreview, error)
	ExportToFolder(ctx context.Context, executionID uuid.UUID, outputDir string, storageClient storage.StorageInterface) error
	GetExecution(ctx context.Context, executionID uuid.UUID) (*database.Execution, error)
	ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error)
	StopExecution(ctx context.Context, executionID uuid.UUID) error
	GetProjectByName(ctx context.Context, name string) (*database.Project, error)
	GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error)
	CreateProject(ctx context.Context, project *database.Project) error
	ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error)
	GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error)
	GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]map[string]any, error)
	GetProject(ctx context.Context, projectID uuid.UUID) (*database.Project, error)
	UpdateProject(ctx context.Context, project *database.Project) error
	DeleteProject(ctx context.Context, projectID uuid.UUID) error
	ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error)
	DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error
	CheckAutomationHealth(ctx context.Context) (bool, error)
}

// Handler contains all HTTP handlers
type replayRenderer interface {
	Render(ctx context.Context, spec *services.ReplayMovieSpec, format services.RenderFormat, filename string) (*services.RenderedMedia, error)
}

// Handler contains all HTTP handlers
type Handler struct {
	workflowService   WorkflowService
	workflowValidator *workflowvalidator.Validator
	repo              database.Repository
	wsHub             wsHub.HubInterface
	storage           storage.StorageInterface
	recordingService  services.RecordingServiceInterface
	recordingsRoot    string
	replayRenderer    replayRenderer
	log               *logrus.Logger
	upgrader          websocket.Upgrader
	wsAllowAll        bool
	wsAllowedOrigins  []string

	// AI subhandlers
	screenshotHandler      *ai.ScreenshotHandler
	domHandler             *ai.DOMHandler
	elementAnalysisHandler *ai.ElementAnalysisHandler
	aiAnalysisHandler      *ai.AIAnalysisHandler
}

const (
	recordingUploadLimitBytes = 200 * 1024 * 1024 // 200MB
	recordingImportTimeout    = 2 * time.Minute
)

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

// NewHandler creates a new handler instance
func NewHandler(repo database.Repository, wsHub *wsHub.Hub, log *logrus.Logger, allowAllOrigins bool, allowedOrigins []string) *Handler {
	// Initialize MinIO client for screenshot serving
	storageClient, err := storage.NewMinIOClient(log)
	if err != nil {
		log.WithError(err).Warn("Failed to initialize MinIO client for handlers - screenshot serving will be disabled")
	}

	recordingsRoot := paths.ResolveRecordingsRoot(log)
	recordingService := services.NewRecordingService(repo, storageClient, wsHub, log, recordingsRoot)
	// Wire automation stack (feature-flag protected inside service).
	autoExecutor := autoexecutor.NewSimpleExecutor(nil)
	autoEngineFactory, engErr := autoengine.DefaultFactory(log)
	if engErr != nil {
		log.WithError(engErr).Warn("Failed to initialize automation engine; automation executor will be disabled")
	}
	autoRecorder := autorecorder.NewDBRecorder(repo, storageClient, log)

	workflowSvc := services.NewWorkflowServiceWithDeps(repo, wsHub, log, services.WorkflowServiceOptions{
		Executor:         autoExecutor,
		EngineFactory:    autoEngineFactory,
		ArtifactRecorder: autoRecorder,
	})
	replayRenderer := services.NewReplayRenderer(log, recordingsRoot)

	allowedCopy := append([]string(nil), allowedOrigins...)

	validatorInstance, err := workflowvalidator.NewValidator()
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize workflow validator")
	}

	handler := &Handler{
		workflowService:   workflowSvc,
		workflowValidator: validatorInstance,
		repo:              repo,
		wsHub:             wsHub,
		storage:           storageClient,
		recordingService:  recordingService,
		recordingsRoot:    recordingsRoot,
		replayRenderer:    replayRenderer,
		log:               log,
		wsAllowAll:        allowAllOrigins,
		wsAllowedOrigins:  allowedCopy,
		upgrader:          websocket.Upgrader{},
	}
	handler.upgrader.CheckOrigin = handler.isOriginAllowed

	// Initialize AI subhandlers with dependencies
	handler.domHandler = ai.NewDOMHandler(log)
	handler.screenshotHandler = ai.NewScreenshotHandler(log)
	handler.elementAnalysisHandler = ai.NewElementAnalysisHandler(log)
	handler.aiAnalysisHandler = ai.NewAIAnalysisHandler(log, handler.domHandler)

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
