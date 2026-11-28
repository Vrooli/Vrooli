package stt

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

func TestEngineImpactPrimitiveHumanAndJSONOutput(t *testing.T) {
	var gotEngine string
	app := mountSTT(t, &fakeSvc{
		engineImpact: func(req *sttv1.GetEngineSwitchImpactRequest) (*sttv1.GetEngineSwitchImpactResponse, error) {
			gotEngine = req.GetFromEngineId()
			return &sttv1.GetEngineSwitchImpactResponse{Resource: "whisper", ConsumersKnown: true}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "engine"}}}
	humanCtx, humanBuf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"engine": "whisper-local"},
	})
	require.NoError(t, h.engineImpactPrimitive().Run(humanCtx))
	require.Contains(t, humanBuf.String(), "Switching away from \"whisper-local\" would stop using resource \"whisper\".")

	jsonCtx, jsonBuf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"engine": "whisper-local"}, JSON: true,
	})
	require.NoError(t, h.engineImpactPrimitive().Run(jsonCtx))
	var response sttv1.GetEngineSwitchImpactResponse
	require.NoError(t, protojson.Unmarshal(jsonBuf.Bytes(), &response))
	require.Equal(t, "whisper", response.GetResource())
	require.True(t, response.GetConsumersKnown())
	require.NotContains(t, jsonBuf.String(), "Switching away from")
	require.Equal(t, "whisper-local", gotEngine)
}

func mountSTTH2C(t *testing.T, svc *fakeSvc) *cliapp.ScenarioApp {
	t.Helper()
	runtimePath, runtimeHandler := sttconnect.NewSTTServiceHandler(svc)
	adminPath, adminHandler := sttconnect.NewSTTAdminServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(runtimePath, runtimeHandler)
	mux.Handle(adminPath, adminHandler)
	srv := httptest.NewUnstartedServer(mux)
	srv.Config.Protocols = new(http.Protocols)
	srv.Config.Protocols.SetHTTP1(true)
	srv.Config.Protocols.SetUnencryptedHTTP2(true)
	srv.Start()
	t.Cleanup(srv.Close)

	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:           "scenario-test",
		Version:        "0.0.0-test",
		Description:    "scenario CLI test",
		DefaultAPIBase: srv.URL,
		AllowAnonymous: true,
	})
	require.NoError(t, err)
	return core
}

// Happy path: transcribe reads file bytes, defaults format to "wav",
// returns the trace + transcript.

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

func TestStreamConfigGetJSONPreservesLegacyText(t *testing.T) {
	app := mountSTT(t, &fakeSvc{getCfg: func() (*sttv1.StreamConfig, error) {
		return &sttv1.StreamConfig{StreamingMode: sttv1.StreamingMode_STREAMING_MODE_AUTO}, nil
	}})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{JSON: true})
	require.NoError(t, h.streamConfigGet(ctx))
	require.Contains(t, buf.String(), "Streaming STT pipeline:\n")
	require.NotContains(t, buf.String(), "\"config\"")
}

// speaker-config builds a field mask from the provided flags and prints the
// resolved config. --mode/--threshold/--enabled must map to the right paths.

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

func TestFormatsJSONUsesCanonicalProtoShape(t *testing.T) {
	app := mountSTT(t, &fakeSvc{getFormats: func() *sttv1.GetSupportedFormatsResponse {
		return &sttv1.GetSupportedFormatsResponse{
			AcceptedFormats: []commonv1.AudioFormat{commonv1.AudioFormat_AUDIO_FORMAT_WAV},
			FfmpegAvailable: true, CanonicalSampleRateHz: 16000, CanonicalChannels: 1,
		}
	}})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{JSON: true})
	require.NoError(t, h.formats(ctx))
	var response sttv1.GetSupportedFormatsResponse
	require.NoError(t, protojson.Unmarshal(buf.Bytes(), &response))
	require.Equal(t, []commonv1.AudioFormat{commonv1.AudioFormat_AUDIO_FORMAT_WAV}, response.GetAcceptedFormats())
	require.True(t, response.GetFfmpegAvailable())
	require.NotContains(t, buf.String(), "STT ingress — accepted input formats:")
}

func TestEnginesJSONUsesCanonicalProtoShape(t *testing.T) {
	app := mountSTT(t, &fakeSvc{listEngines: func() *sttv1.ListEnginesResponse {
		return &sttv1.ListEnginesResponse{Engines: []*sttv1.EngineInfo{{
			Id: "sherpa-streaming", DisplayName: "Sherpa Zipformer", Available: true,
			NativeStreaming: true, IsActive: true, Verdict: "selected", Reason: "operator_pin",
		}}}
	}})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{JSON: true})
	require.NoError(t, h.enginesPrimitive().Run(ctx))
	var response sttv1.ListEnginesResponse
	require.NoError(t, protojson.Unmarshal(buf.Bytes(), &response))
	require.Len(t, response.GetEngines(), 1)
	engine := response.GetEngines()[0]
	require.Equal(t, "sherpa-streaming", engine.GetId())
	require.True(t, engine.GetAvailable())
	require.True(t, engine.GetIsActive())
	require.NotContains(t, buf.String(), "STT engines (manifest-derived):")
}

// Error path: transcribe fails on the server — handler returns wrapped error
// and the original file is not touched (no side effects on input).
