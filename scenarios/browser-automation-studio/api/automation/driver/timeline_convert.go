package driver

import (
	"encoding/json"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	"google.golang.org/protobuf/encoding/protojson"
)

// RecordedActionFromTimelineEntry converts a TimelineEntry into the legacy RecordedAction shape.
// This keeps older workflow generation and UI code working while the wire format is TimelineEntry.
func RecordedActionFromTimelineEntry(entry *bastimeline.TimelineEntry) RecordedAction {
	if entry == nil {
		return RecordedAction{}
	}

	action := entry.GetAction()
	actionType := enums.ActionTypeToString(action.GetType())
	timestamp := time.Now()
	if entry.GetTimestamp() != nil {
		parsed := entry.GetTimestamp().AsTime()
		if !parsed.IsZero() {
			timestamp = parsed
		}
	}

	sessionID := extractSessionID(entry.GetContext())
	selectorPrimary := selectorFromActionParams(action)

	var selector *SelectorSet
	if selectorPrimary != "" || hasSelectorCandidates(action.GetMetadata()) {
		selector = &SelectorSet{
			Primary:    selectorPrimary,
			Candidates: selectorCandidatesFromMetadata(action.GetMetadata()),
		}
	}

	payload := payloadFromActionParams(action)
	if len(payload) == 0 {
		payload = nil
	}

	elementMeta := elementMetaFromMetadata(action.GetMetadata())
	boundingBox := boundingBoxFromMetadata(action.GetMetadata())
	cursorPos := cursorPosFromTelemetry(entry.GetTelemetry())

	return RecordedAction{
		ID:          entry.GetId(),
		SessionID:   sessionID,
		SequenceNum: int(entry.GetSequenceNum()),
		Timestamp:   timestamp.UTC().Format(time.RFC3339Nano),
		DurationMs:  int(entry.GetDurationMs()),
		ActionType:  actionType,
		Confidence:  action.GetMetadata().GetConfidence(),
		Selector:    selector,
		ElementMeta: elementMeta,
		BoundingBox: boundingBox,
		Payload:     payload,
		URL:         urlFromEntry(entry, action),
		FrameID:     entry.GetTelemetry().GetFrameId(),
		CursorPos:   cursorPos,
	}
}

func extractSessionID(context *basbase.EventContext) string {
	if context == nil {
		return ""
	}
	switch origin := context.GetOrigin().(type) {
	case *basbase.EventContext_SessionId:
		return origin.SessionId
	default:
		return ""
	}
}

func selectorFromActionParams(action *basactions.ActionDefinition) string {
	if action == nil {
		return ""
	}
	switch params := action.GetParams().(type) {
	case *basactions.ActionDefinition_Click:
		return params.Click.GetSelector()
	case *basactions.ActionDefinition_Input:
		return params.Input.GetSelector()
	case *basactions.ActionDefinition_Hover:
		return params.Hover.GetSelector()
	case *basactions.ActionDefinition_Focus:
		return params.Focus.GetSelector()
	case *basactions.ActionDefinition_Assert:
		return params.Assert.GetSelector()
	case *basactions.ActionDefinition_SelectOption:
		return params.SelectOption.GetSelector()
	case *basactions.ActionDefinition_Scroll:
		return params.Scroll.GetSelector()
	case *basactions.ActionDefinition_Screenshot:
		return params.Screenshot.GetSelector()
	case *basactions.ActionDefinition_Blur:
		return params.Blur.GetSelector()
	default:
		return ""
	}
}

func selectorCandidatesFromMetadata(meta *basactions.ActionMetadata) []SelectorCandidate {
	if meta == nil {
		return nil
	}
	candidates := meta.GetSelectorCandidates()
	if len(candidates) == 0 {
		return nil
	}
	result := make([]SelectorCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		result = append(result, SelectorCandidate{
			Type:        enums.SelectorTypeToString(candidate.GetType()),
			Value:       candidate.GetValue(),
			Confidence:  candidate.GetConfidence(),
			Specificity: int(candidate.GetSpecificity()),
		})
	}
	return result
}

func hasSelectorCandidates(meta *basactions.ActionMetadata) bool {
	return meta != nil && len(meta.GetSelectorCandidates()) > 0
}

func elementMetaFromMetadata(meta *basactions.ActionMetadata) *ElementMeta {
	if meta == nil {
		return nil
	}
	snapshot := meta.GetElementSnapshot()
	if snapshot == nil {
		return nil
	}

	attributes := map[string]string{}
	for key, value := range snapshot.GetAttributes() {
		attributes[key] = value
	}

	return &ElementMeta{
		TagName:    snapshot.GetTagName(),
		ID:         snapshot.GetId(),
		ClassName:  snapshot.GetClassName(),
		InnerText:  snapshot.GetInnerText(),
		Attributes: attributes,
		IsVisible:  snapshot.GetIsVisible(),
		IsEnabled:  snapshot.GetIsEnabled(),
		Role:       snapshot.GetRole(),
		AriaLabel:  snapshot.GetAriaLabel(),
	}
}

func boundingBoxFromMetadata(meta *basactions.ActionMetadata) *contracts.BoundingBox {
	if meta == nil {
		return nil
	}
	bbox := meta.GetCapturedBoundingBox()
	if bbox == nil {
		return nil
	}
	return &contracts.BoundingBox{
		X:      bbox.GetX(),
		Y:      bbox.GetY(),
		Width:  bbox.GetWidth(),
		Height: bbox.GetHeight(),
	}
}

func cursorPosFromTelemetry(telemetry *basdomain.ActionTelemetry) *contracts.Point {
	if telemetry == nil {
		return nil
	}
	cursor := telemetry.GetCursorPosition()
	if cursor == nil {
		return nil
	}
	return &contracts.Point{
		X: cursor.GetX(),
		Y: cursor.GetY(),
	}
}

func urlFromEntry(entry *bastimeline.TimelineEntry, action *basactions.ActionDefinition) string {
	if action != nil {
		if params, ok := action.GetParams().(*basactions.ActionDefinition_Navigate); ok && params.Navigate != nil {
			if params.Navigate.GetUrl() != "" {
				return params.Navigate.GetUrl()
			}
		}
	}
	if telemetry := entry.GetTelemetry(); telemetry != nil {
		return telemetry.GetUrl()
	}
	return ""
}

func payloadFromActionParams(action *basactions.ActionDefinition) map[string]interface{} {
	if action == nil {
		return nil
	}
	payload := map[string]interface{}{}

	switch params := action.GetParams().(type) {
	case *basactions.ActionDefinition_Click:
		click := params.Click
		if click == nil {
			break
		}
		if click.Button != nil {
			payload["button"] = enums.MouseButtonToString(click.GetButton())
		}
		if len(click.Modifiers) > 0 {
			modifiers := make([]string, 0, len(click.Modifiers))
			for _, mod := range click.Modifiers {
				modifiers = append(modifiers, enums.KeyboardModifierToString(mod))
			}
			payload["modifiers"] = modifiers
		}
		if click.ClickCount != nil {
			payload["clickCount"] = click.GetClickCount()
		}
		if click.DelayMs != nil {
			payload["delay"] = click.GetDelayMs()
		}
	case *basactions.ActionDefinition_Input:
		input := params.Input
		if input == nil {
			break
		}
		if input.Value != "" {
			payload["text"] = input.Value
		}
		if input.Submit != nil {
			payload["submit"] = input.GetSubmit()
		}
		if input.DelayMs != nil {
			payload["delayMs"] = input.GetDelayMs()
		}
	case *basactions.ActionDefinition_Navigate:
		nav := params.Navigate
		if nav == nil {
			break
		}
		if nav.WaitForSelector != nil {
			payload["waitForSelector"] = nav.GetWaitForSelector()
		}
		if nav.TimeoutMs != nil {
			payload["timeoutMs"] = nav.GetTimeoutMs()
		}
	case *basactions.ActionDefinition_Scroll:
		scroll := params.Scroll
		if scroll == nil {
			break
		}
		if scroll.DeltaY != nil {
			payload["scrollY"] = float64(scroll.GetDeltaY())
		} else if scroll.Y != nil {
			payload["scrollY"] = float64(scroll.GetY())
		}
	case *basactions.ActionDefinition_SelectOption:
		selectOpt := params.SelectOption
		if selectOpt == nil {
			break
		}
		switch selector := selectOpt.SelectBy.(type) {
		case *basactions.SelectParams_Value:
			payload["value"] = selector.Value
		case *basactions.SelectParams_Label:
			payload["value"] = selector.Label
		case *basactions.SelectParams_Index:
			payload["value"] = selector.Index
		}
	case *basactions.ActionDefinition_Keyboard:
		keyboard := params.Keyboard
		if keyboard == nil {
			break
		}
		if keyboard.Key != nil {
			payload["key"] = keyboard.GetKey()
		} else if len(keyboard.Keys) > 0 {
			payload["key"] = keyboard.Keys[0]
		}
		if len(keyboard.Modifiers) > 0 {
			modifiers := make([]string, 0, len(keyboard.Modifiers))
			for _, mod := range keyboard.Modifiers {
				modifiers = append(modifiers, enums.KeyboardModifierToString(mod))
			}
			payload["modifiers"] = modifiers
		}
	case *basactions.ActionDefinition_Wait:
		wait := params.Wait
		if wait == nil {
			break
		}
		if wait.TimeoutMs != nil {
			payload["timeoutMs"] = wait.GetTimeoutMs()
		}
	case *basactions.ActionDefinition_Assert:
		assert := params.Assert
		if assert == nil {
			break
		}
		payload["mode"] = assert.GetMode().String()
		if assert.Expected != nil {
			payload["expected"] = jsonValueToInterface(assert.Expected)
		}
		if assert.Negated != nil {
			payload["negated"] = assert.GetNegated()
		}
		if assert.CaseSensitive != nil {
			payload["caseSensitive"] = assert.GetCaseSensitive()
		}
		if assert.AttributeName != nil {
			payload["attribute"] = assert.GetAttributeName()
		}
	case *basactions.ActionDefinition_Screenshot:
		screenshot := params.Screenshot
		if screenshot == nil {
			break
		}
		if screenshot.FullPage != nil {
			payload["fullPage"] = screenshot.GetFullPage()
		}
		if screenshot.Quality != nil {
			payload["quality"] = screenshot.GetQuality()
		}
	case *basactions.ActionDefinition_DragDrop:
		dragDrop := params.DragDrop
		if dragDrop == nil {
			break
		}
		if dragDrop.TargetSelector != nil {
			payload["targetSelector"] = dragDrop.GetTargetSelector()
		}
	}

	return payload
}

func jsonValueToInterface(value *commonv1.JsonValue) interface{} {
	if value == nil {
		return nil
	}
	data, err := protojson.Marshal(value)
	if err != nil {
		return nil
	}
	var output interface{}
	if err := json.Unmarshal(data, &output); err != nil {
		return nil
	}
	return output
}
