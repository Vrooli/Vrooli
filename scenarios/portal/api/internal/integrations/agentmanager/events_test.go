package agentmanager

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebSocketURLConvertsHTTPBase(t *testing.T) {
	got, err := WebSocketURL("http://localhost:17400")
	require.NoError(t, err)
	require.Equal(t, "ws://localhost:17400/api/v1/ws", got)
}

func TestDecodeWebSocketLineMapsRunEvents(t *testing.T) {
	line := []byte(`{"type":"run_event","payload":{"runId":"run-1","sequence":7,"eventType":"RUN_EVENT_TYPE_MESSAGE","data":{"message":"edited files"}}}`)

	events, err := DecodeWebSocketLine(line, "run-1")

	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, EventKindMessage, events[0].Kind)
	require.Equal(t, int64(7), events[0].Sequence)
	require.Contains(t, events[0].Text, "edited files")
}

func TestDecodeWebSocketLineMarksTerminalStatusDone(t *testing.T) {
	line := []byte(`{"type":"run_status","payload":{"id":"run-1","status":"complete"}}`)

	events, err := DecodeWebSocketLine(line, "run-1")

	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, EventKindDone, events[0].Kind)
	require.True(t, events[0].Done)
}

func TestDecodeWebSocketLineSupportsProtoJSONEnvelope(t *testing.T) {
	line := []byte(`{"type":"AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_PROGRESS","run_progress":{"run_id":"run-1","percent_complete":42,"phase":"execute","current_action":"editing"}}`)

	events, err := DecodeWebSocketLine(line, "run-1")

	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, EventKindProgress, events[0].Kind)
	require.Equal(t, "run-1", events[0].RunID)
	require.Contains(t, events[0].Text, "42%")
	require.Contains(t, events[0].Text, "editing")
}

func TestDecodeWebSocketLineNormalizesProtoJSONTerminalStatus(t *testing.T) {
	line := []byte(`{"type":"AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS","run_status":{"run_id":"run-1","status":"RUN_STATUS_FAILED"}}`)

	events, err := DecodeWebSocketLine(line, "run-1")

	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, EventKindDone, events[0].Kind)
	require.True(t, events[0].Done)
	require.Equal(t, "Agent run failed", events[0].Text)
}
