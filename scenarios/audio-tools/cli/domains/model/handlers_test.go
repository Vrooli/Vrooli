package model

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models"
	modelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models/models_v1connect"
)

type fakeSvc struct {
	modelsconnect.UnimplementedModelRegistryServiceHandler
	list    func(*modelsv1.ListModelsRequest) (*modelsv1.ListModelsResponse, error)
	resolve func(*modelsv1.ResolveModelRequest) (*modelsv1.ResolveModelResponse, error)
}

func (f *fakeSvc) ListModels(_ context.Context, req *connect.Request[modelsv1.ListModelsRequest]) (*connect.Response[modelsv1.ListModelsResponse], error) {
	resp, err := f.list(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeSvc) ResolveModel(_ context.Context, req *connect.Request[modelsv1.ResolveModelRequest]) (*connect.Response[modelsv1.ResolveModelResponse], error) {
	resp, err := f.resolve(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func mountModel(t *testing.T, svc modelsconnect.ModelRegistryServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, handler := modelsconnect.NewModelRegistryServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return cliapptest.NewTestApp(t, mux)
}

// modelSchema derives the runtime argument schema from the manifest command,
// so handler tests exercise the same flag contract the CLI dispatcher uses.
func modelSchema(t *testing.T, command string) cliapp.ArgSchema {
	t.Helper()
	manifest, err := cliapp.ParseManifest(cliapptest.ReadManifest(t))
	require.NoError(t, err)
	group := manifest.FindGroup(GroupName)
	require.NotNil(t, group, "manifest group %q must exist", GroupName)
	for _, candidate := range group.Commands {
		if candidate.Name != command {
			continue
		}
		schema, err := cliapp.ManifestArgs(candidate)
		require.NoError(t, err)
		return schema
	}
	t.Fatalf("manifest group %q has no command %q", GroupName, command)
	return cliapp.ArgSchema{}
}

func sampleRecord() *modelsv1.ModelRecord {
	return &modelsv1.ModelRecord{
		Id:              "whisper-large-v3",
		Name:            "Whisper Large v3",
		Capability:      "stt",
		Runtime:         "faster-whisper",
		Languages:       []string{"en", "es"},
		License:         "MIT",
		SizeMbApprox:    1530,
		CommercialUse:   modelsv1.CommercialUse_COMMERCIAL_USE_ALLOWED,
		Readiness:       modelsv1.Readiness_READINESS_AVAILABLE,
		Roles:           []string{"default", "fast"},
		DefaultEligible: true,
		IsDefault:       true,
	}
}

func TestListHappyPathRendersTable(t *testing.T) {
	app := mountModel(t, &fakeSvc{
		list: func(req *modelsv1.ListModelsRequest) (*modelsv1.ListModelsResponse, error) {
			require.Equal(t, "stt", req.GetCapability())
			require.Equal(t, "en-US", req.GetLanguage())
			require.True(t, req.GetCommercialOnly())
			require.True(t, req.GetOnlyReady())
			return &modelsv1.ListModelsResponse{Models: []*modelsv1.ModelRecord{
				sampleRecord(),
				{
					Id:            "kyutai-stt-1b",
					Capability:    "stt",
					Runtime:       "kyutai",
					License:       "CC-BY-4.0",
					CommercialUse: modelsv1.CommercialUse_COMMERCIAL_USE_UNKNOWN,
					Readiness:     modelsv1.Readiness_READINESS_CANDIDATE,
				},
			}}, nil
		},
	})

	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, modelSchema(t, "list"), cliapptest.TestRunContextOptions{
		Flags:     map[string]string{"capability": "stt", "language": "en-US"},
		BoolFlags: map[string]bool{"commercial-only": true, "only-ready": true},
	})
	require.NoError(t, modelBindings(h)["ModelRegistryService.ListModels"].Run(ctx))

	out := buf.String()
	for _, want := range []string{"ID", "CAPABILITY", "READINESS", "COMMERCIAL", "LICENSE", "SIZE", "DEFAULT"} {
		require.Contains(t, out, want)
	}
	require.Contains(t, out, "whisper-large-v3")
	require.Contains(t, out, "kyutai-stt-1b")
	require.Contains(t, out, "available")
	require.Contains(t, out, "candidate")
	require.Contains(t, out, "allowed")
	require.Contains(t, out, "unknown")
	require.Contains(t, out, "MIT")
	require.Contains(t, out, "1530 MB")
	require.Contains(t, out, "Defaults: stt=whisper-large-v3 (1530 MB)")
	require.Contains(t, out, "yes")
}

func TestListJSONIsProtoWireShape(t *testing.T) {
	app := mountModel(t, &fakeSvc{
		list: func(_ *modelsv1.ListModelsRequest) (*modelsv1.ListModelsResponse, error) {
			return &modelsv1.ListModelsResponse{Models: []*modelsv1.ModelRecord{sampleRecord()}}, nil
		},
	})

	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, modelSchema(t, "list"), cliapptest.TestRunContextOptions{JSON: true})
	require.NoError(t, modelBindings(h)["ModelRegistryService.ListModels"].Run(ctx))

	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Contains(t, got, "models")
	models, ok := got["models"].([]any)
	require.True(t, ok, "models must be a JSON array")
	require.Len(t, models, 1)
	record, ok := models[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "whisper-large-v3", record["id"])
	require.Equal(t, "COMMERCIAL_USE_ALLOWED", record["commercial_use"])
	require.Equal(t, "READINESS_AVAILABLE", record["readiness"])
	require.Equal(t, true, record["default_eligible"])
}

func TestListAPIErrorIsWrapped(t *testing.T) {
	app := mountModel(t, &fakeSvc{
		list: func(_ *modelsv1.ListModelsRequest) (*modelsv1.ListModelsResponse, error) {
			return nil, connect.NewError(connect.CodeInternal, errors.New("registry unavailable"))
		},
	})

	h := newHandlers(app)
	ctx, _ := cliapptest.NewCapturedRunContext(app, modelSchema(t, "list"), cliapptest.TestRunContextOptions{})
	err := modelBindings(h)["ModelRegistryService.ListModels"].Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "model list")
	require.Contains(t, err.Error(), "internal")
}

func TestResolveHappyPathShowsFacts(t *testing.T) {
	app := mountModel(t, &fakeSvc{
		resolve: func(req *modelsv1.ResolveModelRequest) (*modelsv1.ResolveModelResponse, error) {
			require.Equal(t, "tts", req.GetCapability())
			require.Empty(t, req.GetLanguage())
			require.True(t, req.GetCommercial())
			record := sampleRecord()
			record.Capability = "tts"
			record.Notes = "Qualified Kokoro default; conditional Kale Nano product route runs first when healthy, with Kokoro as fallback. Kale remains an unqualified candidate."
			return &modelsv1.ResolveModelResponse{
				Model:       record,
				Explanation: "selected: readiness=available, roles=default,fast",
			}, nil
		},
	})

	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, modelSchema(t, "resolve"), cliapptest.TestRunContextOptions{
		Flags:     map[string]string{"capability": "tts"},
		BoolFlags: map[string]bool{"commercial": true},
	})
	require.NoError(t, modelBindings(h)["ModelRegistryService.ResolveModel"].Run(ctx))

	out := buf.String()
	require.Contains(t, out, "Resolved model: whisper-large-v3")
	require.Contains(t, out, "TTS route notes: Qualified Kokoro default; conditional Kale Nano product route runs first when healthy, with Kokoro as fallback. Kale remains an unqualified candidate.")
	require.Contains(t, out, "selected: readiness=available, roles=default,fast")
	require.Contains(t, out, "runtime:")
	require.Contains(t, out, "faster-whisper")
	require.Contains(t, out, "readiness:")
	require.Contains(t, out, "available")
	require.Contains(t, out, "license:")
	require.Contains(t, out, "MIT")
	require.Contains(t, out, "roles:")
	require.Contains(t, out, "default,fast")
}

func TestResolveCanRequestEngineScopedDefault(t *testing.T) {
	app := mountModel(t, &fakeSvc{
		resolve: func(req *modelsv1.ResolveModelRequest) (*modelsv1.ResolveModelResponse, error) {
			require.Equal(t, "stt", req.GetCapability())
			require.Equal(t, "sherpa-streaming", req.GetEngineId())
			return &modelsv1.ResolveModelResponse{
			Model:       &modelsv1.ModelRecord{Id: "sherpa-zipformer-en-2023-06-26", Capability: "stt", IsDefault: true, EngineDefaults: []string{"sherpa-streaming"}},
				Explanation: "selected sherpa-zipformer-en-2023-06-26 for capability=stt engine=sherpa-streaming",
			}, nil
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, modelSchema(t, "resolve"), cliapptest.TestRunContextOptions{
		Flags: map[string]string{"capability": "stt", "engine": "sherpa-streaming"},
	})
	require.NoError(t, modelBindings(h)["ModelRegistryService.ResolveModel"].Run(ctx))
	require.Contains(t, buf.String(), "Resolved model: sherpa-zipformer-en-2023-06-26")
	require.Contains(t, buf.String(), "Engine-scoped resolution: sherpa-streaming")
	require.Contains(t, buf.String(), "engine defaults: sherpa-streaming")
}

func TestResolveJSONIsProtoWireShape(t *testing.T) {
	app := mountModel(t, &fakeSvc{
		resolve: func(_ *modelsv1.ResolveModelRequest) (*modelsv1.ResolveModelResponse, error) {
			return &modelsv1.ResolveModelResponse{Model: sampleRecord(), Explanation: "selected"}, nil
		},
	})

	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, modelSchema(t, "resolve"), cliapptest.TestRunContextOptions{
		Flags: map[string]string{"capability": "stt"},
		JSON:  true,
	})
	require.NoError(t, modelBindings(h)["ModelRegistryService.ResolveModel"].Run(ctx))

	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Contains(t, got, "model")
	require.Equal(t, "selected", got["explanation"])
}

func TestResolveNotFoundIsNonZeroError(t *testing.T) {
	app := mountModel(t, &fakeSvc{
		resolve: func(_ *modelsv1.ResolveModelRequest) (*modelsv1.ResolveModelResponse, error) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("no eligible model"))
		},
	})

	h := newHandlers(app)
	ctx, _ := cliapptest.NewCapturedRunContext(app, modelSchema(t, "resolve"), cliapptest.TestRunContextOptions{
		Flags: map[string]string{"capability": "speaker"},
	})
	err := modelBindings(h)["ModelRegistryService.ResolveModel"].Run(ctx)
	require.Error(t, err, "NOT_FOUND must become a non-zero CLI error")
	require.Contains(t, err.Error(), "model resolve")
	require.Contains(t, err.Error(), "not_found")
	require.Contains(t, err.Error(), "no eligible model")
}

func TestResolveRequiresCapability(t *testing.T) {
	schema := modelSchema(t, "resolve")
	var stdout, stderr strings.Builder
	_, err := cliapptest.NewTestRunContextFromArgs(schema, []string{}, &cliapp.ScenarioApp{}, &stdout, &stderr)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required flag --capability")
}

func TestFormatRecordRowTruncatesLongFields(t *testing.T) {
	record := &modelsv1.ModelRecord{
		Id:            "this-model-id-is-far-too-long-for-the-column",
		Capability:    "stt",
		License:       "a-very-long-license-name-that-must-be-truncated",
		Readiness:     modelsv1.Readiness_READINESS_AVAILABLE,
		CommercialUse: modelsv1.CommercialUse_COMMERCIAL_USE_ALLOWED,
	}
	row := formatRecordRow(record)
	require.Contains(t, row, "…")
	require.True(t, strings.HasPrefix(row, "this-model-id-is-far-too-long…"), "got %q", row)
	require.NotContains(t, row, "must-be-truncated")
}
