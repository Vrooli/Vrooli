package settings

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
func TestByokUpsertReadsSecretFromStdin(t *testing.T) {
	secret := "sk-abc"
	app := mountSettings(t, &fakeSvc{
		upsertFn: func(req *settv1.UpsertBYOKCredentialRequest) (*settv1.BYOKCredentialSummary, error) {
			require.Equal(t, "openai-tts", req.GetProviderId())
			require.Equal(t, "tts", req.GetCapability())
			require.Equal(t, secret, req.GetApiKey())
			return &settv1.BYOKCredentialSummary{
				ProviderId:  "openai-tts",
				Capability:  "tts",
				Fingerprint: "sk-***abc",
			}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "provider"}, {Name: "capability"}, {Name: "key-stdin", Bool: true}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags:     map[string]string{"provider": "openai-tts", "capability": "tts"},
		BoolFlags: map[string]bool{"key-stdin": true},
	})
	require.NoError(t, h.byokUpsertFrom(ctx, bytes.NewBufferString(secret+"\n")))
	require.Contains(t, buf.String(), "Stored openai-tts/tts fingerprint=sk-***abc")
	require.NotContains(t, buf.String(), secret)
}

func TestByokUpsertRejectsOversizedStdinWithoutEchoingSecret(t *testing.T) {
	secret := strings.Repeat("s", maxBYOKSecretBytes+1)
	app := mountSettings(t, &fakeSvc{upsertFn: func(*settv1.UpsertBYOKCredentialRequest) (*settv1.BYOKCredentialSummary, error) {
		t.Fatal("oversized secret must not be sent to the API")
		return nil, nil
	}})
	h := newHandlers(app)
	err := h.byokUpsertFrom(byokRunContext(t, app, true), strings.NewReader(secret))
	require.EqualError(t, err, "API key from stdin exceeds the 8 KiB limit")
	require.NotContains(t, err.Error(), secret)
}

func TestByokUpsertRequiresKeyStdinFlag(t *testing.T) {
	app := mountSettings(t, &fakeSvc{upsertFn: func(*settv1.UpsertBYOKCredentialRequest) (*settv1.BYOKCredentialSummary, error) {
		t.Fatal("missing key-stdin must not call the API")
		return nil, nil
	}})
	h := newHandlers(app)
	err := h.byokUpsertFrom(byokRunContext(t, app, false), strings.NewReader("secret"))
	require.EqualError(t, err, "provide the API key through stdin with --key-stdin")
	require.NotContains(t, err.Error(), "secret")
}

func byokRunContext(t *testing.T, app *cliapp.ScenarioApp, stdinFlag bool) cliapp.RunContext {
	t.Helper()
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "provider"}, {Name: "capability"}, {Name: "key-stdin", Bool: true}}}
	boolFlags := map[string]bool{}
	if stdinFlag {
		boolFlags["key-stdin"] = true
	}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"provider": "openai-tts", "capability": "tts"}, BoolFlags: boolFlags,
	})
	return ctx
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, fmt.Errorf("sensitive reader detail") }

func TestReadBYOKSecretReturnsGenericReadError(t *testing.T) {
	_, err := readBYOKSecret(failingReader{})
	require.EqualError(t, err, "could not read API key from stdin")
	require.NotContains(t, err.Error(), "sensitive reader detail")
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
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "provider"}, {Name: "capability"}, {Name: "key-stdin", Bool: true}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags:     map[string]string{"provider": "openai-tts", "capability": "tts"},
		BoolFlags: map[string]bool{"key-stdin": true},
	})
	err := h.byokUpsertFrom(ctx, bytes.NewBufferString("junk"))
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "upsert byok"), "want operation tag, got %q", err.Error())
}
