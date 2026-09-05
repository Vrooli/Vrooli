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

	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
	settconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings/settings_v1connect"

	testutil "github.com/vrooli/cli-core/cliapptest"
)

type fakeSvc struct {
	settconnect.UnimplementedSettingsServiceHandler
	upsertFn func(*settv1.UpsertBYOKCredentialRequest) (*settv1.BYOKCredentialSummary, error)
	listFn   func() ([]*settv1.BYOKCredentialSummary, error)
}

func (f *fakeSvc) UpsertBYOKCredential(_ context.Context, req *connect.Request[settv1.UpsertBYOKCredentialRequest]) (*connect.Response[settv1.UpsertBYOKCredentialResponse], error) {
	c, err := f.upsertFn(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&settv1.UpsertBYOKCredentialResponse{Credential: c}), nil
}

func (f *fakeSvc) ListBYOKCredentials(_ context.Context, _ *connect.Request[settv1.ListBYOKCredentialsRequest]) (*connect.Response[settv1.ListBYOKCredentialsResponse], error) {
	creds, err := f.listFn()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&settv1.ListBYOKCredentialsResponse{Credentials: creds}), nil
}

func mountSettings(t *testing.T, svc settconnect.SettingsServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, h := settconnect.NewSettingsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	return testutil.NewTestApp(t, mux)
}

// Happy path: byok-upsert forwards provider/capability/key, prints the
// returned fingerprint.
func TestByokUpsertHappyPath(t *testing.T) {
	app := mountSettings(t, &fakeSvc{
		upsertFn: func(req *settv1.UpsertBYOKCredentialRequest) (*settv1.BYOKCredentialSummary, error) {
			require.Equal(t, "openai-tts", req.GetProviderId())
			require.Equal(t, "tts", req.GetCapability())
			require.Equal(t, "sk-abc", req.GetApiKey())
			return &settv1.BYOKCredentialSummary{
				ProviderId:  "openai-tts",
				Capability:  "tts",
				Fingerprint: "sk-***abc",
			}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "provider"}, {Name: "capability"}, {Name: "key"}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"provider": "openai-tts", "capability": "tts", "key": "sk-abc"},
	})
	require.NoError(t, h.byokUpsert(ctx))
	require.Contains(t, buf.String(), "Stored openai-tts/tts fingerprint=sk-***abc")
}

// Happy path: empty list — handler prints the "(no credentials)" sentinel.
func TestByokListEmpty(t *testing.T) {
	app := mountSettings(t, &fakeSvc{
		listFn: func() ([]*settv1.BYOKCredentialSummary, error) { return nil, nil },
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.byokList(ctx))
	require.Contains(t, buf.String(), "(no credentials)")
}

// Error path: upsert rejects an invalid key — handler surfaces the
// wrapped Connect error with the "upsert byok" operation tag.
func TestByokUpsertInvalidKey(t *testing.T) {
	app := mountSettings(t, &fakeSvc{
		upsertFn: func(_ *settv1.UpsertBYOKCredentialRequest) (*settv1.BYOKCredentialSummary, error) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("api key looks malformed"))
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "provider"}, {Name: "capability"}, {Name: "key"}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"provider": "openai-tts", "capability": "tts", "key": "junk"},
	})
	err := h.byokUpsert(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "upsert byok"), "want operation tag, got %q", err.Error())
}
