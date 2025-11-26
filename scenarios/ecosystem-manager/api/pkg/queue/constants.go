package queue

import (
	"strings"
	"time"
)

// Task execution timing constraints
const (
	// MinIdleTimeout is the minimum idle timeout duration for task execution
	MinIdleTimeout = 2 * time.Minute

	// MaxIdleTimeoutFactor determines the idle timeout as a fraction of total timeout (defense-in-depth)
	MaxIdleTimeoutFactor = 0.9

	// IdleCheckInterval is how often we check for idle processes
	IdleCheckInterval = 30 * time.Second

	// ProcessTerminationTimeout is how long to wait for graceful process termination
	ProcessTerminationTimeout = 5 * time.Second

	// ProcessKillTimeout is the extended timeout for forced termination
	ProcessKillTimeout = 10 * time.Second
)

// Rate limiting constants
const (
	// DefaultRateLimitRetry is the default pause duration when rate limited (30 minutes)
	DefaultRateLimitRetry = 1800

	// MinRateLimitPause is the minimum rate limit pause duration (5 minutes)
	MinRateLimitPause = 300

	// MaxRateLimitPause is the maximum rate limit pause duration (4 hours)
	MaxRateLimitPause = 14400

	// CriticalRateLimitPause is the pause for critical rate limits (30 minutes)
	CriticalRateLimitPause = 1800

	// RateLimitDetectionWindow is the time window for detecting early rate limit failures
	RateLimitDetectionWindow = 1 * time.Minute
)

// Process cleanup constants
const (
	// AgentShutdownTimeout is how long to wait for an agent to shut down
	AgentShutdownTimeout = 5 * time.Second

	// AgentShutdownPollInterval is how often to check if an agent has shut down
	AgentShutdownPollInterval = 200 * time.Millisecond

	// ProcessTermRetryDelay is how long to wait before retrying process termination
	ProcessTermRetryDelay = 200 * time.Millisecond

	// ProcessCleanupDelay is how long to wait after killing a process
	ProcessCleanupDelay = 100 * time.Millisecond

	// AgentCleanupRetryDelay is how long to wait before retrying agent cleanup
	AgentCleanupRetryDelay = 500 * time.Millisecond

	// AgentRegistryTimeout is the timeout for agent registry operations (list, stop, cleanup)
	AgentRegistryTimeout = 10 * time.Second

	// AgentStopTimeout is the timeout for stopping a single agent
	AgentStopTimeout = 5 * time.Second
)

// Task reconciliation constants
const (
	// InitialReconcileDelay is how long to wait before initial in-progress reconciliation
	InitialReconcileDelay = 2 * time.Second

	// ProcessLoopSafetyInterval controls how often the processor self-wakes
	ProcessLoopSafetyInterval = 30 * time.Second

	// TimeoutWatchdogInterval is how frequently the watchdog scans for timed-out tasks
	TimeoutWatchdogInterval = 30 * time.Second

	// MaxTurnsCleanupDelay is the additional wait after MAX_TURNS before cleanup
	MaxTurnsCleanupDelay = 500 * time.Millisecond
)

// Retry and backoff constants
const (
	// MaxAgentCleanupRetries is the maximum number of retry attempts for agent cleanup
	MaxAgentCleanupRetries = 3

	// AgentCleanupBackoffBase is the base delay for exponential backoff in agent cleanup retries
	AgentCleanupBackoffBase = 500 * time.Millisecond
)

// Cache and history constants
const (
	// ExecutionHistoryCacheTTL is how long to cache execution history before reloading from disk
	ExecutionHistoryCacheTTL = 10 * time.Second
)

// Task log buffer constants
const (
	// MaxTaskLogEntries is the maximum number of log entries to keep per task
	MaxTaskLogEntries = 2000

	// TaskLogBufferInitialCapacity is the initial capacity for task log buffers
	TaskLogBufferInitialCapacity = 64

	// ScannerBufferSize is the buffer size for log scanners
	ScannerBufferSize = 1024

	// ScannerMaxTokenSize is the maximum token size for log scanners
	ScannerMaxTokenSize = 1024 * 1024

	// PipeClosureTimeout is the maximum time to wait for pipe readers to finish after process exit
	// This prevents indefinite hangs when pipes don't close cleanly
	PipeClosureTimeout = 5 * time.Second
)

// File and size calculation constants
const (
	// PromptFilePermissions is the file permission mode for saved prompt files
	PromptFilePermissions = 0644

	// BytesPerKilobyte is the number of bytes in a kilobyte
	BytesPerKilobyte = 1024.0

	// KilobytesPerMegabyte is the number of kilobytes in a megabyte
	KilobytesPerMegabyte = 1024.0
)

// HTTP status codes for error detection
const (
	// HTTPStatusTooManyRequests indicates a rate limit error (HTTP 429)
	HTTPStatusTooManyRequests = 429
)

// External CLI commands
const (
	// ClaudeCodeResourceCommand is the command to invoke the Claude Code resource
	ClaudeCodeResourceCommand = "resource-claude-code"
)

// Agent naming
const (
	// AgentTagPrefix is the prefix used for all Claude Code agent tags
	AgentTagPrefix = "ecosystem-"

	// PromptFilePrefix is the prefix for temporary prompt files in /tmp
	PromptFilePrefix = "ecosystem-prompt-"

	// RecyclerPromptPrefix is the prefix for recycler prompt IDs
	RecyclerPromptPrefix = "ecosystem-recycler-"
)

// makeAgentTag creates an agent tag from a task ID
func makeAgentTag(taskID string) string {
	return AgentTagPrefix + taskID
}

// parseAgentTag extracts the task ID from an agent tag
func parseAgentTag(agentTag string) string {
	return strings.TrimPrefix(agentTag, AgentTagPrefix)
}

// isValidAgentTag checks if a tag has the correct agent prefix
func isValidAgentTag(tag string) bool {
	return strings.HasPrefix(tag, AgentTagPrefix)
}
