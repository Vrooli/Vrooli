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
	// DatabaseURL is the PostgreSQL connection string.
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
}

// =============================================================================
// DEFAULTS
// =============================================================================

// DefaultLevers returns the default configuration that works for most use cases.
func DefaultLevers() Levers {
	return Levers{
		Execution: ExecutionLevers{
			DefaultTimeout:     30 * time.Minute,
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
