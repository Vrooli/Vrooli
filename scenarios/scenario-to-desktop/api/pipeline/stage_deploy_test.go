package pipeline

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/deploy"
)

// testDeployFactory creates a factory that returns clients pointed at the test server.
func testDeployFactory(serverURL string) LPBSClientFactory {
	return func(_, token string) *deploy.LPBSClient {
		return deploy.NewLPBSClientWithResolver(
			func(_ context.Context) (string, error) { return serverURL, nil },
			http.DefaultClient,
			token,
		)
	}
}

// newTestDeployServer creates an LPBS test server that handles profile listing,
// profile testing, and proxy requests (presign/commit/apply).
func newTestDeployServer(t *testing.T, s3URL string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/admin/remote-profiles" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode([]deploy.RemoteProfile{
				{ID: 1, Tag: "prod", APIBase: "https://prod.example.com/api/v1", Status: "active"},
			})
		case r.URL.Path == "/api/v1/admin/remote-profiles/1/test" && r.Method == "POST":
			_, _ = w.Write([]byte(`{"ok":true}`))
		case r.URL.Path == "/api/v1/admin/remote-profiles/1/proxy" && r.Method == "POST":
			body, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			_ = json.Unmarshal(body, &payload)
			path, _ := payload["path"].(string)
			switch {
			case strings.Contains(path, "presign-upload"):
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"upload_url": s3URL + "/bucket/object",
					"bucket":     "test-bucket",
					"object_key": "uploads/artifact",
				})
			case strings.Contains(path, "commit"):
				_, _ = w.Write([]byte(`{"id":42}`))
			case strings.Contains(path, "apply"):
				_, _ = w.Write([]byte(`{"ok":true}`))
			default:
				t.Errorf("unexpected proxy path: %s", path)
				w.WriteHeader(http.StatusBadRequest)
			}
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func newTestS3Server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func TestDeployStage_Name(t *testing.T) {
	stage := NewDeployStage()
	if stage.Name() != StageDeploy {
		t.Errorf("expected %q, got %q", StageDeploy, stage.Name())
	}
}

func TestDeployStage_Dependencies(t *testing.T) {
	stage := NewDeployStage()
	deps := stage.Dependencies()
	if len(deps) != 1 || deps[0] != StageSmokeTest {
		t.Errorf("expected [%q], got %v", StageSmokeTest, deps)
	}
}

func TestDeployStage_CanSkip(t *testing.T) {
	stage := NewDeployStage()

	// No deploy config → skip
	if !stage.CanSkip(&StageInput{Config: &Config{}}) {
		t.Error("expected CanSkip=true when DeployConfig is nil")
	}

	// With deploy config → don't skip
	if stage.CanSkip(&StageInput{Config: &Config{DeployConfig: &DeployConfig{AppKey: "test"}}}) {
		t.Error("expected CanSkip=false when DeployConfig is set")
	}
}

func TestDeployStage_Execute_NilConfig(t *testing.T) {
	stage := NewDeployStage(WithDeployTimeProvider(newMockTP()))

	result := stage.Execute(context.Background(), &StageInput{
		Config: &Config{},
	})

	if result.Status != StatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
	if !strings.Contains(result.Error, "deploy config not set") {
		t.Errorf("unexpected error: %s", result.Error)
	}
}

func TestDeployStage_Execute_MissingServiceToken(t *testing.T) {
	// Ensure env var is unset and secrets file lookup cannot find a value.
	t.Setenv("LPBS_SERVICE_SECRET", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("VROOLI_ROOT", t.TempDir())
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmpWD := t.TempDir()
	if err := os.Chdir(tmpWD); err != nil {
		t.Fatalf("chdir tempdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	stage := NewDeployStage(WithDeployTimeProvider(newMockTP()))

	result := stage.Execute(context.Background(), &StageInput{
		Config: &Config{
			DeployConfig: &DeployConfig{
				ScenarioName:  "lpbs",
				RemoteProfile: "prod",
				AppKey:        "my-app",
			},
		},
	})

	if result.Status != StatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
	if !strings.Contains(result.Error, "LPBS_SERVICE_SECRET") {
		t.Errorf("unexpected error: %s", result.Error)
	}
}

func TestDeployStage_Execute_InlineConfig(t *testing.T) {
	s3Server := newTestS3Server()
	defer s3Server.Close()

	lpbsServer := newTestDeployServer(t, s3Server.URL)
	defer lpbsServer.Close()

	// Create temp artifact
	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "app.exe")
	_ = os.WriteFile(artifactPath, []byte("binary"), 0o644)

	t.Setenv("LPBS_SERVICE_SECRET", "test-token")

	stage := NewDeployStage(
		WithDeployClientFactory(testDeployFactory(lpbsServer.URL)),
		WithDeployTimeProvider(newMockTP()),
	)

	result := stage.Execute(context.Background(), &StageInput{
		Config: &Config{
			Version: "1.0.0",
			DeployConfig: &DeployConfig{
				ScenarioName:  "lpbs",
				RemoteProfile: "prod",
				AppKey:        "my-app",
			},
		},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{
				"win": {Status: BuildStatusReady, Artifact: artifactPath},
			},
		},
	})

	if result.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s: %s", result.Status, result.Error)
	}

	// Verify details
	deployResult, ok := result.Details.(*DeployResult)
	if !ok {
		t.Fatalf("expected *DeployResult, got %T", result.Details)
	}
	if len(deployResult.Artifacts) != 1 {
		t.Errorf("expected 1 artifact, got %d", len(deployResult.Artifacts))
	}
	if deployResult.Artifacts[0].ArtifactID != 42 {
		t.Errorf("expected artifact ID 42, got %d", deployResult.Artifacts[0].ArtifactID)
	}
	if deployResult.UpdateURL == "" {
		t.Error("expected update URL to be derived")
	}
}

func TestDeployStage_Execute_SavedTarget(t *testing.T) {
	s3Server := newTestS3Server()
	defer s3Server.Close()

	lpbsServer := newTestDeployServer(t, s3Server.URL)
	defer lpbsServer.Close()

	// Create temp artifact
	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "app.exe")
	_ = os.WriteFile(artifactPath, []byte("binary"), 0o644)

	// Set up target repo
	targetDir := t.TempDir()
	repo := deploy.NewTargetRepository(filepath.Join(targetDir, "deploy-targets.json"))
	_ = repo.Save("production", &deploy.DeployTarget{
		Label:         "Production",
		ScenarioName:  "lpbs",
		RemoteProfile: "prod",
	})

	t.Setenv("LPBS_SERVICE_SECRET", "test-token")

	stage := NewDeployStage(
		WithDeployClientFactory(testDeployFactory(lpbsServer.URL)),
		WithDeployTargetRepo(repo),
		WithDeployTimeProvider(newMockTP()),
	)

	result := stage.Execute(context.Background(), &StageInput{
		Config: &Config{
			Version: "1.0.0",
			DeployConfig: &DeployConfig{
				TargetName: "production",
				AppKey:     "my-app",
			},
		},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{
				"linux": {Status: BuildStatusReady, Artifact: artifactPath},
			},
		},
	})

	if result.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s: %s", result.Status, result.Error)
	}
}

func TestDeployStage_Execute_NoArtifacts(t *testing.T) {
	s3Server := newTestS3Server()
	defer s3Server.Close()

	lpbsServer := newTestDeployServer(t, s3Server.URL)
	defer lpbsServer.Close()

	t.Setenv("LPBS_SERVICE_SECRET", "test-token")

	stage := NewDeployStage(
		WithDeployClientFactory(testDeployFactory(lpbsServer.URL)),
		WithDeployTimeProvider(newMockTP()),
	)

	result := stage.Execute(context.Background(), &StageInput{
		Config: &Config{
			DeployConfig: &DeployConfig{
				ScenarioName:  "lpbs",
				RemoteProfile: "prod",
				AppKey:        "my-app",
			},
		},
		// No BuildResult → no artifacts
	})

	if result.Status != StatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
	if !strings.Contains(result.Error, "no built artifacts") {
		t.Errorf("unexpected error: %s", result.Error)
	}
}

func TestDeployStage_Execute_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	stage := NewDeployStage(WithDeployTimeProvider(newMockTP()))

	result := stage.Execute(ctx, &StageInput{
		Config: &Config{
			DeployConfig: &DeployConfig{
				ScenarioName:  "lpbs",
				RemoteProfile: "prod",
				AppKey:        "my-app",
			},
		},
	})

	if result.Status != StatusCancelled {
		t.Errorf("expected cancelled, got %s", result.Status)
	}
}

func TestDeployStage_Execute_MissingInlineConfig(t *testing.T) {
	t.Setenv("LPBS_SERVICE_SECRET", "test-token")

	stage := NewDeployStage(WithDeployTimeProvider(newMockTP()))

	// Missing scenario_name
	result := stage.Execute(context.Background(), &StageInput{
		Config: &Config{
			DeployConfig: &DeployConfig{
				RemoteProfile: "prod",
				AppKey:        "my-app",
			},
		},
	})
	if result.Status != StatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
	if !strings.Contains(result.Error, "scenario_name") {
		t.Errorf("expected scenario_name error, got: %s", result.Error)
	}

	// Missing remote_profile
	result = stage.Execute(context.Background(), &StageInput{
		Config: &Config{
			DeployConfig: &DeployConfig{
				ScenarioName: "lpbs",
				AppKey:       "my-app",
			},
		},
	})
	if result.Status != StatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
	if !strings.Contains(result.Error, "remote_profile") {
		t.Errorf("expected remote_profile error, got: %s", result.Error)
	}
}

func TestDeployStage_Execute_TargetNotFound(t *testing.T) {
	t.Setenv("LPBS_SERVICE_SECRET", "test-token")

	targetDir := t.TempDir()
	repo := deploy.NewTargetRepository(filepath.Join(targetDir, "deploy-targets.json"))

	stage := NewDeployStage(
		WithDeployTargetRepo(repo),
		WithDeployTimeProvider(newMockTP()),
	)

	result := stage.Execute(context.Background(), &StageInput{
		Config: &Config{
			DeployConfig: &DeployConfig{
				TargetName: "nonexistent",
				AppKey:     "my-app",
			},
		},
	})
	if result.Status != StatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
	if !strings.Contains(result.Error, "nonexistent") {
		t.Errorf("expected target name in error, got: %s", result.Error)
	}
}

func TestDeployStage_Execute_RemoteProfileTestFails(t *testing.T) {
	// LPBS server that fails profile test
	lpbsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/admin/remote-profiles" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode([]deploy.RemoteProfile{
				{ID: 1, Tag: "prod"},
			})
		case r.URL.Path == "/api/v1/admin/remote-profiles/1/test" && r.Method == "POST":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("session expired"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer lpbsServer.Close()

	t.Setenv("LPBS_SERVICE_SECRET", "test-token")

	stage := NewDeployStage(
		WithDeployClientFactory(testDeployFactory(lpbsServer.URL)),
		WithDeployTimeProvider(newMockTP()),
	)

	result := stage.Execute(context.Background(), &StageInput{
		Config: &Config{
			DeployConfig: &DeployConfig{
				ScenarioName:  "lpbs",
				RemoteProfile: "prod",
				AppKey:        "my-app",
			},
		},
	})

	if result.Status != StatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
	if !strings.Contains(result.Error, "deploy failed") {
		t.Errorf("expected deploy failed error, got: %s", result.Error)
	}
}

// newMockTP creates a mock time provider for deploy stage tests.
func newMockTP() *mockTimeProvider {
	return &mockTimeProvider{now: 1000}
}

// testDMClientFactory creates a factory pointing at a test DM server.
func testDMClientFactory(serverURL string) DMClientFactory {
	return func(_ context.Context) (*deploy.DMClient, error) {
		return deploy.NewDMClientWithResolver(
			func(_ context.Context) (string, error) { return serverURL, nil },
			http.DefaultClient,
		), nil
	}
}

// newTestDMServer creates a deployment-manager test server.
// gateReady controls whether the release gate is immediately ready or blocked.
func newTestDMServer(t *testing.T, gateReady bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/approvals") && r.Method == "POST":
			_ = json.NewEncoder(w).Encode(deploy.Approval{
				ID:       "appr-1",
				Platform: "win",
				Status:   "pending",
			})
		case strings.Contains(r.URL.Path, "/release-gate") && r.Method == "GET":
			if gateReady {
				_ = json.NewEncoder(w).Encode(deploy.ReleaseGateStatus{
					Ready: true,
					Platforms: []deploy.PlatformGateStatus{
						{Platform: "win", Status: "approved", Ready: true},
					},
				})
			} else {
				_ = json.NewEncoder(w).Encode(deploy.ReleaseGateStatus{
					Ready: false,
					Platforms: []deploy.PlatformGateStatus{
						{Platform: "win", Status: "pending", Ready: false},
					},
				})
			}
		default:
			t.Errorf("unexpected DM request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestDeployStage_Execute_GateReady(t *testing.T) {
	s3Server := newTestS3Server()
	defer s3Server.Close()

	lpbsServer := newTestDeployServer(t, s3Server.URL)
	defer lpbsServer.Close()

	dmServer := newTestDMServer(t, true)
	defer dmServer.Close()

	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "app.exe")
	_ = os.WriteFile(artifactPath, []byte("binary"), 0o644)

	t.Setenv("LPBS_SERVICE_SECRET", "test-token")

	stage := NewDeployStage(
		WithDeployClientFactory(testDeployFactory(lpbsServer.URL)),
		WithDeployDMClientFactory(testDMClientFactory(dmServer.URL)),
		WithDeployTimeProvider(newMockTP()),
	)

	result := stage.Execute(context.Background(), &StageInput{
		Config: &Config{
			Version: "1.0.0",
			DeployConfig: &DeployConfig{
				ScenarioName:               "lpbs",
				RemoteProfile:              "prod",
				AppKey:                     "my-app",
				DeploymentManagerProfileID: "profile-1",
			},
		},
		Provenance: &BuildProvenance{GitCommitHash: "abc123"},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{
				"win": {Status: BuildStatusReady, Artifact: artifactPath},
			},
		},
	})

	if result.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s: %s", result.Status, result.Error)
	}
}

func TestDeployStage_Execute_GateBlocked_Timeout(t *testing.T) {
	s3Server := newTestS3Server()
	defer s3Server.Close()

	lpbsServer := newTestDeployServer(t, s3Server.URL)
	defer lpbsServer.Close()

	dmServer := newTestDMServer(t, false)
	defer dmServer.Close()

	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "app.exe")
	_ = os.WriteFile(artifactPath, []byte("binary"), 0o644)

	t.Setenv("LPBS_SERVICE_SECRET", "test-token")

	stage := NewDeployStage(
		WithDeployClientFactory(testDeployFactory(lpbsServer.URL)),
		WithDeployDMClientFactory(testDMClientFactory(dmServer.URL)),
		WithDeployTimeProvider(newMockTP()),
	)

	result := stage.Execute(context.Background(), &StageInput{
		Config: &Config{
			Version: "1.0.0",
			DeployConfig: &DeployConfig{
				ScenarioName:               "lpbs",
				RemoteProfile:              "prod",
				AppKey:                     "my-app",
				DeploymentManagerProfileID: "profile-1",
				GateTimeout:                "100ms",
				GatePollInterval:           "10ms",
			},
		},
		Provenance: &BuildProvenance{GitCommitHash: "abc123"},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{
				"win": {Status: BuildStatusReady, Artifact: artifactPath},
			},
		},
	})

	if result.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", result.Status)
	}
	if !strings.Contains(result.Error, "timed out") {
		t.Errorf("expected timeout error, got: %s", result.Error)
	}
}

func TestDeployStage_Execute_DMUnreachable(t *testing.T) {
	s3Server := newTestS3Server()
	defer s3Server.Close()

	lpbsServer := newTestDeployServer(t, s3Server.URL)
	defer lpbsServer.Close()

	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "app.exe")
	_ = os.WriteFile(artifactPath, []byte("binary"), 0o644)

	t.Setenv("LPBS_SERVICE_SECRET", "test-token")

	stage := NewDeployStage(
		WithDeployClientFactory(testDeployFactory(lpbsServer.URL)),
		WithDeployDMClientFactory(testDMClientFactory("http://127.0.0.1:1")),
		WithDeployTimeProvider(newMockTP()),
	)

	result := stage.Execute(context.Background(), &StageInput{
		Config: &Config{
			Version: "1.0.0",
			DeployConfig: &DeployConfig{
				ScenarioName:               "lpbs",
				RemoteProfile:              "prod",
				AppKey:                     "my-app",
				DeploymentManagerProfileID: "profile-1",
			},
		},
		Provenance: &BuildProvenance{GitCommitHash: "abc123"},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{
				"win": {Status: BuildStatusReady, Artifact: artifactPath},
			},
		},
	})

	if result.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", result.Status)
	}
}

func TestDeployStage_Execute_NoProfileID_SkipsGate(t *testing.T) {
	s3Server := newTestS3Server()
	defer s3Server.Close()

	lpbsServer := newTestDeployServer(t, s3Server.URL)
	defer lpbsServer.Close()

	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "app.exe")
	_ = os.WriteFile(artifactPath, []byte("binary"), 0o644)

	t.Setenv("LPBS_SERVICE_SECRET", "test-token")

	stage := NewDeployStage(
		WithDeployClientFactory(testDeployFactory(lpbsServer.URL)),
		WithDeployTimeProvider(newMockTP()),
	)

	result := stage.Execute(context.Background(), &StageInput{
		Config: &Config{
			Version: "1.0.0",
			DeployConfig: &DeployConfig{
				ScenarioName:  "lpbs",
				RemoteProfile: "prod",
				AppKey:        "my-app",
			},
		},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{
				"win": {Status: BuildStatusReady, Artifact: artifactPath},
			},
		},
	})

	if result.Status != StatusCompleted {
		t.Fatalf("expected completed (gate skipped), got %s: %s", result.Status, result.Error)
	}
}

func TestDeployStage_Execute_GateBlocked_ThenClears(t *testing.T) {
	s3Server := newTestS3Server()
	defer s3Server.Close()

	lpbsServer := newTestDeployServer(t, s3Server.URL)
	defer lpbsServer.Close()

	var callCount int
	dmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/approvals") && r.Method == "POST":
			_ = json.NewEncoder(w).Encode(deploy.Approval{
				ID:       "appr-1",
				Platform: "win",
				Status:   "pending",
			})
		case strings.Contains(r.URL.Path, "/release-gate") && r.Method == "GET":
			callCount++
			if callCount >= 3 {
				_ = json.NewEncoder(w).Encode(deploy.ReleaseGateStatus{
					Ready: true,
					Platforms: []deploy.PlatformGateStatus{
						{Platform: "win", Status: "approved", Ready: true},
					},
				})
			} else {
				_ = json.NewEncoder(w).Encode(deploy.ReleaseGateStatus{
					Ready: false,
					Platforms: []deploy.PlatformGateStatus{
						{Platform: "win", Status: "pending", Ready: false},
					},
				})
			}
		default:
			t.Errorf("unexpected DM request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer dmServer.Close()

	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "app.exe")
	_ = os.WriteFile(artifactPath, []byte("binary"), 0o644)

	t.Setenv("LPBS_SERVICE_SECRET", "test-token")

	stage := NewDeployStage(
		WithDeployClientFactory(testDeployFactory(lpbsServer.URL)),
		WithDeployDMClientFactory(testDMClientFactory(dmServer.URL)),
		WithDeployTimeProvider(newMockTP()),
	)

	var gateBlocked bool
	result := stage.Execute(context.Background(), &StageInput{
		Config: &Config{
			Version: "1.0.0",
			DeployConfig: &DeployConfig{
				ScenarioName:               "lpbs",
				RemoteProfile:              "prod",
				AppKey:                     "my-app",
				DeploymentManagerProfileID: "profile-1",
				GateTimeout:                "30s",
				GatePollInterval:           "10ms",
			},
		},
		Provenance: &BuildProvenance{GitCommitHash: "abc123"},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{
				"win": {Status: BuildStatusReady, Artifact: artifactPath},
			},
		},
		GateStateReporter: func(blocked bool) {
			gateBlocked = blocked
		},
	})

	if result.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s: %s", result.Status, result.Error)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 gate checks, got %d", callCount)
	}
	_ = gateBlocked
}
