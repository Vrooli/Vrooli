package agentmanager

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"

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

func TestStreamCancellationInterruptsSilentRead(t *testing.T) {
	subscribed := make(chan struct{})
	peerClosed := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		close(subscribed)
		_, _, _ = conn.ReadMessage()
		close(peerClosed)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- NewWebSocketEventSource(server.URL, nil).StreamRunEvents(ctx, "run-1", func(ActivityEvent) error { return nil })
	}()
	select {
	case <-subscribed:
	case <-time.After(time.Second):
		t.Fatal("subscription timeout")
	}
	cancel()
	select {
	case err := <-done:
		require.True(t, errors.Is(err, context.Canceled), "%v", err)
	case <-time.After(time.Second):
		t.Fatal("cancel did not interrupt silent read")
	}
	select {
	case <-peerClosed:
	case <-time.After(time.Second):
		t.Fatal("websocket connection remained open")
	}
}

func TestSelectedRunRejectsForeignAndUnscopedEvents(t *testing.T) {
	for _, line := range []string{
		`{"type":"run_status","payload":{"id":"other","status":"complete"}}`,
		`{"type":"run_progress","payload":{"runId":"other","percentComplete":99}}`,
		`{"type":"run_event","payload":{"runId":"other","eventType":"message"}}`,
		`{"type":"run_status","payload":{"status":"complete"}}`,
		`{"type":"run_progress","payload":{"percentComplete":99}}`,
		`{"type":"run_event","payload":{"eventType":"message"}}`,
		`{"type":"log","payload":"unscoped"}`,
	} {
		t.Run(line, func(t *testing.T) {
			events, err := DecodeWebSocketLine([]byte(line), "run-1")
			require.NoError(t, err)
			require.Empty(t, events)
		})
	}
	events, err := DecodeWebSocketLine([]byte(`{"type":"run_event","runId":"run-1","payload":{"eventType":"message"}}`), "run-1")
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "run-1", events[0].RunID)
}
