package appdelivery_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/appdelivery"
	appconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/appdelivery/appdelivery_v1connect"

	appDeliveryH "notification-hub/handlers/appdelivery"
	"notification-hub/internal/hub"
	"notification-hub/internal/modules"
)

func newClient(t *testing.T) (appconnect.AppDeliveryServiceClient, *hub.Service) {
	t.Helper()
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	service := hub.New(database.NewFromPrimary(primary), nil, nil)
	service.SetWebPushPublicKey("hub-public-key")
	service.SetAppEligibility(func(_ context.Context, scenario string) (string, error) {
		return "https://" + scenario + ".example.test", nil
	})
	router := mux.NewRouter()
	appDeliveryH.ModuleWithVerifier(service, nil).Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return appconnect.NewAppDeliveryServiceClient(server.Client(), server.URL), service
}

func request[T any](msg *T, scenario string) *connect.Request[T] {
	req := connect.NewRequest(msg)
	req.Header().Set("X-Vrooli-Identity-Subject", "alice")
	if scenario != "" {
		req.Header().Set(appDeliveryH.ScenarioHeader, scenario)
	}
	return req
}

func TestAppDeliveryServiceEnablesConfirmsListsAndDisablesAnApp(t *testing.T) {
	ctx := context.Background()
	client, service := newClient(t)
	require.NoError(t, service.RegisterPushSubscription(ctx, "alice", hub.PushSubscription{
		Endpoint: "https://web.push.apple.com/chimesail", P256DH: "k", Auth: "a", Origin: "https://notification-hub.example.test",
		UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_7 like Mac OS X)",
	}))

	state, err := client.GetAppState(ctx, request(&v1.GetAppStateRequest{UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_7 like Mac OS X)"}, "web-console"))
	require.NoError(t, err)
	require.True(t, state.Msg.GetEligible())
	require.Equal(t, "hub-public-key", state.Msg.GetVapidPublicKey())
	require.Len(t, state.Msg.GetDevices(), 1)
	device := state.Msg.GetDevices()[0]
	require.True(t, device.GetBestMatch())
	require.Equal(t, "iPhone", device.GetLabel())

	enabled, err := client.EnableApp(ctx, request(&v1.EnableAppRequest{
		Endpoint: "https://web.push.apple.com/app", P256Dh: "k", Auth: "a", Origin: "https://web-console.example.test",
		Permission: "granted", PairWithGroup: device.GetGroup(),
	}, "web-console"))
	require.NoError(t, err)
	require.True(t, enabled.Msg.GetSubscription().GetHealthy())
	require.False(t, enabled.Msg.GetSubscription().GetStandalone())

	confirmed, err := client.ConfirmApp(ctx, request(&v1.ConfirmAppRequest{Endpoint: "https://web.push.apple.com/app", Permission: "granted"}, "web-console"))
	require.NoError(t, err)
	require.True(t, confirmed.Msg.GetRegistered())
	unknown, err := client.ConfirmApp(ctx, request(&v1.ConfirmAppRequest{Endpoint: "https://web.push.apple.com/other", Permission: "granted"}, "web-console"))
	require.NoError(t, err)
	require.False(t, unknown.Msg.GetRegistered(), "an unknown app is told to re-enable, not refused")

	devices, err := client.ListDeviceApps(ctx, request(&v1.ListDeviceAppsRequest{}, ""))
	require.NoError(t, err)
	require.Len(t, devices.Msg.GetDevices(), 1)
	require.Len(t, devices.Msg.GetDevices()[0].GetApps(), 2)

	_, err = client.DisableApp(ctx, request(&v1.DisableAppRequest{Endpoint: "https://web.push.apple.com/app"}, "web-console"))
	require.NoError(t, err)
	after, err := client.GetAppState(ctx, request(&v1.GetAppStateRequest{Endpoint: "https://web.push.apple.com/app"}, "web-console"))
	require.NoError(t, err)
	require.Nil(t, after.Msg.GetSubscription())
}

func TestAppDeliveryServiceRefusesMissingScenarioWrongOriginAndAnonymousCalls(t *testing.T) {
	ctx := context.Background()
	client, _ := newClient(t)

	_, err := client.GetAppState(ctx, request(&v1.GetAppStateRequest{}, ""))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "the scenario must come from the scenario web server")

	_, err = client.EnableApp(ctx, request(&v1.EnableAppRequest{
		Endpoint: "https://web.push.apple.com/app", P256Dh: "k", Auth: "a", Origin: "https://agent-manager.example.test", Standalone: true,
	}, "web-console"))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "another scenario's origin cannot register as web-console")

	_, err = client.RemoveDeviceApp(ctx, request(&v1.RemoveDeviceAppRequest{SubscriptionId: "missing"}, ""))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	anonymous := connect.NewRequest(&v1.ListDeviceAppsRequest{})
	_, err = client.ListDeviceApps(ctx, anonymous)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
