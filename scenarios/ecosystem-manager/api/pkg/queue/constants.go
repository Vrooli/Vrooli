package queue

import (
	"strings"
	"time"
)

// Task execution timing constraints
const (
	// MinIdleTimeout is the minimum idle timeout duration for task execution
	MinIdleTimeout = 2 * time.Minute

	// MaxIdleTimeoutFactor determines the idle timeout as a fraction of total timeout
	MaxIdleTimeoutFactor = 0.5 // Half of total timeout

	// DefaultIdleTimeoutCap is the maximum idle timeout duration
	DefaultIdleTimeoutCap = 5 * time.Minute

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
)

// Task reconciliation constants
const (
	// InitialReconcileDelay is how long to wait before initial in-progress reconciliation
	InitialReconcileDelay = 2 * time.Second
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
)

// makeAgentTag creates an agent tag from a task ID
func makeAgentTag(taskID string) string {
	return AgentTagPrefix + taskID
}

// parseAgentTag extracts the task ID from an agent tag
func parseAgentTag(agentTag string) string {
	return strings.TrimPrefix(agentTag, AgentTagPrefix)
}
