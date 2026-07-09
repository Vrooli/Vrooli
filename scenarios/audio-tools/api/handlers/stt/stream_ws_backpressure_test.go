package stt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/testutil/mocks"
)

// newWedgeKyutaiServer is a fake kyutai-stt resource that first floods a large
// burst of partial frames, then a single durable segment, then done. The burst
// is sized to overflow the OS/socket buffers so, if the relay ever stops
// draining the kyutai socket, the server BLOCKS mid-burst and never reaches the
// durable segment or done. doneWritten fires only if the server successfully
// wrote the terminal done — i.e. the relay kept draining despite a stalled
// browser consumer.
func newWedgeKyutaiServer(t *testing.T, partialCount, partialBytes int, segText string) (baseURL, wsEndpoint string, doneWritten <-chan struct{}) {
	t.Helper()
	done := make(chan struct{})
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
		// Read the start header; ignore everything else the relay sends.
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		payload := strings.Repeat("x", partialBytes)
		for i := 0; i < partialCount; i++ {
			frame := fmt.Sprintf(`{"type":"partial","text":"%s%d"}`, payload, i)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(frame)); err != nil {
				return // relay stopped draining -> server blocked/closed: wedge
			}
		}
		seg := fmt.Sprintf(`{"type":"segment","text":"%s","start_ms":0,"end_ms":100}`, segText)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(seg)); err != nil {
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"done"}`)); err != nil {
			return
		}
		close(done)
	})
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	return srv.URL, "ws" + strings.TrimPrefix(srv.URL, "http") + "/v1/stream", done
}

// kyutaiWedgeDeps wires a real Chain + Selector whose Local-tier "kyutai" engine
// is the KyutaiProvider pointed at the fake wedge server.
func kyutaiWedgeDeps(t *testing.T, baseURL, wsEndpoint string) Deps {
	t.Helper()
	kp := sttchain.NewKyutaiProvider(baseURL)
	kp.StreamEndpoint = wsEndpoint
	chain := sttchain.NewChain(sttchain.Options{
		EnableLocal:  true,
		LocalEngines: map[string]sttchain.Provider{"kyutai": kp},
	})
	return Deps{
		Chain:        chain,
		Selector:     sttpkg.NewSelector(chain),
		Logger:       &mocks.FakeLogger{},
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"auto","engine_id":"kyutai"}`},
	}
}

// TestStreamWS_StalledConsumerDoesNotWedgeKyutaiReader is the Phase 1 handler
// red oracle for the backpressure-wedge plan.
//
// Root cause: the relay chain (provider events(16) -> passthrough(unbuffered
// inner) -> egress -> handler events(16) -> blocking writeJSON) is fully
// synchronous. A browser consumer that stops reading back-pressures every hop
// until the provider stops reading the kyutai socket, which freezes the backend.
// Durable events queued behind the partial firehose (the committed segment and
// the terminal done) are then never delivered.
//
// Desired behaviour: partials are disposable (coalesced/dropped under pressure)
// while the relay ALWAYS drains the kyutai socket, so a stalled consumer can
// never stop the backend from making progress; the durable segment and done are
// still produced by the backend.
//
// The oracle stalls the browser consumer and asserts the fake backend still
// reaches its terminal done. It FAILS today (the server wedges mid-partial-burst
// and never writes done) and PASSES once the relay decouples drain-from-forward
// and coalesces partials.
func TestStreamWS_StalledConsumerDoesNotWedgeKyutaiReader(t *testing.T) {
	// ~60 MiB of partials: comfortably above this host's autotuned socket
	// buffers (tcp_rmem max 33 MiB + tcp_wmem max 4 MiB + the pipeline's ~32
	// in-flight slots), so a relay that stops draining the kyutai socket blocks
	// well before the durable segment/done rather than having the burst silently
	// absorbed by kernel buffers.
	baseURL, wsEndpoint, doneWritten := newWedgeKyutaiServer(t, 30000, 2048, "committed words")

	r := mux.NewRouter()
	r.Handle("/api/v1/voice/stream", StreamWSHandler(kyutaiWedgeDeps(t, baseURL, wsEndpoint))).Methods(http.MethodGet)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	wsu := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/voice/stream?language=en"
	c, resp, err := websocket.DefaultDialer.Dial(wsu, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	t.Cleanup(func() { _ = c.Close() })

	// The browser consumer STALLS: after connecting it never reads a frame,
	// modelling a consumer that cannot sustain the partial rate. A backpressure-
	// safe relay must still let the backend reach `done`.
	select {
	case <-doneWritten:
		// Relay kept draining kyutai despite the stalled consumer — fixed.
	case <-time.After(6 * time.Second):
		t.Fatal("kyutai reader wedged: a stalled browser consumer stopped the relay from draining the " +
			"kyutai socket, so the backend never reached its terminal done. Partials must be droppable and " +
			"the relay must always drain the backend socket.")
	}

	// Now that the consumer resumes, the durable segment + done must still be
	// delivered losslessly (partials may have been coalesced).
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	sawSegment, sawDone := false, false
	for !sawDone {
		_, raw, rerr := c.ReadMessage()
		if rerr != nil {
			break
		}
		var msg wsMessage
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}
		switch msg.Type {
		case wsMsgSegmentFinal:
			if msg.Text == "committed words" {
				sawSegment = true
			}
		case wsMsgDone:
			sawDone = true
		}
	}
	require.True(t, sawSegment, "the durable committed segment must be delivered to the consumer")
	require.True(t, sawDone, "the terminal done must be delivered to the consumer")
}
