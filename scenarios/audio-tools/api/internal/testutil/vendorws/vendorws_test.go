package vendorws

import (
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// TestDeepgramServer_ScriptedFramesAndInboundCapture is the rig's
// self-test. It proves the script is replayed, inbound frames are
// captured, and the connection closes cleanly.
func TestDeepgramServer_ScriptedFramesAndInboundCapture(t *testing.T) {
	var mu sync.Mutex
	var inbound [][]byte
	srv := NewDeepgramServer(Options{
		Script: []Frame{
			{Text: EncodeJSON(map[string]any{"type": "Results", "channel": map[string]any{"alternatives": []any{map[string]any{"transcript": "hello"}}}})},
			{Text: EncodeJSON(map[string]any{"type": "Results", "is_final": true, "channel": map[string]any{"alternatives": []any{map[string]any{"transcript": "hello world"}}}})},
		},
		OnMessage: func(_ int, p []byte) {
			mu.Lock()
			inbound = append(inbound, append([]byte(nil), p...))
			mu.Unlock()
		},
	})
	t.Cleanup(srv.Close)

	wsURL, _ := url.Parse(srv.URL)
	wsURL.Scheme = "ws"
	c, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.WriteMessage(websocket.BinaryMessage, []byte("pcm-bytes")))

	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	for i := 0; i < 2; i++ {
		_, raw, err := c.ReadMessage()
		require.NoError(t, err)
		require.Contains(t, string(raw), "Results")
	}

	// Server should close after script; expect Close frame next.
	_, _, err = c.ReadMessage()
	require.Error(t, err)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(inbound) >= 1
	}, 2*time.Second, 25*time.Millisecond)
}

// TestOpenAIRealtimeServer_BasicHandshake confirms the second constructor
// is wired and produces the same upgrade/script behaviour.
func TestOpenAIRealtimeServer_BasicHandshake(t *testing.T) {
	srv := NewOpenAIRealtimeServer(Options{
		Script: []Frame{{Text: EncodeJSON(map[string]any{"type": "transcription.delta", "delta": "hi"})}},
	})
	t.Cleanup(srv.Close)

	wsURL, _ := url.Parse(srv.URL)
	wsURL.Scheme = "ws"
	c, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := c.ReadMessage()
	require.NoError(t, err)
	require.Contains(t, string(raw), "transcription.delta")
}

func TestKyutaiServer_PreludeWaitsForPCMAndReplaysBinary(t *testing.T) {
	var inbound [][]byte
	srv := NewKyutaiServer(Options{
		Prelude:       []Frame{{Text: `{"type":"ready"}`}},
		Script:        []Frame{{Binary: []byte{1, 2, 3}}, {Text: `{"type":"done"}`}},
		WaitForFrames: 1,
		OnMessage: func(_ int, payload []byte) {
			inbound = append(inbound, append([]byte(nil), payload...))
		},
	})
	t.Cleanup(srv.Close)

	wsURL, _ := url.Parse(srv.URL)
	wsURL.Scheme = "ws"
	client, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))

	_, ready, err := client.ReadMessage()
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"ready"}`, string(ready))
	require.NoError(t, client.WriteMessage(websocket.BinaryMessage, []byte("pcm")))
	messageType, binary, err := client.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, websocket.BinaryMessage, messageType)
	require.Equal(t, []byte{1, 2, 3}, binary)
	_, done, err := client.ReadMessage()
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"done"}`, string(done))
	require.Eventually(t, func() bool { return len(inbound) == 1 }, time.Second, 10*time.Millisecond)
}
