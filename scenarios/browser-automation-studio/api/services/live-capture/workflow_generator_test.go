package livecapture

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/browser-automation-studio/automation/compiler"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

func TestMergeConsecutiveActions_EmptySlice(t *testing.T) {
	result := MergeConsecutiveActions(nil)
	if result != nil {
		t.Errorf("Expected nil for nil input, got %v", result)
	}

	result = MergeConsecutiveActions([]driver.RecordedAction{})
	if len(result) != 0 {
		t.Errorf("Expected empty slice for empty input, got %v", result)
	}
}

func TestGenerateWorkflowWithPagesReplaysPopupTabAlternation(t *testing.T) {
	initialPageID := uuid.New()
	popupPageID := uuid.New()
	createdAt, err := time.Parse(time.RFC3339Nano, "2026-09-24T12:00:02Z")
	require.NoError(t, err)
	initialPage := &domain.Page{
		ID: initialPageID, DriverPageID: "main", URL: "https://fixture.invalid",
		IsInitial: true, Status: domain.PageStatusActive,
	}
	popupPage := &domain.Page{
		ID: popupPageID, DriverPageID: "popup", URL: "https://fixture.invalid",
		OpenerID: &initialPageID, CreatedAt: createdAt, Status: domain.PageStatusActive,
	}
	selector := &driver.SelectorSet{Primary: "#same"}
	recorded := []driver.RecordedAction{
		{ActionType: "click", DriverPageID: "main", URL: "https://fixture.invalid", Timestamp: "2026-09-24T12:00:01Z", Selector: selector},
		{ActionType: "click", DriverPageID: "popup", URL: "https://fixture.invalid", Timestamp: "2026-09-24T12:00:03Z", Selector: selector},
		{ActionType: "click", DriverPageID: "main", URL: "https://fixture.invalid", Timestamp: "2026-09-24T12:00:04Z", Selector: selector},
	}

	workflow, err := NewWorkflowGenerator().GenerateWorkflowWithPages(recorded, []*domain.Page{initialPage, popupPage})
	require.NoError(t, err)
	require.Len(t, workflow.Nodes, 7)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_CLICK, workflow.Nodes[0].Action.Type)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_TAB_SWITCH, workflow.Nodes[1].Action.Type)
	require.Equal(t, basactions.TabSwitchAction_TAB_SWITCH_ACTION_SWITCH, workflow.Nodes[1].Action.GetTabSwitch().Action)
	require.EqualValues(t, 1, workflow.Nodes[1].Action.GetTabSwitch().GetIndex())
	require.Equal(t, basactions.ActionType_ACTION_TYPE_WAIT, workflow.Nodes[2].Action.Type)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_CLICK, workflow.Nodes[3].Action.Type)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_TAB_SWITCH, workflow.Nodes[4].Action.Type)
	require.Equal(t, basactions.TabSwitchAction_TAB_SWITCH_ACTION_SWITCH, workflow.Nodes[4].Action.GetTabSwitch().Action)
	require.EqualValues(t, 0, workflow.Nodes[4].Action.GetTabSwitch().GetIndex())
	require.Equal(t, basactions.ActionType_ACTION_TYPE_WAIT, workflow.Nodes[5].Action.Type)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_CLICK, workflow.Nodes[6].Action.Type)
	_, instructions, err := compiler.CompileWorkflowToContracts(context.Background(), uuid.New(), &basapi.WorkflowSummary{
		Id: uuid.NewString(), FlowDefinition: workflow,
	})
	require.NoError(t, err)
	require.Len(t, instructions, len(workflow.Nodes))
	require.Equal(t, basactions.ActionType_ACTION_TYPE_TAB_SWITCH, instructions[1].Action.Type)
	require.EqualValues(t, 1, instructions[1].Action.GetTabSwitch().GetIndex())
}

func TestGenerateWorkflowWithPagesClosesPopupBetweenRecordedActions(t *testing.T) {
	mainID := uuid.New()
	popupID := uuid.New()
	createdAt, err := time.Parse(time.RFC3339Nano, "2026-09-24T12:00:02Z")
	require.NoError(t, err)
	closedAt, err := time.Parse(time.RFC3339Nano, "2026-09-24T12:00:04Z")
	require.NoError(t, err)
	mainPage := &domain.Page{
		ID: mainID, DriverPageID: "main", URL: "https://fixture.invalid",
		IsInitial: true, Status: domain.PageStatusActive,
	}
	popupPage := &domain.Page{
		ID: popupID, DriverPageID: "popup", URL: "https://fixture.invalid/popup",
		OpenerID: &mainID, CreatedAt: createdAt, ClosedAt: &closedAt, Status: domain.PageStatusClosed,
	}
	selector := &driver.SelectorSet{Primary: "#same"}
	recorded := []driver.RecordedAction{
		{ActionType: "click", DriverPageID: "main", URL: mainPage.URL, Timestamp: "2026-09-24T12:00:01Z", Selector: selector},
		{ActionType: "click", DriverPageID: "popup", URL: popupPage.URL, Timestamp: "2026-09-24T12:00:03Z", Selector: selector},
		{ActionType: "click", DriverPageID: "main", URL: mainPage.URL, Timestamp: "2026-09-24T12:00:05Z", Selector: selector},
	}
	workflow, err := NewWorkflowGenerator().GenerateWorkflowWithPages(recorded, []*domain.Page{mainPage, popupPage})
	require.NoError(t, err)
	require.Len(t, workflow.Nodes, 7)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_TAB_SWITCH, workflow.Nodes[4].Action.Type)
	require.Equal(t, basactions.TabSwitchAction_TAB_SWITCH_ACTION_CLOSE, workflow.Nodes[4].Action.GetTabSwitch().Action)
	require.EqualValues(t, 1, workflow.Nodes[4].Action.GetTabSwitch().GetIndex())
	require.Equal(t, basactions.ActionType_ACTION_TYPE_CLICK, workflow.Nodes[6].Action.Type)
	_, instructions, err := compiler.CompileWorkflowToContracts(context.Background(), uuid.New(), &basapi.WorkflowSummary{
		Id: uuid.NewString(), FlowDefinition: workflow,
	})
	require.NoError(t, err)
	require.Len(t, instructions, len(workflow.Nodes))
	require.Equal(t, basactions.TabSwitchAction_TAB_SWITCH_ACTION_CLOSE, instructions[4].Action.GetTabSwitch().Action)

	recorded = append(recorded, driver.RecordedAction{
		ActionType: "click", DriverPageID: "popup", URL: popupPage.URL,
		Timestamp: "2026-09-24T12:00:06Z", Selector: selector,
	})
	_, err = NewWorkflowGenerator().GenerateWorkflowWithPages(recorded, []*domain.Page{mainPage, popupPage})
	require.ErrorContains(t, err, "is not replayable")
}

func TestGenerateWorkflowWithPagesRejectsUnreplayableCloseOrdering(t *testing.T) {
	cases := []struct {
		name          string
		secondCreated string
		secondAction  string
		wantError     string
	}{
		{name: "close shares action time", secondCreated: "2026-09-24T12:00:02Z", secondAction: "2026-09-24T12:00:04Z", wantError: "is ambiguous"},
		{name: "action timestamp missing", secondCreated: "2026-09-24T12:00:02Z", secondAction: "", wantError: "is ambiguous"},
		{name: "close would remove last replay tab", secondCreated: "2026-09-24T12:00:05Z", secondAction: "2026-09-24T12:00:05Z", wantError: "last replay tab"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mainID, secondID := uuid.New(), uuid.New()
			closedAt, err := time.Parse(time.RFC3339Nano, "2026-09-24T12:00:04Z")
			require.NoError(t, err)
			createdAt, err := time.Parse(time.RFC3339Nano, tc.secondCreated)
			require.NoError(t, err)
			mainPage := &domain.Page{ID: mainID, DriverPageID: "main", IsInitial: true, Status: domain.PageStatusClosed, ClosedAt: &closedAt}
			secondPage := &domain.Page{ID: secondID, DriverPageID: "second", URL: "https://second.invalid", Status: domain.PageStatusActive, CreatedAt: createdAt}
			_, err = NewWorkflowGenerator().GenerateWorkflowWithPages([]driver.RecordedAction{
				{ActionType: "click", DriverPageID: "main", URL: "https://main.invalid", Timestamp: "2026-09-24T12:00:01Z", Selector: &driver.SelectorSet{Primary: "#same"}},
				{ActionType: "click", DriverPageID: "second", URL: secondPage.URL, Timestamp: tc.secondAction, Selector: &driver.SelectorSet{Primary: "#same"}},
			}, []*domain.Page{mainPage, secondPage})
			require.ErrorContains(t, err, tc.wantError)
		})
	}
}

func TestGenerateWorkflowWithPagesRejectsSimultaneousOpenPageCloses(t *testing.T) {
	mainID, popupID, thirdID := uuid.New(), uuid.New(), uuid.New()
	createdAt, err := time.Parse(time.RFC3339Nano, "2026-09-24T12:00:02Z")
	require.NoError(t, err)
	thirdCreatedAt, err := time.Parse(time.RFC3339Nano, "2026-09-24T12:00:03.500Z")
	require.NoError(t, err)
	closedAt, err := time.Parse(time.RFC3339Nano, "2026-09-24T12:00:04Z")
	require.NoError(t, err)
	mainPage := &domain.Page{ID: mainID, DriverPageID: "main", IsInitial: true, Status: domain.PageStatusClosed, ClosedAt: &closedAt}
	popupPage := &domain.Page{ID: popupID, DriverPageID: "popup", OpenerID: &mainID, CreatedAt: createdAt, Status: domain.PageStatusClosed, ClosedAt: &closedAt}
	thirdPage := &domain.Page{ID: thirdID, DriverPageID: "third", URL: "https://third.invalid", CreatedAt: thirdCreatedAt, Status: domain.PageStatusActive}
	_, err = NewWorkflowGenerator().GenerateWorkflowWithPages([]driver.RecordedAction{
		{ActionType: "click", DriverPageID: "main", Timestamp: "2026-09-24T12:00:01Z", Selector: &driver.SelectorSet{Primary: "#same"}},
		{ActionType: "click", DriverPageID: "popup", Timestamp: "2026-09-24T12:00:03Z", Selector: &driver.SelectorSet{Primary: "#same"}},
		{ActionType: "click", DriverPageID: "third", Timestamp: "2026-09-24T12:00:05Z", Selector: &driver.SelectorSet{Primary: "#same"}},
	}, []*domain.Page{mainPage, popupPage, thirdPage})
	require.ErrorContains(t, err, "share an ambiguous timestamp")
}

func TestGenerateWorkflowWithPagesOpensIndependentTabAtFirstUse(t *testing.T) {
	initialPageID := uuid.New()
	secondPageID := uuid.New()
	selector := &driver.SelectorSet{Primary: "#same"}
	workflow, err := NewWorkflowGenerator().GenerateWorkflowWithPages([]driver.RecordedAction{
		{ActionType: "click", DriverPageID: "main", URL: "https://fixture.invalid", Timestamp: "2026-09-24T12:00:01Z", Selector: selector},
		{ActionType: "click", DriverPageID: "second", URL: "https://second.invalid", Timestamp: "2026-09-24T12:00:02Z", Selector: selector},
	}, []*domain.Page{
		{ID: initialPageID, DriverPageID: "main", IsInitial: true, Status: domain.PageStatusActive},
		{ID: secondPageID, DriverPageID: "second", URL: "https://second.invalid", Status: domain.PageStatusActive},
	})
	require.NoError(t, err)
	require.Len(t, workflow.Nodes, 4)
	open := workflow.Nodes[1].Action.GetTabSwitch()
	require.NotNil(t, open)
	require.Equal(t, basactions.TabSwitchAction_TAB_SWITCH_ACTION_OPEN, open.Action)
	require.Equal(t, "https://second.invalid", open.GetUrl())
}

func TestGenerateWorkflowWithPagesBindsInitialNavigationToInitialPage(t *testing.T) {
	initialPageID := uuid.New()
	workflow, err := NewWorkflowGenerator().GenerateWorkflowWithPages([]driver.RecordedAction{
		{ActionType: "navigate", URL: "https://fixture.invalid", Timestamp: "2026-09-25T12:00:00Z"},
		{ActionType: "input", DriverPageID: "main", Timestamp: "2026-09-25T12:00:01Z", Selector: &driver.SelectorSet{Primary: "#input"}, Payload: map[string]any{"text": "intermediate value"}},
		{ActionType: "input", DriverPageID: "main", Timestamp: "2026-09-25T12:00:02Z", Selector: &driver.SelectorSet{Primary: "#input"}, Payload: map[string]any{"text": "final value"}},
	}, []*domain.Page{{ID: initialPageID, DriverPageID: "main", URL: "https://fixture.invalid", IsInitial: true, Status: domain.PageStatusActive}})
	require.NoError(t, err)
	require.Len(t, workflow.Nodes, 3)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_NAVIGATE, workflow.Nodes[0].Action.Type)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_WAIT, workflow.Nodes[1].Action.Type)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_INPUT, workflow.Nodes[2].Action.Type)
	require.Equal(t, "final value", workflow.Nodes[2].Action.GetInput().GetValue())
	require.True(t, workflow.Nodes[2].Action.GetInput().GetClearFirst())
}

func TestGenerateWorkflowWithPagesRejectsAmbiguousPopupTiming(t *testing.T) {
	initialPageID := uuid.New()
	popupPageID := uuid.New()
	oldCreation := time.Date(2026, time.September, 24, 11, 59, 0, 0, time.UTC)
	_, err := NewWorkflowGenerator().GenerateWorkflowWithPages([]driver.RecordedAction{
		{ActionType: "click", DriverPageID: "main", Timestamp: "2026-09-24T12:00:01Z", Selector: &driver.SelectorSet{Primary: "#open"}},
		{ActionType: "click", DriverPageID: "popup", Timestamp: "2026-09-24T12:00:03Z", Selector: &driver.SelectorSet{Primary: "#same"}},
	}, []*domain.Page{
		{ID: initialPageID, DriverPageID: "main", IsInitial: true, Status: domain.PageStatusActive},
		{ID: popupPageID, DriverPageID: "popup", OpenerID: &initialPageID, CreatedAt: oldCreation, Status: domain.PageStatusActive},
	})
	require.ErrorContains(t, err, "ambiguous opener timing")
}

func TestGenerateWorkflowWithPagesRejectsMissingActionIdentity(t *testing.T) {
	initialPageID := uuid.New()
	popupPageID := uuid.New()
	_, err := NewWorkflowGenerator().GenerateWorkflowWithPages([]driver.RecordedAction{
		{ActionType: "click", Selector: &driver.SelectorSet{Primary: "#same"}},
		{ActionType: "click", Selector: &driver.SelectorSet{Primary: "#same"}},
	}, []*domain.Page{
		{ID: initialPageID, DriverPageID: "main", IsInitial: true, Status: domain.PageStatusActive},
		{ID: popupPageID, DriverPageID: "popup", OpenerID: &initialPageID, Status: domain.PageStatusActive},
	})
	require.ErrorContains(t, err, "missing its logical page identity")
}

func TestMergeConsecutiveActions_SingleAction(t *testing.T) {
	actions := []driver.RecordedAction{
		{ActionType: "click", Selector: &driver.SelectorSet{Primary: "#btn"}},
	}
	result := MergeConsecutiveActions(actions)
	if len(result) != 1 {
		t.Errorf("Expected 1 action, got %d", len(result))
	}
}

func TestMergeConsecutiveActions_MergesConsecutiveTypeActions(t *testing.T) {
	selector := &driver.SelectorSet{Primary: "#input"}
	actions := []driver.RecordedAction{
		{ActionType: "type", Selector: selector, Payload: map[string]interface{}{"text": "Hello"}},
		{ActionType: "type", Selector: selector, Payload: map[string]interface{}{"text": "Hello "}},
		{ActionType: "type", Selector: selector, Payload: map[string]interface{}{"text": "Hello World"}},
	}

	result := MergeConsecutiveActions(actions)

	if len(result) != 1 {
		t.Fatalf("Expected 1 merged action, got %d", len(result))
	}
	if result[0].Payload["text"] != "Hello World" {
		t.Errorf("Expected merged text 'Hello World', got %v", result[0].Payload["text"])
	}
}

func TestMergeConsecutiveActions_DoesNotMergeTypeOnDifferentSelectors(t *testing.T) {
	actions := []driver.RecordedAction{
		{ActionType: "type", Selector: &driver.SelectorSet{Primary: "#input1"}, Payload: map[string]interface{}{"text": "First"}},
		{ActionType: "type", Selector: &driver.SelectorSet{Primary: "#input2"}, Payload: map[string]interface{}{"text": "Second"}},
	}

	result := MergeConsecutiveActions(actions)

	if len(result) != 2 {
		t.Errorf("Expected 2 actions (different selectors), got %d", len(result))
	}
}

func TestMergeConsecutiveActions_MergesConsecutiveScrollActions(t *testing.T) {
	actions := []driver.RecordedAction{
		{ActionType: "scroll", Payload: map[string]interface{}{"scrollY": 100.0}},
		{ActionType: "scroll", Payload: map[string]interface{}{"scrollY": 200.0}},
		{ActionType: "scroll", Payload: map[string]interface{}{"scrollY": 500.0}},
	}

	result := MergeConsecutiveActions(actions)

	if len(result) != 1 {
		t.Fatalf("Expected 1 merged scroll action, got %d", len(result))
	}
	if result[0].Payload["scrollY"] != 500.0 {
		t.Errorf("Expected final scrollY 500.0, got %v", result[0].Payload["scrollY"])
	}
}

func TestMergeConsecutiveActions_SkipsFocusBeforeType(t *testing.T) {
	selector := &driver.SelectorSet{Primary: "#input"}
	actions := []driver.RecordedAction{
		{ActionType: "focus", Selector: selector},
		{ActionType: "type", Selector: selector, Payload: map[string]interface{}{"text": "test"}},
	}

	result := MergeConsecutiveActions(actions)

	if len(result) != 1 {
		t.Fatalf("Expected 1 action (focus skipped), got %d", len(result))
	}
	if result[0].ActionType != "type" {
		t.Errorf("Expected type action, got %s", result[0].ActionType)
	}
}

func TestMergeConsecutiveActions_KeepsFocusWithoutFollowingType(t *testing.T) {
	selector := &driver.SelectorSet{Primary: "#input"}
	actions := []driver.RecordedAction{
		{ActionType: "focus", Selector: selector},
		{ActionType: "click", Selector: &driver.SelectorSet{Primary: "#btn"}},
	}

	result := MergeConsecutiveActions(actions)

	if len(result) != 2 {
		t.Errorf("Expected 2 actions (focus kept), got %d", len(result))
	}
}

func TestMergeConsecutiveActions_MixedActions(t *testing.T) {
	inputSelector := &driver.SelectorSet{Primary: "#input"}
	actions := []driver.RecordedAction{
		{ActionType: "click", Selector: &driver.SelectorSet{Primary: "#btn"}},
		{ActionType: "focus", Selector: inputSelector},
		{ActionType: "type", Selector: inputSelector, Payload: map[string]interface{}{"text": "Hello"}},
		{ActionType: "type", Selector: inputSelector, Payload: map[string]interface{}{"text": "Hello World"}},
		{ActionType: "scroll", Payload: map[string]interface{}{"scrollY": 100.0}},
		{ActionType: "scroll", Payload: map[string]interface{}{"scrollY": 300.0}},
		{ActionType: "click", Selector: &driver.SelectorSet{Primary: "#submit"}},
	}

	result := MergeConsecutiveActions(actions)

	// Expected: click, type (merged, focus skipped), scroll (merged), click
	if len(result) != 4 {
		t.Fatalf("Expected 4 actions after merging, got %d", len(result))
	}

	if result[0].ActionType != "click" {
		t.Errorf("Action 0: expected click, got %s", result[0].ActionType)
	}
	if result[1].ActionType != "type" || result[1].Payload["text"] != "Hello World" {
		t.Errorf("Action 1: expected merged type with 'Hello World', got %v", result[1])
	}
	if result[2].ActionType != "scroll" || result[2].Payload["scrollY"] != 300.0 {
		t.Errorf("Action 2: expected merged scroll with 300.0, got %v", result[2])
	}
	if result[3].ActionType != "click" {
		t.Errorf("Action 3: expected click, got %s", result[3].ActionType)
	}
}

func TestApplyActionRange_EmptySlice(t *testing.T) {
	result := ApplyActionRange(nil, 0, 5)
	if result != nil {
		t.Errorf("Expected nil for nil input, got %v", result)
	}

	result = ApplyActionRange([]driver.RecordedAction{}, 0, 5)
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %v", result)
	}
}

func TestApplyActionRange_ValidRange(t *testing.T) {
	actions := []driver.RecordedAction{
		{ActionType: "a"},
		{ActionType: "b"},
		{ActionType: "c"},
		{ActionType: "d"},
		{ActionType: "e"},
	}

	result := ApplyActionRange(actions, 1, 3)

	if len(result) != 3 {
		t.Fatalf("Expected 3 actions, got %d", len(result))
	}
	if result[0].ActionType != "b" || result[1].ActionType != "c" || result[2].ActionType != "d" {
		t.Errorf("Expected [b,c,d], got %v", result)
	}
}

func TestApplyActionRange_ClampsNegativeStart(t *testing.T) {
	actions := []driver.RecordedAction{
		{ActionType: "a"},
		{ActionType: "b"},
		{ActionType: "c"},
	}

	result := ApplyActionRange(actions, -5, 1)

	if len(result) != 2 {
		t.Fatalf("Expected 2 actions, got %d", len(result))
	}
	if result[0].ActionType != "a" {
		t.Errorf("Expected first action 'a', got %s", result[0].ActionType)
	}
}

func TestApplyActionRange_ClampsEndBeyondLength(t *testing.T) {
	actions := []driver.RecordedAction{
		{ActionType: "a"},
		{ActionType: "b"},
		{ActionType: "c"},
	}

	result := ApplyActionRange(actions, 1, 100)

	if len(result) != 2 {
		t.Fatalf("Expected 2 actions, got %d", len(result))
	}
	if result[0].ActionType != "b" || result[1].ActionType != "c" {
		t.Errorf("Expected [b,c], got %v", result)
	}
}

func TestApplyActionRange_FullRange(t *testing.T) {
	actions := []driver.RecordedAction{
		{ActionType: "a"},
		{ActionType: "b"},
		{ActionType: "c"},
	}

	result := ApplyActionRange(actions, 0, 2)

	if len(result) != 3 {
		t.Errorf("Expected all 3 actions, got %d", len(result))
	}
}

func TestGenerateWorkflow_CreatesNodesAndEdges(t *testing.T) {
	gen := NewWorkflowGenerator()
	actions := []driver.RecordedAction{
		{ActionType: "navigate", URL: "https://example.com"},
		{ActionType: "click", Selector: &driver.SelectorSet{Primary: "#btn"}},
	}

	result, err := gen.GenerateWorkflow(actions)
	require.NoError(t, err)
	nodes, edges := result.Nodes, result.Edges

	// Should have at least 2 action nodes (may have wait nodes inserted)
	if len(nodes) < 2 {
		t.Errorf("Expected at least 2 nodes, got %d", len(nodes))
	}

	// Should have at least 1 edge connecting them
	if len(edges) < 1 {
		t.Errorf("Expected at least 1 edge, got %d", len(edges))
	}
}

func TestGenerateWorkflow_EmptyActions(t *testing.T) {
	gen := NewWorkflowGenerator()
	result, err := gen.GenerateWorkflow(nil)
	require.NoError(t, err)
	require.Empty(t, result.Nodes)
	require.Empty(t, result.Edges)
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc..."},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestExtractHostname(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com", "https://example.com"},
		{"https://example.com/path", "https://example.com/path"},
		{"https://very-long-domain-name-that-exceeds-fifty-characters.example.com/path", "https://very-long-domain-name-that-exceeds-fifty-c..."},
	}

	for _, tt := range tests {
		result := extractHostname(tt.input)
		if result != tt.expected {
			t.Errorf("extractHostname(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestGenerateClickLabel(t *testing.T) {
	tests := []struct {
		name     string
		action   driver.RecordedAction
		expected string
	}{
		{
			name:     "no element meta",
			action:   driver.RecordedAction{ActionType: "click"},
			expected: "Click element",
		},
		{
			name: "with inner text",
			action: driver.RecordedAction{
				ActionType:  "click",
				ElementMeta: &driver.ElementMeta{InnerText: "Submit", TagName: "BUTTON"},
			},
			expected: "Click: Submit",
		},
		{
			name: "with aria label",
			action: driver.RecordedAction{
				ActionType:  "click",
				ElementMeta: &driver.ElementMeta{AriaLabel: "Close dialog", TagName: "BUTTON"},
			},
			expected: "Click: Close dialog",
		},
		{
			name: "with tag name only",
			action: driver.RecordedAction{
				ActionType:  "click",
				ElementMeta: &driver.ElementMeta{TagName: "BUTTON"},
			},
			expected: "Click BUTTON",
		},
		{
			name: "long inner text truncated",
			action: driver.RecordedAction{
				ActionType:  "click",
				ElementMeta: &driver.ElementMeta{InnerText: "This is a very long button text that should be truncated"},
			},
			expected: "Click: This is a very long ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateClickLabel(tt.action)
			if result != tt.expected {
				t.Errorf("generateClickLabel() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestMergeSnapshotsPreservesHistoryAndTarget(t *testing.T) {
	first := driver.RecordedAction{ActionType: "type", PageID: "one", FrameID: "main", Selector: &driver.SelectorSet{Primary: "#input"}, Payload: map[string]any{"text": "first"}}
	for _, value := range []string{"replacement", ""} {
		next := first
		next.Payload = map[string]any{"text": value}
		result := MergeConsecutiveActions([]driver.RecordedAction{first, next})
		require.Len(t, result, 1)
		require.Equal(t, value, result[0].Payload["text"])
		require.Equal(t, "first", first.Payload["text"])
	}
	for _, field := range []string{"page", "driver page", "frame", "frame path", "selector", "url", "submit"} {
		t.Run(field, func(t *testing.T) {
			next := first
			next.Payload = map[string]any{"text": "second"}
			previous := first
			switch field {
			case "page":
				next.PageID = "two"
			case "driver page":
				next.DriverPageID = "two"
			case "frame":
				next.FrameID = "child"
			case "frame path":
				previous.FrameID, next.FrameID = "frame", "frame"
				previous.FramePath = []string{"#left"}
				next.FramePath = []string{"#right"}
			case "selector":
				next.Selector = &driver.SelectorSet{Primary: "#other"}
			case "url":
				next.URL = "https://next.invalid"
			case "submit":
				previous.Payload = map[string]any{"text": "first", "submit": true}
			}
			require.Len(t, MergeConsecutiveActions([]driver.RecordedAction{previous, next}), 2)
		})
	}
}

func TestGenerateWorkflowSwitchesBetweenMainAndChildFrameForSameSelector(t *testing.T) {
	selector := &driver.SelectorSet{Primary: "#submit"}
	recorded := []driver.RecordedAction{
		{ActionType: "click", PageID: "page", DriverPageID: "driver-page", URL: "https://fixture.invalid", Selector: selector},
		{ActionType: "click", PageID: "page", DriverPageID: "driver-page", URL: "https://fixture.invalid", FrameID: "child", FramePath: []string{"#child"}, Selector: selector},
		{ActionType: "click", PageID: "page", DriverPageID: "driver-page", URL: "https://fixture.invalid", Selector: selector},
	}

	workflow, err := NewWorkflowGenerator().GenerateWorkflow(recorded)
	require.NoError(t, err)
	require.Len(t, workflow.Nodes, 7)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_CLICK, workflow.Nodes[0].Action.Type)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_FRAME_SWITCH, workflow.Nodes[1].Action.Type)
	require.Equal(t, basactions.FrameSwitchAction_FRAME_SWITCH_ACTION_ENTER, workflow.Nodes[1].Action.GetFrameSwitch().Action)
	require.Equal(t, "#child", workflow.Nodes[1].Action.GetFrameSwitch().GetSelector())
	require.Equal(t, basactions.ActionType_ACTION_TYPE_WAIT, workflow.Nodes[2].Action.Type)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_CLICK, workflow.Nodes[3].Action.Type)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_FRAME_SWITCH, workflow.Nodes[4].Action.Type)
	require.Equal(t, basactions.FrameSwitchAction_FRAME_SWITCH_ACTION_PARENT, workflow.Nodes[4].Action.GetFrameSwitch().Action)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_WAIT, workflow.Nodes[5].Action.Type)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_CLICK, workflow.Nodes[6].Action.Type)
}

func TestMergeScrollKeepsAxesAndSeparateTargets(t *testing.T) {
	first := driver.RecordedAction{ActionType: "scroll", PageID: "one", Selector: &driver.SelectorSet{Primary: "#pane"}, Payload: map[string]any{"scrollX": 10.0, "scrollY": 20.0}}
	next := first
	next.Payload = map[string]any{"scrollX": 30.0, "scrollY": 40.0}
	merged := MergeConsecutiveActions([]driver.RecordedAction{first, next})
	require.Len(t, merged, 1)
	require.Equal(t, next.Payload, merged[0].Payload)
	require.Equal(t, 10.0, first.Payload["scrollX"])
	next.PageID = "two"
	require.Len(t, MergeConsecutiveActions([]driver.RecordedAction{first, next}), 2)
	next.PageID = "one"
	next.Selector = &driver.SelectorSet{Primary: "#other"}
	require.Len(t, MergeConsecutiveActions([]driver.RecordedAction{first, next}), 2)
}

func TestMergeScrollDoesNotLosePartialAxisUpdates(t *testing.T) {
	first := driver.RecordedAction{ActionType: "scroll", Payload: map[string]any{"scrollX": 100.0, "scrollY": 200.0}}
	next := driver.RecordedAction{ActionType: "scroll", Payload: map[string]any{"scrollY": 300.0}}
	require.Len(t, MergeConsecutiveActions([]driver.RecordedAction{first, next}), 2)
	next.Payload = map[string]any{"deltaY": 10.0}
	require.Len(t, MergeConsecutiveActions([]driver.RecordedAction{next, next}), 2)
}
