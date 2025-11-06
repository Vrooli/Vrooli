package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/handlers/ai"
	"github.com/vrooli/browser-automation-studio/services"
	"github.com/vrooli/browser-automation-studio/storage"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
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
	GetExecution(ctx context.Context, executionID uuid.UUID) (*database.Execution, error)
	ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error)
	StopExecution(ctx context.Context, executionID uuid.UUID) error
	GetProjectByName(ctx context.Context, name string) (*database.Project, error)
	GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error)
	CreateProject(ctx context.Context, project *database.Project) error
	ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error)
	GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error)
	GetProject(ctx context.Context, projectID uuid.UUID) (*database.Project, error)
	UpdateProject(ctx context.Context, project *database.Project) error
	DeleteProject(ctx context.Context, projectID uuid.UUID) error
	ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error)
	DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error
}

// Handler contains all HTTP handlers
type replayRenderer interface {
	Render(ctx context.Context, spec *services.ReplayMovieSpec, format services.RenderFormat, filename string) (*services.RenderedMedia, error)
}

// Handler contains all HTTP handlers
type Handler struct {
	workflowService  WorkflowService
	repo             database.Repository
	browserless      *browserless.Client
	wsHub            wsHub.HubInterface
	storage          storage.StorageInterface
	recordingService services.RecordingServiceInterface
	recordingsRoot   string
	replayRenderer   replayRenderer
	log              *logrus.Logger
	upgrader         websocket.Upgrader

	// AI subhandlers
	screenshotHandler      *ai.ScreenshotHandler
	domHandler             *ai.DOMHandler
	elementAnalysisHandler *ai.ElementAnalysisHandler
	aiAnalysisHandler      *ai.AIAnalysisHandler
}

const (
	recordingUploadLimitBytes = 200 * 1024 * 1024 // 200MB
	recordingImportTimeout    = 2 * 60 * 1000     // 2 minutes in milliseconds
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
func NewHandler(repo database.Repository, browserless *browserless.Client, wsHub *wsHub.Hub, log *logrus.Logger) *Handler {
	// Initialize MinIO client for screenshot serving
	storageClient, err := storage.NewMinIOClient(log)
	if err != nil {
		log.WithError(err).Warn("Failed to initialize MinIO client for handlers - screenshot serving will be disabled")
	}

	recordingsRoot := resolveRecordingsRoot(log)
	recordingService := services.NewRecordingService(repo, storageClient, wsHub, log, recordingsRoot)
	workflowSvc := services.NewWorkflowService(repo, browserless, wsHub, log)
	replayRenderer := services.NewReplayRenderer(log, recordingsRoot)

	handler := &Handler{
		workflowService:  workflowSvc,
		repo:             repo,
		browserless:      browserless,
		wsHub:            wsHub,
		storage:          storageClient,
		recordingService: recordingService,
		recordingsRoot:   recordingsRoot,
		replayRenderer:   replayRenderer,
		log:              log,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for now - in production, validate origins properly
				return true
			},
		},
	}

	// Initialize AI subhandlers with dependencies
	handler.domHandler = ai.NewDOMHandler(log)
	handler.screenshotHandler = ai.NewScreenshotHandler(log)
	handler.elementAnalysisHandler = ai.NewElementAnalysisHandler(log)
	handler.aiAnalysisHandler = ai.NewAIAnalysisHandler(log, handler.domHandler)

	return handler
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

func resolveRecordingsRoot(log *logrus.Logger) string {
	if value := strings.TrimSpace(os.Getenv("BAS_RECORDINGS_ROOT")); value != "" {
		if abs, err := filepath.Abs(value); err == nil {
			return abs
		}
		if log != nil {
			log.WithField("path", value).Warn("Using BAS_RECORDINGS_ROOT without normalization")
		}
		return value
	}

	cwd, err := os.Getwd()
	if err != nil {
		if log != nil {
			log.WithError(err).Warn("Failed to resolve working directory for recordings root; using relative default")
		}
		return filepath.Join("scenarios", "browser-automation-studio", "data", "recordings")
	}

	root := filepath.Join(cwd, "scenarios", "browser-automation-studio", "data", "recordings")
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	return root
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
