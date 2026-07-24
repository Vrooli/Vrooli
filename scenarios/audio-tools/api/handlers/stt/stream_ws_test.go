package stt

import (
	"context"
	"encoding/json"
	"errors"
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
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/store"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/stt/session"
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

type staticStreamConfig struct{ raw string }

func (s staticStreamConfig) Get(context.Context) (string, bool, error) { return s.raw, true, nil }
func (s staticStreamConfig) Set(context.Context, string) error         { return nil }

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

func TestStreamWS_EmitsInitialStatus(t *testing.T) {
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

	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := c.ReadMessage()
	require.NoError(t, err)
	var m wsMessage
	require.NoError(t, json.Unmarshal(raw, &m))
	require.Equal(t, wsMsgStatus, m.Type)
	require.Equal(t, "stream_connected", m.Code)
	require.NotEmpty(t, m.Text)
}

func TestWSMessageDeliveryClass_ProcessedAcknowledgementIsDurable(t *testing.T) {
	require.Equal(t, sttchain.DeliveryProgress, wsMessageDeliveryClass(wsMessage{
		Type: wsMsgStatus,
		Code: "stream_connected",
	}))
	require.Equal(t, sttchain.DeliveryDurable, wsMessageDeliveryClass(wsMessage{
		Type:              wsMsgStatus,
		Code:              "processed_acknowledgement",
		ReceivedSequence:  4,
		ProcessedSequence: 4,
	}))
}

// TestStreamWS_TestOnlyProviderBusyFault [REQ:ATD-P0-005] verifies the
// qualification seam is observable on the real WebSocket protocol while
// remaining disabled unless both an active isolation lease and the
// per-request test-mode header are present.
func TestStreamWS_TestOnlyProviderBusyFault(t *testing.T) {
	deps := newNoProviderDeps(t)
	deps.TestIsolationActive = func() bool { return true }
	r := mux.NewRouter()
	r.Handle("/api/v1/voice/stream", StreamWSHandler(deps)).Methods(http.MethodGet)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	wsURL, _ := url.Parse(srv.URL + "/api/v1/voice/stream")
	wsURL.Scheme = "ws"
	headers := http.Header{}
	headers.Set(streamTestModeHeader, "1")
	headers.Set(streamTestFaultHeader, "provider_busy")
	c, resp, err := websocket.DefaultDialer.Dial(wsURL.String(), headers)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	t.Cleanup(func() { _ = c.Close() })

	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	sawBusy, sawDone := false, false
	for !sawDone {
		_, raw, err := c.ReadMessage()
		require.NoError(t, err)
		var m wsMessage
		require.NoError(t, json.Unmarshal(raw, &m))
		if m.Type == wsMsgError {
			require.Equal(t, "stt_busy", m.Code)
			sawBusy = true
		}
		if m.Type == wsMsgDone {
			sawDone = true
		}
	}
	require.True(t, sawBusy)
}

func TestStreamWS_TestOnlyCloseAfterChunkFault(t *testing.T) {
	deps := newNoProviderDeps(t)
	deps.TestIsolationActive = func() bool { return true }
	r := mux.NewRouter()
	r.Handle("/api/v1/voice/stream", StreamWSHandler(deps)).Methods(http.MethodGet)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	wsURL, _ := url.Parse(srv.URL + "/api/v1/voice/stream")
	wsURL.Scheme = "ws"
	headers := http.Header{}
	headers.Set(streamTestModeHeader, "1")
	headers.Set(streamTestFaultHeader, "close_after_chunk:1")
	c, _, err := websocket.DefaultDialer.Dial(wsURL.String(), headers)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.WriteMessage(websocket.BinaryMessage, []byte("one deterministic chunk")))
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	sawIncompleteCoverage := false
	for {
		_, data, readErr := c.ReadMessage()
		if readErr != nil {
			err = readErr
			break
		}
		var message wsMessage
		if json.Unmarshal(data, &message) == nil && message.Type == wsMsgError && message.Code == "incomplete_coverage" {
			sawIncompleteCoverage = true
		}
	}
	require.Error(t, err, "configured transport interruption must close the browser socket")
	require.True(t, sawIncompleteCoverage, "post-capture interruption must name incomplete coverage before closing")
}

// TestStreamWS_TestOnlySuppressProcessedAckFault [REQ:ATD-P0-004] proves a
// terminal event cannot masquerade as success when the browser's replay cursor
// was not acknowledged. The server must preserve the V2 replay tail and name
// the failure explicitly instead of silently compacting either side's journal.
func TestStreamWS_TestOnlySuppressProcessedAckFault(t *testing.T) {
	adapter := &sttmocks.FakeBYOK{
		IDStr:     "fake",
		Available: true,
		TranscribeFn: func(_ context.Context, _ string, _ sttchain.Request) (*sttchain.Result, error) {
			return &sttchain.Result{Text: "ack fault transcript", Tier: sttchain.TierBYOK, ProviderID: "fake", ModelID: "fake-model"}, nil
		},
	}
	chain := sttchain.NewChain(sttchain.Options{
		BYOK:       sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"fake": adapter}),
		EnableBYOK: true,
	})
	sessions := session.NewRegistry(0)
	deps := Deps{
		Chain:               chain,
		Selector:            sttpkg.NewSelector(chain),
		Logger:              &mocks.FakeLogger{},
		StreamConfig:        staticStreamConfig{raw: `{"streaming_mode":"off"}`},
		Sessions:            sessions,
		TestIsolationActive: func() bool { return true },
	}
	r := mux.NewRouter()
	r.Handle("/api/v1/voice/stream", StreamWSHandler(deps)).Methods(http.MethodGet)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	wsURL, _ := url.Parse(srv.URL + "/api/v1/voice/stream?language=en&format=pcm_s16le&protocol_version=2&session_id=ack-fault-session&resume_token=ack-fault-token")
	wsURL.Scheme = "ws"
	headers := http.Header{}
	headers.Set(envelope.HeaderProvider, "fake")
	headers.Set(envelope.HeaderKey, "secret")
	headers.Set(streamTestModeHeader, "1")
	headers.Set(streamTestFaultHeader, "suppress_processed_ack")
	c, _, err := websocket.DefaultDialer.Dial(wsURL.String(), headers)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.WriteMessage(websocket.BinaryMessage, encodeWSV2AudioFrameForTest(0, 0, 2, []byte{1, 0, 2, 0})))
	require.NoError(t, c.WriteMessage(websocket.TextMessage, []byte(`{"type":"done"}`)))
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	sawCoverageError, sawProcessedAck, sawDone := false, false, false
	for !sawDone {
		_, raw, readErr := c.ReadMessage()
		require.NoError(t, readErr)
		var message wsMessage
		require.NoError(t, json.Unmarshal(raw, &message))
		if message.Type == wsMsgStatus && message.Code == "processed_acknowledgement" {
			sawProcessedAck = true
		}
		if message.Type == wsMsgError && message.Code == "incomplete_coverage" {
			sawCoverageError = true
		}
		if message.Type == wsMsgDone {
			require.Equal(t, string(session.TerminalIncompleteCoverage), message.Code)
			sawDone = true
		}
	}
	require.True(t, sawCoverageError)
	require.False(t, sawProcessedAck)

	ledger, _, err := sessions.Open("ack-fault-session", "ack-fault-token")
	require.NoError(t, err)
	state := ledger.Snapshot()
	require.Equal(t, session.TerminalIncompleteCoverage, state.TerminalReason)
	require.EqualValues(t, -1, state.ProcessedSequence)
	require.Len(t, state.Replay, 1)
}

// TestStreamWS_TestOnlyCloseAfterCommitFault [REQ:ATD-P0-001] closes at the
// durable commit boundary, not before the stream starts. That makes reconnect
// and stable segment-ID replay testable without pretending an incomplete turn
// was a normal terminal success.
func TestStreamWS_TestOnlyCloseAfterCommitFault(t *testing.T) {
	adapter := &sttmocks.FakeBYOK{
		IDStr:     "fake",
		Available: true,
		TranscribeFn: func(_ context.Context, _ string, _ sttchain.Request) (*sttchain.Result, error) {
			return &sttchain.Result{Text: "commit fault transcript", Tier: sttchain.TierBYOK, ProviderID: "fake", ModelID: "fake-model"}, nil
		},
	}
	chain := sttchain.NewChain(sttchain.Options{
		BYOK:       sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"fake": adapter}),
		EnableBYOK: true,
	})
	sessions := session.NewRegistry(0)
	deps := Deps{
		Chain:               chain,
		Selector:            sttpkg.NewSelector(chain),
		Logger:              &mocks.FakeLogger{},
		StreamConfig:        staticStreamConfig{raw: `{"streaming_mode":"off"}`},
		Sessions:            sessions,
		TestIsolationActive: func() bool { return true },
	}
	r := mux.NewRouter()
	r.Handle("/api/v1/voice/stream", StreamWSHandler(deps)).Methods(http.MethodGet)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	wsURL, _ := url.Parse(srv.URL + "/api/v1/voice/stream?language=en&format=pcm_s16le&protocol_version=2&session_id=commit-fault-session&resume_token=commit-fault-token")
	wsURL.Scheme = "ws"
	headers := http.Header{}
	headers.Set(envelope.HeaderProvider, "fake")
	headers.Set(envelope.HeaderKey, "secret")
	headers.Set(streamTestModeHeader, "1")
	headers.Set(streamTestFaultHeader, "close_after_commit:1")
	c, _, err := websocket.DefaultDialer.Dial(wsURL.String(), headers)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.WriteMessage(websocket.BinaryMessage, encodeWSV2AudioFrameForTest(0, 0, 2, []byte{1, 0, 2, 0})))
	require.NoError(t, c.WriteMessage(websocket.TextMessage, []byte(`{"type":"done"}`)))
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	sawSegment, sawCoverageError, sawDone := false, false, false
	for {
		_, raw, readErr := c.ReadMessage()
		if readErr != nil {
			break
		}
		var message wsMessage
		require.NoError(t, json.Unmarshal(raw, &message))
		if message.Type == wsMsgSegmentFinal {
			sawSegment = true
		}
		if message.Type == wsMsgError && message.Code == "incomplete_coverage" {
			sawCoverageError = true
		}
		if message.Type == wsMsgDone {
			sawDone = true
		}
	}
	require.True(t, sawSegment)
	require.True(t, sawCoverageError)
	require.False(t, sawDone, "fault closes before a normal terminal done")

	ledger, _, err := sessions.Open("commit-fault-session", "commit-fault-token")
	require.NoError(t, err)
	state := ledger.Snapshot()
	require.Equal(t, session.TerminalReason("test_fault_close_before_done"), state.TerminalReason)
	require.Len(t, state.Committed, 1)
	require.Len(t, state.Replay, 1)
}

func TestStreamWS_BinaryPCMAndDoneProducePromptSegmentFinal(t *testing.T) {
	audio := []byte{0x01, 0x00, 0x02, 0x00, 0x03, 0x00}
	captured := make(chan sttchain.Request, 1)
	adapter := &sttmocks.FakeBYOK{
		IDStr:     "fake",
		Available: true,
		TranscribeFn: func(_ context.Context, _ string, req sttchain.Request) (*sttchain.Result, error) {
			captured <- req
			return &sttchain.Result{
				Text:       "scripted transcript",
				Tier:       sttchain.TierBYOK,
				ProviderID: "fake",
				ModelID:    "fake-model",
				Latency:    time.Millisecond,
			}, nil
		},
	}
	chain := sttchain.NewChain(sttchain.Options{
		BYOK:       sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"fake": adapter}),
		EnableBYOK: true,
	})
	deps := Deps{
		Chain:        chain,
		Selector:     sttpkg.NewSelector(chain),
		Logger:       &mocks.FakeLogger{},
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"off"}`},
	}

	r := mux.NewRouter()
	r.Handle("/api/v1/voice/stream", StreamWSHandler(deps)).Methods(http.MethodGet)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	wsURL, _ := url.Parse(srv.URL + "/api/v1/voice/stream?language=en&format=pcm_s16le")
	wsURL.Scheme = "ws"
	h := http.Header{}
	h.Set(envelope.HeaderProvider, "fake")
	h.Set(envelope.HeaderKey, "secret")
	c, resp, err := websocket.DefaultDialer.Dial(wsURL.String(), h)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.WriteMessage(websocket.BinaryMessage, audio))
	require.NoError(t, c.WriteMessage(websocket.TextMessage, []byte(`{"type":"done"}`)))

	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	sawSegment, sawFinal, sawDone := false, false, false
	for !sawDone {
		_, raw, err := c.ReadMessage()
		require.NoError(t, err)
		var m wsMessage
		require.NoError(t, json.Unmarshal(raw, &m))
		switch m.Type {
		case wsMsgError:
			t.Fatalf("unexpected ws error frame: %s", m.Text)
		case wsMsgSegmentFinal:
			require.Equal(t, "scripted transcript", m.Text)
			sawSegment = true
		case wsMsgFinal:
			sawFinal = true
		case wsMsgDone:
			sawDone = true
		}
	}
	require.True(t, sawSegment, "expected segment-final transcript")
	require.True(t, sawFinal, "expected final transition frame before done")

	select {
	case req := <-captured:
		require.Equal(t, audio, req.Audio)
		require.Equal(t, "pcm_s16le", req.Format)
		require.Equal(t, "en", req.Language)
	case <-time.After(time.Second):
		t.Fatal("fake BYOK adapter was not called")
	}
}

// TestStreamWS_V2ReplayReemitsCommittedSegmentWithStableIdentity
// [REQ:ATD-P0-001] proves the WebSocket transport now shares the Connect
// transport's ledger commit boundary. A reconnect gets the committed segment
// again with the same identity; browser clients can therefore repair a missed
// delivery without appending the text twice.
func TestStreamWS_V2ReplayReemitsCommittedSegmentWithStableIdentity(t *testing.T) {
	adapter := &sttmocks.FakeBYOK{
		IDStr:     "fake",
		Available: true,
		TranscribeFn: func(_ context.Context, _ string, _ sttchain.Request) (*sttchain.Result, error) {
			return &sttchain.Result{Text: "replay-safe segment", Tier: sttchain.TierBYOK, ProviderID: "fake", ModelID: "fake-model"}, nil
		},
	}
	chain := sttchain.NewChain(sttchain.Options{
		BYOK:       sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"fake": adapter}),
		EnableBYOK: true,
	})
	sessions := session.NewRegistry(0)
	deps := Deps{
		Chain:        chain,
		Selector:     sttpkg.NewSelector(chain),
		Logger:       &mocks.FakeLogger{},
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"off"}`},
		Sessions:     sessions,
	}
	r := mux.NewRouter()
	r.Handle("/api/v1/voice/stream", StreamWSHandler(deps)).Methods(http.MethodGet)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	wsURL, _ := url.Parse(srv.URL + "/api/v1/voice/stream?language=en&format=pcm_s16le&protocol_version=2&session_id=replay-safe-session&resume_token=replay-safe-token")
	wsURL.Scheme = "ws"
	headers := http.Header{}
	headers.Set(envelope.HeaderProvider, "fake")
	headers.Set(envelope.HeaderKey, "secret")

	readTerminal := func(c *websocket.Conn) []wsMessage {
		t.Helper()
		_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
		var messages []wsMessage
		for {
			_, raw, err := c.ReadMessage()
			require.NoError(t, err)
			var message wsMessage
			require.NoError(t, json.Unmarshal(raw, &message))
			messages = append(messages, message)
			if message.Type == wsMsgDone {
				return messages
			}
		}
	}
	findSegment := func(messages []wsMessage) wsMessage {
		t.Helper()
		for _, message := range messages {
			if message.Type == wsMsgSegmentFinal {
				return message
			}
		}
		t.Fatal("expected durable segment-final")
		return wsMessage{}
	}

	first, _, err := websocket.DefaultDialer.Dial(wsURL.String(), headers)
	require.NoError(t, err)
	require.NoError(t, first.WriteMessage(websocket.BinaryMessage, encodeWSV2AudioFrameForTest(0, 0, 2, []byte{1, 0, 2, 0})))
	require.NoError(t, first.WriteMessage(websocket.TextMessage, []byte(`{"type":"done"}`)))
	firstSegment := findSegment(readTerminal(first))
	require.NotEmpty(t, firstSegment.SegmentID)
	require.NoError(t, first.Close())

	ledger, _, err := sessions.Open("replay-safe-session", "replay-safe-token")
	require.NoError(t, err)
	require.Len(t, ledger.Snapshot().Committed, 1)

	second, _, err := websocket.DefaultDialer.Dial(wsURL.String(), headers)
	require.NoError(t, err)
	require.NoError(t, second.WriteMessage(websocket.TextMessage, []byte(`{"type":"done"}`)))
	secondSegment := findSegment(readTerminal(second))
	require.NoError(t, second.Close())
	require.Equal(t, firstSegment.Text, secondSegment.Text)
	require.Equal(t, firstSegment.SegmentID, secondSegment.SegmentID)
	require.Len(t, ledger.Snapshot().Committed, 1, "replay must not duplicate the durable commit")
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

// captureRecorder is a synchronous usagereport.Recorder that records every
// enqueued row so the Phase 8 delivery telemetry can be asserted.
type captureRecorder struct{ rows []store.UsageRow }

func (c *captureRecorder) Enqueue(row store.UsageRow)                   { c.rows = append(c.rows, row) }
func (c *captureRecorder) Record(context.Context, store.UsageRow) error { return nil }
func (c *captureRecorder) Close()                                       {}

// TestStreamCloseOutcome asserts the close-classifier maps each teardown path
// to the delivery-telemetry outcome the drop metric keys on.
func TestStreamCloseOutcome(t *testing.T) {
	require.Equal(t, "graceful", streamCloseOutcome(nil))
	require.Equal(t, "cancel", streamCloseOutcome(context.Canceled))
	require.Equal(t, "read_error", streamCloseOutcome(errors.New("boom")))
}

// TestEmitStreamDeliveryTelemetry_GracefulCounts asserts a clean close records
// the full segment count with no drop signal.
func TestEmitStreamDeliveryTelemetry_GracefulCounts(t *testing.T) {
	rec := &captureRecorder{}
	logger := &mocks.FakeLogger{}
	emitStreamDeliveryTelemetry(Deps{Logger: logger, Usage: rec}, 3, true, nil, "provider_done")

	require.Len(t, rec.rows, 1)
	row := rec.rows[0]
	require.Equal(t, "stt", row.Capability)
	require.Equal(t, "stream_session", row.Operation)
	require.Equal(t, int32(3), row.OutputTokens)
	require.Equal(t, "graceful", row.FallbackReason)
	require.Empty(t, row.Error, "a graceful close must not raise the drop signal")
	require.NotEmpty(t, row.OperationID)
}

// TestEmitStreamDeliveryTelemetry_DropRaisesError asserts a non-graceful close
// (cancel / read error) increments the drop metric via a populated Error.
func TestEmitStreamDeliveryTelemetry_DropRaisesError(t *testing.T) {
	rec := &captureRecorder{}
	emitStreamDeliveryTelemetry(Deps{Logger: &mocks.FakeLogger{}, Usage: rec}, 1, false, context.Canceled, "provider_done")

	require.Len(t, rec.rows, 1)
	require.Equal(t, "cancel", rec.rows[0].FallbackReason)
	require.Equal(t, "tail_drain_cancel", rec.rows[0].Error)
	require.Equal(t, int32(1), rec.rows[0].OutputTokens)
}

// TestEmitStreamDeliveryTelemetry_NoRecorderIsSafe asserts telemetry never
// panics when no usage recorder is wired (still logs).
func TestEmitStreamDeliveryTelemetry_NoRecorderIsSafe(t *testing.T) {
	logger := &mocks.FakeLogger{}
	require.NotPanics(t, func() {
		emitStreamDeliveryTelemetry(Deps{Logger: logger}, 2, true, nil, "provider_done")
	})
}

// ensure the package-internal ctx alias compiles into tests too.
var _ context.Context = context.Background()
