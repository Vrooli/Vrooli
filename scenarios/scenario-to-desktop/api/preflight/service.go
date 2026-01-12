package preflight

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	bundleruntime "scenario-to-desktop-runtime"
	runtimeapi "scenario-to-desktop-runtime/api"
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

	if request.SessionStop && strings.TrimSpace(request.SessionID) != "" {
		if s.sessions == nil {
			return nil, &StatusError{Status: http.StatusInternalServerError, Err: errors.New("session store not configured")}
		}
		if s.sessions.Stop(request.SessionID) {
			return &Response{Status: "stopped", SessionID: request.SessionID}, nil
		}
		return nil, &StatusError{Status: http.StatusNotFound, Err: fmt.Errorf("preflight session not found: %s", request.SessionID)}
	}

	if request.StatusOnly && strings.TrimSpace(request.SessionID) == "" {
		return nil, &StatusError{Status: http.StatusBadRequest, Err: errors.New("session_id is required for status_only")}
	}

	manifestPath, err := filepath.Abs(request.BundleManifestPath)
	if err != nil {
		return nil, &StatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("resolve bundle_manifest_path: %w", err)}
	}
	if _, err := os.Stat(manifestPath); err != nil {
		return nil, &StatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("bundle manifest not found: %w", err)}
	}

	manifest, err := bundlemanifest.LoadManifest(manifestPath)
	if err != nil {
		return nil, &StatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("load bundle manifest: %w", err)}
	}
	if err := manifest.Validate(runtime.GOOS, runtime.GOARCH); err != nil {
		return nil, &StatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("validate bundle manifest: %w", err)}
	}

	bundleRoot := strings.TrimSpace(request.BundleRoot)
	if bundleRoot == "" {
		bundleRoot = filepath.Dir(manifestPath)
	}
	bundleRoot, err = filepath.Abs(bundleRoot)
	if err != nil {
		return nil, &StatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("resolve bundle_root: %w", err)}
	}

	timeout := preflightTimeout(request.TimeoutSeconds)

	var handle *RuntimeHandle
	if request.StatusOnly {
		if s.sessions == nil {
			return nil, &StatusError{Status: http.StatusInternalServerError, Err: errors.New("session store not configured")}
		}
		session, ok := s.sessions.Get(request.SessionID)
		if !ok {
			return nil, &StatusError{Status: http.StatusNotFound, Err: fmt.Errorf("preflight session not found: %s", request.SessionID)}
		}
		s.sessions.Refresh(session, request.SessionTTLSeconds)
		handle = runtimeHandleFromSession(session)
	} else if request.StartServices {
		if s.sessions == nil {
			return nil, &StatusError{Status: http.StatusInternalServerError, Err: errors.New("session store not configured")}
		}
		if existingID := strings.TrimSpace(request.SessionID); existingID != "" {
			s.sessions.Stop(existingID)
		}
		session, err := s.sessions.Create(manifest, bundleRoot, request.SessionTTLSeconds)
		if err != nil {
			return nil, err
		}
		handle = runtimeHandleFromSession(session)
	} else {
		handle, err = s.newDryRunRuntime(manifest, bundleRoot, timeout)
		if err != nil {
			return nil, err
		}
		defer func() {
			if handle.Cleanup != nil {
				handle.Cleanup()
			}
		}()
	}

	runtimeClient := handle.Client

	runtimeStatus, statusErr := runtimeClient.Status()
	if statusErr == nil && runtimeStatus != nil {
		runtimeStatus.BundleRoot = bundleRoot
	}

	fingerprints := collectServiceFingerprints(manifest, bundleRoot)

	if len(request.Secrets) > 0 {
		if err := runtimeClient.ApplySecrets(request.Secrets); err != nil {
			return nil, fmt.Errorf("apply secrets: %w", err)
		}
	}

	var validation *runtimeapi.BundleValidationResult
	if !request.StatusOnly {
		validation, err = runtimeClient.Validate()
		if err != nil {
			return nil, fmt.Errorf("validate bundle: %w", err)
		}
	}

	var secrets []Secret
	if !request.StatusOnly {
		secrets, err = runtimeClient.Secrets()
		if err != nil {
			return nil, fmt.Errorf("fetch secrets: %w", err)
		}
	}

	ready, waitedSeconds, err := runtimeClient.Ready(request, timeout)
	if err != nil {
		return nil, fmt.Errorf("fetch readiness: %w", err)
	}
	ready.SnapshotAt = s.now().Format(time.RFC3339)
	ready.WaitedSeconds = waitedSeconds

	ports, err := runtimeClient.Ports()
	if err != nil {
		return nil, fmt.Errorf("fetch ports: %w", err)
	}

	telemetryResp, err := runtimeClient.Telemetry()
	if err != nil {
		return nil, fmt.Errorf("fetch telemetry: %w", err)
	}

	logTails := runtimeClient.LogTails(request)
	checks := buildPreflightChecks(manifest, validation, &ready, secrets, ports, telemetryResp, logTails, request)

	sessionID := handle.SessionID
	expiresAt := ""
	if !handle.ExpiresAt.IsZero() {
		expiresAt = handle.ExpiresAt.Format(time.RFC3339)
	}

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
		Errors: func() []string {
			if statusErr != nil {
				return []string{fmt.Sprintf("runtime status: %v", statusErr)}
			}
			return nil
		}(),
		SessionID: sessionID,
		ExpiresAt: expiresAt,
	}, nil
}

// RunPreflightJob executes asynchronous preflight validation.
func (s *DefaultService) RunPreflightJob(jobID string, request Request) {
	fail := func(stepID string, err error) {
		s.jobs.SetStep(jobID, stepID, "fail", err.Error())
		s.jobs.Finish(jobID, "failed", err.Error())
	}

	if strings.TrimSpace(request.BundleManifestPath) == "" {
		fail("validation", errors.New("bundle_manifest_path is required"))
		return
	}

	s.jobs.SetStep(jobID, "validation", "running", "loading manifest")

	manifestPath, err := filepath.Abs(request.BundleManifestPath)
	if err != nil {
		fail("validation", fmt.Errorf("resolve bundle_manifest_path: %w", err))
		return
	}
	if _, err := os.Stat(manifestPath); err != nil {
		fail("validation", fmt.Errorf("bundle manifest not found: %w", err))
		return
	}

	manifest, err := bundlemanifest.LoadManifest(manifestPath)
	if err != nil {
		fail("validation", fmt.Errorf("load bundle manifest: %w", err))
		return
	}
	if err := manifest.Validate(runtime.GOOS, runtime.GOARCH); err != nil {
		fail("validation", fmt.Errorf("validate bundle manifest: %w", err))
		return
	}

	bundleRoot := strings.TrimSpace(request.BundleRoot)
	if bundleRoot == "" {
		bundleRoot = filepath.Dir(manifestPath)
	}
	bundleRoot, err = filepath.Abs(bundleRoot)
	if err != nil {
		fail("validation", fmt.Errorf("resolve bundle_root: %w", err))
		return
	}

	timeout := preflightTimeout(request.TimeoutSeconds)

	s.jobs.SetStep(jobID, "runtime", "running", "starting runtime control API")

	var handle *RuntimeHandle
	if request.StartServices {
		session, err := s.sessions.Create(manifest, bundleRoot, request.SessionTTLSeconds)
		if err != nil {
			fail("runtime", err)
			return
		}
		handle = runtimeHandleFromSession(session)
	} else {
		handle, err = s.newDryRunRuntime(manifest, bundleRoot, timeout)
		if err != nil {
			fail("runtime", err)
			return
		}
		defer func() {
			if handle.Cleanup != nil {
				handle.Cleanup()
			}
		}()
	}

	s.jobs.SetStep(jobID, "runtime", "pass", "control API online")

	runtimeClient := handle.Client

	runtimeStatus, statusErr := runtimeClient.Status()
	if statusErr == nil && runtimeStatus != nil {
		runtimeStatus.BundleRoot = bundleRoot
		s.jobs.SetResult(jobID, func(prev *Response) *Response {
			return updatePreflightResult(prev, func(next *Response) {
				next.Runtime = runtimeStatus
			})
		})
	} else if statusErr != nil {
		s.jobs.SetResult(jobID, func(prev *Response) *Response {
			return updatePreflightResult(prev, func(next *Response) {
				next.Errors = append(next.Errors, fmt.Sprintf("runtime status: %v", statusErr))
			})
		})
	}

	fingerprints := collectServiceFingerprints(manifest, bundleRoot)
	if len(fingerprints) > 0 {
		s.jobs.SetResult(jobID, func(prev *Response) *Response {
			return updatePreflightResult(prev, func(next *Response) {
				next.Fingerprints = fingerprints
			})
		})
	}

	s.jobs.SetStep(jobID, "secrets", "running", "applying secrets")
	if err := runtimeClient.ApplySecrets(request.Secrets); err != nil {
		fail("secrets", fmt.Errorf("apply secrets: %w", err))
		return
	}

	s.jobs.SetStep(jobID, "validation", "running", "validating bundle")
	validation, err := runtimeClient.Validate()
	if err != nil {
		fail("validation", fmt.Errorf("validate bundle: %w", err))
		return
	}
	s.jobs.SetResult(jobID, func(prev *Response) *Response {
		return updatePreflightResult(prev, func(next *Response) {
			next.Validation = validation
		})
	})
	s.jobs.SetStep(jobID, "validation", validationStepState(validation), "")

	secrets, err := runtimeClient.Secrets()
	if err != nil {
		fail("secrets", fmt.Errorf("fetch secrets: %w", err))
		return
	}
	s.jobs.SetResult(jobID, func(prev *Response) *Response {
		return updatePreflightResult(prev, func(next *Response) {
			next.Secrets = secrets
		})
	})
	s.jobs.SetStep(jobID, "secrets", secretsStepState(secrets), "")

	s.jobs.SetStep(jobID, "services", "running", "checking readiness")
	ready, waitedSeconds, err := runtimeClient.Ready(request, timeout)
	if err != nil {
		fail("services", fmt.Errorf("fetch readiness: %w", err))
		return
	}
	ready.SnapshotAt = s.now().Format(time.RFC3339)
	ready.WaitedSeconds = waitedSeconds
	s.jobs.SetResult(jobID, func(prev *Response) *Response {
		return updatePreflightResult(prev, func(next *Response) {
			next.Ready = &ready
		})
	})
	s.jobs.SetStep(jobID, "services", readinessStepState(&ready, request), "")

	s.jobs.SetStep(jobID, "diagnostics", "running", "collecting diagnostics")
	ports, err := runtimeClient.Ports()
	if err != nil {
		fail("diagnostics", fmt.Errorf("fetch ports: %w", err))
		return
	}
	telemetryResp, err := runtimeClient.Telemetry()
	if err != nil {
		fail("diagnostics", fmt.Errorf("fetch telemetry: %w", err))
		return
	}
	logTails := runtimeClient.LogTails(request)
	checks := buildPreflightChecks(manifest, validation, &ready, secrets, ports, telemetryResp, logTails, request)
	s.jobs.SetResult(jobID, func(prev *Response) *Response {
		return updatePreflightResult(prev, func(next *Response) {
			next.Ports = ports
			next.Telemetry = telemetryResp
			next.LogTails = logTails
			next.Checks = checks
		})
	})
	s.jobs.SetStep(jobID, "diagnostics", diagnosticsStepState(ports, telemetryResp, logTails, request), "")

	s.jobs.SetResult(jobID, func(prev *Response) *Response {
		return updatePreflightResult(prev, func(next *Response) {
			next.SessionID = handle.SessionID
			if !handle.ExpiresAt.IsZero() {
				next.ExpiresAt = handle.ExpiresAt.Format(time.RFC3339)
			}
		})
	})

	s.jobs.Finish(jobID, "completed", "")
}

// Helper functions

func runtimeHandleFromSession(session *Session) *RuntimeHandle {
	if session == nil {
		return &RuntimeHandle{}
	}
	return &RuntimeHandle{
		Client:    NewHTTPRuntimeClient(session.BaseURL, session.Token, session.Manifest),
		SessionID: session.ID,
		ExpiresAt: session.ExpiresAt,
	}
}

func defaultDryRunRuntime(manifest *bundlemanifest.Manifest, bundleRoot string, timeout time.Duration) (*RuntimeHandle, error) {
	appData, err := os.MkdirTemp("", "s2d-preflight-*")
	if err != nil {
		return nil, fmt.Errorf("create preflight app data: %w", err)
	}

	supervisor, err := bundleruntime.NewSupervisor(bundleruntime.Options{
		Manifest:   manifest,
		BundlePath: bundleRoot,
		AppDataDir: appData,
		DryRun:     true,
	})
	if err != nil {
		_ = os.RemoveAll(appData)
		return nil, fmt.Errorf("init runtime: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := supervisor.Start(ctx); err != nil {
		cancel()
		_ = os.RemoveAll(appData)
		return nil, fmt.Errorf("start runtime: %w", err)
	}

	fileTimeout := timeout / 3
	if fileTimeout < 500*time.Millisecond {
		fileTimeout = 500 * time.Millisecond
	}

	tokenPath := bundlemanifest.ResolvePath(appData, manifest.IPC.AuthTokenRel)
	tokenBytes, err := readFileWithRetry(tokenPath, fileTimeout)
	if err != nil {
		cancel()
		_ = supervisor.Shutdown(context.Background())
		_ = os.RemoveAll(appData)
		return nil, fmt.Errorf("read auth token: %w", err)
	}
	token := strings.TrimSpace(string(tokenBytes))

	portPath := filepath.Join(appData, "runtime", "ipc_port")
	port, err := readPortFileWithRetry(portPath, fileTimeout)
	if err != nil {
		cancel()
		_ = supervisor.Shutdown(context.Background())
		_ = os.RemoveAll(appData)
		return nil, fmt.Errorf("read ipc_port: %w", err)
	}

	baseURL := fmt.Sprintf("http://%s:%d", manifest.IPC.Host, port)
	client := &http.Client{Timeout: 2 * time.Second}
	if err := waitForRuntimeHealth(client, baseURL, timeout); err != nil {
		cancel()
		_ = supervisor.Shutdown(context.Background())
		_ = os.RemoveAll(appData)
		return nil, err
	}

	cleanup := func() {
		cancel()
		_ = supervisor.Shutdown(context.Background())
		_ = os.RemoveAll(appData)
	}

	return &RuntimeHandle{
		Client:  NewHTTPRuntimeClient(baseURL, token, manifest),
		Cleanup: cleanup,
	}, nil
}

func preflightTimeout(seconds int) time.Duration {
	timeout := time.Duration(seconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	if timeout > 2*time.Minute {
		timeout = 2 * time.Minute
	}
	return timeout
}

func readFileWithRetry(path string, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	return nil, lastErr
}

func readPortFileWithRetry(path string, timeout time.Duration) (int, error) {
	data, err := readFileWithRetry(path, timeout)
	if err != nil {
		return 0, err
	}
	var port int
	_, err = fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &port)
	if err != nil {
		return 0, err
	}
	return port, nil
}

func waitForRuntimeHealth(client *http.Client, baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/healthz", nil)
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("runtime control API not responding within %s", timeout)
}

func buildPreflightChecks(manifest *bundlemanifest.Manifest, validation *runtimeapi.BundleValidationResult, ready *Ready, secrets []Secret, ports map[string]map[string]int, telemetry *Telemetry, logTails []LogTail, request Request) []Check {
	checks := []Check{}

	if validation != nil {
		checks = append(checks, Check{
			ID:     "bundle-validation",
			Step:   "validation",
			Name:   "Bundle manifest + files",
			Status: validationStepState(validation),
			Detail: fmt.Sprintf("%d errors, %d warnings", len(validation.Errors), len(validation.Warnings)),
		})
	}

	if ready != nil {
		status := "fail"
		if ready.Ready {
			status = "pass"
		}
		checks = append(checks, Check{
			ID:     "runtime-ready",
			Step:   "services",
			Name:   "Runtime readiness",
			Status: status,
			Detail: fmt.Sprintf("ready=%t", ready.Ready),
		})
	}

	if len(secrets) > 0 {
		checks = append(checks, Check{
			ID:     "secrets",
			Step:   "secrets",
			Name:   "Secrets configured",
			Status: secretsStepState(secrets),
			Detail: fmt.Sprintf("%d secrets", len(secrets)),
		})
	}

	if request.StartServices && (len(ports) > 0 || telemetry != nil || len(logTails) > 0) {
		checks = append(checks, Check{
			ID:     "diagnostics",
			Step:   "diagnostics",
			Name:   "Diagnostics captured",
			Status: diagnosticsStepState(ports, telemetry, logTails, request),
			Detail: fmt.Sprintf("%d services", len(ports)),
		})
	}

	assetIssues := map[string]issue{}
	if validation != nil {
		for _, err := range validation.Errors {
			if err.Code == "asset_missing" {
				key := fmt.Sprintf("%s:%s", err.Service, err.Path)
				assetIssues[key] = issue{status: "fail", detail: err.Message}
			}
		}
		for _, missing := range validation.MissingAssets {
			key := fmt.Sprintf("%s:%s", missing.ServiceID, missing.Path)
			assetIssues[key] = issue{status: "fail", detail: "missing asset"}
		}
		for _, warn := range validation.Warnings {
			if warn.Code == "asset_size_exceeded" {
				key := fmt.Sprintf("%s:%s", warn.Service, warn.Path)
				assetIssues[key] = issue{status: "warning", detail: warn.Message}
			}
		}
	}

	for key, iss := range assetIssues {
		checks = append(checks, Check{
			ID:     "asset:" + key,
			Step:   "validation",
			Name:   "Asset validation",
			Status: iss.status,
			Detail: iss.detail,
		})
	}

	return checks
}

func updatePreflightResult(prev *Response, update func(next *Response)) *Response {
	var next Response
	if prev != nil {
		next = *prev
	} else {
		next = Response{Status: "running"}
	}
	update(&next)
	return &next
}

func validationStepState(validation *runtimeapi.BundleValidationResult) string {
	if validation == nil {
		return "skipped"
	}
	if validation.Valid {
		return "pass"
	}
	return "fail"
}

func secretsStepState(secrets []Secret) string {
	if len(secrets) == 0 {
		return "skipped"
	}
	for _, secret := range secrets {
		if secret.Required && !secret.HasValue {
			return "warning"
		}
	}
	return "pass"
}

func readinessStepState(ready *Ready, request Request) string {
	if ready == nil {
		return "skipped"
	}
	if !request.StartServices || request.StatusOnly {
		return "skipped"
	}
	if ready.Ready {
		return "pass"
	}
	return "warning"
}

func diagnosticsStepState(ports map[string]map[string]int, telemetry *Telemetry, logTails []LogTail, request Request) string {
	if !request.StartServices {
		return "skipped"
	}
	if len(ports) > 0 || (telemetry != nil && telemetry.Path != "") || len(logTails) > 0 {
		return "pass"
	}
	return "warning"
}

func collectServiceFingerprints(manifest *bundlemanifest.Manifest, bundleRoot string) []ServiceFingerprint {
	if manifest == nil {
		return nil
	}
	platform := bundlemanifest.PlatformKey(runtime.GOOS, runtime.GOARCH)
	results := make([]ServiceFingerprint, 0, len(manifest.Services))
	for _, svc := range manifest.Services {
		fp := ServiceFingerprint{
			ServiceID: svc.ID,
			Platform:  platform,
		}
		bin, ok := manifest.ResolveBinary(svc)
		if !ok || strings.TrimSpace(bin.Path) == "" {
			fp.Error = "binary not resolved for platform"
			results = append(results, fp)
			continue
		}
		fp.BinaryPath = bin.Path
		resolved := bundlemanifest.ResolvePath(bundleRoot, bin.Path)
		fp.BinaryResolvedPath = resolved
		info, err := os.Stat(resolved)
		if err != nil {
			fp.Error = fmt.Sprintf("stat binary: %v", err)
			results = append(results, fp)
			continue
		}
		if info.IsDir() {
			fp.Error = "binary path is a directory"
			results = append(results, fp)
			continue
		}
		fp.BinarySizeBytes = info.Size()
		fp.BinaryMtime = info.ModTime().Format(time.RFC3339)
		hash, err := sha256File(resolved)
		if err != nil {
			fp.Error = fmt.Sprintf("hash binary: %v", err)
			results = append(results, fp)
			continue
		}
		fp.BinarySHA256 = hash
		results = append(results, fp)
	}
	return results
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
