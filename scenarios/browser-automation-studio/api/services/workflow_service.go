package services

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoexec "github.com/vrooli/browser-automation-studio/automation/executor"
	autorecorder "github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/database"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

const (
	workflowJSONStartMarker = "<WORKFLOW_JSON>"
	workflowJSONEndMarker   = "</WORKFLOW_JSON>"
	projectSyncCooldown     = 30 * time.Second
)

var ErrWorkflowVersionConflict = errors.New("workflow version conflict")
var ErrWorkflowVersionNotFound = errors.New("workflow version not found")
var ErrWorkflowRestoreProjectMismatch = errors.New("workflow does not belong to a project")
var ErrWorkflowNameConflict = errors.New("workflow name already exists in this project")

// WorkflowVersionSummary captures version metadata alongside high-level definition statistics so
// the UI can render history timelines without rehydrating full workflow payloads on every row.
type WorkflowVersionSummary struct {
	Version           int              `json:"version"`
	WorkflowID        uuid.UUID        `json:"workflow_id"`
	CreatedAt         time.Time        `json:"created_at"`
	CreatedBy         string           `json:"created_by"`
	ChangeDescription string           `json:"change_description"`
	DefinitionHash    string           `json:"definition_hash"`
	NodeCount         int              `json:"node_count"`
	EdgeCount         int              `json:"edge_count"`
	FlowDefinition    database.JSONMap `json:"flow_definition"`
}

// WorkflowService handles workflow business logic
type WorkflowService struct {
	repo             database.Repository
	wsHub            *wsHub.Hub
	log              *logrus.Logger
	aiClient         *OpenRouterClient
	executor         autoexec.Executor
	engineFactory    autoengine.Factory
	artifactRecorder autorecorder.Recorder
	syncLocks        sync.Map
	filePathCache    sync.Map
	executionCancels sync.Map
	projectSyncTimes sync.Map
}

// AIWorkflowError represents a structured error returned by the AI generator when
// it cannot produce a valid workflow definition for the given prompt.
type AIWorkflowError struct {
	Reason string
}

// Error implements the error interface.
func (e *AIWorkflowError) Error() string {
	return e.Reason
}

// WorkflowUpdateInput describes the mutable fields for a workflow save operation. The UI and CLI send both the
// JSON graph definition and an explicit nodes/edges payload; we keep both so agents can hand-edit the file without
// worrying about schema drift. ExpectedVersion enables optimistic locking so we do not clobber filesystem edits that
// were synchronized after the client loaded the workflow.
type WorkflowUpdateInput struct {
	Name              string
	Description       string
	FolderPath        string
	Tags              []string
	FlowDefinition    map[string]any
	Nodes             []any
	Edges             []any
	ChangeDescription string
	Source            string
	ExpectedVersion   *int
}

// ExecutionExportPreview summarises the export readiness state for an execution.
type ExecutionExportPreview struct {
	ExecutionID         uuid.UUID        `json:"execution_id"`
	SpecID              string           `json:"spec_id"`
	Status              string           `json:"status"`
	Message             string           `json:"message"`
	CapturedFrameCount  int              `json:"captured_frame_count"`
	AvailableAssetCount int              `json:"available_asset_count"`
	TotalDurationMs     int              `json:"total_duration_ms"`
	Package             *ReplayMovieSpec `json:"package,omitempty"`
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService(repo database.Repository, wsHub *wsHub.Hub, log *logrus.Logger) *WorkflowService {
	return NewWorkflowServiceWithDeps(repo, wsHub, log, WorkflowServiceOptions{})
}

// WorkflowServiceOptions allow injecting the refactored engine/executor/recorder
// stack without disrupting existing call sites.
type WorkflowServiceOptions struct {
	Executor         autoexec.Executor
	EngineFactory    autoengine.Factory
	ArtifactRecorder autorecorder.Recorder
}

// NewWorkflowServiceWithDeps allows advanced configuration for upcoming engine
// abstraction work while keeping the legacy constructor stable.
func NewWorkflowServiceWithDeps(repo database.Repository, wsHub *wsHub.Hub, log *logrus.Logger, opts WorkflowServiceOptions) *WorkflowService {
	svc := &WorkflowService{
		repo:             repo,
		wsHub:            wsHub,
		log:              log,
		aiClient:         NewOpenRouterClient(log),
		executor:         opts.Executor,
		engineFactory:    opts.EngineFactory,
		artifactRecorder: opts.ArtifactRecorder,
	}

	return svc
}

func cloneJSONMap(source database.JSONMap) database.JSONMap {
	if source == nil {
		return nil
	}
	clone := make(database.JSONMap, len(source))
	for k, v := range source {
		clone[k] = deepCloneInterface(v)
	}
	return clone
}

func deepCloneInterface(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for k, v := range typed {
			cloned[k] = deepCloneInterface(v)
		}
		return cloned
	case database.JSONMap:
		return cloneJSONMap(typed)
	case []any:
		result := make([]any, len(typed))
		for i := range typed {
			result[i] = deepCloneInterface(typed[i])
		}
		return result
	case pq.StringArray:
		return append(pq.StringArray{}, typed...)
	default:
		return typed
	}
}

func workflowDefinitionStats(def database.JSONMap) (nodeCount, edgeCount int) {
	if def == nil {
		return 0, 0
	}
	nodes := toInterfaceSlice(def["nodes"])
	edges := toInterfaceSlice(def["edges"])
	return len(nodes), len(edges)
}

var (
	previewDataKeys = map[string]struct{}{
		"previewscreenshot":           {},
		"previewscreenshotcapturedat": {},
		"previewscreenshotsourceurl":  {},
		"previewimage":                {},
	}
	previewNestedKeys = map[string]struct{}{
		"screenshot": {},
		"image":      {},
		"dataurl":    {},
		"data_url":   {},
		"thumbnail":  {},
	}
)

func stripPreviewData(value any) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		cleaned, modified := stripPreviewDataMap(typed)
		if !modified {
			return typed, false
		}
		return cleaned, true
	case database.JSONMap:
		cleaned, modified := stripPreviewDataMap(map[string]any(typed))
		if !modified {
			return typed, false
		}
		return database.JSONMap(cleaned), true
	default:
		return value, false
	}
}

func stripPreviewDataMap(data map[string]any) (map[string]any, bool) {
	if data == nil {
		return nil, false
	}
	modified := false
	cleaned := make(map[string]any, len(data))
	for key, value := range data {
		lowerKey := strings.ToLower(key)
		if _, skip := previewDataKeys[lowerKey]; skip {
			modified = true
			continue
		}
		if lowerKey == "preview" {
			if previewMap, ok := toStringAnyMap(value); ok {
				sanitizedPreview, previewModified := stripPreviewPreviewMap(previewMap)
				if previewModified {
					modified = true
				}
				if len(sanitizedPreview) == 0 {
					continue
				}
				cleaned[key] = sanitizedPreview
				continue
			}
		}
		cleaned[key] = value
	}
	if !modified {
		return data, false
	}
	return cleaned, true
}

func stripPreviewPreviewMap(data map[string]any) (map[string]any, bool) {
	if data == nil {
		return nil, false
	}
	modified := false
	cleaned := make(map[string]any, len(data))
	for key, value := range data {
		lowerKey := strings.ToLower(key)
		if _, skip := previewNestedKeys[lowerKey]; skip {
			if str, ok := value.(string); ok {
				trimmed := strings.TrimSpace(strings.ToLower(str))
				if strings.HasPrefix(trimmed, "data:image/") {
					modified = true
					continue
				}
			}
		}
		cleaned[key] = value
	}
	if !modified {
		return data, false
	}
	return cleaned, true
}

func toStringAnyMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case database.JSONMap:
		return map[string]any(typed), true
	default:
		return nil, false
	}
}

func sanitizeWorkflowDefinition(def database.JSONMap) database.JSONMap {
	if def == nil {
		return nil
	}
	nodes := toInterfaceSlice(def["nodes"])
	modified := false
	for i, rawNode := range nodes {
		if sanitized, changed := sanitizeWorkflowNode(rawNode); changed {
			nodes[i] = sanitized
			modified = true
		}
	}
	if modified {
		def["nodes"] = nodes
	}
	return def
}

func sanitizeWorkflowNode(raw any) (any, bool) {
	switch typed := raw.(type) {
	case map[string]any:
		if data, ok := typed["data"]; ok {
			if cleaned, changed := stripPreviewData(data); changed {
				typed["data"] = cleaned
				return typed, true
			}
		}
	case database.JSONMap:
		nodeMap := map[string]any(typed)
		if data, ok := nodeMap["data"]; ok {
			if cleaned, changed := stripPreviewData(data); changed {
				nodeMap["data"] = cleaned
				return database.JSONMap(nodeMap), true
			}
		}
	}
	return raw, false
}

// CheckHealth checks the health of all dependencies
// Workflow methods

// CreateWorkflow creates a new workflow without a project. This now delegates to CreateWorkflowWithProject and will
// return an error because workflows must belong to a project to ensure filesystem synchronization.
