package stt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/store"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/testutil/mocks"
)

// guardRecorder is a goroutine-safe usage recorder: the handler enqueues the
// per-session delivery telemetry row from its own goroutine while the test
// reads it, so both sides must synchronize. got signals each Enqueue.
type guardRecorder struct {
	mu   sync.Mutex
	rows []store.UsageRow
	got  chan struct{}
}

func (g *guardRecorder) Enqueue(row store.UsageRow) {
	g.mu.Lock()
	g.rows = append(g.rows, row)
	g.mu.Unlock()
	select {
	case g.got <- struct{}{}:
	default:
	}
}
func (g *guardRecorder) Record(context.Context, store.UsageRow) error { return nil }
func (g *guardRecorder) Close()                                       {}
func (g *guardRecorder) first(t *testing.T) store.UsageRow {
	t.Helper()
	select {
	case <-g.got:
	case <-time.After(3 * time.Second):
		t.Fatal("delivery telemetry row never recorded")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	require.Len(t, g.rows, 1, "exactly one session row expected")
	return g.rows[0]
}

// newContinuousSpeechKyutaiServer models a long, pause-free dictation: for each
// of segCount committed segments it emits a burst of interim partials followed
// by a durable segment, then the terminal done. This is the backend shape a
// continuous speaker produces (kyutai force-commits durable segments mid-
// utterance while partials stream).
func newContinuousSpeechKyutaiServer(t *testing.T, segCount, partialsPerSeg int) (baseURL, wsEndpoint string) {
	t.Helper()
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	m := http.NewServeMux()
	m.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","model_loaded":true,"device":"cpu"}`))
	})
	m.HandleFunc("/v1/stream", func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ready"}`)); err != nil {
			return
		}
		// Match the resource contract: recognition output begins only after the
		// admitted session has accepted its first PCM frame. Emitting the entire
		// transcript before this read races the provider's readiness-gated pump
		// and makes the relay guard exercise an impossible backend lifecycle.
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		for s := 0; s < segCount; s++ {
			for p := 0; p < partialsPerSeg; p++ {
				frame := fmt.Sprintf(`{"type":"partial","text":"seg %d word %d"}`, s, p)
				if err := conn.WriteMessage(websocket.TextMessage, []byte(frame)); err != nil {
					return
				}
			}
			seg := fmt.Sprintf(`{"type":"segment","text":"segment %d","start_ms":%d,"end_ms":%d}`, s, s*1000, s*1000+900)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(seg)); err != nil {
				return
			}
		}
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"done"}`))
	})
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	return srv.URL, "ws" + strings.TrimPrefix(srv.URL, "http") + "/v1/stream"
}

// newContendedKyutaiServer models the post-fix Kyutai single-model policy:
// the first real audio stream owns the model and keeps decoding; a second real
// audio stream receives a typed busy error instead of killing the holder.
func newContendedKyutaiServer(t *testing.T) (baseURL, wsEndpoint string) {
	t.Helper()
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	var streamCount int32
	contenderHit := make(chan struct{})
	m := http.NewServeMux()
	m.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","model_loaded":true,"device":"cpu"}`))
	})
	m.HandleFunc("/v1/stream", func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ready"}`)); err != nil {
			return
		}
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		n := atomic.AddInt32(&streamCount, 1)
		if n > 1 {
			close(contenderHit)
			_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","code":"stt_busy","message":"kyutai model busy: active streaming session"}`))
			_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"done"}`))
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"segment","text":"holder head","start_ms":0,"end_ms":900}`)); err != nil {
			return
		}
		select {
		case <-contenderHit:
		case <-time.After(3 * time.Second):
			t.Error("contending stream never reached kyutai")
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"segment","text":"holder tail","start_ms":1000,"end_ms":1900}`)); err != nil {
			return
		}
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"done"}`))
	})
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	return srv.URL, "ws" + strings.TrimPrefix(srv.URL, "http") + "/v1/stream"
}

// TestStreamWS_ContinuousSpeechDeliversAllSegmentsGracefully is the always-on
// continuous-speech delivery guard (plan Phase 7). It exercises the full relay
// (provider → passthrough → egress → coalescing writer) against a backend that
// streams many durable segments interleaved with a partial firehose, and
// asserts the invariants that the backpressure-wedge fix guarantees:
//
//   - MONOTONIC, LOSSLESS durable delivery: every committed segment reaches the
//     browser exactly once, in ascending segment index.
//   - the terminal done arrives.
//   - ZERO non-graceful closes: emitStreamDeliveryTelemetry records a graceful
//     outcome with no drop signal (the drop counter, wired here into a HARD
//     assertion, stays clear).
//
// This is the permanent regression guard for the failure class: any relay
// change that reintroduces a wedge or a tail drop trips one of these.
func TestStreamWS_ContinuousSpeechDeliversAllSegmentsGracefully(t *testing.T) {
	const segCount = 20
	baseURL, wsEndpoint := newContinuousSpeechKyutaiServer(t, segCount, 8)

	kp := sttchain.NewKyutaiProvider(baseURL)
	kp.StreamEndpoint = wsEndpoint
	chain := sttchain.NewChain(sttchain.Options{
		EnableLocal:  true,
		LocalEngines: map[string]sttchain.Provider{"kyutai": kp},
	})
	rec := &guardRecorder{got: make(chan struct{}, 1)}
	deps := Deps{
		Chain:        chain,
		Selector:     sttpkg.NewSelector(chain),
		Logger:       &mocks.FakeLogger{},
		Usage:        rec,
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"auto","engine_id":"kyutai"}`},
	}

	r := mux.NewRouter()
	r.Handle("/api/v1/voice/stream", StreamWSHandler(deps)).Methods(http.MethodGet)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	wsu := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/voice/stream?language=en"
	c, resp, err := websocket.DefaultDialer.Dial(wsu, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	t.Cleanup(func() { _ = c.Close() })
	require.NoError(t, c.WriteMessage(websocket.BinaryMessage, []byte{0x01, 0x02}))

	// Normal consumer: drain messages, recording segment indices in arrival
	// order. On the terminal done, send a client done so the reader closes
	// gracefully (mirrors a real stop).
	_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	var segIndices []int
	sawDone := false
	for !sawDone {
		_, raw, rerr := c.ReadMessage()
		require.NoError(t, rerr)
		var msg wsMessage
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}
		switch msg.Type {
		case wsMsgSegmentFinal:
			segIndices = append(segIndices, msg.SegmentIndex)
		case wsMsgDone:
			sawDone = true
			_ = c.WriteMessage(websocket.TextMessage, []byte(`{"type":"done"}`))
		}
	}

	// Monotonic, lossless durable delivery: exactly 0..segCount-1 in order.
	require.Len(t, segIndices, segCount, "every committed segment must be delivered exactly once")
	for i, idx := range segIndices {
		require.Equal(t, i, idx, "segment indices must be delivered monotonically with no gaps")
	}

	// Drop-counter HARD assertion: the session closed gracefully with no drop.
	row := rec.first(t)
	require.Equal(t, "graceful", row.FallbackReason, "continuous speech must close gracefully")
	require.Empty(t, row.Error, "the drop counter must stay clear — no tail drop under continuous speech")
	require.Equal(t, int32(segCount), row.OutputTokens, "the session must report every committed segment")
}

func TestStreamWS_ContendingAudioStreamDoesNotDropActiveHolderTail(t *testing.T) {
	baseURL, wsEndpoint := newContendedKyutaiServer(t)

	kp := sttchain.NewKyutaiProvider(baseURL)
	kp.StreamEndpoint = wsEndpoint
	chain := sttchain.NewChain(sttchain.Options{
		EnableLocal:  true,
		LocalEngines: map[string]sttchain.Provider{"kyutai": kp},
	})
	deps := Deps{
		Chain:        chain,
		Selector:     sttpkg.NewSelector(chain),
		Logger:       &mocks.FakeLogger{},
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"auto","engine_id":"kyutai"}`},
	}

	r := mux.NewRouter()
	r.Handle("/api/v1/voice/stream", StreamWSHandler(deps)).Methods(http.MethodGet)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	wsu := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/voice/stream?language=en"
	holder, resp, err := websocket.DefaultDialer.Dial(wsu, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	t.Cleanup(func() { _ = holder.Close() })
	require.NoError(t, holder.WriteMessage(websocket.BinaryMessage, []byte{0x01, 0x02}))

	_ = holder.SetReadDeadline(time.Now().Add(5 * time.Second))
	var holderSegments []string
	for len(holderSegments) == 0 {
		_, raw, rerr := holder.ReadMessage()
		require.NoError(t, rerr)
		var msg wsMessage
		require.NoError(t, json.Unmarshal(raw, &msg))
		if msg.Type == wsMsgSegmentFinal {
			holderSegments = append(holderSegments, msg.Text)
		}
	}
	require.Equal(t, []string{"holder head"}, holderSegments)

	contender, resp, err := websocket.DefaultDialer.Dial(wsu, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	t.Cleanup(func() { _ = contender.Close() })
	require.NoError(t, contender.WriteMessage(websocket.BinaryMessage, []byte{0x03, 0x04}))

	_ = contender.SetReadDeadline(time.Now().Add(5 * time.Second))
	sawBusy := false
	sawContenderDone := false
	for !sawContenderDone {
		_, raw, rerr := contender.ReadMessage()
		require.NoError(t, rerr)
		var msg wsMessage
		require.NoError(t, json.Unmarshal(raw, &msg))
		switch msg.Type {
		case wsMsgError:
			require.Equal(t, "stt_busy", msg.Code)
			sawBusy = true
		case wsMsgDone:
			sawContenderDone = true
		}
	}
	require.True(t, sawBusy, "contending audio stream must receive typed busy instead of killing the holder")

	sawHolderDone := false
	for !sawHolderDone {
		_, raw, rerr := holder.ReadMessage()
		require.NoError(t, rerr)
		var msg wsMessage
		require.NoError(t, json.Unmarshal(raw, &msg))
		switch msg.Type {
		case wsMsgError:
			t.Fatalf("active holder got unexpected error after contention: code=%s text=%s", msg.Code, msg.Text)
		case wsMsgSegmentFinal:
			holderSegments = append(holderSegments, msg.Text)
		case wsMsgDone:
			sawHolderDone = true
			_ = holder.WriteMessage(websocket.TextMessage, []byte(`{"type":"done"}`))
		}
	}
	require.Equal(t, []string{"holder head", "holder tail"}, holderSegments, "active holder must keep its committed tail after a contender is rejected")
}
