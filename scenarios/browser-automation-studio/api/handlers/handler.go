package handlers

import (
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
	"github.com/vrooli/browser-automation-studio/services"
	"github.com/vrooli/browser-automation-studio/storage"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// Handler contains all HTTP handlers
type Handler struct {
	workflowService  *services.WorkflowService
	repo             database.Repository
	browserless      *browserless.Client
	wsHub            *wsHub.Hub
	storage          *storage.MinIOClient
	recordingService *services.RecordingService
	recordingsRoot   string
	log              *logrus.Logger
	upgrader         websocket.Upgrader
}

const (
	recordingUploadLimitBytes = 200 * 1024 * 1024 // 200MB
	recordingImportTimeout    = 2 * 60 * 1000      // 2 minutes in milliseconds
)

type workflowResponse struct {
	*database.Workflow
	WorkflowID uuid.UUID     `json:"workflow_id"`
	Nodes      []interface{} `json:"nodes"`
	Edges      []interface{} `json:"edges"`
}

// HealthResponse represents the health check response following Vrooli standards
type HealthResponse struct {
	Status       string                 `json:"status"`
	Service      string                 `json:"service"`
	Timestamp    string                 `json:"timestamp"`
	Readiness    bool                   `json:"readiness"`
	Version      string                 `json:"version,omitempty"`
	Dependencies map[string]interface{} `json:"dependencies,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
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

	return &Handler{
		workflowService:  services.NewWorkflowService(repo, browserless, wsHub, log),
		repo:             repo,
		browserless:      browserless,
		wsHub:            wsHub,
		storage:          storageClient,
		recordingService: recordingService,
		recordingsRoot:   recordingsRoot,
		log:              log,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for now - in production, validate origins properly
				return true
			},
		},
	}
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

func normalizeWorkflowFlowDefinition(workflow *database.Workflow) ([]interface{}, []interface{}) {
	if workflow == nil {
		return []interface{}{}, []interface{}{}
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

func toInterfaceSlice(value interface{}) []interface{} {
	switch v := value.(type) {
	case nil:
		return []interface{}{}
	case []interface{}:
		return v
	case []map[string]interface{}:
		result := make([]interface{}, len(v))
		for i := range v {
			result[i] = v[i]
		}
		return result
	case []database.JSONMap:
		result := make([]interface{}, len(v))
		for i := range v {
			result[i] = v[i]
		}
		return result
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			return []interface{}{}
		}
		var result []interface{}
		if err := json.Unmarshal(bytes, &result); err != nil {
			return []interface{}{}
		}
		return result
	}
}
