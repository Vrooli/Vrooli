package settings

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
	settconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings/settings_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/shared"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"

	testutil "github.com/vrooli/cli-core/cliapptest"
)

type fakeProviderSettings struct {
	settconnect.UnimplementedSettingsServiceHandler
	cfg    func() (*settv1.ProviderConfig, error)
	list   func() (*settv1.ListBYOKCredentialsResponse, error)
	upsert func(*settv1.UpsertBYOKCredentialRequest) (*settv1.UpsertBYOKCredentialResponse, error)
	delete func(*settv1.DeleteBYOKCredentialRequest) error
}

func (f *fakeProviderSettings) ListBYOKCredentials(context.Context, *connect.Request[settv1.ListBYOKCredentialsRequest]) (*connect.Response[settv1.ListBYOKCredentialsResponse], error) {
	resp, err := f.list()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeProviderSettings) UpsertBYOKCredential(_ context.Context, req *connect.Request[settv1.UpsertBYOKCredentialRequest]) (*connect.Response[settv1.UpsertBYOKCredentialResponse], error) {
	resp, err := f.upsert(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeProviderSettings) DeleteBYOKCredential(_ context.Context, req *connect.Request[settv1.DeleteBYOKCredentialRequest]) (*connect.Response[settv1.DeleteBYOKCredentialResponse], error) {
	if err := f.delete(req.Msg); err != nil {
		return nil, err
	}
	return connect.NewResponse(&settv1.DeleteBYOKCredentialResponse{}), nil
}

func (f *fakeProviderSettings) GetProviderConfig(_ context.Context, _ *connect.Request[settv1.GetProviderConfigRequest]) (*connect.Response[settv1.GetProviderConfigResponse], error) {
	c, err := f.cfg()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&settv1.GetProviderConfigResponse{Config: c}), nil
}

type fakeProviderTTS struct {
	ttsconnect.UnimplementedTTSServiceHandler
	status func() (*ttsv1.Status, error)
}

func (f *fakeProviderTTS) GetStatus(_ context.Context, _ *connect.Request[ttsv1.GetStatusRequest]) (*connect.Response[ttsv1.GetStatusResponse], error) {
	s, err := f.status()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ttsv1.GetStatusResponse{Status: s}), nil
}

func mountProvidersMatrix(t *testing.T, s settconnect.SettingsServiceHandler, tt ttsconnect.TTSServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	mux := http.NewServeMux()
	sp, sh := settconnect.NewSettingsServiceHandler(s)
	tp, th := ttsconnect.NewTTSServiceHandler(tt)
	mux.Handle(sp, sh)
	mux.Handle(tp, th)
	return testutil.NewTestApp(t, mux)
}

func TestProvidersHappyPath(t *testing.T) {
	app := mountProvidersMatrix(t,
		&fakeProviderSettings{cfg: func() (*settv1.ProviderConfig, error) {
			return &settv1.ProviderConfig{ByokEnabled: true, VrooliEnabled: false, LocalEnabled: true}, nil
		}},
		&fakeProviderTTS{status: func() (*ttsv1.Status, error) {
			return &ttsv1.Status{Availability: []*sharedv1.ProviderHealth{
				{Tier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL, ProviderId: "kokoro", State: sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE},
				{Tier: commonv1.ProviderTier_PROVIDER_TIER_BYOK, ProviderId: "openai-tts", State: sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE},
				{Tier: commonv1.ProviderTier_PROVIDER_TIER_VROOLI, State: sharedv1.ProviderState_PROVIDER_STATE_UNKNOWN},
			}}, nil
		}},
	)
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.providers(ctx))
	out := buf.String()
	want := "Provider routing\n  BYOK   enabled=true\n  Vrooli enabled=false\n  Local  enabled=true\n\nTTS availability\n  local   kokoro       up\n  byok    openai-tts   down\n  vrooli  not-configured unknown\n"
	require.Equal(t, want, out)

	jsonCtx, jsonBuf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{JSON: true})
	require.NoError(t, h.providers(jsonCtx))
	require.Equal(t, want, jsonBuf.String())
}

func TestProvidersSettingsError(t *testing.T) {
	app := mountProvidersMatrix(t,
		&fakeProviderSettings{cfg: func() (*settv1.ProviderConfig, error) {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("settings down"))
		}},
		&fakeProviderTTS{status: func() (*ttsv1.Status, error) {
			t.Fatal("TTS GetStatus must not be called when settings fails")
			return nil, nil
		}},
	)
	h := newHandlers(app)
	ctx, _ := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	err := h.providers(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "get provider config"), "want operation tag, got %q", err.Error())
}

func TestSettingsProviderPrimitivePreservesOutputModes(t *testing.T) {
	settings := &fakeProviderSettings{cfg: func() (*settv1.ProviderConfig, error) {
		return &settv1.ProviderConfig{ByokEnabled: true, VrooliEnabled: false, LocalEnabled: true, WhisperUrl: "http://whisper", KokoroUrl: "http://kokoro", OllamaUrl: "http://ollama", LpbsBaseUrl: "http://lpbs"}, nil
	}}
	app := mountProvidersMatrix(t, settings, &fakeProviderTTS{status: func() (*ttsv1.Status, error) { return &ttsv1.Status{}, nil }})
	h := newHandlers(app)
	want := "BYOK   enabled=true\nVrooli enabled=false\nLocal  enabled=true\nWhisper URL: http://whisper\nKokoro URL:  http://kokoro\nOllama URL:  http://ollama\nLPBS Base:   http://lpbs\n"
	assertBothModes(t, app, h.provider, want)
}

func TestBYOKListPrimitivePreservesOutputModes(t *testing.T) {
	settings := &fakeProviderSettings{
		list: func() (*settv1.ListBYOKCredentialsResponse, error) {
			return &settv1.ListBYOKCredentialsResponse{Credentials: []*settv1.BYOKCredentialSummary{{ProviderId: "openai-tts", Capability: "tts", Fingerprint: "sk-***abcd"}}}, nil
		},
	}
	app := mountProvidersMatrix(t, settings, &fakeProviderTTS{status: func() (*ttsv1.Status, error) { return &ttsv1.Status{}, nil }})
	h := newHandlers(app)
	assertBothModes(t, app, h.byokList, "openai-tts           tts        sk-***abcd\n")
}

func TestBYOKListEmptyPrimitivePreservesOutputModes(t *testing.T) {
	settings := &fakeProviderSettings{list: func() (*settv1.ListBYOKCredentialsResponse, error) {
		return &settv1.ListBYOKCredentialsResponse{}, nil
	}}
	app := mountProvidersMatrix(t, settings, &fakeProviderTTS{status: func() (*ttsv1.Status, error) { return &ttsv1.Status{}, nil }})
	h := newHandlers(app)
	assertBothModes(t, app, h.byokList, "(no credentials)\n")
}

func TestBYOKUpsertPrimitivePreservesCredentialSemanticsAndOutputModes(t *testing.T) {
	const secret = "sensitive-api-key"
	var got *settv1.UpsertBYOKCredentialRequest
	settings := &fakeProviderSettings{upsert: func(req *settv1.UpsertBYOKCredentialRequest) (*settv1.UpsertBYOKCredentialResponse, error) {
		got = req
		return &settv1.UpsertBYOKCredentialResponse{Credential: &settv1.BYOKCredentialSummary{ProviderId: req.GetProviderId(), Capability: req.GetCapability(), Fingerprint: "sk-***abcd"}}, nil
	}}
	app := mountProvidersMatrix(t, settings, &fakeProviderTTS{status: func() (*ttsv1.Status, error) { return &ttsv1.Status{}, nil }})
	h := newHandlers(app)
	want := "Stored openai-tts/tts fingerprint=sk-***abcd\n"
	for _, jsonMode := range []bool{false, true} {
		schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "provider"}, {Name: "capability"}, {Name: "key-stdin", Bool: true}}}
		ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapp.TestRunContextOptions{JSON: jsonMode, Flags: map[string]string{"provider": "openai-tts", "capability": "tts"}, BoolFlags: map[string]bool{"key-stdin": true}})
		require.NoError(t, h.byokUpsertFrom(ctx, strings.NewReader(secret)))
		require.Equal(t, want, buf.String())
		require.NotContains(t, buf.String(), secret)
	}
	require.Equal(t, "openai-tts", got.GetProviderId())
	require.Equal(t, "tts", got.GetCapability())
	require.Equal(t, secret, got.GetApiKey())
	require.NotContains(t, want, secret)
}

func TestBYOKDeletePrimitivePreservesOutputModesAndRequest(t *testing.T) {
	var got *settv1.DeleteBYOKCredentialRequest
	settings := &fakeProviderSettings{delete: func(req *settv1.DeleteBYOKCredentialRequest) error { got = req; return nil }}
	app := mountProvidersMatrix(t, settings, &fakeProviderTTS{status: func() (*ttsv1.Status, error) { return &ttsv1.Status{}, nil }})
	h := newHandlers(app)
	assertBothModesWithFlags(t, app, h.byokDelete, "Deleted.\n", map[string]string{"provider": "openai-tts", "capability": "tts"})
	require.Equal(t, "openai-tts", got.GetProviderId())
	require.Equal(t, "tts", got.GetCapability())
}

func assertBothModes(t *testing.T, app *cliapp.ScenarioApp, run func(cliapp.RunContext) error, want string) {
	assertBothModesWithFlags(t, app, run, want, nil)
}

func assertBothModesWithFlags(t *testing.T, app *cliapp.ScenarioApp, run func(cliapp.RunContext) error, want string, flags map[string]string) {
	t.Helper()
	for _, jsonMode := range []bool{false, true} {
		schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "provider"}, {Name: "capability"}, {Name: "key-stdin", Bool: true}}}
		ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapp.TestRunContextOptions{JSON: jsonMode, Flags: flags})
		require.NoError(t, run(ctx))
		require.Equal(t, want, buf.String())
	}
}
