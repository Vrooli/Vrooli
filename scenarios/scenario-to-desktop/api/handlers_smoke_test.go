package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestSmokeTestStartHandler_NoArtifact(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()
	t.Setenv("VROOLI_ROOT", env.TempDir)

	req := createJSONRequest("POST", "/api/v1/desktop/smoke-test/start", map[string]interface{}{
		"scenario_name": "missing-artifact",
	})
	w := httptest.NewRecorder()

	env.Server.smokeTestStartHandler(w, req)

	assertErrorResponse(t, w, http.StatusNotFound, "no matching installer")
}

func TestSmokeTestStartHandler_FailsFast(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()
	t.Setenv("VROOLI_ROOT", env.TempDir)

	scenario := "smoke-fast"
	artifact := createSmokeTestArtifact(t, env.Server, scenario)
	if artifact == "" {
		t.Fatalf("expected artifact to be created")
	}

	req := createJSONRequest("POST", "/api/v1/desktop/smoke-test/start", map[string]interface{}{
		"scenario_name": scenario,
	})
	w := httptest.NewRecorder()

	env.Server.smokeTestStartHandler(w, req)
	response := assertJSONResponse(t, w, http.StatusOK)
	id := assertFieldExists(t, response, "smoke_test_id").(string)

	status := waitForSmokeStatus(t, env.Server, id)
	if status.Status != "failed" {
		t.Fatalf("expected status failed, got %s", status.Status)
	}
	if status.Error == "" || status.Error == "smoke test cancelled" {
		t.Fatalf("expected missing dependency error, got %q", status.Error)
	}
}

func TestSmokeTestCancelHandler_WithCancel(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()
	t.Setenv("VROOLI_ROOT", env.TempDir)

	id := "cancel-me"
	env.Server.smokeTests.Save(&SmokeTestStatus{
		SmokeTestID:  id,
		ScenarioName: "cancel-scenario",
		Platform:     currentSmokeTestPlatform(),
		Status:       "running",
		StartedAt:    time.Now(),
	})

	cancelCalled := false
	env.Server.setSmokeTestCancel(id, func() {
		cancelCalled = true
	})

	req := createJSONRequest("POST", "/api/v1/desktop/smoke-test/cancel/"+id, nil)
	req = mux.SetURLVars(req, map[string]string{"smoke_test_id": id})
	w := httptest.NewRecorder()

	env.Server.smokeTestCancelHandler(w, req)
	assertJSONResponse(t, w, http.StatusOK)
	if !cancelCalled {
		t.Fatalf("expected cancel to be called")
	}
}

func TestSmokeTestCancelHandler_NoCancel(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()
	t.Setenv("VROOLI_ROOT", env.TempDir)

	id := "cancel-missing"
	env.Server.smokeTests.Save(&SmokeTestStatus{
		SmokeTestID:  id,
		ScenarioName: "cancel-scenario",
		Platform:     currentSmokeTestPlatform(),
		Status:       "running",
		StartedAt:    time.Now(),
	})

	req := createJSONRequest("POST", "/api/v1/desktop/smoke-test/cancel/"+id, nil)
	req = mux.SetURLVars(req, map[string]string{"smoke_test_id": id})
	w := httptest.NewRecorder()

	env.Server.smokeTestCancelHandler(w, req)
	assertJSONResponse(t, w, http.StatusOK)

	status, ok := env.Server.smokeTests.Get(id)
	if !ok {
		t.Fatalf("expected status to exist")
	}
	if status.Status != "failed" {
		t.Fatalf("expected status failed, got %s", status.Status)
	}
	if status.Error == "" {
		t.Fatalf("expected cancel error message")
	}
}

func waitForSmokeStatus(t *testing.T, server *Server, id string) *SmokeTestStatus {
	t.Helper()
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			status, _ := server.smokeTests.Get(id)
			return status
		case <-ticker.C:
			status, ok := server.smokeTests.Get(id)
			if !ok {
				continue
			}
			if status.Status != "running" {
				return status
			}
		}
	}
}

func createSmokeTestArtifact(t *testing.T, server *Server, scenario string) string {
	t.Helper()
	platform := currentSmokeTestPlatform()
	desktopPath := server.standardOutputPath(scenario)
	distPath := filepath.Join(desktopPath, "dist-electron")
	if err := os.MkdirAll(distPath, 0755); err != nil {
		t.Fatalf("failed to create dist path: %v", err)
	}

	filename := "smoke-test"
	switch platform {
	case "win":
		filename += ".exe"
	case "mac":
		filename += ".zip"
	default:
		filename += ".AppImage"
	}
	artifactPath := filepath.Join(distPath, filename)
	if err := os.WriteFile(artifactPath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write artifact: %v", err)
	}

	return artifactPath
}

func TestSmokeTestRunnerCancellation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()
	t.Setenv("VROOLI_ROOT", env.TempDir)

	id := "cancel-runner"
	scenario := "runner-scenario"
	artifact := createSmokeTestArtifact(t, env.Server, scenario)

	env.Server.smokeTests.Save(&SmokeTestStatus{
		SmokeTestID:  id,
		ScenarioName: scenario,
		Platform:     currentSmokeTestPlatform(),
		Status:       "running",
		ArtifactPath: artifact,
		StartedAt:    time.Now(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	env.Server.setSmokeTestCancel(id, cancel)
	cancel()

	env.Server.performSmokeTest(ctx, id, scenario, env.Server.standardOutputPath(scenario), artifact, currentSmokeTestPlatform())

	status, ok := env.Server.smokeTests.Get(id)
	if !ok {
		t.Fatalf("expected status to exist")
	}
	if status.Status != "failed" {
		t.Fatalf("expected status failed after cancel, got %s", status.Status)
	}
}
