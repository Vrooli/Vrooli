package telemetry

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
)

func TestRecordedActionToTelemetry(t *testing.T) {
	t.Run("nil action returns nil", func(t *testing.T) {
		result := RecordedActionToTelemetry(nil)
		assert.Nil(t, result)
	})

	t.Run("basic recording action converts correctly", func(t *testing.T) {
		action := &driver.RecordedAction{
			ID:          "action-1",
			SessionID:   "session-1",
			SequenceNum: 1,
			Timestamp:   "2024-01-15T10:00:00Z",
			DurationMs:  100,
			ActionType:  "click",
			Confidence:  0.95,
			URL:         "https://example.com",
			FrameID:     "main",
			Selector: &driver.SelectorSet{
				Primary: "#submit-btn",
				Candidates: []driver.SelectorCandidate{
					{Type: "css", Value: "#submit-btn", Confidence: 0.95, Specificity: 100},
				},
			},
			ElementMeta: &driver.ElementMeta{
				TagName:   "button",
				ID:        "submit-btn",
				InnerText: "Submit",
				IsVisible: true,
				IsEnabled: true,
				AriaLabel: "Submit form",
			},
			BoundingBox: &contracts.BoundingBox{X: 100, Y: 200, Width: 80, Height: 30},
			CursorPos:   &contracts.Point{X: 140, Y: 215},
		}

		result := RecordedActionToTelemetry(action)

		require.NotNil(t, result)
		assert.Equal(t, "action-1", result.ID)
		assert.Equal(t, 1, result.SequenceNum)
		assert.Equal(t, basactions.ActionType_ACTION_TYPE_CLICK, result.ActionType)
		assert.Equal(t, "#submit-btn", result.Selector)
		assert.Equal(t, 0.95, result.SelectorConfidence)
		assert.Equal(t, "https://example.com", result.URL)
		assert.Equal(t, "main", result.FrameID)
		assert.True(t, result.Success) // Recording always succeeds
		assert.Equal(t, 100, result.DurationMs)

		// Check element snapshot
		require.NotNil(t, result.ElementSnapshot)
		assert.Equal(t, "button", result.ElementSnapshot.TagName)
		assert.Equal(t, "submit-btn", result.ElementSnapshot.Id)
		assert.Equal(t, "Submit form", result.ElementSnapshot.AriaLabel)

		// Check bounding box
		require.NotNil(t, result.BoundingBox)
		assert.Equal(t, float64(100), result.BoundingBox.X)
		assert.Equal(t, float64(200), result.BoundingBox.Y)

		// Check cursor position
		require.NotNil(t, result.CursorPosition)
		assert.Equal(t, float64(140), result.CursorPosition.X)
		assert.Equal(t, float64(215), result.CursorPosition.Y)

		// Check origin is recording
		require.NotNil(t, result.Origin)
		origin, ok := result.Origin.(*RecordingOrigin)
		require.True(t, ok)
		assert.Equal(t, "session-1", origin.SessionID)
		assert.Equal(t, 0.95, origin.Confidence)
		assert.Len(t, origin.SelectorCandidates, 1)
		assert.False(t, origin.NeedsConfirmation) // High confidence
	})

	t.Run("low confidence triggers needs confirmation", func(t *testing.T) {
		action := &driver.RecordedAction{
			ID:         "action-2",
			ActionType: "click",
			Confidence: 0.5, // Low confidence
			Selector: &driver.SelectorSet{
				Primary: "#btn",
				Candidates: []driver.SelectorCandidate{
					{Type: "css", Value: "#btn", Confidence: 0.5},
					{Type: "xpath", Value: "//button", Confidence: 0.4},
				},
			},
		}

		result := RecordedActionToTelemetry(action)

		require.NotNil(t, result)
		origin, ok := result.Origin.(*RecordingOrigin)
		require.True(t, ok)
		assert.True(t, origin.NeedsConfirmation) // Should be true due to low confidence
	})

	t.Run("navigate action normalizes URL from payload", func(t *testing.T) {
		action := &driver.RecordedAction{
			ID:         "action-3",
			ActionType: "navigate",
			URL:        "https://old.com",
			Payload: map[string]interface{}{
				"targetUrl": "https://new.com",
			},
		}

		result := RecordedActionToTelemetry(action)

		require.NotNil(t, result)
		// The params should have the targetUrl normalized to url
		url, ok := result.Params["url"].(string)
		require.True(t, ok)
		assert.Equal(t, "https://new.com", url)
	})

	t.Run("label generation from element meta", func(t *testing.T) {
		tests := []struct {
			name     string
			meta     *driver.ElementMeta
			expected string
		}{
			{
				name: "uses aria-label first",
				meta: &driver.ElementMeta{
					TagName:   "button",
					AriaLabel: "Close dialog",
					InnerText: "X",
				},
				expected: "Click: Close dialog",
			},
			{
				name: "falls back to inner text",
				meta: &driver.ElementMeta{
					TagName:   "button",
					InnerText: "Submit Form",
				},
				expected: "Click: \"Submit Form\"",
			},
			{
				name: "falls back to ID",
				meta: &driver.ElementMeta{
					TagName: "div",
					ID:      "main-container",
				},
				expected: "Click: #main-container",
			},
			{
				name: "falls back to tag name",
				meta: &driver.ElementMeta{
					TagName: "span",
				},
				expected: "Click: span",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				action := &driver.RecordedAction{
					ID:          "action-label-test",
					ActionType:  "click",
					ElementMeta: tt.meta,
				}

				result := RecordedActionToTelemetry(action)

				require.NotNil(t, result)
				assert.Equal(t, tt.expected, result.Label)
			})
		}
	})
}

func TestStepOutcomeToTelemetry(t *testing.T) {
	executionID := uuid.New()

	t.Run("basic step outcome converts correctly", func(t *testing.T) {
		startTime := time.Now()
		outcome := contracts.StepOutcome{
			StepIndex:          5,
			Attempt:            1,
			NodeID:             "node-click-1",
			StepType:           "click",
			Success:            true,
			StartedAt:          startTime,
			DurationMs:         150,
			FinalURL:           "https://example.com/after",
			UsedSelector:       "#button",
			SelectorConfidence: 1.0,
			SelectorMatchCount: 1,
			ElementBoundingBox: &basbase.BoundingBox{X: 50, Y: 100, Width: 100, Height: 40},
			ClickPosition:      &basbase.Point{X: 100, Y: 120},
		}

		result := StepOutcomeToTelemetry(outcome, executionID)

		require.NotNil(t, result)
		assert.Contains(t, result.ID, executionID.String())
		assert.Contains(t, result.ID, "step-5")
		assert.Contains(t, result.ID, "attempt-1")
		assert.Equal(t, 5, result.SequenceNum)
		assert.Equal(t, 5, result.StepIndex)
		assert.Equal(t, "node-click-1", result.NodeID)
		assert.Equal(t, basactions.ActionType_ACTION_TYPE_CLICK, result.ActionType)
		assert.Equal(t, "node-click-1", result.Label) // Uses node ID as label
		assert.True(t, result.Success)
		assert.Equal(t, 150, result.DurationMs)
		assert.Equal(t, "https://example.com/after", result.FinalURL)
		assert.Equal(t, "#button", result.Selector)
		assert.Equal(t, 1.0, result.SelectorConfidence)
		assert.Equal(t, 1, result.SelectorMatchCount)

		// Check bounding box
		require.NotNil(t, result.BoundingBox)
		assert.Equal(t, float64(50), result.BoundingBox.X)

		// Check click position
		require.NotNil(t, result.ClickPosition)
		assert.Equal(t, float64(100), result.ClickPosition.X)

		// Check origin is execution
		require.NotNil(t, result.Origin)
		origin, ok := result.Origin.(*ExecutionOrigin)
		require.True(t, ok)
		assert.Equal(t, executionID, origin.ExecutionID)
		assert.Equal(t, 5, origin.StepIndex)
		assert.Equal(t, 1, origin.Attempt)
	})

	t.Run("failed step outcome with failure info", func(t *testing.T) {
		outcome := contracts.StepOutcome{
			StepIndex: 3,
			Attempt:   2,
			NodeID:    "node-assert-1",
			StepType:  "assert",
			Success:   false,
			Failure: &contracts.StepFailure{
				Kind:      contracts.FailureKindEngine,
				Code:      "ELEMENT_NOT_FOUND",
				Message:   "Selector did not match any elements",
				Retryable: true,
			},
		}

		result := StepOutcomeToTelemetry(outcome, executionID)

		require.NotNil(t, result)
		assert.False(t, result.Success)

		require.NotNil(t, result.Failure)
		assert.Equal(t, string(contracts.FailureKindEngine), result.Failure.Kind)
		assert.Equal(t, "ELEMENT_NOT_FOUND", result.Failure.Code)
		assert.Equal(t, "Selector did not match any elements", result.Failure.Message)
		assert.True(t, result.Failure.Retryable)
	})

	t.Run("step outcome with assertion", func(t *testing.T) {
		outcome := contracts.StepOutcome{
			StepIndex: 1,
			NodeID:    "node-assert-1",
			StepType:  "assert",
			Success:   true,
			Assertion: &contracts.AssertionOutcome{
				Mode:          "exists",
				Selector:      "#success-message",
				Success:       true,
				Negated:       false,
				CaseSensitive: true,
				Message:       "Element found",
			},
		}

		result := StepOutcomeToTelemetry(outcome, executionID)

		require.NotNil(t, result)
		origin, ok := result.Origin.(*ExecutionOrigin)
		require.True(t, ok)

		require.NotNil(t, origin.Assertion)
		assert.Equal(t, "exists", origin.Assertion.Mode)
		assert.Equal(t, "#success-message", origin.Assertion.Selector)
		assert.True(t, origin.Assertion.Success)
		assert.True(t, origin.Assertion.CaseSensitive)
	})

	t.Run("step outcome with extracted data", func(t *testing.T) {
		outcome := contracts.StepOutcome{
			StepIndex: 2,
			NodeID:    "node-extract-1",
			StepType:  "evaluate",
			Success:   true,
			ExtractedData: map[string]any{
				"price":    "$99.99",
				"quantity": 5,
				"inStock":  true,
			},
		}

		result := StepOutcomeToTelemetry(outcome, executionID)

		require.NotNil(t, result)
		origin, ok := result.Origin.(*ExecutionOrigin)
		require.True(t, ok)

		require.NotNil(t, origin.ExtractedData)
		assert.Equal(t, "$99.99", origin.ExtractedData["price"])
		assert.Equal(t, 5, origin.ExtractedData["quantity"])
		assert.Equal(t, true, origin.ExtractedData["inStock"])
	})

	t.Run("step outcome with screenshot and DOM", func(t *testing.T) {
		outcome := contracts.StepOutcome{
			StepIndex: 1,
			NodeID:    "node-1",
			StepType:  "click",
			Success:   true,
			Screenshot: &contracts.Screenshot{
				Data:      []byte("fake-jpeg-data"),
				MediaType: "image/jpeg",
				Width:     1280,
				Height:    720,
			},
			DOMSnapshot: &contracts.DOMSnapshot{
				HTML:    "<html><body>...</body></html>",
				Preview: "Page content...",
			},
		}

		result := StepOutcomeToTelemetry(outcome, executionID)

		require.NotNil(t, result)

		require.NotNil(t, result.Screenshot)
		assert.Equal(t, []byte("fake-jpeg-data"), result.Screenshot.Data)
		assert.Equal(t, "image/jpeg", result.Screenshot.MediaType)
		assert.Equal(t, 1280, result.Screenshot.Width)
		assert.Equal(t, 720, result.Screenshot.Height)

		require.NotNil(t, result.DOMSnapshot)
		assert.Equal(t, "<html><body>...</body></html>", result.DOMSnapshot.HTML)
		assert.Equal(t, "Page content...", result.DOMSnapshot.Preview)
	})
}

func TestTelemetryToTimelineEntry(t *testing.T) {
	t.Run("nil telemetry returns nil", func(t *testing.T) {
		result := TelemetryToTimelineEntry(nil)
		assert.Nil(t, result)
	})

	t.Run("basic recording telemetry converts to TimelineEntry", func(t *testing.T) {
		timestamp := time.Now()
		tel := &ActionTelemetry{
			ID:                 "action-1",
			SequenceNum:        1,
			ActionType:         basactions.ActionType_ACTION_TYPE_CLICK,
			Label:              "Click: Submit",
			Timestamp:          timestamp,
			DurationMs:         100,
			Selector:           "#submit",
			SelectorConfidence: 0.95,
			URL:                "https://example.com",
			Success:            true,
			Params: map[string]any{
				"selector": "#submit",
			},
			Origin: &RecordingOrigin{
				SessionID:  "session-1",
				Confidence: 0.95,
				Source:     basbase.RecordingSource_RECORDING_SOURCE_AUTO,
			},
		}

		entry := TelemetryToTimelineEntry(tel)

		require.NotNil(t, entry)
		assert.Equal(t, "action-1", entry.Id)
		assert.Equal(t, int32(1), entry.SequenceNum)

		// Check action definition
		require.NotNil(t, entry.Action)
		assert.Equal(t, basactions.ActionType_ACTION_TYPE_CLICK, entry.Action.Type)
		require.NotNil(t, entry.Action.Metadata)
		require.NotNil(t, entry.Action.Metadata.Label)
		assert.Equal(t, "Click: Submit", *entry.Action.Metadata.Label)

		// Check telemetry
		require.NotNil(t, entry.Telemetry)
		assert.Equal(t, "https://example.com", entry.Telemetry.Url)

		// Check context
		require.NotNil(t, entry.Context)
		require.NotNil(t, entry.Context.Success)
		assert.True(t, *entry.Context.Success)

		// Check recording origin
		sessionID := entry.Context.GetSessionId()
		assert.Equal(t, "session-1", sessionID)
	})

	t.Run("execution telemetry converts to TimelineEntry", func(t *testing.T) {
		executionID := uuid.New()
		timestamp := time.Now()
		tel := &ActionTelemetry{
			ID:          "step-1",
			SequenceNum: 1,
			StepIndex:   1,
			NodeID:      "node-1",
			ActionType:  basactions.ActionType_ACTION_TYPE_NAVIGATE,
			Label:       "node-1",
			Timestamp:   timestamp,
			DurationMs:  500,
			URL:         "https://example.com",
			FinalURL:    "https://example.com/page",
			Success:     true,
			Params: map[string]any{
				"url": "https://example.com",
			},
			Origin: &ExecutionOrigin{
				ExecutionID: executionID,
				StepIndex:   1,
				Attempt:     1,
				MaxAttempts: 3,
			},
		}

		entry := TelemetryToTimelineEntry(tel)

		require.NotNil(t, entry)
		assert.Equal(t, "step-1", entry.Id)
		require.NotNil(t, entry.NodeId)
		assert.Equal(t, "node-1", *entry.NodeId)
		require.NotNil(t, entry.StepIndex)
		assert.Equal(t, int32(1), *entry.StepIndex)

		// Check context for execution origin
		require.NotNil(t, entry.Context)
		executionIDStr := entry.Context.GetExecutionId()
		assert.Equal(t, executionID.String(), executionIDStr)

		// Check retry status
		require.NotNil(t, entry.Context.RetryStatus)
		assert.Equal(t, int32(1), entry.Context.RetryStatus.CurrentAttempt)
		assert.Equal(t, int32(3), entry.Context.RetryStatus.MaxAttempts)
	})

	t.Run("failure info is included", func(t *testing.T) {
		tel := &ActionTelemetry{
			ID:         "step-failed",
			ActionType: basactions.ActionType_ACTION_TYPE_CLICK,
			Success:    false,
			Failure: &FailureInfo{
				Kind:    "timeout",
				Code:    "ELEMENT_TIMEOUT",
				Message: "Element not found within timeout",
			},
			Origin: &ExecutionOrigin{
				ExecutionID: uuid.New(),
			},
		}

		entry := TelemetryToTimelineEntry(tel)

		require.NotNil(t, entry)
		require.NotNil(t, entry.Context)
		require.NotNil(t, entry.Context.Success)
		assert.False(t, *entry.Context.Success)
		require.NotNil(t, entry.Context.Error)
		assert.Equal(t, "Element not found within timeout", *entry.Context.Error)
		require.NotNil(t, entry.Context.ErrorCode)
		assert.Equal(t, "ELEMENT_TIMEOUT", *entry.Context.ErrorCode)
	})
}

func TestOriginTypes(t *testing.T) {
	t.Run("RecordingOrigin implements ActionOrigin", func(t *testing.T) {
		var origin ActionOrigin = &RecordingOrigin{SessionID: "test"}
		assert.Equal(t, "recording", origin.OriginType())
	})

	t.Run("ExecutionOrigin implements ActionOrigin", func(t *testing.T) {
		var origin ActionOrigin = &ExecutionOrigin{ExecutionID: uuid.New()}
		assert.Equal(t, "execution", origin.OriginType())
	})
}

func TestParamNormalization(t *testing.T) {
	t.Run("select normalizes selectedText to label", func(t *testing.T) {
		action := &driver.RecordedAction{
			ID:         "action-1",
			ActionType: "select",
			Payload: map[string]interface{}{
				"selectedText":  "Option A",
				"selectedIndex": 0,
			},
		}

		result := RecordedActionToTelemetry(action)

		require.NotNil(t, result)
		assert.Equal(t, "Option A", result.Params["label"])
		assert.Equal(t, 0, result.Params["index"])
	})

	t.Run("scroll normalizes scrollX/scrollY", func(t *testing.T) {
		action := &driver.RecordedAction{
			ID:         "action-1",
			ActionType: "scroll",
			Payload: map[string]interface{}{
				"scrollX": 100,
				"scrollY": 200,
			},
		}

		result := RecordedActionToTelemetry(action)

		require.NotNil(t, result)
		assert.Equal(t, 100, result.Params["x"])
		assert.Equal(t, 200, result.Params["y"])
	})

	t.Run("wait normalizes ms to durationMs", func(t *testing.T) {
		action := &driver.RecordedAction{
			ID:         "action-1",
			ActionType: "wait",
			Payload: map[string]interface{}{
				"ms": 1000,
			},
		}

		result := RecordedActionToTelemetry(action)

		require.NotNil(t, result)
		assert.Equal(t, 1000, result.Params["durationMs"])
	})
}
