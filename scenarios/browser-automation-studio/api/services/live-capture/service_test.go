package livecapture

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	autosession "github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/services/recording"
	"github.com/vrooli/browser-automation-studio/services/recording/persistence"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/compiler"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
)

func TestNewService_InitializesComponents(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel) // Suppress logs during tests

	// NewService without a working driver will have nil sessions, but should still
	// create the generator. Pass nil for unified recording service in tests.
	svc := NewService(log, nil)

	// Generator should be created even if session manager fails
	if svc.generator == nil {
		t.Error("Expected generator to be initialized")
	}
}

func TestService_CreateSession_RequiresSessionManager(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	// Create service with nil session manager
	svc := &Service{
		sessions:  nil,
		generator: NewWorkflowGenerator(),
		log:       log,
	}

	_, err := svc.CreateSession(context.Background(), &SessionConfig{
		ViewportWidth:  1280,
		ViewportHeight: 720,
	})

	if err == nil {
		t.Error("Expected error when session manager is nil")
	}
	if err.Error() != "session manager not initialized" {
		t.Errorf("Expected 'session manager not initialized' error, got: %v", err)
	}
}

// Note: Testing "no actions to convert" error path would require either:
// 1. A mock session manager, or
// 2. Providing no actions AND expecting the service to fetch from session
// The ApplyActionRange function returns all actions when range is invalid,
// so we can't easily test the empty actions path without mocking.

func TestService_GenerateWorkflow_WithActions(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := &Service{
		sessions:  nil,
		generator: NewWorkflowGenerator(),
		log:       log,
	}

	actions := []driver.RecordedAction{
		{ActionType: "navigate", URL: "https://example.com"},
		{ActionType: "click", Selector: &driver.SelectorSet{Primary: "#btn"}},
	}

	result, err := svc.GenerateWorkflow(context.Background(), "test-session", &GenerateWorkflowConfig{
		Actions: actions,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ActionCount != 2 {
		t.Errorf("Expected ActionCount 2, got %d", result.ActionCount)
	}
	if result.FlowDefinition == nil {
		t.Error("Expected FlowDefinition to be non-nil")
	}
	if result.NodeCount < 2 {
		t.Errorf("Expected at least 2 nodes, got %d", result.NodeCount)
	}
}

func TestService_GenerateWorkflow_AppliesActionRange(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := &Service{
		sessions:  nil,
		generator: NewWorkflowGenerator(),
		log:       log,
	}

	actions := []driver.RecordedAction{
		{ActionType: "navigate", URL: "https://example.com"},
		{ActionType: "click", Selector: &driver.SelectorSet{Primary: "#btn1"}},
		{ActionType: "click", Selector: &driver.SelectorSet{Primary: "#btn2"}},
		{ActionType: "click", Selector: &driver.SelectorSet{Primary: "#btn3"}},
	}

	// Only use actions at index 1 and 2
	result, err := svc.GenerateWorkflow(context.Background(), "test-session", &GenerateWorkflowConfig{
		Actions:     actions,
		ActionRange: &ActionRange{Start: 1, End: 2},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ActionCount != 2 {
		t.Errorf("Expected ActionCount 2 (after range filter), got %d", result.ActionCount)
	}
}

func TestService_DriverClient_ReturnsNilWhenNoSessions(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := &Service{
		sessions:  nil,
		generator: NewWorkflowGenerator(),
		log:       log,
	}

	client := svc.DriverClient()
	if client != nil {
		t.Error("Expected nil DriverClient when sessions is nil")
	}
}

func TestService_Sessions_ReturnsNilWhenNotInitialized(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := &Service{
		sessions:  nil,
		generator: NewWorkflowGenerator(),
		log:       log,
	}

	sessions := svc.Sessions()
	if sessions != nil {
		t.Error("Expected nil Sessions when not initialized")
	}
}

// Note: GetSession requires a valid sessions manager and will panic if nil.
// Testing with nil sessions is not meaningful since callers should check Sessions() first.

// Recorded payloads must survive the complete service-to-typed-candidate boundary.
func TestGenerateWorkflowPreservesRecordedSemantics(t *testing.T) {
	tests := []struct {
		name, kind string
		payload    map[string]any
		check      func(*testing.T, *basactions.ActionDefinition)
	}{
		{"control double click", "click", map[string]any{"button": "right", "clickCount": 2, "modifiers": []string{"ctrl", "shift"}, "delay": 25}, func(t *testing.T, a *basactions.ActionDefinition) {
			p := a.GetClick()
			require.NotNil(t, p)
			require.Equal(t, int32(2), p.GetClickCount())
			require.Equal(t, int32(25), p.GetDelayMs())
			require.Equal(t, basactions.MouseButton_MOUSE_BUTTON_RIGHT, p.GetButton())
			require.Equal(t, []basactions.KeyboardModifier{basactions.KeyboardModifier_KEYBOARD_MODIFIER_CTRL, basactions.KeyboardModifier_KEYBOARD_MODIFIER_SHIFT}, p.Modifiers)
		}},
		{"keyboard modifiers", "keyboard", map[string]any{"key": "a", "modifiers": []any{"ctrl"}}, func(t *testing.T, a *basactions.ActionDefinition) {
			require.Equal(t, []basactions.KeyboardModifier{basactions.KeyboardModifier_KEYBOARD_MODIFIER_CTRL}, a.GetKeyboard().Modifiers)
		}},
		{"both scroll axes", "scroll", map[string]any{"scrollX": 200.0, "scrollY": 400.0, "deltaX": 50.0, "deltaY": 90.0}, func(t *testing.T, a *basactions.ActionDefinition) {
			p := a.GetScroll()
			require.Equal(t, int32(200), p.GetX())
			require.Equal(t, int32(400), p.GetY())
			require.Nil(t, p.DeltaX)
			require.Nil(t, p.DeltaY)
		}},
		{"focus stays focus", "focus", nil, func(t *testing.T, a *basactions.ActionDefinition) {
			require.NotNil(t, a.GetFocus())
			require.Equal(t, "#fixture", a.GetFocus().Selector)
		}},
		{"blur stays blur", "blur", nil, func(t *testing.T, a *basactions.ActionDefinition) {
			require.NotNil(t, a.GetBlur())
			require.Equal(t, "#fixture", a.GetBlur().GetSelector())
		}},
		{"drag complete", "dragDrop", map[string]any{"targetSelector": "#target"}, func(t *testing.T, a *basactions.ActionDefinition) {
			require.NotNil(t, a.GetDragDrop())
			require.Equal(t, "#fixture", a.GetDragDrop().SourceSelector)
			require.Equal(t, "#target", a.GetDragDrop().GetTargetSelector())
		}},
		{"browser drop", "drag-drop", map[string]any{"phase": "drop", "sourceSelector": "#source", "targetSelector": "#target"}, func(t *testing.T, a *basactions.ActionDefinition) {
			require.NotNil(t, a.GetDragDrop())
			require.Equal(t, "#source", a.GetDragDrop().SourceSelector)
			require.Equal(t, "#target", a.GetDragDrop().GetTargetSelector())
		}},
		{"empty input replaces", "type", map[string]any{"text": "", "submit": true}, func(t *testing.T, a *basactions.ActionDefinition) {
			require.NotNil(t, a.GetInput())
			require.Equal(t, "", a.GetInput().Value)
			require.True(t, a.GetInput().GetClearFirst())
			require.True(t, a.GetInput().GetSubmit())
		}},
		{"select empty option", "select", map[string]any{"value": ""}, func(t *testing.T, a *basactions.ActionDefinition) {
			require.IsType(t, &basactions.SelectParams_Value{}, a.GetSelectOption().SelectBy)
			require.Equal(t, "", a.GetSelectOption().GetValue())
		}},
		{"hover", "hover", map[string]any{"timeoutMs": 2500}, func(t *testing.T, a *basactions.ActionDefinition) {
			require.Equal(t, "#fixture", a.GetHover().Selector)
			require.Equal(t, int32(2500), a.GetHover().GetTimeoutMs())
		}},
		{"selector wait", "wait", map[string]any{"timeoutMs": 3500}, func(t *testing.T, a *basactions.ActionDefinition) {
			require.Equal(t, "#fixture", a.GetWait().GetSelector())
			require.Equal(t, int32(3500), a.GetWait().GetTimeoutMs())
		}},
		{"screenshot options", "screenshot", map[string]any{"fullPage": true, "quality": 90}, func(t *testing.T, a *basactions.ActionDefinition) {
			require.True(t, a.GetScreenshot().GetFullPage())
			require.Equal(t, int32(90), a.GetScreenshot().GetQuality())
		}},
		{"assertion", "assert", map[string]any{"mode": "text_equals", "expected": "expected"}, func(t *testing.T, a *basactions.ActionDefinition) {
			require.Equal(t, basbase.AssertionMode_ASSERTION_MODE_TEXT_EQUALS, a.GetAssert().Mode)
			require.NotNil(t, a.GetAssert().Expected)
		}},
	}
	svc := &Service{generator: NewWorkflowGenerator()}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.GenerateWorkflow(context.Background(), "fixture", &GenerateWorkflowConfig{Actions: []driver.RecordedAction{{ActionType: tt.kind, Selector: &driver.SelectorSet{Primary: "#fixture"}, Payload: tt.payload}}})
			require.NoError(t, err)
			flow := result.FlowDefinition
			require.Len(t, flow.Nodes, 1)
			bytes, err := protojson.Marshal(flow)
			require.NoError(t, err)
			restored := &basworkflows.WorkflowDefinitionV2{}
			require.NoError(t, protojson.Unmarshal(bytes, restored))
			require.True(t, proto.Equal(flow, restored))
			tt.check(t, flow.Nodes[0].Action)
			_, instructions, err := compiler.CompileWorkflowToContracts(context.Background(), uuid.New(), &basapi.WorkflowSummary{Id: uuid.NewString(), FlowDefinition: flow})
			require.NoError(t, err)
			require.Len(t, instructions, 1)
			require.True(t, proto.Equal(flow.Nodes[0].Action, instructions[0].Action))
			tt.check(t, instructions[0].Action)
		})
	}
}

func TestGenerateWorkflowRejectsUnrepresentableRecording(t *testing.T) {
	svc := &Service{generator: NewWorkflowGenerator()}
	cases := map[string][]driver.RecordedAction{
		"missing input snapshot": {{ActionType: "type", Selector: &driver.SelectorSet{Primary: "#fixture"}}},
		"ambiguous page":         {{ActionType: "click", PageID: "first", Selector: &driver.SelectorSet{Primary: "#one"}}, {ActionType: "click", Selector: &driver.SelectorSet{Primary: "#two"}}},
		"unknown action":         {{ActionType: "not-a-recorded-action", Selector: &driver.SelectorSet{Primary: "#fixture"}}},
		"frame":                  {{ActionType: "click", FrameID: "child", Selector: &driver.SelectorSet{Primary: "#fixture"}}},
		"multiple pages":         {{ActionType: "click", PageID: "first", Selector: &driver.SelectorSet{Primary: "#one"}}, {ActionType: "click", PageID: "second", Selector: &driver.SelectorSet{Primary: "#two"}}},
		"unfinished drag":        {{ActionType: "drag-drop", Payload: map[string]any{"phase": "start"}, Selector: &driver.SelectorSet{Primary: "#source"}}},
	}
	for name, actions := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := svc.GenerateWorkflow(context.Background(), "fixture", &GenerateWorkflowConfig{Actions: actions})
			require.Error(t, err)
			require.Nil(t, result)
		})
	}
}

func TestGenerateWorkflowJoinsDragPhases(t *testing.T) {
	svc := &Service{generator: NewWorkflowGenerator()}
	actions := []driver.RecordedAction{
		{ActionType: "drag-drop", Selector: &driver.SelectorSet{Primary: "#source"}, Payload: map[string]any{"phase": "start"}},
		{ActionType: "drag-drop", Selector: &driver.SelectorSet{Primary: "#target"}, Payload: map[string]any{"phase": "drop", "sourceSelector": "#source", "targetSelector": "#target"}},
	}
	result, err := svc.GenerateWorkflow(context.Background(), "fixture", &GenerateWorkflowConfig{Actions: actions})
	require.NoError(t, err)
	require.Len(t, result.FlowDefinition.Nodes, 1)
	require.Equal(t, "#source", result.FlowDefinition.Nodes[0].Action.GetDragDrop().SourceSelector)
	require.Equal(t, "#target", result.FlowDefinition.Nodes[0].Action.GetDragDrop().GetTargetSelector())
	require.Equal(t, "drag-drop", actions[1].ActionType)
}

func TestGenerateWorkflowNavigationAndDurationWait(t *testing.T) {
	flow, err := NewWorkflowGenerator().GenerateWorkflow([]driver.RecordedAction{
		{ActionType: "navigate", URL: "https://fixture.invalid", Payload: map[string]any{"waitForSelector": "#ready", "timeoutMs": 4000}},
		{ActionType: "wait", Payload: map[string]any{"timeoutMs": 75}},
	})
	require.NoError(t, err)
	require.Len(t, flow.Nodes, 2)
	p := flow.Nodes[0].Action.GetNavigate()
	require.Equal(t, "https://fixture.invalid", p.Url)
	require.Equal(t, "#ready", p.GetWaitForSelector())
	require.Equal(t, int32(4000), p.GetTimeoutMs())
	require.Equal(t, int32(75), flow.Nodes[1].Action.GetWait().GetDurationMs())
}

func TestCreateSessionJournalFailureReleasesBrowser(t *testing.T) {
	for _, response := range []string{`{"success":true}`, `{}`, `{"success":false}`} {
		t.Run(response, func(t *testing.T) {
			var closed atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/session/start":
					_, _ = w.Write([]byte(`{"session_id":"failed-journal","lease_id":"lease"}`))
				case "/session/failed-journal/close":
					closed.Add(1)
					_, _ = w.Write([]byte(response))
				default:
					t.Errorf("unexpected driver request: %s", r.URL.Path)
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			t.Setenv(driver.PlaywrightDriverEnv, server.URL)
			manager, err := autosession.NewManager()
			require.NoError(t, err)
			repo := persistence.NewMockRepository()
			repo.CreateSessionErr = errors.New("journal storage unavailable")
			svc := &Service{sessions: manager, log: logrus.New(), unifiedRecordingSvc: recording.NewService(repo, recording.ServiceConfig{})}
			result, err := svc.CreateSession(context.Background(), &SessionConfig{})
			require.ErrorContains(t, err, "journal storage unavailable")
			require.Nil(t, result)
			require.Equal(t, int32(1), closed.Load())
			if response == `{"success":true}` {
				require.Zero(t, manager.ActiveCount())
			} else {
				require.Equal(t, 1, manager.ActiveCount(), "unacknowledged cleanup must retain recovery ownership")
				require.Contains(t, err.Error(), "acknowledge")
			}
		})
	}
}

// [REQ:BAS-RH-J17] No recording mutation may bypass the Session that owns its lease.
func TestService_RecordingUsesOwnedSession(t *testing.T) {
	const owner = "a0d790ac-1ea8-41cf-8b60-82d02c4d087b"
	const lease = "record-lease"
	const recordingID = "9c45d4a0-5333-4b36-8197-bdba6b6c8f36"
	var effects atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/session/start" {
			_ = json.NewEncoder(w).Encode(map[string]any{"session_id": "record-session", "lease_id": lease})
			return
		}
		var envelope map[string]any
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Error(err)
		}
		if envelope["execution_id"] != owner || envelope["lease_id"] != lease {
			t.Errorf("lost session ownership: %v", envelope)
		}
		if r.URL.Path == "/session/record-session/record/start" {
			for field, suffix := range map[string]string{"callback_url": "action", "frame_callback_url": "frame", "page_callback_url": "page-event"} {
				if envelope[field] != "http://fixture.invalid:9999/api/v1/recordings/live/record-session/"+suffix {
					t.Errorf("lost callback %s: %v", field, envelope[field])
				}
			}
			if envelope["frame_quality"] != float64(73) || envelope["frame_fps"] != float64(12) {
				t.Errorf("lost controls: %v", envelope)
			}
		}
		effects.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"session_id": "record-session", "recording_id": recordingID, "action_count": 7, "started_at": "2026-09-22T12:00:00.123Z", "stopped_at": "2026-09-22T12:01:00.456Z"})
	}))
	defer srv.Close()
	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	manager := autosession.NewManagerWithClient(client)
	_, err = manager.Create(context.Background(), autosession.Spec{ExecutionID: uuid.MustParse(owner), Mode: autosession.ModeRecording})
	require.NoError(t, err)
	service := NewServiceWithManager(manager, logrus.New(), nil)
	cfg := &RecordingConfig{APIHost: "fixture.invalid", APIPort: "9999", FrameQuality: 73, FrameFPS: 12}
	start, err := service.StartRecording(context.Background(), "record-session", cfg)
	require.NoError(t, err)
	require.Equal(t, recordingID, start.RecordingID)
	stop, err := service.StopRecording(context.Background(), "record-session")
	require.NoError(t, err)
	require.Equal(t, recordingID, stop.RecordingID)
	require.Equal(t, "2026-09-22T12:01:00.456Z", stop.StoppedAt)
	require.NoError(t, service.ForwardInput(context.Background(), "record-session", []byte(`{"type":"pointer","action":"click","x":12}`)))
	require.Equal(t, int32(3), effects.Load())
	_, err = service.StartRecording(context.Background(), "unknown-session", cfg)
	require.Error(t, err)
	_, err = service.StopRecording(context.Background(), "unknown-session")
	require.Error(t, err)
	require.Error(t, service.ForwardInput(context.Background(), "unknown-session", []byte(`{"type":"pointer","action":"click"}`)))
	require.Equal(t, int32(3), effects.Load(), "unknown owner must not reach the driver")
}

// [REQ:BAS-RH-J17] Restoring the initial saved tab must use its owned Go Session.
func TestRestoreTabsInitialNavigationCarriesOwnership(t *testing.T) {
	owner := uuid.New()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session/start" {
			_, _ = w.Write([]byte(`{"session_id":"restored","lease_id":"saved-tab-lease"}`))
			return
		}
		calls.Add(1)
		require.Equal(t, "/session/restored/record/navigate", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, map[string]any{"execution_id": owner.String(), "lease_id": "saved-tab-lease", "url": "https://saved.test"}, body)
		_, _ = w.Write([]byte(`{"url":"https://saved.test","title":"saved tab"}`))
	}))
	defer server.Close()
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	manager := autosession.NewManagerWithClient(client)
	_, err = manager.Create(context.Background(), autosession.Spec{ExecutionID: owner, Mode: autosession.ModeRecording})
	require.NoError(t, err)
	service := NewServiceWithManager(manager, logrus.New(), nil)
	tabs := []sessionprofilepersistence.TabState{{URL: "https://saved.test", IsActive: true}}
	restored, err := service.RestoreTabs(context.Background(), "restored", tabs)
	require.NoError(t, err)
	require.Equal(t, "https://saved.test", restored.InitialURL)
	require.Equal(t, []HistoryEntryInfo{{URL: "https://saved.test", Title: "saved tab"}}, restored.HistoryEntries)
	_, err = service.RestoreTabs(context.Background(), "unknown", tabs)
	require.Error(t, err)
	_, err = service.RestoreTabs(context.Background(), "restored", []sessionprofilepersistence.TabState{{URL: "about:blank"}})
	require.NoError(t, err)
	require.Equal(t, int32(1), calls.Load())
}
