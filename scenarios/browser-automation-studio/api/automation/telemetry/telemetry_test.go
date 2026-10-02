package telemetry

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

// [REQ:BAS-RH-J24] Retained event context must preserve the branch decision.
func TestConditionalOutcomeSurvivesTimelineConversion(t *testing.T) {
	outcome := contracts.StepOutcome{StepType: "conditional", Success: true, Condition: &contracts.ConditionOutcome{
		Type: "expression", Expression: "return false;", Outcome: true, Negated: true, Actual: false, Expected: true,
	}}
	entry := BuildExecutionTimelineEntry(outcome, uuid.New())
	b, err := protojson.Marshal(entry)
	require.NoError(t, err)
	var wire map[string]any
	require.NoError(t, json.Unmarshal(b, &wire))
	context, ok := wire["context"].(map[string]any)
	require.True(t, ok)
	condition, ok := context["condition"].(map[string]any)
	require.True(t, ok, "condition evidence disappeared from the retained entry")
	assert.Equal(t, "return false;", condition["expression"])
	assert.Equal(t, true, condition["negated"])
	assert.Equal(t, true, condition["outcome"])
	assert.Equal(t, map[string]any{"boolValue": false}, condition["actual"])
	assert.Equal(t, map[string]any{"boolValue": true}, condition["expected"])
}

func TestRecordingTimelineEntryPreservesRecordingContext(t *testing.T) {
	action := &driver.RecordedAction{
		ID: "recorded-1", SessionID: "session-7", SequenceNum: 3,
		Timestamp: "2026-10-01T08:00:00Z", DurationMs: 15, ActionType: "navigate",
		Confidence: 0.91, URL: "https://example.test", FrameID: "main",
		Payload:     map[string]any{"targetUrl": "https://target.test"},
		Selector:    &driver.SelectorSet{Primary: "#go", Candidates: []driver.SelectorCandidate{{Type: "css", Value: "#go", Confidence: 0.91, Specificity: 8}}},
		ElementMeta: &driver.ElementMeta{TagName: "button", ID: "go", AriaLabel: "Go"},
		BoundingBox: &contracts.BoundingBox{X: 1, Y: 2, Width: 3, Height: 4},
		CursorPos:   &contracts.Point{X: 2, Y: 3},
	}
	entry := BuildRecordingTimelineEntry(action)
	require.NotNil(t, entry)
	assert.Equal(t, "recorded-1", entry.Id)
	assert.Equal(t, int32(3), entry.SequenceNum)
	assert.Equal(t, "https://target.test", entry.Action.GetNavigate().Url)
	assert.Equal(t, "Navigate: Go", *entry.Action.Metadata.Label)
	assert.Equal(t, "button", entry.Action.Metadata.ElementSnapshot.TagName)
	require.Len(t, entry.Action.Metadata.SelectorCandidates, 1)
	assert.Equal(t, "#go", entry.Action.Metadata.SelectorCandidates[0].Value)
	assert.Equal(t, "https://example.test", entry.Telemetry.Url)
	assert.Equal(t, "main", *entry.Telemetry.FrameId)
	assert.Equal(t, float64(1), entry.Telemetry.ElementBoundingBox.X)
	assert.Equal(t, float64(2), entry.Telemetry.CursorPosition.X)
	assert.Equal(t, "session-7", entry.Context.GetSessionId())
	assert.True(t, *entry.Context.Success)
	assert.Equal(t, basbase.RecordingSource_RECORDING_SOURCE_AUTO, *entry.Context.Source)
	assert.False(t, *entry.Context.NeedsConfirmation)
}

func TestRecordingTimelineEntryNormalizesSelectionAndConfirmation(t *testing.T) {
	entry := BuildRecordingTimelineEntry(&driver.RecordedAction{
		ID: "select-1", SessionID: "session-7", SequenceNum: 1,
		ActionType: "select", Confidence: 0.5,
		Payload: map[string]any{"selectedIndex": 2},
		Selector: &driver.SelectorSet{Primary: "#choice", Candidates: []driver.SelectorCandidate{
			{Type: "css", Value: "#choice", Confidence: 0.5},
			{Type: "xpath", Value: "//select", Confidence: 0.4},
		}},
	})
	require.NotNil(t, entry.Action.GetSelectOption())
	assert.Equal(t, int32(2), entry.Action.GetSelectOption().GetIndex())
	assert.True(t, *entry.Context.NeedsConfirmation)
}

func TestExecutionTimelineEntryPreservesExecutionContext(t *testing.T) {
	executionID := uuid.New()
	started := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	entry := BuildExecutionTimelineEntry(contracts.StepOutcome{
		StepIndex: 4, Attempt: 2, NodeID: "node-4", StepType: "click", Success: false,
		StartedAt: started, DurationMs: 37, FinalURL: "https://example.test/after",
		SelectorConfidence: 0.75,
		ElementSnapshot:    &basdomain.ElementMeta{TagName: "button"},
		ElementBoundingBox: &basbase.BoundingBox{X: 2, Y: 3, Width: 4, Height: 5},
		ClickPosition:      &basbase.Point{X: 3, Y: 4},
		Screenshot:         &contracts.Screenshot{MediaType: "image/png", Width: 640, Height: 480},
		Assertion:          &contracts.AssertionOutcome{Mode: "visible", Selector: "#save", Success: true, Message: "visible"},
		ExtractedData:      map[string]any{"price": 12},
		Failure:            &contracts.StepFailure{Code: "CLICK_FAILED", Message: "click rejected", Retryable: true},
		Condition:          &contracts.ConditionOutcome{Type: "expression", Expression: "ready", Outcome: true},
	}, executionID)

	require.Equal(t, executionID.String()+"-step-4-attempt-2", entry.Id)
	require.Equal(t, int32(4), entry.SequenceNum)
	require.Equal(t, started, entry.Timestamp.AsTime())
	require.Equal(t, "node-4", *entry.NodeId)
	require.NotNil(t, entry.Action.Metadata.Confidence)
	assert.Equal(t, 0.75, *entry.Action.Metadata.Confidence)
	assert.Equal(t, "button", entry.Action.Metadata.ElementSnapshot.TagName)
	assert.Equal(t, float64(2), entry.Telemetry.ElementBoundingBox.X)
	assert.Equal(t, "image/png", entry.Telemetry.Screenshot.ContentType)
	assert.Equal(t, int32(640), entry.Telemetry.Screenshot.Width)
	assert.Equal(t, "https://example.test/after", entry.Telemetry.Url)
	require.NotNil(t, entry.Context.Origin)
	assert.Equal(t, executionID.String(), entry.Context.GetExecutionId())
	require.NotNil(t, entry.Context.RetryStatus)
	assert.Equal(t, int32(2), entry.Context.RetryStatus.CurrentAttempt)
	assert.Equal(t, "CLICK_FAILED", *entry.Context.ErrorCode)
	assert.Equal(t, "click rejected", *entry.Context.Error)
	require.NotNil(t, entry.Context.Assertion)
	assert.Equal(t, "#save", entry.Context.Assertion.Selector)
	assert.Equal(t, "visible", *entry.Context.Assertion.Message)
	assert.NotEmpty(t, entry.Context.ExtractedData)
	require.NotNil(t, entry.Context.Condition)
	assert.Equal(t, "ready", *entry.Context.Condition.Expression)
}
