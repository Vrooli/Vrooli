package constants

import "time"

// HTTP Request Timeouts
const (
	// DefaultRequestTimeout is used for standard CRUD operations that don't require
	// complex processing or external service calls.
	DefaultRequestTimeout = 5 * time.Second

	// ExtendedRequestTimeout is used for operations that may take longer but don't
	// involve AI processing (e.g., bulk operations, complex queries).
	ExtendedRequestTimeout = 10 * time.Second

	// AIRequestTimeout is used for AI-powered operations like workflow generation
	// and modification, which require inference from external AI services.
	AIRequestTimeout = 45 * time.Second

	// AIAnalysisTimeout is used for complex AI analysis operations that process
	// screenshots, DOM trees, and require multi-step inference.
	AIAnalysisTimeout = 60 * time.Second

	// ElementAnalysisTimeout is used for element analysis operations that involve
	// screenshot capture and DOM processing.
	ElementAnalysisTimeout = 30 * time.Second

	// ExecutionCompletionTimeout bounds how long the API will wait when callers
	// set wait_for_completion=true on execution endpoints.
	ExecutionCompletionTimeout = 2 * time.Minute
)

// Database Operation Timeouts
const (
	// DatabasePingTimeout is used for health checks and connection verification.
	DatabasePingTimeout = 2 * time.Second

	// DatabaseQueryTimeout is used for standard database queries.
	DatabaseQueryTimeout = 5 * time.Second

	// DatabaseMigrationTimeout is used for schema migrations which may take longer.
	DatabaseMigrationTimeout = 30 * time.Second
)

// External Service Timeouts
const (
	// BrowserEngineHealthTimeout is used for browser engine health check requests.
	BrowserEngineHealthTimeout = 5 * time.Second

	// ShortLivedBackgroundContext is used for background operations that should
	// complete quickly (e.g., connection initialization).
	ShortLivedBackgroundContext = 1 * time.Second
)
