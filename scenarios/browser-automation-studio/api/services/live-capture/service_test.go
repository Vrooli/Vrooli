package livecapture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	autosession "github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/services/recording"
	"github.com/vrooli/browser-automation-studio/services/recording/persistence"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/browser-automation-studio/automation/compiler"
	"github.com/vrooli/browser-automation-studio/domain"
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

func TestService_GenerateWorkflowUsesTrackedPageBindings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session/start":
			_, _ = w.Write([]byte(`{"session_id":"capture-tabs","lease_id":"capture-lease","active_page_id":"main-page"}`))
		case "/session/capture-tabs/close":
			_, _ = w.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	manager := autosession.NewManagerWithClient(client)
	owner, err := manager.Create(context.Background(), autosession.Spec{ExecutionID: uuid.New(), Mode: autosession.ModeRecording})
	require.NoError(t, err)
	defer func() { require.NoError(t, manager.Close(context.Background(), owner.ID())) }()
	owner.InitializePageTracking("https://fixture.invalid")
	pages := owner.Pages()
	initialID := pages.GetInitialPageID()
	require.Equal(t, "main-page", pages.GetDriverPageID(initialID))

	firstActionTime := time.Now().UTC().Add(time.Millisecond)
	popupCreatedAt := firstActionTime.Add(time.Millisecond)
	popupID := pages.AddPage(&domain.Page{
		DriverPageID: "popup-page", URL: "https://fixture.invalid/popup",
		OpenerID: &initialID, CreatedAt: popupCreatedAt, Status: domain.PageStatusActive,
	}).ID
	selector := &driver.SelectorSet{Primary: "#same"}
	formatTime := func(value time.Time) string { return value.Format(time.RFC3339Nano) }
	actions := []driver.RecordedAction{
		{ActionType: "click", DriverPageID: "main-page", Timestamp: formatTime(firstActionTime), Selector: selector},
		{ActionType: "click", DriverPageID: "popup-page", Timestamp: formatTime(popupCreatedAt.Add(time.Millisecond)), Selector: selector},
		{ActionType: "click", DriverPageID: "main-page", Timestamp: formatTime(popupCreatedAt.Add(2 * time.Millisecond)), Selector: selector},
	}

	service := NewServiceWithManager(manager, logrus.New(), nil)
	result, err := service.GenerateWorkflow(context.Background(), owner.ID(), &GenerateWorkflowConfig{Actions: actions})
	require.NoError(t, err)
	boundPopupID := pages.GetPageIDByDriverID("popup-page")
	require.NotNil(t, boundPopupID)
	require.Equal(t, popupID, *boundPopupID)
	require.Len(t, result.FlowDefinition.Nodes, 7)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_TAB_SWITCH, result.FlowDefinition.Nodes[1].Action.Type)
	require.EqualValues(t, 1, result.FlowDefinition.Nodes[1].Action.GetTabSwitch().GetIndex())
	require.Equal(t, basactions.ActionType_ACTION_TYPE_TAB_SWITCH, result.FlowDefinition.Nodes[4].Action.Type)
	require.EqualValues(t, 0, result.FlowDefinition.Nodes[4].Action.GetTabSwitch().GetIndex())
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
		"multiple driver pages":  {{ActionType: "click", DriverPageID: "driver-first", Selector: &driver.SelectorSet{Primary: "#same"}}, {ActionType: "click", DriverPageID: "driver-second", Selector: &driver.SelectorSet{Primary: "#same"}}},
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
			if envelope["routed_test_mode"] != true {
				t.Errorf("routed test mode was not propagated to callback owner: %v", envelope)
			}
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
		if r.URL.Path == "/session/record-session/record/input" {
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "applied_sequence": 1, "coalesced_count": 0})
			return
		}
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
	start, err := service.StartRecording(coredb.WithTestMode(context.Background()), "record-session", cfg)
	require.NoError(t, err)
	require.Equal(t, recordingID, start.RecordingID)
	stop, err := service.StopRecording(context.Background(), "record-session")
	require.NoError(t, err)
	require.Equal(t, recordingID, stop.RecordingID)
	require.Equal(t, "2026-09-22T12:01:00.456Z", stop.StoppedAt)
	receipt, err := service.ForwardInput(context.Background(), "record-session", []byte(`{"type":"pointer","action":"click","x":12}`))
	require.NoError(t, err)
	require.Equal(t, uint64(1), receipt.AppliedSequence)
	require.Equal(t, int32(3), effects.Load())
	_, err = service.StartRecording(context.Background(), "unknown-session", cfg)
	require.Error(t, err)
	_, err = service.StopRecording(context.Background(), "unknown-session")
	require.Error(t, err)
	_, err = service.ForwardInput(context.Background(), "unknown-session", []byte(`{"type":"pointer","action":"click"}`))
	require.Error(t, err)
	require.Equal(t, int32(3), effects.Load(), "unknown owner must not reach the driver")
}

// [REQ:BAS-RH-J17] Restoring the initial saved tab must use its owned Go Session.
func TestRestoreTabsInitialNavigationCarriesOwnership(t *testing.T) {
	owner := uuid.New()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session/start" {
			_, _ = w.Write([]byte(`{"session_id":"restored","lease_id":"saved-tab-lease","active_page_id":"initial-driver-page"}`))
			return
		}
		calls.Add(1)
		require.Equal(t, "/session/restored/record/navigate", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, map[string]any{"execution_id": owner.String(), "lease_id": "saved-tab-lease", "url": "https://saved.test"}, body)
		_, _ = w.Write([]byte(`{"driver_page_id":"initial-driver-page","url":"https://saved.test","title":"saved tab"}`))
	}))
	defer server.Close()
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	manager := autosession.NewManagerWithClient(client)
	restoredSession, err := manager.Create(context.Background(), autosession.Spec{ExecutionID: owner, Mode: autosession.ModeRecording})
	require.NoError(t, err)
	restoredSession.InitializePageTracking("about:blank")
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

// [REQ:BAS-RH-J03] Tab receipts establish identity before recording callbacks exist.
func TestCreatePageRegistersReceiptBeforeRecording(t *testing.T) {
	for _, callbackFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(callbackFirst), func(t *testing.T) {
			executionID := uuid.New()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/session/start" {
					_, _ = w.Write([]byte(`{"session_id":"tabs","lease_id":"lease","active_page_id":"initial-driver"}`))
					return
				}
				require.Equal(t, "/session/tabs/record/new-page", r.URL.Path)
				var body map[string]string
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, map[string]string{"url": "https://second.test", "execution_id": executionID.String(), "lease_id": "lease"}, body)
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"driver_page_id":"second-driver","url":"https://second.test","title":"second","favicon_url":"https://second.test/custom.svg"}`))
			}))
			defer server.Close()
			client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
			require.NoError(t, err)
			manager := autosession.NewManagerWithClient(client)
			owner, err := manager.Create(context.Background(), autosession.Spec{ExecutionID: executionID, Mode: autosession.ModeRecording})
			require.NoError(t, err)
			owner.InitializePageTracking("https://original.test")
			pages := owner.Pages()
			initial := pages.GetInitialPageID()
			require.Equal(t, "initial-driver", pages.GetDriverPageID(initial))
			var callbackID uuid.UUID
			if callbackFirst {
				callbackID = pages.AddPage(&domain.Page{DriverPageID: "second-driver", URL: "about:blank"}).ID
			}
			service := NewServiceWithManager(manager, logrus.New(), nil)
			receipt, err := service.CreatePage(context.Background(), owner.ID(), "https://second.test")
			require.NoError(t, err)
			require.Equal(t, "second-driver", receipt.DriverPageID)
			require.Equal(t, "https://second.test/custom.svg", receipt.FaviconURL)
			require.Equal(t, 2, pages.PageCount())
			second := pages.GetPageIDByDriverID("second-driver")
			require.NotNil(t, second)
			require.Equal(t, *second, pages.GetActivePageID())
			if callbackFirst {
				require.Equal(t, callbackID, *second)
			}
			original, ok := pages.GetPage(initial)
			require.True(t, ok)
			require.Equal(t, "https://original.test", original.URL)
			require.Equal(t, "https://second.test", pages.GetActivePage().URL)
		})
	}
}

// [REQ:BAS-RH-J01] [REQ:BAS-RH-J03] Browser and API agree on restored locations and selected tab.
func TestRestoreTabsPreservesLocationsAndSelection(t *testing.T) {
	for _, tc := range []struct {
		name           string
		active         int
		failSwitch     bool
		failNavigation string
	}{
		{name: "first", active: 0},
		{name: "middle", active: 1},
		{name: "last", active: 2},
		{name: "no saved selection", active: -1},
		{name: "failed switch", active: 0, failSwitch: true},
		{name: "failed initial navigation", failNavigation: "initial"},
		{name: "failed additional navigation", failNavigation: "additional"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actualURLs := []string{"https://restored.test/one", "https://restored.test/two", "https://restored.test/three"}
			titles := []string{"current one", "current two", "current three"}
			var active atomic.Value
			active.Store("page-0")
			var created, switches atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/session/start":
					_ = json.NewEncoder(w).Encode(map[string]string{"session_id": "restore", "lease_id": "lease", "active_page_id": "page-0"})
				case "/session/restore/record/navigate":
					if tc.failNavigation == "initial" {
						http.Error(w, "controlled initial failure", 503)
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]string{"driver_page_id": "page-0", "url": actualURLs[0], "title": titles[0]})
				case "/session/restore/record/new-page":
					i := int(created.Add(1))
					if tc.failNavigation == "additional" {
						http.Error(w, "controlled additional failure", 503)
						return
					}
					id := fmt.Sprintf("page-%d", i)
					active.Store(id)
					w.WriteHeader(http.StatusCreated)
					_ = json.NewEncoder(w).Encode(map[string]string{"driver_page_id": id, "url": actualURLs[i], "title": titles[i]})
				case "/session/restore/record/active-page":
					switches.Add(1)
					if tc.failSwitch {
						http.Error(w, "controlled switch failure", http.StatusServiceUnavailable)
						return
					}
					var body map[string]string
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					active.Store(body["page_id"])
					_, _ = w.Write([]byte(`{}`))
				default:
					t.Errorf("unexpected request %s", r.URL.Path)
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
			require.NoError(t, err)
			manager := autosession.NewManagerWithClient(client)
			owner, err := manager.Create(context.Background(), autosession.Spec{ExecutionID: uuid.New(), Mode: autosession.ModeRecording})
			require.NoError(t, err)
			owner.InitializePageTracking("about:blank")
			service := NewServiceWithManager(manager, logrus.New(), nil)
			saved := make([]sessionprofilepersistence.TabState, 3)
			for i := range saved {
				saved[i] = sessionprofilepersistence.TabState{URL: fmt.Sprintf("https://saved.test/%d", i), Title: "stale saved title", IsActive: i == tc.active, Order: i}
			}
			restored, err := service.RestoreTabs(context.Background(), owner.ID(), saved)
			if tc.failNavigation != "" {
				require.ErrorContains(t, err, "restore")
				require.NotNil(t, restored)
				require.Equal(t, 1, owner.Pages().PageCount())
				require.Zero(t, switches.Load())
				if tc.failNavigation == "initial" {
					require.Zero(t, created.Load())
				} else {
					require.Equal(t, int32(1), created.Load())
				}
				return
			}
			if tc.failSwitch {
				require.ErrorContains(t, err, "switch")
				require.NotNil(t, restored)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, 3, owner.Pages().PageCount())
			for i := range saved {
				pageID := owner.Pages().GetPageIDByDriverID(fmt.Sprintf("page-%d", i))
				require.NotNil(t, pageID)
				page, ok := owner.Pages().GetPage(*pageID)
				require.True(t, ok)
				assert.Equal(t, actualURLs[i], page.URL)
				assert.Equal(t, titles[i], page.Title)
				assert.Equal(t, HistoryEntryInfo{URL: actualURLs[i], Title: titles[i]}, restored.HistoryEntries[i])
			}
			expected := tc.active
			if expected < 0 || tc.failSwitch {
				expected = 2
			}
			require.Equal(t, fmt.Sprintf("page-%d", expected), active.Load())
			require.Equal(t, fmt.Sprintf("page-%d", expected), owner.Pages().GetDriverPageID(owner.Pages().GetActivePageID()))
			if tc.active == 2 || tc.active < 0 {
				require.Zero(t, switches.Load(), "already selected page needs no browser command")
			}
			require.Equal(t, actualURLs[0], restored.InitialURL)
			require.Equal(t, titles[0], restored.InitialTitle)
			require.Len(t, restored.Tabs, 2)
			for i, tab := range restored.Tabs {
				assert.Equal(t, actualURLs[i+1], tab.URL)
				assert.Equal(t, titles[i+1], tab.Title)
			}
		})
	}
}

// [REQ:BAS-RH-J03] [REQ:BAS-RH-J17] Completion cannot rebind a tab transaction to a replacement Session.
func TestTabTransactionKeepsAdmittedSession(t *testing.T) {
	for _, operation := range []string{"activate", "restore", "close"} {
		t.Run(operation, func(t *testing.T) {
			var manager *autosession.Manager
			var starts, additional atomic.Int32
			var replacement atomic.Pointer[autosession.Session]
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/session/start" {
					n := starts.Add(1)
					_ = json.NewEncoder(w).Encode(map[string]string{"session_id": "same", "lease_id": fmt.Sprintf("lease-%d", n), "active_page_id": fmt.Sprintf("page-%d", n)})
					return
				}
				if r.URL.Path == "/session/same/record/new-page" {
					additional.Add(1)
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"driver_page_id":"late-page","url":"https://late.test"}`))
					return
				}
				if replacement.Load() == nil {
					fresh, err := manager.Create(context.Background(), autosession.Spec{ExecutionID: uuid.New(), Mode: autosession.ModeRecording})
					require.NoError(t, err)
					fresh.InitializePageTracking("https://replacement.test")
					replacement.Store(fresh)
				}
				if r.URL.Path == "/session/same/record/navigate" {
					_, _ = w.Write([]byte(`{"driver_page_id":"page-1","url":"https://restored.test","title":"Restored"}`))
				} else if r.URL.Path == "/session/same/record/close-page" {
					_, _ = w.Write([]byte(`{"closed_page_id":"page-1","active_page_id":""}`))
				} else {
					_, _ = w.Write([]byte(`{}`))
				}
			}))
			defer server.Close()
			client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
			require.NoError(t, err)
			manager = autosession.NewManagerWithClient(client)
			owner, err := manager.Create(context.Background(), autosession.Spec{ExecutionID: uuid.New(), Mode: autosession.ModeRecording})
			require.NoError(t, err)
			owner.InitializePageTracking("https://original.test")
			initial := owner.Pages().GetInitialPageID()
			service := NewServiceWithManager(manager, logrus.New(), nil)
			if operation == "activate" {
				target := owner.Pages().AddPage(&domain.Page{DriverPageID: "target", URL: "https://target.test"})
				err = service.ActivatePage(context.Background(), owner.ID(), target.ID)
			} else if operation == "close" {
				_, err = service.ClosePage(context.Background(), owner.ID(), initial)
			} else {
				_, err = service.RestoreTabs(context.Background(), owner.ID(), []sessionprofilepersistence.TabState{{URL: "https://saved.test", IsActive: true}, {URL: "https://another.test"}})
			}
			assert.ErrorContains(t, err, "ownership changed")
			assert.Zero(t, additional.Load(), "restoration must not create pages under the replacement owner")
			assert.Equal(t, initial, owner.Pages().GetActivePageID())
			assert.Equal(t, "https://original.test", owner.Pages().GetActivePage().URL)
			fresh := replacement.Load()
			require.NotNil(t, fresh)
			assert.Equal(t, 1, fresh.Pages().PageCount())
			assert.Equal(t, "https://replacement.test", fresh.Pages().GetActivePage().URL)
		})
	}
}

// [REQ:BAS-RH-J01] [REQ:BAS-RH-J03] Admission reflects actual navigation or preserves recoverable failure.
func TestCreateSessionInitialNavigationReceipt(t *testing.T) {
	for _, fault := range []string{"redirect", "blank", "navigation", "wrong receipt", "cleanup", "cancelled"} {
		t.Run(fault, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var closed, navigated atomic.Int32
			var execution atomic.Value
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/session/start":
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					execution.Store(body["execution_id"].(string))
					_, _ = w.Write([]byte(`{"session_id":"initial-session","lease_id":"initial-lease","active_page_id":"initial-page"}`))
				case "/session/initial-session/record/navigate":
					navigated.Add(1)
					if fault == "cancelled" {
						cancel()
					}
					if fault == "navigation" || fault == "cleanup" || fault == "cancelled" {
						http.Error(w, "controlled navigation failure", http.StatusServiceUnavailable)
						return
					}
					pageID := "initial-page"
					if fault == "wrong receipt" {
						pageID = "unregistered-page"
					}
					_ = json.NewEncoder(w).Encode(map[string]string{"driver_page_id": pageID, "url": "https://actual.test/final", "title": "Actual title", "favicon_url": "https://actual.test/icon.svg"})
				case "/session/initial-session/close":
					closed.Add(1)
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					assert.Equal(t, execution.Load(), body["execution_id"])
					assert.Equal(t, "initial-lease", body["lease_id"])
					if fault == "cleanup" {
						http.Error(w, "controlled cleanup failure", 503)
						return
					}
					_, _ = w.Write([]byte(`{"success":true}`))
				default:
					t.Errorf("unexpected driver request %s", r.URL.Path)
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
			require.NoError(t, err)
			manager := autosession.NewManagerWithClient(client)
			repo := persistence.NewMockRepository()
			service := NewServiceWithManager(manager, logrus.New(), recording.NewService(repo, recording.ServiceConfig{}))
			cfg := &SessionConfig{InitialURL: "https://requested.test/redirect"}
			if fault == "blank" {
				cfg.InitialURL = ""
			}
			result, err := service.CreateSession(ctx, cfg)
			if fault == "redirect" || fault == "blank" {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.NotNil(t, result.Close)
				assert.Zero(t, closed.Load())
				if fault == "blank" {
					assert.Zero(t, navigated.Load())
					return
				}
				owner, ok := manager.Get(result.SessionID)
				require.True(t, ok)
				assert.Equal(t, "https://actual.test/final", owner.Pages().GetActivePage().URL)
				assert.Equal(t, "Actual title", owner.Pages().GetActivePage().Title)
				assert.Equal(t, "https://actual.test/icon.svg", owner.Pages().GetActivePage().FaviconURL)
				assert.Equal(t, &HistoryEntryInfo{URL: "https://actual.test/final", Title: "Actual title"}, result.InitialNavigation)
				return
			}
			assert.ErrorContains(t, err, "initial")
			assert.Nil(t, result)
			assert.Equal(t, int32(1), closed.Load())
			if fault == "cleanup" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "controlled navigation failure")
				assert.Contains(t, err.Error(), "controlled cleanup failure")
				assert.Equal(t, 1, manager.ActiveCount(), "failed cleanup retains retry ownership")
			} else {
				assert.Zero(t, manager.ActiveCount())
			}
		})
	}
}

// [REQ:BAS-RH-J01] Profile and API reads capture detached pages and selection together.
func TestPageReadSnapshots(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/session/start", r.URL.Path)
		_, _ = w.Write([]byte(`{"session_id":"snapshot","lease_id":"lease","active_page_id":"initial-driver"}`))
	}))
	defer server.Close()
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	manager := autosession.NewManagerWithClient(client)
	owner, err := manager.Create(context.Background(), autosession.Spec{ExecutionID: uuid.New(), Mode: autosession.ModeRecording})
	require.NoError(t, err)
	owner.InitializePageTracking("https://before.test")
	svc := NewServiceWithManager(manager, logrus.New(), nil)
	tracker := owner.Pages()
	initial := tracker.GetInitialPageID()
	apiReceipt, err := svc.GetPages("snapshot")
	require.NoError(t, err)
	profileReceipt, selected, err := svc.GetOpenPages("snapshot")
	require.NoError(t, err)
	assert.Equal(t, initial, selected)
	tracker.UpdatePageInfo(initial, "https://after.test", "After", nil)
	assert.Equal(t, "https://before.test", apiReceipt.Pages[0].URL)
	assert.Equal(t, "https://before.test", profileReceipt[0].URL)

	// Every writer transition retains at least one selected open page. A read
	// spanning two transitions must not combine the list and selection from each.
	start, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		<-start
		previous := initial
		for i := 0; i < 300; i++ {
			next := tracker.AddPage(&domain.Page{DriverPageID: fmt.Sprintf("page-%d", i)})
			assert.NoError(t, tracker.SetActivePage(next.ID))
			_, closeErr := tracker.ClosePage(previous)
			assert.NoError(t, closeErr)
			previous = next.ID
		}
	}()
	close(start)
	for i := 0; i < 300; i++ {
		pages, active, err := svc.GetOpenPages("snapshot")
		require.NoError(t, err)
		matched := false
		for _, page := range pages {
			assert.Equal(t, domain.PageStatusActive, page.Status)
			matched = matched || page.ID == active
		}
		assert.True(t, matched, "selected page must belong to the same open-page snapshot")
	}
	<-done
	_, closeErr := tracker.ClosePage(tracker.GetActivePageID())
	require.NoError(t, closeErr)
	pages, active, err := svc.GetOpenPages("snapshot")
	require.NoError(t, err)
	assert.Empty(t, pages)
	assert.Equal(t, uuid.Nil, active)
	apiReceipt, err = svc.GetPages("snapshot")
	require.NoError(t, err)
	assert.Len(t, apiReceipt.Pages, 301)
	assert.Empty(t, apiReceipt.ActivePageID)
}
