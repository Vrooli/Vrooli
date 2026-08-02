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
	"os"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/fallback"

	"github.com/google/uuid"
)

// Continuation errors
var (
	// ErrContinuationNotSupported indicates the runner doesn't support
	// session continuation. Session-expiry on the wire is now signalled
	// via *domain.RunnerError with ErrCodeRunnerSessionExpired (see
	// codecs.Codec.ClassifyTerminalError).
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
	// Returns a typed *domain.RunnerError (with code RUNNER_SESSION_EXPIRED
	// or RUNNER_SESSION_STATE_LOST) when the codec recognises a known
	// session/state failure shape. Returns ErrContinuationNotSupported
	// if the runner doesn't support continuation at all.
	Continue(ctx context.Context, req ContinueRequest) (*ExecuteResult, error)

	// Stop attempts to gracefully stop a running agent.
	// Returns an error if the agent cannot be stopped.
	Stop(ctx context.Context, runID uuid.UUID) error

	// IsAvailable checks if this runner is currently available.
	// Returns false with a reason if the runner cannot be used.
	IsAvailable(ctx context.Context) (bool, string)

	// ProbeModel performs a lightweight check that the given model ID is
	// usable by this runner. Implementations should avoid full inference —
	// the goal is to surface obviously-broken configurations at startup, not
	// to validate every possible failure mode. The empty string represents
	// the runner-default sentinel; probes should treat it as "accept".
	//
	// Returns nil when the model appears usable (or when the runner cannot
	// cheaply tell). Authoritative "this model is dead" signal comes from
	// runtime classification via Classify (see fallback.ClassifiedError).
	ProbeModel(ctx context.Context, modelID string) error

	// Classify converts a non-success Execute outcome (stderr +
	// exitCode) into a typed *fallback.ClassifiedError. Delegates to
	// the codec's structured-signal classifier with TextClassifier as
	// the residual safety net. Returns nil when stderr is empty AND
	// exitCode == 0.
	//
	// DOC: scenarios/agent-manager/docs/internal/EVENT_TAXONOMY.md
	// (model.fallback.attempted reason field).
	Classify(stderr string, exitCode int) *fallback.ClassifiedError
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

	// SupportsToolRestriction reports whether this runner can enforce the
	// canonical AllowedTools restriction for an individual launch.
	SupportsToolRestriction bool

	// ToolRestrictionMappings maps canonical profile tools to native runner
	// names. An empty map is the explicit unsupported stance.
	ToolRestrictionMappings map[string]string

	// SupportsEffort reports whether this runner applies the canonical effort
	// control to each launch and continuation.
	SupportsEffort bool

	// EffortMappings documents canonical-to-native effort translation. An empty
	// map is the explicit unsupported stance unless EffortModelSpecific is true.
	EffortMappings map[string]string

	// EffortModelSpecific means the runner accepts effort only for a declared
	// provider/model domain. Such runners intentionally publish no universal
	// mapping; their codec validates the selected model before launch.
	EffortModelSpecific bool

	// MaxTurns is the maximum number of turns this runner supports (0 = unlimited).
	MaxTurns int

	// SupportedModels contains only runtime-discovered model facts. Concrete
	// coding-agent selections come from resource-owned role resolution and are
	// captured in immutable run snapshots, never composed into runner mechanics.
	SupportedModels []string

	// SupportsRunnerDefault reports whether omitting the model flag is a valid
	// execution choice for this runner.
	SupportsRunnerDefault bool

	// DynamicModelPrefixes names runtime-discovered namespaces owned by the
	// codec.
	DynamicModelPrefixes []string

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

	// SandboxID identifies the workspace-sandbox container for this run when
	// the run is sandboxed. Required for protected-mode launches (the runner
	// uses it to construct a SandboxLauncher via SandboxLauncherFactory).
	// nil for in-place runs.
	SandboxID *uuid.UUID

	// Prompt is the user message for the agent (context data, task question).
	// For runners that support system prompts, this contains only the user-facing
	// content. For runners that don't, SystemPrompt is prepended to this.
	Prompt string

	// SystemPrompt contains stable instructions/methodology that should be
	// delivered via the runner's system prompt mechanism when supported.
	// Claude Code: passed via --append-system-prompt flag.
	// Codex/OpenCode: prepended to Prompt with <system-instructions> tags.
	SystemPrompt string

	// EventSink receives events as they occur during execution.
	EventSink EventSink

	// Environment contains additional environment variables.
	Environment map[string]string

	// Attachments contains image/file attachments for this request.
	Attachments []Attachment

	// Transcript config enables durable stdout capture and replay across
	// agent-manager restarts. When nil, runners fall back to the legacy
	// in-process streaming path.
	Transcript *TranscriptConfig
}

// EffectivePrompt returns the prompt with the system prompt prepended using
// XML tags, for runners that don't support a native system prompt mechanism.
// If SystemPrompt is empty, returns Prompt unchanged.
func (r *ExecuteRequest) EffectivePrompt() string {
	if r.SystemPrompt == "" {
		return r.Prompt
	}
	if r.Prompt == "" {
		return r.SystemPrompt
	}
	return "<system-instructions>\n" + r.SystemPrompt + "\n</system-instructions>\n\n" + r.Prompt
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

	// Transcript config enables durable stdout capture and replay across
	// agent-manager restarts for continuation turns.
	Transcript *TranscriptConfig

	// ResolvedConfig carries the run's resolved sandbox/runner config so
	// continuation routes through the same Launcher (host vs sandbox) as
	// the original Execute call. Populated by the orchestrator from the
	// stored Run.ResolvedConfig.
	ResolvedConfig *domain.RunConfig

	// SandboxID identifies the workspace-sandbox container the original
	// Execute call ran in, when the run was sandboxed. Required for
	// protected-mode routing on the continuation; nil otherwise.
	SandboxID *uuid.UUID
}

// GetConfig returns the resolved configuration for the continuation,
// falling back to the runtime default. Mirrors ExecuteRequest.GetConfig
// so launcherSelector.PickFor can route both call shapes uniformly.
func (r *ContinueRequest) GetConfig() *domain.RunConfig {
	if r.ResolvedConfig != nil {
		return r.ResolvedConfig
	}
	return domain.DefaultRunConfig()
}

type TranscriptConfig struct {
	TranscriptPath string
	StderrPath     string
	StdoutFile     *os.File
	StderrFile     *os.File
	OnProcessStart func(pid, pgid int) error
	OnAdvance      func(cursor, lastSeq int64) error
	OnSessionID    func(sessionID string) error
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

	// Result is the canonical provenance-bearing terminal output. Summary is
	// derived from it for compatibility after resolution.
	Result *domain.RunResult

	// ErrorMessage contains any error message if Success is false.
	ErrorMessage string

	// Duration is how long the execution took.
	Duration time.Duration

	// Metrics contains execution metrics.
	Metrics ExecutionMetrics

	// SessionID is the runner-specific session identifier for conversation continuation.
	// Populated from runner stream events (session_id, thread_id, sessionID).
	SessionID string

	// TerminalError carries a typed error when the runner detected a
	// terminal failure (e.g. ErrSandboxNoExitInfo bubbling up from the
	// sandbox launcher's Wait). When non-nil, the orchestration layer
	// promotes it to e.execErr so the typed-error path classifies the
	// failure correctly (SANDBOX_NO_EXIT_INFO, etc.). Optional —
	// runners that don't have typed errors leave it nil and orchestration
	// falls back to ErrorMessage as before.
	TerminalError error
}

// ExecutionMetrics contains statistics about the execution.
type ExecutionMetrics struct {
	TurnsUsed           int
	TokensInput         int
	TokensOutput        int
	CacheReadTokens     int
	CacheCreationTokens int
	ToolCallCount       int
	// SuccessfulToolResults counts tool results that reported success
	// (ToolResultEventData.Success). It is the "did the agent's actions
	// actually land?" signal: a run can emit a tool call yet produce no
	// successful result (e.g. a write to a hallucinated path, or a tool
	// call narrated as text the runner never executed). Codecs use this to
	// distinguish real work from a silent no-op in PostClassify.
	SuccessfulToolResults int
	CostEstimateUSD       float64
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

// SequencedEventSink is an optional EventSink extension that exposes the last
// persisted run_events.sequence value after Emit() succeeds.
type SequencedEventSink interface {
	EventSink
	LastSequence() int64
}

// TranscriptParseResult is the normalized output from a runner-specific
// transcript parser when consuming durable stdout from transcript.ndjson.
type TranscriptParseResult struct {
	Events    []*domain.RunEvent
	SessionID string
	Terminal  *TranscriptTerminal
	Err       error
}

// TranscriptTerminal captures runner-native terminal state discovered from the
// transcript itself, which is required when the orchestrator dies before it can
// synthesize final status events.
type TranscriptTerminal struct {
	Success      bool
	ExitCode     int
	ErrorMessage string
	Summary      *domain.RunSummary
}

// TranscriptParser is an optional runner seam used by transcript recovery.
// Implementations should reuse the same per-runner parsing logic used for live
// execution rather than introducing a second translation layer.
type TranscriptParser interface {
	ParseTranscriptLine(runID uuid.UUID, line string) TranscriptParseResult
}

// TranscriptParserFactory creates a fresh parser for one logical transcript
// consumption. Callers that tail live output and then perform a final drain
// must share the returned parser across both Consume calls so replay state
// follows the transcript stream rather than resetting per line.
type TranscriptParserFactory interface {
	NewTranscriptParser() TranscriptParser
}

// TranscriptModelSetter supplies the resolved model to transcript-derived
// parsers. Recovery/import paths do not replay BuildArgs, so they must carry
// this attribution explicitly instead of silently emitting an unlabeled use.
type TranscriptModelSetter interface {
	SetTranscriptModel(string)
}

// AgentLaunchInfo exposes the per-agent facts the interactive execution
// substrate needs to build a launch command for the real interactive CLI: the
// per-run tag env key (so the reconciler can attribute the process from
// /proc) and the resolved binary path. core.Runner satisfies this by
// delegating to its codec, mirroring how it satisfies [TranscriptParser].
// Callers resolve it via Registry.Get(runnerType) + a type assertion, the same
// pattern transcript recovery uses.
type AgentLaunchInfo interface {
	Type() domain.RunnerType
	TagEnvKey() string
	BinaryPath() string
	ControlArgs(cfg *domain.RunConfig) ([]string, error)
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
