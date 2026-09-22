package handlers

import (
	"encoding/binary"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDecodeDriverFrame(t *testing.T) {
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xD9}
	payload, header, err := decodeDriverFrame(jpeg)
	if err != nil || header != nil || string(payload) != string(jpeg) {
		t.Fatalf("JPEG decoding = payload=%v header=%v err=%v", payload, header, err)
	}

	headerJSON := []byte(`{"frame_id":"frame-1","capture_ms":4}`)
	packet := make([]byte, 4+len(headerJSON)+len(jpeg))
	binary.BigEndian.PutUint32(packet[:4], uint32(len(headerJSON)))
	copy(packet[4:], headerJSON)
	copy(packet[4+len(headerJSON):], jpeg)
	payload, header, err = decodeDriverFrame(packet)
	if err != nil || header == nil || header.FrameID != "frame-1" || string(payload) != string(jpeg) {
		t.Fatalf("header decoding = payload=%v header=%+v err=%v", payload, header, err)
	}
}

func TestDriverFrameStatsExcludeIdleSocketWait(t *testing.T) {
	h, _, tempDir, _ := createTestHandlerWithRecordMode(t)
	defer os.RemoveAll(tempDir)
	collector := h.perfRegistry.GetOrCreate("idle")
	done := make(chan struct{})
	r := chi.NewRouter()
	r.Get("/{sessionId}", func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		h.HandleDriverFrameStream(w, r)
	})
	server := httptest.NewServer(r)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/idle", nil)
	require.NoError(t, err)
	defer conn.Close()
	// Deliberate absence of frames is idle time, never frame processing.
	time.Sleep(100 * time.Millisecond)
	header := []byte(`{"frame_id":"idle-1","capture_ms":2}`)
	packet := make([]byte, 4+len(header)+4)
	binary.BigEndian.PutUint32(packet, uint32(len(header)))
	copy(packet[4:], header)
	copy(packet[4+len(header):], []byte{0xff, 0xd8, 0xff, 0xd9})
	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, packet))
	require.NoError(t, conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")))
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("frame handler did not finish")
	}
	frames := collector.GetRecentFrames(1)
	require.Len(t, frames, 1)
	require.Zero(t, frames[0].APIReceiveMs, "message availability is not a transit-time measurement")
	require.GreaterOrEqual(t, frames[0].APITotalMs, frames[0].APIBroadcastMs)
}
