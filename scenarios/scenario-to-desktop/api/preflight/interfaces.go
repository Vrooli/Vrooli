// Package preflight provides bundle preflight validation services.
// This domain validates bundles before packaging by running dry-run
// tests and checking service configurations.
package preflight

import (
	"context"
	"time"

	runtimeapi "scenario-to-desktop-runtime/api"
	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

// Service orchestrates preflight validation for bundles.
// It coordinates between session stores, job stores, and runtime clients
// to perform synchronous or asynchronous bundle validation.
type Service interface {
	// RunBundlePreflight executes synchronous preflight validation.
	RunBundlePreflight(request Request) (*Response, error)

	// RunPreflightJob executes asynchronous preflight validation.
	// The job status can be polled via GetJob.
	RunPreflightJob(jobID string, request Request)

	// CreateJob creates a new async preflight job.
	CreateJob() *Job

	// GetJob retrieves a job by ID.
	GetJob(id string) (*Job, bool)

	// GetSession retrieves a session by ID.
	GetSession(id string) (*Session, bool)

	// StartJanitor starts the background cleanup goroutine.
	StartJanitor()
}

// SessionStore defines the interface for preflight session management.
// Sessions are long-lived runtime environments used for interactive preflight.
type SessionStore interface {
	// Create creates a new preflight session with the given manifest and bundle root.
	Create(manifest *bundlemanifest.Manifest, bundleRoot string, ttlSeconds int) (*Session, error)

	// Get retrieves a session by ID, checking expiry.
	// Returns nil and false if session not found or expired.
	Get(id string) (*Session, bool)

	// Refresh extends the TTL of an existing session.
	Refresh(session *Session, ttlSeconds int)

	// Stop stops and cleans up a preflight session.
	// Returns true if session was found and stopped.
	Stop(id string) bool

	// Cleanup removes expired sessions. Called periodically by janitor.
	Cleanup()
}

// JobStore defines the interface for preflight job management.
// Jobs represent async preflight operations that can be polled for status.
type JobStore interface {
	// Create creates a new async preflight job and returns it.
	Create() *Job

	// Get retrieves a job by ID.
	Get(id string) (*Job, bool)

	// Update updates a job using a modifier function.
	Update(id string, fn func(job *Job))

	// SetStep updates the state of a specific step in a job.
	SetStep(id, stepID, state, detail string)

	// SetResult updates the result of a job using an updater function.
	SetResult(id string, updater func(prev *Response) *Response)

	// Finish marks a job as complete with final status.
	Finish(id string, status, errMsg string)

	// Cleanup removes completed jobs older than the expiration threshold.
	Cleanup()
}

// RuntimeClient abstracts communication with the bundle runtime control API.
// This enables testing seams for runtime interactions.
type RuntimeClient interface {
	// Status returns runtime instance metadata.
	Status() (*Runtime, error)

	// Validate validates the bundle configuration.
	Validate() (*runtimeapi.BundleValidationResult, error)

	// Secrets retrieves the list of secrets from the runtime.
	Secrets() ([]Secret, error)

	// ApplySecrets applies secrets to the runtime.
	ApplySecrets(secrets map[string]string) error

	// Ready checks if the runtime is ready, with polling support.
	Ready(request Request, timeout time.Duration) (Ready, int, error)

	// Ports retrieves the port mappings from the runtime.
	Ports() (map[string]map[string]int, error)

	// Telemetry retrieves telemetry configuration.
	Telemetry() (*Telemetry, error)

	// LogTails retrieves log tails for services.
	LogTails(request Request) []LogTail
}

// RuntimeFactory creates runtime clients and handles for preflight operations.
type RuntimeFactory interface {
	// NewDryRunRuntime creates a dry-run runtime for validation.
	NewDryRunRuntime(ctx context.Context, manifest *bundlemanifest.Manifest, bundleRoot string, timeout time.Duration) (*RuntimeHandle, error)
}

// TimeProvider abstracts time for deterministic testing.
type TimeProvider interface {
	// Now returns the current time.
	Now() time.Time
}
