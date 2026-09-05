package driver

import (
	"testing"

	"github.com/stretchr/testify/require"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
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
