package stt

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

func TestTranscribeHappyPath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "in.wav")
	require.NoError(t, os.WriteFile(file, []byte("PCMDATA"), 0o600))

	fake := &fakeSvc{
		transcribe: func(req *sttv1.TranscribeRequest) (*sttv1.TranscribeResponse, error) {
			require.Equal(t, []byte("PCMDATA"), req.GetAudio())
			require.Equal(t, commonv1.AudioFormat_AUDIO_FORMAT_WAV, req.GetFormat())
			return &sttv1.TranscribeResponse{
				Text: "hello", ProviderTier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL, ProviderId: "whisper", LatencyMs: 99,
			}, nil
		},
	}
	app := mountSTT(t, fake)
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "file"}, {Name: "language"}, {Name: "format"}, {Name: "model"}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"file": file},
	})
	require.NoError(t, h.transcribe(ctx))
	out := buf.String()
	require.Contains(t, out, "Transcribed via local/whisper")
	require.Contains(t, out, "hello")
	require.Empty(t, fake.transcribeModelID)
}

func TestTranscribeModelFlagSelectsModelHeader(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "in.wav")
	require.NoError(t, os.WriteFile(file, []byte("PCMDATA"), 0o600))
	fake := &fakeSvc{transcribe: func(_ *sttv1.TranscribeRequest) (*sttv1.TranscribeResponse, error) {
		return &sttv1.TranscribeResponse{Text: "hello"}, nil
	}}
	app := mountSTT(t, fake)
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "file"}, {Name: "language"}, {Name: "format"}, {Name: "model"}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{Flags: map[string]string{"file": file, "model": "  rumble-2-flash  "}})
	require.NoError(t, h.transcribe(ctx))
	require.Equal(t, "rumble-2-flash", fake.transcribeModelID)
}

func TestTranscribeStreamUsesH2CGRPCTransport(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "in.wav")
	require.NoError(t, os.WriteFile(file, []byte("PCMDATA"), 0o600))

	app := mountSTTH2C(t, &fakeSvc{
		transcribeStream: func(_ context.Context, stream *connect.BidiStream[sttv1.TranscribeStreamRequest, sttv1.TranscribeStreamEvent]) error {
			var sawStart, sawChunk, sawEnd bool
			for {
				msg, err := stream.Receive()
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				switch p := msg.GetPayload().(type) {
				case *sttv1.TranscribeStreamRequest_Start:
					sawStart = true
					require.Equal(t, int32(2), p.Start.ProtocolVersion)
					require.NotEmpty(t, p.Start.SessionId)
					require.NotEmpty(t, p.Start.ResumeToken)
					require.Equal(t, "kyutai", p.Start.GetConfig().GetEngineId())
				case *sttv1.TranscribeStreamRequest_AudioChunk:
					sawChunk = true
					require.Equal(t, []byte("PCMDATA"), p.AudioChunk.Audio)
					require.Equal(t, uint64(0), p.AudioChunk.Sequence)
					require.Len(t, p.AudioChunk.Sha256, 32)
				case *sttv1.TranscribeStreamRequest_End:
					sawEnd = true
					require.True(t, sawStart)
					require.True(t, sawChunk)
					err := stream.Send(&sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_Segment{
						Segment: &sttv1.StreamSegment{
							Text:         "hello stream",
							ProviderTier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL,
							ModelId:      "fake-model",
							LatencyMs:    7,
						},
					}})
					require.NoError(t, err)
					err = stream.Send(&sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_Done{
						Done: &sttv1.StreamDone{
							FinalText:    "hello stream",
							ProviderTier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL,
							ModelId:      "fake-model",
							LatencyMs:    7,
						},
					}})
					require.NoError(t, err)
				}
			}
			require.True(t, sawEnd)
			return nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "file"}, {Name: "language"}, {Name: "engine"}, {Name: "chunk-bytes"}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"file": file, "engine": "kyutai", "chunk-bytes": "999"}, JSON: true,
	})
	require.NoError(t, h.transcribeStream(ctx))
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)
	var segment, done sttv1.TranscribeStreamEvent
	require.NoError(t, protojson.Unmarshal([]byte(lines[0]), &segment))
	require.NoError(t, protojson.Unmarshal([]byte(lines[1]), &done))
	require.Equal(t, "hello stream", segment.GetSegment().GetText())
	require.Equal(t, "fake-model", segment.GetSegment().GetModelId())
	require.Equal(t, "hello stream", done.GetDone().GetFinalText())

	humanCtx, humanBuf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"file": file, "engine": "kyutai", "chunk-bytes": "999"},
	})
	require.NoError(t, h.transcribeStream(humanCtx))
	require.Contains(t, humanBuf.String(), "segment [local/fake-model 7ms]: hello stream")
	require.Contains(t, humanBuf.String(), "done [local/fake-model 7ms]: hello stream")
}

func TestSendTranscribeAudioReportsCanceledSend(t *testing.T) {
	app := mountSTTH2C(t, &fakeSvc{})
	h := newHandlers(app)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stream := h.stream.TranscribeStream(ctx)

	const payloadMarker = "private-audio-payload"
	err := sendTranscribeAudio(stream, []byte(payloadMarker), 2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "transcribe-stream send audio chunk 0")
	require.NotContains(t, err.Error(), payloadMarker)
}

// Happy path: stream-config get prints the resolved levers using the
// documented defaults when fields are zero.

func TestTranscribeServerError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "in.wav")
	require.NoError(t, os.WriteFile(file, []byte("X"), 0o600))

	app := mountSTT(t, &fakeSvc{
		transcribe: func(_ *sttv1.TranscribeRequest) (*sttv1.TranscribeResponse, error) {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("whisper down"))
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "file"}, {Name: "language"}, {Name: "format"}, {Name: "model"}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"file": file},
	})
	err := h.transcribe(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "transcribe"), "want operation tag, got %q", err.Error())
}

func TestTranscribeStreamReturnsErrorForFailedTerminal(t *testing.T) {
	app := mountSTT(t, &fakeSvc{})
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{JSON: true})
	err := renderTranscribeStreamEvent(ctx, &sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_Done{
		Done: &sttv1.StreamDone{TerminalReason: "input_failed"},
	}})
	require.EqualError(t, err, "transcribe-stream terminated: input_failed")
}

func TestTranscribeStreamAcceptsCompletedTerminal(t *testing.T) {
	app := mountSTT(t, &fakeSvc{})
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{JSON: true})
	err := renderTranscribeStreamEvent(ctx, &sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_Done{
		Done: &sttv1.StreamDone{TerminalReason: "completed"},
	}})
	require.NoError(t, err)
}

// Error path: transcribe with a missing file path fails before any RPC,
// so the fake panics if called (proving the path-read happens client-side).

func TestTranscribeMissingFile(t *testing.T) {
	app := mountSTT(t, &fakeSvc{
		transcribe: func(_ *sttv1.TranscribeRequest) (*sttv1.TranscribeResponse, error) {
			t.Fatal("transcribe must not reach the server when file read fails")
			return nil, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "file"}, {Name: "language"}, {Name: "format"}, {Name: "model"}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"file": "/no/such/path.wav"},
	})
	err := h.transcribe(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "read"), "want read error tag, got %q", err.Error())
}
