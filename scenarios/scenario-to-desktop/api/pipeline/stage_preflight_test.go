package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"scenario-to-desktop-api/bundle"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/preflight"
	"testing"
	"time"
)

// =============================================================================
// Mock preflight service that captures the request
// =============================================================================

type mockPreflightService struct {
	lastRequest *preflight.Request
	response    *preflight.Response
	err         error
}

func (m *mockPreflightService) RunBundlePreflight(request preflight.Request) (*preflight.Response, error) {
	m.lastRequest = &request
	if m.err != nil {
		return nil, m.err
	}
	if m.response == nil {
		return &preflight.Response{
			Status: "ok",
			Checks: []preflight.Check{
				{Name: "manifest", Status: "pass", Step: "validation"},
			},
		}, nil
	}
	return m.response, nil
}

func (m *mockPreflightService) CreateJob() *preflight.Job {
	return nil
}

func (m *mockPreflightService) GetJob(id string) (*preflight.Job, bool) {
	return nil, false
}

func (m *mockPreflightService) GetSession(id string) (*preflight.Session, bool) {
	return nil, false
}

func (m *mockPreflightService) RunPreflightJob(jobID string, request preflight.Request) {}

func (m *mockPreflightService) StartJanitor() {}

// =============================================================================
// Bundle root path resolution tests
// =============================================================================

// TestPreflightStage_BundleRootResolution verifies that the preflight stage
// correctly passes the bundle directory as the bundle root, not its parent.
// This is a critical test for issue: "binary not found" errors caused by
// incorrect path resolution.
func TestPreflightStage_BundleRootResolution(t *testing.T) {
	// Create a temp directory structure matching production layout
	tmpDir := t.TempDir()
	bundleDir := filepath.Join(tmpDir, "platforms", "electron", "bundle")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a minimal manifest
	manifestPath := filepath.Join(bundleDir, "bundle.json")
	manifestContent := `{
		"schema_version": "v0.1",
		"target": "desktop",
		"app": {"name": "test", "version": "1.0.0"},
		"ipc": {"host": "127.0.0.1", "port": 39200, "auth_token_path": "runtime/token"},
		"telemetry": {"file": "telemetry.jsonl"},
		"services": []
	}`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0o644); err != nil {
		t.Fatal(err)
	}

	mockSvc := &mockPreflightService{}
	mockTime := &mockTimeProvider{now: time.Now().Unix()}
	stage := NewPreflightStage(
		WithPreflightService(mockSvc),
		WithPreflightTimeProvider(mockTime),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName:   "test",
			DeploymentMode: DeploymentModeBundled, // Explicitly set bundled mode for this test
		},
		BundleResult: &bundle.PackageResult{
			BundleDir:    bundleDir,
			ManifestPath: manifestPath,
		},
		Logger: &mockLogger{},
	}

	result := stage.Execute(context.Background(), input)

	// The stage should pass the request to the service
	if mockSvc.lastRequest == nil {
		t.Fatal("preflight service was not called")
	}

	// Critical assertion: BundleRoot should be the bundle directory, not its parent
	// Bug was: bundleRoot = filepath.Dir(bundleDir) which gave the parent
	// Fix is: bundleRoot = bundleDir
	if mockSvc.lastRequest.BundleRoot != bundleDir {
		t.Errorf("BundleRoot incorrect:\n  got:  %s\n  want: %s\n  (parent would be): %s",
			mockSvc.lastRequest.BundleRoot,
			bundleDir,
			filepath.Dir(bundleDir))
	}

	// Verify manifest path is correct
	if mockSvc.lastRequest.BundleManifestPath != manifestPath {
		t.Errorf("BundleManifestPath incorrect:\n  got:  %s\n  want: %s",
			mockSvc.lastRequest.BundleManifestPath,
			manifestPath)
	}

	// Stage should complete successfully
	if result.Status != StatusCompleted {
		t.Errorf("expected status %q, got %q", StatusCompleted, result.Status)
		if result.Error != "" {
			t.Logf("error: %s", result.Error)
		}
	}
}

// TestPreflightStage_BinaryPathResolution verifies the complete path chain
// from manifest binary paths to resolved filesystem paths.
func TestPreflightStage_BinaryPathResolution(t *testing.T) {
	tmpDir := t.TempDir()
	bundleDir := filepath.Join(tmpDir, "platforms", "electron", "bundle")
	binDir := filepath.Join(bundleDir, "bin", "api", "linux-x64")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a dummy binary
	binaryPath := filepath.Join(binDir, "test-api")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\nexit 0"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create manifest with relative binary path
	manifestPath := filepath.Join(bundleDir, "bundle.json")
	manifestContent := `{
		"schema_version": "v0.1",
		"target": "desktop",
		"app": {"name": "test", "version": "1.0.0"},
		"ipc": {"host": "127.0.0.1", "port": 39200, "auth_token_path": "runtime/token"},
		"telemetry": {"file": "telemetry.jsonl"},
		"services": [{
			"id": "test-api",
			"type": "api-binary",
			"binaries": {
				"linux-x64": {"path": "bin/api/linux-x64/test-api"}
			},
			"health": {"type": "tcp", "port_name": "api"},
			"readiness": {"type": "tcp", "port_name": "api"}
		}]
	}`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0o644); err != nil {
		t.Fatal(err)
	}

	mockSvc := &mockPreflightService{}
	mockTime := &mockTimeProvider{now: time.Now().Unix()}
	stage := NewPreflightStage(
		WithPreflightService(mockSvc),
		WithPreflightTimeProvider(mockTime),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName:   "test",
			DeploymentMode: DeploymentModeBundled, // Explicitly set bundled mode for this test
		},
		BundleResult: &bundle.PackageResult{
			BundleDir:    bundleDir,
			ManifestPath: manifestPath,
		},
		Logger: &mockLogger{},
	}

	_ = stage.Execute(context.Background(), input)

	if mockSvc.lastRequest == nil {
		t.Fatal("preflight service was not called")
	}

	// Verify the binary would resolve correctly with the bundle root
	expectedBinaryPath := filepath.Join(mockSvc.lastRequest.BundleRoot, "bin", "api", "linux-x64", "test-api")
	if _, err := os.Stat(expectedBinaryPath); err != nil {
		t.Errorf("Binary path resolution failed:\n  bundleRoot: %s\n  expected binary at: %s\n  error: %v",
			mockSvc.lastRequest.BundleRoot,
			expectedBinaryPath,
			err)
	}
}

// TestPreflightStage_NoBundleResult verifies proper error handling when bundle result is missing.
func TestPreflightStage_NoBundleResult(t *testing.T) {
	mockSvc := &mockPreflightService{}
	mockTime := &mockTimeProvider{now: time.Now().Unix()}
	stage := NewPreflightStage(
		WithPreflightService(mockSvc),
		WithPreflightTimeProvider(mockTime),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName:   "test",
			DeploymentMode: DeploymentModeBundled, // Bundled mode requires bundle result
		},
		BundleResult: nil, // Missing bundle result
		Logger:       &mockLogger{},
	}

	result := stage.Execute(context.Background(), input)

	if result.Status != StatusFailed {
		t.Errorf("expected status %q, got %q", StatusFailed, result.Status)
	}

	if mockSvc.lastRequest != nil {
		t.Error("preflight service should not be called when bundle result is missing")
	}
}

// =============================================================================
// Mock bundleability checker for testing fail-fast behavior
// =============================================================================

type mockBundleabilityChecker struct {
	result *generation.BundleabilityResult
	err    error
}

func (m *mockBundleabilityChecker) CheckBundleability(scenarioName string) (*generation.BundleabilityResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

// TestPreflightStage_Bundleability_FailsFast verifies that unbundleable scenarios fail early.
func TestPreflightStage_Bundleability_FailsFast(t *testing.T) {
	// Create a temp directory structure for the bundle
	tmpDir := t.TempDir()
	bundleDir := filepath.Join(tmpDir, "platforms", "electron", "bundle")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(bundleDir, "bundle.json")
	manifestContent := `{
		"schema_version": "v0.1",
		"target": "desktop",
		"app": {"name": "test", "version": "1.0.0"},
		"ipc": {"host": "127.0.0.1", "port": 39200, "auth_token_path": "runtime/token"},
		"telemetry": {"file": "telemetry.jsonl"},
		"services": []
	}`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0o644); err != nil {
		t.Fatal(err)
	}

	mockSvc := &mockPreflightService{}
	mockTime := &mockTimeProvider{now: time.Now().Unix()}
	mockBundleCheck := &mockBundleabilityChecker{
		result: &generation.BundleabilityResult{
			Bundleable:           false,
			UnbundleableResource: "postgres",
			UnbundleableReason:   "Requires external database server",
			Alternatives:         []string{"sqlite"},
		},
	}

	stage := NewPreflightStage(
		WithPreflightService(mockSvc),
		WithPreflightTimeProvider(mockTime),
		WithBundleabilityChecker(mockBundleCheck),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName:   "test-scenario",
			DeploymentMode: DeploymentModeBundled,
		},
		BundleResult: &bundle.PackageResult{
			BundleDir:    bundleDir,
			ManifestPath: manifestPath,
		},
		Logger: &mockLogger{},
	}

	result := stage.Execute(context.Background(), input)

	// Should fail fast without calling preflight service
	if result.Status != StatusFailed {
		t.Errorf("expected status %q, got %q", StatusFailed, result.Status)
	}

	if mockSvc.lastRequest != nil {
		t.Error("preflight service should not be called when scenario is unbundleable")
	}

	// Error should mention the resource
	if result.Error == "" {
		t.Error("expected error message")
	}
}

// TestPreflightStage_Bundleability_WarnsWithSwap verifies that scenarios with swaps proceed with warning.
func TestPreflightStage_Bundleability_WarnsWithSwap(t *testing.T) {
	tmpDir := t.TempDir()
	bundleDir := filepath.Join(tmpDir, "platforms", "electron", "bundle")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(bundleDir, "bundle.json")
	manifestContent := `{
		"schema_version": "v0.1",
		"target": "desktop",
		"app": {"name": "test", "version": "1.0.0"},
		"ipc": {"host": "127.0.0.1", "port": 39200, "auth_token_path": "runtime/token"},
		"telemetry": {"file": "telemetry.jsonl"},
		"services": []
	}`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0o644); err != nil {
		t.Fatal(err)
	}

	mockSvc := &mockPreflightService{}
	mockTime := &mockTimeProvider{now: time.Now().Unix()}
	mockBundleCheck := &mockBundleabilityChecker{
		result: &generation.BundleabilityResult{
			Bundleable:        true,
			RequiredResources: []string{"postgres"},
			Warnings: []generation.SwapWarning{
				{
					Resource:     "postgres",
					Alternatives: []string{"sqlite"},
					Message:      "Scenario requires 'postgres' which is not supported. Swap to 'sqlite' declared.",
				},
			},
		},
	}

	stage := NewPreflightStage(
		WithPreflightService(mockSvc),
		WithPreflightTimeProvider(mockTime),
		WithBundleabilityChecker(mockBundleCheck),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName:   "test-scenario",
			DeploymentMode: DeploymentModeBundled,
		},
		BundleResult: &bundle.PackageResult{
			BundleDir:    bundleDir,
			ManifestPath: manifestPath,
		},
		Logger: &mockLogger{},
	}

	result := stage.Execute(context.Background(), input)

	// Should proceed (preflight service should be called)
	if mockSvc.lastRequest == nil {
		t.Error("preflight service should be called when scenario has swap declared")
	}

	// Should complete successfully
	if result.Status != StatusCompleted {
		t.Errorf("expected status %q, got %q (error: %s)", StatusCompleted, result.Status, result.Error)
	}

	// Should have warning in logs
	hasWarning := false
	for _, log := range result.Logs {
		if containsSubstring(log, "Warning") && containsSubstring(log, "postgres") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Errorf("expected warning about postgres swap in logs, got: %v", result.Logs)
	}
}

// TestPreflightStage_Bundleability_ExternalServer_SkipsCheck verifies check is skipped for external-server mode.
func TestPreflightStage_Bundleability_ExternalServer_SkipsCheck(t *testing.T) {
	tmpDir := t.TempDir()
	bundleDir := filepath.Join(tmpDir, "platforms", "electron", "bundle")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(bundleDir, "bundle.json")
	manifestContent := `{
		"schema_version": "v0.1",
		"target": "desktop",
		"app": {"name": "test", "version": "1.0.0"},
		"ipc": {"host": "127.0.0.1", "port": 39200, "auth_token_path": "runtime/token"},
		"telemetry": {"file": "telemetry.jsonl"},
		"services": []
	}`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0o644); err != nil {
		t.Fatal(err)
	}

	mockSvc := &mockPreflightService{}
	mockTime := &mockTimeProvider{now: time.Now().Unix()}

	bundleCheckCalled := false
	mockBundleCheck := &mockBundleabilityChecker{
		result: &generation.BundleabilityResult{
			Bundleable:           false,
			UnbundleableResource: "postgres",
			UnbundleableReason:   "Requires external database server",
		},
	}

	// Wrap to track if called
	originalResult := mockBundleCheck.result
	mockBundleCheck = &mockBundleabilityChecker{
		result: originalResult,
	}

	stage := NewPreflightStage(
		WithPreflightService(mockSvc),
		WithPreflightTimeProvider(mockTime),
		WithBundleabilityChecker(&trackingBundleabilityChecker{
			inner:  mockBundleCheck,
			called: &bundleCheckCalled,
		}),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName:   "test-scenario",
			DeploymentMode: DeploymentModeExternalServer, // External server mode - should skip check
		},
		BundleResult: &bundle.PackageResult{
			BundleDir:    bundleDir,
			ManifestPath: manifestPath,
		},
		Logger: &mockLogger{},
	}

	result := stage.Execute(context.Background(), input)

	// Bundleability check should not be called for external-server mode
	if bundleCheckCalled {
		t.Error("bundleability check should not be called for external-server mode")
	}

	// Should be skipped (external-server mode skips preflight entirely)
	if result.Status != StatusSkipped {
		t.Errorf("expected status %q, got %q (error: %s)", StatusSkipped, result.Status, result.Error)
	}
}

func TestPreflightStageRecordsResourceEligibilityWarnings(t *testing.T) {
	stage := NewPreflightStage()
	result := newStageResult(stage.Name(), NewRealTimeProvider())
	input := &StageInput{ResourceDeploymentPlan: &ResourceDeploymentPlan{Resources: []ResourceDeploymentPlanItem{
		{Resource: "vault", OS: "windows", Architecture: "amd64", Support: "unsupported", Eligibility: "ineligible", EligibilityReason: "credential storage is unavailable"},
		{Resource: "postgres", Bundling: "host-required", Requires: []string{"docker"}},
		{Resource: "host-safeguard", Bundling: "prohibited", Limitations: []string{"host mutation is forbidden"}},
	}}}

	stage.appendResourceEligibilityWarnings(input, result)
	if len(result.Logs) != 4 {
		t.Fatalf("warning logs = %v, want stage-start plus three entries", result.Logs)
	}
	if !containsSubstring(result.Logs[1], "vault") || !containsSubstring(result.Logs[1], "windows-amd64") || !containsSubstring(result.Logs[1], "credential storage") {
		t.Fatalf("ineligible warning = %q", result.Logs[1])
	}
	if !containsSubstring(result.Logs[2], "postgres") || !containsSubstring(result.Logs[2], "docker") {
		t.Fatalf("host-required warning = %q", result.Logs[2])
	}
	if !containsSubstring(result.Logs[3], "host-safeguard") || !containsSubstring(result.Logs[3], "prohibited") {
		t.Fatalf("prohibited warning = %q", result.Logs[3])
	}
}

// trackingBundleabilityChecker wraps a checker to track if it was called.
type trackingBundleabilityChecker struct {
	inner  *mockBundleabilityChecker
	called *bool
}

func (t *trackingBundleabilityChecker) CheckBundleability(scenarioName string) (*generation.BundleabilityResult, error) {
	*t.called = true
	return t.inner.CheckBundleability(scenarioName)
}

// containsSubstring is a helper for checking if a string contains a substring.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
