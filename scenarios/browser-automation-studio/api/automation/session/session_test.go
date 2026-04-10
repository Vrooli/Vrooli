package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
)

func TestCloseWithArtifacts(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/session/sess-123/close", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":     true,
			"video_paths": []string{"/tmp/video-1.webm", "/tmp/video-2.webm"},
		})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:     "sess-123",
		mode:   ModeExecution,
		client: client,
	}

	resp, err := sess.CloseWithArtifacts(context.Background())
	if err != nil {
		t.Fatalf("close: %v", err)
	}

	if resp == nil || len(resp.VideoPaths) != 2 {
		t.Fatalf("expected 2 video paths, got %#v", resp)
	}
}

// =============================================================================
// Mode Guard Tests
// =============================================================================

func TestSession_Run_RejectsRecordingMode(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:   "test-session",
		mode: ModeRecording,
	}

	_, err := sess.Run(context.Background(), contracts.CompiledInstruction{})
	if err == nil {
		t.Error("expected error when running in recording mode")
	}

	if !strings.Contains(err.Error(), "recording-only mode") {
		t.Errorf("expected error message about recording-only mode, got: %v", err)
	}
}

func TestSession_ForwardInput_RejectsExecutionMode(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:   "test-session",
		mode: ModeExecution,
	}

	err := sess.ForwardInput(context.Background(), []byte("{}"))
	if err == nil {
		t.Error("expected error when forwarding input in execution mode")
	}

	if !strings.Contains(err.Error(), "execution-only mode") {
		t.Errorf("expected error message about execution-only mode, got: %v", err)
	}
}

func TestSession_Navigate_RejectsClosedSession(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:     "test-session",
		mode:   ModeRecording,
		closed: true,
	}

	_, err := sess.Navigate(context.Background(), "https://example.com")
	if err == nil {
		t.Error("expected error when navigating in closed session")
	}

	if !strings.Contains(err.Error(), "session closed") {
		t.Errorf("expected error message about closed session, got: %v", err)
	}
}

func TestSession_ReportAction_InvokesCallback(t *testing.T) {
	t.Parallel()

	var receivedSessionID string
	var receivedAction *RecordedActionInfo

	sess := &Session{
		id:   "test-session",
		mode: ModeRecording,
		recording: &RecordingCallbacks{
			OnAction: func(sessionID string, action *RecordedActionInfo) {
				receivedSessionID = sessionID
				receivedAction = action
			},
		},
	}

	action := &RecordedActionInfo{
		ID:         uuid.New().String(),
		ActionType: "click",
		URL:        "https://example.com",
		Selector:   "#button",
		Confidence: 0.95,
	}

	sess.ReportAction(action)

	if receivedSessionID != "test-session" {
		t.Errorf("expected session ID 'test-session', got '%s'", receivedSessionID)
	}

	if receivedAction == nil {
		t.Fatal("expected action to be received")
	}

	if receivedAction.ActionType != "click" {
		t.Errorf("expected action type 'click', got '%s'", receivedAction.ActionType)
	}

	if receivedAction.Selector != "#button" {
		t.Errorf("expected selector '#button', got '%s'", receivedAction.Selector)
	}
}

func TestSession_ReportAction_NilRecording(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:        "test-session",
		mode:      ModeExecution,
		recording: nil, // No recording callbacks configured
	}

	// Should not panic when recording is nil
	action := &RecordedActionInfo{
		ID:         uuid.New().String(),
		ActionType: "click",
	}

	sess.ReportAction(action) // Should be a no-op, not panic
}

func TestSession_ReportPageEvent_InvokesCallback(t *testing.T) {
	t.Parallel()

	var receivedSessionID string
	var receivedEvent *PageEventInfo

	sess := &Session{
		id:   "test-session",
		mode: ModeRecording,
		recording: &RecordingCallbacks{
			OnPageEvent: func(sessionID string, event *PageEventInfo) {
				receivedSessionID = sessionID
				receivedEvent = event
			},
		},
	}

	pageID := uuid.New()
	event := &PageEventInfo{
		Type:   "page_created",
		PageID: pageID,
		URL:    "https://example.com/new-tab",
		Title:  "New Tab",
	}

	sess.ReportPageEvent(event)

	if receivedSessionID != "test-session" {
		t.Errorf("expected session ID 'test-session', got '%s'", receivedSessionID)
	}

	if receivedEvent == nil {
		t.Fatal("expected event to be received")
	}

	if receivedEvent.Type != "page_created" {
		t.Errorf("expected event type 'page_created', got '%s'", receivedEvent.Type)
	}

	if receivedEvent.URL != "https://example.com/new-tab" {
		t.Errorf("expected URL 'https://example.com/new-tab', got '%s'", receivedEvent.URL)
	}
}

func TestSession_ReportPageEvent_NilCallback(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:   "test-session",
		mode: ModeRecording,
		recording: &RecordingCallbacks{
			OnAction:    nil,
			OnPageEvent: nil, // No page event callback
		},
	}

	// Should not panic when OnPageEvent is nil
	event := &PageEventInfo{
		Type:   "page_created",
		PageID: uuid.New(),
		URL:    "https://example.com",
	}

	sess.ReportPageEvent(event) // Should be a no-op, not panic
}

func TestSession_HybridMode_AllowsRunAndForwardInput(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	handler.HandleFunc("/session/hybrid-sess/run", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
		})
	})
	handler.HandleFunc("/session/hybrid-sess/record/input", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:     "hybrid-sess",
		mode:   ModeHybrid,
		client: client,
	}

	// Hybrid mode should allow Run (doesn't reject like recording mode)
	_, runErr := sess.Run(context.Background(), contracts.CompiledInstruction{})
	// Run will fail because we didn't set up a proper instruction, but it shouldn't
	// fail due to mode restriction
	if runErr != nil && strings.Contains(runErr.Error(), "recording-only mode") {
		t.Error("hybrid mode should not reject Run operations")
	}

	// Hybrid mode should allow ForwardInput (doesn't reject like execution mode)
	fwdErr := sess.ForwardInput(context.Background(), []byte("{}"))
	if fwdErr != nil && strings.Contains(fwdErr.Error(), "execution-only mode") {
		t.Error("hybrid mode should not reject ForwardInput operations")
	}
}

func TestSession_Accessors(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:   "test-session-123",
		mode: ModeRecording,
		actualViewport: &driver.ActualViewport{
			Width:  1920,
			Height: 1080,
			Source: driver.ViewportSourceRequested,
		},
	}

	if sess.ID() != "test-session-123" {
		t.Errorf("expected ID 'test-session-123', got '%s'", sess.ID())
	}

	if sess.Mode() != ModeRecording {
		t.Errorf("expected mode Recording, got %v", sess.Mode())
	}

	if sess.ActualViewport() == nil {
		t.Fatal("expected non-nil ActualViewport")
	}

	if sess.ActualViewport().Width != 1920 {
		t.Errorf("expected viewport width 1920, got %d", sess.ActualViewport().Width)
	}

	// Pages should be nil if not initialized
	if sess.Pages() != nil {
		t.Error("expected nil Pages before initialization")
	}

	// Initialize page tracking
	sess.InitializePageTracking("https://example.com")

	if sess.Pages() == nil {
		t.Error("expected non-nil Pages after initialization")
	}
}

// =============================================================================
// Recording Lifecycle Tests
// =============================================================================

func TestSession_StartRecording_Success(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	handler.HandleFunc("/session/rec-session/record/start", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
		})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:     "rec-session",
		mode:   ModeRecording,
		client: client,
	}

	err = sess.StartRecording(context.Background(), RecordingConfig{
		ActionCallbackURL: "http://localhost:8080/callback",
		FrameCallbackURL:  "http://localhost:8080/frame",
		Quality:           80,
		FPS:               10,
	})

	if err != nil {
		t.Fatalf("StartRecording failed: %v", err)
	}
}

func TestSession_StartRecording_RejectsExecutionMode(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:   "exec-session",
		mode: ModeExecution,
	}

	err := sess.StartRecording(context.Background(), RecordingConfig{})

	if err == nil {
		t.Error("expected error when starting recording in execution mode")
	}

	if !strings.Contains(err.Error(), "execution-only mode") {
		t.Errorf("expected error about execution-only mode, got: %v", err)
	}
}

func TestSession_StartRecording_RejectsClosedSession(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:     "closed-session",
		mode:   ModeRecording,
		closed: true,
	}

	err := sess.StartRecording(context.Background(), RecordingConfig{})

	if err == nil {
		t.Error("expected error when starting recording on closed session")
	}

	if !strings.Contains(err.Error(), "session closed") {
		t.Errorf("expected error about session closed, got: %v", err)
	}
}

func TestSession_StopRecording_Success(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	handler.HandleFunc("/session/rec-session/record/stop", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
		})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:     "rec-session",
		mode:   ModeRecording,
		client: client,
	}

	err = sess.StopRecording(context.Background())
	if err != nil {
		t.Fatalf("StopRecording failed: %v", err)
	}
}

func TestSession_StopRecording_RejectsClosedSession(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:     "closed-session",
		mode:   ModeRecording,
		closed: true,
	}

	err := sess.StopRecording(context.Background())

	if err == nil {
		t.Error("expected error when stopping recording on closed session")
	}
}

func TestSession_GetRecordingStatus_Success(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	handler.HandleFunc("/session/rec-session/record/status", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"is_recording":  true,
			"action_count":  5,
			"duration_ms":   1234,
		})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:     "rec-session",
		mode:   ModeRecording,
		client: client,
	}

	status, err := sess.GetRecordingStatus(context.Background())
	if err != nil {
		t.Fatalf("GetRecordingStatus failed: %v", err)
	}

	if status == nil {
		t.Fatal("expected non-nil status")
	}

	if !status.IsRecording {
		t.Error("expected IsRecording to be true")
	}

	if status.ActionCount != 5 {
		t.Errorf("expected ActionCount 5, got %d", status.ActionCount)
	}
}

func TestSession_GetRecordingStatus_RejectsClosedSession(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:     "closed-session",
		mode:   ModeRecording,
		closed: true,
	}

	_, err := sess.GetRecordingStatus(context.Background())

	if err == nil {
		t.Error("expected error when getting status of closed session")
	}
}

func TestSession_GetRecordedActions_Success(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	handler.HandleFunc("/session/rec-session/record/actions", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"actions": []map[string]any{
				{"id": "action-1", "action_type": "click"},
				{"id": "action-2", "action_type": "type"},
			},
		})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:     "rec-session",
		mode:   ModeRecording,
		client: client,
	}

	actions, err := sess.GetRecordedActions(context.Background(), false)
	if err != nil {
		t.Fatalf("GetRecordedActions failed: %v", err)
	}

	if len(actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(actions))
	}
}

// =============================================================================
// Screenshot and Storage State Tests
// =============================================================================

func TestSession_CaptureScreenshot_Success(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	handler.HandleFunc("/session/ss-session/record/screenshot", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":       "base64encodeddata",
			"media_type": "image/jpeg",
			"width":      1920,
			"height":     1080,
		})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:     "ss-session",
		mode:   ModeRecording,
		client: client,
	}

	screenshot, err := sess.CaptureScreenshot(context.Background())
	if err != nil {
		t.Fatalf("CaptureScreenshot failed: %v", err)
	}

	if screenshot == nil {
		t.Fatal("expected non-nil screenshot")
	}

	if screenshot.Data != "base64encodeddata" {
		t.Errorf("expected data 'base64encodeddata', got '%s'", screenshot.Data)
	}

	if screenshot.MediaType != "image/jpeg" {
		t.Errorf("expected media_type 'image/jpeg', got '%s'", screenshot.MediaType)
	}

	if screenshot.Width != 1920 {
		t.Errorf("expected width 1920, got %d", screenshot.Width)
	}

	if screenshot.Height != 1080 {
		t.Errorf("expected height 1080, got %d", screenshot.Height)
	}
}

func TestSession_CaptureScreenshot_RejectsClosedSession(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:     "closed-session",
		mode:   ModeRecording,
		closed: true,
	}

	_, err := sess.CaptureScreenshot(context.Background())

	if err == nil {
		t.Error("expected error when capturing screenshot of closed session")
	}
}

func TestSession_GetStorageState_Success(t *testing.T) {
	t.Parallel()

	storageState := map[string]any{
		"cookies": []map[string]any{
			{"name": "session", "value": "abc123"},
		},
		"origins": []map[string]any{},
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/session/storage-session/storage-state", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		// The driver returns storage_state wrapped in a JSON object
		_ = json.NewEncoder(w).Encode(map[string]any{
			"storage_state": storageState,
		})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:     "storage-session",
		mode:   ModeRecording,
		client: client,
	}

	state, err := sess.GetStorageState(context.Background())
	if err != nil {
		t.Fatalf("GetStorageState failed: %v", err)
	}

	if state == nil {
		t.Fatal("expected non-nil storage state")
	}

	// Verify we can unmarshal the JSON
	var parsed map[string]any
	if err := json.Unmarshal(state, &parsed); err != nil {
		t.Fatalf("failed to parse storage state: %v", err)
	}

	cookies, ok := parsed["cookies"].([]any)
	if !ok || len(cookies) != 1 {
		t.Errorf("expected 1 cookie in storage state")
	}
}

func TestSession_GetStorageState_RejectsClosedSession(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:     "closed-session",
		mode:   ModeRecording,
		closed: true,
	}

	_, err := sess.GetStorageState(context.Background())

	if err == nil {
		t.Error("expected error when getting storage state of closed session")
	}
}

// =============================================================================
// DownloadArtifact Tests
// =============================================================================

func TestSession_DownloadArtifact_Success(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	handler.HandleFunc("/artifacts", func(w http.ResponseWriter, r *http.Request) {
		// Verify the path parameter is passed correctly
		path := r.URL.Query().Get("path")
		if path != "/path/to/video.webm" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// Simulate artifact download
		w.Header().Set("Content-Type", "video/webm")
		w.Header().Set("Content-Length", "12")
		_, _ = w.Write([]byte("video-data!!"))
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:     "artifact-session",
		mode:   ModeExecution,
		client: client,
		closed: true, // Intentionally closed - download should still work
	}

	artifact, err := sess.DownloadArtifact(context.Background(), "/path/to/video.webm")
	if err != nil {
		t.Fatalf("DownloadArtifact failed: %v", err)
	}

	if artifact == nil {
		t.Fatal("expected non-nil artifact")
	}
}

func TestSession_DownloadArtifact_EmptyPath(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:     "artifact-session",
		mode:   ModeExecution,
		client: client,
	}

	_, err = sess.DownloadArtifact(context.Background(), "")

	if err == nil {
		t.Error("expected error when path is empty")
	}

	if !strings.Contains(err.Error(), "path required") {
		t.Errorf("expected error about path required, got: %v", err)
	}
}

func TestSession_DownloadArtifact_NilSession(t *testing.T) {
	t.Parallel()

	var sess *Session = nil

	_, err := sess.DownloadArtifact(context.Background(), "/path/to/video.webm")

	if err == nil {
		t.Error("expected error when session is nil")
	}
}

func TestSession_DownloadArtifact_NilClient(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:     "no-client-session",
		mode:   ModeExecution,
		client: nil,
	}

	_, err := sess.DownloadArtifact(context.Background(), "/path/to/video.webm")

	if err == nil {
		t.Error("expected error when client is nil")
	}

	if !strings.Contains(err.Error(), "client unavailable") {
		t.Errorf("expected error about client unavailable, got: %v", err)
	}
}

// =============================================================================
// Other Session Operations Tests
// =============================================================================

func TestSession_UpdateViewport_Success(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	handler.HandleFunc("/session/vp-session/record/viewport", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:     "vp-session",
		mode:   ModeRecording,
		client: client,
	}

	err = sess.UpdateViewport(context.Background(), 1920, 1080)
	if err != nil {
		t.Fatalf("UpdateViewport failed: %v", err)
	}
}

func TestSession_UpdateViewport_RejectsClosedSession(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:     "closed-session",
		mode:   ModeRecording,
		closed: true,
	}

	err := sess.UpdateViewport(context.Background(), 1920, 1080)

	if err == nil {
		t.Error("expected error when updating viewport of closed session")
	}
}

func TestSession_ValidateSelector_Success(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	handler.HandleFunc("/session/sel-session/record/validate-selector", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"valid":        true,
			"match_count":  3,
		})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:     "sel-session",
		mode:   ModeRecording,
		client: client,
	}

	result, err := sess.ValidateSelector(context.Background(), "#my-button")
	if err != nil {
		t.Fatalf("ValidateSelector failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if !result.Valid {
		t.Error("expected selector to be valid")
	}

	if result.MatchCount != 3 {
		t.Errorf("expected match count 3, got %d", result.MatchCount)
	}
}

func TestSession_Reset_Success(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	handler.HandleFunc("/session/reset-session/reset", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:     "reset-session",
		mode:   ModeRecording,
		client: client,
	}

	err = sess.Reset(context.Background())
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
}

func TestSession_Reset_RejectsClosedSession(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:     "closed-session",
		mode:   ModeRecording,
		closed: true,
	}

	err := sess.Reset(context.Background())

	if err == nil {
		t.Error("expected error when resetting closed session")
	}
}

func TestSession_SetActivePage_Success(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	handler.HandleFunc("/session/page-session/record/active-page", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:     "page-session",
		mode:   ModeExecution,
		client: client,
	}

	err = sess.SetActivePage(context.Background(), "driver-page-123")
	if err != nil {
		t.Fatalf("SetActivePage failed: %v", err)
	}
}

func TestSession_ReportFrame_InvokesCallback(t *testing.T) {
	t.Parallel()

	var receivedSessionID string
	var receivedFrame *FrameInfo

	sess := &Session{
		id:   "frame-session",
		mode: ModeRecording,
		recording: &RecordingCallbacks{
			OnFrame: func(sessionID string, frame *FrameInfo) {
				receivedSessionID = sessionID
				receivedFrame = frame
			},
		},
	}

	frame := &FrameInfo{
		Data:      []byte("framedata"),
		MediaType: "image/jpeg",
		Width:     1920,
		Height:    1080,
	}

	sess.ReportFrame(frame)

	if receivedSessionID != "frame-session" {
		t.Errorf("expected session ID 'frame-session', got '%s'", receivedSessionID)
	}

	if receivedFrame == nil {
		t.Fatal("expected frame to be received")
	}

	if string(receivedFrame.Data) != "framedata" {
		t.Errorf("expected frame data 'framedata', got '%s'", string(receivedFrame.Data))
	}
}

func TestSession_ReportFrame_NilCallback(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:   "frame-session",
		mode: ModeRecording,
		recording: &RecordingCallbacks{
			OnFrame: nil, // No frame callback
		},
	}

	frame := &FrameInfo{
		Data:      []byte("framedata"),
		MediaType: "image/jpeg",
	}

	// Should not panic when OnFrame is nil
	sess.ReportFrame(frame)
}

func TestSession_Recording_Accessor(t *testing.T) {
	t.Parallel()

	callbacks := &RecordingCallbacks{
		OnAction: func(sessionID string, action *RecordedActionInfo) {},
	}

	sess := &Session{
		id:        "rec-session",
		mode:      ModeRecording,
		recording: callbacks,
	}

	if sess.Recording() != callbacks {
		t.Error("expected Recording() to return the configured callbacks")
	}

	// Test with nil recording
	sess2 := &Session{
		id:        "no-rec-session",
		mode:      ModeExecution,
		recording: nil,
	}

	if sess2.Recording() != nil {
		t.Error("expected Recording() to return nil when not configured")
	}
}
