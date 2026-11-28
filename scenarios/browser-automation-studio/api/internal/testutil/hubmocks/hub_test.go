package hubmocks

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHubRecordsEnvelopesAndSubscriberBehavior(t *testing.T) {
	hub := New()
	event := struct{ Type string }{Type: "navigation"}
	hub.BroadcastEnvelope(event)
	require.Equal(t, 1, hub.BroadcastEnvelopeCount())
	require.Equal(t, event, hub.LastBroadcastEnvelope())
	hub.ResetBroadcastEnvelopes()
	require.Zero(t, hub.BroadcastEnvelopeCount())
	require.Nil(t, hub.LastBroadcastEnvelope())

	hub.RecordingSubscribers["session"] = true
	require.True(t, hub.HasRecordingFrameSubscribers("session"))
	require.Equal(t, 1, hub.BroadcastTimelineEntry("session", nil).SentCount)
	hub.ExecutionFrameSubscribers["execution"] = true
	require.True(t, hub.HasExecutionFrameSubscribers("execution"))
	hub.ClientCount = 2
	require.Equal(t, 2, hub.GetClientCount())
}
