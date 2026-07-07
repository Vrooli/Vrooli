package sttchain_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/testutil/vendorws"
)

// wsURL rewrites an httptest http:// URL to ws:// so the gorilla dialer accepts it.
func wsURL(httpURL string) string {
	return "ws://" + strings.TrimPrefix(httpURL, "http://")
}

// newKyutaiFlushServer models kyutai's flush-on-end semantics: it emits NOTHING
// until it receives the {"type":"end"} marker, then emits a single trailing
// "segment" (the flushed delayed-streams tail) followed by "done" and closes.
// This lets tests prove the provider sends end + awaits the flush before the
// socket closes on both the graceful and the cancel teardown paths. The
// returned func reports how many end markers the server observed.
func newKyutaiFlushServer(tailText string) (*httptest.Server, func() int) {
	var mu sync.Mutex
	ends := 0
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.TextMessage && strings.Contains(string(data), `"end"`) {
				mu.Lock()
				ends++
				mu.Unlock()
				_ = conn.WriteMessage(websocket.TextMessage, []byte(vendorws.EncodeJSON(
					map[string]any{"type": "segment", "text": tailText, "start_ms": 0, "end_ms": 100})))
				_ = conn.WriteMessage(websocket.TextMessage, []byte(vendorws.EncodeJSON(
					map[string]any{"type": "done"})))
				return
			}
		}
	})
	srv := httptest.NewServer(h)
	return srv, func() int { mu.Lock(); defer mu.Unlock(); return ends }
}

// TestKyutaiProvider_GracefulCloseDrainsFlushTail asserts the normal teardown
// (chunks close) sends the end marker and awaits the server's flushed trailing
// segment before the stream ends — the tail is never dropped.
func TestKyutaiProvider_GracefulCloseDrainsFlushTail(t *testing.T) {
	srv, endCount := newKyutaiFlushServer("flushed tail")
	defer srv.Close()

	p := sttchain.NewKyutaiProvider("http://example.invalid")
	p.StreamEndpoint = wsURL(srv.URL)

	chunks := make(chan sttchain.AudioChunk, 1)
	chunks <- sttchain.AudioChunk{Audio: []byte{0x01, 0x02}}
	close(chunks)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events, err := p.TranscribeStreaming(ctx, sttchain.StreamStart{Language: "en"}, chunks)
	require.NoError(t, err)

	var segments []string
	var done *sttchain.DoneEvent
	for ev := range events {
		switch ev.Kind {
		case sttchain.StreamEventSegment:
			segments = append(segments, ev.Segment.Text)
		case sttchain.StreamEventDone:
			done = ev.Done
		}
	}
	require.Equal(t, []string{"flushed tail"}, segments, "graceful close must await the flushed tail")
	require.NotNil(t, done)
	require.Equal(t, 1, endCount(), "exactly one end marker on graceful close")
}

// TestKyutaiProvider_CancelDrainsTailBeforeClose is the tail-durability
// regression: cancelling the session MID-STREAM (idle timeout / request cancel,
// chunks NOT closed) must still send the end marker and await the flush so the
// trailing segment is delivered, instead of cold-closing the socket and
// dropping it (the pre-fix behaviour).
func TestKyutaiProvider_CancelDrainsTailBeforeClose(t *testing.T) {
	srv, endCount := newKyutaiFlushServer("trailing words")
	defer srv.Close()

	p := sttchain.NewKyutaiProvider("http://example.invalid")
	p.StreamEndpoint = wsURL(srv.URL)

	chunks := make(chan sttchain.AudioChunk) // unbuffered, left OPEN
	ctx, cancel := context.WithCancel(context.Background())
	events, err := p.TranscribeStreaming(ctx, sttchain.StreamStart{Language: "en"}, chunks)
	require.NoError(t, err)

	// Feed one chunk (pump consumes it), then cancel without closing chunks —
	// the teardown-race path that used to lose the tail.
	chunks <- sttchain.AudioChunk{Audio: []byte{0x01, 0x02}}
	cancel()

	var segments []string
	var done *sttchain.DoneEvent
	for ev := range events {
		switch ev.Kind {
		case sttchain.StreamEventSegment:
			segments = append(segments, ev.Segment.Text)
		case sttchain.StreamEventDone:
			done = ev.Done
		}
	}
	require.Equal(t, []string{"trailing words"}, segments,
		"cancel must send end + await flush so the trailing segment is delivered")
	require.NotNil(t, done)
	require.Equal(t, 1, endCount(), "exactly one end marker sent on cancel (no double-write)")
}

func TestKyutaiProvider_TranslatesEventStream(t *testing.T) {
	srv := vendorws.NewKyutaiServer(vendorws.Options{
		Script: []vendorws.Frame{
			{Text: vendorws.EncodeJSON(map[string]any{"type": "partial", "text": "hel"})},
			{Text: vendorws.EncodeJSON(map[string]any{"type": "segment", "text": "hello", "start_ms": 0, "end_ms": 500})},
			{Text: vendorws.EncodeJSON(map[string]any{"type": "segment", "text": "world", "start_ms": 500, "end_ms": 900})},
			{Text: vendorws.EncodeJSON(map[string]any{"type": "done"})},
		},
	})
	defer srv.Close()

	p := sttchain.NewKyutaiProvider("http://example.invalid")
	p.StreamEndpoint = wsURL(srv.URL)

	chunks := make(chan sttchain.AudioChunk, 1)
	chunks <- sttchain.AudioChunk{Audio: []byte{0x01, 0x02, 0x03, 0x04}}
	close(chunks)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events, err := p.TranscribeStreaming(ctx, sttchain.StreamStart{Language: "en"}, chunks)
	require.NoError(t, err)

	var partials, segments []string
	var done *sttchain.DoneEvent
	for ev := range events {
		switch ev.Kind {
		case sttchain.StreamEventPartial:
			partials = append(partials, ev.Partial.Text)
		case sttchain.StreamEventSegment:
			segments = append(segments, ev.Segment.Text)
			require.Equal(t, sttchain.TierLocal, ev.Segment.ProviderTier)
			require.Equal(t, "kyutai", ev.Segment.ProviderID)
		case sttchain.StreamEventDone:
			done = ev.Done
		}
	}
	require.Equal(t, []string{"hel"}, partials)
	require.Equal(t, []string{"hello", "world"}, segments)
	require.NotNil(t, done)
	require.Equal(t, "hello world", done.FinalText)
	require.Equal(t, sttchain.TierLocal, done.LockedTier)
}

func TestKyutaiProvider_SendsStartHeaderThenPCM(t *testing.T) {
	var mu sync.Mutex
	var gotStart string
	var binaryFrames int
	var gotEnd bool
	srv := vendorws.NewKyutaiServer(vendorws.Options{
		Script:        []vendorws.Frame{{Text: vendorws.EncodeJSON(map[string]any{"type": "done"})}},
		WaitForFrames: 4, // start header + 2 PCM frames + end marker
		OnMessage: func(mt int, payload []byte) {
			mu.Lock()
			defer mu.Unlock()
			switch {
			case strings.Contains(string(payload), `"start"`):
				gotStart = string(payload)
			case strings.Contains(string(payload), `"end"`):
				gotEnd = true
			default:
				binaryFrames++
			}
		},
	})
	defer srv.Close()

	p := sttchain.NewKyutaiProvider("http://example.invalid")
	p.StreamEndpoint = wsURL(srv.URL)

	chunks := make(chan sttchain.AudioChunk, 2)
	chunks <- sttchain.AudioChunk{Audio: []byte{0x10, 0x20}}
	chunks <- sttchain.AudioChunk{Audio: []byte{0x30, 0x40}}
	close(chunks)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events, err := p.TranscribeStreaming(ctx, sttchain.StreamStart{Language: "fr", SampleRate: 16000}, chunks)
	require.NoError(t, err)
	for range events { //nolint:revive // drain
	}

	mu.Lock()
	defer mu.Unlock()
	require.Contains(t, gotStart, `"sample_rate":16000`)
	require.Contains(t, gotStart, `"language":"fr"`)
	require.Equal(t, 2, binaryFrames, "both PCM chunks forwarded as binary frames")
	require.True(t, gotEnd, "end marker sent after chunks close")
}

func TestKyutaiProvider_TraitsAndBatch(t *testing.T) {
	p := sttchain.NewKyutaiProvider("http://example.invalid")
	tr := p.Traits()
	require.True(t, tr.Stream)
	require.False(t, tr.Batch)
	require.Equal(t, []sttchain.StrategyKind{sttchain.StrategyPassthrough}, tr.Strategies)
	require.Equal(t, sttchain.TierLocal, p.Type())

	_, err := p.Transcribe(context.Background(), sttchain.Request{})
	require.Error(t, err, "kyutai must refuse unary/batch transcription")
}
