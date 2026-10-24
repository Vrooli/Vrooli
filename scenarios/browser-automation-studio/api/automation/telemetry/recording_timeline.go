package telemetry

import (
	"strings"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BuildRecordingTimelineEntry constructs the proto entry at the raw recording seam.
func BuildRecordingTimelineEntry(action *driver.RecordedAction) *bastimeline.TimelineEntry {
	if action == nil {
		return nil
	}
	actionType := strings.ToLower(strings.TrimSpace(action.ActionType))
	if actionType == "reload" || actionType == "goback" || actionType == "goforward" {
		actionType = "navigate"
	}
	kind := enums.StringToActionType(actionType)
	params := normalizeRecordingParams(action)
	entry := &bastimeline.TimelineEntry{
		Id: action.ID, SequenceNum: int32(action.SequenceNum),
		Action:    &basactions.ActionDefinition{Type: kind, Metadata: &basactions.ActionMetadata{}},
		Telemetry: &basdomain.ActionTelemetry{Url: action.URL},
		Context:   &basbase.EventContext{Origin: &basbase.EventContext_SessionId{SessionId: action.SessionID}},
	}
	setTypedParams(entry.Action, kind, params)
	if action.FrameID != "" {
		entry.Telemetry.FrameId = &action.FrameID
	}
	if action.DurationMs > 0 {
		duration := int32(action.DurationMs)
		entry.DurationMs = &duration
	}
	if ts, err := time.Parse(time.RFC3339Nano, action.Timestamp); err == nil {
		entry.Timestamp = timestamppb.New(ts)
	} else if ts, err := time.Parse(time.RFC3339, action.Timestamp); err == nil {
		entry.Timestamp = timestamppb.New(ts)
	}
	if label := generateRecordingLabel(action); label != "" {
		entry.Action.Metadata.Label = &label
	}
	if action.Confidence > 0 {
		entry.Action.Metadata.Confidence = &action.Confidence
	}
	if action.ElementMeta != nil {
		entry.Action.Metadata.ElementSnapshot = convertDriverElementMeta(action.ElementMeta)
	}
	if action.BoundingBox != nil {
		entry.Action.Metadata.CapturedBoundingBox = action.BoundingBox
	}
	if !entry.Timestamp.AsTime().IsZero() {
		entry.Action.Metadata.CapturedAt = entry.Timestamp
	}
	if action.Selector != nil {
		entry.Action.Metadata.SelectorCandidates = convertSelectorCandidates(action.Selector)
	}
	if action.BoundingBox != nil {
		entry.Telemetry.ElementBoundingBox = action.BoundingBox
	}
	if action.CursorPos != nil {
		entry.Telemetry.CursorPosition = action.CursorPos
	}
	success := true
	needsConfirmation := shouldRequireConfirmation(action)
	source := basbase.RecordingSource_RECORDING_SOURCE_AUTO
	entry.Context.Success = &success
	entry.Context.Source = &source
	entry.Context.NeedsConfirmation = &needsConfirmation
	return entry
}
