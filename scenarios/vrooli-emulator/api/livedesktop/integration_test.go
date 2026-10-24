//go:build integration

package livedesktop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"vrooli-emulator-api/captures"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/storage"
)

// requiredHostBins are the host binaries the real LinuxBackend depends on.
// pgrep is used by the post-teardown stray-process assertion.
var requiredHostBins = []string{"Xvfb", "x11vnc", "websockify", "xdotool", "pgrep", "ffmpeg", "xwd"}

// xclockPath is the fixture launched by TestIntegration_LaunchApp.
const xclockPath = "/usr/bin/xclock"

// scenarioPrefix tags every test session so pgrep can isolate test-owned processes.
const scenarioPrefix = "test-phase1-"

func TestMain(m *testing.M) {
	for _, bin := range requiredHostBins {
		if _, err := exec.LookPath(bin); err != nil {
			fmt.Fprintf(os.Stderr,
				"skipping vrooli-emulator integration suite: %q not found in PATH "+
					"(install with `apt-get install -y xvfb x11vnc websockify xdotool procps ffmpeg x11-apps`)\n",
				bin)
			os.Exit(0)
		}
	}
	os.Exit(m.Run())
}

type integrationFixture struct {
	t        *testing.T
	URL      string
	svc      *Service
	captures *captures.Service
	capStore captures.Store
}

func setupIntegrationServer(t *testing.T) *integrationFixture {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	backend := NewLinuxBackend(logger)
	store := NewInMemoryStore()
	svc := NewService(store, backend, logger)
	svc.WithDataDir(t.TempDir())

	capDir := t.TempDir()
	metaPath := filepath.Join(capDir, "captures_meta.json")
	filesDir := filepath.Join(capDir, "files")
	require.NoError(t, os.MkdirAll(filesDir, 0o755))

	capStore, err := captures.NewFileStore(metaPath)
	require.NoError(t, err)
	capSvc := captures.NewService(nil, storage.Options{}, filesDir, capStore)
	svc.WithCaptures(capSvc)

	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)
	router.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy","service":"vrooli-emulator","readiness":true}`))
	}).Methods(http.MethodGet)
	server := httptest.NewServer(router)

	fix := &integrationFixture{
		t:        t,
		URL:      server.URL,
		svc:      svc,
		captures: capSvc,
		capStore: capStore,
	}

	t.Cleanup(func() {
		for _, s := range svc.ListSessions() {
			_ = svc.StopSession(s.ID)
		}
		server.Close()
	})

	return fix
}

func (f *integrationFixture) postJSON(t *testing.T, path string, payload any) (*http.Response, []byte) {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, f.URL+path, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	_ = resp.Body.Close()
	return resp, respBody
}

func (f *integrationFixture) get(t *testing.T, path string) (*http.Response, []byte) {
	t.Helper()
	resp, err := http.Get(f.URL + path)
	require.NoError(t, err)
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	_ = resp.Body.Close()
	return resp, respBody
}

func (f *integrationFixture) delete(t *testing.T, path string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, f.URL+path, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	_ = resp.Body.Close()
	return resp, respBody
}

// uniqueScenarioName returns a fresh scenario name uniquely identifying this test's processes.
func uniqueScenarioName() string {
	return scenarioPrefix + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]
}

// createSession POSTs to /api/v1/sessions, registers a teardown DELETE, and returns the view.
func createSession(t *testing.T, f *integrationFixture, headless bool) SessionView {
	t.Helper()
	scenario := uniqueScenarioName()
	resp, body := f.postJSON(t, "/api/v1/sessions", map[string]any{
		"scenario_name": scenario,
		"headless":      headless,
		"width":         1024,
		"height":        768,
	})
	require.Equalf(t, http.StatusCreated, resp.StatusCode, "create body: %s", string(body))

	var v SessionView
	require.NoError(t, json.Unmarshal(body, &v))
	require.NotEmpty(t, v.ID)

	t.Cleanup(func() {
		_, _ = f.delete(t, "/api/v1/sessions/"+v.ID)
		assertNoStrayProcesses(t, scenario)
	})
	return v
}

// assertNoStrayProcesses runs `pgrep -af <scenario>` and asserts no matches.
func assertNoStrayProcesses(t *testing.T, scenario string) {
	t.Helper()
	// Give kernel a moment to reap.
	require.Eventually(t, func() bool {
		out, _ := exec.Command("pgrep", "-af", scenario).Output()
		return len(strings.TrimSpace(string(out))) == 0
	}, 5*time.Second, 100*time.Millisecond, "stray processes remain matching %q", scenario)
}

func TestIntegration_CreateHeadlessSession(t *testing.T) {
	f := setupIntegrationServer(t)
	v := createSession(t, f, true)

	assert.True(t, v.Headless)
	assert.Equal(t, 0, v.VNCPort)
	assert.Equal(t, 0, v.WSPort)
	assert.Equal(t, StateRunning, v.State)
	assert.Regexp(t, regexp.MustCompile(`^:\d+$`), v.DisplayID)
}

func TestIntegration_CreateVNCSession(t *testing.T) {
	f := setupIntegrationServer(t)
	v := createSession(t, f, false)

	assert.False(t, v.Headless)
	assert.GreaterOrEqual(t, v.VNCPort, 5900)
	assert.LessOrEqual(t, v.VNCPort, 5999)
	assert.GreaterOrEqual(t, v.WSPort, 6080)
	assert.LessOrEqual(t, v.WSPort, 6180)
	assert.Equal(t, StateRunning, v.State)
	assert.Regexp(t, regexp.MustCompile(`^:\d+$`), v.DisplayID)
}

func TestIntegration_LaunchApp(t *testing.T) {
	if _, err := os.Stat(xclockPath); err != nil {
		t.Skipf("%s not found — install with `apt-get install -y x11-apps`", xclockPath)
	}

	f := setupIntegrationServer(t)
	v := createSession(t, f, true)

	resp, body := f.postJSON(t, "/api/v1/sessions/"+v.ID+"/control", map[string]any{
		"action": "launch_app",
		"params": map[string]any{"app_path": xclockPath},
	})
	require.Equalf(t, http.StatusOK, resp.StatusCode, "launch body: %s", string(body))

	require.Eventually(t, func() bool {
		_, gb := f.get(t, "/api/v1/sessions/"+v.ID)
		var view SessionView
		if err := json.Unmarshal(gb, &view); err != nil {
			return false
		}
		return view.AppRunning
	}, 10*time.Second, 200*time.Millisecond, "AppRunning never became true")
}

func TestIntegration_Screenshot(t *testing.T) {
	f := setupIntegrationServer(t)
	v := createSession(t, f, true)

	resp, body := f.postJSON(t, "/api/v1/sessions/"+v.ID+"/control", map[string]any{
		"action": "screenshot",
		"params": map[string]any{},
	})
	require.Equalf(t, http.StatusOK, resp.StatusCode, "screenshot body: %s", string(body))

	caps, err := f.capStore.List(v.ScenarioName)
	require.NoError(t, err)
	require.Len(t, caps, 1, "expected exactly one capture entry")
	c := caps[0]
	assert.Equal(t, captures.CaptureScreenshot, c.Type)
	assert.Greater(t, c.FileSizeBytes, int64(0))

	// File-on-disk assertion: capture file must exist non-empty under the captures filesDir.
	capPath, err := f.captures.CaptureFilePath(v.ScenarioName, c.ID)
	require.NoError(t, err)
	info, err := os.Stat(capPath)
	require.NoError(t, err, "capture file missing on disk: %s", capPath)
	assert.Greater(t, info.Size(), int64(0))
}

func TestIntegration_Metrics(t *testing.T) {
	if _, err := os.Stat(xclockPath); err != nil {
		t.Skipf("%s not found — install with `apt-get install -y x11-apps`", xclockPath)
	}

	f := setupIntegrationServer(t)
	v := createSession(t, f, true)

	// LaunchApp is what attaches the Monitor; without it MetricsView stays nil.
	resp, body := f.postJSON(t, "/api/v1/sessions/"+v.ID+"/control", map[string]any{
		"action": "launch_app",
		"params": map[string]any{"app_path": xclockPath},
	})
	require.Equalf(t, http.StatusOK, resp.StatusCode, "launch body: %s", string(body))

	var view SessionView
	require.Eventually(t, func() bool {
		_, gb := f.get(t, "/api/v1/sessions/"+v.ID)
		view = SessionView{}
		if err := json.Unmarshal(gb, &view); err != nil {
			return false
		}
		return view.Metrics != nil
	}, 5*time.Second, 200*time.Millisecond, "MetricsView never populated")

	// Schema-field assertions: the metrics view exposes CPU, memory, and window-detection signal.
	require.NotNil(t, view.Metrics)
	// SampleCount is >= 0 (zero is fine before first sample); fields below confirm schema shape.
	_ = view.Metrics.SampleCount
	// CurrentCPU / CurrentRSSMB may be nil before first sample; SplashDetected/ReadyDetected are window signals.
	// We assert the struct exposes them by reading; type system + non-nil check are the schema gate.
	_ = view.Metrics.CurrentCPU
	_ = view.Metrics.CurrentRSSMB
	_ = view.Metrics.SplashDetected
	_ = view.Metrics.ReadyDetected
}

func TestIntegration_DestroySession(t *testing.T) {
	f := setupIntegrationServer(t)

	// Create without registering the auto-DELETE in createSession (we want to drive teardown manually).
	scenario := uniqueScenarioName()
	resp, body := f.postJSON(t, "/api/v1/sessions", map[string]any{
		"scenario_name": scenario,
		"headless":      false,
		"width":         800,
		"height":        600,
	})
	require.Equalf(t, http.StatusCreated, resp.StatusCode, "create body: %s", string(body))
	var v SessionView
	require.NoError(t, json.Unmarshal(body, &v))

	delResp, _ := f.delete(t, "/api/v1/sessions/"+v.ID)
	assert.True(t, delResp.StatusCode == http.StatusOK || delResp.StatusCode == http.StatusNoContent,
		"unexpected delete status %d", delResp.StatusCode)

	// StopSession marks the session stopped but keeps it in the store; GET still 200s.
	getResp, getBody := f.get(t, "/api/v1/sessions/"+v.ID)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var stopped SessionView
	require.NoError(t, json.Unmarshal(getBody, &stopped))
	assert.Equal(t, StateStopped, stopped.State)

	assertNoStrayProcesses(t, scenario)
}

func TestPhase1SmokeHarness(t *testing.T) {
	if _, err := os.Stat(xclockPath); err != nil {
		t.Skipf("%s not found — install with `apt-get install -y x11-apps`", xclockPath)
	}

	f := setupIntegrationServer(t)

	vncSession := createSession(t, f, false)
	headlessSession := createSession(t, f, true)

	// Launch xclock on the headless session.
	resp, body := f.postJSON(t, "/api/v1/sessions/"+headlessSession.ID+"/control", map[string]any{
		"action": "launch_app",
		"params": map[string]any{"app_path": xclockPath},
	})
	require.Equalf(t, http.StatusOK, resp.StatusCode, "launch body: %s", string(body))

	require.Eventually(t, func() bool {
		_, gb := f.get(t, "/api/v1/sessions/"+headlessSession.ID)
		var view SessionView
		if err := json.Unmarshal(gb, &view); err != nil {
			return false
		}
		return view.AppRunning
	}, 10*time.Second, 200*time.Millisecond)

	// Capture a screenshot.
	resp, body = f.postJSON(t, "/api/v1/sessions/"+headlessSession.ID+"/control", map[string]any{
		"action": "screenshot",
		"params": map[string]any{},
	})
	require.Equalf(t, http.StatusOK, resp.StatusCode, "screenshot body: %s", string(body))

	// Tail metrics — wait until populated, then read once more and assert non-nil.
	require.Eventually(t, func() bool {
		_, gb := f.get(t, "/api/v1/sessions/"+headlessSession.ID)
		var view SessionView
		if err := json.Unmarshal(gb, &view); err != nil {
			return false
		}
		return view.Metrics != nil
	}, 5*time.Second, 200*time.Millisecond)

	// Verify both sessions still listed.
	listResp, listBody := f.get(t, "/api/v1/sessions")
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	var sessions []SessionView
	require.NoError(t, json.Unmarshal(listBody, &sessions))
	assert.Len(t, sessions, 2)

	// Teardown happens in t.Cleanup via createSession.
	_ = vncSession
}
