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
	"time"

	"scenario-to-desktop-api/procmetrics"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHandler() (*Handler, *Service) {
	store := NewInMemoryStore()
	backend := newMockBackend()
	svc := NewService(store, backend, newTestLogger(), "")
	handler := NewHandler(svc)
	return handler, svc
}

func newTestRouter(h *Handler) *mux.Router {
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return r
}

func TestStartSession_HTTP(t *testing.T) {
	h, _ := newTestHandler()
	router := newTestRouter(h)

	body := `{"width":1024,"height":768,"scenario_name":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/livedesktop/sessions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var resp SessionView
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.ID)
	assert.Equal(t, "test", resp.ScenarioName)
	assert.Equal(t, SessionState("running"), resp.State)
}

func TestListSessions_HTTP(t *testing.T) {
	h, svc := newTestHandler()
	router := newTestRouter(h)

	// Create a session first
	_, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "s1"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/livedesktop/sessions", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var sessions []SessionView
	err = json.NewDecoder(rr.Body).Decode(&sessions)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
}

func TestGetSession_NotFound(t *testing.T) {
	h, _ := newTestHandler()
	router := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/livedesktop/sessions/fake-id-123", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestLaunchElectronValidationOperationalResolverErrorsStayServerErrors(t *testing.T) {
	for _, resolverErr := range []string{
		"invalid renderer configuration from discovery",
		"required UI_PORT detector is unavailable",
		"ambiguous renderer configuration",
	} {
		t.Run(resolverErr, func(t *testing.T) {
			h, service := newTestHandler()
			session, err := service.StartSession(context.Background(), SessionConfig{ScenarioName: "demo"})
			require.NoError(t, err)
			rendererCalls, apiCalls := 0, 0
			service.WithRendererURLResolver(func(context.Context, string) (string, error) {
				rendererCalls++
				return "", fmt.Errorf("%s", resolverErr)
			})
			service.WithValidationAPIPortResolver(func(context.Context, string) (int, error) {
				apiCalls++
				return 19349, nil
			})

			body := `{"app_path":"/unused","context_id":"ctx","scenario_name":"demo","artifact_digest":"sha256:artifact","target_id":"target","journey_id":"journey","profile_id":"profile","isolation_lease_id":"lease","renderer_url_prefix":"http://127.0.0.1:0"}`
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/livedesktop/sessions/%s/launch-electron-validation", session.ID), bytes.NewBufferString(body))
			recorder := httptest.NewRecorder()
			newTestRouter(h).ServeHTTP(recorder, req)

			assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			var response map[string]string
			require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
			assert.Equal(t, "resolve Electron validation renderer: "+resolverErr, response["error"])
			assert.Equal(t, 1, rendererCalls)
			assert.Zero(t, apiCalls)
		})
	}
}

func TestLaunchElectronValidationMissingIdentityIsBadRequest(t *testing.T) {
	h, service := newTestHandler()
	session, err := service.StartSession(context.Background(), SessionConfig{ScenarioName: "demo"})
	require.NoError(t, err)
	rendererCalls := 0
	service.WithRendererURLResolver(func(context.Context, string) (string, error) {
		rendererCalls++
		return "http://127.0.0.1:41235", nil
	})

	body := `{"app_path":"/unused","scenario_name":"demo","artifact_digest":"sha256:artifact","target_id":"target","journey_id":"journey","profile_id":"profile","isolation_lease_id":"lease","renderer_id":"renderer"}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/livedesktop/sessions/%s/launch-electron-validation", session.ID), bytes.NewBufferString(body))
	recorder := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	var response map[string]string
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
	assert.Equal(t, "electron validation context_id is required", response["error"])
	assert.Zero(t, rendererCalls)
}

func TestStopSession_HTTP(t *testing.T) {
	h, svc := newTestHandler()
	router := newTestRouter(h)

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/livedesktop/sessions/%s", session.ID), nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	err = json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "stopped", resp["status"])
}

func TestHeartbeat_HTTP(t *testing.T) {
	h, svc := newTestHandler()
	router := newTestRouter(h)

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/livedesktop/sessions/%s/heartbeat", session.ID), nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	err = json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp["status"])
}

func TestLaunchApp_MissingPath(t *testing.T) {
	h, svc := newTestHandler()
	router := newTestRouter(h)

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/livedesktop/sessions/%s/launch", session.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Empty app_path triggers auto-discovery, which fails when vrooliRoot is not configured
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var resp map[string]string
	err = json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "auto-discover artifact")
}

func TestGetMetrics_WithMonitor(t *testing.T) {
	h, svc := newTestHandler()
	router := newTestRouter(h)

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	splashDur := int64(500)
	readyDur := int64(1200)
	now := time.Now()
	splashAt := now.Add(-700 * time.Millisecond)
	monitor := newMockMonitor(&procmetrics.Report{
		Startup: procmetrics.StartupTiming{
			LaunchAt:         now.Add(-1200 * time.Millisecond),
			SplashVisibleAt:  &splashAt,
			SplashDurationMs: &splashDur,
			ReadyAt:          &now,
			ReadyMs:          &readyDur,
		},
		Samples: []procmetrics.Sample{
			{Timestamp: now, CPUPercent: 15.0, RSSBytes: 100 * 1024 * 1024, PeakBytes: 150 * 1024 * 1024, Threads: 4},
		},
	})
	session.SetMonitor(monitor)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/livedesktop/sessions/%s/metrics", session.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var report procmetrics.Report
	err = json.NewDecoder(rr.Body).Decode(&report)
	require.NoError(t, err)
	require.NotNil(t, report.Startup.ReadyMs)
	assert.Equal(t, int64(1200), *report.Startup.ReadyMs)
	assert.Len(t, report.Samples, 1)
}

func TestGetMetrics_NoMonitor(t *testing.T) {
	h, svc := newTestHandler()
	router := newTestRouter(h)

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/livedesktop/sessions/%s/metrics", session.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var body map[string]string
	err = json.NewDecoder(rr.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "no_monitor", body["status"])
}

func TestGetMetrics_SessionNotFound(t *testing.T) {
	h, _ := newTestHandler()
	router := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/livedesktop/sessions/nonexistent/metrics", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestGetSession_IncludesMetricsInView(t *testing.T) {
	h, svc := newTestHandler()
	router := newTestRouter(h)

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	splashDur := int64(300)
	readyDur := int64(800)
	now := time.Now()
	splashAt := now.Add(-500 * time.Millisecond)
	monitor := newMockMonitor(&procmetrics.Report{
		Startup: procmetrics.StartupTiming{
			LaunchAt:         now.Add(-800 * time.Millisecond),
			SplashVisibleAt:  &splashAt,
			SplashDurationMs: &splashDur,
			ReadyAt:          &now,
			ReadyMs:          &readyDur,
		},
		Samples: []procmetrics.Sample{
			{Timestamp: now, CPUPercent: 10.0, RSSBytes: 50 * 1024 * 1024, PeakBytes: 80 * 1024 * 1024, Threads: 2},
		},
	})
	session.SetMonitor(monitor)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/livedesktop/sessions/%s", session.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var view SessionView
	err = json.NewDecoder(rr.Body).Decode(&view)
	require.NoError(t, err)
	require.NotNil(t, view.Metrics)
	assert.True(t, view.Metrics.SplashDetected)
	assert.True(t, view.Metrics.ReadyDetected)
	require.NotNil(t, view.Metrics.ReadyDurationMs)
	assert.Equal(t, int64(800), *view.Metrics.ReadyDurationMs)
}

func TestControlActionHTTPValidatesAndExecutesDesktopControls(t *testing.T) {
	h, svc := newTestHandler()
	router := newTestRouter(h)
	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)
	for _, check := range []struct {
		body string
		want int
	}{
		{"not json", http.StatusBadRequest},
		{`{}`, http.StatusBadRequest},
		{`{"action":"unknown"}`, http.StatusBadRequest},
		{`{"action":"clipboard_write","params":{"content":"copied"}}`, http.StatusOK},
		{`{"action":"clipboard_read","params":{}}`, http.StatusOK},
	} {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/livedesktop/sessions/%s/control", session.ID), bytes.NewBufferString(check.body))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != check.want {
			t.Fatalf("control %s status=%d want=%d body=%s", check.body, response.Code, check.want, response.Body.String())
		}
	}
}

func TestServeFileHTTPRejectsTraversalAndServesSessionFile(t *testing.T) {
	h, svc := newTestHandler()
	dataDir := t.TempDir()
	svc.WithDataDir(dataDir)
	sessionDir := filepath.Join(dataDir, "sessions", "session-1")
	require.NoError(t, os.MkdirAll(sessionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sessionDir, "capture.png"), []byte("image data"), 0o644))
	valid := httptest.NewRecorder()
	h.serveFile(valid, mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/", nil), map[string]string{"id": "session-1", "filename": "capture.png"}))
	if valid.Code != http.StatusOK || valid.Body.String() != "image data" {
		t.Fatalf("served file status=%d body=%q", valid.Code, valid.Body.String())
	}
	for _, filename := range []string{"../secret", `..\\secret`} {
		response := httptest.NewRecorder()
		h.serveFile(response, mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/", nil), map[string]string{"id": "session-1", "filename": filename}))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("traversal %q status=%d", filename, response.Code)
		}
	}
}
