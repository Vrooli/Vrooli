// Package runner provides the runner adapter interface and implementations.
//
// This package defines the SEAM for agent execution. All agent runners
// (claude-code, codex, opencode) implement the Runner interface, enabling:
// - Swappable runner implementations
// - Easy testing with mock runners
// - New runners added without changing orchestration code
package runner

import (
	"context"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// Runner Interface - The primary seam for agent execution
// -----------------------------------------------------------------------------

// Runner is the interface that all agent runner adapters must implement.
// This is the core seam for agent execution, allowing different agent types
// to be used interchangeably.
type Runner interface {
	// Type returns the runner type identifier.
	Type() domain.RunnerType

	// Capabilities returns what this runner supports.
	Capabilities() Capabilities

	// Execute runs the agent with the given configuration.
	// This is a blocking call that returns when the agent completes.
	// Events are streamed to the provided EventSink during execution.
	Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error)

	// Stop attempts to gracefully stop a running agent.
	// Returns an error if the agent cannot be stopped.
	Stop(ctx context.Context, runID uuid.UUID) error

	// IsAvailable checks if this runner is currently available.
	// Returns false with a reason if the runner cannot be used.
	IsAvailable(ctx context.Context) (bool, string)
}

// Capabilities describes what features a runner supports.
type Capabilities struct {
	// SupportsMessages indicates the runner can capture structured messages.
	SupportsMessages bool

	// SupportsToolEvents indicates the runner captures tool call/result events.
	SupportsToolEvents bool

	// SupportsCostTracking indicates the runner reports token usage and cost.
	SupportsCostTracking bool

	// SupportsStreaming indicates the runner can stream events in real-time.
	SupportsStreaming bool

	// SupportsCancellation indicates the runner supports mid-execution cancellation.
	SupportsCancellation bool

	// MaxTurns is the maximum number of turns this runner supports (0 = unlimited).
	MaxTurns int

	// SupportedModels lists the models this runner can use.
	SupportedModels []string
}

// ExecuteRequest contains everything needed to execute an agent.
type ExecuteRequest struct {
	// RunID identifies this execution for tracking and cancellation.
	RunID uuid.UUID

	// Profile contains the agent configuration.
	Profile *domain.AgentProfile

	// Task contains the work to be performed.
	Task *domain.Task

	// WorkingDir is the directory where the agent should execute.
	// For sandboxed runs, this is the sandbox merged directory.
	// For in-place runs, this is the actual project directory.
	WorkingDir string

	// Prompt is the initial prompt/instruction for the agent.
	Prompt string

	// EventSink receives events as they occur during execution.
	EventSink EventSink

	// Environment contains additional environment variables.
	Environment map[string]string
}

// ExecuteResult contains the outcome of an agent execution.
type ExecuteResult struct {
	// Success indicates whether the agent completed without errors.
	Success bool

	// ExitCode is the agent process exit code.
	ExitCode int

	// Summary contains the structured output from the agent.
	Summary *domain.RunSummary

	// ErrorMessage contains any error message if Success is false.
	ErrorMessage string

	// Duration is how long the execution took.
	Duration time.Duration

	// Metrics contains execution metrics.
	Metrics ExecutionMetrics
}

// ExecutionMetrics contains statistics about the execution.
type ExecutionMetrics struct {
	TurnsUsed       int
	TokensInput     int
	TokensOutput    int
	ToolCallCount   int
	CostEstimateUSD float64
}

// -----------------------------------------------------------------------------
// EventSink - Interface for receiving execution events
// -----------------------------------------------------------------------------

// EventSink receives events during agent execution.
// This interface allows the orchestration layer to capture events
// without the runner knowing how they will be stored or streamed.
type EventSink interface {
	// Emit sends an event to the sink.
	// This should be non-blocking; implementations should buffer if needed.
	Emit(event *domain.RunEvent) error

	// Close signals that no more events will be sent.
	Close() error
}

// -----------------------------------------------------------------------------
// Registry - Runner registration and lookup
// -----------------------------------------------------------------------------

// Registry manages available runner implementations.
type Registry interface {
	// Register adds a runner to the registry.
	Register(runner Runner) error

	// Get retrieves a runner by type.
	Get(runnerType domain.RunnerType) (Runner, error)

	// List returns all registered runners.
	List() []Runner

	// Available returns runners that are currently available.
	Available(ctx context.Context) []Runner
}

// -----------------------------------------------------------------------------
// Factory - Runner creation
// -----------------------------------------------------------------------------

// Factory creates runner instances with specific configurations.
type Factory interface {
	// Create creates a runner for the specified type.
	Create(runnerType domain.RunnerType, config Config) (Runner, error)
}

// Config holds common runner configuration.
type Config struct {
	// BinaryPath is the path to the runner binary (if applicable).
	BinaryPath string

	// Timeout is the default execution timeout.
	Timeout time.Duration

	// WorkDir is the default working directory.
	WorkDir string

	// Environment contains default environment variables.
	Environment map[string]string
}
