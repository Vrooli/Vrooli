// Package config provides a unified control surface for the browser-automation-studio API.
//
// # Control Surface Design
//
// This file defines tunable levers grouped by operational concern:
//   - Timeouts: Request and operation timeouts (the main reliability vs speed tradeoff)
//   - Database: Connection pooling and retry behavior
//   - Execution: Workflow execution timeout calculation
//   - WebSocket: Client connection and buffer settings
//   - Events: Automation event backpressure limits
//   - Recording: Record mode session settings
//   - AI/DOM: AI analysis and DOM extraction limits
//   - Replay: Video rendering and capture settings
//   - Storage: Screenshot and artifact storage limits
//   - HTTP: API request limits and pagination defaults
//   - Entitlement: Subscription verification and feature gating
//   - Performance: Debug performance mode for frame streaming
//
// # Adding New Levers
//
// 1. Add to appropriate group in Config struct
// 2. Add environment variable parsing in loadFromEnv()
// 3. Add validation in Validate()
// 4. Document the tradeoff in comments
// 5. Set a sane default that works for common usage
//
// # Usage
//
//	cfg := config.Load()
//	timeout := cfg.Timeouts.DefaultRequest
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
)

// Config holds all tunable configuration for the browser-automation-studio API.
// Values can be overridden via environment variables.
type Config struct {
	// Timeouts control how long operations wait before failing.
	// Tradeoff: Higher = more tolerant of slow operations, longer waits on failures.
	// Lower = faster failure detection, may fail on slow networks/pages.
	Timeouts TimeoutsConfig

	// Database controls connection pooling and retry behavior.
	Database DatabaseConfig

	// Execution controls workflow execution timeout calculation.
	Execution ExecutionConfig

	// WebSocket controls client connection and buffer settings.
	WebSocket WebSocketConfig

	// Events controls automation event backpressure limits.
	Events EventsConfig

	// Recording controls record mode session settings.
	Recording RecordingConfig

	// AI controls AI analysis and DOM extraction limits.
	AI AIConfig

	// Replay controls video rendering and capture settings.
	Replay ReplayConfig

	// Storage controls screenshot and artifact storage limits.
	Storage StorageConfig

	// ArtifactLimits controls size limits for collected artifacts.
	// These are global defaults that can be overridden per-execution.
	ArtifactLimits ArtifactLimitsConfig

	// HTTP controls API request limits and pagination defaults.
	HTTP HTTPConfig

	// Entitlement controls subscription verification and feature gating.
	Entitlement EntitlementConfig

	// Performance controls debug performance mode for frame streaming diagnostics.
	// When enabled, detailed timing data is collected and exposed for bottleneck analysis.
	Performance PerformanceConfig
}

// TimeoutsConfig groups timeout-related settings.
// These control the reliability vs speed tradeoff across the API.
type TimeoutsConfig struct {
	// DefaultRequest is used for standard CRUD operations that don't require
	// complex processing or external service calls.
	// Env: BAS_TIMEOUT_DEFAULT_REQUEST_MS (default: 5000)
	DefaultRequest time.Duration

	// ExtendedRequest is used for operations that may take longer but don't
	// involve AI processing (e.g., bulk operations, complex queries).
	// Env: BAS_TIMEOUT_EXTENDED_REQUEST_MS (default: 10000)
	ExtendedRequest time.Duration

	// AIRequest is used for AI-powered operations like workflow generation
	// and modification, which require inference from external AI services.
	// Env: BAS_TIMEOUT_AI_REQUEST_MS (default: 45000)
	AIRequest time.Duration

	// AIAnalysis is used for complex AI analysis operations that process
	// screenshots, DOM trees, and require multi-step inference.
	// Env: BAS_TIMEOUT_AI_ANALYSIS_MS (default: 60000)
	AIAnalysis time.Duration

	// ElementAnalysis is used for element analysis operations that involve
	// screenshot capture and DOM processing.
	// Env: BAS_TIMEOUT_ELEMENT_ANALYSIS_MS (default: 30000)
	ElementAnalysis time.Duration

	// ExecutionCompletion bounds how long the API will wait when callers
	// set wait_for_completion=true on execution endpoints.
	// Env: BAS_TIMEOUT_EXECUTION_COMPLETION_MS (default: 120000)
	ExecutionCompletion time.Duration

	// DatabasePing is used for health checks and connection verification.
	// Env: BAS_TIMEOUT_DATABASE_PING_MS (default: 2000)
	DatabasePing time.Duration

	// DatabaseQuery is used for standard database queries.
	// Env: BAS_TIMEOUT_DATABASE_QUERY_MS (default: 5000)
	DatabaseQuery time.Duration

	// DatabaseMigration is used for schema migrations which may take longer.
	// Env: BAS_TIMEOUT_DATABASE_MIGRATION_MS (default: 30000)
	DatabaseMigration time.Duration

	// BrowserEngineHealth is used for browser engine health check requests.
	// Env: BAS_TIMEOUT_BROWSER_ENGINE_HEALTH_MS (default: 5000)
	BrowserEngineHealth time.Duration

	// GlobalRequest is the overall request timeout for the HTTP server.
	// Env: BAS_TIMEOUT_GLOBAL_REQUEST_MS (default: 900000, 15 minutes)
	GlobalRequest time.Duration

	// RecordModeSession is used for record mode session operations.
	// Env: BAS_TIMEOUT_RECORD_MODE_SESSION_MS (default: 30000)
	RecordModeSession time.Duration

	// StartupHealthCheck is used for startup health check operations.
	// Env: BAS_TIMEOUT_STARTUP_HEALTH_CHECK_MS (default: 10000)
	StartupHealthCheck time.Duration
}

// DatabaseConfig controls database connection and retry behavior.
type DatabaseConfig struct {
	// MaxOpenConns is the maximum number of open connections to the database.
	// Env: BAS_DB_MAX_OPEN_CONNS (default: 25)
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections in the pool.
	// Env: BAS_DB_MAX_IDLE_CONNS (default: 5)
	MaxIdleConns int

	// ConnMaxLifetime is how long a connection can be reused.
	// Env: BAS_DB_CONN_MAX_LIFETIME_MS (default: 300000, 5 minutes)
	ConnMaxLifetime time.Duration

	// MaxRetries is the maximum number of connection retry attempts.
	// Env: BAS_DB_MAX_RETRIES (default: 10)
	MaxRetries int

	// BaseRetryDelay is the initial delay between retry attempts.
	// Env: BAS_DB_BASE_RETRY_DELAY_MS (default: 1000)
	BaseRetryDelay time.Duration

	// MaxRetryDelay is the maximum delay between retry attempts.
	// Env: BAS_DB_MAX_RETRY_DELAY_MS (default: 30000)
	MaxRetryDelay time.Duration

	// RetryJitterFactor controls the random jitter added to retry delays (0-1).
	// Higher = more variance, helps prevent thundering herd.
	// Env: BAS_DB_RETRY_JITTER_FACTOR (default: 0.25)
	RetryJitterFactor float64
}

// ExecutionConfig controls workflow execution timeout calculation.
// Formula: base + (stepCount * perStep), clamped to [min, max]
type ExecutionConfig struct {
	// BaseTimeout is the minimum overhead for session setup, network round-trips, etc.
	// Env: BAS_EXECUTION_BASE_TIMEOUT_MS (default: 30000)
	BaseTimeout time.Duration

	// PerStepTimeout is the additional time allocated per instruction in a simple workflow.
	// Env: BAS_EXECUTION_PER_STEP_TIMEOUT_MS (default: 10000)
	PerStepTimeout time.Duration

	// PerStepSubflowTimeout increases per-step allocation for workflows with nested execution.
	// Env: BAS_EXECUTION_PER_STEP_SUBFLOW_TIMEOUT_MS (default: 15000)
	PerStepSubflowTimeout time.Duration

	// MinTimeout prevents unreasonably short timeouts for small workflows.
	// Env: BAS_EXECUTION_MIN_TIMEOUT_MS (default: 90000)
	MinTimeout time.Duration

	// MaxTimeout prevents timeouts from exceeding the HTTP client timeout.
	// Env: BAS_EXECUTION_MAX_TIMEOUT_MS (default: 270000, 4.5 minutes)
	MaxTimeout time.Duration

	// MaxSubflowDepth limits recursion depth for nested subflows.
	// Env: BAS_EXECUTION_MAX_SUBFLOW_DEPTH (default: 5)
	MaxSubflowDepth int

	// DefaultEntryTimeout is the default timeout for entry selector probes (ms).
	// Env: BAS_EXECUTION_DEFAULT_ENTRY_TIMEOUT_MS (default: 3000)
	DefaultEntryTimeout time.Duration

	// MinEntryTimeout is the minimum timeout for entry selector probes (ms).
	// Env: BAS_EXECUTION_MIN_ENTRY_TIMEOUT_MS (default: 250)
	MinEntryTimeout time.Duration

	// DefaultRetryDelay is the initial delay between retry attempts.
	// Env: BAS_EXECUTION_DEFAULT_RETRY_DELAY_MS (default: 750)
	DefaultRetryDelay time.Duration

	// DefaultBackoffFactor is the multiplier for exponential backoff on retries.
	// Env: BAS_EXECUTION_DEFAULT_BACKOFF_FACTOR (default: 1.5)
	DefaultBackoffFactor float64

	// CompletionPollInterval is how often to poll for execution completion.
	// Env: BAS_EXECUTION_COMPLETION_POLL_INTERVAL_MS (default: 250)
	CompletionPollInterval time.Duration

	// AdhocCleanupInterval is how often to check for expired adhoc executions.
	// Env: BAS_EXECUTION_ADHOC_CLEANUP_INTERVAL_MS (default: 5000)
	AdhocCleanupInterval time.Duration

	// AdhocRetentionPeriod is how long to retain completed adhoc workflows before cleanup.
	// Env: BAS_EXECUTION_ADHOC_RETENTION_PERIOD_MS (default: 600000, 10 minutes)
	AdhocRetentionPeriod time.Duration

	// HeartbeatInterval controls the cadence of mid-step heartbeats emitted while
	// instructions are running. Set to 0 to disable heartbeats (not recommended
	// unless running in extremely constrained environments).
	// Env: BAS_EXECUTION_HEARTBEAT_INTERVAL_MS (default: 2000)
	HeartbeatInterval time.Duration
}

// WebSocketConfig controls WebSocket hub settings.
type WebSocketConfig struct {
	// ClientSendBufferSize is the channel buffer size for outgoing JSON messages per client.
	// Env: BAS_WS_CLIENT_SEND_BUFFER_SIZE (default: 256)
	ClientSendBufferSize int

	// ClientBinaryBufferSize is the channel buffer size for binary frame data per client.
	// Holds ~2 seconds at 60 FPS.
	// Env: BAS_WS_CLIENT_BINARY_BUFFER_SIZE (default: 120)
	ClientBinaryBufferSize int

	// ClientReadLimit is the maximum message size the client can send.
	// Env: BAS_WS_CLIENT_READ_LIMIT (default: 512)
	ClientReadLimit int64
}

// EventsConfig controls automation event buffering and drop policy thresholds.
// These settings bound memory usage and govern when non-critical telemetry
// (heartbeat/console/network) may be dropped under backpressure.
type EventsConfig struct {
	// PerExecutionBuffer limits the number of buffered events per execution.
	// Env: BAS_EVENTS_PER_EXECUTION_BUFFER (default: 200)
	PerExecutionBuffer int

	// PerAttemptBuffer limits the number of buffered events per step attempt.
	// Env: BAS_EVENTS_PER_ATTEMPT_BUFFER (default: 50)
	PerAttemptBuffer int
}

// ToEventBufferLimits converts EventsConfig to contracts.EventBufferLimits for use by event sinks.
// Returns validated limits, falling back to defaults if config values are invalid.
func (e EventsConfig) ToEventBufferLimits() autocontracts.EventBufferLimits {
	limits := autocontracts.EventBufferLimits{
		PerExecution: e.PerExecutionBuffer,
		PerAttempt:   e.PerAttemptBuffer,
	}
	if limits.Validate() != nil {
		return autocontracts.DefaultEventBufferLimits
	}
	return limits
}

// EventBufferLimitsFromConfig returns validated event buffer limits from the global config.
// This is the single source of truth for event buffer configuration.
func EventBufferLimitsFromConfig() autocontracts.EventBufferLimits {
	return Load().Events.ToEventBufferLimits()
}

// RecordingConfig controls record mode and recording session settings.
type RecordingConfig struct {
	// DefaultViewportWidth is the default viewport width for recording sessions.
	// Env: BAS_RECORDING_DEFAULT_VIEWPORT_WIDTH (default: 1280)
	DefaultViewportWidth int

	// DefaultViewportHeight is the default viewport height for recording sessions.
	// Env: BAS_RECORDING_DEFAULT_VIEWPORT_HEIGHT (default: 720)
	DefaultViewportHeight int

	// DefaultStreamQuality is the default JPEG quality for frame streaming (1-100).
	// Env: BAS_RECORDING_DEFAULT_STREAM_QUALITY (default: 55)
	DefaultStreamQuality int

	// DefaultStreamFPS is the default frames per second for streaming (1-60).
	// Env: BAS_RECORDING_DEFAULT_STREAM_FPS (default: 6)
	DefaultStreamFPS int

	// InputTimeout is the timeout for forwarding input events to the driver.
	// Env: BAS_RECORDING_INPUT_TIMEOUT_MS (default: 2000)
	InputTimeout time.Duration

	// ConnectionIdleTimeout is how long to keep driver connections warm.
	// Env: BAS_RECORDING_CONN_IDLE_TIMEOUT_MS (default: 90000)
	ConnectionIdleTimeout time.Duration

	// MaxArchiveBytes is the maximum size for recording archive uploads.
	// Larger archives may contain more frames/assets but consume more memory.
	// Env: BAS_RECORDING_MAX_ARCHIVE_BYTES (default: 209715200, 200MB)
	MaxArchiveBytes int64

	// MaxFrames is the maximum number of frames allowed in a recording.
	// Higher = more complex recordings, Lower = faster processing.
	// Env: BAS_RECORDING_MAX_FRAMES (default: 400)
	MaxFrames int

	// MaxFrameBytes is the maximum size per frame asset.
	// Env: BAS_RECORDING_MAX_FRAME_BYTES (default: 12582912, 12MB)
	MaxFrameBytes int64

	// DefaultFrameDurationMs is the default duration between frames when not specified.
	// Env: BAS_RECORDING_DEFAULT_FRAME_DURATION_MS (default: 1600)
	DefaultFrameDurationMs int

	// ImportTimeout is the maximum duration allowed for recording imports.
	// Env: BAS_RECORDING_IMPORT_TIMEOUT_MS (default: 120000, 2 minutes)
	ImportTimeout time.Duration
}

// AIConfig controls AI analysis and DOM extraction limits.
type AIConfig struct {
	// DOMMaxDepth is the maximum depth to traverse when extracting DOM tree.
	// Lower = smaller payload, may miss nested elements.
	// Env: BAS_AI_DOM_MAX_DEPTH (default: 6)
	DOMMaxDepth int

	// DOMMaxChildrenPerNode limits children per node during extraction.
	// Env: BAS_AI_DOM_MAX_CHILDREN_PER_NODE (default: 12)
	DOMMaxChildrenPerNode int

	// DOMMaxTotalNodes limits total nodes in extracted DOM tree.
	// Env: BAS_AI_DOM_MAX_TOTAL_NODES (default: 800)
	DOMMaxTotalNodes int

	// DOMTextLimit truncates text content to this length.
	// Env: BAS_AI_DOM_TEXT_LIMIT (default: 120)
	DOMTextLimit int

	// DOMExtractionWaitMs is the wait time after navigation before DOM extraction.
	// Env: BAS_AI_DOM_EXTRACTION_WAIT_MS (default: 750)
	DOMExtractionWaitMs int

	// PreviewMinViewportDimension is the minimum viewport dimension for previews.
	// Env: BAS_AI_PREVIEW_MIN_VIEWPORT (default: 200)
	PreviewMinViewportDimension int

	// PreviewMaxViewportDimension is the maximum viewport dimension for previews.
	// Env: BAS_AI_PREVIEW_MAX_VIEWPORT (default: 10000)
	PreviewMaxViewportDimension int

	// PreviewDefaultViewportWidth is the default viewport width for previews.
	// Env: BAS_AI_PREVIEW_DEFAULT_WIDTH (default: 1920)
	PreviewDefaultViewportWidth int

	// PreviewDefaultViewportHeight is the default viewport height for previews.
	// Env: BAS_AI_PREVIEW_DEFAULT_HEIGHT (default: 1080)
	PreviewDefaultViewportHeight int

	// PreviewWaitMs is the wait time after navigation before screenshot.
	// Env: BAS_AI_PREVIEW_WAIT_MS (default: 1200)
	PreviewWaitMs int

	// PreviewTimeoutMs is the timeout for preview operations.
	// Env: BAS_AI_PREVIEW_TIMEOUT_MS (default: 20000)
	PreviewTimeoutMs int
}

// ReplayConfig controls video rendering and capture settings.
type ReplayConfig struct {
	// CaptureIntervalMs is the interval between frame captures for video generation.
	// Lower = smoother video, more frames, larger files.
	// Env: BAS_REPLAY_CAPTURE_INTERVAL_MS (default: 40)
	CaptureIntervalMs int

	// CaptureTailMs is additional capture time after the last action.
	// Env: BAS_REPLAY_CAPTURE_TAIL_MS (default: 320)
	CaptureTailMs int

	// PresentationWidth is the default output video width.
	// Env: BAS_REPLAY_PRESENTATION_WIDTH (default: 1280)
	PresentationWidth int

	// PresentationHeight is the default output video height.
	// Env: BAS_REPLAY_PRESENTATION_HEIGHT (default: 720)
	PresentationHeight int

	// MaxCaptureFrames limits the maximum number of frames to capture.
	// Env: BAS_REPLAY_MAX_CAPTURE_FRAMES (default: 720)
	MaxCaptureFrames int

	// RenderTimeout is the timeout for video rendering operations.
	// Env: BAS_REPLAY_RENDER_TIMEOUT_MS (default: 960000, 16 minutes)
	RenderTimeout time.Duration
}

// StorageConfig controls screenshot and artifact storage limits.
type StorageConfig struct {
	// MaxEmbeddedExternalBytes is the maximum size for embedded external files.
	// Env: BAS_STORAGE_MAX_EMBEDDED_EXTERNAL_BYTES (default: 5242880, 5MB)
	MaxEmbeddedExternalBytes int64
}

// ArtifactLimitsConfig controls size limits for collected artifacts.
// These are global defaults that can be overridden per-execution via ArtifactCollectionConfig.
type ArtifactLimitsConfig struct {
	// MaxScreenshotBytes is the maximum screenshot size before truncation.
	// Env: BAS_ARTIFACT_MAX_SCREENSHOT_BYTES (default: 524288, 512KB)
	MaxScreenshotBytes int

	// MaxDOMSnapshotBytes is the maximum DOM snapshot HTML size before truncation.
	// Env: BAS_ARTIFACT_MAX_DOM_SNAPSHOT_BYTES (default: 524288, 512KB)
	MaxDOMSnapshotBytes int

	// MaxConsoleEntryBytes is the maximum size per console log entry.
	// Env: BAS_ARTIFACT_MAX_CONSOLE_ENTRY_BYTES (default: 16384, 16KB)
	MaxConsoleEntryBytes int

	// MaxNetworkPreviewBytes is the maximum request/response body preview size.
	// Env: BAS_ARTIFACT_MAX_NETWORK_PREVIEW_BYTES (default: 65536, 64KB)
	MaxNetworkPreviewBytes int

	// DefaultProfile is the default artifact collection profile when not specified.
	// Valid values: "full", "standard", "minimal", "debug", "none"
	// Env: BAS_ARTIFACT_DEFAULT_PROFILE (default: "full")
	DefaultProfile string
}

// HTTPConfig controls API request limits and pagination defaults.
// These settings balance usability against resource consumption.
type HTTPConfig struct {
	// MaxBodyBytes is the maximum request body size for JSON payloads.
	// Applies to most API endpoints; uploads use separate limits.
	// Env: BAS_HTTP_MAX_BODY_BYTES (default: 1048576, 1MB)
	MaxBodyBytes int64

	// DefaultPageLimit is the default number of items returned when no limit is specified.
	// Tradeoff: Higher = more data per request, fewer round trips; Lower = faster responses.
	// Env: BAS_HTTP_DEFAULT_PAGE_LIMIT (default: 100)
	DefaultPageLimit int

	// MaxPageLimit is the maximum allowed page size for list endpoints.
	// Prevents clients from requesting too much data in one request.
	// Env: BAS_HTTP_MAX_PAGE_LIMIT (default: 500)
	MaxPageLimit int

	// CORSMaxAge is the max-age for Access-Control-Max-Age header (seconds).
	// Higher = fewer preflight requests; Lower = more up-to-date CORS policy.
	// Env: BAS_HTTP_CORS_MAX_AGE (default: 300)
	CORSMaxAge int
}

// EntitlementConfig controls subscription verification and feature gating.
// This enables the monetization model by connecting to the landing-page-business-suite
// entitlement service to verify user subscriptions and enforce tier-based limits.
type EntitlementConfig struct {
	// Enabled controls whether entitlement checking is active.
	// When false, all features are available without restrictions (development mode).
	// Env: BAS_ENTITLEMENT_ENABLED (default: false)
	Enabled bool

	// ServiceURL is the base URL of the entitlement service (landing-page-business-suite).
	// Must include protocol (https://) and no trailing slash.
	// Env: BAS_ENTITLEMENT_SERVICE_URL (default: "")
	ServiceURL string

	// CacheTTL is how long to cache entitlement responses before re-checking.
	// Tradeoff: Higher = fewer network calls, slower to reflect subscription changes.
	// Lower = more responsive to upgrades/downgrades, more API load.
	// Env: BAS_ENTITLEMENT_CACHE_TTL_MS (default: 300000, 5 minutes)
	CacheTTL time.Duration

	// RequestTimeout is the timeout for entitlement service requests.
	// Should be short to avoid blocking user operations.
	// Env: BAS_ENTITLEMENT_REQUEST_TIMEOUT_MS (default: 5000)
	RequestTimeout time.Duration

	// OfflineGracePeriod is how long to allow operations when the entitlement service is unreachable.
	// Uses cached entitlements during this period; after expiry, falls back to free tier.
	// Env: BAS_ENTITLEMENT_OFFLINE_GRACE_PERIOD_MS (default: 86400000, 24 hours)
	OfflineGracePeriod time.Duration

	// DefaultTier is the tier to use when no subscription is found or service is unavailable.
	// Env: BAS_ENTITLEMENT_DEFAULT_TIER (default: "free")
	DefaultTier string

	// TierLimits defines the execution limits per tier per calendar month.
	// Parsed from JSON: {"free": 50, "solo": 200, "pro": -1, "studio": -1, "business": -1}
	// -1 means unlimited. These can be overridden via BAS_ENTITLEMENT_TIER_LIMITS_JSON.
	TierLimits map[string]int

	// WatermarkTiers lists tiers that should have watermarks applied to exports.
	// Env: BAS_ENTITLEMENT_WATERMARK_TIERS (default: "free,solo")
	WatermarkTiers []string

	// AITiers lists tiers that have access to AI-powered features.
	// Env: BAS_ENTITLEMENT_AI_TIERS (default: "pro,studio,business")
	AITiers []string

	// RecordingTiers lists tiers that have access to live recording features.
	// Env: BAS_ENTITLEMENT_RECORDING_TIERS (default: "solo,pro,studio,business")
	RecordingTiers []string
}

// PerformanceConfig controls debug performance mode for frame streaming diagnostics.
// When enabled, timing data is collected at each stage of the streaming pipeline
// to identify bottlenecks and track optimization progress.
type PerformanceConfig struct {
	// Enabled activates debug performance mode for frame streaming.
	// When true, the API parses timing headers from driver frames and collects metrics.
	// Env: BAS_PERF_ENABLED (default: false)
	Enabled bool

	// LogSummaryInterval controls how often performance summaries are logged (in frames).
	// 0 disables periodic logging. Useful for continuous monitoring.
	// Env: BAS_PERF_LOG_SUMMARY_INTERVAL (default: 60)
	LogSummaryInterval int

	// ExposeEndpoint enables the /debug/performance HTTP endpoint.
	// When true, clients can query current performance stats via HTTP.
	// Env: BAS_PERF_EXPOSE_ENDPOINT (default: true)
	ExposeEndpoint bool

	// BufferSize is the number of frame timings to retain for percentile analysis.
	// Higher = more accurate percentiles, more memory. Ring buffer evicts oldest.
	// Env: BAS_PERF_BUFFER_SIZE (default: 100)
	BufferSize int

	// StreamToWebSocket sends aggregated stats to subscribed UI clients.
	// When true, perf_stats messages are broadcast periodically over WebSocket.
	// Env: BAS_PERF_STREAM_TO_WEBSOCKET (default: true)
	StreamToWebSocket bool
}

var (
	globalConfig *Config
	configOnce   sync.Once
)

// Load returns the global configuration, loading from environment variables on first call.
// The configuration is loaded once and cached for the lifetime of the process.
func Load() *Config {
	configOnce.Do(func() {
		globalConfig = loadFromEnv()
	})
	return globalConfig
}

// loadFromEnv creates a new Config populated from environment variables.
func loadFromEnv() *Config {
	return &Config{
		Timeouts: TimeoutsConfig{
			DefaultRequest:      parseDurationMs("BAS_TIMEOUT_DEFAULT_REQUEST_MS", 5000),
			ExtendedRequest:     parseDurationMs("BAS_TIMEOUT_EXTENDED_REQUEST_MS", 10000),
			AIRequest:           parseDurationMs("BAS_TIMEOUT_AI_REQUEST_MS", 45000),
			AIAnalysis:          parseDurationMs("BAS_TIMEOUT_AI_ANALYSIS_MS", 60000),
			ElementAnalysis:     parseDurationMs("BAS_TIMEOUT_ELEMENT_ANALYSIS_MS", 30000),
			ExecutionCompletion: parseDurationMs("BAS_TIMEOUT_EXECUTION_COMPLETION_MS", 120000),
			DatabasePing:        parseDurationMs("BAS_TIMEOUT_DATABASE_PING_MS", 2000),
			DatabaseQuery:       parseDurationMs("BAS_TIMEOUT_DATABASE_QUERY_MS", 5000),
			DatabaseMigration:   parseDurationMs("BAS_TIMEOUT_DATABASE_MIGRATION_MS", 30000),
			BrowserEngineHealth: parseDurationMs("BAS_TIMEOUT_BROWSER_ENGINE_HEALTH_MS", 5000),
			GlobalRequest:       parseDurationMs("BAS_TIMEOUT_GLOBAL_REQUEST_MS", 900000),
			RecordModeSession:   parseDurationMs("BAS_TIMEOUT_RECORD_MODE_SESSION_MS", 30000),
			StartupHealthCheck:  parseDurationMs("BAS_TIMEOUT_STARTUP_HEALTH_CHECK_MS", 10000),
		},
		Database: DatabaseConfig{
			MaxOpenConns:      parseInt("BAS_DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:      parseInt("BAS_DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime:   parseDurationMs("BAS_DB_CONN_MAX_LIFETIME_MS", 300000),
			MaxRetries:        parseInt("BAS_DB_MAX_RETRIES", 10),
			BaseRetryDelay:    parseDurationMs("BAS_DB_BASE_RETRY_DELAY_MS", 1000),
			MaxRetryDelay:     parseDurationMs("BAS_DB_MAX_RETRY_DELAY_MS", 30000),
			RetryJitterFactor: parseFloat("BAS_DB_RETRY_JITTER_FACTOR", 0.25),
		},
		Execution: ExecutionConfig{
			BaseTimeout:            parseDurationMs("BAS_EXECUTION_BASE_TIMEOUT_MS", 30000),
			PerStepTimeout:         parseDurationMs("BAS_EXECUTION_PER_STEP_TIMEOUT_MS", 10000),
			PerStepSubflowTimeout:  parseDurationMs("BAS_EXECUTION_PER_STEP_SUBFLOW_TIMEOUT_MS", 15000),
			MinTimeout:             parseDurationMs("BAS_EXECUTION_MIN_TIMEOUT_MS", 90000),
			MaxTimeout:             parseDurationMs("BAS_EXECUTION_MAX_TIMEOUT_MS", 270000),
			MaxSubflowDepth:        parseInt("BAS_EXECUTION_MAX_SUBFLOW_DEPTH", 5),
			DefaultEntryTimeout:    parseDurationMs("BAS_EXECUTION_DEFAULT_ENTRY_TIMEOUT_MS", 3000),
			MinEntryTimeout:        parseDurationMs("BAS_EXECUTION_MIN_ENTRY_TIMEOUT_MS", 250),
			DefaultRetryDelay:      parseDurationMs("BAS_EXECUTION_DEFAULT_RETRY_DELAY_MS", 750),
			DefaultBackoffFactor:   parseFloat("BAS_EXECUTION_DEFAULT_BACKOFF_FACTOR", 1.5),
			CompletionPollInterval: parseDurationMs("BAS_EXECUTION_COMPLETION_POLL_INTERVAL_MS", 250),
			AdhocCleanupInterval:   parseDurationMs("BAS_EXECUTION_ADHOC_CLEANUP_INTERVAL_MS", 5000),
			AdhocRetentionPeriod:   parseDurationMs("BAS_EXECUTION_ADHOC_RETENTION_PERIOD_MS", 600000),
			HeartbeatInterval:      parseDurationMs("BAS_EXECUTION_HEARTBEAT_INTERVAL_MS", 2000),
		},
		WebSocket: WebSocketConfig{
			ClientSendBufferSize:   parseInt("BAS_WS_CLIENT_SEND_BUFFER_SIZE", 256),
			ClientBinaryBufferSize: parseInt("BAS_WS_CLIENT_BINARY_BUFFER_SIZE", 120),
			ClientReadLimit:        parseInt64("BAS_WS_CLIENT_READ_LIMIT", 512),
		},
		Events: EventsConfig{
			PerExecutionBuffer: parseInt("BAS_EVENTS_PER_EXECUTION_BUFFER", 200),
			PerAttemptBuffer:   parseInt("BAS_EVENTS_PER_ATTEMPT_BUFFER", 50),
		},
		Recording: RecordingConfig{
			DefaultViewportWidth:   parseInt("BAS_RECORDING_DEFAULT_VIEWPORT_WIDTH", 1280),
			DefaultViewportHeight:  parseInt("BAS_RECORDING_DEFAULT_VIEWPORT_HEIGHT", 720),
			DefaultStreamQuality:   parseInt("BAS_RECORDING_DEFAULT_STREAM_QUALITY", 55),
			DefaultStreamFPS:       parseInt("BAS_RECORDING_DEFAULT_STREAM_FPS", 6),
			InputTimeout:           parseDurationMs("BAS_RECORDING_INPUT_TIMEOUT_MS", 2000),
			ConnectionIdleTimeout:  parseDurationMs("BAS_RECORDING_CONN_IDLE_TIMEOUT_MS", 90000),
			MaxArchiveBytes:        parseInt64("BAS_RECORDING_MAX_ARCHIVE_BYTES", 209715200),
			MaxFrames:              parseInt("BAS_RECORDING_MAX_FRAMES", 400),
			MaxFrameBytes:          parseInt64("BAS_RECORDING_MAX_FRAME_BYTES", 12582912),
			DefaultFrameDurationMs: parseInt("BAS_RECORDING_DEFAULT_FRAME_DURATION_MS", 1600),
			ImportTimeout:          parseDurationMs("BAS_RECORDING_IMPORT_TIMEOUT_MS", 120000),
		},
		AI: AIConfig{
			DOMMaxDepth:                  parseInt("BAS_AI_DOM_MAX_DEPTH", 6),
			DOMMaxChildrenPerNode:        parseInt("BAS_AI_DOM_MAX_CHILDREN_PER_NODE", 12),
			DOMMaxTotalNodes:             parseInt("BAS_AI_DOM_MAX_TOTAL_NODES", 800),
			DOMTextLimit:                 parseInt("BAS_AI_DOM_TEXT_LIMIT", 120),
			DOMExtractionWaitMs:          parseInt("BAS_AI_DOM_EXTRACTION_WAIT_MS", 750),
			PreviewMinViewportDimension:  parseInt("BAS_AI_PREVIEW_MIN_VIEWPORT", 200),
			PreviewMaxViewportDimension:  parseInt("BAS_AI_PREVIEW_MAX_VIEWPORT", 10000),
			PreviewDefaultViewportWidth:  parseInt("BAS_AI_PREVIEW_DEFAULT_WIDTH", 1920),
			PreviewDefaultViewportHeight: parseInt("BAS_AI_PREVIEW_DEFAULT_HEIGHT", 1080),
			PreviewWaitMs:                parseInt("BAS_AI_PREVIEW_WAIT_MS", 1200),
			PreviewTimeoutMs:             parseInt("BAS_AI_PREVIEW_TIMEOUT_MS", 20000),
		},
		Replay: ReplayConfig{
			CaptureIntervalMs:  parseInt("BAS_REPLAY_CAPTURE_INTERVAL_MS", 40),
			CaptureTailMs:      parseInt("BAS_REPLAY_CAPTURE_TAIL_MS", 320),
			PresentationWidth:  parseInt("BAS_REPLAY_PRESENTATION_WIDTH", 1280),
			PresentationHeight: parseInt("BAS_REPLAY_PRESENTATION_HEIGHT", 720),
			MaxCaptureFrames:   parseInt("BAS_REPLAY_MAX_CAPTURE_FRAMES", 720),
			RenderTimeout:      parseDurationMs("BAS_REPLAY_RENDER_TIMEOUT_MS", 960000),
		},
		Storage: StorageConfig{
			MaxEmbeddedExternalBytes: parseInt64("BAS_STORAGE_MAX_EMBEDDED_EXTERNAL_BYTES", 5242880),
		},
		ArtifactLimits: ArtifactLimitsConfig{
			MaxScreenshotBytes:     parseInt("BAS_ARTIFACT_MAX_SCREENSHOT_BYTES", 524288),
			MaxDOMSnapshotBytes:    parseInt("BAS_ARTIFACT_MAX_DOM_SNAPSHOT_BYTES", 524288),
			MaxConsoleEntryBytes:   parseInt("BAS_ARTIFACT_MAX_CONSOLE_ENTRY_BYTES", 16384),
			MaxNetworkPreviewBytes: parseInt("BAS_ARTIFACT_MAX_NETWORK_PREVIEW_BYTES", 65536),
			DefaultProfile:         parseString("BAS_ARTIFACT_DEFAULT_PROFILE", "full"),
		},
		HTTP: HTTPConfig{
			MaxBodyBytes:     parseInt64("BAS_HTTP_MAX_BODY_BYTES", 1048576),
			DefaultPageLimit: parseInt("BAS_HTTP_DEFAULT_PAGE_LIMIT", 100),
			MaxPageLimit:     parseInt("BAS_HTTP_MAX_PAGE_LIMIT", 500),
			CORSMaxAge:       parseInt("BAS_HTTP_CORS_MAX_AGE", 300),
		},
		Entitlement: EntitlementConfig{
			Enabled:            parseBool("BAS_ENTITLEMENT_ENABLED", false),
			ServiceURL:         parseString("BAS_ENTITLEMENT_SERVICE_URL", ""),
			CacheTTL:           parseDurationMs("BAS_ENTITLEMENT_CACHE_TTL_MS", 300000),
			RequestTimeout:     parseDurationMs("BAS_ENTITLEMENT_REQUEST_TIMEOUT_MS", 5000),
			OfflineGracePeriod: parseDurationMs("BAS_ENTITLEMENT_OFFLINE_GRACE_PERIOD_MS", 86400000),
			DefaultTier:        parseString("BAS_ENTITLEMENT_DEFAULT_TIER", "free"),
			TierLimits:         parseTierLimits("BAS_ENTITLEMENT_TIER_LIMITS_JSON"),
			WatermarkTiers:     parseStringList("BAS_ENTITLEMENT_WATERMARK_TIERS", "free,solo"),
			AITiers:            parseStringList("BAS_ENTITLEMENT_AI_TIERS", "pro,studio,business"),
			RecordingTiers:     parseStringList("BAS_ENTITLEMENT_RECORDING_TIERS", "solo,pro,studio,business"),
		},
		Performance: PerformanceConfig{
			Enabled:            parseBool("BAS_PERF_ENABLED", false),
			LogSummaryInterval: parseInt("BAS_PERF_LOG_SUMMARY_INTERVAL", 60),
			ExposeEndpoint:     parseBool("BAS_PERF_EXPOSE_ENDPOINT", true),
			BufferSize:         parseInt("BAS_PERF_BUFFER_SIZE", 100),
			StreamToWebSocket:  parseBool("BAS_PERF_STREAM_TO_WEBSOCKET", true),
		},
	}
}

// Validate checks that all configuration values are within acceptable ranges.
// Returns an error describing the first invalid value found.
func (c *Config) Validate() error {
	// Timeouts validation
	if c.Timeouts.DefaultRequest <= 0 {
		return fmt.Errorf("DefaultRequest timeout must be positive, got %v", c.Timeouts.DefaultRequest)
	}
	if c.Timeouts.GlobalRequest < c.Timeouts.DefaultRequest {
		return fmt.Errorf("GlobalRequest timeout (%v) must be >= DefaultRequest (%v)", c.Timeouts.GlobalRequest, c.Timeouts.DefaultRequest)
	}

	// Database validation
	if c.Database.MaxOpenConns < 1 {
		return fmt.Errorf("MaxOpenConns must be at least 1, got %d", c.Database.MaxOpenConns)
	}
	if c.Database.MaxIdleConns < 0 || c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("MaxIdleConns (%d) must be between 0 and MaxOpenConns (%d)", c.Database.MaxIdleConns, c.Database.MaxOpenConns)
	}
	if c.Database.RetryJitterFactor < 0 || c.Database.RetryJitterFactor > 1 {
		return fmt.Errorf("RetryJitterFactor must be between 0 and 1, got %f", c.Database.RetryJitterFactor)
	}

	// Execution validation
	if c.Execution.MinTimeout > c.Execution.MaxTimeout {
		return fmt.Errorf("MinTimeout (%v) cannot be greater than MaxTimeout (%v)", c.Execution.MinTimeout, c.Execution.MaxTimeout)
	}
	if c.Execution.MaxSubflowDepth < 1 {
		return fmt.Errorf("MaxSubflowDepth must be at least 1, got %d", c.Execution.MaxSubflowDepth)
	}
	if c.Execution.DefaultBackoffFactor < 1 {
		return fmt.Errorf("DefaultBackoffFactor must be at least 1, got %f", c.Execution.DefaultBackoffFactor)
	}
	if c.Execution.HeartbeatInterval < 0 {
		return fmt.Errorf("HeartbeatInterval must be non-negative, got %v", c.Execution.HeartbeatInterval)
	}

	// WebSocket validation
	if c.WebSocket.ClientSendBufferSize < 1 {
		return fmt.Errorf("ClientSendBufferSize must be at least 1, got %d", c.WebSocket.ClientSendBufferSize)
	}
	if c.WebSocket.ClientReadLimit < 1 {
		return fmt.Errorf("ClientReadLimit must be at least 1, got %d", c.WebSocket.ClientReadLimit)
	}

	// Events validation
	if c.Events.PerExecutionBuffer < 1 {
		return fmt.Errorf("PerExecutionBuffer must be at least 1, got %d", c.Events.PerExecutionBuffer)
	}
	if c.Events.PerAttemptBuffer < 1 {
		return fmt.Errorf("PerAttemptBuffer must be at least 1, got %d", c.Events.PerAttemptBuffer)
	}

	// Recording validation
	if c.Recording.DefaultViewportWidth < 1 || c.Recording.DefaultViewportHeight < 1 {
		return fmt.Errorf("DefaultViewport dimensions must be positive, got %dx%d", c.Recording.DefaultViewportWidth, c.Recording.DefaultViewportHeight)
	}
	if c.Recording.DefaultStreamQuality < 1 || c.Recording.DefaultStreamQuality > 100 {
		return fmt.Errorf("DefaultStreamQuality must be between 1 and 100, got %d", c.Recording.DefaultStreamQuality)
	}
	if c.Recording.DefaultStreamFPS < 1 || c.Recording.DefaultStreamFPS > 60 {
		return fmt.Errorf("DefaultStreamFPS must be between 1 and 60, got %d", c.Recording.DefaultStreamFPS)
	}
	if c.Recording.MaxArchiveBytes < 1048576 { // 1MB minimum
		return fmt.Errorf("MaxArchiveBytes must be at least 1MB, got %d", c.Recording.MaxArchiveBytes)
	}
	if c.Recording.MaxFrames < 1 {
		return fmt.Errorf("MaxFrames must be at least 1, got %d", c.Recording.MaxFrames)
	}
	if c.Recording.MaxFrameBytes < 10240 { // 10KB minimum
		return fmt.Errorf("MaxFrameBytes must be at least 10KB, got %d", c.Recording.MaxFrameBytes)
	}

	// AI validation
	if c.AI.DOMMaxDepth < 1 {
		return fmt.Errorf("DOMMaxDepth must be at least 1, got %d", c.AI.DOMMaxDepth)
	}
	if c.AI.DOMMaxTotalNodes < 1 {
		return fmt.Errorf("DOMMaxTotalNodes must be at least 1, got %d", c.AI.DOMMaxTotalNodes)
	}

	// Replay validation
	if c.Replay.CaptureIntervalMs < 1 {
		return fmt.Errorf("CaptureIntervalMs must be at least 1, got %d", c.Replay.CaptureIntervalMs)
	}
	if c.Replay.MaxCaptureFrames < 1 {
		return fmt.Errorf("MaxCaptureFrames must be at least 1, got %d", c.Replay.MaxCaptureFrames)
	}

	// Storage validation
	if c.Storage.MaxEmbeddedExternalBytes < 1024 {
		return fmt.Errorf("MaxEmbeddedExternalBytes must be at least 1024, got %d", c.Storage.MaxEmbeddedExternalBytes)
	}

	// ArtifactLimits validation
	if c.ArtifactLimits.MaxScreenshotBytes < 1024 {
		return fmt.Errorf("ArtifactLimits.MaxScreenshotBytes must be at least 1024, got %d", c.ArtifactLimits.MaxScreenshotBytes)
	}
	if c.ArtifactLimits.MaxDOMSnapshotBytes < 1024 {
		return fmt.Errorf("ArtifactLimits.MaxDOMSnapshotBytes must be at least 1024, got %d", c.ArtifactLimits.MaxDOMSnapshotBytes)
	}
	if c.ArtifactLimits.MaxConsoleEntryBytes < 256 {
		return fmt.Errorf("ArtifactLimits.MaxConsoleEntryBytes must be at least 256, got %d", c.ArtifactLimits.MaxConsoleEntryBytes)
	}
	if c.ArtifactLimits.MaxNetworkPreviewBytes < 256 {
		return fmt.Errorf("ArtifactLimits.MaxNetworkPreviewBytes must be at least 256, got %d", c.ArtifactLimits.MaxNetworkPreviewBytes)
	}
	validProfiles := map[string]bool{"full": true, "standard": true, "minimal": true, "debug": true, "none": true}
	if c.ArtifactLimits.DefaultProfile != "" && !validProfiles[c.ArtifactLimits.DefaultProfile] {
		return fmt.Errorf("ArtifactLimits.DefaultProfile must be one of: full, standard, minimal, debug, none; got %q", c.ArtifactLimits.DefaultProfile)
	}

	// HTTP validation
	if c.HTTP.MaxBodyBytes < 1024 { // 1KB minimum
		return fmt.Errorf("MaxBodyBytes must be at least 1KB, got %d", c.HTTP.MaxBodyBytes)
	}
	if c.HTTP.DefaultPageLimit < 1 {
		return fmt.Errorf("DefaultPageLimit must be at least 1, got %d", c.HTTP.DefaultPageLimit)
	}
	if c.HTTP.MaxPageLimit < c.HTTP.DefaultPageLimit {
		return fmt.Errorf("MaxPageLimit (%d) cannot be less than DefaultPageLimit (%d)", c.HTTP.MaxPageLimit, c.HTTP.DefaultPageLimit)
	}
	if c.HTTP.CORSMaxAge < 0 {
		return fmt.Errorf("CORSMaxAge must be non-negative, got %d", c.HTTP.CORSMaxAge)
	}

	// Execution adhoc cleanup validation
	if c.Execution.AdhocCleanupInterval <= 0 {
		return fmt.Errorf("AdhocCleanupInterval must be positive, got %v", c.Execution.AdhocCleanupInterval)
	}
	if c.Execution.AdhocRetentionPeriod <= 0 {
		return fmt.Errorf("AdhocRetentionPeriod must be positive, got %v", c.Execution.AdhocRetentionPeriod)
	}

	// Entitlement validation
	if c.Entitlement.Enabled {
		if c.Entitlement.ServiceURL == "" {
			return fmt.Errorf("Entitlement.ServiceURL is required when entitlements are enabled")
		}
		if c.Entitlement.CacheTTL <= 0 {
			return fmt.Errorf("Entitlement.CacheTTL must be positive, got %v", c.Entitlement.CacheTTL)
		}
		if c.Entitlement.RequestTimeout <= 0 {
			return fmt.Errorf("Entitlement.RequestTimeout must be positive, got %v", c.Entitlement.RequestTimeout)
		}
	}
	if c.Entitlement.TierLimits == nil {
		return fmt.Errorf("Entitlement.TierLimits cannot be nil")
	}

	// Performance validation
	if c.Performance.LogSummaryInterval < 0 {
		return fmt.Errorf("Performance.LogSummaryInterval must be non-negative, got %d", c.Performance.LogSummaryInterval)
	}
	if c.Performance.BufferSize < 1 {
		return fmt.Errorf("Performance.BufferSize must be at least 1, got %d", c.Performance.BufferSize)
	}

	return nil
}

// parseDurationMs parses an environment variable as milliseconds and returns a time.Duration.
func parseDurationMs(envVar string, defaultMs int) time.Duration {
	value := strings.TrimSpace(os.Getenv(envVar))
	if value == "" {
		return time.Duration(defaultMs) * time.Millisecond
	}
	ms, err := strconv.Atoi(value)
	if err != nil || ms < 0 {
		return time.Duration(defaultMs) * time.Millisecond
	}
	return time.Duration(ms) * time.Millisecond
}

// parseInt parses an environment variable as an integer.
func parseInt(envVar string, defaultVal int) int {
	value := strings.TrimSpace(os.Getenv(envVar))
	if value == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return defaultVal
	}
	return v
}

// parseInt64 parses an environment variable as an int64.
func parseInt64(envVar string, defaultVal int64) int64 {
	value := strings.TrimSpace(os.Getenv(envVar))
	if value == "" {
		return defaultVal
	}
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultVal
	}
	return v
}

// parseFloat parses an environment variable as a float64.
func parseFloat(envVar string, defaultVal float64) float64 {
	value := strings.TrimSpace(os.Getenv(envVar))
	if value == "" {
		return defaultVal
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultVal
	}
	return v
}

// parseBool parses an environment variable as a boolean.
func parseBool(envVar string, defaultVal bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(envVar)))
	if value == "" {
		return defaultVal
	}
	return value == "true" || value == "1" || value == "yes"
}

// parseString returns an environment variable value or a default.
func parseString(envVar string, defaultVal string) string {
	value := strings.TrimSpace(os.Getenv(envVar))
	if value == "" {
		return defaultVal
	}
	return value
}

// parseStringList parses a comma-separated environment variable into a slice.
func parseStringList(envVar string, defaultVal string) []string {
	value := strings.TrimSpace(os.Getenv(envVar))
	if value == "" {
		value = defaultVal
	}
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// parseTierLimits parses tier limits from JSON or returns defaults.
// JSON format: {"free": 50, "solo": 200, "pro": -1}
// -1 means unlimited.
func parseTierLimits(envVar string) map[string]int {
	defaults := map[string]int{
		"free":     50,
		"solo":     200,
		"pro":      -1, // unlimited
		"studio":   -1, // unlimited
		"business": -1, // unlimited
	}

	value := strings.TrimSpace(os.Getenv(envVar))
	if value == "" {
		return defaults
	}

	// Try to parse as JSON
	result := make(map[string]int)
	// Simple JSON parsing without importing encoding/json to keep config lightweight
	// Format: {"key": value, "key2": value2}
	value = strings.Trim(value, "{}")
	if value == "" {
		return defaults
	}

	pairs := strings.Split(value, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.Trim(strings.TrimSpace(parts[0]), "\"'")
		valStr := strings.TrimSpace(parts[1])
		if val, err := strconv.Atoi(valStr); err == nil {
			result[key] = val
		}
	}

	// Merge with defaults (defaults take precedence for missing keys)
	for k, v := range defaults {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}

	return result
}
