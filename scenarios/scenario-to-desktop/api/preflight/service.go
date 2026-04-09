package preflight

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

// DefaultService is the default implementation of Service.
type DefaultService struct {
	sessions         SessionStore
	jobs             JobStore
	newDryRunRuntime func(manifest *bundlemanifest.Manifest, bundleRoot string, timeout time.Duration) (*RuntimeHandle, error)
	now              func() time.Time
}

// ServiceOption configures a DefaultService.
type ServiceOption func(*DefaultService)

// WithSessionStore sets a custom session store.
func WithSessionStore(store SessionStore) ServiceOption {
	return func(s *DefaultService) {
		s.sessions = store
	}
}

// WithJobStore sets a custom job store.
func WithJobStore(store JobStore) ServiceOption {
	return func(s *DefaultService) {
		s.jobs = store
	}
}

// WithRuntimeFactory sets a custom runtime factory function.
func WithRuntimeFactory(factory func(manifest *bundlemanifest.Manifest, bundleRoot string, timeout time.Duration) (*RuntimeHandle, error)) ServiceOption {
	return func(s *DefaultService) {
		s.newDryRunRuntime = factory
	}
}

// WithTimeProvider sets a custom time provider.
func WithTimeProvider(now func() time.Time) ServiceOption {
	return func(s *DefaultService) {
		s.now = now
	}
}

// NewService creates a new preflight service.
func NewService(opts ...ServiceOption) *DefaultService {
	s := &DefaultService{
		sessions:         NewInMemorySessionStore(),
		jobs:             NewInMemoryJobStore(),
		newDryRunRuntime: defaultDryRunRuntime,
		now:              time.Now,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// StartJanitor starts the background cleanup goroutine.
func (s *DefaultService) StartJanitor() {
	if s.sessions == nil || s.jobs == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			s.sessions.Cleanup()
			s.jobs.Cleanup()
		}
	}()
}

// CreateJob creates a new async preflight job.
func (s *DefaultService) CreateJob() *Job {
	if s.jobs == nil {
		return nil
	}
	return s.jobs.Create()
}

// GetJob retrieves a job by ID.
func (s *DefaultService) GetJob(id string) (*Job, bool) {
	if s.jobs == nil {
		return nil, false
	}
	return s.jobs.Get(id)
}

// GetSession retrieves a session by ID.
func (s *DefaultService) GetSession(id string) (*Session, bool) {
	if s.sessions == nil {
		return nil, false
	}
	return s.sessions.Get(id)
}

// RunBundlePreflight executes synchronous preflight validation.
func (s *DefaultService) RunBundlePreflight(request Request) (*Response, error) {
	if strings.TrimSpace(request.BundleManifestPath) == "" {
		return nil, &StatusError{Status: http.StatusBadRequest, Err: errors.New("bundle_manifest_path is required")}
	}

	if resp, err := s.handleSessionStop(request); resp != nil || err != nil {
		return resp, err
	}

	if request.StatusOnly && strings.TrimSpace(request.SessionID) == "" {
		return nil, &StatusError{Status: http.StatusBadRequest, Err: errors.New("session_id is required for status_only")}
	}

	manifest, manifestPath, err := loadAndValidateManifest(request.BundleManifestPath)
	if err != nil {
		return nil, err
	}

	bundleRoot, err := resolveBundleRoot(request.BundleRoot, manifestPath)
	if err != nil {
		return nil, err
	}

	timeout := preflightTimeout(request.TimeoutSeconds)

	handle, cleanup, err := s.acquireRuntimeHandle(request, manifest, bundleRoot, timeout)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return s.collectPreflightData(handle, manifest, bundleRoot, request, timeout)
}

// handleSessionStop handles the session stop sub-command of a preflight request.
// Returns (nil, nil) if this is not a stop request.
func (s *DefaultService) handleSessionStop(request Request) (*Response, error) {
	if !request.SessionStop || strings.TrimSpace(request.SessionID) == "" {
		return nil, nil
	}
	if s.sessions == nil {
		return nil, &StatusError{Status: http.StatusInternalServerError, Err: errors.New("session store not configured")}
	}
	if s.sessions.Stop(request.SessionID) {
		return &Response{Status: "stopped", SessionID: request.SessionID}, nil
	}
	return nil, &StatusError{Status: http.StatusNotFound, Err: fmt.Errorf("preflight session not found: %s", request.SessionID)}
}

// acquireRuntimeHandle obtains a runtime handle based on the request mode.
// Returns (handle, cleanupFunc, error). cleanupFunc may be nil.
func (s *DefaultService) acquireRuntimeHandle(request Request, manifest *bundlemanifest.Manifest, bundleRoot string, timeout time.Duration) (*RuntimeHandle, func(), error) {
	switch {
	case request.StatusOnly:
		if s.sessions == nil {
			return nil, nil, &StatusError{Status: http.StatusInternalServerError, Err: errors.New("session store not configured")}
		}
		session, ok := s.sessions.Get(request.SessionID)
		if !ok {
			return nil, nil, &StatusError{Status: http.StatusNotFound, Err: fmt.Errorf("preflight session not found: %s", request.SessionID)}
		}
		s.sessions.Refresh(session, request.SessionTTLSeconds)
		return runtimeHandleFromSession(session), nil, nil

	case request.StartServices:
		if s.sessions == nil {
			return nil, nil, &StatusError{Status: http.StatusInternalServerError, Err: errors.New("session store not configured")}
		}
		if existingID := strings.TrimSpace(request.SessionID); existingID != "" {
			s.sessions.Stop(existingID)
		}
		session, err := s.sessions.Create(manifest, bundleRoot, request.SessionTTLSeconds)
		if err != nil {
			return nil, nil, err
		}
		return runtimeHandleFromSession(session), nil, nil

	default:
		handle, err := s.newDryRunRuntime(manifest, bundleRoot, timeout)
		if err != nil {
			return nil, nil, err
		}
		return handle, func() {
			if handle.Cleanup != nil {
				handle.Cleanup()
			}
		}, nil
	}
}

// collectPreflightData gathers all preflight diagnostic data from the runtime.
func (s *DefaultService) collectPreflightData(handle *RuntimeHandle, manifest *bundlemanifest.Manifest, bundleRoot string, request Request, timeout time.Duration) (*Response, error) {
	client := handle.Client

	runtimeStatus, statusErr := client.Status()
	if statusErr == nil && runtimeStatus != nil {
		runtimeStatus.BundleRoot = bundleRoot
	}

	fingerprints := collectServiceFingerprints(manifest, bundleRoot)

	if len(request.Secrets) > 0 {
		if err := client.ApplySecrets(request.Secrets); err != nil {
			return nil, fmt.Errorf("apply secrets: %w", err)
		}
	}

	validation, secrets, err := fetchValidationData(client, request)
	if err != nil {
		return nil, err
	}

	ready, waitedSeconds, err := client.Ready(request, timeout)
	if err != nil {
		return nil, fmt.Errorf("fetch readiness: %w", err)
	}
	ready.SnapshotAt = s.now().Format(time.RFC3339)
	ready.WaitedSeconds = waitedSeconds

	ports, telemetryResp, logTails, err := fetchDiagnostics(client, request)
	if err != nil {
		return nil, err
	}

	checks := buildPreflightChecks(manifest, validation, &ready, secrets, ports, telemetryResp, logTails, request)

	return &Response{
		Status:       "ok",
		Validation:   validation,
		Ready:        &ready,
		Secrets:      secrets,
		Ports:        ports,
		Telemetry:    telemetryResp,
		LogTails:     logTails,
		Checks:       checks,
		Runtime:      runtimeStatus,
		Fingerprints: fingerprints,
		Errors:       statusErrorSlice(statusErr),
		SessionID:    handle.SessionID,
		ExpiresAt:    formatExpiresAt(handle.ExpiresAt),
	}, nil
}
