package audio

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

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
	audioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio/audio_v1connect"

	"audio-tools/cli/internal/testutil"
)

// fakeAudio implements just the methods we need; the rest fall through
// to the generated Unimplemented stub so the interface stays satisfied.
type fakeAudio struct {
	audioconnect.UnimplementedAudioProcessingServiceHandler
	transcode func(*audiov1.TranscodeRequest) (*audiov1.TranscodeResponse, error)
}

func (f *fakeAudio) Transcode(_ context.Context, req *connect.Request[audiov1.TranscodeRequest]) (*connect.Response[audiov1.TranscodeResponse], error) {
	resp, err := f.transcode(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func mountAudio(t *testing.T, svc audioconnect.AudioProcessingServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, h := audioconnect.NewAudioProcessingServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	return testutil.NewTestApp(t, mux)
}

// Happy path: transcode reads input bytes, the fake service returns
// transformed audio, the handler writes them to --output and renders a
// human-friendly summary.
func TestTranscodeHappyPath(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.bin")
	out := filepath.Join(dir, "out.wav")
	require.NoError(t, os.WriteFile(in, []byte("RAW-BYTES"), 0o600))

	app := mountAudio(t, &fakeAudio{
		transcode: func(req *audiov1.TranscodeRequest) (*audiov1.TranscodeResponse, error) {
			require.Equal(t, []byte("RAW-BYTES"), req.GetAudio())
			require.Equal(t, "wav", req.GetOutputFormat())
			return &audiov1.TranscodeResponse{
				Audio:       []byte("WAV-OK"),
				ContentType: "audio/wav",
			}, nil
		},
	})

	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "input"}, {Name: "output"}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"input": in, "output": out},
	})
	require.NoError(t, h.transcode(ctx))

	written, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, []byte("WAV-OK"), written)
	require.Contains(t, buf.String(), "Wrote 6 bytes")
}

// Error path: server returns a Connect error; handler must surface it
// through WrapAPIError and must NOT write the output file.
func TestTranscodeServerError(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.bin")
	out := filepath.Join(dir, "out.wav")
	require.NoError(t, os.WriteFile(in, []byte("x"), 0o600))

	app := mountAudio(t, &fakeAudio{
		transcode: func(_ *audiov1.TranscodeRequest) (*audiov1.TranscodeResponse, error) {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("ffmpeg unavailable"))
		},
	})

	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "input"}, {Name: "output"}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"input": in, "output": out},
	})

	err := h.transcode(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "transcode"), "error should mention operation, got %q", err.Error())

	_, statErr := os.Stat(out)
	require.True(t, os.IsNotExist(statErr), "output file must not be created on server error")
}

// Register smoke: confirms the audio group wires correctly.
func TestRegister(t *testing.T) {
	app := mountAudio(t, &fakeAudio{})
	group := Register(app)
	require.Equal(t, "audio", group.Name)
	require.NotEmpty(t, group.Subcommands)
}
