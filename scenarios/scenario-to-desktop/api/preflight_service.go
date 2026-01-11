package main

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

type preflightRuntimeHandle struct {
	client    PreflightRuntimeClient
	cleanup   func()
	sessionID string
	expiresAt time.Time
}

type PreflightService struct {
	sessions         PreflightSessionStore
	jobs             PreflightJobStore
	newDryRunRuntime func(manifest *bundlemanifest.Manifest, bundleRoot string, timeout time.Duration) (*preflightRuntimeHandle, error)
	now              func() time.Time
}

func NewPreflightService(sessions PreflightSessionStore, jobs PreflightJobStore) *PreflightService {
	return &PreflightService{
		sessions:         sessions,
		jobs:             jobs,
		newDryRunRuntime: defaultDryRunRuntime,
		now:              time.Now,
	}
}

func (s *PreflightService) StartJanitor() {
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

func (s *PreflightService) CreateJob() *preflightJob {
	if s.jobs == nil {
		return nil
	}
	return s.jobs.Create()
}

func (s *PreflightService) GetJob(id string) (*preflightJob, bool) {
	if s.jobs == nil {
		return nil, false
	}
	return s.jobs.Get(id)
}

func (s *PreflightService) GetSession(id string) (*preflightSession, bool) {
	if s.sessions == nil {
		return nil, false
	}
	return s.sessions.Get(id)
}

func (s *PreflightService) RunBundlePreflight(request BundlePreflightRequest) (*BundlePreflightResponse, error) {
	if strings.TrimSpace(request.BundleManifestPath) == "" {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: errors.New("bundle_manifest_path is required")}
	}

	if request.SessionStop && strings.TrimSpace(request.SessionID) != "" {
		if s.sessions == nil {
			return nil, &preflightStatusError{Status: http.StatusInternalServerError, Err: errors.New("session store not configured")}
		}
		if s.sessions.Stop(request.SessionID) {
			return &BundlePreflightResponse{Status: "stopped", SessionID: request.SessionID}, nil
		}
		return nil, &preflightStatusError{Status: http.StatusNotFound, Err: fmt.Errorf("preflight session not found: %s", request.SessionID)}
	}

	if request.StatusOnly && strings.TrimSpace(request.SessionID) == "" {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: errors.New("session_id is required for status_only")}
	}

	manifestPath, err := filepath.Abs(request.BundleManifestPath)
	if err != nil {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("resolve bundle_manifest_path: %w", err)}
	}
	if _, err := os.Stat(manifestPath); err != nil {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("bundle manifest not found: %w", err)}
	}

	manifest, err := bundlemanifest.LoadManifest(manifestPath)
	if err != nil {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("load bundle manifest: %w", err)}
	}
	if err := manifest.Validate(runtime.GOOS, runtime.GOARCH); err != nil {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("validate bundle manifest: %w", err)}
	}

	bundleRoot := strings.TrimSpace(request.BundleRoot)
	if bundleRoot == "" {
		bundleRoot = filepath.Dir(manifestPath)
	}
	bundleRoot, err = filepath.Abs(bundleRoot)
	if err != nil {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("resolve bundle_root: %w", err)}
	}

	timeout := preflightTimeout(request.TimeoutSeconds)

	var handle *preflightRuntimeHandle
	if request.StatusOnly {
		if s.sessions == nil {
			return nil, &preflightStatusError{Status: http.StatusInternalServerError, Err: errors.New("session store not configured")}
		}
		session, ok := s.sessions.Get(request.SessionID)
		if !ok {
			return nil, &preflightStatusError{Status: http.StatusNotFound, Err: fmt.Errorf("preflight session not found: %s", request.SessionID)}
		}
		s.sessions.Refresh(session, request.SessionTTLSeconds)
		handle = runtimeHandleFromSession(session)
	} else if request.StartServices {
		if s.sessions == nil {
			return nil, &preflightStatusError{Status: http.StatusInternalServerError, Err: errors.New("session store not configured")}
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
			if handle.cleanup != nil {
				handle.cleanup()
			}
		}()
	}

	runtimeClient := handle.client

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

	var secrets []BundlePreflightSecret
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

	sessionID := handle.sessionID
	expiresAt := ""
	if !handle.expiresAt.IsZero() {
		expiresAt = handle.expiresAt.Format(time.RFC3339)
	}

	return &BundlePreflightResponse{
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

func (s *PreflightService) RunPreflightJob(jobID string, request BundlePreflightRequest) {
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

	var handle *preflightRuntimeHandle
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
			if handle.cleanup != nil {
				handle.cleanup()
			}
		}()
	}

	s.jobs.SetStep(jobID, "runtime", "pass", "control API online")

	runtimeClient := handle.client

	runtimeStatus, statusErr := runtimeClient.Status()
	if statusErr == nil && runtimeStatus != nil {
		runtimeStatus.BundleRoot = bundleRoot
		s.jobs.SetResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
			return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
				next.Runtime = runtimeStatus
			})
		})
	} else if statusErr != nil {
		s.jobs.SetResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
			return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
				next.Errors = append(next.Errors, fmt.Sprintf("runtime status: %v", statusErr))
			})
		})
	}

	fingerprints := collectServiceFingerprints(manifest, bundleRoot)
	if len(fingerprints) > 0 {
		s.jobs.SetResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
			return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
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
	s.jobs.SetResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
		return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
			next.Validation = validation
		})
	})
	s.jobs.SetStep(jobID, "validation", validationStepState(validation), "")

	secrets, err := runtimeClient.Secrets()
	if err != nil {
		fail("secrets", fmt.Errorf("fetch secrets: %w", err))
		return
	}
	s.jobs.SetResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
		return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
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
	s.jobs.SetResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
		return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
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
	s.jobs.SetResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
		return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
			next.Ports = ports
			next.Telemetry = telemetryResp
			next.LogTails = logTails
			next.Checks = checks
		})
	})
	s.jobs.SetStep(jobID, "diagnostics", diagnosticsStepState(ports, telemetryResp, logTails, request), "")

	s.jobs.SetResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
		return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
			next.SessionID = handle.sessionID
			if !handle.expiresAt.IsZero() {
				next.ExpiresAt = handle.expiresAt.Format(time.RFC3339)
			}
		})
	})

	s.jobs.Finish(jobID, "completed", "")
}

func runtimeHandleFromSession(session *preflightSession) *preflightRuntimeHandle {
	if session == nil {
		return &preflightRuntimeHandle{}
	}
	return &preflightRuntimeHandle{
		client:    newRuntimeClient(session.baseURL, session.token, session.manifest),
		sessionID: session.id,
		expiresAt: session.expiresAt,
	}
}

func defaultDryRunRuntime(manifest *bundlemanifest.Manifest, bundleRoot string, timeout time.Duration) (*preflightRuntimeHandle, error) {
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

	return &preflightRuntimeHandle{
		client:  newRuntimeClient(baseURL, token, manifest),
		cleanup: cleanup,
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

func buildPreflightChecks(manifest *bundlemanifest.Manifest, validation *runtimeapi.BundleValidationResult, ready *BundlePreflightReady, secrets []BundlePreflightSecret, ports map[string]map[string]int, telemetry *BundlePreflightTelemetry, logTails []BundlePreflightLogTail, request BundlePreflightRequest) []BundlePreflightCheck {
	checks := []BundlePreflightCheck{}

	if validation != nil {
		checks = append(checks, BundlePreflightCheck{
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
		checks = append(checks, BundlePreflightCheck{
			ID:     "runtime-ready",
			Step:   "services",
			Name:   "Runtime readiness",
			Status: status,
			Detail: fmt.Sprintf("ready=%t", ready.Ready),
		})
	}

	if len(secrets) > 0 {
		checks = append(checks, BundlePreflightCheck{
			ID:     "secrets",
			Step:   "secrets",
			Name:   "Secrets configured",
			Status: secretsStepState(secrets),
			Detail: fmt.Sprintf("%d secrets", len(secrets)),
		})
	}

	if request.StartServices && (len(ports) > 0 || telemetry != nil || len(logTails) > 0) {
		checks = append(checks, BundlePreflightCheck{
			ID:     "diagnostics",
			Step:   "diagnostics",
			Name:   "Diagnostics captured",
			Status: diagnosticsStepState(ports, telemetry, logTails, request),
			Detail: fmt.Sprintf("%d services", len(ports)),
		})
	}

	assetIssues := map[string]preflightIssue{}
	if validation != nil {
		for _, err := range validation.Errors {
			if err.Code == "asset_missing" {
				key := fmt.Sprintf("%s:%s", err.Service, err.Path)
				assetIssues[key] = preflightIssue{status: "fail", detail: err.Message}
			}
		}
		for _, missing := range validation.MissingAssets {
			key := fmt.Sprintf("%s:%s", missing.ServiceID, missing.Path)
			assetIssues[key] = preflightIssue{status: "fail", detail: "missing asset"}
		}
		for _, warn := range validation.Warnings {
			if warn.Code == "asset_size_exceeded" {
				key := fmt.Sprintf("%s:%s", warn.Service, warn.Path)
				assetIssues[key] = preflightIssue{status: "warning", detail: warn.Message}
			}
		}
	}

	for key, issue := range assetIssues {
		checks = append(checks, BundlePreflightCheck{
			ID:     "asset:" + key,
			Step:   "validation",
			Name:   "Asset validation",
			Status: issue.status,
			Detail: issue.detail,
		})
	}

	return checks
}

func updatePreflightResult(prev *BundlePreflightResponse, update func(next *BundlePreflightResponse)) *BundlePreflightResponse {
	var next BundlePreflightResponse
	if prev != nil {
		next = *prev
	} else {
		next = BundlePreflightResponse{Status: "running"}
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

func secretsStepState(secrets []BundlePreflightSecret) string {
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

func readinessStepState(ready *BundlePreflightReady, request BundlePreflightRequest) string {
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

func diagnosticsStepState(ports map[string]map[string]int, telemetry *BundlePreflightTelemetry, logTails []BundlePreflightLogTail, request BundlePreflightRequest) string {
	if !request.StartServices {
		return "skipped"
	}
	if len(ports) > 0 || (telemetry != nil && telemetry.Path != "") || len(logTails) > 0 {
		return "pass"
	}
	return "warning"
}

func collectServiceFingerprints(manifest *bundlemanifest.Manifest, bundleRoot string) []BundlePreflightServiceFingerprint {
	if manifest == nil {
		return nil
	}
	platform := bundlemanifest.PlatformKey(runtime.GOOS, runtime.GOARCH)
	results := make([]BundlePreflightServiceFingerprint, 0, len(manifest.Services))
	for _, svc := range manifest.Services {
		fp := BundlePreflightServiceFingerprint{
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
