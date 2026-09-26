package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// completedTerminal models an acknowledged operation for closed-session guards.
func completedTerminal() *terminalOperation {
	done := make(chan struct{})
	close(done)
	return &terminalOperation{done: done}
}

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

func TestCloseWithArtifacts_TreatsAbsentDriverSessionAsTerminal(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/session/sess-absent/close", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	terminalCalls := 0
	sess := &Session{
		id:         "sess-absent",
		mode:       ModeExecution,
		client:     client,
		onTerminal: func() { terminalCalls++ },
	}

	artifacts, err := sess.CloseWithArtifacts(context.Background())
	if err != nil {
		t.Fatalf("close absent session: %v", err)
	}
	if artifacts != nil {
		t.Fatalf("expected no artifacts for already-absent session, got %#v", artifacts)
	}
	if terminalCalls != 1 {
		t.Fatalf("expected one terminal callback, got %d", terminalCalls)
	}
	if !sess.isClosed() {
		t.Fatal("expected absent session to remain terminal")
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

	_, err := sess.ForwardInput(context.Background(), []byte("{}"))
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
		id:       "test-session",
		mode:     ModeRecording,
		terminal: completedTerminal(),
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
	_, fwdErr := sess.ForwardInput(context.Background(), []byte("{}"))
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
		var envelope map[string]any
		_ = json.NewDecoder(r.Body).Decode(&envelope)
		if envelope["execution_id"] != "record-owner" || envelope["lease_id"] != "record-lease" {
			t.Errorf("recording mutation lacks immutable ownership: %v", envelope)
		}
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
		id:          "rec-session",
		executionID: "record-owner",
		leaseID:     "record-lease",
		mode:        ModeRecording,
		client:      client,
	}

	_, err = sess.StartRecording(context.Background(), &driver.StartRecordingRequest{
		CallbackURL:      "http://localhost:8080/callback",
		FrameCallbackURL: "http://localhost:8080/frame",
		FrameQuality:     80,
		FrameFPS:         10,
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

	_, err := sess.StartRecording(context.Background(), &driver.StartRecordingRequest{})

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
		id:       "closed-session",
		mode:     ModeRecording,
		terminal: completedTerminal(),
	}

	_, err := sess.StartRecording(context.Background(), &driver.StartRecordingRequest{})

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
		var envelope map[string]any
		_ = json.NewDecoder(r.Body).Decode(&envelope)
		if envelope["execution_id"] != "record-owner" || envelope["lease_id"] != "record-lease" {
			t.Errorf("recording mutation lacks immutable ownership: %v", envelope)
		}
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
		id:          "rec-session",
		executionID: "record-owner",
		leaseID:     "record-lease",
		mode:        ModeRecording,
		client:      client,
	}

	_, err = sess.StopRecording(context.Background())
	if err != nil {
		t.Fatalf("StopRecording failed: %v", err)
	}
}

func TestSession_StopRecording_RejectsClosedSession(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:       "closed-session",
		mode:     ModeRecording,
		terminal: completedTerminal(),
	}

	_, err := sess.StopRecording(context.Background())

	if err == nil {
		t.Error("expected error when stopping recording on closed session")
	}
}

// [REQ:BAS-RH-J17] Committed entries are acknowledged only under this Session's lease.
func TestSession_AcknowledgeRecordedActions_Ownership(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.Equal(t, "/session/session/record/actions/ack", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, map[string]any{"execution_id": "owner", "lease_id": "lease", "entry_ids": []any{"committed"}}, body)
		_, _ = w.Write([]byte(`{"entry_ids":["committed"]}`))
	}))
	defer server.Close()
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	sess := &Session{id: "session", executionID: "owner", leaseID: "lease", client: client, mode: ModeRecording}
	require.NoError(t, sess.AcknowledgeRecordedActions(context.Background(), []string{"committed"}))
	sess.terminal = completedTerminal()
	require.EqualError(t, sess.AcknowledgeRecordedActions(context.Background(), []string{"committed"}), "session closed")
	require.Equal(t, int32(1), calls.Load())
}

func TestSession_GetRecordingStatus_Success(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	handler.HandleFunc("/session/rec-session/record/status", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"is_recording": true,
			"action_count": 5,
			"duration_ms":  1234,
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
		id:       "closed-session",
		mode:     ModeRecording,
		terminal: completedTerminal(),
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

	actions, err := sess.GetRecordedActions(context.Background())
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
		id:       "closed-session",
		mode:     ModeRecording,
		terminal: completedTerminal(),
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
		id:       "closed-session",
		mode:     ModeRecording,
		terminal: completedTerminal(),
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
		id:       "artifact-session",
		mode:     ModeExecution,
		client:   client,
		terminal: completedTerminal(), // Intentionally closed - download should still work
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
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "owner", body["execution_id"])
		require.Equal(t, "lease", body["lease_id"])
		require.Equal(t, "page", body["expected_page_id"])
		_ = json.NewEncoder(w).Encode(map[string]any{"session_id": "vp-session", "driver_page_id": "page", "width": 1920, "height": 1080})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:          "vp-session",
		executionID: "owner", leaseID: "lease",
		mode:   ModeRecording,
		client: client,
	}

	_, err = sess.UpdateViewport(context.Background(), 1920, 1080, "page")
	if err != nil {
		t.Fatalf("UpdateViewport failed: %v", err)
	}
}

func TestSession_UpdateViewport_RejectsClosedSession(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:       "closed-session",
		mode:     ModeRecording,
		terminal: completedTerminal(),
	}

	_, err := sess.UpdateViewport(context.Background(), 1920, 1080, "page")

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
			"valid":       true,
			"match_count": 3,
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
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("reset owner envelope: %v", err)
		}
		if body["execution_id"] != "reset-owner" || body["lease_id"] != "reset-lease" {
			t.Errorf("reset omitted admitted ownership: %#v", body)
		}
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
		id:          "reset-session",
		executionID: "reset-owner",
		leaseID:     "reset-lease",
		mode:        ModeRecording,
		client:      client,
	}

	err = sess.Reset(context.Background())
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
}

func TestSession_ResetRequiresAcknowledgment(t *testing.T) {
	t.Parallel()
	for _, response := range []string{`{}`, `{"success":false}`} {
		t.Run(response, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(response))
			}))
			defer srv.Close()
			client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
			if err != nil {
				t.Fatal(err)
			}
			sess := &Session{id: "reset-session", executionID: "owner", leaseID: "lease", mode: ModeRecording, client: client}
			if err := sess.Reset(context.Background()); err == nil {
				t.Fatal("reset without acknowledgment reported success")
			}
		})
	}
}

func TestSession_Reset_RejectsClosedSession(t *testing.T) {
	t.Parallel()

	sess := &Session{
		id:       "closed-session",
		mode:     ModeRecording,
		terminal: completedTerminal(),
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
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["execution_id"] != "page-owner" || body["lease_id"] != "page-lease" || body["page_id"] != "driver-page-123" {
			t.Errorf("page switch lost immutable authority: %v", body)
		}
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
		id:          "page-session",
		executionID: "page-owner",
		leaseID:     "page-lease",
		mode:        ModeExecution,
		client:      client,
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

// A caller joining finalization must observe completion or its own cancellation,
// never the in-progress closed marker as an acknowledged terminal operation.
func TestTerminalOperationWaiterCancellation(t *testing.T) {
	for _, primaryClose := range []bool{true, false} {
		for _, joinClose := range []bool{true, false} {
			name := "release"
			if primaryClose {
				name = "close"
			}
			if joinClose {
				name += "-close"
			} else {
				name += "-release"
			}
			t.Run(name, func(t *testing.T) {
				entered, release := make(chan struct{}), make(chan struct{})
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					close(entered)
					<-release
					_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "video_paths": []string{"/owned/video.webm"}})
				}))
				defer server.Close()
				client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
				if err != nil {
					t.Fatal(err)
				}
				sess := &Session{id: "fixture", executionID: "owner", leaseID: "lease", client: client}
				call := func(ctx context.Context, closeSession bool) error {
					if closeSession {
						_, e := sess.CloseWithArtifacts(ctx)
						return e
					}
					return sess.Release(ctx)
				}
				primary := make(chan error, 1)
				go func() { primary <- call(context.Background(), primaryClose) }()
				<-entered
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if err := call(ctx, joinClose); !errors.Is(err, context.Canceled) {
					t.Errorf("joining pending terminal operation returned %v; want caller cancellation", err)
				}
				close(release)
				if err := <-primary; err != nil {
					t.Fatalf("waiter cancellation interrupted owner: %v", err)
				}
			})
		}
	}
}

// The observed context supplies a deterministic waiter-entry barrier without
// inspecting Session's locks or adding sleeps to the HTTP completion oracle.
type terminalWaitContext struct {
	context.Context
	waiting chan struct{}
	once    sync.Once
}

func (c *terminalWaitContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.waiting) })
	return c.Context.Done()
}

func TestTerminalOperationSharesAcknowledgedResult(t *testing.T) {
	acknowledged := &driver.CloseSessionResponse{Success: true, VideoPaths: []string{"/owned/video.webm"}}
	cases := map[string]struct {
		close, fail bool
		want        *driver.CloseSessionResponse
	}{
		"close-success": {true, false, acknowledged}, "release-success": {false, false, nil},
		"close-failure": {true, true, nil}, "release-failure": {false, true, nil},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			var unblock sync.Once
			var requests, notifications atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				first := requests.Add(1) == 1
				if first {
					close(entered)
					<-release
				}
				if first && tc.fail {
					http.Error(w, "synthetic finalization failure", http.StatusServiceUnavailable)
					return
				}
				_ = json.NewEncoder(w).Encode(acknowledged)
			}))
			defer server.Close()
			defer unblock.Do(func() { close(release) })
			client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
			require.NoError(t, err)
			sess := &Session{id: "fixture", executionID: "owner", leaseID: "lease", client: client, onTerminal: func() { notifications.Add(1) }}
			type result struct {
				artifacts *driver.CloseSessionResponse
				err       error
			}
			primary := make(chan result, 1)
			go func() {
				if tc.close {
					a, e := sess.CloseWithArtifacts(context.Background())
					primary <- result{a, e}
				} else {
					primary <- result{nil, sess.Release(context.Background())}
				}
			}()
			<-entered
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			waiting := &terminalWaitContext{Context: ctx, waiting: make(chan struct{})}
			joined := make(chan result, 1)
			go func() { a, e := sess.CloseWithArtifacts(waiting); joined <- result{a, e} }()
			select {
			case <-waiting.waiting:
			case early := <-joined:
				t.Fatalf("close returned before HTTP acknowledgment: %#v", early)
			}
			require.Zero(t, notifications.Load(), "terminal callback before acknowledgment")
			unblock.Do(func() { close(release) })
			first, second := <-primary, <-joined
			require.EqualValues(t, 1, requests.Load(), "duplicate concurrent cleanup request")
			require.Equal(t, tc.want, first.artifacts)
			require.Equal(t, tc.want, second.artifacts)
			if tc.fail {
				require.Error(t, first.err)
				require.ErrorIs(t, second.err, first.err, "waiters must share the same failed attempt")
				require.False(t, sess.isClosed(), "failed cleanup must retain ownership")
				require.Zero(t, notifications.Load())
				a, e := sess.CloseWithArtifacts(context.Background())
				require.NoError(t, e)
				require.Equal(t, acknowledged, a, "explicit retry must retain artifacts")
				require.EqualValues(t, 2, requests.Load())
			} else {
				require.NoError(t, first.err)
				require.NoError(t, second.err)
				_, e := sess.CloseWithArtifacts(context.Background())
				require.NoError(t, e)
				require.EqualValues(t, 1, requests.Load(), "terminal retry repeated cleanup")
			}
			require.True(t, sess.isClosed())
			require.EqualValues(t, 1, notifications.Load(), "terminal ownership notification must occur once")
		})
	}
}

func TestTerminalOperationRejectsMissingAcknowledgment(t *testing.T) {
	for _, operation := range []string{"close", "release"} {
		for _, body := range []string{`{}`, `{"success":false}`, `{"success":false,"trace_path":"/partial/trace.zip"}`} {
			t.Run(operation+"/"+body, func(t *testing.T) {
				var requests, notifications atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if requests.Add(1) == 1 {
						_, _ = w.Write([]byte(body))
						return
					}
					_, _ = w.Write([]byte(`{"success":true}`))
				}))
				defer server.Close()
				client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
				require.NoError(t, err)
				sess := &Session{id: "fixture", client: client, onTerminal: func() { notifications.Add(1) }}
				var observedArtifacts *driver.CloseSessionResponse
				call := func() error {
					if operation == "close" {
						var e error
						observedArtifacts, e = sess.CloseWithArtifacts(context.Background())
						return e
					}
					return sess.Release(context.Background())
				}
				require.Error(t, call(), "HTTP success alone does not acknowledge the terminal operation")
				if operation == "close" {
					var expected driver.CloseSessionResponse
					require.NoError(t, json.Unmarshal([]byte(body), &expected))
					require.Equal(t, &expected, observedArtifacts, "partial artifact metadata must accompany the explicit failure")
				}
				require.False(t, sess.isClosed(), "missing acknowledgment must retain ownership")
				require.Zero(t, notifications.Load())
				require.NoError(t, call(), "explicit retry with acknowledgment should complete")
				require.EqualValues(t, 1, notifications.Load())
				require.EqualValues(t, 2, requests.Load())
			})
		}
	}
}

func TestRunTransportsAdmittedExecutionLease(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		bodies = append(bodies, body)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	current := &Session{id: "leased-session", executionID: "execution-current", leaseID: "lease-current", mode: ModeExecution, client: client}
	instruction := contracts.CompiledInstruction{NodeID: "click", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK, Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: "#button"}}}}
	for i := 0; i < 2; i++ {
		_, err := current.Run(context.Background(), instruction)
		require.NoError(t, err)
	}
	require.Len(t, bodies, 2)
	for _, body := range bodies {
		require.Equal(t, "execution-current", body["execution_id"])
		require.Equal(t, "lease-current", body["lease_id"])
		require.NotNil(t, body["instruction"])
	}
}

func TestRunAllocatesDistinctTransportOperations(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		bodies = append(bodies, body)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	current := &Session{id: "session", executionID: "execution", leaseID: "lease", mode: ModeExecution, client: client}
	instruction := contracts.CompiledInstruction{NodeID: "loop-node", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK}}
	for i := 0; i < 2; i++ {
		_, err := current.Run(context.Background(), instruction)
		require.NoError(t, err)
	}
	require.Len(t, bodies, 2)
	require.Equal(t, float64(1), bodies[0]["operation_sequence"])
	require.Equal(t, float64(2), bodies[1]["operation_sequence"])
	require.NotEmpty(t, bodies[0]["invocation_id"])
	require.NotEqual(t, bodies[0]["invocation_id"], bodies[1]["invocation_id"])
	require.Equal(t, float64(1), bodies[0]["attempt"])
}

func TestRunAmbiguousResponseCannotAuthorizeNewAttempt(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			effects := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				effects++
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"success":`)) // Effect occurred, response cannot establish its outcome.
			}))
			defer server.Close()
			client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
			require.NoError(t, err)
			current := &Session{id: "session", executionID: "execution", leaseID: "lease", mode: ModeExecution, client: client}
			outcome, err := current.Run(context.Background(), contracts.CompiledInstruction{NodeID: "effect", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK}})
			require.Error(t, err)
			require.NotNil(t, outcome.Failure, "executor needs explicit repeat policy after an ambiguous response")
			require.False(t, outcome.Failure.Retryable)
			require.Equal(t, "INSTRUCTION_OUTCOME_UNCERTAIN", outcome.Failure.Code)
			require.Equal(t, 1, effects)
		})
	}
}

// [REQ:BAS-RH-J17] Live input carries the immutable lease already held by Session.
func TestSession_ForwardInputCarriesOwnership(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input map[string]any
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if input["execution_id"] != "input-owner" || input["lease_id"] != "input-lease" {
			t.Errorf("input lacks immutable ownership: %v", input)
		}
		if input["type"] != "pointer" || input["action"] != "click" || input["x"] != float64(12) {
			t.Errorf("input payload changed: %v", input)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "applied_sequence": 1, "coalesced_count": 0})
	}))
	defer srv.Close()
	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	sess := &Session{id: "input-session", executionID: "input-owner", leaseID: "input-lease", mode: ModeRecording, client: client}
	if _, err := sess.ForwardInput(context.Background(), []byte(`{"type":"pointer","action":"click","x":12}`)); err != nil {
		t.Fatal(err)
	}
}

// [REQ:BAS-RH-J17] All navigation commands retain the same immutable Session authority.
func TestSessionNavigationCarriesImmutableLease(t *testing.T) {
	for _, operation := range []driver.HistoryNavigation{"navigate", driver.HistoryReload, driver.HistoryBack, driver.HistoryForward} {
		t.Run(string(operation), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, "/session/owned/record/"+string(operation), r.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				expected := map[string]any{"execution_id": "owner", "lease_id": "lease", "wait_until": "domcontentloaded", "timeout_ms": float64(1234)}
				if operation == "navigate" {
					expected["url"] = "https://fixture.test"
					expected["capture"] = true
				}
				require.Equal(t, expected, body)
				_, _ = w.Write([]byte(`{"driver_page_id":"initial-driver-page","session_id":"owned","url":"https://fixture.test","title":"fixture","can_go_back":true,"can_go_forward":false}`))
			}))
			defer server.Close()
			client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
			require.NoError(t, err)
			session := &Session{id: "owned", executionID: "owner", leaseID: "lease", mode: ModeRecording, client: client}
			invoke := func() error {
				if operation == "navigate" {
					_, err := session.Navigate(context.Background(), "https://fixture.test", WithWaitUntil("domcontentloaded"), WithNavigateTimeout(1234), WithCapture(true))
					return err
				}
				_, err := session.NavigateHistory(context.Background(), operation, &driver.HistoryNavigationRequest{WaitUntil: "domcontentloaded", TimeoutMs: 1234})
				return err
			}
			require.NoError(t, invoke())
			session.terminal = completedTerminal()
			require.EqualError(t, invoke(), "session closed")
			require.Equal(t, int32(1), calls.Load())
		})
	}
}

// [REQ:BAS-RH-J22] A frame cannot outlive a lease or follow another selected page.
func TestFramePageOwnership(t *testing.T) {
	owner := &Session{id: "session", executionID: "execution", leaseID: "lease"}
	owner.InitializePageTracking("https://red.test")
	owner.Pages().SetInitialPageDriverID("red")
	valid := driver.FrameSource{SessionID: "session", ExecutionID: "execution", LeaseID: "lease", PageID: "red"}
	id, ok := owner.FramePage(&valid)
	require.True(t, ok)
	require.Equal(t, owner.Pages().GetActivePageID(), id)
	for _, field := range []string{"nil", "session", "execution", "lease", "page", "empty lease"} {
		t.Run(field, func(t *testing.T) {
			source := valid
			switch field {
			case "session":
				source.SessionID = "other"
			case "execution":
				source.ExecutionID = "other"
			case "lease":
				source.LeaseID = "other"
			case "page":
				source.PageID = "other"
			case "empty lease":
				source.LeaseID = ""
			}
			candidate := &source
			if field == "nil" {
				candidate = nil
			}
			_, accepted := owner.FramePage(candidate)
			require.False(t, accepted)
		})
	}
	owner.mu.Lock()
	owner.terminal = completedTerminal()
	owner.mu.Unlock()
	_, ok = owner.FramePage(&valid)
	require.False(t, ok)
}

// [REQ:BAS-RH-J03] Browser reads retain the immutable lease and reject terminal Sessions.
func TestSessionHistoryReadCarriesImmutableLease(t *testing.T) {
	for _, stack := range []bool{false, true} {
		t.Run(fmt.Sprint(stack), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, "owner&value", r.URL.Query().Get("execution_id"))
				require.Equal(t, "lease&value", r.URL.Query().Get("lease_id"))
				require.Equal(t, "page&value", r.URL.Query().Get("expected_page_id"))
				_, _ = w.Write([]byte(`{"session_id":"owned","can_go_back":true,"back_stack":[]}`))
			}))
			defer server.Close()
			client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
			require.NoError(t, err)
			owner := &Session{id: "owned", executionID: "owner&value", leaseID: "lease&value", mode: ModeRecording, client: client}
			invoke := func() error {
				if stack {
					_, err := owner.GetNavigationStack(context.Background(), "page&value")
					return err
				}
				_, err := owner.GetNavigationState(context.Background(), "page&value")
				return err
			}
			require.NoError(t, invoke())
			owner.terminal = completedTerminal()
			require.EqualError(t, invoke(), "session closed")
			require.Equal(t, int32(1), calls.Load())
		})
	}
}
