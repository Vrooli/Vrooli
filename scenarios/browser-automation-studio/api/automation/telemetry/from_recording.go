package telemetry

import (
	"strings"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
)

// RecordedActionToTelemetry converts a recorded action from the
// playwright-driver to the unified ActionTelemetry format.
func RecordedActionToTelemetry(action *driver.RecordedAction) *ActionTelemetry {
	if action == nil {
		return nil
	}

	tel := &ActionTelemetry{
		ID:          action.ID,
		SequenceNum: action.SequenceNum,
		ActionType:  enums.StringToActionType(action.ActionType),
		Params:      normalizeRecordingParams(action),
		URL:         action.URL,
		FrameID:     action.FrameID,
		Success:     true, // Recording capture always succeeds
		DurationMs:  action.DurationMs,
	}

	// Parse timestamp
	if action.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, action.Timestamp); err == nil {
			tel.Timestamp = ts
		} else if ts, err := time.Parse(time.RFC3339, action.Timestamp); err == nil {
			tel.Timestamp = ts
		}
	}

	// Element context
	if action.Selector != nil {
		tel.Selector = action.Selector.Primary
	}
	tel.SelectorConfidence = action.Confidence
	if action.ElementMeta != nil {
		tel.ElementSnapshot = convertDriverElementMeta(action.ElementMeta)
	}
	if action.BoundingBox != nil {
		tel.BoundingBox = action.BoundingBox
	}
	if action.CursorPos != nil {
		tel.CursorPosition = action.CursorPos
	}

	// Generate label from element info
	tel.Label = generateRecordingLabel(action)

	// Recording-specific origin
	tel.Origin = &RecordingOrigin{
		SessionID:          action.SessionID,
		Confidence:         action.Confidence,
		SelectorCandidates: convertSelectorCandidates(action.Selector),
		NeedsConfirmation:  shouldRequireConfirmation(action),
		Source:             basbase.RecordingSource_RECORDING_SOURCE_AUTO,
	}

	return tel
}

// normalizeRecordingParams converts RecordedAction payload to standardized params.
// Handles recording-specific field name aliases.
func normalizeRecordingParams(action *driver.RecordedAction) map[string]any {
	params := make(map[string]any)

	// Copy all payload fields
	for k, v := range action.Payload {
		params[k] = v
	}

	// Add selector if present
	if action.Selector != nil && action.Selector.Primary != "" {
		params["selector"] = action.Selector.Primary
	}

	// Normalize recording-specific field names:

	// Navigate: prefer targetUrl from payload, fall back to action.URL
	if _, hasURL := params["url"]; !hasURL {
		if targetURL, ok := params["targetUrl"].(string); ok {
			params["url"] = targetURL
		} else {
			params["url"] = action.URL
		}
	}

	// Select: normalize "selectedText" → "label", "selectedIndex" → "index"
	if selectedText, ok := params["selectedText"].(string); ok && selectedText != "" {
		if _, hasLabel := params["label"]; !hasLabel {
			params["label"] = selectedText
		}
	}
	if selectedIndex, ok := params["selectedIndex"]; ok {
		if _, hasIndex := params["index"]; !hasIndex {
			params["index"] = selectedIndex
		}
	}

	// Scroll: normalize "scrollX" → "x", "scrollY" → "y"
	if scrollX, ok := params["scrollX"]; ok {
		if _, hasX := params["x"]; !hasX {
			params["x"] = scrollX
		}
	}
	if scrollY, ok := params["scrollY"]; ok {
		if _, hasY := params["y"]; !hasY {
			params["y"] = scrollY
		}
	}

	// Wait: normalize "ms" → "durationMs"
	if ms, ok := params["ms"]; ok {
		if _, hasDuration := params["durationMs"]; !hasDuration {
			params["durationMs"] = ms
		}
	}

	// Input: normalize "delay" → "delayMs"
	if delay, ok := params["delay"]; ok {
		if _, hasDelayMs := params["delayMs"]; !hasDelayMs {
			params["delayMs"] = delay
		}
	}

	// Assert: default mode to "exists" if not specified
	if _, hasMode := params["mode"]; !hasMode {
		if _, hasAssertMode := params["assertMode"]; !hasAssertMode {
			params["mode"] = "exists"
		}
	}

	return params
}

func convertDriverElementMeta(meta *driver.ElementMeta) *basdomain.ElementMeta {
	if meta == nil {
		return nil
	}
	result := &basdomain.ElementMeta{
		TagName:   meta.TagName,
		Id:        meta.ID,
		ClassName: meta.ClassName,
		InnerText: meta.InnerText,
		IsVisible: meta.IsVisible,
		IsEnabled: meta.IsEnabled,
		Role:      meta.Role,
		AriaLabel: meta.AriaLabel,
	}
	if len(meta.Attributes) > 0 {
		result.Attributes = meta.Attributes
	}
	return result
}

func convertSelectorCandidates(selector *driver.SelectorSet) []*basdomain.SelectorCandidate {
	if selector == nil || len(selector.Candidates) == 0 {
		return nil
	}
	candidates := make([]*basdomain.SelectorCandidate, 0, len(selector.Candidates))
	for _, c := range selector.Candidates {
		candidates = append(candidates, &basdomain.SelectorCandidate{
			Type:        enums.StringToSelectorType(c.Type),
			Value:       c.Value,
			Confidence:  c.Confidence,
			Specificity: int32(c.Specificity),
		})
	}
	return candidates
}

func shouldRequireConfirmation(action *driver.RecordedAction) bool {
	if action.Selector == nil || len(action.Selector.Candidates) == 0 {
		return false
	}
	return len(action.Selector.Candidates) > 1 || action.Confidence < 0.8
}

func generateRecordingLabel(action *driver.RecordedAction) string {
	actionType := action.ActionType

	if action.ElementMeta != nil {
		if action.ElementMeta.AriaLabel != "" {
			return capitalize(actionType) + ": " + truncate(action.ElementMeta.AriaLabel, 30)
		}
		if action.ElementMeta.InnerText != "" {
			return capitalize(actionType) + ": \"" + truncate(action.ElementMeta.InnerText, 30) + "\""
		}
		if action.ElementMeta.ID != "" {
			return capitalize(actionType) + ": #" + action.ElementMeta.ID
		}
		return capitalize(actionType) + ": " + action.ElementMeta.TagName
	}

	if actionType == "navigate" {
		if targetURL, ok := action.Payload["targetUrl"].(string); ok {
			return "Navigate to " + truncate(targetURL, 40)
		}
		return "Navigate to " + truncate(action.URL, 40)
	}

	return capitalize(actionType)
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
