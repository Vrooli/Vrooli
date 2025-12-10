package workflow

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	autoexec "github.com/vrooli/browser-automation-studio/automation/executor"
	autorecorder "github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"github.com/vrooli/browser-automation-studio/services/ai"
	"github.com/vrooli/browser-automation-studio/services/export"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

const (
	workflowJSONStartMarker = "<WORKFLOW_JSON>"
	workflowJSONEndMarker   = "</WORKFLOW_JSON>"
	projectSyncCooldown     = 30 * time.Second
)

// Type alias for ReplayMovieSpec from export package
type ReplayMovieSpec = export.ReplayMovieSpec

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
	log              *logrus.Logger
	aiClient         ai.AIClient
	executor         autoexec.Executor
	engineFactory    autoengine.Factory
	artifactRecorder autorecorder.Recorder
	planCompiler     autoexec.PlanCompiler
	eventSinkFactory func() autoevents.Sink
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
func NewWorkflowService(repo database.Repository, wsHub wsHub.HubInterface, log *logrus.Logger) *WorkflowService {
	return NewWorkflowServiceWithDeps(repo, wsHub, log, WorkflowServiceOptions{})
}

// WorkflowServiceOptions allow injecting the refactored engine/executor/recorder
// stack without disrupting existing call sites.
type WorkflowServiceOptions struct {
	Executor         autoexec.Executor
	EngineFactory    autoengine.Factory
	ArtifactRecorder autorecorder.Recorder
	PlanCompiler     autoexec.PlanCompiler
	AIClient         ai.AIClient
	EventSinkFactory func() autoevents.Sink
}

// NewWorkflowServiceWithDeps allows advanced configuration for upcoming engine
// abstraction work while keeping the legacy constructor stable.
func NewWorkflowServiceWithDeps(repo database.Repository, wsHub wsHub.HubInterface, log *logrus.Logger, opts WorkflowServiceOptions) *WorkflowService {
	aiClient := opts.AIClient
	if aiClient == nil {
		aiClient = ai.NewOpenRouterClient(log)
	}

	eventSinkFactory := opts.EventSinkFactory
	if eventSinkFactory == nil {
		eventSinkFactory = func() autoevents.Sink {
			return autoevents.NewWSHubSink(wsHub, log, autocontracts.DefaultEventBufferLimits)
		}
	}

	svc := &WorkflowService{
		repo:             repo,
		log:              log,
		aiClient:         aiClient,
		executor:         opts.Executor,
		engineFactory:    opts.EngineFactory,
		artifactRecorder: opts.ArtifactRecorder,
		planCompiler:     opts.PlanCompiler,
		eventSinkFactory: eventSinkFactory,
	}

	return svc
}

// newEventSink constructs an event sink for automation lifecycle notifications.
// The sink creation is intentionally deferred so orchestration code remains
// agnostic to websocket implementation details.
func (s *WorkflowService) newEventSink() autoevents.Sink {
	if s == nil {
		return nil
	}
	if s.eventSinkFactory != nil {
		return s.eventSinkFactory()
	}
	return autoevents.NewWSHubSink(nil, s.log, autocontracts.DefaultEventBufferLimits)
}

var (
	_ CatalogService   = (*WorkflowService)(nil)
	_ ExecutionService = (*WorkflowService)(nil)
	_ ExportService    = (*WorkflowService)(nil)
)

// cloneJSONMap creates a deep copy of a database.JSONMap using typeconv utilities.
// Handles database.JSONMap and database.StringArray as special domain-specific types.
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

// deepCloneInterface delegates to typeconv.DeepCloneValue with additional handling
// for domain-specific types (database.JSONMap, database.StringArray).
func deepCloneInterface(value any) any {
	switch typed := value.(type) {
	case database.JSONMap:
		return cloneJSONMap(typed)
	case database.StringArray:
		return append(database.StringArray{}, typed...)
	default:
		return typeconv.DeepCloneValue(typed)
	}
}

func workflowDefinitionStats(def database.JSONMap) (nodeCount, edgeCount int) {
	if def == nil {
		return 0, 0
	}
	nodes := ToInterfaceSlice(def["nodes"])
	edges := ToInterfaceSlice(def["edges"])
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
	nodes := ToInterfaceSlice(def["nodes"])
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
