package sttchain_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
)

// newStalledReaderServer models a kyutai socket whose consumer stops reading
// after the start header: it upgrades, reads exactly the start frame, then
// holds the connection OPEN without reading any further frames. Once the OS
// socket buffer fills, the provider's next blocking WriteMessage stalls. The
// server closes only when its request context is cancelled (srv.Close()).
func newStalledReaderServer() *httptest.Server {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Read only the start header, then stop reading entirely so the
		// client's writes back-pressure. Keep the conn open (block on the
		// request context) so writes stall rather than erroring on close.
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		<-r.Context().Done()
	})
	return httptest.NewServer(h)
}

// TestKyutaiProvider_CancelUnderWriteBackpressureCompletesTeardown is the
// Phase 1 provider red oracle for the backpressure-wedge plan.
//
// Root cause (provider_kyutai.go): writeFrame holds writeMu across a blocking
// conn.WriteMessage. When the consumer stops reading, the pump's binary write
// stalls WHILE HOLDING writeMu; on ctx cancel the watcher's sendEnd() can never
// acquire writeMu, so the drain-then-close teardown never runs and the events
// channel is never closed — teardown hangs forever.
//
// Desired behaviour (the fix): decode/send are decoupled so a stalled consumer
// can never freeze teardown — cancelling the session always completes the
// drain-then-close within the bounded window and closes the events channel.
//
// This test FAILS against unmodified code (the events channel never closes; the
// bounded wait below fires) and PASSES once the writeMu-across-blocking-write
// coupling is retired.
func TestKyutaiProvider_CancelUnderWriteBackpressureCompletesTeardown(t *testing.T) {
	srv := newStalledReaderServer()
	defer srv.Close()

	p := sttchain.NewKyutaiProvider("http://example.invalid")
	p.StreamEndpoint = wsURL(srv.URL)

	chunks := make(chan sttchain.AudioChunk) // unbuffered, left OPEN (cancel path)
	ctx, cancel := context.WithCancel(context.Background())
	events, err := p.TranscribeStreaming(ctx, sttchain.StreamStart{Language: "en"}, chunks)
	require.NoError(t, err)

	// Feed a payload large enough that a single WriteMessage exceeds the OS
	// socket buffer and blocks against the non-reading server, so the pump
	// stalls mid-write while holding writeMu.
	big := make([]byte, 8<<20) // 8 MiB
	chunks <- sttchain.AudioChunk{Audio: big}

	// Let the pump enter the blocked 8 MiB write while holding writeMu before
	// we cancel — this pins the deadlock ordering (pump holds the lock, the
	// cancel watcher's sendEnd then blocks acquiring it) rather than racing it.
	time.Sleep(300 * time.Millisecond)
	cancel()

	// Drain the events channel. The contract: cancelling always completes
	// teardown (channel closes) within a bounded window. kyutaiDrainTimeout is
	// 5s; allow generous margin. Today this never closes -> the timer fires and
	// the oracle is RED.
	drained := make(chan struct{})
	go func() {
		for range events { //nolint:revive // drain to completion
		}
		close(drained)
	}()

	select {
	case <-drained:
		// teardown completed — the fixed behaviour.
	case <-time.After(9 * time.Second):
		t.Fatal("teardown wedged: events channel never closed after cancel under write backpressure " +
			"(writeMu held across a blocking write — the provider must not couple decode/send this way)")
	}

	// Sanity: the endpoint was the stalled server (guards against a mis-dial
	// making the test trivially green).
	require.True(t, strings.HasPrefix(p.StreamEndpoint, "ws://"))
}
