package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"scenario-to-desktop-api/bundle"
	"scenario-to-desktop-api/preflight"
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
			ScenarioName: "test",
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
			ScenarioName: "test",
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
		Config:       &Config{ScenarioName: "test"},
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
