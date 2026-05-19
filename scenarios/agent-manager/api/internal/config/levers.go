// Package config provides configuration loading and management.
//
// This file defines the CONTROL SURFACE for agent-manager: the set of
// meaningful, safe levers that operators and agents can tune without
// touching implementation code.
//
// DESIGN PRINCIPLES:
// - Fewer, well-chosen knobs over many obscure ones
// - Clear defaults that work for common usage
// - Safe bounds on all values to prevent catastrophic misconfiguration
// - Grouped by operator mental model, not implementation structure
//
// DOC: scenarios/agent-manager/docs/reference/configuration.md
// (full lever-by-lever reference, organized by section).
// DOC: scenarios/agent-manager/docs/internal/TEMPORAL-FLOWS.md
// (which lever controls which timing surface).

package config

import (
	"fmt"
	"time"

	"agent-manager/internal/domain"
)

// =============================================================================
// CONTROL SURFACE DEFINITION
// =============================================================================

// Levers contains all tunable parameters for agent-manager.
// This is the primary control surface - operators adjust these to
// customize behavior without code changes.
type Levers struct {
	// Execution controls how agent runs behave
	Execution ExecutionLevers `json:"execution"`

	// Safety controls accident prevention and isolation
	Safety SafetyLevers `json:"safety"`

	// Concurrency controls parallelism and resource usage
	Concurrency ConcurrencyLevers `json:"concurrency"`

	// Approval controls review workflow behavior
	Approval ApprovalLevers `json:"approval"`

	// Runners controls agent runner behavior
	Runners RunnerLevers `json:"runners"`

	// Server controls API server behavior
	Server ServerLevers `json:"server"`

	// Storage controls persistence settings
	Storage StorageLevers `json:"storage"`

	// Heartbeat controls run-lifecycle cadence (heartbeat, checkpoint,
	// staleness, teardown, retries) — internal levers, not user-facing.
	Heartbeat HeartbeatLevers `json:"heartbeat"`

	// Recovery controls transcript-tail and resume-after-restart timing.
	Recovery RecoveryLevers `json:"recovery"`

	// Scanner controls stdout/transcript byte buffer ceilings.
	Scanner ScannerLevers `json:"scanner"`

	// Diagnostics controls heuristic windows used by run-outcome
	// classification (silent-launch detection, message truncation).
	Diagnostics DiagnosticsLevers `json:"diagnostics"`

	// Observability controls structured logging format and verbosity.
	// DOC: scenarios/agent-manager/docs/internal/SEAMS.md
	// (the obs.Logger seam) and obs/log.go (stable key constants).
	Observability ObservabilityLevers `json:"observability"`

	// Spawn controls runner-startup serialization. The dispatcher caps
	// how many runs may be in the codex bootstrap window simultaneously
	// and exposes queue depth so callers see backpressure.
	// DOC: scenarios/agent-manager/docs/internal/SEAMS.md
	// (the spawn.Dispatcher seam) and spawn/dispatcher.go.
	Spawn SpawnLevers `json:"spawn"`

	// Sandbox controls workspace-sandbox availability checks and bounded
	// retries around run-time sandbox operations.
	// DOC: scenarios/agent-manager/docs/internal/SEAMS.md
	// DOC: scenarios/agent-manager/docs/internal/TEMPORAL-FLOWS.md
	Sandbox SandboxLevers `json:"sandbox"`
}

// =============================================================================
// EXECUTION LEVERS
// =============================================================================

// ExecutionLevers control how agent runs execute.
type ExecutionLevers struct {
	// DefaultTimeout is the maximum execution time for a run if not specified.
	// Higher = more time for complex tasks, but longer potential resource usage.
	// Range: 1m to 4h. Default: 30m.
	DefaultTimeout time.Duration `json:"defaultTimeout"`

	// DefaultMaxTurns limits conversation turns if not specified by profile.
	// Higher = more agent autonomy, but potential for runaway loops.
	// Range: 1 to 1000. Default: 100.
	DefaultMaxTurns int `json:"defaultMaxTurns"`

	// EventBufferSize controls how many events are buffered before flushing.
	// Higher = better throughput, but more memory usage and potential event loss.
	// Range: 10 to 10000. Default: 100.
	EventBufferSize int `json:"eventBufferSize"`

	// EventFlushInterval controls how often buffered events are flushed.
	// Lower = more responsive event streaming, but more I/O overhead.
	// Range: 100ms to 30s. Default: 1s.
	EventFlushInterval time.Duration `json:"eventFlushInterval"`
}

// =============================================================================
// SAFETY LEVERS
// =============================================================================

// SafetyLevers control accident prevention mechanisms.
// These exist to prevent accidental damage, not adversarial attacks.
type SafetyLevers struct {
	// RequireSandboxByDefault makes all runs use sandbox unless explicitly overridden.
	// true = safer but slower due to sandbox creation overhead.
	// Default: true (sandbox-first philosophy).
	RequireSandboxByDefault bool `json:"requireSandboxByDefault"`

	// AllowInPlaceOverride permits explicit requests to skip sandbox.
	// false = no runs ever execute in-place, regardless of policy.
	// Default: true (allow controlled override).
	AllowInPlaceOverride bool `json:"allowInPlaceOverride"`

	// MaxFilesPerRun limits files a single run can modify.
	// Higher = more capability, but larger blast radius on mistakes.
	// Range: 1 to 10000. Default: 500.
	MaxFilesPerRun int `json:"maxFilesPerRun"`

	// MaxBytesPerRun limits total bytes a single run can change.
	// Higher = more capability, but larger potential damage.
	// Range: 1KB to 1GB. Default: 50MB.
	MaxBytesPerRun int64 `json:"maxBytesPerRun"`

	// DenyPathPatterns are glob patterns that no run can modify.
	// These provide hard guardrails regardless of policy.
	// Default: [".git/**", ".env*", "**/secrets/**", "**/*.key"]
	DenyPathPatterns []string `json:"denyPathPatterns"`
}

// =============================================================================
// CONCURRENCY LEVERS
// =============================================================================

// ConcurrencyLevers control parallelism and resource limits.
type ConcurrencyLevers struct {
	// MaxConcurrentRuns limits total simultaneous runs across all scopes.
	// Higher = more parallelism, but more resource usage.
	// Range: 1 to 100. Default: 10.
	MaxConcurrentRuns int `json:"maxConcurrentRuns"`

	// MaxConcurrentPerScope limits runs in a single scope path.
	// Higher = more parallelism in one area, but higher conflict risk.
	// Range: 1 to 10. Default: 1 (fully exclusive scope locks).
	MaxConcurrentPerScope int `json:"maxConcurrentPerScope"`

	// ScopeLockTTL is how long a scope lock is held before auto-release.
	// Higher = more protection against orphaned locks, but longer waits.
	// Range: 5m to 24h. Default: 30m.
	ScopeLockTTL time.Duration `json:"scopeLockTTL"`

	// ScopeLockRefreshInterval is how often active runs refresh their locks.
	// Lower = less chance of accidental lock expiry during long runs.
	// Range: 30s to 10m. Default: 5m.
	ScopeLockRefreshInterval time.Duration `json:"scopeLockRefreshInterval"`

	// QueueWaitTimeout is how long to wait for capacity before failing.
	// Higher = more patience for busy systems, but longer user waits.
	// Range: 0 (fail fast) to 30m. Default: 5m.
	QueueWaitTimeout time.Duration `json:"queueWaitTimeout"`
}

// =============================================================================
// APPROVAL LEVERS
// =============================================================================

// ApprovalLevers control the review workflow.
type ApprovalLevers struct {
	// RequireApprovalByDefault makes all runs need review before applying changes.
	// false = auto-apply successful runs (dangerous, use with caution).
	// Default: true.
	RequireApprovalByDefault bool `json:"requireApprovalByDefault"`

	// AutoApprovePatterns are scope patterns that skip approval workflow.
	// Use for low-risk areas like test files or documentation.
	// Default: [] (no auto-approval).
	AutoApprovePatterns []string `json:"autoApprovePatterns"`

	// ReviewTimeoutDays is how long runs wait for review before stale warning.
	// After this, operators are notified of pending reviews.
	// Range: 1 to 90. Default: 7.
	ReviewTimeoutDays int `json:"reviewTimeoutDays"`

	// AllowPartialApproval enables approving individual files from a run.
	// true = more flexibility, false = all-or-nothing review.
	// Default: true.
	AllowPartialApproval bool `json:"allowPartialApproval"`
}

// =============================================================================
// RUNNER LEVERS
// =============================================================================

// RunnerLevers control agent runner behavior.
type RunnerLevers struct {
	// ClaudeCodePath is the path to the claude-code binary.
	// Default: "claude" (assumes in PATH).
	ClaudeCodePath string `json:"claudeCodePath"`

	// CodexPath is the path to the codex binary.
	// Default: "codex" (assumes in PATH).
	CodexPath string `json:"codexPath"`

	// OpenCodePath is the path to the opencode binary.
	// Default: "opencode" (assumes in PATH).
	OpenCodePath string `json:"opencodePath"`

	// FallbackRunnerTypes is the ordered list of runners to try if the primary fails.
	// Empty disables automatic fallback.
	FallbackRunnerTypes []string `json:"fallbackRunnerTypes"`

	// HealthCheckInterval is how often to verify runner availability.
	// Lower = faster detection of unavailable runners.
	// Range: 10s to 5m. Default: 1m.
	HealthCheckInterval time.Duration `json:"healthCheckInterval"`

	// StartupGracePeriod is how long to wait for a runner to become available.
	// Useful when runners are slow to initialize.
	// Range: 0 to 5m. Default: 30s.
	StartupGracePeriod time.Duration `json:"startupGracePeriod"`

	// ProbeTimeout bounds a single runner-availability probe (e.g. `codex
	// status`, `opencode --version`). Kept short so an unhealthy binary
	// does not stall startup or the orchestrator's IsAvailable path.
	// Range: 1s to 30s. Default: 5s.
	ProbeTimeout time.Duration `json:"probeTimeout"`

	// UseCliDefaultModel, when true, causes orchestration to skip passing
	// a model flag to the runner CLI (no `-m`/`--model`). The agent runs
	// with whatever model the installed CLI is configured to use (e.g.
	// the model selected via `codex /model` or `claude /model`).
	//
	// Use this when you want the runner's subscription/plan default to
	// govern pricing instead of agent-manager's preset chain — for
	// example, a ChatGPT Plus account that already pins a model. Profile
	// model presets and the model-fallback chain are bypassed entirely
	// while this is on.
	// Default: false.
	UseCliDefaultModel bool `json:"useCliDefaultModel"`
}

// =============================================================================
// SERVER LEVERS
// =============================================================================

// ServerLevers control API server behavior.
type ServerLevers struct {
	// Port is the HTTP port to listen on.
	// Default: "8080".
	Port string `json:"port"`

	// ReadTimeout is the maximum duration for reading request body.
	// Range: 5s to 5m. Default: 30s.
	ReadTimeout time.Duration `json:"readTimeout"`

	// WriteTimeout is the maximum duration for writing response.
	// Range: 5s to 10m. Default: 30s.
	WriteTimeout time.Duration `json:"writeTimeout"`

	// IdleTimeout is the maximum duration between requests before closing.
	// Range: 30s to 10m. Default: 2m.
	IdleTimeout time.Duration `json:"idleTimeout"`

	// MaxRequestBodyBytes limits request body size.
	// Range: 1KB to 100MB. Default: 10MB.
	MaxRequestBodyBytes int64 `json:"maxRequestBodyBytes"`
}

// =============================================================================
// STORAGE LEVERS
// =============================================================================

// StorageLevers control persistence settings.
type StorageLevers struct {
	// DatabaseURL is the SQLite database path or connection string.
	// Required for persistence.
	DatabaseURL string `json:"databaseUrl"`

	// MaxOpenConns limits concurrent database connections.
	// Higher = more throughput, but more database load.
	// Range: 5 to 100. Default: 25.
	MaxOpenConns int `json:"maxOpenConns"`

	// MaxIdleConns limits idle database connections.
	// Higher = faster connection reuse, but more memory.
	// Range: 1 to 50. Default: 5.
	MaxIdleConns int `json:"maxIdleConns"`

	// ConnMaxLifetime is how long a connection lives before recycling.
	// Lower = more connection overhead, but fresher connections.
	// Range: 1m to 1h. Default: 5m.
	ConnMaxLifetime time.Duration `json:"connMaxLifetime"`

	// EventRetentionDays is how long to keep run events.
	// Higher = more history, but more storage.
	// Range: 1 to 365. Default: 30.
	EventRetentionDays int `json:"eventRetentionDays"`

	// ArtifactRetentionDays is how long to keep run artifacts.
	// Higher = more history, but more storage.
	// Range: 1 to 365. Default: 90.
	ArtifactRetentionDays int `json:"artifactRetentionDays"`

	// RunStateRetentionDays is how long the reconciler keeps on-disk
	// run-state directories (transcripts, scratchpads) for terminal
	// (Complete / Failed / Cancelled) runs before sweeping them.
	// Range: 1 to 365. Default: 7.
	RunStateRetentionDays int `json:"runStateRetentionDays"`
}

// =============================================================================
// HEARTBEAT LEVERS
// =============================================================================

// HeartbeatLevers control run-execution lifecycle cadence.
// These are internal levers — not exposed via env vars by default.
type HeartbeatLevers struct {
	// RunHeartbeatInterval is how often the run-executor heartbeat goroutine
	// updates last_heartbeat_at on the run row. The reconciler uses
	// last_heartbeat_at + StaleThreshold to detect stuck runs, so this must
	// fire well below StaleThreshold.
	// Range: 1s to 5m. Default: 15s.
	RunHeartbeatInterval time.Duration `json:"runHeartbeatInterval"`

	// CheckpointInterval is how often the run-executor persists checkpoint
	// state for resumption. Lower = finer-grained recovery, higher overhead.
	// Range: 5s to 10m. Default: 1m.
	CheckpointInterval time.Duration `json:"checkpointInterval"`

	// StaleThreshold is how long without a heartbeat before the reconciler
	// considers a run stale. Must comfortably exceed RunHeartbeatInterval to
	// tolerate slow DB writes and long tool calls.
	// Range: 1m to 30m. Default: 5m.
	StaleThreshold time.Duration `json:"staleThreshold"`

	// TeardownTimeout bounds the detached context used by finalize() for
	// sandbox Delete/Stop calls. Independent of run timeout — teardown must
	// complete even when the run deadline already expired.
	// Range: 5s to 5m. Default: 30s.
	TeardownTimeout time.Duration `json:"teardownTimeout"`

	// MaxRetriesPerPhase bounds retries for transient phase failures.
	// Range: 0 to 10. Default: 3.
	MaxRetriesPerPhase int `json:"maxRetriesPerPhase"`

	// AgentTickInterval is how often runner-internal heartbeat goroutines
	// (e.g. claude-code stream-stall watchdog) wake up to evaluate the
	// idle threshold. Smaller = faster idle reporting, more wake-ups.
	// Range: 100ms to 10s. Default: 2s.
	AgentTickInterval time.Duration `json:"agentTickInterval"`

	// AgentIdleThreshold is how long the runner-internal stream watchdog
	// waits without observing a stdout event before emitting a debug log
	// event. Used by claude-code's heartbeat to surface stream stalls.
	// Range: 1s to 5m. Default: 30s.
	AgentIdleThreshold time.Duration `json:"agentIdleThreshold"`

	// RunnerSignalGracePeriod is how long Runner.Stop() waits for a
	// SIGTERM-style graceful shutdown before escalating to SIGKILL.
	// Same value is used as the context timeout for Stop's HTTP/IPC call.
	// Range: 1s to 1m. Default: 5s.
	RunnerSignalGracePeriod time.Duration `json:"runnerSignalGracePeriod"`
}

// =============================================================================
// RECOVERY LEVERS
// =============================================================================

// RecoveryLevers control resume-after-restart timing.
type RecoveryLevers struct {
	// TranscriptTailInterval is how often the recovery tailer polls the
	// transcript file for new lines after reattaching to a live run.
	// Range: 50ms to 5s. Default: 100ms.
	TranscriptTailInterval time.Duration `json:"transcriptTailInterval"`

	// TranscriptPollInterval is the default poll interval for
	// runner.Consume when no caller-supplied value is given.
	// Range: 50ms to 5s. Default: 100ms.
	TranscriptPollInterval time.Duration `json:"transcriptPollInterval"`
}

// =============================================================================
// SCANNER LEVERS
// =============================================================================

// ScannerLevers control byte-buffer ceilings used when consuming
// runner stdout and transcript files.
type ScannerLevers struct {
	// StdoutMaxLineBytes is the maximum line length bufio.Scanner will
	// accept when reading runner stdout. Long lines beyond this trip
	// scanner.Err() = bufio.ErrTooLong.
	// Range: 64KB to 64MB. Default: 10MB.
	StdoutMaxLineBytes int `json:"stdoutMaxLineBytes"`

	// TranscriptMaxLineBytes is the maximum line length when reading
	// the persisted NDJSON transcript on resume.
	// Range: 64KB to 64MB. Default: 10MB.
	TranscriptMaxLineBytes int `json:"transcriptMaxLineBytes"`
}

// =============================================================================
// DIAGNOSTICS LEVERS
// =============================================================================

// DiagnosticsLevers control heuristic windows used by run-outcome
// classification.
type DiagnosticsLevers struct {
	// LaunchFailedMaxDuration is the upper bound for "ran too fast to be a
	// real run." When a sandbox-protected run exits within this window with
	// no message events, validateOutcome reclassifies the failure as a
	// launch failure (e.g. bwrap setup error).
	// Range: 100ms to 30s. Default: 2s.
	LaunchFailedMaxDuration time.Duration `json:"launchFailedMaxDuration"`

	// RateLimitMessageMaxLen truncates rate-limit error messages before
	// surfacing to operators. Keeps logs readable when providers return
	// long structured error bodies.
	// Range: 64 to 8192. Default: 512.
	RateLimitMessageMaxLen int `json:"rateLimitMessageMaxLen"`
}

// =============================================================================
// OBSERVABILITY LEVERS
// =============================================================================

// ObservabilityLevers control structured-logging output. Logging in
// agent-manager is centralised in internal/orchestration/obs; these
// levers feed obs.Init at server startup.
type ObservabilityLevers struct {
	// LogFormat selects the slog handler. Allowed values: "text" (human-
	// readable, default for development) and "json" (one structured JSON
	// object per line, suitable for shipping to log aggregators).
	// Default: "text".
	LogFormat string `json:"logFormat"`

	// LogLevel sets the minimum slog level to emit. Allowed values:
	// "debug", "info", "warn", "error".
	// Default: "info".
	LogLevel string `json:"logLevel"`
}

// =============================================================================
// SPAWN LEVERS
// =============================================================================

// SpawnLevers control runner-startup serialization in
// orchestration/spawn.Dispatcher. The defaults are conservative because
// codex's bootstrap window (SQLite WAL contention + rollout-file open
// race) burst-fails silently when N>1 starts overlap; lift them only
// once the burst-test confirms the runner tolerates parallelism.
type SpawnLevers struct {
	// MaxStartingConcurrency caps how many runs may be in the codex-
	// bootstrap window simultaneously. Default 1 (strict serialization).
	// Range: 1 to 16.
	MaxStartingConcurrency int `json:"maxStartingConcurrency"`

	// MinSpacing is the minimum delay between two successive
	// slot-acquisition events. Zero disables spacing. Use this to
	// stretch out spawns when MaxStartingConcurrency > 1 still
	// produces transient races.
	// Range: 0 to 30s.
	MinSpacing time.Duration `json:"minSpacing"`

	// QueueCapacity is the maximum number of runs that may be queued
	// (not yet started) before Enqueue returns CapacityExceededError.
	// Set to zero to derive from Concurrency.MaxConcurrentRuns at
	// orchestrator construction time (default = MaxConcurrentRuns * 2).
	// Range: 0 (auto) or 1 to 1024 (explicit).
	QueueCapacity int `json:"queueCapacity"`
}

// =============================================================================
// SANDBOX LEVERS
// =============================================================================

// SandboxLevers controls run-time workspace-sandbox availability checks and
// operation retries. These levers apply when a run needs workspace-sandbox;
// agent-manager bootstrap dependency startup remains owned by Vrooli lifecycle.
type SandboxLevers struct {
	// AvailabilityCheckTimeout bounds a single workspace-sandbox health check.
	// Range: 100ms to 10s. Default: 2s.
	AvailabilityCheckTimeout time.Duration `json:"availabilityCheckTimeout"`

	// EnsureStartTimeout bounds an on-demand lifecycle start + health-poll
	// attempt when workspace-sandbox is unavailable during a run.
	// Range: 5s to 2m. Default: 60s.
	EnsureStartTimeout time.Duration `json:"ensureStartTimeout"`

	// EnsurePollInterval controls health polling while waiting for
	// workspace-sandbox to become ready after lifecycle start.
	// Range: 50ms to 5s and less than EnsureStartTimeout. Default: 500ms.
	EnsurePollInterval time.Duration `json:"ensurePollInterval"`

	// OperationMaxAttempts bounds transient sandbox operation retries.
	// Range: 1 to 8. Default: 4.
	OperationMaxAttempts int `json:"operationMaxAttempts"`

	// OperationInitialBackoff is the first retry delay after a transient
	// workspace-sandbox operation failure.
	// Range: 25ms to 5s. Default: 250ms.
	OperationInitialBackoff time.Duration `json:"operationInitialBackoff"`

	// OperationMaxBackoff caps exponential retry delay.
	// Range: OperationInitialBackoff to 30s. Default: 2s.
	OperationMaxBackoff time.Duration `json:"operationMaxBackoff"`
}

// =============================================================================
// DEFAULTS
// =============================================================================

// DefaultLevers returns the default configuration that works for most use cases.
func DefaultLevers() Levers {
	return Levers{
		Execution: ExecutionLevers{
			DefaultTimeout:     60 * time.Minute,
			DefaultMaxTurns:    100,
			EventBufferSize:    100,
			EventFlushInterval: 1 * time.Second,
		},
		Safety: SafetyLevers{
			RequireSandboxByDefault: true,
			AllowInPlaceOverride:    true,
			MaxFilesPerRun:          500,
			MaxBytesPerRun:          50 * 1024 * 1024, // 50MB
			DenyPathPatterns: []string{
				".git/**",
				".env*",
				"**/secrets/**",
				"**/*.key",
				"**/*.pem",
				"**/credentials*",
			},
		},
		Concurrency: ConcurrencyLevers{
			MaxConcurrentRuns:        10,
			MaxConcurrentPerScope:    1,
			ScopeLockTTL:             30 * time.Minute,
			ScopeLockRefreshInterval: 5 * time.Minute,
			QueueWaitTimeout:         5 * time.Minute,
		},
		Approval: ApprovalLevers{
			RequireApprovalByDefault: true,
			AutoApprovePatterns:      []string{},
			ReviewTimeoutDays:        7,
			AllowPartialApproval:     true,
		},
		Runners: RunnerLevers{
			ClaudeCodePath:      "claude",
			CodexPath:           "codex",
			OpenCodePath:        "opencode",
			FallbackRunnerTypes: nil,
			HealthCheckInterval: 1 * time.Minute,
			StartupGracePeriod:  30 * time.Second,
			ProbeTimeout:        5 * time.Second,
		},
		Server: ServerLevers{
			Port:                "8080",
			ReadTimeout:         30 * time.Second,
			WriteTimeout:        30 * time.Second,
			IdleTimeout:         2 * time.Minute,
			MaxRequestBodyBytes: 10 * 1024 * 1024, // 10MB
		},
		Storage: StorageLevers{
			DatabaseURL:           "",
			MaxOpenConns:          25,
			MaxIdleConns:          5,
			ConnMaxLifetime:       5 * time.Minute,
			EventRetentionDays:    30,
			ArtifactRetentionDays: 90,
			RunStateRetentionDays: 7,
		},
		Heartbeat: HeartbeatLevers{
			RunHeartbeatInterval:    15 * time.Second,
			CheckpointInterval:      1 * time.Minute,
			StaleThreshold:          5 * time.Minute,
			TeardownTimeout:         30 * time.Second,
			MaxRetriesPerPhase:      3,
			AgentTickInterval:       2 * time.Second,
			AgentIdleThreshold:      30 * time.Second,
			RunnerSignalGracePeriod: 5 * time.Second,
		},
		Recovery: RecoveryLevers{
			TranscriptTailInterval: 100 * time.Millisecond,
			TranscriptPollInterval: 100 * time.Millisecond,
		},
		Scanner: ScannerLevers{
			StdoutMaxLineBytes:     10 * 1024 * 1024,
			TranscriptMaxLineBytes: 10 * 1024 * 1024,
		},
		Diagnostics: DiagnosticsLevers{
			LaunchFailedMaxDuration: 2 * time.Second,
			RateLimitMessageMaxLen:  512,
		},
		Observability: ObservabilityLevers{
			LogFormat: "text",
			LogLevel:  "info",
		},
		Spawn: SpawnLevers{
			MaxStartingConcurrency: 1,
			MinSpacing:             0,
			QueueCapacity:          0, // 0 means "auto-derive from MaxConcurrentRuns"
		},
		Sandbox: SandboxLevers{
			AvailabilityCheckTimeout: 2 * time.Second,
			EnsureStartTimeout:       60 * time.Second,
			EnsurePollInterval:       500 * time.Millisecond,
			OperationMaxAttempts:     4,
			OperationInitialBackoff:  250 * time.Millisecond,
			OperationMaxBackoff:      2 * time.Second,
		},
	}
}

// =============================================================================
// VALIDATION
// =============================================================================

// Validate checks all lever values are within safe bounds.
// Returns nil if valid, or an error describing the problem.
func (l *Levers) Validate() error {
	if err := l.Execution.Validate(); err != nil {
		return wrapConfigSection("execution", err)
	}
	if err := l.Safety.Validate(); err != nil {
		return wrapConfigSection("safety", err)
	}
	if err := l.Concurrency.Validate(); err != nil {
		return wrapConfigSection("concurrency", err)
	}
	if err := l.Approval.Validate(); err != nil {
		return wrapConfigSection("approval", err)
	}
	if err := l.Runners.Validate(); err != nil {
		return wrapConfigSection("runners", err)
	}
	if err := l.Server.Validate(); err != nil {
		return wrapConfigSection("server", err)
	}
	if err := l.Storage.Validate(); err != nil {
		return wrapConfigSection("storage", err)
	}
	if err := l.Heartbeat.Validate(); err != nil {
		return wrapConfigSection("heartbeat", err)
	}
	if err := l.Recovery.Validate(); err != nil {
		return wrapConfigSection("recovery", err)
	}
	if err := l.Scanner.Validate(); err != nil {
		return wrapConfigSection("scanner", err)
	}
	if err := l.Diagnostics.Validate(); err != nil {
		return wrapConfigSection("diagnostics", err)
	}
	if err := l.Observability.Validate(); err != nil {
		return wrapConfigSection("observability", err)
	}
	if err := l.Spawn.Validate(); err != nil {
		return wrapConfigSection("spawn", err)
	}
	if err := l.Sandbox.Validate(); err != nil {
		return wrapConfigSection("sandbox", err)
	}
	return nil
}

func (e *ExecutionLevers) Validate() error {
	if e.DefaultTimeout < time.Minute || e.DefaultTimeout > 4*time.Hour {
		return domain.NewConfigInvalidError("defaultTimeout", fmt.Sprintf("must be between 1m and 4h, got %v", e.DefaultTimeout), nil)
	}
	if e.DefaultMaxTurns < 1 || e.DefaultMaxTurns > 1000 {
		return domain.NewConfigInvalidError("defaultMaxTurns", fmt.Sprintf("must be between 1 and 1000, got %d", e.DefaultMaxTurns), nil)
	}
	if e.EventBufferSize < 10 || e.EventBufferSize > 10000 {
		return domain.NewConfigInvalidError("eventBufferSize", fmt.Sprintf("must be between 10 and 10000, got %d", e.EventBufferSize), nil)
	}
	if e.EventFlushInterval < 100*time.Millisecond || e.EventFlushInterval > 30*time.Second {
		return domain.NewConfigInvalidError("eventFlushInterval", fmt.Sprintf("must be between 100ms and 30s, got %v", e.EventFlushInterval), nil)
	}
	return nil
}

func (s *SafetyLevers) Validate() error {
	if s.MaxFilesPerRun < 1 || s.MaxFilesPerRun > 10000 {
		return domain.NewConfigInvalidError("maxFilesPerRun", fmt.Sprintf("must be between 1 and 10000, got %d", s.MaxFilesPerRun), nil)
	}
	if s.MaxBytesPerRun < 1024 || s.MaxBytesPerRun > 1024*1024*1024 {
		return domain.NewConfigInvalidError("maxBytesPerRun", fmt.Sprintf("must be between 1KB and 1GB, got %d", s.MaxBytesPerRun), nil)
	}
	return nil
}

func (c *ConcurrencyLevers) Validate() error {
	if c.MaxConcurrentRuns < 1 || c.MaxConcurrentRuns > 100 {
		return domain.NewConfigInvalidError("maxConcurrentRuns", fmt.Sprintf("must be between 1 and 100, got %d", c.MaxConcurrentRuns), nil)
	}
	if c.MaxConcurrentPerScope < 1 || c.MaxConcurrentPerScope > 10 {
		return domain.NewConfigInvalidError("maxConcurrentPerScope", fmt.Sprintf("must be between 1 and 10, got %d", c.MaxConcurrentPerScope), nil)
	}
	if c.ScopeLockTTL < 5*time.Minute || c.ScopeLockTTL > 24*time.Hour {
		return domain.NewConfigInvalidError("scopeLockTTL", fmt.Sprintf("must be between 5m and 24h, got %v", c.ScopeLockTTL), nil)
	}
	if c.ScopeLockRefreshInterval < 30*time.Second || c.ScopeLockRefreshInterval > 10*time.Minute {
		return domain.NewConfigInvalidError("scopeLockRefreshInterval", fmt.Sprintf("must be between 30s and 10m, got %v", c.ScopeLockRefreshInterval), nil)
	}
	if c.QueueWaitTimeout < 0 || c.QueueWaitTimeout > 30*time.Minute {
		return domain.NewConfigInvalidError("queueWaitTimeout", fmt.Sprintf("must be between 0 and 30m, got %v", c.QueueWaitTimeout), nil)
	}
	return nil
}

func (a *ApprovalLevers) Validate() error {
	if a.ReviewTimeoutDays < 1 || a.ReviewTimeoutDays > 90 {
		return domain.NewConfigInvalidError("reviewTimeoutDays", fmt.Sprintf("must be between 1 and 90, got %d", a.ReviewTimeoutDays), nil)
	}
	return nil
}

func (r *RunnerLevers) Validate() error {
	for _, runnerType := range r.FallbackRunnerTypes {
		if !domain.RunnerType(runnerType).IsValid() {
			return domain.NewConfigInvalidError("fallbackRunnerTypes", fmt.Sprintf("contains invalid runner type: %s", runnerType), nil)
		}
	}
	if r.HealthCheckInterval < 10*time.Second || r.HealthCheckInterval > 5*time.Minute {
		return domain.NewConfigInvalidError("healthCheckInterval", fmt.Sprintf("must be between 10s and 5m, got %v", r.HealthCheckInterval), nil)
	}
	if r.StartupGracePeriod < 0 || r.StartupGracePeriod > 5*time.Minute {
		return domain.NewConfigInvalidError("startupGracePeriod", fmt.Sprintf("must be between 0 and 5m, got %v", r.StartupGracePeriod), nil)
	}
	if r.ProbeTimeout < time.Second || r.ProbeTimeout > 30*time.Second {
		return domain.NewConfigInvalidError("probeTimeout", fmt.Sprintf("must be between 1s and 30s, got %v", r.ProbeTimeout), nil)
	}
	return nil
}

func (s *ServerLevers) Validate() error {
	if s.Port == "" {
		return domain.NewConfigMissingError("port", "value is required", nil)
	}
	if s.ReadTimeout < 5*time.Second || s.ReadTimeout > 5*time.Minute {
		return domain.NewConfigInvalidError("readTimeout", fmt.Sprintf("must be between 5s and 5m, got %v", s.ReadTimeout), nil)
	}
	if s.WriteTimeout < 5*time.Second || s.WriteTimeout > 10*time.Minute {
		return domain.NewConfigInvalidError("writeTimeout", fmt.Sprintf("must be between 5s and 10m, got %v", s.WriteTimeout), nil)
	}
	if s.IdleTimeout < 30*time.Second || s.IdleTimeout > 10*time.Minute {
		return domain.NewConfigInvalidError("idleTimeout", fmt.Sprintf("must be between 30s and 10m, got %v", s.IdleTimeout), nil)
	}
	if s.MaxRequestBodyBytes < 1024 || s.MaxRequestBodyBytes > 100*1024*1024 {
		return domain.NewConfigInvalidError("maxRequestBodyBytes", fmt.Sprintf("must be between 1KB and 100MB, got %d", s.MaxRequestBodyBytes), nil)
	}
	return nil
}

func (s *StorageLevers) Validate() error {
	if s.MaxOpenConns < 5 || s.MaxOpenConns > 100 {
		return domain.NewConfigInvalidError("maxOpenConns", fmt.Sprintf("must be between 5 and 100, got %d", s.MaxOpenConns), nil)
	}
	if s.MaxIdleConns < 1 || s.MaxIdleConns > 50 {
		return domain.NewConfigInvalidError("maxIdleConns", fmt.Sprintf("must be between 1 and 50, got %d", s.MaxIdleConns), nil)
	}
	if s.ConnMaxLifetime < time.Minute || s.ConnMaxLifetime > time.Hour {
		return domain.NewConfigInvalidError("connMaxLifetime", fmt.Sprintf("must be between 1m and 1h, got %v", s.ConnMaxLifetime), nil)
	}
	if s.EventRetentionDays < 1 || s.EventRetentionDays > 365 {
		return domain.NewConfigInvalidError("eventRetentionDays", fmt.Sprintf("must be between 1 and 365, got %d", s.EventRetentionDays), nil)
	}
	if s.ArtifactRetentionDays < 1 || s.ArtifactRetentionDays > 365 {
		return domain.NewConfigInvalidError("artifactRetentionDays", fmt.Sprintf("must be between 1 and 365, got %d", s.ArtifactRetentionDays), nil)
	}
	if s.RunStateRetentionDays < 1 || s.RunStateRetentionDays > 365 {
		return domain.NewConfigInvalidError("runStateRetentionDays", fmt.Sprintf("must be between 1 and 365, got %d", s.RunStateRetentionDays), nil)
	}
	return nil
}

func (h *HeartbeatLevers) Validate() error {
	if h.RunHeartbeatInterval < time.Second || h.RunHeartbeatInterval > 5*time.Minute {
		return domain.NewConfigInvalidError("runHeartbeatInterval", fmt.Sprintf("must be between 1s and 5m, got %v", h.RunHeartbeatInterval), nil)
	}
	if h.CheckpointInterval < 5*time.Second || h.CheckpointInterval > 10*time.Minute {
		return domain.NewConfigInvalidError("checkpointInterval", fmt.Sprintf("must be between 5s and 10m, got %v", h.CheckpointInterval), nil)
	}
	if h.StaleThreshold < time.Minute || h.StaleThreshold > 30*time.Minute {
		return domain.NewConfigInvalidError("staleThreshold", fmt.Sprintf("must be between 1m and 30m, got %v", h.StaleThreshold), nil)
	}
	if h.StaleThreshold <= h.RunHeartbeatInterval {
		return domain.NewConfigInvalidError("staleThreshold", fmt.Sprintf("must exceed runHeartbeatInterval (%v), got %v", h.RunHeartbeatInterval, h.StaleThreshold), nil)
	}
	if h.TeardownTimeout < 5*time.Second || h.TeardownTimeout > 5*time.Minute {
		return domain.NewConfigInvalidError("teardownTimeout", fmt.Sprintf("must be between 5s and 5m, got %v", h.TeardownTimeout), nil)
	}
	if h.MaxRetriesPerPhase < 0 || h.MaxRetriesPerPhase > 10 {
		return domain.NewConfigInvalidError("maxRetriesPerPhase", fmt.Sprintf("must be between 0 and 10, got %d", h.MaxRetriesPerPhase), nil)
	}
	if h.AgentTickInterval < 100*time.Millisecond || h.AgentTickInterval > 10*time.Second {
		return domain.NewConfigInvalidError("agentTickInterval", fmt.Sprintf("must be between 100ms and 10s, got %v", h.AgentTickInterval), nil)
	}
	if h.AgentIdleThreshold < time.Second || h.AgentIdleThreshold > 5*time.Minute {
		return domain.NewConfigInvalidError("agentIdleThreshold", fmt.Sprintf("must be between 1s and 5m, got %v", h.AgentIdleThreshold), nil)
	}
	if h.RunnerSignalGracePeriod < time.Second || h.RunnerSignalGracePeriod > time.Minute {
		return domain.NewConfigInvalidError("runnerSignalGracePeriod", fmt.Sprintf("must be between 1s and 1m, got %v", h.RunnerSignalGracePeriod), nil)
	}
	return nil
}

func (r *RecoveryLevers) Validate() error {
	if r.TranscriptTailInterval < 50*time.Millisecond || r.TranscriptTailInterval > 5*time.Second {
		return domain.NewConfigInvalidError("transcriptTailInterval", fmt.Sprintf("must be between 50ms and 5s, got %v", r.TranscriptTailInterval), nil)
	}
	if r.TranscriptPollInterval < 50*time.Millisecond || r.TranscriptPollInterval > 5*time.Second {
		return domain.NewConfigInvalidError("transcriptPollInterval", fmt.Sprintf("must be between 50ms and 5s, got %v", r.TranscriptPollInterval), nil)
	}
	return nil
}

func (s *ScannerLevers) Validate() error {
	const minBuf = 64 * 1024
	const maxBuf = 64 * 1024 * 1024
	if s.StdoutMaxLineBytes < minBuf || s.StdoutMaxLineBytes > maxBuf {
		return domain.NewConfigInvalidError("stdoutMaxLineBytes", fmt.Sprintf("must be between 64KB and 64MB, got %d", s.StdoutMaxLineBytes), nil)
	}
	if s.TranscriptMaxLineBytes < minBuf || s.TranscriptMaxLineBytes > maxBuf {
		return domain.NewConfigInvalidError("transcriptMaxLineBytes", fmt.Sprintf("must be between 64KB and 64MB, got %d", s.TranscriptMaxLineBytes), nil)
	}
	return nil
}

func (d *DiagnosticsLevers) Validate() error {
	if d.LaunchFailedMaxDuration < 100*time.Millisecond || d.LaunchFailedMaxDuration > 30*time.Second {
		return domain.NewConfigInvalidError("launchFailedMaxDuration", fmt.Sprintf("must be between 100ms and 30s, got %v", d.LaunchFailedMaxDuration), nil)
	}
	if d.RateLimitMessageMaxLen < 64 || d.RateLimitMessageMaxLen > 8192 {
		return domain.NewConfigInvalidError("rateLimitMessageMaxLen", fmt.Sprintf("must be between 64 and 8192, got %d", d.RateLimitMessageMaxLen), nil)
	}
	return nil
}

func (s *SpawnLevers) Validate() error {
	if s.MaxStartingConcurrency < 1 || s.MaxStartingConcurrency > 16 {
		return domain.NewConfigInvalidError("maxStartingConcurrency", fmt.Sprintf("must be between 1 and 16, got %d", s.MaxStartingConcurrency), nil)
	}
	if s.MinSpacing < 0 || s.MinSpacing > 30*time.Second {
		return domain.NewConfigInvalidError("minSpacing", fmt.Sprintf("must be between 0 and 30s, got %v", s.MinSpacing), nil)
	}
	// QueueCapacity == 0 means "auto-derive at orchestrator wiring time".
	if s.QueueCapacity < 0 || s.QueueCapacity > 1024 {
		return domain.NewConfigInvalidError("queueCapacity", fmt.Sprintf("must be 0 (auto) or between 1 and 1024, got %d", s.QueueCapacity), nil)
	}
	if s.QueueCapacity > 0 && s.QueueCapacity < s.MaxStartingConcurrency {
		return domain.NewConfigInvalidError("queueCapacity", fmt.Sprintf("must be >= maxStartingConcurrency (%d), got %d", s.MaxStartingConcurrency, s.QueueCapacity), nil)
	}
	return nil
}

func (s *SandboxLevers) Validate() error {
	if s.AvailabilityCheckTimeout < 100*time.Millisecond || s.AvailabilityCheckTimeout > 10*time.Second {
		return domain.NewConfigInvalidError("availabilityCheckTimeout", fmt.Sprintf("must be between 100ms and 10s, got %v", s.AvailabilityCheckTimeout), nil)
	}
	if s.EnsureStartTimeout < 5*time.Second || s.EnsureStartTimeout > 2*time.Minute {
		return domain.NewConfigInvalidError("ensureStartTimeout", fmt.Sprintf("must be between 5s and 2m, got %v", s.EnsureStartTimeout), nil)
	}
	if s.EnsurePollInterval < 50*time.Millisecond || s.EnsurePollInterval > 5*time.Second {
		return domain.NewConfigInvalidError("ensurePollInterval", fmt.Sprintf("must be between 50ms and 5s, got %v", s.EnsurePollInterval), nil)
	}
	if s.EnsurePollInterval >= s.EnsureStartTimeout {
		return domain.NewConfigInvalidError("ensurePollInterval", fmt.Sprintf("must be less than ensureStartTimeout (%v), got %v", s.EnsureStartTimeout, s.EnsurePollInterval), nil)
	}
	if s.OperationMaxAttempts < 1 || s.OperationMaxAttempts > 8 {
		return domain.NewConfigInvalidError("operationMaxAttempts", fmt.Sprintf("must be between 1 and 8, got %d", s.OperationMaxAttempts), nil)
	}
	if s.OperationInitialBackoff < 25*time.Millisecond || s.OperationInitialBackoff > 5*time.Second {
		return domain.NewConfigInvalidError("operationInitialBackoff", fmt.Sprintf("must be between 25ms and 5s, got %v", s.OperationInitialBackoff), nil)
	}
	if s.OperationMaxBackoff < s.OperationInitialBackoff || s.OperationMaxBackoff > 30*time.Second {
		return domain.NewConfigInvalidError("operationMaxBackoff", fmt.Sprintf("must be between operationInitialBackoff (%v) and 30s, got %v", s.OperationInitialBackoff, s.OperationMaxBackoff), nil)
	}
	return nil
}

func (o *ObservabilityLevers) Validate() error {
	switch o.LogFormat {
	case "text", "json":
	default:
		return domain.NewConfigInvalidError("logFormat", fmt.Sprintf("must be \"text\" or \"json\", got %q", o.LogFormat), nil)
	}
	switch o.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return domain.NewConfigInvalidError("logLevel", fmt.Sprintf("must be one of debug|info|warn|error, got %q", o.LogLevel), nil)
	}
	return nil
}

func wrapConfigSection(section string, err error) error {
	if err == nil {
		return nil
	}
	if cfgErr, ok := err.(*domain.ConfigError); ok {
		if cfgErr.Setting != "" {
			cfgErr.Setting = section + "." + cfgErr.Setting
		} else {
			cfgErr.Setting = section
		}
		return cfgErr
	}
	return domain.NewConfigInvalidError(section, err.Error(), err)
}
