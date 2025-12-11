package constants

import (
	"time"

	"github.com/vrooli/browser-automation-studio/config"
)

// Timeouts returns the timeouts configuration from the control surface.
// This provides access to all configurable timeout values.
func Timeouts() config.TimeoutsConfig {
	return config.Load().Timeouts
}

// HTTP Request Timeouts - These are variable accessors that read from config.
// For backwards compatibility, we provide these as variables that are initialized
// from the config package. New code should use config.Load().Timeouts directly.
var (
	// DefaultRequestTimeout is used for standard CRUD operations that don't require
	// complex processing or external service calls.
	// Configurable via BAS_TIMEOUT_DEFAULT_REQUEST_MS (default: 5000)
	DefaultRequestTimeout = config.Load().Timeouts.DefaultRequest

	// ExtendedRequestTimeout is used for operations that may take longer but don't
	// involve AI processing (e.g., bulk operations, complex queries).
	// Configurable via BAS_TIMEOUT_EXTENDED_REQUEST_MS (default: 10000)
	ExtendedRequestTimeout = config.Load().Timeouts.ExtendedRequest

	// AIRequestTimeout is used for AI-powered operations like workflow generation
	// and modification, which require inference from external AI services.
	// Configurable via BAS_TIMEOUT_AI_REQUEST_MS (default: 45000)
	AIRequestTimeout = config.Load().Timeouts.AIRequest

	// AIAnalysisTimeout is used for complex AI analysis operations that process
	// screenshots, DOM trees, and require multi-step inference.
	// Configurable via BAS_TIMEOUT_AI_ANALYSIS_MS (default: 60000)
	AIAnalysisTimeout = config.Load().Timeouts.AIAnalysis

	// ElementAnalysisTimeout is used for element analysis operations that involve
	// screenshot capture and DOM processing.
	// Configurable via BAS_TIMEOUT_ELEMENT_ANALYSIS_MS (default: 30000)
	ElementAnalysisTimeout = config.Load().Timeouts.ElementAnalysis

	// ExecutionCompletionTimeout bounds how long the API will wait when callers
	// set wait_for_completion=true on execution endpoints.
	// Configurable via BAS_TIMEOUT_EXECUTION_COMPLETION_MS (default: 120000)
	ExecutionCompletionTimeout = config.Load().Timeouts.ExecutionCompletion
)

// Database Operation Timeouts
var (
	// DatabasePingTimeout is used for health checks and connection verification.
	// Configurable via BAS_TIMEOUT_DATABASE_PING_MS (default: 2000)
	DatabasePingTimeout = config.Load().Timeouts.DatabasePing

	// DatabaseQueryTimeout is used for standard database queries.
	// Configurable via BAS_TIMEOUT_DATABASE_QUERY_MS (default: 5000)
	DatabaseQueryTimeout = config.Load().Timeouts.DatabaseQuery

	// DatabaseMigrationTimeout is used for schema migrations which may take longer.
	// Configurable via BAS_TIMEOUT_DATABASE_MIGRATION_MS (default: 30000)
	DatabaseMigrationTimeout = config.Load().Timeouts.DatabaseMigration
)

// External Service Timeouts
var (
	// BrowserEngineHealthTimeout is used for browser engine health check requests.
	// Configurable via BAS_TIMEOUT_BROWSER_ENGINE_HEALTH_MS (default: 5000)
	BrowserEngineHealthTimeout = config.Load().Timeouts.BrowserEngineHealth

	// ShortLivedBackgroundContext is used for background operations that should
	// complete quickly (e.g., connection initialization).
	// This is intentionally not configurable as it's a fixed short duration.
	ShortLivedBackgroundContext = 1 * time.Second
)
