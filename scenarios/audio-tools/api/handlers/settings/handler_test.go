package settings_test

import (
	"context"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	settingsH "audio-tools/handlers/settings"
	"audio-tools/internal/byokstore"
	"audio-tools/internal/logx"
	intsettings "audio-tools/internal/settings"
	"audio-tools/internal/store"
	"audio-tools/internal/testutil/db"

	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
	settconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings/settings_v1connect"
)

func newServer(t *testing.T) (string, settconnect.SettingsServiceClient) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(intsettings.Schema), apidb.SchemaProviderFunc(byokstore.Schema)))
	routed := apidb.NewFromPrimary(d)

	k := make([]byte, 32)
	_, _ = rand.Read(k)
	enc, err := byokstore.NewEncryptor(k)
	require.NoError(t, err)

	m := settingsH.Module(settingsH.Deps{
		ProviderConfig: store.NewProviderConfigStore(routed, store.ProviderConfig{BYOKEnabled: true, LocalEnabled: true, AvailTTLBYOKSeconds: 300, AvailTTLVrooliSecs: 30}),
		BYOK:           byokstore.New(enc, store.NewBYOKStore(routed)),
		VoiceOverrides: store.NewVoiceOverrideStore(routed),
		Logger:         logx.Std{},
	})
	r := mux.NewRouter()
	m.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	c := settconnect.NewSettingsServiceClient(http.DefaultClient, srv.URL)
	return srv.URL, c
}

func TestSettings_ProviderConfigRoundTrip(t *testing.T) {
	_, c := newServer(t)
	ctx := context.Background()
	res, err := c.GetProviderConfig(ctx, connect.NewRequest(&settv1.GetProviderConfigRequest{}))
	require.NoError(t, err)
	require.True(t, res.Msg.GetConfig().GetByokEnabled())

	upd, err := c.UpdateProviderConfig(ctx, connect.NewRequest(&settv1.UpdateProviderConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"byok_enabled", "whisper_url"}},
		Config: &settv1.ProviderConfig{
			ByokEnabled: false,
			WhisperUrl:  "http://w2",
		},
	}))
	require.NoError(t, err)
	require.False(t, upd.Msg.GetConfig().GetByokEnabled())
	require.Equal(t, "http://w2", upd.Msg.GetConfig().GetWhisperUrl())
}

func TestSettings_BYOKLifecycle(t *testing.T) {
	_, c := newServer(t)
	ctx := context.Background()

	list, err := c.ListBYOKCredentials(ctx, connect.NewRequest(&settv1.ListBYOKCredentialsRequest{}))
	require.NoError(t, err)
	require.Len(t, list.Msg.GetCredentials(), 0)

	up, err := c.UpsertBYOKCredential(ctx, connect.NewRequest(&settv1.UpsertBYOKCredentialRequest{
		ProviderId: "openai-tts", Capability: "tts",
		Secret: &settv1.UpsertBYOKCredentialRequest_ApiKey{ApiKey: "sk-secretsecret"},
	}))
	require.NoError(t, err)
	require.Contains(t, up.Msg.GetCredential().GetFingerprint(), "***")

	list, err = c.ListBYOKCredentials(ctx, connect.NewRequest(&settv1.ListBYOKCredentialsRequest{}))
	require.NoError(t, err)
	require.Len(t, list.Msg.GetCredentials(), 1)

	_, err = c.DeleteBYOKCredential(ctx, connect.NewRequest(&settv1.DeleteBYOKCredentialRequest{
		ProviderId: "openai-tts", Capability: "tts",
	}))
	require.NoError(t, err)

	list, err = c.ListBYOKCredentials(ctx, connect.NewRequest(&settv1.ListBYOKCredentialsRequest{}))
	require.NoError(t, err)
	require.Len(t, list.Msg.GetCredentials(), 0)
}

func TestSettings_BYOKInvalidCapability(t *testing.T) {
	_, c := newServer(t)
	_, err := c.UpsertBYOKCredential(context.Background(), connect.NewRequest(&settv1.UpsertBYOKCredentialRequest{
		ProviderId: "x", Capability: "bogus",
		Secret: &settv1.UpsertBYOKCredentialRequest_ApiKey{ApiKey: "k"},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestSettings_VoiceOverrides(t *testing.T) {
	_, c := newServer(t)
	ctx := context.Background()
	_, err := c.SetVoiceOverride(ctx, connect.NewRequest(&settv1.SetVoiceOverrideRequest{
		Override: &settv1.VoiceOverride{CanonicalVoice: "voice.feminine.warm", TierProvider: "byok:elevenlabs", AdapterVoice: "Rachel"},
	}))
	require.NoError(t, err)
	got, err := c.GetVoiceOverrides(ctx, connect.NewRequest(&settv1.GetVoiceOverridesRequest{}))
	require.NoError(t, err)
	require.Len(t, got.Msg.GetOverrides(), 1)
	require.Equal(t, "Rachel", got.Msg.GetOverrides()[0].GetAdapterVoice())
	// Empty adapter deletes.
	_, err = c.SetVoiceOverride(ctx, connect.NewRequest(&settv1.SetVoiceOverrideRequest{
		Override: &settv1.VoiceOverride{CanonicalVoice: "voice.feminine.warm", TierProvider: "byok:elevenlabs", AdapterVoice: ""},
	}))
	require.NoError(t, err)
	got, err = c.GetVoiceOverrides(ctx, connect.NewRequest(&settv1.GetVoiceOverridesRequest{}))
	require.NoError(t, err)
	require.Len(t, got.Msg.GetOverrides(), 0)
}
