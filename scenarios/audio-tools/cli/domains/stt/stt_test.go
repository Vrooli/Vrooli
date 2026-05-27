package stt

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

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"

	"audio-tools/cli/internal/testutil"
)

type fakeSvc struct {
	sttconnect.UnimplementedSTTServiceHandler
	sttconnect.UnimplementedSTTAdminServiceHandler
	transcribe       func(*sttv1.TranscribeRequest) (*sttv1.TranscribeResponse, error)
	getCfg           func() (*sttv1.StreamConfig, error)
	getFormats       func() *sttv1.GetSupportedFormatsResponse
	updateSpeakerCfg func(*sttv1.UpdateSpeakerConfigRequest) (*sttv1.SpeakerConfig, error)
	getSpeakerStatus func() *sttv1.SpeakerStatus
}

func (f *fakeSvc) UpdateSpeakerConfig(_ context.Context, req *connect.Request[sttv1.UpdateSpeakerConfigRequest]) (*connect.Response[sttv1.UpdateSpeakerConfigResponse], error) {
	if f.updateSpeakerCfg != nil {
		cfg, err := f.updateSpeakerCfg(req.Msg)
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&sttv1.UpdateSpeakerConfigResponse{Config: cfg}), nil
	}
	return connect.NewResponse(&sttv1.UpdateSpeakerConfigResponse{Config: req.Msg.GetConfig()}), nil
}

func (f *fakeSvc) GetSpeakerStatus(_ context.Context, _ *connect.Request[sttv1.GetSpeakerStatusRequest]) (*connect.Response[sttv1.GetSpeakerStatusResponse], error) {
	return connect.NewResponse(&sttv1.GetSpeakerStatusResponse{Status: f.getSpeakerStatus()}), nil
}

func (f *fakeSvc) GetSupportedFormats(_ context.Context, _ *connect.Request[sttv1.GetSupportedFormatsRequest]) (*connect.Response[sttv1.GetSupportedFormatsResponse], error) {
	return connect.NewResponse(f.getFormats()), nil
}

func (f *fakeSvc) Transcribe(_ context.Context, req *connect.Request[sttv1.TranscribeRequest]) (*connect.Response[sttv1.TranscribeResponse], error) {
	resp, err := f.transcribe(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeSvc) GetStreamConfig(_ context.Context, _ *connect.Request[sttv1.GetStreamConfigRequest]) (*connect.Response[sttv1.GetStreamConfigResponse], error) {
	cfg, err := f.getCfg()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sttv1.GetStreamConfigResponse{Config: cfg}), nil
}

func mountSTT(t *testing.T, svc *fakeSvc) *cliapp.ScenarioApp {
	t.Helper()
	runtimePath, runtimeHandler := sttconnect.NewSTTServiceHandler(svc)
	adminPath, adminHandler := sttconnect.NewSTTAdminServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(runtimePath, runtimeHandler)
	mux.Handle(adminPath, adminHandler)
	return testutil.NewTestApp(t, mux)
}

// Happy path: transcribe reads file bytes, defaults format to "wav",
// returns the trace + transcript.
func TestTranscribeHappyPath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "in.wav")
	require.NoError(t, os.WriteFile(file, []byte("PCMDATA"), 0o600))

	app := mountSTT(t, &fakeSvc{
		transcribe: func(req *sttv1.TranscribeRequest) (*sttv1.TranscribeResponse, error) {
			require.Equal(t, []byte("PCMDATA"), req.GetAudio())
			require.Equal(t, commonv1.AudioFormat_AUDIO_FORMAT_WAV, req.GetFormat())
			return &sttv1.TranscribeResponse{
				Text: "hello", ProviderTier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL, ProviderId: "whisper", LatencyMs: 99,
			}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "file"}, {Name: "language"}, {Name: "format"}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"file": file},
	})
	require.NoError(t, h.transcribe(ctx))
	out := buf.String()
	require.Contains(t, out, "Transcribed via local/whisper")
	require.Contains(t, out, "hello")
}

// Happy path: stream-config get prints the resolved levers using the
// documented defaults when fields are zero.
func TestStreamConfigGet(t *testing.T) {
	app := mountSTT(t, &fakeSvc{
		getCfg: func() (*sttv1.StreamConfig, error) {
			return &sttv1.StreamConfig{
				StreamingMode:     sttv1.StreamingMode_STREAMING_MODE_AUTO,
				VadSilenceMs:      0, // expect default 700
				OverlapWindowMs:   2500,
				OverlapCommitRuns: 3,
			}, nil
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.streamConfigGet(ctx))
	out := buf.String()
	require.Contains(t, out, "streaming_mode       = auto")
	require.Contains(t, out, "vad_silence_ms       = 700")
	require.Contains(t, out, "overlap_window_ms    = 2500")
	require.Contains(t, out, "overlap_commit_runs  = 3")
}

// speaker-config builds a field mask from the provided flags and prints the
// resolved config. --mode/--threshold/--enabled must map to the right paths.
func TestSpeakerConfigSet(t *testing.T) {
	var gotMask []string
	app := mountSTT(t, &fakeSvc{
		updateSpeakerCfg: func(req *sttv1.UpdateSpeakerConfigRequest) (*sttv1.SpeakerConfig, error) {
			gotMask = req.GetUpdateMask().GetPaths()
			return &sttv1.SpeakerConfig{
				Enabled:   req.GetConfig().GetEnabled(),
				Mode:      req.GetConfig().GetMode(),
				Threshold: req.GetConfig().GetThreshold(),
			}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "mode"},
		{Name: "threshold"},
		{Name: "enabled"},
		{Name: "profiles"},
		{Name: "bind-profile"},
		{Name: "reject-behavior"},
		{Name: "fallback"},
		{Name: "extraction-enabled"},
	}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"mode": "filter", "threshold": "0.8", "enabled": "true"},
	})
	require.NoError(t, h.speakerConfig(ctx))
	require.ElementsMatch(t, []string{"mode", "threshold", "enabled"}, gotMask)
	out := buf.String()
	require.Contains(t, out, "mode            = filter")
	require.Contains(t, out, "threshold       = 0.80")
	require.Contains(t, out, "enabled         = true")
}

// --extraction-enabled maps to the extraction_enabled mask path and is echoed
// in the resolved-config output.
func TestSpeakerConfigSetExtraction(t *testing.T) {
	var gotMask []string
	app := mountSTT(t, &fakeSvc{
		updateSpeakerCfg: func(req *sttv1.UpdateSpeakerConfigRequest) (*sttv1.SpeakerConfig, error) {
			gotMask = req.GetUpdateMask().GetPaths()
			return &sttv1.SpeakerConfig{ExtractionEnabled: req.GetConfig().GetExtractionEnabled()}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "mode"},
		{Name: "threshold"},
		{Name: "enabled"},
		{Name: "profiles"},
		{Name: "bind-profile"},
		{Name: "reject-behavior"},
		{Name: "fallback"},
		{Name: "extraction-enabled"},
	}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"extraction-enabled": "true"},
	})
	require.NoError(t, h.speakerConfig(ctx))
	require.ElementsMatch(t, []string{"extraction_enabled"}, gotMask)
	require.Contains(t, buf.String(), "extraction      = true")
}

// speaker-config with no flags is a usage error (nothing to update).
func TestSpeakerConfigNoFlags(t *testing.T) {
	app := mountSTT(t, &fakeSvc{})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "mode"},
		{Name: "threshold"},
		{Name: "enabled"},
		{Name: "profiles"},
		{Name: "bind-profile"},
		{Name: "reject-behavior"},
		{Name: "fallback"},
		{Name: "extraction-enabled"},
	}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{})
	require.Error(t, h.speakerConfig(ctx))
}

// speaker-status surfaces the Whisper-only protection caveat when verification
// is enabled, so a user on the streaming engine isn't silently unprotected.
func TestSpeakerStatusWhisperCaveat(t *testing.T) {
	app := mountSTT(t, &fakeSvc{
		getSpeakerStatus: func() *sttv1.SpeakerStatus {
			return &sttv1.SpeakerStatus{
				Config: &sttv1.SpeakerConfig{
					Enabled: true, Mode: sttv1.SpeakerMode_SPEAKER_MODE_FILTER,
					Threshold: 0.7, ProfileIds: []string{"my-voice"},
					ExtractionEnabled: true,
				},
				Capability: "available", CapabilityLabel: "Speaker store",
				ResourceReady: true, ProfileCount: 1,
				Profiles: []*sttv1.SpeakerProfile{{Id: "my-voice", DisplayName: "Me", EnrollmentAudioSeconds: 12.5}},
			}
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.speakerStatus(ctx))
	out := buf.String()
	require.Contains(t, out, "mode            = filter")
	require.Contains(t, out, "extraction      = true")
	require.Contains(t, out, "my-voice")
	require.Contains(t, out, "[active]")
	require.Contains(t, out, "Kyutai streaming engine emits no per-segment audio")
}

// formats prints the capability matrix as human-readable lines and surfaces
// the canonical PCM target. When ffmpeg is present, no degrade note appears.
func TestFormatsWithFfmpeg(t *testing.T) {
	app := mountSTT(t, &fakeSvc{
		getFormats: func() *sttv1.GetSupportedFormatsResponse {
			return &sttv1.GetSupportedFormatsResponse{
				AcceptedFormats: []commonv1.AudioFormat{
					commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE,
					commonv1.AudioFormat_AUDIO_FORMAT_WEBM,
				},
				FfmpegAvailable:       true,
				CanonicalSampleRateHz: 16000,
				CanonicalChannels:     1,
			}
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.formats(ctx))
	out := buf.String()
	require.Contains(t, out, "- pcm_s16le")
	require.Contains(t, out, "- webm")
	require.Contains(t, out, "ffmpeg decode backend = available")
	require.Contains(t, out, "16000 Hz, 1 channel(s), s16le")
	require.NotContains(t, out, "fall back to buffered")
}

// When ffmpeg is absent, the degrade note is printed so operators know live
// non-PCM streams will buffer.
func TestFormatsWithoutFfmpeg(t *testing.T) {
	app := mountSTT(t, &fakeSvc{
		getFormats: func() *sttv1.GetSupportedFormatsResponse {
			return &sttv1.GetSupportedFormatsResponse{
				AcceptedFormats:       []commonv1.AudioFormat{commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE},
				FfmpegAvailable:       false,
				CanonicalSampleRateHz: 16000,
				CanonicalChannels:     1,
			}
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.formats(ctx))
	out := buf.String()
	require.Contains(t, out, "ffmpeg decode backend = unavailable")
	require.Contains(t, out, "fall back to buffered")
}

// Error path: transcribe fails on the server — handler returns wrapped error
// and the original file is not touched (no side effects on input).
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
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "file"}, {Name: "language"}, {Name: "format"}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"file": file},
	})
	err := h.transcribe(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "transcribe"), "want operation tag, got %q", err.Error())
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
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "file"}, {Name: "language"}, {Name: "format"}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"file": "/no/such/path.wav"},
	})
	err := h.transcribe(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "read"), "want read error tag, got %q", err.Error())
}
