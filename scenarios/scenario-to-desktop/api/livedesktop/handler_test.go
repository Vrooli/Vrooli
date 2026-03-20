package livedesktop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scenario-to-desktop-api/screenrecording"
)

func newTestHandler() (*Handler, *Service) {
	store := NewInMemoryStore()
	dm := &mockDisplayManager{
		display: &screenrecording.ManagedDisplay{DisplayID: ":99"},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := NewService(store, dm, logger)
	svc.startVNC = mockVNCStart(5900, 6080)
	svc.stopVNC = mockVNCStop
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

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	err = json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "app_path is required")
}
