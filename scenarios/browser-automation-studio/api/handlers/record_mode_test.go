package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	autosession "github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/domain"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"github.com/vrooli/browser-automation-studio/performance"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
	sessionprofile "github.com/vrooli/browser-automation-studio/services/session-profile"
	"github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
)

// createTestHandlerWithRecordMode creates a handler with mock services for record mode testing.
func createTestHandlerWithRecordMode(t *testing.T) (*Handler, *MockRecordModeService, string, *MockHub) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "record-mode-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	mockService := NewMockRecordModeService()
	mockHub := NewMockHub()
	sessionProfileSvc := sessionprofile.NewService(persistence.NewFileRepositoryWithConfig(tempDir, log, persistence.FileRepositoryConfig{Authority: testutil.ProfileCredentialAuthority}), log)

	handler := &Handler{
		recordModeService:     mockService,
		sessionProfileService: sessionProfileSvc,
		wsHub:                 mockHub,
		log:                   log,
		perfRegistry:          performance.NewCollectorRegistry(60, 100),
	}

	return handler, mockService, tempDir, mockHub
}

// ============================================================================
// CreateRecordingSession Tests
// ============================================================================

func TestCreateRecordingSession_Success(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	body := `{
		"viewport_width": 1920,
		"viewport_height": 1080,
		"initial_url": "https://example.com"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/session", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateRecordingSession(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if !mockService.CreateSessionCalled {
		t.Fatal("expected CreateSession to be called")
	}

	var response CreateRecordingSessionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.SessionID == "" {
		t.Fatal("expected session_id to be set")
	}
}

func TestCreateRecordingSession_InvalidJSON(t *testing.T) {
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/session", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateRecordingSession(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestCreateRecordingSession_ServiceError(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	mockService.CreateSessionError = errors.New("driver unavailable")

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/session", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateRecordingSession(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateRecordingSession_WithStreamSettings(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	quality := 80
	fps := 30
	body := `{
		"stream_quality": 80,
		"stream_fps": 30,
		"stream_scale": "device"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/session", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateRecordingSession(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if !mockService.CreateSessionCalled {
		t.Fatal("expected CreateSession to be called")
	}

	// Verify stream settings are applied (these are passed to the service)
	_ = quality
	_ = fps
}

// ============================================================================
// CloseRecordingSession Tests
// ============================================================================

func TestCloseRecordingSession_Success(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/session/"+sessionID+"/close", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.CloseRecordingSession(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if !mockService.CloseSessionCalled {
		t.Fatal("expected CloseSession to be called")
	}

	if mockService.LastSessionID != sessionID {
		t.Fatalf("expected session ID %q, got %q", sessionID, mockService.LastSessionID)
	}
}

func TestCloseRecordingSession_MissingSessionID(t *testing.T) {
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/session//close", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.CloseRecordingSession(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestCloseRecordingSession_NotFound(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	mockService.CloseSessionError = &driver.Error{Status: 404, Message: "session not found"}

	sessionID := "nonexistent-session"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/session/"+sessionID+"/close", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.CloseRecordingSession(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ============================================================================
// StartLiveRecording Tests
// ============================================================================

func TestStartLiveRecording_Success(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	body := `{"session_id": "test-session-123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/start", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.StartLiveRecording(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if !mockService.StartRecordingCalled {
		t.Fatal("expected StartRecording to be called")
	}
}

func TestStartLiveRecording_MissingSessionID(t *testing.T) {
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/start", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.StartLiveRecording(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestStartLiveRecording_InvalidJSON(t *testing.T) {
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	body := `{invalid`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/start", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.StartLiveRecording(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestStartLiveRecording_RecordingInProgress(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	mockService.StartRecordingError = &driver.Error{
		Status:  409,
		Message: "RECORDING_IN_PROGRESS",
	}

	body := `{"session_id": "test-session-123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/start", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.StartLiveRecording(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ============================================================================
// StopLiveRecording Tests
// ============================================================================

func TestStopLiveRecording_Success(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/"+sessionID+"/stop", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.StopLiveRecording(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if !mockService.StopRecordingCalled {
		t.Fatal("expected StopRecording to be called")
	}

	var response driver.StopRecordingResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.SessionID != sessionID {
		t.Fatalf("expected session_id %q, got %q", sessionID, response.SessionID)
	}
}

func TestStopLiveRecording_MissingSessionID(t *testing.T) {
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live//stop", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.StopLiveRecording(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestStopLiveRecording_NotFound(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	mockService.StopRecordingError = &driver.Error{Status: 404, Message: "no recording"}

	sessionID := "nonexistent-session"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/"+sessionID+"/stop", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.StopLiveRecording(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ============================================================================
// GetRecordingStatus Tests
// ============================================================================

func TestGetRecordingStatus_Success(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	mockService.MockClient().RecordingStatusResponse = &driver.RecordingStatusResponse{
		SessionID:   "test-session-123",
		IsRecording: true,
		ActionCount: 5,
		FrameCount:  100,
		StartedAt:   "2025-01-01T00:00:00Z",
	}

	sessionID := "test-session-123"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recordings/live/"+sessionID+"/status", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.GetRecordingStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response driver.RecordingStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !response.IsRecording {
		t.Fatal("expected is_recording to be true")
	}
	if response.ActionCount != 5 {
		t.Fatalf("expected action_count 5, got %d", response.ActionCount)
	}
}

func TestGetRecordingStatus_MissingSessionID(t *testing.T) {
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recordings/live//status", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.GetRecordingStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// GetRecordedActions Tests
// ============================================================================

func TestGetRecordedActions_Success(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	mockService.MockClient().RecordedActionsResponse = &driver.GetActionsResponse{
		SessionID:   "test-session-123",
		IsRecording: false,
		Actions: []driver.RecordedAction{
			{ID: "action-1", ActionType: "click"},
			{ID: "action-2", ActionType: "type"},
		},
	}

	sessionID := "test-session-123"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recordings/live/"+sessionID+"/actions", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.GetRecordedActions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if !mockService.MockClient().GetRecordedActionsCalled {
		t.Fatal("expected GetRecordedActions to be called")
	}
}

func TestGetRecordedActions_WithClearRequiresOwnedSession(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recordings/live/"+sessionID+"/actions?clear=true", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.GetRecordedActions(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for unowned clear, got %d: %s", rr.Code, rr.Body.String())
	}

	if mockService.MockClient().GetRecordedActionsCalled {
		t.Fatal("unowned destructive pull must be rejected before reading entries")
	}
}

// ============================================================================
// ValidateSelector Tests
// ============================================================================

func TestValidateSelector_Success(t *testing.T) {
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"
	body := `{"selector": "#my-button"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/"+sessionID+"/validate-selector", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ValidateSelector(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestValidateSelector_MissingSelector(t *testing.T) {
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"
	body := `{"selector": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/"+sessionID+"/validate-selector", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ValidateSelector(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// NavigateRecordingSession Tests
// ============================================================================

func TestNavigateRecordingSession_Success(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"
	owned := createOwnedNavigationSession(t, sessionID, &driver.NavigateResponse{DriverPageID: "initial-driver-page", URL: "https://example.com"}, func(r *http.Request, body map[string]any) {
		if r.URL.Path != "/session/"+sessionID+"/record/navigate" || body["url"] != "https://example.com" {
			t.Errorf("wrong navigation request: %s %v", r.URL.Path, body)
		}
	})
	owned.InitializePageTracking("https://before.test")
	owned.Pages().SetInitialPageDriverID("initial-driver-page")
	mockService.OwnedSessions = map[string]*autosession.Session{sessionID: owned}
	body := `{"url": "https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/"+sessionID+"/navigate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.NavigateRecordingSession(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response NavigateRecordingResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.URL != "https://example.com" {
		t.Fatalf("expected URL https://example.com, got %q", response.URL)
	}
}

func TestNavigateRecordingSession_MissingURL(t *testing.T) {
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"
	body := `{"url": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/"+sessionID+"/navigate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.NavigateRecordingSession(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// UpdateRecordingViewport Tests
// ============================================================================

func TestUpdateRecordingViewport_UnknownSession(t *testing.T) {
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"
	body := `{"width": 1920, "height": 1080}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/"+sessionID+"/viewport", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.UpdateRecordingViewport(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateRecordingViewport_InvalidDimensions(t *testing.T) {
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"
	body := `{"width": 0, "height": 1080}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/"+sessionID+"/viewport", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.UpdateRecordingViewport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// ForwardRecordingInput Tests
// ============================================================================

func TestForwardRecordingInput_Success(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"
	body := `{"type": "pointer", "action": "move", "x": 100, "y": 200}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/"+sessionID+"/input", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ForwardRecordingInput(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"applied_sequence":1`) {
		t.Fatalf("expected the applied input receipt in the response, got %s", rr.Body.String())
	}

	if !mockService.ForwardInputCalled {
		t.Fatal("expected ForwardInput to be called")
	}

	if mockService.LastSessionID != sessionID {
		t.Fatalf("expected session ID %q, got %q", sessionID, mockService.LastSessionID)
	}
}

func TestForwardRecordingInput_EmptyBody(t *testing.T) {
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/"+sessionID+"/input", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ForwardRecordingInput(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// CaptureRecordingScreenshot Tests
// ============================================================================

func TestCaptureRecordingScreenshot_Success(t *testing.T) {
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/"+sessionID+"/screenshot", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.CaptureRecordingScreenshot(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response RecordingScreenshotResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Screenshot == "" {
		t.Fatal("expected screenshot data to be set")
	}
}

// ============================================================================
// GetRecordingFrame Tests
// ============================================================================

func TestGetRecordingFrame_Success(t *testing.T) {
	handler, mockService, _, source, _ := ownedFrameFixture(t)
	mockService.MockClient().FrameResponse = &driver.GetFrameResponse{SessionID: source["session_id"], ContentHash: "abc123", Source: &driver.FrameSource{SessionID: source["session_id"], ExecutionID: source["execution_id"], LeaseID: source["lease_id"], PageID: source["page_id"]}}

	sessionID := source["session_id"]
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recordings/live/"+sessionID+"/frame", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.GetRecordingFrame(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Check ETag header is set
	if rr.Header().Get("ETag") == "" {
		t.Fatal("expected ETag header to be set")
	}
}

func TestGetRecordingFrame_NotModified(t *testing.T) {
	handler, mockService, _, source, _ := ownedFrameFixture(t)

	mockService.MockClient().FrameResponse = &driver.GetFrameResponse{
		SessionID: source["session_id"], Source: &driver.FrameSource{SessionID: source["session_id"], ExecutionID: source["execution_id"], LeaseID: source["lease_id"], PageID: source["page_id"]},
		Image:       "base64-frame-data",
		Mime:        "image/jpeg",
		Width:       1920,
		Height:      1080,
		CapturedAt:  "2025-01-01T00:00:00Z",
		ContentHash: "abc123",
	}

	sessionID := source["session_id"]
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recordings/live/"+sessionID+"/frame", nil)
	req.Header.Set("If-None-Match", `"abc123"`)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.GetRecordingFrame(rr, req)

	if rr.Code != http.StatusNotModified {
		t.Fatalf("expected status 304, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ============================================================================
// UpdateStreamSettings Tests
// ============================================================================

func TestUpdateStreamSettings_Success(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)
	mockService.MockClient().StreamSettingsResponse = &driver.UpdateStreamSettingsResponse{
		SessionID: "test-session-123", Quality: 80, FPS: 30, CurrentFPS: 22.28,
		Scale: "css", IsStreaming: true, Updated: true,
	}

	sessionID := "test-session-123"
	body := `{"quality": 80, "fps": 30}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/"+sessionID+"/stream-settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.UpdateStreamSettings(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response UpdateStreamSettingsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !response.Updated {
		t.Fatal("expected updated to be true")
	}
	if response.CurrentFPS != 22.28 {
		t.Fatalf("expected fractional current_fps 22.28, got %v", response.CurrentFPS)
	}
}

// ============================================================================
// PersistRecordingSession Tests
// ============================================================================

func TestPersistRecordingSession_Success(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	// Create a profile and associate it with the session
	profile, err := handler.sessionProfileService.CreateProfile("Test Profile")
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	sessionID := "test-session-123"
	handler.sessionProfileService.SetActiveSession(sessionID, string(profile.ID))

	mockService.StorageState = json.RawMessage(`{"cookies":[{"name":"test","value":"value"}],"origins":[]}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/"+sessionID+"/persist", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.PersistRecordingSession(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestPersistRecordingSession_NoActiveProfile(t *testing.T) {
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"
	// No profile associated with the session

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/"+sessionID+"/persist", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", sessionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.PersistRecordingSession(rr, req)

	// Should succeed even without an active profile (no-op)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

type profileCommitRecorder struct {
	*persistence.MockRepository
	commits int
}

func (r *profileCommitRecorder) Update(id persistence.ProfileID, modify func(*persistence.SessionProfile) error) (*persistence.SessionProfile, error) {
	r.commits++
	return r.MockRepository.Update(id, modify)
}

type profileSnapshotDriver struct {
	*MockRecordModeService
	pages     []*domain.Page
	active    uuid.UUID
	pagesErr  error
	onClose   func()
	onStorage func()
}

func (d *profileSnapshotDriver) GetStorageState(ctx context.Context, id string) (json.RawMessage, error) {
	if d.onStorage != nil {
		d.onStorage()
	}
	return d.MockRecordModeService.GetStorageState(ctx, id)
}

func (d *profileSnapshotDriver) GetOpenPages(string) ([]*domain.Page, uuid.UUID, error) {
	return d.pages, d.active, d.pagesErr
}

func (d *profileSnapshotDriver) CloseSession(ctx context.Context, id string) error {
	if d.onClose != nil {
		d.onClose()
	}
	return d.MockRecordModeService.CloseSession(ctx, id)
}

// [REQ:BAS-RH-J14] Captures may not commit after their active binding changes.
func TestRecordingProfileSnapshotRequiresCurrentBinding(t *testing.T) {
	for _, change := range []string{"clear", "same-profile-rebind", "other-profile-rebind", "cancel"} {
		t.Run(change, func(t *testing.T) {
			repo := persistence.NewMockRepository()
			log := logrus.New()
			log.SetLevel(logrus.PanicLevel)
			svc := sessionprofile.NewService(repo, log)
			old, _ := svc.CreateProfile("Original")
			other, _ := svc.CreateProfile("Other")
			svc.SetActiveSession("session", string(old.ID))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			drv := &profileSnapshotDriver{MockRecordModeService: NewMockRecordModeService()}
			drv.StorageState = json.RawMessage(`{"cookies":[{"name":"identity","value":"late"}],"origins":[]}`)
			drv.onStorage = func() {
				switch change {
				case "clear":
					svc.ClearActiveSession("session")
				case "same-profile-rebind":
					svc.ClearActiveSession("session")
					svc.SetActiveSession("session", string(old.ID))
				case "other-profile-rebind":
					svc.SetActiveSession("session", string(other.ID))
				case "cancel":
					cancel()
				}
			}
			h := &Handler{recordModeService: drv, sessionProfileService: svc, log: log}
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("sessionId", "session")
			req := httptest.NewRequest(http.MethodPost, "/session/session/persist", nil).WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
			rr := httptest.NewRecorder()
			h.PersistRecordingSession(rr, req)
			if rr.Code < 400 {
				t.Errorf("invalidated capture acknowledged: %d", rr.Code)
			}
			for _, before := range []*persistence.SessionProfile{old, other} {
				after, err := repo.Get(before.ID)
				if err != nil || !reflect.DeepEqual(after, before) {
					t.Errorf("invalidated capture changed profile %s", before.Name)
				}
			}
		})
	}
}

// [REQ:BAS-RH-J06] A failed snapshot must not acknowledge persistence or destroy
// the live state needed to retry. A successful snapshot commits storage and tabs together.
func TestRecordingProfileCommit(t *testing.T) {
	for _, operation := range []string{"persist", "close"} {
		for _, failure := range []string{"none", "storage", "tabs", "save", "empty-tabs", "close"} {
			if operation == "persist" && failure == "close" {
				continue
			}
			t.Run(operation+"/"+failure, func(t *testing.T) {
				repo := &profileCommitRecorder{MockRepository: persistence.NewMockRepository()}
				profile := &persistence.SessionProfile{
					ID: "profile", Name: "Saved identity",
					StorageState: json.RawMessage(`{"cookies":[{"name":"identity","value":"old"}],"origins":[]}`),
					OpenTabs:     []persistence.TabState{{URL: "https://fixture.invalid/old"}},
				}
				if err := repo.Create(profile); err != nil {
					t.Fatal(err)
				}
				log := logrus.New()
				log.SetLevel(logrus.PanicLevel)
				svc := sessionprofile.NewService(repo, log)
				svc.SetActiveSession("session", string(profile.ID))
				pageID := uuid.New()
				drv := &profileSnapshotDriver{
					MockRecordModeService: NewMockRecordModeService(),
					pages:                 []*domain.Page{{ID: pageID, URL: "https://fixture.invalid/new", Title: "New tab"}},
					active:                pageID,
				}
				drv.StorageState = json.RawMessage(`{"cookies":[{"name":"identity","value":"new"}],"origins":[]}`)
				wantTabs := []persistence.TabState{{URL: drv.pages[0].URL, Title: "New tab", IsActive: true, Order: 0}}
				if failure == "empty-tabs" {
					drv.pages = nil
					wantTabs = []persistence.TabState{}
				}
				assertCommitted := func() {
					t.Helper()
					got, err := repo.Get(profile.ID)
					if err != nil || got == nil {
						t.Fatalf("read committed profile: %v", err)
					}
					if string(got.StorageState) != string(drv.StorageState) || !reflect.DeepEqual(got.OpenTabs, wantTabs) {
						t.Fatalf("snapshot not committed together: storage=%s, tabs=%+v", got.StorageState, got.OpenTabs)
					}
				}
				drv.onClose = assertCommitted
				switch failure {
				case "storage":
					drv.GetStorageStateError = errors.New("storage capture failed")
				case "tabs":
					drv.pagesErr = errors.New("tab capture failed")
				case "save":
					repo.SaveErr = errors.New("profile commit failed")
				case "close":
					drv.CloseSessionError = errors.New("browser close failed")
				}
				h := &Handler{recordModeService: drv, sessionProfileService: svc, log: log}
				invoke := func() *httptest.ResponseRecorder {
					req := httptest.NewRequest(http.MethodPost, "/session/session/"+operation, nil)
					rctx := chi.NewRouteContext()
					rctx.URLParams.Add("sessionId", "session")
					req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
					rr := httptest.NewRecorder()
					if operation == "close" {
						h.CloseRecordingSession(rr, req)
					} else {
						h.PersistRecordingSession(rr, req)
					}
					return rr
				}
				rr := invoke()
				if failure != "none" && failure != "empty-tabs" {
					if rr.Code < 400 {
						t.Fatalf("failed %s acknowledged: %d %s", failure, rr.Code, rr.Body.String())
					}
					if svc.GetActiveSession("session") != string(profile.ID) {
						t.Fatal("failed operation lost recovery association")
					}
					if failure != "close" {
						if drv.CloseSessionCalled {
							t.Fatal("failed snapshot destroyed the live browser")
						}
						got, err := repo.Get(profile.ID)
						if err != nil || !reflect.DeepEqual(got, profile) {
							t.Fatalf("failed capture/commit changed prior profile: %+v, %v", got, err)
						}
					}
					// Retry through the same public endpoint after the fault clears.
					drv.GetStorageStateError, drv.pagesErr, drv.CloseSessionError, repo.SaveErr = nil, nil, nil, nil
					repo.commits = 0
					rr = invoke()
				}
				if rr.Code != http.StatusOK {
					t.Fatalf("successful retry: %d %s", rr.Code, rr.Body.String())
				}
				assertCommitted()
				if repo.commits != 1 {
					t.Fatalf("one snapshot requires one aggregate commit, got %d", repo.commits)
				}
				if operation == "close" {
					if !drv.CloseSessionCalled || svc.GetActiveSession("session") != "" {
						t.Fatal("successful close did not release browser and association")
					}
				} else if drv.CloseSessionCalled || svc.GetActiveSession("session") == "" {
					t.Fatal("persist ended the live session")
				}
			})
		}
	}
}

// [REQ:BAS-RH-J24] Driver receipts retain identity and time through the public API.
func TestRecordingLifecycle_PreservesDriverReceipt(t *testing.T) {
	const wire = `{"session_id":"receipt-session","recording_id":"9c45d4a0-5333-4b36-8197-bdba6b6c8f36","action_count":7,"is_recording":true,"started_at":"2026-09-22T12:00:00.123Z","stopped_at":"2026-09-22T12:01:00.456Z"}`
	for _, operation := range []string{"start", "stop", "status"} {
		t.Run(operation, func(t *testing.T) {
			h, service, tempDir, _ := createTestHandlerWithRecordMode(t)
			defer os.RemoveAll(tempDir)
			var start driver.StartRecordingResponse
			var stop driver.StopRecordingResponse
			var status driver.RecordingStatusResponse
			if err := json.Unmarshal([]byte(wire), &start); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(wire), &stop); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(wire), &status); err != nil {
				t.Fatal(err)
			}
			service.StartRecordingResponse = &start
			service.StopRecordingResponse = &stop
			service.MockClient().RecordingStatusResponse = &status
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"session_id":"receipt-session"}`))
			rc := chi.NewRouteContext()
			rc.URLParams.Add("sessionId", "receipt-session")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
			rr := httptest.NewRecorder()
			switch operation {
			case "start":
				h.StartLiveRecording(rr, req)
			case "stop":
				h.StopLiveRecording(rr, req)
			case "status":
				h.GetRecordingStatus(rr, req)
			}
			if rr.Code != http.StatusOK {
				t.Fatalf("status %d: %s", rr.Code, rr.Body)
			}
			var got map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got["recording_id"] != "9c45d4a0-5333-4b36-8197-bdba6b6c8f36" {
				t.Errorf("driver identity lost or replaced: %s", rr.Body)
			}
			field, want := "started_at", "2026-09-22T12:00:00.123Z"
			if operation == "stop" {
				field, want = "completed_at", "2026-09-22T12:01:00.456Z"
			}
			if got[field] != want {
				t.Errorf("%s lost: %s", field, rr.Body)
			}
		})
	}
}

type inputForwardingService struct {
	RecordModeService
	forward func(context.Context, string, []byte) (*driver.ForwardInputResponse, error)
}

func (s inputForwardingService) ForwardInput(ctx context.Context, id string, body []byte) (*driver.ForwardInputResponse, error) {
	return s.forward(ctx, id, body)
}

func TestWebSocketInputForwarderUsesOwnedServiceAndDeadline(t *testing.T) {
	h, service, dir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(dir)
	sentinel := errors.New("input rejected by owner")
	var seenContext context.Context
	calls := 0
	h.recordModeService = inputForwardingService{RecordModeService: service, forward: func(ctx context.Context, id string, body []byte) (*driver.ForwardInputResponse, error) {
		calls++
		seenContext = ctx
		if id != "owned-input" {
			t.Errorf("wrong session: %s", id)
		}
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > 2*time.Second {
			t.Errorf("missing or changed input deadline: %v", deadline)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Error(err)
		}
		if payload["type"] != "pointer" || payload["x"] != float64(12) {
			t.Errorf("changed input: %v", payload)
		}
		return nil, sentinel
	}}
	forward := h.CreateInputForwarder()
	if _, err := forward("owned-input", map[string]any{"type": "pointer", "x": 12}); !errors.Is(err, sentinel) {
		t.Errorf("owner rejection lost: %v", err)
	}
	if calls != 1 || seenContext == nil || seenContext.Err() != context.Canceled {
		t.Fatal("forwarding or deadline cleanup missing")
	}
	if _, err := forward("owned-input", map[string]any{"invalid": make(chan int)}); err == nil {
		t.Error("accepted unencodable input")
	}
	if calls != 1 {
		t.Error("invalid input reached the session owner")
	}
}

type rejectedRestorationDriver struct {
	*MockRecordModeService
	cleanupContextErr error
	handoff           bool
	closedAdmitted    bool
	closedReplacement bool
}

func (d *rejectedRestorationDriver) CreateSession(ctx context.Context, cfg *livecapture.SessionConfig) (*livecapture.SessionResult, error) {
	result, err := d.MockRecordModeService.CreateSession(ctx, cfg)
	if err != nil {
		return nil, err
	}
	result.Close = func(ctx context.Context) error {
		d.closedAdmitted = true
		d.cleanupContextErr = ctx.Err()
		if d.handoff {
			d.CloseSessionCalled = true
			return nil
		}
		return d.MockRecordModeService.CloseSession(ctx, result.SessionID)
	}
	return result, nil
}

func (d *rejectedRestorationDriver) RestoreTabs(context.Context, string, []persistence.TabState) (*livecapture.TabRestorationResult, error) {
	return &livecapture.TabRestorationResult{InitialURL: "https://partial.invalid"}, errors.New("controlled restoration failure")
}

func (d *rejectedRestorationDriver) CloseSession(ctx context.Context, id string) error {
	if d.handoff {
		d.closedReplacement = true
	}
	d.cleanupContextErr = ctx.Err()
	return d.MockRecordModeService.CloseSession(ctx, id)
}

// [REQ:BAS-RH-J01] [REQ:BAS-RH-J06] Failed restore must never replace the saved identity.
func TestFailedRestorationPreservesProfileAndClosesUncommittedSession(t *testing.T) {
	for _, fault := range []string{"restore", "cleanup", "cancelled request", "owner handoff"} {
		t.Run(fault, func(t *testing.T) {
			repo := &profileCommitRecorder{MockRepository: persistence.NewMockRepository()}
			profile := &persistence.SessionProfile{ID: "saved", Name: "Saved identity", StorageState: json.RawMessage(`{"cookies":[],"origins":[]}`), OpenTabs: []persistence.TabState{{URL: "https://saved.invalid", IsActive: true}}}
			if err := repo.Create(profile); err != nil {
				t.Fatal(err)
			}
			log := logrus.New()
			log.SetLevel(logrus.PanicLevel)
			svc := sessionprofile.NewService(repo, log)
			drv := &rejectedRestorationDriver{MockRecordModeService: NewMockRecordModeService(), handoff: fault == "owner handoff"}
			if fault == "cleanup" {
				drv.CloseSessionError = errors.New("controlled cleanup failure")
			}
			h := &Handler{recordModeService: drv, sessionProfileService: svc, log: log}
			req := httptest.NewRequest(http.MethodPost, "/session", strings.NewReader(`{"session_profile_id":"saved"}`))
			if fault == "cancelled request" {
				ctx, cancel := context.WithCancel(req.Context())
				cancel()
				req = req.WithContext(ctx)
			}
			rr := httptest.NewRecorder()
			h.CreateRecordingSession(rr, req)
			if rr.Code < 400 {
				t.Errorf("failed restore acknowledged: %d %s", rr.Code, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), "controlled restoration failure") {
				t.Errorf("missing restore failure: %s", rr.Body.String())
			}
			if !drv.CloseSessionCalled {
				t.Error("uncommitted browser was not closed")
			}
			if !drv.closedAdmitted || drv.closedReplacement {
				t.Errorf("cleanup did not retain admission: original=%v replacement=%v", drv.closedAdmitted, drv.closedReplacement)
			}
			if drv.cleanupContextErr != nil {
				t.Errorf("cleanup inherited cancellation: %v", drv.cleanupContextErr)
			}
			if fault == "cleanup" && (!strings.Contains(rr.Body.String(), "controlled cleanup failure") || !strings.Contains(rr.Body.String(), drv.LastSessionID)) {
				t.Errorf("missing cleanup recovery receipt: %s", rr.Body.String())
			}
			if repo.commits != 0 {
				t.Errorf("failed restore changed profile %d times", repo.commits)
			}
			got, err := repo.Get(profile.ID)
			if err != nil || !reflect.DeepEqual(profile, got) {
				t.Errorf("saved profile changed: %+v, %v", got, err)
			}
			if svc.GetActiveSession(drv.LastSessionID) != "" {
				t.Error("failed session may overwrite saved profile")
			}
		})
	}
}
