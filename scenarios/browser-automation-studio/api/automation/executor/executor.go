package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/automation/events"
	"github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/database"
)

// Request wires together the dependencies required to run a compiled plan.
type Request struct {
	Plan              contracts.ExecutionPlan
	EngineName        string
	EngineFactory     engine.Factory
	Recorder          recorder.Recorder
	EventSink         events.Sink
	HeartbeatInterval time.Duration
	ReuseMode         engine.SessionReuseMode
	WorkflowResolver  WorkflowResolver // Required for subflow resolution.
	PlanCompiler      PlanCompiler     // Optional; defaults to engine-registered compiler.
	MaxSubflowDepth   int              // Optional; defaults to 5.
	SubflowStack      []uuid.UUID      // Internal: call stack to avoid recursion.
	EngineCaps        *contracts.EngineCapabilities
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
	GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*database.Workflow, error)
}
