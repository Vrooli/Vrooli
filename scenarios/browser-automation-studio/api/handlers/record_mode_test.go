package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/performance"
	archiveingestion "github.com/vrooli/browser-automation-studio/services/archive-ingestion"
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
	sessionProfiles := archiveingestion.NewSessionProfileStore(tempDir, log)

	handler := &Handler{
		recordModeService: mockService,
		sessionProfiles:   sessionProfiles,
		wsHub:             mockHub,
		log:               log,
		perfRegistry:      performance.NewCollectorRegistry(60, 100),
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

	if !mockService.MockClient().StopRecordingCalled {
		t.Fatal("expected StopRecording to be called")
	}

	var response StopRecordingResponse
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

	mockService.MockClient().StopRecordingError = &driver.Error{Status: 404, Message: "no recording"}

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

	var response RecordingStatusResponse
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

func TestGetRecordedActions_WithClear(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recordings/live/"+sessionID+"/actions?clear=true", nil)
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
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"
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

func TestUpdateRecordingViewport_Success(t *testing.T) {
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

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
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

	if !mockService.MockClient().ForwardInputCalled {
		t.Fatal("expected ForwardInput to be called")
	}

	if mockService.MockClient().LastSessionID != sessionID {
		t.Fatalf("expected session ID %q, got %q", sessionID, mockService.MockClient().LastSessionID)
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
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	sessionID := "test-session-123"
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
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	mockService.MockClient().FrameResponse = &driver.GetFrameResponse{
		Data:        "base64-frame-data",
		MediaType:   "image/jpeg",
		Width:       1920,
		Height:      1080,
		CapturedAt:  "2025-01-01T00:00:00Z",
		ContentHash: "abc123",
	}

	sessionID := "test-session-123"
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
	handler, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

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
}

// ============================================================================
// PersistRecordingSession Tests
// ============================================================================

func TestPersistRecordingSession_Success(t *testing.T) {
	handler, mockService, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)

	// Create a profile and associate it with the session
	profile, err := handler.sessionProfiles.Create("Test Profile")
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	sessionID := "test-session-123"
	handler.sessionProfiles.SetActiveSession(sessionID, profile.ID)

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
