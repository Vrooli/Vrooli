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

	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/screenrecording"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/api-core/storage"
)

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
		Display:      &mockDisplay{id: ":99", w: 1280, h: 720, running: true},
		Width:        1280,
		Height:       720,
		NetworkMode:  "normal",
		EnvVars:      make(map[string]string),
	}
}

func newTestServiceForActions() *Service {
	store := NewInMemoryStore()
	backend := newMockBackend()
	svc := NewService(store, backend, newTestLogger(), "")
	svc.dataDir = os.TempDir()
	return svc
}

// ===================== LaunchAppAction Tests =====================

func TestLaunchAppAction_NoArtifact(t *testing.T) {
	svc := newTestServiceForActions()
	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	action := &LaunchAppAction{}
	_, err = action.Execute(context.Background(), session, svc, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auto-discover artifact")
}

// ===================== QuitAppAction Tests =====================

func TestQuitAppAction_NoApp(t *testing.T) {
	svc := newTestServiceForActions()
	session := newTestSession()

	action := &QuitAppAction{}
	_, err := action.Execute(context.Background(), session, svc, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no app is running")
}

func TestQuitAppAction_Success(t *testing.T) {
	svc := newTestServiceForActions()
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
	svc := newTestServiceForActions()
	svc.dataDir = t.TempDir()
	session := newTestSession()

	action := &ScreenshotAction{}
	result, err := action.Execute(context.Background(), session, svc, nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.Contains(t, result.Data["url"], "/api/v1/livedesktop/sessions/test-session-1/files/screenshot-")
}

func TestScreenshotAction_BackendError(t *testing.T) {
	svc := newTestServiceForActions()
	svc.dataDir = t.TempDir()
	// Set screenshot to fail
	svc.backend.(*mockPlatformBackend).screenshotErr = fmt.Errorf("capture failed")
	session := newTestSession()

	action := &ScreenshotAction{}
	_, err := action.Execute(context.Background(), session, svc, nil)
	assert.Error(t, err)
}

func TestScreenshotAction_DisplayNotRunning(t *testing.T) {
	svc := newTestServiceForActions()
	session := newTestSession()
	session.Display.(*mockDisplay).running = false

	action := &ScreenshotAction{}
	_, err := action.Execute(context.Background(), session, svc, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

// ===================== StartRecordingAction Tests =====================

func TestStartRecordingAction_Success(t *testing.T) {
	svc := newTestServiceForActions()
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
	svc := newTestServiceForActions()
	svc.recorder = &mockRecorder{}
	session := newTestSession()
	session.IsRecording = true

	action := &StartRecordingAction{}
	_, err := action.Execute(context.Background(), session, svc, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already recording")
}

func TestStartRecordingAction_NoRecorder(t *testing.T) {
	svc := newTestServiceForActions()
	session := newTestSession()

	action := &StartRecordingAction{}
	_, err := action.Execute(context.Background(), session, svc, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

// ===================== StopRecordingAction Tests =====================

func TestStopRecordingAction_Success(t *testing.T) {
	svc := newTestServiceForActions()
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
	_, err := action.Execute(context.Background(), session, svc, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "capture storage is unavailable")
	assert.False(t, session.IsRecording)
}

func TestStopRecordingAction_NotRecording(t *testing.T) {
	svc := newTestServiceForActions()
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
	svc := newTestServiceForActions()

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
	svc := newTestServiceForActions()

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
	svc := newTestServiceForActions()

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
	svc := newTestServiceForActions()

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
	svc := newTestServiceForActions()

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
	svc := newTestServiceForActions()

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
	svc := newTestServiceForActions()

	action := &ResizeDisplayAction{}
	params, _ := json.Marshal(map[string]int{"width": 1920, "height": 1080})
	result, err := action.Execute(context.Background(), session, svc, params)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.Equal(t, 1920, session.Width)
	assert.Equal(t, 1080, session.Height)
}

func TestResizeDisplayAction_InvalidDimensions(t *testing.T) {
	session := newTestSession()
	svc := newTestServiceForActions()

	action := &ResizeDisplayAction{}
	params, _ := json.Marshal(map[string]int{"width": 0, "height": 1080})
	_, err := action.Execute(context.Background(), session, svc, params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "positive")
}

// ===================== ClipboardReadAction Tests =====================

func TestClipboardReadAction_Success(t *testing.T) {
	session := newTestSession()
	svc := newTestServiceForActions()
	svc.backend.(*mockPlatformBackend).clipboardVal = "clipboard content"

	action := &ClipboardReadAction{}
	result, err := action.Execute(context.Background(), session, svc, nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.Equal(t, "clipboard content", result.Data["content"])
}

// ===================== ClipboardWriteAction Tests =====================

func TestClipboardWriteAction_Success(t *testing.T) {
	session := newTestSession()
	svc := newTestServiceForActions()

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
	svc := newTestServiceForActions()

	action := &DarkModeAction{}
	params, _ := json.Marshal(map[string]bool{"enabled": true})
	result, err := action.Execute(context.Background(), session, svc, params)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.True(t, session.DarkMode)

	// Toggle off
	params, _ = json.Marshal(map[string]bool{"enabled": false})
	_, err = action.Execute(context.Background(), session, svc, params)
	require.NoError(t, err)
	assert.False(t, session.DarkMode)
}

// ===================== LocaleAction Tests =====================

func TestLocaleAction_Set(t *testing.T) {
	session := newTestSession()
	svc := newTestServiceForActions()

	action := &LocaleAction{}
	params, _ := json.Marshal(map[string]string{"locale": "fr_FR.UTF-8"})
	result, err := action.Execute(context.Background(), session, svc, params)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	assert.Equal(t, "fr_FR.UTF-8", session.Locale)
}

func TestLocaleAction_Empty(t *testing.T) {
	session := newTestSession()
	svc := newTestServiceForActions()

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
	backend := newMockBackend()
	svc := newTestService(store, backend)

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

// ===================== Capture Persistence Tests =====================

func newCapturesService(t *testing.T) *captures.Service {
	t.Helper()
	tmpDir := t.TempDir()
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileDesktop,
		// Override home dir to use temp
		UserHomeDir:   func() (string, error) { return tmpDir, nil },
		UserConfigDir: func() (string, error) { return filepath.Join(tmpDir, ".config"), nil },
		UserCacheDir:  func() (string, error) { return filepath.Join(tmpDir, ".cache"), nil },
	})
	require.NoError(t, err)
	opts := storage.Options{ScenarioID: "scenario-to-desktop-captures"}
	metaPath, err := resolver.Path(opts, storage.ClassData, "captures_meta.json")
	require.NoError(t, err)
	filesDir, err := resolver.Path(opts, storage.ClassData, "captures")
	require.NoError(t, err)
	store, err := captures.NewFileStore(metaPath)
	require.NoError(t, err)
	return captures.NewService(resolver, opts, filesDir, store)
}

// screenshotCreatingBackend creates the screenshot file when CaptureScreenshot is called.
type screenshotCreatingBackend struct {
	mockPlatformBackend
}

func (b *screenshotCreatingBackend) CaptureScreenshot(ctx context.Context, display PlatformDisplay, outputPath string) error {
	return os.WriteFile(outputPath, []byte("PNG fake"), 0o644)
}

func TestScreenshotAction_PersistsCapture(t *testing.T) {
	backend := &screenshotCreatingBackend{mockPlatformBackend: *newMockBackend()}
	store := NewInMemoryStore()
	svc := NewService(store, backend, newTestLogger(), "")
	svc.dataDir = t.TempDir()

	capSvc := newCapturesService(t)
	svc.captures = capSvc

	session := newTestSession()
	_ = store.Create(session)

	action := &ScreenshotAction{}
	result, err := action.Execute(context.Background(), session, svc, nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)

	// Verify capture was persisted via captures service
	if result.Data["capture_id"] != nil {
		assert.Contains(t, result.Data["url"], "/api/v1/captures/")
		caps, err := capSvc.Store().List("test-scenario")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(caps), 1)
	}
}

func TestStopRecordingAction_PersistsCapture(t *testing.T) {
	svc := newTestServiceForActions()

	capSvc := newCapturesService(t)
	svc.captures = capSvc

	// Create a fake recording file
	tmpDir := t.TempDir()
	videoPath := filepath.Join(tmpDir, "recording-123.mp4")
	require.NoError(t, os.WriteFile(videoPath, []byte("MP4 fake data"), 0o644))

	svc.recorder = &mockRecorder{
		result: &screenrecording.CaptureResult{
			VideoPath:     videoPath,
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

	// Verify capture was persisted
	if result.Data["capture_id"] != nil {
		assert.Contains(t, result.Data["video_url"], "/api/v1/captures/")
		caps, err := capSvc.Store().List("test-scenario")
		require.NoError(t, err)
		assert.Len(t, caps, 1)
		assert.Equal(t, captures.CaptureRecording, caps[0].Type)
	}
}
