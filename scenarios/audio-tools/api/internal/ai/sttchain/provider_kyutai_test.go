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
		// The resource admits start/control before PCM. Tests that model flush
		// semantics must expose ready or the provider correctly retains audio.
		_, start, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(vendorws.EncodeJSON(map[string]any{"type": "ready"}))); err != nil {
			return
		}
		_ = start
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

func TestKyutaiProvider_ZeroAudioDoesNotDialBackend(t *testing.T) {
	var mu sync.Mutex
	hits := 0
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
		conn, err := up.Upgrade(w, r, nil)
		if err == nil {
			_ = conn.Close()
		}
	}))
	defer srv.Close()

	p := sttchain.NewKyutaiProvider("http://example.invalid")
	p.StreamEndpoint = wsURL(srv.URL)

	chunks := make(chan sttchain.AudioChunk)
	close(chunks)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	events, err := p.TranscribeStreaming(ctx, sttchain.StreamStart{Language: "en"}, chunks)
	require.NoError(t, err)

	var done *sttchain.DoneEvent
	for ev := range events {
		if ev.Kind == sttchain.StreamEventDone {
			done = ev.Done
		}
	}

	mu.Lock()
	defer mu.Unlock()
	require.NotNil(t, done, "zero-audio stream should complete gracefully")
	require.Equal(t, 0, hits, "zero-audio pre-connect must not dial kyutai or contend for MODEL.lock")
}

func TestKyutaiProvider_UnexpectedBackendCloseEmitsErrorBeforeDone(t *testing.T) {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Read start, declare ready, then one audio frame and close without done.
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteMessage(websocket.TextMessage, []byte(vendorws.EncodeJSON(map[string]any{"type": "ready"})))
		_, _, _ = conn.ReadMessage()
	}))
	defer srv.Close()

	p := sttchain.NewKyutaiProvider("http://example.invalid")
	p.StreamEndpoint = wsURL(srv.URL)

	chunks := make(chan sttchain.AudioChunk, 1)
	chunks <- sttchain.AudioChunk{Audio: []byte{0x01, 0x02}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events, err := p.TranscribeStreaming(ctx, sttchain.StreamStart{Language: "en"}, chunks)
	require.NoError(t, err)

	var kinds []sttchain.StreamEventKind
	var sawErr error
	for ev := range events {
		kinds = append(kinds, ev.Kind)
		if ev.Kind == sttchain.StreamEventError {
			sawErr = ev.Error
		}
	}

	require.NotNil(t, sawErr, "backend WS close without done must be surfaced as StreamEventError")
	require.GreaterOrEqual(t, len(kinds), 2)
	require.Equal(t, sttchain.StreamEventError, kinds[len(kinds)-2], "error must precede terminal done")
	require.Equal(t, sttchain.StreamEventDone, kinds[len(kinds)-1], "stream still terminates with Done after error")
}

// TestKyutaiProvider_PartialFloodDoesNotBlockWebSocketReader is the provider
// regression guard for deterministic long-form replay. The resource can emit
// partials much faster than the downstream Segmenter consumes them; partials
// are explicitly droppable progress, so they must never stop this adapter from
// reading ping/control frames or the following durable terminal events.
func TestKyutaiProvider_PartialFloodDoesNotBlockWebSocketReader(t *testing.T) {
	doneWritten := make(chan struct{})
	allowClose := make(chan struct{})
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if _, _, err = conn.ReadMessage(); err != nil {
			return
		}
		if err = conn.WriteMessage(websocket.TextMessage, []byte(vendorws.EncodeJSON(map[string]any{"type": "ready"}))); err != nil {
			return
		}
		if _, _, err = conn.ReadMessage(); err != nil {
			return
		}
		payload := strings.Repeat("x", 2048)
		for i := 0; i < 30000; i++ {
			if err = conn.WriteMessage(websocket.TextMessage, []byte(vendorws.EncodeJSON(map[string]any{"type": "partial", "text": payload}))); err != nil {
				return
			}
		}
		if err = conn.WriteMessage(websocket.TextMessage, []byte(vendorws.EncodeJSON(map[string]any{"type": "segment", "text": "durable tail", "start_ms": 0, "end_ms": 100}))); err != nil {
			return
		}
		if err = conn.WriteMessage(websocket.TextMessage, []byte(vendorws.EncodeJSON(map[string]any{"type": "done"}))); err == nil {
			close(doneWritten)
			<-allowClose
		}
	}))
	t.Cleanup(func() { close(allowClose) })
	defer srv.Close()

	p := sttchain.NewKyutaiProvider("http://example.invalid")
	p.StreamEndpoint = wsURL(srv.URL)
	chunks := make(chan sttchain.AudioChunk, 1)
	chunks <- sttchain.AudioChunk{Audio: []byte{0x01, 0x02}}
	close(chunks)
	events, err := p.TranscribeStreaming(context.Background(), sttchain.StreamStart{}, chunks)
	require.NoError(t, err)

	select {
	case <-doneWritten:
	case <-time.After(6 * time.Second):
		t.Fatal("partial flood blocked the Kyutai WebSocket reader before durable terminal events; partials must be droppable")
	}

	sawSegment, sawDone := false, false
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for !sawSegment || !sawDone {
		select {
		case ev, ok := <-events:
			if !ok {
				t.Fatalf("event stream closed before durable terminal events: segment=%t done=%t", sawSegment, sawDone)
			}
			if ev.Kind == sttchain.StreamEventSegment && ev.Segment.Text == "durable tail" {
				sawSegment = true
			}
			if ev.Kind == sttchain.StreamEventDone {
				sawDone = true
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for durable terminal events: segment=%t done=%t", sawSegment, sawDone)
		}
	}
	require.True(t, sawSegment)
	require.True(t, sawDone)
}

func TestKyutaiProvider_BoundsInFlightAudioUntilProcessedCredit(t *testing.T) {
	const expectedWindow = 8
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	ninthArrivedEarly := make(chan bool, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if _, _, err = conn.ReadMessage(); err != nil {
			return
		}
		if err = conn.WriteMessage(websocket.TextMessage, []byte(vendorws.EncodeJSON(map[string]any{"type": "ready"}))); err != nil {
			return
		}
		for i := 0; i < expectedWindow; i++ {
			mt, _, readErr := conn.ReadMessage()
			if readErr != nil || mt != websocket.BinaryMessage {
				return
			}
		}
		_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		mt, _, readErr := conn.ReadMessage()
		ninthArrivedEarly <- readErr == nil && mt == websocket.BinaryMessage
		_ = conn.SetReadDeadline(time.Time{})
		if err = conn.WriteMessage(websocket.TextMessage, []byte(vendorws.EncodeJSON(map[string]any{"type": "processed", "processed_batches": expectedWindow}))); err != nil {
			return
		}
		for i := 0; i < 2; i++ {
			mt, _, readErr = conn.ReadMessage()
			if readErr != nil || mt != websocket.BinaryMessage {
				return
			}
		}
		_, _, _ = conn.ReadMessage() // end
		_ = conn.WriteMessage(websocket.TextMessage, []byte(vendorws.EncodeJSON(map[string]any{"type": "done"})))
	}))
	defer srv.Close()

	p := sttchain.NewKyutaiProvider("http://example.invalid")
	p.StreamEndpoint = wsURL(srv.URL)
	chunks := make(chan sttchain.AudioChunk, 10)
	for i := 0; i < 10; i++ {
		chunks <- sttchain.AudioChunk{Audio: []byte{byte(i), 0x01}}
	}
	close(chunks)
	events, err := p.TranscribeStreaming(context.Background(), sttchain.StreamStart{}, chunks)
	require.NoError(t, err)
	for range events {
	}
	require.False(t, <-ninthArrivedEarly, "the ninth PCM batch must wait for resource processed credit")
}

func TestKyutaiProvider_TranslatesEventStream(t *testing.T) {
	srv := vendorws.NewKyutaiServer(vendorws.Options{
		Prelude: []vendorws.Frame{{Text: vendorws.EncodeJSON(map[string]any{"type": "ready"})}},
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

func TestKyutaiProvider_MapsAdmissionLifecycleAndApproximateSpan(t *testing.T) {
	srv := vendorws.NewKyutaiServer(vendorws.Options{Prelude: []vendorws.Frame{
		{Text: vendorws.EncodeJSON(map[string]any{"type": "queued", "position": 2})},
		{Text: vendorws.EncodeJSON(map[string]any{"type": "ready"})},
	}, Script: []vendorws.Frame{
		{Text: vendorws.EncodeJSON(map[string]any{"type": "segment", "text": "hello", "start_ms": 10, "end_ms": 30})},
		{Text: vendorws.EncodeJSON(map[string]any{"type": "done"})},
	}})
	defer srv.Close()
	p := sttchain.NewKyutaiProvider("http://example.invalid")
	p.StreamEndpoint = wsURL(srv.URL)
	chunks := make(chan sttchain.AudioChunk, 1)
	chunks <- sttchain.AudioChunk{Audio: []byte{1, 2}, StartSample: 1_000}
	close(chunks)
	events, err := p.TranscribeStreaming(context.Background(), sttchain.StreamStart{SessionID: "session", Generation: 3}, chunks)
	require.NoError(t, err)
	var states []string
	var segment *sttchain.SegmentEvent
	for event := range events {
		if event.SessionStatus != nil {
			states = append(states, event.SessionStatus.State)
		}
		if event.Segment != nil {
			segment = event.Segment
		}
	}
	require.Equal(t, []string{"queued", "ready"}, states)
	require.NotNil(t, segment)
	require.Equal(t, int64(1_160), segment.StartSample)
	require.Equal(t, int64(1_480), segment.EndSample)
	require.Equal(t, "approximate", segment.AlignmentQuality)
}

func TestKyutaiProvider_DoesNotSendPCMBeforeAdmissionReady(t *testing.T) {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	preReadyAudio := make(chan bool, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Start is allowed while queued. Any PCM before ready violates the
		// resource admission contract and bypasses retained replay coverage.
		type inbound struct {
			mt  int
			err error
		}
		inboundFrames := make(chan inbound, 4)
		go func() {
			for {
				mt, _, readErr := conn.ReadMessage()
				inboundFrames <- inbound{mt: mt, err: readErr}
				if readErr != nil {
					return
				}
			}
		}()
		if first := <-inboundFrames; first.err != nil {
			return
		}
		select {
		case early := <-inboundFrames:
			preReadyAudio <- early.err == nil && early.mt == websocket.BinaryMessage
		case <-time.After(75 * time.Millisecond):
			preReadyAudio <- false
		}
		_ = conn.WriteMessage(websocket.TextMessage, []byte(vendorws.EncodeJSON(map[string]any{"type": "ready"})))
		// Drain the released PCM and terminal control before completing.
		for i := 0; i < 2; i++ {
			if frame := <-inboundFrames; frame.err != nil {
				return
			}
		}
		_ = conn.WriteMessage(websocket.TextMessage, []byte(vendorws.EncodeJSON(map[string]any{"type": "done"})))
	}))
	defer srv.Close()

	p := sttchain.NewKyutaiProvider("http://example.invalid")
	p.StreamEndpoint = wsURL(srv.URL)
	chunks := make(chan sttchain.AudioChunk, 1)
	chunks <- sttchain.AudioChunk{Audio: []byte{1, 2, 3, 4}}
	close(chunks)
	events, err := p.TranscribeStreaming(context.Background(), sttchain.StreamStart{}, chunks)
	require.NoError(t, err)
	for range events {
	}
	require.False(t, <-preReadyAudio, "PCM must remain retained until the resource declares ready")
}

func TestKyutaiProvider_SendsStartHeaderThenPCM(t *testing.T) {
	var mu sync.Mutex
	var gotStart string
	var binaryFrames int
	var gotEnd bool
	srv := vendorws.NewKyutaiServer(vendorws.Options{
		Prelude:       []vendorws.Frame{{Text: vendorws.EncodeJSON(map[string]any{"type": "ready"})}},
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

func TestNewKyutaiProvider_UsesExplicitModelProvenance(t *testing.T) {
	t.Setenv("AUDIO_KYUTAI_MODEL_ID", "kyutai/stt-1b-en_fr@operator-pin")
	require.Equal(t, "kyutai/stt-1b-en_fr@operator-pin", sttchain.NewKyutaiProvider("http://example.invalid").Model())
}
