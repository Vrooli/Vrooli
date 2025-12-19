package telemetry

import (
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// TelemetryToTimelineEntry converts the unified ActionTelemetry to
// the proto TimelineEntry format for UI streaming.
//
// This is the SINGLE, SHARED converter used by both recording and execution paths.
func TelemetryToTimelineEntry(tel *ActionTelemetry) *bastimeline.TimelineEntry {
	if tel == nil {
		return nil
	}

	entry := &bastimeline.TimelineEntry{
		Id:          tel.ID,
		SequenceNum: int32(tel.SequenceNum),
	}

	// Optional fields
	if tel.NodeID != "" {
		entry.NodeId = &tel.NodeID
	}
	if tel.StepIndex > 0 {
		stepIndex := int32(tel.StepIndex)
		entry.StepIndex = &stepIndex
	}
	if !tel.Timestamp.IsZero() {
		entry.Timestamp = contracts.TimeToTimestamp(tel.Timestamp)
	}
	if tel.DurationMs > 0 {
		d := int32(tel.DurationMs)
		entry.DurationMs = &d
	}

	// Build components
	entry.Action = buildActionDefinition(tel)
	entry.Telemetry = buildProtoActionTelemetry(tel)
	entry.Context = buildEventContext(tel)

	return entry
}

func buildActionDefinition(tel *ActionTelemetry) *basactions.ActionDefinition {
	def := &basactions.ActionDefinition{
		Type: tel.ActionType,
	}

	// Build typed params using shared typeconv builders
	setTypedParams(def, tel.ActionType, tel.Params)

	// Build metadata
	def.Metadata = &basactions.ActionMetadata{}
	if tel.Label != "" {
		def.Metadata.Label = &tel.Label
	}
	if tel.SelectorConfidence > 0 {
		def.Metadata.Confidence = &tel.SelectorConfidence
	}
	if tel.ElementSnapshot != nil {
		def.Metadata.ElementSnapshot = tel.ElementSnapshot
	}
	if tel.BoundingBox != nil {
		def.Metadata.CapturedBoundingBox = tel.BoundingBox
	}
	if !tel.Timestamp.IsZero() {
		def.Metadata.CapturedAt = contracts.TimeToTimestamp(tel.Timestamp)
	}

	// Add selector candidates for recording origin
	if origin, ok := tel.Origin.(*RecordingOrigin); ok {
		if len(origin.SelectorCandidates) > 0 {
			def.Metadata.SelectorCandidates = origin.SelectorCandidates
		}
	}

	return def
}

// setTypedParams sets the typed params on the ActionDefinition based on action type.
func setTypedParams(def *basactions.ActionDefinition, actionType basactions.ActionType, params map[string]any) {
	if params == nil {
		return
	}

	switch actionType {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		def.Params = &basactions.ActionDefinition_Navigate{
			Navigate: typeconv.BuildNavigateParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_CLICK:
		def.Params = &basactions.ActionDefinition_Click{
			Click: typeconv.BuildClickParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_INPUT:
		def.Params = &basactions.ActionDefinition_Input{
			Input: typeconv.BuildInputParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SCROLL:
		def.Params = &basactions.ActionDefinition_Scroll{
			Scroll: typeconv.BuildScrollParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_HOVER:
		def.Params = &basactions.ActionDefinition_Hover{
			Hover: typeconv.BuildHoverParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_FOCUS:
		def.Params = &basactions.ActionDefinition_Focus{
			Focus: typeconv.BuildFocusParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_BLUR:
		def.Params = &basactions.ActionDefinition_Blur{
			Blur: typeconv.BuildBlurParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SELECT:
		def.Params = &basactions.ActionDefinition_SelectOption{
			SelectOption: typeconv.BuildSelectParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_KEYBOARD:
		def.Params = &basactions.ActionDefinition_Keyboard{
			Keyboard: typeconv.BuildKeyboardParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_WAIT:
		def.Params = &basactions.ActionDefinition_Wait{
			Wait: typeconv.BuildWaitParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_ASSERT:
		def.Params = &basactions.ActionDefinition_Assert{
			Assert: typeconv.BuildAssertParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SCREENSHOT:
		def.Params = &basactions.ActionDefinition_Screenshot{
			Screenshot: typeconv.BuildScreenshotParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_EVALUATE:
		def.Params = &basactions.ActionDefinition_Evaluate{
			Evaluate: typeconv.BuildEvaluateParams(params),
		}
	default:
		// Don't set params for unknown types
	}
}

func buildProtoActionTelemetry(tel *ActionTelemetry) *basdomain.ActionTelemetry {
	at := &basdomain.ActionTelemetry{
		Url: tel.URL,
	}

	if tel.FrameID != "" {
		at.FrameId = &tel.FrameID
	}

	// Screenshot
	if tel.Screenshot != nil {
		at.Screenshot = &basdomain.TimelineScreenshot{
			Width:       int32(tel.Screenshot.Width),
			Height:      int32(tel.Screenshot.Height),
			ContentType: tel.Screenshot.MediaType,
		}
	}

	// DOM snapshot
	if tel.DOMSnapshot != nil {
		if tel.DOMSnapshot.Preview != "" {
			at.DomSnapshotPreview = &tel.DOMSnapshot.Preview
		}
		if tel.DOMSnapshot.HTML != "" {
			at.DomSnapshotHtml = &tel.DOMSnapshot.HTML
		}
	}

	// Geometry
	at.ElementBoundingBox = tel.BoundingBox
	at.ClickPosition = tel.ClickPosition
	at.CursorPosition = tel.CursorPosition
	at.CursorTrail = tel.CursorTrail
	at.HighlightRegions = tel.HighlightRegions
	at.MaskRegions = tel.MaskRegions

	if tel.ZoomFactor != 0 {
		at.ZoomFactor = &tel.ZoomFactor
	}

	// Console logs
	if len(tel.ConsoleLogs) > 0 {
		at.ConsoleLogs = make([]*basdomain.ConsoleLogEntry, len(tel.ConsoleLogs))
		for i, log := range tel.ConsoleLogs {
			entry := &basdomain.ConsoleLogEntry{
				Level: enums.StringToLogLevel(log.Type),
				Text:  log.Text,
			}
			if log.Stack != "" {
				entry.Stack = &log.Stack
			}
			if log.Location != "" {
				entry.Location = &log.Location
			}
			if !log.Timestamp.IsZero() {
				entry.Timestamp = contracts.TimeToTimestamp(log.Timestamp)
			}
			at.ConsoleLogs[i] = entry
		}
	}

	// Network events
	if len(tel.Network) > 0 {
		at.NetworkEvents = make([]*basdomain.NetworkEvent, len(tel.Network))
		for i, evt := range tel.Network {
			event := &basdomain.NetworkEvent{
				Type: enums.StringToNetworkEventType(evt.Type),
				Url:  evt.URL,
			}
			if evt.Method != "" {
				event.Method = &evt.Method
			}
			if evt.ResourceType != "" {
				event.ResourceType = &evt.ResourceType
			}
			if evt.Status != 0 {
				status := int32(evt.Status)
				event.Status = &status
			}
			if evt.OK {
				event.Ok = &evt.OK
			}
			if evt.Failure != "" {
				event.Failure = &evt.Failure
			}
			if !evt.Timestamp.IsZero() {
				event.Timestamp = contracts.TimeToTimestamp(evt.Timestamp)
			}
			at.NetworkEvents[i] = event
		}
	}

	return at
}

func buildEventContext(tel *ActionTelemetry) *basbase.EventContext {
	ctx := &basbase.EventContext{
		Success: &tel.Success,
	}

	// Handle origin-specific context
	switch origin := tel.Origin.(type) {
	case *RecordingOrigin:
		ctx.Origin = &basbase.EventContext_SessionId{
			SessionId: origin.SessionID,
		}
		ctx.Source = &origin.Source
		ctx.NeedsConfirmation = &origin.NeedsConfirmation

	case *ExecutionOrigin:
		ctx.Origin = &basbase.EventContext_ExecutionId{
			ExecutionId: origin.ExecutionID.String(),
		}
		ctx.RetryStatus = &basbase.RetryStatus{
			CurrentAttempt: int32(origin.Attempt),
			MaxAttempts:    int32(origin.MaxAttempts),
			Configured:     origin.Attempt > 0,
		}

		// Assertion result
		if origin.Assertion != nil {
			ctx.Assertion = &basbase.AssertionResult{
				Success:       origin.Assertion.Success,
				Mode:          enums.StringToAssertionMode(origin.Assertion.Mode),
				Selector:      origin.Assertion.Selector,
				Negated:       origin.Assertion.Negated,
				CaseSensitive: origin.Assertion.CaseSensitive,
			}
			if origin.Assertion.Message != "" {
				ctx.Assertion.Message = &origin.Assertion.Message
			}
			if origin.Assertion.Expected != nil {
				ctx.Assertion.Expected = typeconv.AnyToJsonValue(origin.Assertion.Expected)
			}
			if origin.Assertion.Actual != nil {
				ctx.Assertion.Actual = typeconv.AnyToJsonValue(origin.Assertion.Actual)
			}
		}

		// Extracted data
		if len(origin.ExtractedData) > 0 {
			ctx.ExtractedData = make(map[string]*commonv1.JsonValue, len(origin.ExtractedData))
			for k, v := range origin.ExtractedData {
				if jsonVal := typeconv.AnyToJsonValue(v); jsonVal != nil {
					ctx.ExtractedData[k] = jsonVal
				}
			}
		}
	}

	// Failure info (common to both origins)
	if tel.Failure != nil {
		if tel.Failure.Message != "" {
			ctx.Error = &tel.Failure.Message
		}
		if tel.Failure.Code != "" {
			ctx.ErrorCode = &tel.Failure.Code
		}
	}

	return ctx
}
