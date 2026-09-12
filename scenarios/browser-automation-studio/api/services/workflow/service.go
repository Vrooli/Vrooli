package workflow

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	autoexec "github.com/vrooli/browser-automation-studio/automation/executor"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/ai"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/readiness"
	sessionprofile "github.com/vrooli/browser-automation-studio/services/session-profile"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

const (
	workflowJSONStartMarker = "<WORKFLOW_JSON>"
	workflowJSONEndMarker   = "</WORKFLOW_JSON>"
	projectSyncCooldown     = 30 * time.Second
)

// Type alias for ReplayMovieSpec from export package
type ReplayMovieSpec = export.ReplayMovieSpec

var (
	ErrWorkflowVersionConflict        = errors.New("workflow version conflict")
	ErrWorkflowVersionNotFound        = errors.New("workflow version not found")
	ErrWorkflowRestoreProjectMismatch = errors.New("workflow does not belong to a project")
	ErrWorkflowNameConflict           = errors.New("workflow name already exists in this project")
	ErrWorkflowCaseExpectationMissing = errors.New("case workflows must include at least one assertion node or an expected outcome")
)

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
	repo                  database.Repository
	log                   *logrus.Logger
	aiClient              ai.AIClient
	executor              autoexec.Executor
	engineFactory         autoengine.Factory
	artifactRecorder      executionwriter.ExecutionWriter
	planCompiler          autoexec.PlanCompiler
	eventSinkFactory      func() autoevents.Sink
	executionDataRoot     string
	projectRoot           func(context.Context) (string, error)
	sessionProfileService *sessionprofile.Service
	syncLocks             sync.Map
	filePathCache         sync.Map
	executionCancels      sync.Map
	projectSyncTimes      sync.Map

	// readinessResolver settles a run's opening navigation on the target
	// scenario's declared surfaces instead of on navigation timing alone. It is
	// optional: nil keeps every run on generic navigation.
	readinessResolver readiness.Resolver
}

// SetReadinessResolver installs the declared-readiness resolver. Wiring is
// separate from construction so a run works with or without Experience Manager,
// which is the same posture the capture handler takes.
func (s *WorkflowService) SetReadinessResolver(resolver readiness.Resolver) {
	s.readinessResolver = resolver
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
	ArtifactRecorder executionwriter.ExecutionWriter
	PlanCompiler     autoexec.PlanCompiler
	AIClient         ai.AIClient
	EventSinkFactory func() autoevents.Sink
	// ExecutionDataRoot controls where execution artifacts and proto snapshots are persisted.
	// When empty, defaults to "/tmp/bas-executions" for backward compatibility with earlier recorder defaults.
	ExecutionDataRoot string
	// ProjectRoot resolves the base directory for project-backed workflow files.
	// It is request-aware so routed validation can lease an isolated filesystem.
	ProjectRoot func(context.Context) (string, error)
	// SessionProfileService provides access to session profiles for authenticated execution.
	// When set, workflows can use session_profile_id to inject storage state (cookies, localStorage).
	SessionProfileService *sessionprofile.Service
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
		limits := eventBufferLimits()
		eventSinkFactory = func() autoevents.Sink {
			return autoevents.NewWSHubSink(wsHub, log, limits)
		}
	}

	svc := &WorkflowService{
		repo:                  repo,
		log:                   log,
		aiClient:              aiClient,
		executor:              opts.Executor,
		engineFactory:         opts.EngineFactory,
		artifactRecorder:      opts.ArtifactRecorder,
		planCompiler:          opts.PlanCompiler,
		eventSinkFactory:      eventSinkFactory,
		executionDataRoot:     strings.TrimSpace(opts.ExecutionDataRoot),
		projectRoot:           opts.ProjectRoot,
		sessionProfileService: opts.SessionProfileService,
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
	limits := eventBufferLimits()
	if s.eventSinkFactory != nil {
		return s.eventSinkFactory()
	}
	return autoevents.NewWSHubSink(nil, s.log, limits)
}

// eventBufferLimits returns validated event buffer limits sourced from config.
// Delegates to config.EventBufferLimitsFromConfig() for centralized configuration.
func eventBufferLimits() autocontracts.EventBufferLimits {
	return config.EventBufferLimitsFromConfig()
}

// CheckHealth checks the health of all dependencies
// Workflow methods

// CreateWorkflow creates a new workflow without a project. This now delegates to CreateWorkflowWithProject and will
// return an error because workflows must belong to a project to ensure filesystem synchronization.
