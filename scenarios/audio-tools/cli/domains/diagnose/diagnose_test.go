package diagnose

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

	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
	settconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings/settings_v1connect"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"

	"audio-tools/cli/internal/testutil"
)

type fakeSettings struct {
	settconnect.UnimplementedSettingsServiceHandler
	cfg func() (*settv1.ProviderConfig, error)
}

func (f *fakeSettings) GetProviderConfig(_ context.Context, _ *connect.Request[settv1.GetProviderConfigRequest]) (*connect.Response[settv1.GetProviderConfigResponse], error) {
	c, err := f.cfg()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&settv1.GetProviderConfigResponse{Config: c}), nil
}

type fakeTTS struct {
	ttsconnect.UnimplementedTTSServiceHandler
	status func() (*ttsv1.Status, error)
}

func (f *fakeTTS) GetStatus(_ context.Context, _ *connect.Request[ttsv1.GetStatusRequest]) (*connect.Response[ttsv1.GetStatusResponse], error) {
	s, err := f.status()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ttsv1.GetStatusResponse{Status: s}), nil
}

func mountDiagnose(t *testing.T, s settconnect.SettingsServiceHandler, tt ttsconnect.TTSServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	mux := http.NewServeMux()
	sp, sh := settconnect.NewSettingsServiceHandler(s)
	tp, th := ttsconnect.NewTTSServiceHandler(tt)
	mux.Handle(sp, sh)
	mux.Handle(tp, th)
	return testutil.NewTestApp(t, mux)
}

// Happy path: both upstream services answer; output contains routing
// flags and per-tier availability rows.
func TestProvidersHappyPath(t *testing.T) {
	app := mountDiagnose(t,
		&fakeSettings{cfg: func() (*settv1.ProviderConfig, error) {
			return &settv1.ProviderConfig{ByokEnabled: true, VrooliEnabled: false, LocalEnabled: true}, nil
		}},
		&fakeTTS{status: func() (*ttsv1.Status, error) {
			return &ttsv1.Status{Availability: []*ttsv1.ProviderAvailability{
				{Tier: "local", ProviderId: "kokoro", Available: true},
				{Tier: "byok", ProviderId: "openai-tts", Available: false},
			}}, nil
		}},
	)
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.providers(ctx))
	out := buf.String()
	require.Contains(t, out, "BYOK   enabled=true")
	require.Contains(t, out, "Vrooli enabled=false")
	require.Contains(t, out, "local")
	require.Contains(t, out, "kokoro")
	require.Contains(t, out, "up")
	require.Contains(t, out, "down")
}

// Error path: settings service is unavailable — handler stops before
// querying TTS and surfaces a wrapped "get provider config" error.
func TestProvidersSettingsError(t *testing.T) {
	app := mountDiagnose(t,
		&fakeSettings{cfg: func() (*settv1.ProviderConfig, error) {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("settings down"))
		}},
		&fakeTTS{status: func() (*ttsv1.Status, error) {
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
