package stt

import (
	"context"
	"crypto/sha256"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/logx"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/stt/segmenter/testaudio"
	"audio-tools/internal/stt/session"
	"audio-tools/internal/sttengine"
	"audio-tools/internal/testutil/mocks"

	"github.com/vrooli/api-core/schedule"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

func newBidiServiceClient(t *testing.T, d Deps) sttconnect.STTServiceClient {
	t.Helper()
	if d.Logger == nil {
		d.Logger = logx.Std{}
	}
	if d.Clock == nil {
		d.Clock = schedule.System()
	}
	connectPath, h := sttconnect.NewSTTServiceHandler(NewConnectHandler(d))
	mux := http.NewServeMux()
	mux.Handle(connectPath, h)
	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return sttconnect.NewSTTServiceClient(srv.Client(), srv.URL)
}

func sendStream(t *testing.T, stream *connect.BidiStreamForClient[sttv1.TranscribeStreamRequest, sttv1.TranscribeStreamEvent], msgs ...*sttv1.TranscribeStreamRequest) error {
	t.Helper()
	for _, msg := range msgs {
		if err := stream.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

func receiveUntilError(t *testing.T, stream *connect.BidiStreamForClient[sttv1.TranscribeStreamRequest, sttv1.TranscribeStreamEvent]) error {
	t.Helper()
	for {
		_, err := stream.Receive()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func receiveEvents(t *testing.T, stream *connect.BidiStreamForClient[sttv1.TranscribeStreamRequest, sttv1.TranscribeStreamEvent]) ([]*sttv1.TranscribeStreamEvent, error) {
	t.Helper()
	var events []*sttv1.TranscribeStreamEvent
	for {
		ev, err := stream.Receive()
		if err == io.EOF {
			return events, nil
		}
		if err != nil {
			// Connect reports a benign teardown artifact when the server
			// finishes before the client half-closes the bidi request. The
			// events already delivered are the contract under test.
			if strings.Contains(err.Error(), "final message has protocol-specific flags") {
				return events, nil
			}
			return events, err
		}
		events = append(events, ev)
	}
}

func startMsg(session, token string) *sttv1.TranscribeStreamRequest {
	return &sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_Start{Start: &sttv1.StreamStart{
		ProtocolVersion: 2, SessionId: session, ResumeToken: token,
	}}}
}

func chunkMsg(seq uint64, audio []byte) *sttv1.TranscribeStreamRequest {
	return &sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_AudioChunk{AudioChunk: &sttv1.StreamAudioChunk{
		Audio: audio, Sequence: seq, StartSample: 0, EndSample: int64(len(audio) / 2),
	}}}
}

func TestTranscribeStream_RejectsUnconfiguredPipeline(t *testing.T) {
	client := newBidiServiceClient(t, Deps{})
	stream := client.TranscribeStream(context.Background())
	_ = sendStream(t, stream, startMsg("s", "t"))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(receiveUntilError(t, stream)))
}

func TestTranscribeStream_RejectsClientThatNeverSendsStart(t *testing.T) {
	client := newBidiServiceClient(t, Deps{Chain: sttchain.NewChain(sttchain.Options{}), Selector: sttpkg.NewSelector(newFakeBatchExecutor("x"))})
	stream := client.TranscribeStream(context.Background())
	require.NoError(t, stream.CloseRequest())
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(receiveUntilError(t, stream)))
}

func TestTranscribeStream_RejectsNonStartFirstMessage(t *testing.T) {
	client := newBidiServiceClient(t, Deps{Chain: sttchain.NewChain(sttchain.Options{}), Selector: sttpkg.NewSelector(newFakeBatchExecutor("x"))})
	stream := client.TranscribeStream(context.Background())
	require.NoError(t, sendStream(t, stream, chunkMsg(0, []byte{0, 0})))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(receiveUntilError(t, stream)))
}

func TestTranscribeStream_RequiresProtocolV2Identity(t *testing.T) {
	client := newBidiServiceClient(t, Deps{Chain: sttchain.NewChain(sttchain.Options{}), Selector: sttpkg.NewSelector(newFakeBatchExecutor("x"))})
	stream := client.TranscribeStream(context.Background())
	require.NoError(t, sendStream(t, stream, &sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_Start{Start: &sttv1.StreamStart{ProtocolVersion: 2}}}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(receiveUntilError(t, stream)))
}

func TestTranscribeStreamAcknowledgesAdmissionBeforeAudio(t *testing.T) {
	sessions := session.NewRegistry(0)
	client := newBidiServiceClient(t, Deps{
		Chain: sttchain.NewChain(sttchain.Options{}),
		Selector: sttpkg.NewSelector(newFakeBatchExecutor("hello")),
		Sessions: sessions,
	})
	stream := client.TranscribeStream(context.Background())
	require.NoError(t, sendStream(t, stream, startMsg("admission-session", "admission-token")))

	accepted, err := stream.Receive()
	require.NoError(t, err)
	status := accepted.GetSessionStatus()
	require.NotNil(t, status)
	require.Equal(t, "accepted", status.GetState())
	require.Equal(t, "admission-session", status.GetSessionId())
	require.Equal(t, uint64(0), status.GetGeneration())

	require.NoError(t, sendStream(t, stream, &sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_End{End: &sttv1.StreamEnd{}}}))
	events, err := receiveEvents(t, stream)
	require.NoError(t, err)
	require.NotEmpty(t, events)
}

func TestTranscribeStream_RejectsUnknownEngineSelection(t *testing.T) {
	registry, err := sttengine.Load([]byte(`{
      "engines": [{"id": "kyutai", "displayName": "Kyutai", "kind": "local_resource", "resource": "kyutai-stt", "provides": {"nativeStreaming": true}, "strategies": ["passthrough"]}],
      "speakerIsolation": {"active": "verification", "methods": {"verification": {"backendResource": "sherpa-onnx"}}}
    }`))
	require.NoError(t, err)
	client := newBidiServiceClient(t, Deps{
		Chain: sttchain.NewChain(sttchain.Options{}), Selector: sttpkg.NewSelector(newFakeBatchExecutor("x")), Registry: registry,
	})
	stream := client.TranscribeStream(context.Background())
	start := startMsg("s", "t")
	start.GetStart().Config = &sttv1.StreamConfig{EngineId: "missing-engine"}
	require.NoError(t, sendStream(t, stream, start))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(receiveUntilError(t, stream)))
}

func TestTranscribeStream_DuplicateChunkIsIgnored(t *testing.T) {
	c := sttchain.NewChain(sttchain.Options{})
	client := newBidiServiceClient(t, Deps{
		Chain: c, Selector: sttpkg.NewSelector(newFakeBatchExecutor("hello world")),
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"off"}`},
	})
	audio := testaudio.SpeechTonePauseTone3s()
	stream := client.TranscribeStream(context.Background())
	require.NoError(t, sendStream(t, stream,
		startMsg("duplicate-session", "duplicate-token"),
		chunkMsg(0, audio),
		chunkMsg(0, audio),
		&sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_End{End: &sttv1.StreamEnd{}}},
	))
	events, err := receiveEvents(t, stream)
	require.NoError(t, err)
	var segments int
	var done *sttv1.StreamDone
	for _, ev := range events {
		if seg := ev.GetSegment(); seg != nil {
			segments++
		}
		if d := ev.GetDone(); d != nil {
			done = d
		}
	}
	require.Equal(t, 1, segments, "the duplicate sequence must not produce a second segment")
	require.NotNil(t, done)
	require.NotEqual(t, "input_failed", done.GetTerminalReason())
}

func TestTranscribeStream_QuietInputTimesOutAndPersistsFailure(t *testing.T) {
	sessions := session.NewRegistry(0)
	client := newBidiServiceClient(t, Deps{
		Chain: sttchain.NewChain(sttchain.Options{}), Selector: sttpkg.NewSelector(newFakeBatchExecutor("hello")),
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"off"}`}, Sessions: sessions,
		StreamIdleTimeout: 40 * time.Millisecond,
	})
	stream := client.TranscribeStream(context.Background())
	require.NoError(t, sendStream(t, stream, startMsg("quiet-session", "quiet-token")))
	_, err := receiveEvents(t, stream)
	require.NoError(t, err)
	snapshot, ok := sessions.Snapshot("quiet-session")
	require.True(t, ok)
	require.Equal(t, session.TerminalReason("input_idle_timeout"), snapshot.TerminalReason)
}

func TestTranscribeStream_AudioActivityExtendsIdleWindow(t *testing.T) {
	sessions := session.NewRegistry(0)
	client := newBidiServiceClient(t, Deps{
		Chain: sttchain.NewChain(sttchain.Options{}), Selector: sttpkg.NewSelector(newFakeBatchExecutor("hello")),
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"off"}`}, Sessions: sessions,
		StreamIdleTimeout: 150 * time.Millisecond,
	})
	stream := client.TranscribeStream(context.Background())
	require.NoError(t, sendStream(t, stream, startMsg("active-session", "active-token")))
	for seq := uint64(0); seq < 5; seq++ {
		time.Sleep(75 * time.Millisecond)
		require.NoError(t, sendStream(t, stream, chunkMsg(seq, []byte{0, 0})))
	}
	require.NoError(t, sendStream(t, stream, &sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_End{End: &sttv1.StreamEnd{}}}))
	events, err := receiveEvents(t, stream)
	require.NoError(t, err)
	var done *sttv1.StreamDone
	for _, event := range events {
		if value := event.GetDone(); value != nil {
			done = value
		}
	}
	require.NotNil(t, done)
	require.NotEqual(t, "input_idle_timeout", done.GetTerminalReason())
	snapshot, ok := sessions.Snapshot("active-session")
	require.True(t, ok)
	require.NotEqual(t, session.TerminalReason("input_idle_timeout"), snapshot.TerminalReason)
}

func TestTranscribeStream_DigestMismatchFailsTheInput(t *testing.T) {
	c := sttchain.NewChain(sttchain.Options{})
	client := newBidiServiceClient(t, Deps{
		Chain: c, Selector: sttpkg.NewSelector(newFakeBatchExecutor("hello")),
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"off"}`},
	})
	audio := testaudio.SpeechTonePauseTone3s()
	chunk := chunkMsg(0, audio)
	wrong := sha256.Sum256([]byte("not-the-audio"))
	chunk.GetAudioChunk().Sha256 = wrong[:]

	stream := client.TranscribeStream(context.Background())
	require.NoError(t, sendStream(t, stream,
		startMsg("digest-session", "digest-token"),
		chunk,
		&sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_End{End: &sttv1.StreamEnd{}}},
	))
	events, err := receiveEvents(t, stream)
	require.NoError(t, err)
	var done *sttv1.StreamDone
	for _, ev := range events {
		if d := ev.GetDone(); d != nil {
			done = d
		}
	}
	require.NotNil(t, done)
	require.Equal(t, "input_failed", done.GetTerminalReason())
}

func TestTranscribeStream_DuplicateStartFailsClosed(t *testing.T) {
	c := sttchain.NewChain(sttchain.Options{})
	logger := mocks.NewFakeLogger()
	client := newBidiServiceClient(t, Deps{
		Chain: c, Selector: sttpkg.NewSelector(newFakeBatchExecutor("hello")),
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"off"}`}, Logger: logger,
	})
	stream := client.TranscribeStream(context.Background())
	require.NoError(t, sendStream(t, stream,
		startMsg("dup-start", "dup-start-token"),
		startMsg("dup-start", "dup-start-token"),
		&sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_End{End: &sttv1.StreamEnd{}}},
	))
	events, err := receiveEvents(t, stream)
	require.NoError(t, err)
	var done *sttv1.StreamDone
	for _, ev := range events {
		if d := ev.GetDone(); d != nil {
			done = d
		}
	}
	require.NotNil(t, done)
	require.Equal(t, "input_failed", done.GetTerminalReason())
	require.Len(t, logger.Entries(), 1)
	require.Contains(t, logger.Entries()[0], "event=stt_stream_input_failure stage=validate_message")
	require.Contains(t, logger.Entries()[0], "cause_kind=other")
	require.NotContains(t, logger.Entries()[0], "duplicate start")
}
