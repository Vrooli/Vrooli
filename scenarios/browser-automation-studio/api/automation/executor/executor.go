package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/automation/events"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/config"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

// Request wires together the dependencies required to run a compiled plan.
type Request struct {
	Plan              contracts.ExecutionPlan
	EngineName        string
	EngineFactory     engine.Factory
	Recorder          executionwriter.ExecutionWriter
	EventSink         events.Sink
	HeartbeatInterval time.Duration
	ReuseMode         engine.SessionReuseMode
	WorkflowResolver  WorkflowResolver // Required for subflow resolution.
	PlanCompiler      PlanCompiler     // Optional; defaults to engine-registered compiler.
	MaxSubflowDepth   int              // Optional; defaults to 5.
	SubflowStack      []uuid.UUID      // Internal: call stack to avoid recursion.
	EngineCaps        *contracts.EngineCapabilities

	// Resume support: when set, execution starts from the step after StartFromStepIndex.
	// InitialVariables provides state accumulated from previously completed steps.
	StartFromStepIndex int            // -1 means start from beginning (default).
	InitialVariables   map[string]any // Variables restored from previous execution (merged into store).
	ResumedFromID      *uuid.UUID     // ID of the original execution being resumed.

	// Namespace-aware variable support (Phase 2).
	// These fields map to ExecutionParameters proto fields.
	ProjectRoot   string         // Absolute path to project root for workflowPath resolution.
	InitialStore  map[string]any // Initial @store/ values - pre-seeded runtime state.
	InitialParams map[string]any // Initial @params/ values - workflow input contract.
	Env           map[string]any // Environment values - project/user configuration.

	// ArtifactConfig controls what artifacts are collected during execution.
	// When nil, defaults to "full" profile (all artifacts collected).
	// See: config/artifact_profiles.go for available profiles and settings.
	ArtifactConfig *config.ArtifactCollectionSettings
}

// Executor orchestrates plan execution using an engine, recorder, and event sink.
type Executor interface {
	Execute(ctx context.Context, req Request) error
}

// CapabilityError reports incompatibility between requested plan requirements
// and available engine capabilities.
type CapabilityError struct {
	Engine    string
	Missing   []string
	Warnings  []string
	Reasons   map[string][]string
	Execution uuid.UUID
	Workflow  uuid.UUID
}

func (e *CapabilityError) Error() string {
	return fmt.Sprintf("engine %s cannot satisfy requirements: missing=%v warnings=%v", e.Engine, e.Missing, e.Warnings)
}

// WorkflowResolver fetches workflow definitions for subflow execution.
type WorkflowResolver interface {
	GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowSummary, error)
	GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*basapi.WorkflowSummary, error)
	GetWorkflowByProjectPath(ctx context.Context, callingWorkflowID uuid.UUID, workflowPath string, projectRoot string) (*basapi.WorkflowSummary, error)
}
