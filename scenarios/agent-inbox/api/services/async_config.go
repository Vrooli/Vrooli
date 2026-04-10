// Package services provides application services for the Agent Inbox scenario.
//
// This file defines configuration constants and tunable levers for the async
// tracking system. These values control polling behavior, timeouts, buffer sizes,
// and cleanup intervals.
//
// CONTROL SURFACE DESIGN:
// These levers are designed to be safe defaults that work for most use cases.
// They can be adjusted based on operational needs without code changes by
// extracting them to environment variables or configuration files.
package services

import "time"

// =============================================================================
// Polling Configuration
// =============================================================================
//
// These constants control how the async tracker polls for status updates.
// Polling is used when tools return an operation ID that must be checked
// periodically for completion.

const (
	// DefaultPollInterval is the time between status checks when a tool's
	// AsyncBehavior doesn't specify a custom interval.
	//
	// Trade-off: Shorter = more responsive but higher load on external services.
	// Recommendation: 5s is a good balance for most background operations.
	DefaultPollInterval = 5 * time.Second

	// MinPollInterval is the minimum allowed polling interval.
	// Prevents misconfigured tools from overwhelming external services.
	MinPollInterval = 1 * time.Second

	// DefaultMaxPollDuration is how long the tracker will poll before timing out
	// when a tool's AsyncBehavior doesn't specify a custom duration.
	//
	// Trade-off: Longer = handles slow operations but holds resources longer.
	// Recommendation: 1 hour covers most use cases; tools can override.
	DefaultMaxPollDuration = 1 * time.Hour
)

// =============================================================================
// Channel Buffer Sizes
// =============================================================================
//
// These constants control buffering for async notification channels.
// Proper sizing prevents blocking while avoiding excessive memory use.

const (
	// SubscriberChannelBufferSize is the buffer size for SSE subscriber channels.
	// Each subscriber (UI client) gets a channel with this capacity.
	//
	// Trade-off: Larger = tolerates slower consumers but uses more memory.
	// If full, updates are dropped (non-blocking send).
	SubscriberChannelBufferSize = 100

	// CompletionCallbackBufferSize is the buffer size for completion notification
	// channels used by the AI conversation loop.
	//
	// Trade-off: Sized for concurrent async operations in a single chat.
	// Most chats have <10 concurrent async operations.
	CompletionCallbackBufferSize = 10
)

// =============================================================================
// Cleanup Configuration
// =============================================================================
//
// These constants control automatic cleanup of completed operations.
// Cleanup prevents memory growth from accumulating completed operations.

const (
	// DefaultCleanupInterval is how often the cleanup routine runs.
	// More frequent = faster memory reclamation but more CPU overhead.
	DefaultCleanupInterval = 5 * time.Minute

	// DefaultCleanupRetention is how long completed operations are kept
	// before being eligible for cleanup.
	//
	// Trade-off: Longer retention allows late status queries but uses memory.
	// 30 minutes covers most UI refresh scenarios.
	DefaultCleanupRetention = 30 * time.Minute
)

// =============================================================================
// Auto-Continue Configuration
// =============================================================================
//
// These constants control the automatic continuation loop in streaming
// completions, where tool calls trigger follow-up AI requests.

const (
	// MaxAutoContineIterations limits the tool call -> response -> tool call
	// loop to prevent infinite cycles.
	//
	// Trade-off: Higher allows complex multi-step tool workflows, but
	// risks runaway behavior if AI keeps calling tools.
	MaxAutoContinueIterations = 10
)

// =============================================================================
// Streaming Configuration
// =============================================================================
//
// These constants control SSE streaming behavior.

const (
	// MaxSSEScanTokenSize is the maximum size for SSE line scanning.
	// Large enough to handle base64-encoded images in responses.
	//
	// Trade-off: Larger allows bigger payloads but uses more memory per request.
	// 16MB handles most image generation responses.
	MaxSSEScanTokenSize = 16 * 1024 * 1024 // 16 MB
)

// =============================================================================
// Async Operation Status Values
// =============================================================================
//
// These are the standard status values for async operations.
// Used for consistent status reporting across the system.

const (
	// AsyncStatusPending indicates the operation has been submitted but
	// polling has not yet received a status update.
	AsyncStatusPending = "pending"

	// AsyncStatusRunning indicates the operation is actively executing.
	AsyncStatusRunning = "running"

	// AsyncStatusCompleted indicates successful completion.
	AsyncStatusCompleted = "completed"

	// AsyncStatusFailed indicates the operation failed with an error.
	AsyncStatusFailed = "failed"

	// AsyncStatusTimeout indicates polling exceeded MaxPollDuration.
	AsyncStatusTimeout = "timeout"

	// AsyncStatusCancelled indicates the operation was cancelled by user.
	AsyncStatusCancelled = "cancelled"
)

// IsTerminalStatus returns true if the status indicates the operation
// has finished (successfully or not).
func IsTerminalStatus(status string) bool {
	switch status {
	case AsyncStatusCompleted, AsyncStatusFailed, AsyncStatusTimeout, AsyncStatusCancelled:
		return true
	default:
		return false
	}
}
