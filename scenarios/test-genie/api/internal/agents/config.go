// Package agents provides agent lifecycle management with configurable behavior.
package agents

import (
	"time"

	"test-genie/internal/shared"
)

// Config holds tunable levers for agent behavior.
// These control operational tradeoffs like lock duration vs. responsiveness,
// resource limits vs. flexibility, and retention vs. storage.
//
// # Control Surface Overview
//
// This config exposes the agent management control surface. Levers are grouped by concern:
//
//   - **Locking & Coordination**: How agents coordinate access to shared resources
//   - **Execution Defaults**: Default limits for individual agent runs (can be overridden per-spawn)
//   - **Spawn Limits**: Bounds on batch operations and parallelism
//   - **Retention & Cleanup**: How long data is kept and how often cleanup runs
//   - **Session Management**: Idempotency and duplicate prevention
//
// # Environment Variables
//
// All levers can be overridden via environment variables without code changes:
//
// Locking & Coordination:
//   - AGENT_LOCK_TIMEOUT_MINUTES: How long scope locks last (default: 20)
//   - AGENT_HEARTBEAT_INTERVAL_MINUTES: How often locks renew (default: derived from lock timeout)
//   - AGENT_MAX_HEARTBEAT_FAILURES: Consecutive failures before stop (default: 3)
//
// Execution Defaults:
//   - AGENT_DEFAULT_TIMEOUT_SECONDS: Default per-agent timeout (default: 900 = 15min)
//   - AGENT_DEFAULT_MAX_TURNS: Default max conversation turns (default: 50)
//   - AGENT_DEFAULT_MAX_FILES: Default max files an agent can modify (default: 50)
//   - AGENT_DEFAULT_MAX_BYTES: Default max bytes an agent can write (default: 1048576 = 1MB)
//   - AGENT_DEFAULT_NETWORK_ENABLED: Default network access (default: false)
//
// Spawn Limits:
//   - AGENT_MAX_PROMPTS: Max prompts per spawn request (default: 20)
//   - AGENT_MAX_CONCURRENT: Max parallel agents per spawn (default: 10)
//   - AGENT_DEFAULT_CONCURRENCY: Default concurrency if not specified (default: 3)
//
// Retention & Cleanup:
//   - AGENT_RETENTION_DAYS: Days to keep completed agents (default: 7)
//   - AGENT_CLEANUP_INTERVAL_MINUTES: Cleanup frequency (default: 60)
//
// Session Management:
//   - AGENT_SPAWN_SESSION_TTL_MINUTES: Session duration (default: 30)
//   - AGENT_IDEMPOTENCY_TTL_MINUTES: Idempotency key cache duration (default: 30)
type Config struct {
	// --- Locking & Coordination ---

	// LockTimeoutMinutes controls how long a scope lock remains valid before expiring.
	// Higher = agents can run longer without heartbeat, but stale locks block others longer.
	// Lower = faster recovery from failed agents, but agents must heartbeat more reliably.
	// Range: 5-120 minutes. Default: 20 minutes.
	LockTimeoutMinutes int

	// HeartbeatIntervalMinutes controls how often agents renew their locks.
	// Must be significantly less than LockTimeoutMinutes (typically 1/4 to 1/3).
	// Lower = more network overhead but faster lock recovery on failures.
	// Default: computed as LockTimeoutMinutes / 4 (min 1 minute).
	HeartbeatIntervalMinutes int

	// MaxHeartbeatFailures controls how many consecutive heartbeat failures
	// before an agent is automatically stopped to release locks.
	// Higher = more tolerant of transient failures, but may hold locks longer.
	// Range: 1-10. Default: 3.
	MaxHeartbeatFailures int

	// --- Execution Defaults ---
	// These are system-wide defaults for per-agent execution. Clients can override
	// these on a per-spawn basis, but these serve as the baseline when not specified.

	// DefaultTimeoutSeconds is the default timeout for agent execution.
	// Higher = agents can run longer tasks, but stalled agents block longer.
	// Lower = faster failure detection, but may timeout legitimate long tasks.
	// Range: 60-3600 seconds (1 min to 1 hour). Default: 900 seconds (15 minutes).
	DefaultTimeoutSeconds int

	// DefaultMaxTurns is the default max conversation turns per agent.
	// Higher = agents can have longer conversations, handle complex tasks.
	// Lower = agents complete faster, prevent runaway conversations.
	// Range: 5-200 turns. Default: 50 turns.
	DefaultMaxTurns int

	// DefaultMaxFilesChanged is the default limit on files an agent can modify per run.
	// Higher = agents can handle larger refactoring tasks.
	// Lower = limits blast radius of agent mistakes.
	// Range: 1-500 files. Default: 50 files.
	DefaultMaxFilesChanged int

	// DefaultMaxBytesWritten is the default limit on total bytes written per run.
	// Higher = agents can generate more content (e.g., large test suites).
	// Lower = limits accidental large file generation.
	// Range: 1024-104857600 bytes (1KB-100MB). Default: 1048576 bytes (1MB).
	DefaultMaxBytesWritten int64

	// DefaultNetworkEnabled controls whether agents have network access by default.
	// true = agents can make HTTP requests (needed for API testing, fetching docs).
	// false = agents are network-isolated (safer but more limited).
	// Default: false (secure by default).
	DefaultNetworkEnabled bool

	// --- Retention & Cleanup ---

	// RetentionDays controls how long completed agent records are kept.
	// Higher = more history for debugging/auditing, but more storage.
	// Lower = faster queries, less storage, but less history.
	// Range: 1-365 days. Default: 7 days.
	RetentionDays int

	// CleanupIntervalMinutes controls how often the background cleanup runs.
	// Higher = less CPU overhead, but stale data persists longer.
	// Lower = cleaner database, but more frequent queries.
	// Range: 15-1440 minutes. Default: 60 minutes.
	CleanupIntervalMinutes int

	// --- Spawn Limits ---

	// MaxPromptsPerSpawn limits how many prompts can be in a single spawn request.
	// Higher = more flexibility for batch operations.
	// Lower = prevents excessive resource consumption from single requests.
	// Range: 1-100. Default: 20.
	MaxPromptsPerSpawn int

	// MaxConcurrentAgents limits parallel agent execution per spawn request.
	// Higher = faster batch completion but more resource pressure.
	// Lower = gentler on resources but slower completion.
	// Range: 1-20. Default: 10.
	MaxConcurrentAgents int

	// DefaultConcurrency is used when client doesn't specify concurrency.
	// Should balance between responsiveness and resource consumption.
	// Range: 1-MaxConcurrentAgents. Default: 3.
	DefaultConcurrency int

	// --- Session Management ---

	// SpawnSessionTTLMinutes controls how long spawn sessions last.
	// This prevents duplicate spawns from the same user/browser.
	// Higher = better protection against duplicates across longer tasks.
	// Lower = faster expiry allows re-spawning sooner after network issues.
	// Range: 5-480 minutes. Default: 30 minutes.
	SpawnSessionTTLMinutes int

	// IdempotencyTTLMinutes controls how long idempotency keys are cached.
	// Must be >= SpawnSessionTTLMinutes for consistency.
	// Range: 5-480 minutes. Default: 30 minutes.
	IdempotencyTTLMinutes int
}

// DefaultConfig returns a Config with production-ready defaults.
func DefaultConfig() Config {
	return Config{
		// Locking & Coordination
		LockTimeoutMinutes:       20,
		HeartbeatIntervalMinutes: 5,
		MaxHeartbeatFailures:     3,
		// Execution Defaults
		DefaultTimeoutSeconds:  900,         // 15 minutes
		DefaultMaxTurns:        50,          // 50 conversation turns
		DefaultMaxFilesChanged: 50,          // 50 files
		DefaultMaxBytesWritten: 1024 * 1024, // 1MB
		DefaultNetworkEnabled:  false,       // Network isolated by default
		// Retention & Cleanup
		RetentionDays:          7,
		CleanupIntervalMinutes: 60,
		// Spawn Limits
		MaxPromptsPerSpawn:  20,
		MaxConcurrentAgents: 10,
		DefaultConcurrency:  3,
		// Session Management
		SpawnSessionTTLMinutes: 30,
		IdempotencyTTLMinutes:  30,
	}
}

// LoadConfigFromEnv loads configuration from environment variables,
// falling back to defaults for unset values.
func LoadConfigFromEnv() Config {
	cfg := DefaultConfig()

	// --- Locking & Coordination ---
	cfg.LockTimeoutMinutes = shared.ClampInt(shared.EnvInt("AGENT_LOCK_TIMEOUT_MINUTES", cfg.LockTimeoutMinutes), 5, 120)

	// HeartbeatInterval: use explicit value or derive from lock timeout
	if v := shared.EnvInt("AGENT_HEARTBEAT_INTERVAL_MINUTES", 0); v > 0 {
		cfg.HeartbeatIntervalMinutes = shared.ClampInt(v, 1, cfg.LockTimeoutMinutes/2)
	} else {
		// Derive from lock timeout: 1/4 of lock timeout, minimum 1 minute
		derived := cfg.LockTimeoutMinutes / 4
		if derived < 1 {
			derived = 1
		}
		cfg.HeartbeatIntervalMinutes = derived
	}

	cfg.MaxHeartbeatFailures = shared.ClampInt(shared.EnvInt("AGENT_MAX_HEARTBEAT_FAILURES", cfg.MaxHeartbeatFailures), 1, 10)

	// --- Execution Defaults ---
	cfg.DefaultTimeoutSeconds = shared.ClampInt(shared.EnvInt("AGENT_DEFAULT_TIMEOUT_SECONDS", cfg.DefaultTimeoutSeconds), 60, 3600)
	cfg.DefaultMaxTurns = shared.ClampInt(shared.EnvInt("AGENT_DEFAULT_MAX_TURNS", cfg.DefaultMaxTurns), 5, 200)
	cfg.DefaultMaxFilesChanged = shared.ClampInt(shared.EnvInt("AGENT_DEFAULT_MAX_FILES", cfg.DefaultMaxFilesChanged), 1, 500)
	cfg.DefaultMaxBytesWritten = shared.ClampInt64(shared.EnvInt64("AGENT_DEFAULT_MAX_BYTES", cfg.DefaultMaxBytesWritten), 1024, 104857600)
	cfg.DefaultNetworkEnabled = shared.EnvBool("AGENT_DEFAULT_NETWORK_ENABLED", cfg.DefaultNetworkEnabled)

	// --- Retention & Cleanup ---
	cfg.RetentionDays = shared.ClampInt(shared.EnvInt("AGENT_RETENTION_DAYS", cfg.RetentionDays), 1, 365)
	cfg.CleanupIntervalMinutes = shared.ClampInt(shared.EnvInt("AGENT_CLEANUP_INTERVAL_MINUTES", cfg.CleanupIntervalMinutes), 15, 1440)

	// --- Spawn Limits ---
	cfg.MaxPromptsPerSpawn = shared.ClampInt(shared.EnvInt("AGENT_MAX_PROMPTS", cfg.MaxPromptsPerSpawn), 1, 100)
	cfg.MaxConcurrentAgents = shared.ClampInt(shared.EnvInt("AGENT_MAX_CONCURRENT", cfg.MaxConcurrentAgents), 1, 20)
	cfg.DefaultConcurrency = shared.ClampInt(shared.EnvInt("AGENT_DEFAULT_CONCURRENCY", cfg.DefaultConcurrency), 1, cfg.MaxConcurrentAgents)

	// --- Session Management ---
	cfg.SpawnSessionTTLMinutes = shared.ClampInt(shared.EnvInt("AGENT_SPAWN_SESSION_TTL_MINUTES", cfg.SpawnSessionTTLMinutes), 5, 480)
	cfg.IdempotencyTTLMinutes = shared.ClampInt(shared.EnvInt("AGENT_IDEMPOTENCY_TTL_MINUTES", cfg.IdempotencyTTLMinutes), 5, 480)

	return cfg
}

// LockTimeout returns the lock timeout as a duration.
func (c Config) LockTimeout() time.Duration {
	return time.Duration(c.LockTimeoutMinutes) * time.Minute
}

// HeartbeatInterval returns the heartbeat interval as a duration.
func (c Config) HeartbeatInterval() time.Duration {
	return time.Duration(c.HeartbeatIntervalMinutes) * time.Minute
}

// CleanupInterval returns the cleanup interval as a duration.
func (c Config) CleanupInterval() time.Duration {
	return time.Duration(c.CleanupIntervalMinutes) * time.Minute
}

// RetentionDuration returns the retention period as a duration.
func (c Config) RetentionDuration() time.Duration {
	return time.Duration(c.RetentionDays) * 24 * time.Hour
}

// SpawnSessionTTL returns the spawn session TTL as a duration.
func (c Config) SpawnSessionTTL() time.Duration {
	return time.Duration(c.SpawnSessionTTLMinutes) * time.Minute
}

// IdempotencyTTL returns the idempotency TTL as a duration.
func (c Config) IdempotencyTTL() time.Duration {
	return time.Duration(c.IdempotencyTTLMinutes) * time.Minute
}

// DefaultTimeout returns the default execution timeout as a duration.
func (c Config) DefaultTimeout() time.Duration {
	return time.Duration(c.DefaultTimeoutSeconds) * time.Second
}

// ValidationResult captures what adjustments were made during config loading.
// This provides visibility into implicit changes operators should be aware of.
type ValidationResult struct {
	// Adjustments contains human-readable descriptions of any clamping or corrections.
	Adjustments []string
	// Warnings contains non-fatal issues operators should be aware of.
	Warnings []string
}

// HasChanges returns true if any adjustments or warnings were recorded.
func (r ValidationResult) HasChanges() bool {
	return len(r.Adjustments) > 0 || len(r.Warnings) > 0
}

// Validate ensures the configuration is internally consistent.
// Returns warnings about any automatic corrections made.
func (c Config) Validate() error {
	// HeartbeatInterval must be less than half of LockTimeout
	// to ensure at least one renewal before expiry
	if c.HeartbeatIntervalMinutes >= c.LockTimeoutMinutes/2 {
		// Auto-correct silently - this is a soft constraint
		c.HeartbeatIntervalMinutes = c.LockTimeoutMinutes / 4
		if c.HeartbeatIntervalMinutes < 1 {
			c.HeartbeatIntervalMinutes = 1
		}
	}

	// DefaultConcurrency must not exceed MaxConcurrentAgents
	if c.DefaultConcurrency > c.MaxConcurrentAgents {
		c.DefaultConcurrency = c.MaxConcurrentAgents
	}

	return nil
}

// ValidateWithReport checks configuration and reports any adjustments made.
// This provides visibility into implicit clamping or corrections.
func (c Config) ValidateWithReport() ValidationResult {
	result := ValidationResult{}

	// Check for potential configuration issues
	if c.HeartbeatIntervalMinutes >= c.LockTimeoutMinutes/2 {
		result.Warnings = append(result.Warnings,
			"HeartbeatIntervalMinutes should be less than half of LockTimeoutMinutes for reliable lock renewal")
	}

	if c.DefaultConcurrency > c.MaxConcurrentAgents {
		result.Warnings = append(result.Warnings,
			"DefaultConcurrency was reduced to match MaxConcurrentAgents")
	}

	// Check for non-default security settings
	if c.DefaultNetworkEnabled {
		result.Warnings = append(result.Warnings,
			"Network access is enabled by default. Agents can make outbound requests.")
	}

	// Check for very permissive settings
	if c.DefaultMaxFilesChanged > 200 {
		result.Warnings = append(result.Warnings,
			"DefaultMaxFilesChanged is very high (>200). Agents can modify many files.")
	}

	if c.DefaultMaxBytesWritten > 10*1024*1024 {
		result.Warnings = append(result.Warnings,
			"DefaultMaxBytesWritten is very high (>10MB). Agents can write large amounts of data.")
	}

	return result
}
