package driver

import (
	"testing"

	"github.com/stretchr/testify/require"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/proto"
)

func TestRecordedActionFromTimelineEntry_PreservesNavigateWaitUntil(t *testing.T) {
	waitUntil := basactions.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_NETWORKIDLE
	action := RecordedActionFromTimelineEntry(&bastimeline.TimelineEntry{
		Action: &basactions.ActionDefinition{
			Type: basactions.ActionType_ACTION_TYPE_NAVIGATE,
			Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{
				Url:       "https://example.com",
				WaitUntil: &waitUntil,
			}},
		},
		Context: &basbase.EventContext{},
	})
	require.Equal(t, "networkidle", action.Payload["waitUntil"])
}

func TestRecordedActionFromTimelineEntry_PreservesFrameIdentity(t *testing.T) {
	frameID := "late-frame"
	action := RecordedActionFromTimelineEntry(&bastimeline.TimelineEntry{
		Telemetry: &basdomain.ActionTelemetry{FrameId: &frameID, FramePath: []string{"#outer", "#late-frame"}},
	})
	require.Equal(t, frameID, action.FrameID)
	require.Equal(t, []string{"#outer", "#late-frame"}, action.FramePath)
}

func TestRecordedActionFromTimelineEntry_PreservesDriverPageIdentity(t *testing.T) {
	action := RecordedActionFromTimelineEntry(&bastimeline.TimelineEntry{
		Telemetry: &basdomain.ActionTelemetry{DriverPageId: proto.String("recorded-tab-two")},
	})
	require.Equal(t, "recorded-tab-two", action.DriverPageID)
}

func TestRecordedActionFromTimelineEntry_PreservesEmptyInputSnapshot(t *testing.T) {
	action := RecordedActionFromTimelineEntry(&bastimeline.TimelineEntry{
		Action: &basactions.ActionDefinition{
			Type: basactions.ActionType_ACTION_TYPE_INPUT,
			Params: &basactions.ActionDefinition_Input{Input: &basactions.InputParams{
				Selector: "#input", Value: "",
			}},
		},
	})
	require.Equal(t, "input", action.ActionType)
	require.Contains(t, action.Payload, "text")
	require.Equal(t, "", action.Payload["text"])
}
