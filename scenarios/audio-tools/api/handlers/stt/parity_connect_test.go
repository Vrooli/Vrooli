package stt

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/logx"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/stt/segmenter/testaudio"
	"audio-tools/internal/stt/session"
	"audio-tools/internal/sttengine"

	"github.com/vrooli/api-core/schedule"

	"github.com/stretchr/testify/require"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

// newFakeBatchExecutor returns a deterministic BatchExecutor producing
// a canned Result with the given final text. Provider metadata is fixed
// so the parity projection stays stable across transports/strategies.
func newFakeBatchExecutor(text string) *sttmocks.FakeBatchExecutor {
	return &sttmocks.FakeBatchExecutor{Result: &sttchain.Result{
		Text:       text,
		Tier:       sttchain.TierLocal,
		ProviderID: "fake-local",
		ModelID:    "fake-model",
		Latency:    1 * time.Millisecond,
	}}
}

type eventProjection struct {
	Kind sttchain.StreamEventKind
	Text string
}

func projectFromProto(events []*sttv1.TranscribeStreamEvent) []eventProjection {
	out := make([]eventProjection, 0, len(events))
	for _, ev := range events {
		p := eventProjection{}
		switch e := ev.GetEvent().(type) {
		case *sttv1.TranscribeStreamEvent_Partial:
			p.Kind = sttchain.StreamEventPartial
			p.Text = e.Partial.GetText()
		case *sttv1.TranscribeStreamEvent_Segment:
			p.Kind = sttchain.StreamEventSegment
			p.Text = e.Segment.GetText()
		case *sttv1.TranscribeStreamEvent_WakeWord:
			p.Kind = sttchain.StreamEventWakeWord
		case *sttv1.TranscribeStreamEvent_SpeakerRejection:
			p.Kind = sttchain.StreamEventSpeakerRejection
		case *sttv1.TranscribeStreamEvent_Error:
			p.Kind = sttchain.StreamEventError
		case *sttv1.TranscribeStreamEvent_Done:
			p.Kind = sttchain.StreamEventDone
			p.Text = e.Done.GetFinalText()
		default:
			continue // v2 acknowledgements are transport state, not transcript projection.
		}
		out = append(out, p)
	}
	return out
}

func runConnectBidi(t *testing.T, audio []byte) []eventProjection {
	t.Helper()
	exec := newFakeBatchExecutor("hello world")
	chain := sttchain.NewChain(sttchain.Options{})
	selector := sttpkg.NewSelector(exec)

	// Same mode as the reference in internal/stt/segmenter/parity_test.go:
	// this asserts parity on the explicit unary compatibility path. Auto mode
	// fails closed when no streaming provider is available — a selector error
	// paired with BufferedFallback is not permission to turn an unbounded
	// dictation into an unbounded whole-turn buffer (segmenter.go Run). Leaving
	// the config unset here resolved to auto, so the transport was correctly
	// failing closed while the test still expected the pre-fail-closed
	// projection.
	connectPath, h := sttconnect.NewSTTServiceHandler(NewConnectHandler(Deps{
		Chain:        chain,
		Selector:     selector,
		Logger:       logx.Std{},
		Clock:        schedule.System(),
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"off"}`},
	}))
	mux := http.NewServeMux()
	mux.Handle(connectPath, h)
	// Connect bidi-streaming requires HTTP/2. httptest.NewTLSServer
	// auto-negotiates h2 via ALPN; httptest.NewServer (HTTP/1.1) does
	// not, and the client gets "write envelope: EOF" mid-send.
	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()

	client := sttconnect.NewSTTServiceClient(srv.Client(), srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream := client.TranscribeStream(ctx)
	if err := stream.Send(&sttv1.TranscribeStreamRequest{
		Payload: &sttv1.TranscribeStreamRequest_Start{Start: &sttv1.StreamStart{ProtocolVersion: 2, SessionId: "parity-session", ResumeToken: "parity-token"}},
	}); err != nil {
		t.Fatalf("send start: %v", err)
	}
	if err := stream.Send(&sttv1.TranscribeStreamRequest{
		Payload: &sttv1.TranscribeStreamRequest_AudioChunk{AudioChunk: &sttv1.StreamAudioChunk{Audio: audio, Sequence: 0, StartSample: 0, EndSample: int64(len(audio) / 2)}},
	}); err != nil {
		t.Fatalf("send chunk: %v", err)
	}
	if err := stream.Send(&sttv1.TranscribeStreamRequest{
		Payload: &sttv1.TranscribeStreamRequest_End{End: &sttv1.StreamEnd{}},
	}); err != nil {
		t.Fatalf("send end: %v", err)
	}
	if err := stream.CloseRequest(); err != nil {
		t.Fatalf("close request: %v", err)
	}

	var evs []*sttv1.TranscribeStreamEvent
	for {
		ev, err := stream.Receive()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Logf("stream.Receive error: %v", err)
			break
		}
		evs = append(evs, ev)
	}
	_ = stream.CloseResponse()
	return projectFromProto(evs)
}

// TestStreamingParity_ConnectBidi asserts that the Connect bidi
// transport, driven through a real httptest server, produces the same
// projected event sequence as the Segmenter when both share the same
// Selector and BatchExecutor. Parity here is the contract that prevents
// the two transports from silently drifting; it is the test the plan
// names "TestStreamingParity".
func TestStreamingParity_ConnectBidi(t *testing.T) {
	audio := testaudio.SpeechTonePauseTone3s()
	want := []eventProjection{
		{Kind: sttchain.StreamEventSegment, Text: "hello world"},
		{Kind: sttchain.StreamEventDone, Text: "hello world"},
	}
	got := runConnectBidi(t, audio)
	if len(got) != len(want) {
		t.Fatalf("connect bidi length mismatch:\n got=%+v\nwant=%+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("connect bidi projection mismatch at %d:\n got=%+v\nwant=%+v", i, got[i], want[i])
		}
	}
}

func TestValidateStreamStartEngine_UsesExplicitConnectSelection(t *testing.T) {
	registry, err := sttengine.Load([]byte(`{
      "engines": [{
        "id": "kyutai", "displayName": "Kyutai", "kind": "local_resource",
        "resource": "kyutai-stt", "provides": {"nativeStreaming": true},
        "requires": {"pcm16kMono": true}, "strategies": ["passthrough"]
      }],
      "speakerIsolation": {"active": "verification", "methods": {"verification": {"backendResource": "sherpa-onnx"}}}
    }`))
	require.NoError(t, err)

	start := &sttv1.StreamStart{Config: &sttv1.StreamConfig{EngineId: " kyutai "}}
	require.NoError(t, validateStreamStartEngine(start, registry))
	require.Equal(t, "kyutai", streamStartEngineID(start))

	start.Config.EngineId = "missing-engine"
	require.ErrorContains(t, validateStreamStartEngine(start, registry), "unknown engine_id")
}

// TestConnectBidiKeepsTerminalSessionRecoverable is the Connect-side twin of
// TestStreamWS_V2ReplayReemitsCommittedSegmentWithStableIdentity.
//
// The Connect transport used to remove the session as soon as it had sent the
// terminal event, on the reasoning that delivery ended the session's usefulness.
// Delivered is not acted on: a client that reconnects with the same session id
// has to replay the SAME committed segment, not re-transcribe into a duplicate
// identity. Removing it here made the two transports disagree about a contract
// only one of them had a test for -- which is the drift this parity file exists
// to catch. Retention stays bounded by the registry's recovery expiry.
func TestConnectBidiKeepsTerminalSessionRecoverable(t *testing.T) {
	exec := newFakeBatchExecutor("hello world")
	chain := sttchain.NewChain(sttchain.Options{})
	sessions := session.NewRegistry(0)
	connectPath, h := sttconnect.NewSTTServiceHandler(NewConnectHandler(Deps{
		Chain:        chain,
		Selector:     sttpkg.NewSelector(exec),
		Logger:       logx.Std{},
		Clock:        schedule.System(),
		Sessions:     sessions,
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"off"}`},
	}))
	mux := http.NewServeMux()
	mux.Handle(connectPath, h)
	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()

	client := sttconnect.NewSTTServiceClient(srv.Client(), srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	audio := testaudio.SpeechTonePauseTone3s()
	stream := client.TranscribeStream(ctx)
	require.NoError(t, stream.Send(&sttv1.TranscribeStreamRequest{
		Payload: &sttv1.TranscribeStreamRequest_Start{Start: &sttv1.StreamStart{ProtocolVersion: 2, SessionId: "connect-retain-session", ResumeToken: "connect-retain-token"}},
	}))
	require.NoError(t, stream.Send(&sttv1.TranscribeStreamRequest{
		Payload: &sttv1.TranscribeStreamRequest_AudioChunk{AudioChunk: &sttv1.StreamAudioChunk{Audio: audio, Sequence: 0, StartSample: 0, EndSample: int64(len(audio) / 2)}},
	}))
	require.NoError(t, stream.Send(&sttv1.TranscribeStreamRequest{
		Payload: &sttv1.TranscribeStreamRequest_End{End: &sttv1.StreamEnd{}},
	}))
	require.NoError(t, stream.CloseRequest())
	sawDone := false
	for {
		ev, err := stream.Receive()
		if err != nil {
			break
		}
		if _, ok := ev.GetEvent().(*sttv1.TranscribeStreamEvent_Done); ok {
			sawDone = true
		}
	}
	_ = stream.CloseResponse()
	require.True(t, sawDone, "the turn must reach a terminal done before retention is asserted")

	ledger, resumed, err := sessions.Open("connect-retain-session", "connect-retain-token")
	require.NoError(t, err)
	require.True(t, resumed, "a terminal Connect session must stay resumable for replay")
	require.Len(t, ledger.Snapshot().Committed, 1, "the committed segment must survive for stable-identity replay")
}
