package tts

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"

	"audio-tools/cli/internal/testutil"
)

type fakeSvc struct {
	ttsconnect.UnimplementedTTSServiceHandler
	synth  func(*ttsv1.SynthesizeRequest) (*ttsv1.SynthesizeResponse, error)
	voices func() ([]*ttsv1.Voice, error)
}

func (f *fakeSvc) Synthesize(_ context.Context, req *connect.Request[ttsv1.SynthesizeRequest]) (*connect.Response[ttsv1.SynthesizeResponse], error) {
	resp, err := f.synth(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeSvc) ListVoices(_ context.Context, _ *connect.Request[ttsv1.ListVoicesRequest]) (*connect.Response[ttsv1.ListVoicesResponse], error) {
	vs, err := f.voices()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ttsv1.ListVoicesResponse{Voices: vs}), nil
}

func mountTTS(t *testing.T, svc ttsconnect.TTSServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, h := ttsconnect.NewTTSServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	return testutil.NewTestApp(t, mux)
}

// Happy path: synthesize writes the returned audio bytes to --out and
// emits the human-friendly trace summary.
func TestSynthesizeWritesOutFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.mp3")

	app := mountTTS(t, &fakeSvc{
		synth: func(req *ttsv1.SynthesizeRequest) (*ttsv1.SynthesizeResponse, error) {
			require.Equal(t, "hello", req.GetText())
			require.Equal(t, "voice.neutral.default", req.GetVoice())
			require.Equal(t, "mp3", req.GetResponseFormat())
			require.InDelta(t, 1.0, req.GetSpeed(), 0.0001)
			return &ttsv1.SynthesizeResponse{
				Audio:        []byte("MP3DATA"),
				ContentType:  "audio/mpeg",
				ProviderTier: "local", ProviderId: "kokoro", LatencyMs: 11,
			}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "text"}, {Name: "voice"}, {Name: "speed"}, {Name: "format"}, {Name: "out"}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"text": "hello", "out": out},
	})
	require.NoError(t, h.synthesize(ctx))

	got, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, []byte("MP3DATA"), got)
	require.Contains(t, buf.String(), "Synthesized 7 bytes")
	require.Contains(t, buf.String(), "local/kokoro")
}

// Happy path: voices renders one bullet per canonical voice.
func TestVoicesList(t *testing.T) {
	app := mountTTS(t, &fakeSvc{
		voices: func() ([]*ttsv1.Voice, error) {
			return []*ttsv1.Voice{
				{Id: "voice.neutral.default", Name: "Neutral"},
				{Id: "voice.feminine.warm", Name: "Warm"},
			}, nil
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.voices(ctx))
	out := buf.String()
	require.Contains(t, out, "voice.neutral.default")
	require.Contains(t, out, "Warm")
}

// Error path: synthesize returns Unavailable — handler surfaces a
// wrapped "synthesize" error and does NOT create the --out file.
func TestSynthesizeProviderError(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.mp3")

	app := mountTTS(t, &fakeSvc{
		synth: func(_ *ttsv1.SynthesizeRequest) (*ttsv1.SynthesizeResponse, error) {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("kokoro offline"))
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "text"}, {Name: "voice"}, {Name: "speed"}, {Name: "format"}, {Name: "out"}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"text": "hi", "out": out},
	})
	err := h.synthesize(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "synthesize"), "want operation tag, got %q", err.Error())
	_, statErr := os.Stat(out)
	require.True(t, os.IsNotExist(statErr), "must not write output on error")
}
