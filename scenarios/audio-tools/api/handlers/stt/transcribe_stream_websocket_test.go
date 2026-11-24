package stt

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/logx"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/stt/segmenter/testaudio"
	"audio-tools/internal/stt/session"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/schedule"
	"google.golang.org/protobuf/proto"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

func TestBrowserTranscribeStreamWebSocketUsesV2AdmissionAndDone(t *testing.T) {
	deps := Deps{
		Chain:        sttchain.NewChain(sttchain.Options{}),
		Selector:     sttpkg.NewSelector(newFakeBatchExecutor("hello")),
		Sessions:     session.NewRegistry(0),
		Logger:       logx.Std{},
		Clock:        schedule.System(),
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"off"}`},
	}
	h := NewConnectHandler(deps)
	router := mux.NewRouter()
	router.HandleFunc(BrowserTranscribeStreamWebSocketPath, h.serveTranscribeStreamWebSocket).Methods(http.MethodGet)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	conn, _, err := websocket.DefaultDialer.Dial("ws"+server.URL[len("http"):]+BrowserTranscribeStreamWebSocketPath, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	start := &sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_Start{Start: &sttv1.StreamStart{
		ProtocolVersion: 2, SessionId: "ws-session", ResumeToken: "ws-token",
	}}}
	writeProtoFrame(t, conn, start)
	accepted := readProtoEvent(t, conn)
	require.Equal(t, "accepted", accepted.GetSessionStatus().GetState())

	end := &sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_End{End: &sttv1.StreamEnd{}}}
	writeProtoFrame(t, conn, end)
	var sawDone bool
	for !sawDone {
		event := readProtoEvent(t, conn)
		if event.GetDone() != nil {
			sawDone = true
		}
	}
}

func TestBrowserTranscribeStreamWebSocketRejectsCrossOrigin(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc(BrowserTranscribeStreamWebSocketPath, NewConnectHandler(Deps{Logger: logx.Std{}, Clock: schedule.System()}).serveTranscribeStreamWebSocket).Methods(http.MethodGet)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	dialer := *websocket.DefaultDialer
	headers := http.Header{"Origin": []string{"https://attacker.example"}}
	conn, response, err := dialer.Dial("ws"+server.URL[len("http"):]+BrowserTranscribeStreamWebSocketPath, headers)
	if conn != nil {
		_ = conn.Close()
	}
	require.Error(t, err)
	require.NotNil(t, response)
	require.Equal(t, http.StatusForbidden, response.StatusCode)
}

func TestBrowserTranscribeStreamWebSocketCloseIsNotCleanInputEOF(t *testing.T) {
	result := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		conn, err := transcribeStreamWebSocketUpgrader.Upgrade(w, req, nil)
		if err != nil {
			result <- err
			return
		}
		defer conn.Close()
		stream := &webSocketTranscribeStream{conn: conn}
		_, err = stream.Receive()
		result <- err
	}))
	t.Cleanup(server.Close)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+server.URL[len("http"):], nil)
	require.NoError(t, err)
	require.NoError(t, conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)))
	_ = conn.Close()
	require.ErrorIs(t, <-result, io.ErrUnexpectedEOF)
}

func TestBrowserTranscribeStreamWebSocketResumesInterruptedInput(t *testing.T) {
	sessions := session.NewRegistry(0)
	var recoveredAudio [][]byte
	var recoveredMu sync.Mutex
	executor := newFakeBatchExecutor("recovered transcript")
	executor.ExecuteFn = func(_ context.Context, req sttchain.Request) (*sttchain.Result, error) {
		recoveredMu.Lock()
		defer recoveredMu.Unlock()
		recoveredAudio = append(recoveredAudio, append([]byte(nil), req.Audio...))
		return executor.Result, nil
	}
	h := NewConnectHandler(Deps{
		Chain:        sttchain.NewChain(sttchain.Options{}),
		Selector:     sttpkg.NewSelector(executor),
		Sessions:     sessions,
		Logger:       logx.Std{},
		Clock:        schedule.System(),
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"off"}`},
	})
	firstClosed := make(chan struct{})
	var requestCount int
	var countMu sync.Mutex
	router := mux.NewRouter()
	router.HandleFunc(BrowserTranscribeStreamWebSocketPath, func(w http.ResponseWriter, req *http.Request) {
		countMu.Lock()
		requestCount++
		ordinal := requestCount
		countMu.Unlock()
		h.serveTranscribeStreamWebSocket(w, req)
		if ordinal == 1 {
			close(firstClosed)
		}
	}).Methods(http.MethodGet)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	url := "ws" + server.URL[len("http"):] + BrowserTranscribeStreamWebSocketPath
	open := func() *websocket.Conn {
		t.Helper()
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		require.NoError(t, err)
		return conn
	}
	audio := testaudio.SpeechTonePauseTone3s()

	first := open()
	writeProtoFrame(t, first, startMsg("resume-interrupted", "resume-token"))
	require.Equal(t, "accepted", readProtoEvent(t, first).GetSessionStatus().GetState())
	writeProtoFrame(t, first, chunkMsg(0, audio))
	_ = first.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "interrupted"), time.Now().Add(time.Second))
	_ = first.Close()
	select {
	case <-firstClosed:
	case <-time.After(3 * time.Second):
		t.Fatal("interrupted stream handler did not return")
	}
	ledger, resumed, err := sessions.Open("resume-interrupted", "resume-token")
	require.NoError(t, err)
	require.True(t, resumed)
	state := ledger.Snapshot()
	require.Equal(t, int64(0), state.ReceivedSequence)
	require.Equal(t, session.TerminalNone, state.TerminalReason, fmt.Sprintf("disconnect must remain resumable: %+v", state))
	require.Empty(t, state.Committed, "input without StreamEnd must not create a final commit")

	second := open()
	defer second.Close()
	writeProtoFrame(t, second, startMsg("resume-interrupted", "resume-token"))
	require.Equal(t, "accepted", readProtoEvent(t, second).GetSessionStatus().GetState())
	// The server must recover its retained interval before accepting traffic.
	// A client retry remains legal and must be deduplicated rather than fed twice.
	writeProtoFrame(t, second, chunkMsg(0, audio))
	writeProtoFrame(t, second, &sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_End{End: &sttv1.StreamEnd{}}})
	var done *sttv1.StreamDone
	for done == nil {
		event := readProtoEvent(t, second)
		done = event.GetDone()
	}
	state = ledger.Snapshot()
	require.Equal(t, session.TerminalCompleted, state.TerminalReason)
	require.Equal(t, int64(0), state.ProcessedSequence)
	require.Len(t, state.Committed, 1)
	require.Equal(t, "recovered transcript", done.GetFinalText())
	recoveredMu.Lock()
	defer recoveredMu.Unlock()
	require.Len(t, recoveredAudio, 2, "the interrupted attempt and its recovery retry must each reach recognition once")
	require.True(t, bytes.Equal(audio, recoveredAudio[0]), "the interrupted attempt must receive the complete interval")
	require.True(t, bytes.Equal(audio, recoveredAudio[1]), "resume must recover the complete interval without doubling a client retry")
}

func TestBrowserTranscribeStreamWebSocketBackfillsCommittedSegmentBeforeAudio(t *testing.T) {
	const sessionID = "ws-backfill-session"
	const resumeToken = "ws-backfill-token"
	sessions := session.NewRegistry(0)
	ledger, resumed, err := sessions.Open(sessionID, resumeToken)
	require.NoError(t, err)
	require.False(t, resumed)
	_, err = ledger.Receive(session.Chunk{Sequence: 0, StartSample: 0, EndSample: 1600, Audio: []byte{1, 2}})
	require.NoError(t, err)
	require.NoError(t, ledger.AcknowledgeProcessed(0)) // compacts the producing PCM interval.
	_, err = ledger.Commit(session.Segment{ID: "stable-ws-segment", Text: "committed before disconnect", StartSample: 320, EndSample: 1600})
	require.NoError(t, err)
	before := ledger.Snapshot()
	require.Empty(t, before.Replay)
	require.Equal(t, int64(0), before.ProcessedSequence)

	h := NewConnectHandler(Deps{
		Chain: sttchain.NewChain(sttchain.Options{}), Selector: sttpkg.NewSelector(newFakeBatchExecutor("must not run")),
		Sessions: sessions, Logger: logx.Std{}, Clock: schedule.System(),
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"off"}`},
	})
	returned := make(chan struct{})
	router := mux.NewRouter()
	router.HandleFunc(BrowserTranscribeStreamWebSocketPath, func(w http.ResponseWriter, req *http.Request) {
		h.serveTranscribeStreamWebSocket(w, req)
		close(returned)
	}).Methods(http.MethodGet)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	url := "ws" + server.URL[len("http"):] + BrowserTranscribeStreamWebSocketPath
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	start := startMsg(sessionID, resumeToken)
	start.GetStart().Generation = 9
	writeProtoFrame(t, conn, start)
	accepted := readProtoEvent(t, conn)
	require.Equal(t, "accepted", accepted.GetSessionStatus().GetState())

	// No new PCM is sent: the next event must be backfilled from the durable
	// ledger, proving that acknowledged/compacted audio cannot strand a commit.
	replayed := readProtoEvent(t, conn)
	segment := replayed.GetSegment()
	require.NotNil(t, segment)
	require.Equal(t, "stable-ws-segment", segment.GetSegmentId())
	require.Equal(t, "committed before disconnect", segment.GetText())
	require.Equal(t, int64(320), segment.GetStartSample())
	require.Equal(t, int64(1600), segment.GetEndSample())
	require.Equal(t, uint64(9), segment.GetGeneration())
	after := ledger.Snapshot()
	require.Equal(t, before.ReceivedSequence, after.ReceivedSequence)
	require.Equal(t, before.ProcessedSequence, after.ProcessedSequence)
	require.Equal(t, before.Replay, after.Replay)
	require.Equal(t, before.Committed, after.Committed, "backfill must not append a duplicate commit")

	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "done"), time.Now().Add(time.Second))
	_ = conn.Close()
	select {
	case <-returned:
	case <-time.After(3 * time.Second):
		t.Fatal("resumed WebSocket handler did not return")
	}
}

func writeProtoFrame(t *testing.T, conn *websocket.Conn, request *sttv1.TranscribeStreamRequest) {
	t.Helper()
	data, err := proto.Marshal(request)
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, data))
}

func readProtoEvent(t *testing.T, conn *websocket.Conn) *sttv1.TranscribeStreamEvent {
	t.Helper()
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)
	event := new(sttv1.TranscribeStreamEvent)
	require.NoError(t, proto.Unmarshal(data, event))
	return event
}
