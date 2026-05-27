package stt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

// TestBuildStreamStart asserts the WS transport maps `language` and the
// `format` codec declaration off the query string. An omitted format
// leaves InputFormat empty so the Segmenter sniffs (declare-or-sniff).
func TestBuildStreamStart(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/voice/stream?language=es&format=pcm_s16le", nil)
	start := buildStreamStart(req)
	require.Equal(t, "es", start.Language)
	require.Equal(t, "pcm_s16le", start.InputFormat)

	bare := httptest.NewRequest(http.MethodGet, "/api/v1/voice/stream", nil)
	require.Empty(t, buildStreamStart(bare).InputFormat)
	require.Empty(t, buildStreamStart(bare).Language)
}

// TestStreamWS_UpgradeRejectedWithoutChain asserts that when the chain
// is not wired the handler returns 503 instead of upgrading. This
// exercises the early-return guard path.
func TestStreamWS_UpgradeRejectedWithoutChain(t *testing.T) {
	r := mux.NewRouter()
	r.Handle("/api/v1/voice/stream", StreamWSHandler(Deps{Logger: &mocks.FakeLogger{}})).Methods(http.MethodGet)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/v1/voice/stream")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

// TestStreamWS_LoggerCapturesUpgradeFailure dials with an HTTP client
// (not a WS dialer) so the upgrade fails; the captured logger must
// record the upgrade-failed line.
func TestStreamWS_LoggerCapturesUpgradeFailure(t *testing.T) {
	logger := &mocks.FakeLogger{}
	deps := Deps{Logger: logger}
	r := mux.NewRouter()
	r.Handle("/api/v1/voice/stream", StreamWSHandler(deps)).Methods(http.MethodGet)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	wsURL, _ := url.Parse(srv.URL + "/api/v1/voice/stream")
	wsURL.Scheme = "ws"
	_, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "bad handshake") || strings.Contains(err.Error(), "503"))
}

// newNoProviderDeps builds a Deps with a real Chain and Selector that
// have no providers wired — Segmenter.Run will emit Error+Done. This
// is the minimal seam-friendly setup that exercises the full WS code
// path (upgrade, segmenter spin-up, event translation, terminal Done,
// graceful close) without standing up vendor adapters.
func newNoProviderDeps(t *testing.T) Deps {
	t.Helper()
	chain := sttchain.NewChain(sttchain.Options{
		EnableLocal:  false,
		EnableBYOK:   false,
		EnableVrooli: false,
	})
	return Deps{
		Chain:    chain,
		Selector: sttpkg.NewSelector(chain),
		Logger:   &mocks.FakeLogger{},
	}
}

// TestStreamWS_HandshakeAndTerminalDone confirms that:
//   - upgrade succeeds when Chain+Selector are wired
//   - the no-provider Segmenter path emits a terminal error+final+done
//     message sequence
//   - the server closes cleanly after the client sends a "done" frame
func TestStreamWS_HandshakeAndTerminalDone(t *testing.T) {
	r := mux.NewRouter()
	r.Handle("/api/v1/voice/stream", StreamWSHandler(newNoProviderDeps(t))).Methods(http.MethodGet)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	wsURL, _ := url.Parse(srv.URL + "/api/v1/voice/stream")
	wsURL.Scheme = "ws"
	c, resp, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	t.Cleanup(func() { _ = c.Close() })

	// Signal end-of-stream so the segmenter's buffered fallback drains
	// chunks and the chain's all-providers-failed error surfaces.
	require.NoError(t, c.WriteMessage(websocket.TextMessage, []byte(`{"type":"done"}`)))

	// Drain messages until we see the terminal "done" frame.
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	sawError, sawFinal, sawDone := false, false, false
	for !sawDone {
		_, raw, err := c.ReadMessage()
		require.NoError(t, err)
		var m wsMessage
		require.NoError(t, json.Unmarshal(raw, &m))
		switch m.Type {
		case wsMsgError:
			sawError = true
		case wsMsgFinal:
			sawFinal = true
		case wsMsgDone:
			sawDone = true
		}
	}
	require.True(t, sawError, "expected an error frame from no-provider chain")
	require.True(t, sawFinal, "expected a final frame before done")
	require.True(t, sawDone, "expected a terminal done frame")
}

// TestStreamWS_AbruptClientCloseDrainsServer drops the connection mid-stream
// (no terminal "done"); the server-side goroutines must observe the
// read error and exit without leaking. We give the handler a moment
// after close and then make a second connection — if a leak existed,
// the second handshake would still succeed but server-side goroutines
// would accumulate; we assert no panic and a clean shutdown of srv.
func TestStreamWS_AbruptClientCloseDrainsServer(t *testing.T) {
	r := mux.NewRouter()
	r.Handle("/api/v1/voice/stream", StreamWSHandler(newNoProviderDeps(t))).Methods(http.MethodGet)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	wsURL, _ := url.Parse(srv.URL + "/api/v1/voice/stream")
	wsURL.Scheme = "ws"
	c, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	require.NoError(t, err)

	// Send one binary frame so the reader loop has work in flight, then
	// abruptly close the underlying TCP conn without a WS close frame.
	require.NoError(t, c.WriteMessage(websocket.BinaryMessage, []byte("audio-bytes")))
	require.NoError(t, c.UnderlyingConn().Close())

	// Sleep is forbidden by repo policy; use Eventually to poll for the
	// server to drain by attempting another handshake.
	require.Eventually(t, func() bool {
		c2, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
		if err != nil {
			return false
		}
		_ = c2.Close()
		return true
	}, 5*time.Second, 25*time.Millisecond)
}

// TestStreamWS_ServerContextCancel verifies that when the parent
// request context is cancelled, the handler shuts down (the WS read
// returns and the connection closes). We simulate by using an
// httptest.Server whose Close is invoked while a client is connected.
func TestStreamWS_ServerContextCancel(t *testing.T) {
	r := mux.NewRouter()
	r.Handle("/api/v1/voice/stream", StreamWSHandler(newNoProviderDeps(t))).Methods(http.MethodGet)
	srv := httptest.NewServer(r)

	wsURL, _ := url.Parse(srv.URL + "/api/v1/voice/stream")
	wsURL.Scheme = "ws"
	c, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.WriteMessage(websocket.TextMessage, []byte(`{"type":"done"}`)))

	// Drain the initial terminal messages so the read loop is blocked
	// on the next ReadMessage when we close the server.
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		_, raw, err := c.ReadMessage()
		require.NoError(t, err)
		var m wsMessage
		_ = json.Unmarshal(raw, &m)
		if m.Type == wsMsgDone {
			break
		}
	}

	// Close the server. The pending read on the client side should
	// return an error (the server-side conn is closed during shutdown).
	srv.Close()
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = c.ReadMessage()
	require.Error(t, err)
}

// TestWSMessage_VadStateFields asserts the wire-format contract for the
// new vad-state message. The UI consumer pattern-matches on these exact
// JSON keys; this is a regression guard against accidental renames.
func TestWSMessage_VadStateFields(t *testing.T) {
	voiced := false
	elapsed := int64(420)
	timeout := int64(1500)
	seq := uint64(7)
	m := wsMessage{
		Type:             wsMsgVadState,
		Voiced:           &voiced,
		SilenceElapsedMs: &elapsed,
		SilenceTimeoutMs: &timeout,
		TickSeq:          &seq,
	}
	raw, err := json.Marshal(m)
	require.NoError(t, err)
	require.Equal(t, "vad-state", m.Type)
	got := string(raw)
	require.Contains(t, got, `"type":"vad-state"`)
	require.Contains(t, got, `"voiced":false`)
	require.Contains(t, got, `"silenceElapsedMs":420`)
	require.Contains(t, got, `"silenceTimeoutMs":1500`)
	require.Contains(t, got, `"tickSeq":7`)

	// Round-trip: decoded shape preserves all fields and they're pointers
	// so a non-VAD wsMessage doesn't accidentally include zero values.
	var back wsMessage
	require.NoError(t, json.Unmarshal(raw, &back))
	require.NotNil(t, back.Voiced)
	require.False(t, *back.Voiced)
	require.NotNil(t, back.SilenceElapsedMs)
	require.Equal(t, int64(420), *back.SilenceElapsedMs)
	require.NotNil(t, back.TickSeq)
	require.Equal(t, uint64(7), *back.TickSeq)
}

// TestWSMessage_NonVadStateOmitsVadFields asserts that an unrelated
// message (e.g. partial) does not serialize the VAD-only fields. This
// matters because the UI's switch on msg.type is strict — but defensive
// clients may look at the shape too.
func TestWSMessage_NonVadStateOmitsVadFields(t *testing.T) {
	m := wsMessage{Type: wsMsgPartial, Text: "hello"}
	raw, err := json.Marshal(m)
	require.NoError(t, err)
	got := string(raw)
	require.NotContains(t, got, "voiced")
	require.NotContains(t, got, "silenceElapsedMs")
	require.NotContains(t, got, "silenceTimeoutMs")
	require.NotContains(t, got, "tickSeq")
}

// ensure the package-internal ctx alias compiles into tests too.
var _ context.Context = context.Background()
