// Package vendorws hosts deterministic fake WebSocket servers shaped
// like the streaming endpoints of upstream vendors (Deepgram, OpenAI
// Realtime). It is the substrate Phase E of the streaming-decoupling
// plan will wire into the BYOK adapters; until then it exists so the
// next agent can land coverage diffs next to the implementation in
// one PR rather than having to invent the rig from scratch.
//
// Each constructor returns an *httptest.Server whose handler upgrades
// to a WS, replays a canned script of frames, and closes. Callers
// override behaviour via small functional options (script-of-frames,
// on-message hook). The handlers carry no business logic.
package vendorws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/gorilla/websocket"
)

// Frame is one outbound message the fake server will send. Binary
// frames are sent as opcode 0x2; non-empty Text is sent as opcode 0x1.
type Frame struct {
	Text   string
	Binary []byte
}

// Options shape one fake server.
type Options struct {
	// Prelude is emitted immediately after upgrade, before WaitForFrames gates
	// Script. Streaming resources use it for admission/control lifecycle frames
	// (for example Kyutai's queued/ready) that must precede client audio.
	Prelude []Frame
	// Script is the ordered sequence of frames the server emits after
	// upgrade. Use this for protocol shapes that emit unsolicited
	// transcript updates (Deepgram's "Results" messages).
	Script []Frame

	// OnMessage, if non-nil, is invoked for each inbound frame received
	// from the client. Use this to record audio bytes for assertion.
	OnMessage func(messageType int, payload []byte)

	// CloseAfterScript, if true, closes the connection cleanly after
	// the last scripted frame has been sent. Defaults to true.
	CloseAfterScript bool

	// WaitForFrames, when > 0, makes the writer hold its Script until the
	// reader has received this many inbound frames from the client. This
	// removes the race in tests that assert the client sent a full sequence
	// (start header + N audio frames + end marker) before the server replies
	// and closes — without it the scripted reply + close can land before the
	// client's pump finishes writing. 0 = emit the script immediately.
	WaitForFrames int
}

// NewDeepgramServer returns a fake Deepgram streaming endpoint. The
// Deepgram wire shape is JSON Results frames with `channel.alternatives`
// transcripts. Test callers seed Options.Script with those JSON frames.
func NewDeepgramServer(opts Options) *httptest.Server {
	return newServer(opts, "deepgram")
}

// NewOpenAIRealtimeServer returns a fake OpenAI Realtime API endpoint.
// The Realtime wire shape is bidirectional event JSON with a
// `transcription.delta` partial-text stream.
func NewOpenAIRealtimeServer(opts Options) *httptest.Server {
	return newServer(opts, "openai-realtime")
}

// NewKyutaiServer returns a fake kyutai-stt resource streaming endpoint. The
// resource's stable contract emits JSON Text frames tagged with a "type"
// ("partial" | "segment" | "done" | "error"). Test callers seed Options.Script
// with those frames (via EncodeJSON) and use OnMessage to assert the start
// header + binary PCM frames the KyutaiProvider sends.
func NewKyutaiServer(opts Options) *httptest.Server {
	return newServer(opts, "kyutai")
}

func newServer(opts Options, _ string) *httptest.Server {
	closeAfter := opts.CloseAfterScript || true
	up := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// Reader: forward inbound frames to the hook so callers can
		// assert what the adapter sent (audio frames, control msgs).
		// framesReady closes once WaitForFrames inbound frames have arrived.
		readerDone := make(chan struct{})
		framesReady := make(chan struct{})
		var readyOnce sync.Once
		signalReady := func() { readyOnce.Do(func() { close(framesReady) }) }
		if opts.WaitForFrames <= 0 {
			signalReady()
		}
		go func() {
			defer close(readerDone)
			count := 0
			for {
				mt, data, err := conn.ReadMessage()
				if err != nil {
					return
				}
				if opts.OnMessage != nil {
					opts.OnMessage(mt, data)
				}
				count++
				if opts.WaitForFrames > 0 && count >= opts.WaitForFrames {
					signalReady()
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
		}()

		// Admission/control frames are observable before any client audio.
		for _, f := range opts.Prelude {
			if len(f.Binary) > 0 {
				if err := conn.WriteMessage(websocket.BinaryMessage, f.Binary); err != nil {
					return
				}
				continue
			}
			if f.Text != "" {
				if err := conn.WriteMessage(websocket.TextMessage, []byte(f.Text)); err != nil {
					return
				}
			}
		}

		// Hold the script until the client has sent the frames the test
		// expects (when WaitForFrames is set), so the reply + close cannot
		// race ahead of the client's pump.
		select {
		case <-framesReady:
		case <-ctx.Done():
		}

		// Writer: emit the scripted frames in order.
		for _, f := range opts.Script {
			if len(f.Binary) > 0 {
				if err := conn.WriteMessage(websocket.BinaryMessage, f.Binary); err != nil {
					return
				}
				continue
			}
			if f.Text != "" {
				if err := conn.WriteMessage(websocket.TextMessage, []byte(f.Text)); err != nil {
					return
				}
			}
		}

		if closeAfter {
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		}

		<-readerDone
	})
	return httptest.NewServer(h)
}

// EncodeJSON is a tiny helper for tests to build Text frames out of
// vendor-shaped JSON payloads without import cycles.
func EncodeJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
