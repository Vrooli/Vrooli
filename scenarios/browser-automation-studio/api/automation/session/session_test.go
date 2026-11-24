package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// completedTerminal models an acknowledged operation for closed-session guards.
func completedTerminal() *terminalOperation {
	done := make(chan struct{})
	close(done)
	return &terminalOperation{done: done}
}

type navigationHistoryAuthorizer struct {
	requests []sessionpermission.ActionRequest
	err      error
}

func (a *navigationHistoryAuthorizer) Authorize(_ context.Context, req sessionpermission.ActionRequest) error {
	a.requests = append(a.requests, req)
	return a.err
}

type navigationHistoryCaller struct{}

func (navigationHistoryCaller) BrowserCallerID(context.Context) (string, error) {
	return "run:history-test", nil
}

func TestGetNavigationStackAuthorizesHistoryWithTrustedCurrentURL(t *testing.T) {
	var stackReads int
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session/history-session/record/navigation-state":
			_, _ = w.Write([]byte(`{"session_id":"history-session","url":"https://allowed.example/path","title":"Allowed"}`))
		case "/session/history-session/record/navigation-stack":
			stackReads++
			_, _ = w.Write([]byte(`{"session_id":"history-session","current":{"url":"https://allowed.example/path","title":"Allowed"},"back_stack":[{"url":"https://allowed.example/prior"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	authorizer := &navigationHistoryAuthorizer{}
	owner := &Session{
		id: "history-session", executionID: "exec-history", leaseID: "lease-history",
		initialDriverPageID: "page-history", activeDriverPageID: "page-history", client: client,
		authorizer: authorizer, callerResolver: navigationHistoryCaller{}, expectedCallerID: "run:history-test",
	}

	stack, err := owner.GetNavigationStack(context.Background(), "")
	require.NoError(t, err)
	require.Equal(t, 1, stackReads)
	require.NotNil(t, stack.Current)
	require.Len(t, authorizer.requests, 1)
	require.Equal(t, "history_read", authorizer.requests[0].Kind)
	require.Equal(t, "https://allowed.example/path", authorizer.requests[0].CurrentURL)
	require.Equal(t, "run:history-test", authorizer.requests[0].CallerID)
	require.Equal(t, "lease-history", authorizer.requests[0].LeaseID)
}

func TestBulkServiceWorkerUnregisterPreflightsEveryScopeBeforeEffect(t *testing.T) {
	workerScope := "https://outside.example/app/"
	var unregisterCalls atomic.Int32
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session/sw-session/record/navigation-state":
			_, _ = w.Write([]byte(`{"session_id":"sw-session","url":"https://allowed.example/app/","title":"Allowed"}`))
		case "/session/sw-session/service-workers":
			if r.Method == http.MethodDelete {
				unregisterCalls.Add(1)
				_, _ = w.Write([]byte(`{"session_id":"sw-session","unregistered_count":1}`))
				return
			}
			_, _ = w.Write([]byte(`{"session_id":"sw-session","workers":[{"registrationId":"worker-1","scopeURL":"` + workerScope + `","scriptURL":"https://outside.example/sw.js","status":"running"}],"control":{"mode":"allow"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	policy, err := sessionpermission.New(sessionpermission.Profile{
		Effect: sessionpermission.EffectAct, Approval: sessionpermission.ApprovalNone,
		View: true, Control: true, URLAllow: []string{"https://allowed.example/*"},
	}, nil)
	require.NoError(t, err)
	owner := &Session{
		id: "sw-session", executionID: "exec-sw", leaseID: "lease-sw", client: client,
		initialDriverPageID: "page-sw", activeDriverPageID: "page-sw", authorizer: policy,
		callerResolver: navigationHistoryCaller{}, expectedCallerID: "run:history-test",
	}

	_, err = owner.UnregisterAllServiceWorkers(context.Background())
	require.ErrorIs(t, err, sessionpermission.ErrDenied)
	require.Zero(t, unregisterCalls.Load(), "out-of-scope scope must deny before unregister-all")

	workerScope = "https://allowed.example/app/"
	_, err = owner.UnregisterAllServiceWorkers(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 1, unregisterCalls.Load())

	requiringApproval, err := sessionpermission.New(sessionpermission.Profile{
		Effect: sessionpermission.EffectAct, Approval: sessionpermission.ApprovalEveryAction,
		View: true, Control: true, URLAllow: []string{"https://allowed.example/*"},
	}, nil)
	require.NoError(t, err)
	owner.authorizer = requiringApproval
	_, err = owner.UnregisterAllServiceWorkers(context.Background())
	require.ErrorIs(t, err, sessionpermission.ErrDenied)
	require.EqualValues(t, 1, unregisterCalls.Load(), "missing approval must deny before bulk deletion")
}

func TestSessionRejectsMissingExpectedCallerBeforeDriverRead(t *testing.T) {
	reads := 0
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session/unbound/record/navigation-state" {
			reads++
			_, _ = w.Write([]byte(`{"session_id":"unbound","url":"https://allowed.example/","title":"Allowed"}`))
			return
		}
		http.NotFound(w, r)
	}))
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	authorizer := &navigationHistoryAuthorizer{}
	owner := &Session{id: "unbound", executionID: "exec-unbound", leaseID: "lease-unbound", initialDriverPageID: "page-unbound", activeDriverPageID: "page-unbound", client: client, authorizer: authorizer, callerResolver: navigationHistoryCaller{}}
	_, err = owner.GetNavigationState(context.Background(), "page-unbound")
	require.ErrorContains(t, err, "expected caller binding unavailable")
	require.Zero(t, reads)
	require.Empty(t, authorizer.requests)
}

func TestCaptureObservationChargesOneObserveAdmission(t *testing.T) {
	stateReads, screenshotReads := 0, 0
	var screenshotBody map[string]any
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session/observe-session/record/navigation-state":
			stateReads++
			_, _ = w.Write([]byte(`{"session_id":"observe-session","url":"https://allowed.example/path","title":"Allowed"}`))
		case "/session/observe-session/record/screenshot":
			screenshotReads++
			require.NoError(t, json.NewDecoder(r.Body).Decode(&screenshotBody))
			_, _ = w.Write([]byte(`{"data":"aW1hZ2U=","media_type":"image/png","width":1,"height":1,"captured_at":"now"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	authorizer := &navigationHistoryAuthorizer{}
	owner := &Session{
		id: "observe-session", executionID: "exec-observe", leaseID: "lease-observe",
		initialDriverPageID: "page-observe", activeDriverPageID: "page-observe", client: client,
		authorizer: authorizer, callerResolver: navigationHistoryCaller{}, expectedCallerID: "run:history-test",
	}
	state, screenshot, err := owner.CaptureObservationWithScreenshot(context.Background(), "page-observe", driver.CaptureScreenshotRequest{Format: "png", TestID: "scene-canvas-pixels", WaitUntil: "networkidle", SettleMs: 250})
	require.NoError(t, err)
	require.Equal(t, "https://allowed.example/path", state.URL)
	require.Equal(t, "aW1hZ2U=", screenshot.Data)
	require.Len(t, authorizer.requests, 1)
	require.Equal(t, "observe", authorizer.requests[0].Kind)
	require.Equal(t, "https://allowed.example/path", authorizer.requests[0].CurrentURL)
	require.Equal(t, 2, stateReads) // one admission URL read, one returned state read
	require.Equal(t, 1, screenshotReads)
	require.Equal(t, map[string]any{"execution_id": "exec-observe", "lease_id": "lease-observe", "format": "png", "test_id": "scene-canvas-pixels", "wait_until": "networkidle", "settle_ms": float64(250)}, screenshotBody)
}

func TestSensitiveTypedInputFailsClosedBeforeDriverExecution(t *testing.T) {
	owner := &Session{authorizer: &navigationHistoryAuthorizer{}}
	sensitive := true
	instruction := contracts.CompiledInstruction{Action: &basactions.ActionDefinition{
		Type: basactions.ActionType_ACTION_TYPE_INPUT,
		Params: &basactions.ActionDefinition_Input{Input: &basactions.InputParams{
			Selector: "#password", Value: "raw-secret", IsSensitive: &sensitive,
		}},
	}}
	err := owner.authorizeInstruction(context.Background(), instruction)
	require.ErrorContains(t, err, "raw sensitive input is unsupported")
	require.Empty(t, owner.authorizer.(*navigationHistoryAuthorizer).requests)
}

func TestReleaseInputSendsOwnerBoundReleaseForEveryPointerButton(t *testing.T) {
	var bodies []map[string]any
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "applied_sequence": len(bodies)})
	}))
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	owner := &Session{id: "release-session", executionID: "exec-release", leaseID: "lease-release", client: client}
	require.NoError(t, owner.ReleaseInput(context.Background()))
	require.Len(t, bodies, 5)
	want := []string{"left", "right", "middle", "back", "forward"}
	for i, button := range want {
		require.Equal(t, "pointer", bodies[i]["type"])
		require.Equal(t, "release", bodies[i]["action"])
		require.Equal(t, button, bodies[i]["button"])
		require.Equal(t, "exec-release", bodies[i]["execution_id"])
		require.Equal(t, "lease-release", bodies[i]["lease_id"])
	}
}

func TestCloseWithArtifactsOutcomes(t *testing.T) {
	cases := []struct {
		name           string
		status         int
		wantVideoPaths int
		wantTerminal   bool
	}{
		{name: "closed with artifacts", status: http.StatusOK, wantVideoPaths: 2},
		{name: "already absent is terminal", status: http.StatusNotFound, wantTerminal: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.NewServeMux()
			handler.HandleFunc("/session/test-session/close", func(w http.ResponseWriter, r *http.Request) {
				_ = r.Body.Close()
				if tc.status == http.StatusNotFound {
					http.Error(w, `{"error":"session not found"}`, tc.status)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"success":     true,
					"video_paths": []string{"/tmp/video-1.webm", "/tmp/video-2.webm"},
				})
			})
			server := testutil.StartHTTPServer(t, handler)
			client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
			if err != nil {
				t.Fatalf("client: %v", err)
			}

			terminalCalls := 0
			sess := &Session{
				id:         "test-session",
				mode:       ModeExecution,
				client:     client,
				onTerminal: func() { terminalCalls++ },
			}
			artifacts, err := sess.CloseWithArtifacts(context.Background())
			if err != nil {
				t.Fatalf("close: %v", err)
			}
			if tc.wantVideoPaths > 0 {
				if artifacts == nil || len(artifacts.VideoPaths) != tc.wantVideoPaths {
					t.Fatalf("close artifacts = %#v, want %d video paths", artifacts, tc.wantVideoPaths)
				}
			}
			if tc.wantTerminal {
				if artifacts != nil {
					t.Fatalf("expected no artifacts for absent session, got %#v", artifacts)
				}
				if terminalCalls != 1 || !sess.isClosed() {
					t.Fatalf("absent session terminal state = calls:%d closed:%v", terminalCalls, sess.isClosed())
				}
			}
		})
	}
}

// =============================================================================
// Mode Guard Tests
// =============================================================================

func TestSessionModeGuards(t *testing.T) {
	cases := []struct {
		name      string
		mode      Mode
		invoke    func(*Session) error
		wantError string
	}{
		{name: "run in recording mode", mode: ModeRecording, wantError: "recording-only mode", invoke: func(s *Session) error {
			_, err := s.Run(context.Background(), contracts.CompiledInstruction{})
			return err
		}},
		{name: "forward input in execution mode", mode: ModeExecution, wantError: "execution-only mode", invoke: func(s *Session) error { _, err := s.ForwardInput(context.Background(), []byte("{}")); return err }},
		{name: "start recording in execution mode", mode: ModeExecution, wantError: "execution-only mode", invoke: func(s *Session) error {
			_, err := s.StartRecording(context.Background(), &driver.StartRecordingRequest{})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.invoke(&Session{id: "test-session", mode: tc.mode})
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("error = %v, want message containing %q", err, tc.wantError)
			}
		})
	}
}

func TestSessionClosedOperationsRejectUse(t *testing.T) {
	operations := []struct {
		name              string
		invoke            func(*Session) error
		wantClosedMessage bool
	}{
		{name: "navigate", wantClosedMessage: true, invoke: func(s *Session) error { _, err := s.Navigate(context.Background(), "https://example.com"); return err }},
		{name: "start_recording", wantClosedMessage: true, invoke: func(s *Session) error {
			_, err := s.StartRecording(context.Background(), &driver.StartRecordingRequest{})
			return err
		}},
		{name: "stop_recording", invoke: func(s *Session) error { _, err := s.StopRecording(context.Background()); return err }},
		{name: "get_recording_status", invoke: func(s *Session) error { _, err := s.GetRecordingStatus(context.Background()); return err }},
		{name: "capture_screenshot", invoke: func(s *Session) error { _, err := s.CaptureScreenshot(context.Background()); return err }},
		{name: "get_storage_state", invoke: func(s *Session) error { _, err := s.GetStorageState(context.Background()); return err }},
		{name: "update_viewport", invoke: func(s *Session) error {
			_, err := s.UpdateViewport(context.Background(), 1920, 1080, "page")
			return err
		}},
		{name: "reset", invoke: func(s *Session) error { return s.Reset(context.Background()) }},
	}

	for _, tc := range operations {
		t.Run(tc.name, func(t *testing.T) {
			sess := &Session{id: "closed-session", mode: ModeRecording, terminal: completedTerminal()}
			err := tc.invoke(sess)
			if err == nil {
				t.Fatal("operation succeeded on a closed session")
			}
			if tc.wantClosedMessage && !strings.Contains(err.Error(), "session closed") {
				t.Fatalf("error = %v, want closed-session error", err)
			}
		})
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

	srv := testutil.StartHTTPServer(t, handler)

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

	srv := testutil.StartHTTPServer(t, handler)

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
		CallbackURL:    "http://localhost:8080/callback",
		FrameStreamURL: "ws://localhost:8080/ws/frames",
		FrameQuality:   80,
		FrameFPS:       10,
	})
	if err != nil {
		t.Fatalf("StartRecording failed: %v", err)
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

	srv := testutil.StartHTTPServer(t, handler)

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

func TestSession_StopRecordingChecksGrantBeforeDriverMutation(t *testing.T) {
	for _, historyRead := range []bool{false, true} {
		t.Run(fmt.Sprintf("history_read_%t", historyRead), func(t *testing.T) {
			var stops atomic.Int32
			server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/session/rec-session/record/navigation-state":
					_, _ = w.Write([]byte(`{"session_id":"rec-session","url":"https://allowed.example/app","title":"Allowed"}`))
				case "/session/rec-session/record/stop":
					stops.Add(1)
					_, _ = w.Write([]byte(`{"recording_id":"record-1","session_id":"rec-session","action_count":2,"stopped_at":"2026-11-21T12:00:00Z"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
			require.NoError(t, err)
			policy, err := sessionpermission.New(sessionpermission.Profile{
				Effect: sessionpermission.EffectAct, Approval: sessionpermission.ApprovalNone,
				Control: true, View: true, HistoryRead: historyRead,
				URLAllow: []string{"https://allowed.example/*"},
			}, nil)
			require.NoError(t, err)
			sess := &Session{id: "rec-session", executionID: "record-owner", leaseID: "record-lease",
				mode: ModeRecording, client: client, authorizer: policy,
				callerResolver: navigationHistoryCaller{}, expectedCallerID: "run:history-test",
				initialDriverPageID: "page-1", activeDriverPageID: "page-1"}
			_, err = sess.StopRecording(context.Background())
			if historyRead {
				require.NoError(t, err)
				require.EqualValues(t, 1, stops.Load())
			} else {
				require.ErrorIs(t, err, sessionpermission.ErrDenied)
				require.Zero(t, stops.Load(), "denied stop must not reach the driver mutation")
			}
		})
	}
}

// [REQ:BAS-RH-J17] Committed entries are acknowledged only under this Session's lease.
func TestSession_AcknowledgeRecordedActions_Ownership(t *testing.T) {
	var calls atomic.Int32
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.Equal(t, "/session/session/record/actions/ack", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, map[string]any{"execution_id": "owner", "lease_id": "lease", "entry_ids": []any{"committed"}}, body)
		_, _ = w.Write([]byte(`{"entry_ids":["committed"]}`))
	}))
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	sess := &Session{id: "session", executionID: "owner", leaseID: "lease", client: client, mode: ModeRecording}
	require.NoError(t, sess.AcknowledgeRecordedActions(context.Background(), []string{"committed"}))
	sess.terminal = completedTerminal()
	require.EqualError(t, sess.AcknowledgeRecordedActions(context.Background(), []string{"committed"}), "session closed")
	require.Equal(t, int32(1), calls.Load())
}

func TestSession_AcknowledgeRecordedActionsRequiresHistoryPermission(t *testing.T) {
	var acknowledgements atomic.Int32
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session/ack-session/record/navigation-state":
			_, _ = w.Write([]byte(`{"session_id":"ack-session","url":"https://allowed.example/path","title":"Allowed"}`))
		case "/session/ack-session/record/actions/ack":
			acknowledgements.Add(1)
			_, _ = w.Write([]byte(`{"entry_ids":["committed"]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	authorizer := &navigationHistoryAuthorizer{err: sessionpermission.ErrDenied}
	owner := &Session{
		id: "ack-session", executionID: "ack-owner", leaseID: "ack-lease", mode: ModeRecording,
		initialDriverPageID: "ack-page", activeDriverPageID: "ack-page", client: client,
		authorizer: authorizer, callerResolver: navigationHistoryCaller{}, expectedCallerID: "run:history-test",
	}

	err = owner.AcknowledgeRecordedActions(context.Background(), []string{"committed"})
	require.ErrorIs(t, err, sessionpermission.ErrDenied)
	require.Zero(t, acknowledgements.Load(), "denial must precede timeline acknowledgement")
	require.Len(t, authorizer.requests, 1)
	require.Equal(t, "history_read", authorizer.requests[0].Kind)
	require.Equal(t, "https://allowed.example/path", authorizer.requests[0].CurrentURL)

	authorizer.err = nil
	require.NoError(t, owner.AcknowledgeRecordedActions(context.Background(), []string{"committed"}))
	require.EqualValues(t, 1, acknowledgements.Load())
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

	srv := testutil.StartHTTPServer(t, handler)

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:          "rec-session",
		executionID: "recording-owner", leaseID: "recording-lease",
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

	srv := testutil.StartHTTPServer(t, handler)

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:          "rec-session",
		executionID: "recording-owner", leaseID: "recording-lease",
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
			"data":       "YmFzZTY0ZW5jb2RlZGRhdGE=",
			"media_type": "image/jpeg",
			"width":      1920,
			"height":     1080,
		})
	})

	srv := testutil.StartHTTPServer(t, handler)

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id: "ss-session", executionID: "ss-owner", leaseID: "ss-lease",
		mode: ModeRecording, client: client,
	}

	screenshot, err := sess.CaptureScreenshot(context.Background())
	if err != nil {
		t.Fatalf("CaptureScreenshot failed: %v", err)
	}

	if screenshot == nil {
		t.Fatal("expected non-nil screenshot")
	}

	if screenshot.Data != "YmFzZTY0ZW5jb2RlZGRhdGE=" {
		t.Errorf("expected base64 image data, got '%s'", screenshot.Data)
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

	srv := testutil.StartHTTPServer(t, handler)

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:          "storage-session",
		executionID: "storage-owner",
		leaseID:     "storage-lease",
		mode:        ModeRecording,
		client:      client,
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

	srv := testutil.StartHTTPServer(t, handler)

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

func TestSessionDownloadArtifactRejectsInvalidInputs(t *testing.T) {
	client, err := driver.NewClientWithURL("http://127.0.0.1:39400", driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	cases := []struct {
		name      string
		invoke    func() error
		wantError string
	}{
		{name: "nil session", invoke: func() error {
			var s *Session
			_, err := s.DownloadArtifact(context.Background(), "/path/to/video.webm")
			return err
		}},
		{name: "empty path", wantError: "path required", invoke: func() error {
			s := &Session{id: "artifact-session", mode: ModeExecution, client: client}
			_, err := s.DownloadArtifact(context.Background(), "")
			return err
		}},
		{name: "nil client", wantError: "client unavailable", invoke: func() error {
			s := &Session{id: "no-client-session", mode: ModeExecution}
			_, err := s.DownloadArtifact(context.Background(), "/path/to/video.webm")
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.invoke()
			if err == nil {
				t.Fatal("DownloadArtifact accepted invalid setup")
			}
			if tc.wantError != "" && !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("error = %v, want message containing %q", err, tc.wantError)
			}
		})
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

	srv := testutil.StartHTTPServer(t, handler)

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

	srv := testutil.StartHTTPServer(t, handler)

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id: "sel-session", executionID: "sel-owner", leaseID: "sel-lease",
		mode: ModeRecording, client: client,
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

	srv := testutil.StartHTTPServer(t, handler)

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
			srv := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(response))
			}))
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

	srv := testutil.StartHTTPServer(t, handler)

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
				server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					close(entered)
					<-release
					_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "video_paths": []string{"/owned/video.webm"}})
				}))
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
			server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if requests.Add(1) == 1 {
						_, _ = w.Write([]byte(body))
						return
					}
					_, _ = w.Write([]byte(`{"success":true}`))
				}))
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
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		bodies = append(bodies, body)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
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
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		bodies = append(bodies, body)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
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
			server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				effects++
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"success":`)) // Effect occurred, response cannot establish its outcome.
			}))
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
	srv := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	valid := driver.FrameSource{StreamKind: "recording", SessionID: "session", ExecutionID: "execution", LeaseID: "lease", PageID: "red"}
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
			server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, "owner&value", r.URL.Query().Get("execution_id"))
				require.Equal(t, "lease&value", r.URL.Query().Get("lease_id"))
				require.Equal(t, "page&value", r.URL.Query().Get("expected_page_id"))
				_, _ = w.Write([]byte(`{"session_id":"owned","can_go_back":true,"back_stack":[]}`))
			}))
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
