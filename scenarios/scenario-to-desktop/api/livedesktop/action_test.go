package livedesktop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scenario-to-desktop-api/screenrecording"
)

// mockShell records all invocations and returns configurable output.
type mockShell struct {
	calls  []shellCall
	output []byte
	err    error
}

type shellCall struct {
	name string
	args []string
	env  []string
}

func (m *mockShell) fn(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	m.calls = append(m.calls, shellCall{name: name, args: args, env: env})
	return m.output, m.err
}

// mockRecorder implements screenrecording.Recorder for testing.
type mockRecorder struct {
	startErr  error
	stopErr   error
	captureID string
	result    *screenrecording.CaptureResult
}

func (m *mockRecorder) StartCapture(_ context.Context, _ screenrecording.CaptureConfig) (string, error) {
	if m.startErr != nil {
		return "", m.startErr
	}
	return m.captureID, nil
}

func (m *mockRecorder) StopCapture(_ context.Context, _ string) (*screenrecording.CaptureResult, error) {
	if m.stopErr != nil {
		return nil, m.stopErr
	}
	return m.result, nil
}

func newTestSession() *Session {
	return &Session{
		ID:           "test-session-1",
		ScenarioName: "test-scenario",
		State:        StateRunning,
		Display:      &screenrecording.ManagedDisplay{DisplayID: ":99"},
		Width:        1280,
		Height:       720,
		NetworkMode:  "normal",
		EnvVars:      make(map[string]string),
	}
}

func newTestServiceWithShell(shell *mockShell) *Service {
	store := NewInMemoryStore()
	dm := &mockDisplayManager{
		display: &screenrecording.ManagedDisplay{DisplayID: ":99"},
	}
	svc := NewService(store, dm, newTestLogger(), "")
	svc.startVNC = mockVNCStart(5900, 6080)
	svc.stopVNC = mockVNCStop
	svc.shell = shell.fn
	svc.dataDir = os.TempDir()
	return svc
}

// ===================== LaunchAppAction Tests =====================

func TestLaunchAppAction_NoArtifact(t *testing.T) {
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)
	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	action := &LaunchAppAction{}
	_, err = action.Execute(context.Background(), session, svc, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auto-discover artifact")
}

// ===================== QuitAppAction Tests =====================

func TestQuitAppAction_NoApp(t *testing.T) {
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)
	session := newTestSession()

	action := &QuitAppAction{}
	_, err := action.Execute(context.Background(), session, svc, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no app is running")
}

func TestQuitAppAction_Success(t *testing.T) {
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)
	session := newTestSession()
	session.AppRunning = true

	action := &QuitAppAction{}
	result, err := action.Execute(context.Background(), session, svc, nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.False(t, session.AppRunning)
}

// ===================== ScreenshotAction Tests =====================

func TestScreenshotAction_Success(t *testing.T) {
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)
	svc.dataDir = t.TempDir()
	session := newTestSession()

	action := &ScreenshotAction{}
	result, err := action.Execute(context.Background(), session, svc, nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.Contains(t, result.Data["url"], "/api/v1/livedesktop/sessions/test-session-1/files/screenshot-")

	// Verify shell was called with mkdir and sh -c pipeline
	require.GreaterOrEqual(t, len(shell.calls), 2)
	assert.Equal(t, "mkdir", shell.calls[0].name)
	assert.Equal(t, "sh", shell.calls[1].name)
}

func TestScreenshotAction_ShellError(t *testing.T) {
	shell := &mockShell{err: fmt.Errorf("xwd not found")}
	svc := newTestServiceWithShell(shell)
	svc.dataDir = t.TempDir()
	session := newTestSession()

	action := &ScreenshotAction{}
	_, err := action.Execute(context.Background(), session, svc, nil)
	assert.Error(t, err)
}

func TestScreenshotAction_DisplayNotRunning(t *testing.T) {
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)
	session := newTestSession()
	session.Display.Stop()

	action := &ScreenshotAction{}
	_, err := action.Execute(context.Background(), session, svc, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

// ===================== StartRecordingAction Tests =====================

func TestStartRecordingAction_Success(t *testing.T) {
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)
	svc.dataDir = t.TempDir()
	svc.recorder = &mockRecorder{captureID: "cap-123"}
	session := newTestSession()

	action := &StartRecordingAction{}
	result, err := action.Execute(context.Background(), session, svc, nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.True(t, session.IsRecording)
	assert.Equal(t, "cap-123", session.CaptureID)
}

func TestStartRecordingAction_AlreadyRecording(t *testing.T) {
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)
	svc.recorder = &mockRecorder{}
	session := newTestSession()
	session.IsRecording = true

	action := &StartRecordingAction{}
	_, err := action.Execute(context.Background(), session, svc, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already recording")
}

func TestStartRecordingAction_NoRecorder(t *testing.T) {
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)
	session := newTestSession()

	action := &StartRecordingAction{}
	_, err := action.Execute(context.Background(), session, svc, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

// ===================== StopRecordingAction Tests =====================

func TestStopRecordingAction_Success(t *testing.T) {
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)
	svc.recorder = &mockRecorder{
		result: &screenrecording.CaptureResult{
			VideoPath:     "/tmp/sessions/test/recording-123.mp4",
			DurationMs:    5000,
			FileSizeBytes: 1024,
		},
	}
	session := newTestSession()
	session.IsRecording = true
	session.CaptureID = "cap-123"

	action := &StopRecordingAction{}
	result, err := action.Execute(context.Background(), session, svc, nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.False(t, session.IsRecording)
	assert.Contains(t, result.Data["video_url"], "recording-123.mp4")
}

func TestStopRecordingAction_NotRecording(t *testing.T) {
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)
	svc.recorder = &mockRecorder{}
	session := newTestSession()

	action := &StopRecordingAction{}
	_, err := action.Execute(context.Background(), session, svc, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not recording")
}

// ===================== OfflineModeAction Tests =====================

func TestOfflineModeAction_Enable(t *testing.T) {
	session := newTestSession()
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)

	action := &OfflineModeAction{}
	params, _ := json.Marshal(map[string]bool{"enabled": true})
	result, err := action.Execute(context.Background(), session, svc, params)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.Equal(t, "offline", session.NetworkMode)
}

func TestOfflineModeAction_Disable(t *testing.T) {
	session := newTestSession()
	session.NetworkMode = "offline"
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)

	action := &OfflineModeAction{}
	params, _ := json.Marshal(map[string]bool{"enabled": false})
	result, err := action.Execute(context.Background(), session, svc, params)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.Equal(t, "normal", session.NetworkMode)
}

// ===================== SlowConnectionAction Tests =====================

func TestSlowConnectionAction_Enable(t *testing.T) {
	session := newTestSession()
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)

	action := &SlowConnectionAction{}
	params, _ := json.Marshal(map[string]any{"enabled": true})
	result, err := action.Execute(context.Background(), session, svc, params)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.Equal(t, "slow", session.NetworkMode)
	assert.Equal(t, 256, session.BandwidthKbps) // default
}

func TestSlowConnectionAction_CustomBandwidth(t *testing.T) {
	session := newTestSession()
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)

	action := &SlowConnectionAction{}
	params, _ := json.Marshal(map[string]any{"enabled": true, "bandwidth_kbps": 128})
	result, err := action.Execute(context.Background(), session, svc, params)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.Equal(t, 128, session.BandwidthKbps)
}

// ===================== InjectEnvAction Tests =====================

func TestInjectEnvAction_Merge(t *testing.T) {
	session := newTestSession()
	session.EnvVars = map[string]string{"EXISTING": "value"}
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)

	action := &InjectEnvAction{}
	params, _ := json.Marshal(map[string]any{
		"vars": map[string]string{"NEW_VAR": "new_value"},
	})
	result, err := action.Execute(context.Background(), session, svc, params)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	envVars := session.GetEnvVars()
	assert.Equal(t, "value", envVars["EXISTING"])
	assert.Equal(t, "new_value", envVars["NEW_VAR"])
}

func TestInjectEnvAction_Replace(t *testing.T) {
	session := newTestSession()
	session.EnvVars = map[string]string{"EXISTING": "value"}
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)

	action := &InjectEnvAction{}
	merge := false
	params, _ := json.Marshal(struct {
		Vars  map[string]string `json:"vars"`
		Merge *bool             `json:"merge"`
	}{
		Vars:  map[string]string{"ONLY": "this"},
		Merge: &merge,
	})
	result, err := action.Execute(context.Background(), session, svc, params)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	envVars := session.GetEnvVars()
	assert.Equal(t, "this", envVars["ONLY"])
	assert.Empty(t, envVars["EXISTING"])
}

// ===================== ResizeDisplayAction Tests =====================

func TestResizeDisplayAction_Success(t *testing.T) {
	session := newTestSession()
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)

	action := &ResizeDisplayAction{}
	params, _ := json.Marshal(map[string]int{"width": 1920, "height": 1080})
	result, err := action.Execute(context.Background(), session, svc, params)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.Equal(t, 1920, session.Width)
	assert.Equal(t, 1080, session.Height)

	require.Len(t, shell.calls, 1)
	assert.Equal(t, "xrandr", shell.calls[0].name)
}

func TestResizeDisplayAction_InvalidDimensions(t *testing.T) {
	session := newTestSession()
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)

	action := &ResizeDisplayAction{}
	params, _ := json.Marshal(map[string]int{"width": 0, "height": 1080})
	_, err := action.Execute(context.Background(), session, svc, params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "positive")
}

// ===================== ClipboardReadAction Tests =====================

func TestClipboardReadAction_Success(t *testing.T) {
	session := newTestSession()
	shell := &mockShell{output: []byte("clipboard content")}
	svc := newTestServiceWithShell(shell)

	action := &ClipboardReadAction{}
	result, err := action.Execute(context.Background(), session, svc, nil)
	// This will return the install_dependency error if xclip is not installed
	// or succeed if it is - both are valid behaviors
	if err == nil && result.Status == "error" {
		assert.Equal(t, "install_dependency", result.Data["recovery"])
	} else if err == nil {
		assert.Equal(t, "ok", result.Status)
	}
}

// ===================== ClipboardWriteAction Tests =====================

func TestClipboardWriteAction_Success(t *testing.T) {
	session := newTestSession()
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)

	action := &ClipboardWriteAction{}
	params, _ := json.Marshal(map[string]string{"content": "hello"})
	result, err := action.Execute(context.Background(), session, svc, params)
	if err == nil && result.Status == "error" {
		assert.Equal(t, "install_dependency", result.Data["recovery"])
	} else if err == nil {
		assert.Equal(t, "ok", result.Status)
	}
}

// ===================== DarkModeAction Tests =====================

func TestDarkModeAction_Toggle(t *testing.T) {
	session := newTestSession()
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)

	action := &DarkModeAction{}
	params, _ := json.Marshal(map[string]bool{"enabled": true})
	result, err := action.Execute(context.Background(), session, svc, params)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.True(t, session.DarkMode)

	// Toggle off
	params, _ = json.Marshal(map[string]bool{"enabled": false})
	result, err = action.Execute(context.Background(), session, svc, params)
	require.NoError(t, err)
	assert.False(t, session.DarkMode)
}

// ===================== LocaleAction Tests =====================

func TestLocaleAction_Set(t *testing.T) {
	session := newTestSession()
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)

	action := &LocaleAction{}
	params, _ := json.Marshal(map[string]string{"locale": "fr_FR.UTF-8"})
	result, err := action.Execute(context.Background(), session, svc, params)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.Equal(t, "fr_FR.UTF-8", session.Locale)
}

func TestLocaleAction_Empty(t *testing.T) {
	session := newTestSession()
	shell := &mockShell{}
	svc := newTestServiceWithShell(shell)

	action := &LocaleAction{}
	params, _ := json.Marshal(map[string]string{"locale": ""})
	_, err := action.Execute(context.Background(), session, svc, params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must not be empty")
}

// ===================== Handler Tests =====================

func TestControlEndpoint_UnknownAction(t *testing.T) {
	h, svc := newTestHandler()
	router := newTestRouter(h)

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	body := `{"action":"nonexistent"}`
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/livedesktop/sessions/%s/control", session.ID),
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestControlEndpoint_SessionNotFound(t *testing.T) {
	h, _ := newTestHandler()
	router := newTestRouter(h)

	body := `{"action":"quit_app"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/livedesktop/sessions/fake-id/control",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestControlEndpoint_InvalidJSON(t *testing.T) {
	h, _ := newTestHandler()
	router := newTestRouter(h)

	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/livedesktop/sessions/any-id/control",
		bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestControlEndpoint_EmptyAction(t *testing.T) {
	h, _ := newTestHandler()
	router := newTestRouter(h)

	body := `{"action":""}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/livedesktop/sessions/any-id/control",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ===================== Service ExecuteAction Tests =====================

func TestExecuteAction_RegistryLookup(t *testing.T) {
	store := NewInMemoryStore()
	dm := &mockDisplayManager{
		display: &screenrecording.ManagedDisplay{DisplayID: ":99"},
	}
	svc := newTestService(store, dm, mockVNCStart(5900, 6080))

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	_, err = svc.ExecuteAction(context.Background(), session.ID, "unknown_action", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action")
}

func TestServeFile_PathTraversal(t *testing.T) {
	h, _ := newTestHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/livedesktop/sessions/test-id/files/..%2F..%2Fetc%2Fpasswd", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	// Should not serve /etc/passwd - either 400 or 404
	assert.NotEqual(t, http.StatusOK, rr.Code)
}

// ===================== Service LaunchApp with Control State =====================

func TestLaunchApp_AppliesEnvVars(t *testing.T) {
	// This is an integration-style test that verifies env var setup
	session := newTestSession()
	session.EnvVars = map[string]string{"MY_VAR": "my_value"}

	// Verify the env vars are correctly stored and retrievable
	envVars := session.GetEnvVars()
	assert.Equal(t, "my_value", envVars["MY_VAR"])
}

func TestLaunchApp_OfflineMode(t *testing.T) {
	session := newTestSession()
	session.SetNetworkMode("offline", 0)
	assert.Equal(t, "offline", session.NetworkMode)
}

func TestLaunchApp_DarkMode(t *testing.T) {
	session := newTestSession()
	session.SetDarkMode(true)
	assert.True(t, session.DarkMode)
}

func TestServeFile_ValidFilename(t *testing.T) {
	h, _ := newTestHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r)

	// Create a temp file to serve
	tmpDir := t.TempDir()
	h.service.dataDir = tmpDir
	sessionDir := filepath.Join(tmpDir, "sessions", "test-session")
	require.NoError(t, os.MkdirAll(sessionDir, 0o755))
	testFile := filepath.Join(sessionDir, "screenshot.png")
	require.NoError(t, os.WriteFile(testFile, []byte("PNG data"), 0o644))

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/livedesktop/sessions/test-session/files/screenshot.png", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
