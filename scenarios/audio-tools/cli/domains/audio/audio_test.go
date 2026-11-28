package audio

import (
	"context"
	"encoding/json"
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
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"

	testutil "github.com/vrooli/cli-core/cliapptest"
)

// fakeAudio implements just the methods we need; the rest fall through
// to the generated Unimplemented stub so the interface stays satisfied.
type fakeAudio struct {
	audioconnect.UnimplementedAudioProcessingServiceHandler
	transcode       func(*audiov1.TranscodeRequest) (*audiov1.TranscodeResponse, error)
	trim            func(*audiov1.TrimRequest) (*audiov1.TrimResponse, error)
	merge           func(*audiov1.MergeRequest) (*audiov1.MergeResponse, error)
	split           func(*audiov1.SplitRequest) (*audiov1.SplitResponse, error)
	fade            func(*audiov1.FadeRequest) (*audiov1.FadeResponse, error)
	volume          func(*audiov1.VolumeRequest) (*audiov1.VolumeResponse, error)
	normalize       func(*audiov1.NormalizeRequest) (*audiov1.NormalizeResponse, error)
	extractMetadata func(*audiov1.ExtractMetadataRequest) (*audiov1.ExtractMetadataResponse, error)
}

func (f *fakeAudio) Transcode(_ context.Context, req *connect.Request[audiov1.TranscodeRequest]) (*connect.Response[audiov1.TranscodeResponse], error) {
	resp, err := f.transcode(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeAudio) Trim(_ context.Context, req *connect.Request[audiov1.TrimRequest]) (*connect.Response[audiov1.TrimResponse], error) {
	resp, err := f.trim(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeAudio) Merge(_ context.Context, req *connect.Request[audiov1.MergeRequest]) (*connect.Response[audiov1.MergeResponse], error) {
	resp, err := f.merge(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeAudio) Split(_ context.Context, req *connect.Request[audiov1.SplitRequest]) (*connect.Response[audiov1.SplitResponse], error) {
	resp, err := f.split(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeAudio) Fade(_ context.Context, req *connect.Request[audiov1.FadeRequest]) (*connect.Response[audiov1.FadeResponse], error) {
	resp, err := f.fade(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeAudio) Volume(_ context.Context, req *connect.Request[audiov1.VolumeRequest]) (*connect.Response[audiov1.VolumeResponse], error) {
	resp, err := f.volume(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeAudio) Normalize(_ context.Context, req *connect.Request[audiov1.NormalizeRequest]) (*connect.Response[audiov1.NormalizeResponse], error) {
	resp, err := f.normalize(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeAudio) ExtractMetadata(_ context.Context, req *connect.Request[audiov1.ExtractMetadataRequest]) (*connect.Response[audiov1.ExtractMetadataResponse], error) {
	resp, err := f.extractMetadata(req.Msg)
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
	require.NoError(t, os.WriteFile(out, []byte("old output"), 0o644))
	require.NoError(t, os.Chmod(out, 0o644))

	app := mountAudio(t, &fakeAudio{
		transcode: func(req *audiov1.TranscodeRequest) (*audiov1.TranscodeResponse, error) {
			require.Equal(t, []byte("RAW-BYTES"), req.GetAudio())
			require.Equal(t, commonv1.AudioFormat_AUDIO_FORMAT_WAV, req.GetOutputFormat())
			return &audiov1.TranscodeResponse{
				Audio:       []byte("WAV-OK"),
				ContentType: "audio/wav",
			}, nil
		},
	})

	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "input"},
		{Name: "output"},
		{Name: "output-format", Default: "wav"},
		{Name: "input-format"},
		{Name: "sample-rate", Default: "0"},
		{Name: "channels", Default: "0"},
		{Name: "bitrate", Default: "0"},
	}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"input": in, "output": out},
	})
	require.NoError(t, h.transcode(ctx))

	written, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, []byte("WAV-OK"), written)
	info, err := os.Stat(out)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "audio output must not be readable by group/other users")
	require.Contains(t, buf.String(), "Wrote 6 bytes")
}

func TestTranscodeJSONCompatibility(t *testing.T) {
	dir := t.TempDir()
	in, out := filepath.Join(dir, "in.bin"), filepath.Join(dir, "out.wav")
	require.NoError(t, os.WriteFile(in, []byte("RAW"), 0o600))
	app := mountAudio(t, &fakeAudio{transcode: func(*audiov1.TranscodeRequest) (*audiov1.TranscodeResponse, error) {
		return &audiov1.TranscodeResponse{Audio: []byte("WAV-OK")}, nil
	}})
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "input"}, {Name: "output"}, {Name: "output-format", Default: "wav"}, {Name: "input-format"}, {Name: "sample-rate", Default: "0"}, {Name: "channels", Default: "0"}, {Name: "bitrate", Default: "0"}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{Flags: map[string]string{"input": in, "output": out}, JSON: true})
	require.NoError(t, newHandlers(app).transcodePrimitive().Run(ctx))
	require.JSONEq(t, `{"operation":"transcode","path":"`+out+`","bytes":6}`, buf.String())
	written, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, []byte("WAV-OK"), written)
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
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "input"},
		{Name: "output"},
		{Name: "output-format", Default: "wav"},
		{Name: "input-format"},
		{Name: "sample-rate", Default: "0"},
		{Name: "channels", Default: "0"},
		{Name: "bitrate", Default: "0"},
	}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"input": in, "output": out},
	})

	err := h.transcode(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "transcode"), "error should mention operation, got %q", err.Error())

	_, statErr := os.Stat(out)
	require.True(t, os.IsNotExist(statErr), "output file must not be created on server error")
}

// Register smoke: confirms the audio group wires correctly from the
// manifest (the single source of truth for the command surface).
func TestRegister(t *testing.T) {
	app := mountAudio(t, &fakeAudio{})
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, err)
	group, err := Register(app, raw)
	require.NoError(t, err)
	require.Equal(t, "audio", group.Name)
	require.NotEmpty(t, group.Subcommands)
}

func TestTrimHappyPath(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.bin")
	out := filepath.Join(dir, "out.mp3")
	require.NoError(t, os.WriteFile(in, []byte("TRIM-BYTES"), 0o600))

	app := mountAudio(t, &fakeAudio{
		trim: func(req *audiov1.TrimRequest) (*audiov1.TrimResponse, error) {
			require.Equal(t, []byte("TRIM-BYTES"), req.GetAudio())
			require.Equal(t, commonv1.AudioFormat_AUDIO_FORMAT_MP3, req.GetFormat())
			require.Equal(t, 1.5, req.GetStartSeconds())
			require.Equal(t, 3.0, req.GetEndSeconds())
			return &audiov1.TrimResponse{Audio: []byte("TRIMMED"), ContentType: "audio/mpeg"}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "input"},
		{Name: "output"},
		{Name: "format"},
		{Name: "start-seconds"},
		{Name: "end-seconds"},
	}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"input": in, "output": out, "format": "mp3",
			"start-seconds": "1.5", "end-seconds": "3.0",
		},
	})
	require.NoError(t, h.trim(ctx))

	written, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, []byte("TRIMMED"), written)
	require.Contains(t, buf.String(), "Wrote 7 bytes")
}

func TestMergeReadsEverySource(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.bin")
	b := filepath.Join(dir, "b.bin")
	out := filepath.Join(dir, "merged.wav")
	require.NoError(t, os.WriteFile(a, []byte("A"), 0o600))
	require.NoError(t, os.WriteFile(b, []byte("B"), 0o600))

	app := mountAudio(t, &fakeAudio{
		merge: func(req *audiov1.MergeRequest) (*audiov1.MergeResponse, error) {
			require.Len(t, req.GetSources(), 2)
			require.Equal(t, []byte("A"), req.GetSources()[0].GetAudio())
			require.Equal(t, []byte("B"), req.GetSources()[1].GetAudio())
			require.Equal(t, commonv1.AudioFormat_AUDIO_FORMAT_WAV, req.GetOutputFormat())
			require.Equal(t, 0.25, req.GetCrossfadeSeconds())
			return &audiov1.MergeResponse{Audio: []byte("MERGED"), ContentType: "audio/wav"}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "input", Repeated: true},
		{Name: "output"},
		{Name: "format"},
		{Name: "output-format"},
		{Name: "crossfade-seconds"},
	}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		FlagLists: map[string][]string{"input": {a, b}},
		Flags:     map[string]string{"output": out, "format": "wav", "output-format": "wav", "crossfade-seconds": "0.25"},
	})
	require.NoError(t, h.merge(ctx))

	written, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, []byte("MERGED"), written)
}

func TestSplitWritesOneFilePerChunk(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.bin")
	outDir := filepath.Join(dir, "chunks")
	require.NoError(t, os.WriteFile(in, []byte("SPLIT-BYTES"), 0o600))

	app := mountAudio(t, &fakeAudio{
		split: func(req *audiov1.SplitRequest) (*audiov1.SplitResponse, error) {
			require.Equal(t, 2.0, req.GetChunkSeconds())
			require.Equal(t, commonv1.AudioFormat_AUDIO_FORMAT_WAV, req.GetOutputFormat())
			return &audiov1.SplitResponse{Chunks: []*audiov1.SplitChunk{
				{Audio: []byte("C0"), ContentType: "audio/wav", StartSeconds: 0, DurationSeconds: 2},
				{Audio: []byte("C1"), ContentType: "audio/wav", StartSeconds: 2, DurationSeconds: 2},
			}}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "input"},
		{Name: "output"},
		{Name: "format"},
		{Name: "chunk-seconds"},
		{Name: "boundaries-seconds"},
		{Name: "output-format"},
	}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"input": in, "output": outDir, "format": "wav",
			"chunk-seconds": "2", "output-format": "wav",
		},
	})
	require.NoError(t, h.split(ctx))

	first, err := os.ReadFile(filepath.Join(outDir, "chunk-000.wav"))
	require.NoError(t, err)
	require.Equal(t, []byte("C0"), first)
	second, err := os.ReadFile(filepath.Join(outDir, "chunk-001.wav"))
	require.NoError(t, err)
	require.Equal(t, []byte("C1"), second)
	for _, path := range []string{filepath.Join(outDir, "chunk-000.wav"), filepath.Join(outDir, "chunk-001.wav")} {
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
	dirInfo, err := os.Stat(outDir)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())
	require.Contains(t, buf.String(), "Wrote 2 chunk(s)")
}

func TestSplitJSONReportsWrittenFilesWithoutEmbeddingAudio(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.wav")
	outDir := filepath.Join(dir, "chunks")
	require.NoError(t, os.WriteFile(in, []byte("INPUT"), 0o600))
	app := mountAudio(t, &fakeAudio{
		split: func(*audiov1.SplitRequest) (*audiov1.SplitResponse, error) {
			return &audiov1.SplitResponse{Chunks: []*audiov1.SplitChunk{
				{Audio: []byte("PRIVATE-CHUNK-0"), ContentType: "audio/wav", StartSeconds: 0, DurationSeconds: 0.5},
				{Audio: []byte("PRIVATE-CHUNK-1"), ContentType: "audio/wav", StartSeconds: 0.5, DurationSeconds: 0.5},
			}}, nil
		},
	})
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "input"}, {Name: "output"}, {Name: "format"}, {Name: "chunk-seconds"}, {Name: "boundaries-seconds"}, {Name: "output-format"}}}
	ctx, output := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"input": in, "output": outDir, "format": "wav", "chunk-seconds": "0.5", "output-format": "wav"},
		JSON:  true,
	})
	require.NoError(t, newHandlers(app).splitPrimitive().Run(ctx))

	var result struct {
		Operation       string `json:"operation"`
		OutputDirectory string `json:"output_directory"`
		Chunks          []struct {
			Path            string  `json:"path"`
			StartSeconds    float64 `json:"start_seconds"`
			DurationSeconds float64 `json:"duration_seconds"`
			BytesWritten    int     `json:"bytes_written"`
		} `json:"chunks"`
	}
	require.NoError(t, json.Unmarshal([]byte(output.String()), &result))
	require.Equal(t, "split", result.Operation)
	require.Equal(t, outDir, result.OutputDirectory)
	require.Len(t, result.Chunks, 2)
	require.Equal(t, filepath.Join(outDir, "chunk-000.wav"), result.Chunks[0].Path)
	require.Equal(t, 0.5, result.Chunks[1].StartSeconds)
	require.Equal(t, len("PRIVATE-CHUNK-0"), result.Chunks[0].BytesWritten)
	require.NotContains(t, output.String(), "PRIVATE-CHUNK")
	require.Less(t, output.Len(), 1_000, "machine output should remain metadata-sized")
}

func TestFadeAndVolumeForwardNumbers(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.bin")
	require.NoError(t, os.WriteFile(in, []byte("X"), 0o600))

	app := mountAudio(t, &fakeAudio{
		fade: func(req *audiov1.FadeRequest) (*audiov1.FadeResponse, error) {
			require.Equal(t, 0.5, req.GetFadeInSeconds())
			require.Equal(t, 1.5, req.GetFadeOutSeconds())
			return &audiov1.FadeResponse{Audio: []byte("FADED"), ContentType: "audio/wav"}, nil
		},
		volume: func(req *audiov1.VolumeRequest) (*audiov1.VolumeResponse, error) {
			require.Equal(t, -6.0, req.GetGainDb())
			return &audiov1.VolumeResponse{Audio: []byte("LOUDER"), ContentType: "audio/wav"}, nil
		},
	})
	h := newHandlers(app)

	fadeSchema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "input"},
		{Name: "output"},
		{Name: "format"},
		{Name: "fade-in-seconds"},
		{Name: "fade-out-seconds"},
		{Name: "output-format"},
	}}
	fadeCtx, _ := cliapptest.NewCapturedRunContext(app, fadeSchema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"input": in, "output": filepath.Join(dir, "fade.wav"), "format": "wav",
			"fade-in-seconds": "0.5", "fade-out-seconds": "1.5", "output-format": "wav",
		},
	})
	require.NoError(t, h.fade(fadeCtx))

	volumeSchema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "input"},
		{Name: "output"},
		{Name: "format"},
		{Name: "gain-db"},
		{Name: "output-format"},
	}}
	volumeCtx, _ := cliapptest.NewCapturedRunContext(app, volumeSchema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"input": in, "output": filepath.Join(dir, "volume.wav"), "format": "wav",
			"gain-db": "-6", "output-format": "wav",
		},
	})
	require.NoError(t, h.volume(volumeCtx))
}

func TestExtractMetadataPrintsReadableJSON(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.wav")
	require.NoError(t, os.WriteFile(in, []byte("META"), 0o600))

	app := mountAudio(t, &fakeAudio{
		extractMetadata: func(req *audiov1.ExtractMetadataRequest) (*audiov1.ExtractMetadataResponse, error) {
			require.Equal(t, []byte("META"), req.GetAudio())
			return &audiov1.ExtractMetadataResponse{Metadata: &audiov1.AudioMetadata{
				DurationSeconds: 1.25,
				SampleRate:      16000,
				Channels:        1,
				Bitrate:         128000,
				Codec:           "pcm_s16le",
				Format:          commonv1.AudioFormat_AUDIO_FORMAT_WAV,
				Tags:            map[string]string{"title": "fixture"},
			}}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "input"}, {Name: "output"}, {Name: "format"}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"input": in, "format": "wav", "output": "-"},
	})
	require.NoError(t, h.extractMetadata(ctx))
	require.Regexp(t, `"codec":\s+"pcm_s16le"`, buf.String())
	require.Regexp(t, `"sample_rate":\s+16000`, buf.String())
	require.Regexp(t, `"title":\s+"fixture"`, buf.String())
}

func TestAudioHandlerRejectsMissingInput(t *testing.T) {
	app := mountAudio(t, &fakeAudio{})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "input"},
		{Name: "output"},
		{Name: "format"},
		{Name: "start-seconds"},
		{Name: "end-seconds"},
	}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"output": "out.wav", "format": "wav", "start-seconds": "0"},
	})
	err := h.trim(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--input is required")
}

func TestNormalizeMapsMethodAndTarget(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.bin")
	out := filepath.Join(dir, "out.wav")
	require.NoError(t, os.WriteFile(in, []byte("NORM"), 0o600))

	app := mountAudio(t, &fakeAudio{
		normalize: func(req *audiov1.NormalizeRequest) (*audiov1.NormalizeResponse, error) {
			require.Equal(t, audiov1.NormalizationMethod_NORMALIZATION_METHOD_RMS, req.GetMethod())
			require.Equal(t, -18.0, req.GetTargetLufs())
			require.Equal(t, commonv1.AudioFormat_AUDIO_FORMAT_WAV, req.GetOutputFormat())
			return &audiov1.NormalizeResponse{Audio: []byte("NORMALIZED"), ContentType: "audio/wav"}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "input"},
		{Name: "output"},
		{Name: "format"},
		{Name: "method", Default: "ebu_r128"},
		{Name: "target-lufs", Default: "-16"},
		{Name: "output-format", Default: "wav"},
	}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"input": in, "output": out, "format": "wav",
			"method": "rms", "target-lufs": "-18", "output-format": "wav",
		},
	})
	require.NoError(t, h.normalize(ctx))

	written, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, []byte("NORMALIZED"), written)
}
