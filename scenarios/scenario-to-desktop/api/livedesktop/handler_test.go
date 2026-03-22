package livedesktop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scenario-to-desktop-api/procmetrics"
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
