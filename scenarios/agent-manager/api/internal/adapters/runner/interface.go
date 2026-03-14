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
	"errors"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

// Continuation errors
var (
	// ErrSessionExpired indicates the session no longer exists or has expired.
	ErrSessionExpired = errors.New("session no longer exists or has expired")

	// ErrContinuationNotSupported indicates the runner doesn't support session continuation.
	ErrContinuationNotSupported = errors.New("runner does not support session continuation")
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

	// Continue resumes an existing session with a follow-up message.
	// Uses the stored session_id to continue the conversation.
	// Returns ErrSessionExpired if the session no longer exists.
	// Returns ErrContinuationNotSupported if the runner doesn't support this.
	Continue(ctx context.Context, req ContinueRequest) (*ExecuteResult, error)

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

	// SupportsContinuation indicates the runner can resume previous sessions.
	SupportsContinuation bool

	// SupportsImageAttachments indicates the runner can accept image file attachments.
	SupportsImageAttachments bool

	// MaxTurns is the maximum number of turns this runner supports (0 = unlimited).
	MaxTurns int

	// SupportedModels lists the models this runner can use.
	SupportedModels []string

	// SupportedFeatures lists which typed FeatureFlags this runner supports.
	// Unsupported features are silently ignored during arg building.
	SupportedFeatures []string

	// AllowedExtraFlags is the allowlist of extra CLI flags this runner accepts.
	// Flags not in this list are rejected during validation.
	AllowedExtraFlags []string
}

// ExecuteRequest contains everything needed to execute an agent.
type ExecuteRequest struct {
	// RunID identifies this execution for tracking and cancellation.
	RunID uuid.UUID

	// Tag is a custom identifier for this run (used in logs, process names).
	// If empty, defaults to RunID.String().
	Tag string

	// Profile contains the agent configuration (may be nil if using ResolvedConfig).
	Profile *domain.AgentProfile

	// ResolvedConfig contains the merged config (profile + inline overrides).
	// This takes precedence over Profile when set.
	ResolvedConfig *domain.RunConfig

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

	// Attachments contains image/file attachments for this request.
	Attachments []Attachment
}

// GetTag returns the tag for this request, defaulting to RunID if not set.
func (r *ExecuteRequest) GetTag() string {
	if r.Tag != "" {
		return r.Tag
	}
	return r.RunID.String()
}

// GetConfig returns the effective configuration, preferring ResolvedConfig over Profile.
func (r *ExecuteRequest) GetConfig() *domain.RunConfig {
	if r.ResolvedConfig != nil {
		return r.ResolvedConfig
	}
	if r.Profile != nil {
		cfg := domain.DefaultRunConfig()
		cfg.ApplyProfile(r.Profile)
		return cfg
	}
	return domain.DefaultRunConfig()
}

// ContinueRequest contains parameters for continuing an existing session.
type ContinueRequest struct {
	// RunID identifies the run being continued (for tracking).
	RunID uuid.UUID

	// SessionID is the runner-specific session identifier to resume.
	SessionID string

	// Prompt is the follow-up message to send to the agent.
	Prompt string

	// WorkingDir is the directory where the agent should execute.
	WorkingDir string

	// EventSink receives events as they occur during execution.
	EventSink EventSink

	// Environment contains additional environment variables.
	Environment map[string]string

	// Attachments contains image/file attachments for this request.
	Attachments []Attachment
}

// Attachment represents a file attachment to include in a request.
type Attachment struct {
	ID          string
	FileName    string
	ContentType string
	FilePath    string // Absolute filesystem path
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

	// SessionID is the runner-specific session identifier for conversation continuation.
	// Populated from runner stream events (session_id, thread_id, sessionID).
	SessionID string
}

// ExecutionMetrics contains statistics about the execution.
type ExecutionMetrics struct {
	TurnsUsed           int
	TokensInput         int
	TokensOutput        int
	CacheReadTokens     int
	CacheCreationTokens int
	ToolCallCount       int
	CostEstimateUSD     float64
}

// TotalTokens returns total tokens used, including cache tokens.
func TotalTokens(metrics ExecutionMetrics) int {
	return metrics.TokensInput + metrics.TokensOutput + metrics.CacheReadTokens + metrics.CacheCreationTokens
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
// FlagValidator - Validates runner-specific flags against allowlists
// -----------------------------------------------------------------------------

// FlagValidator validates runner-specific flags against runner allowlists.
// This is a SEAM: testable without real runners, swappable in tests.
type FlagValidator interface {
	ValidateFlags(runnerType domain.RunnerType, flags []string) error
	AllowedFlags(runnerType domain.RunnerType) []string
	SupportedFeatures(runnerType domain.RunnerType) []string
}

// RegistryFlagValidator derives allowlists from runner Capabilities.
type RegistryFlagValidator struct {
	registry Registry
}

// NewRegistryFlagValidator creates a FlagValidator backed by a runner registry.
func NewRegistryFlagValidator(registry Registry) *RegistryFlagValidator {
	return &RegistryFlagValidator{registry: registry}
}

// ValidateFlags checks that all flags are in the runner's allowlist.
func (v *RegistryFlagValidator) ValidateFlags(rt domain.RunnerType, flags []string) error {
	r, err := v.registry.Get(rt)
	if err != nil {
		return err
	}
	allowed := make(map[string]bool)
	for _, f := range r.Capabilities().AllowedExtraFlags {
		allowed[f] = true
	}
	var invalid []string
	for _, flag := range flags {
		name := flag
		if idx := strings.Index(flag, "="); idx > 0 {
			name = flag[:idx]
		}
		if !allowed[name] {
			invalid = append(invalid, flag)
		}
	}
	if len(invalid) > 0 {
		return domain.NewValidationError("extraFlags",
			"runner "+string(rt)+" does not allow: "+strings.Join(invalid, ", ")+
				" (allowed: "+strings.Join(r.Capabilities().AllowedExtraFlags, ", ")+")")
	}
	return nil
}

// AllowedFlags returns the allowlist for the given runner type.
func (v *RegistryFlagValidator) AllowedFlags(rt domain.RunnerType) []string {
	r, err := v.registry.Get(rt)
	if err != nil {
		return nil
	}
	return r.Capabilities().AllowedExtraFlags
}

// SupportedFeatures returns the supported features for the given runner type.
func (v *RegistryFlagValidator) SupportedFeatures(rt domain.RunnerType) []string {
	r, err := v.registry.Get(rt)
	if err != nil {
		return nil
	}
	return r.Capabilities().SupportedFeatures
}

// Verify interface compliance
var _ FlagValidator = (*RegistryFlagValidator)(nil)

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
