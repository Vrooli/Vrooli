package settings_test

import (
	"context"
	"log"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"flow-verifier/handlers/settings"
	internalsettings "flow-verifier/internal/settings"

	settingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/settings"
	settingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/settings/settings_v1connect"
)

func newSettingsClient(t *testing.T, svc *internalsettings.Service) settingsconnect.SettingsServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := settingsconnect.NewSettingsServiceHandler(settings.NewConnectHandler(settings.Deps{Service: svc, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return settingsconnect.NewSettingsServiceClient(server.Client(), server.URL)
}

func newServiceWithFake(t *testing.T) (*internalsettings.Service, *fakeRepo) {
	t.Helper()
	repo := &fakeRepo{stored: internalsettings.DefaultSettings()}
	return internalsettings.NewService(repo), repo
}

func TestConnectGetReturnsDefaults(t *testing.T) {
	svc, _ := newServiceWithFake(t)
	client := newSettingsClient(t, svc)

	resp, err := client.GetSettings(context.Background(), connect.NewRequest(&settingsv1.GetSettingsRequest{}))
	require.NoError(t, err)
	require.Equal(t, "local", resp.Msg.Settings.PrincipalId)
	require.Equal(t, settingsv1.Theme_THEME_SYSTEM, resp.Msg.Settings.Theme)
}

func TestConnectUpdateAppliesMask(t *testing.T) {
	svc, repo := newServiceWithFake(t)
	client := newSettingsClient(t, svc)

	resp, err := client.UpdateSettings(context.Background(), connect.NewRequest(&settingsv1.UpdateSettingsRequest{
		Settings: &settingsv1.Settings{
			Theme:     settingsv1.Theme_THEME_DARK,
			FontScale: settingsv1.FontScale_FONT_SCALE_LG, // ignored — not in the mask
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"theme"}},
	}))
	require.NoError(t, err)
	require.Equal(t, settingsv1.Theme_THEME_DARK, resp.Msg.Settings.Theme)
	require.Equal(t, internalsettings.Theme("dark"), repo.stored.Theme)
	// FontScale was not in the mask; the stored value must still be the default.
	require.Equal(t, internalsettings.FontScaleMd, repo.stored.FontScale)
}

func TestConnectUpdateInvalidEnumRejected(t *testing.T) {
	svc, _ := newServiceWithFake(t)
	client := newSettingsClient(t, svc)

	_, err := client.UpdateSettings(context.Background(), connect.NewRequest(&settingsv1.UpdateSettingsRequest{
		Settings:   &settingsv1.Settings{Theme: settingsv1.Theme_THEME_UNSPECIFIED},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"theme"}},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// fakeRepo is a one-row in-memory settings repository.
type fakeRepo struct {
	stored internalsettings.Settings
}

func (f *fakeRepo) Get(_ context.Context, _ string) (internalsettings.Settings, error) {
	return f.stored, nil
}

func (f *fakeRepo) Upsert(_ context.Context, s internalsettings.Settings) (internalsettings.Settings, error) {
	f.stored = s
	return s, nil
}

// silence unused-import linter when -ldflags trimming kicks in
var _ = log.Default
