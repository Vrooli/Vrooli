package recipients_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/recipients"
	recipientsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/recipients/recipients_v1connect"

	recipientsH "notification-hub/handlers/recipients"
	"notification-hub/internal/hub"
	"notification-hub/internal/modules"
)

func TestConfiguredRecipientMetadataReadsNeedNoLoginAndDoNotCreateRows(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	service.SetDefaultRecipient("alice")
	device, err := service.UpsertDevice(ctx, "alice", hub.Device{Name: "Phone", MachineID: "local"})
	require.NoError(t, err)
	window, err := service.SetQuietWindow(ctx, "alice", hub.QuietWindow{Weekday: 2, Start: "21:00", End: "22:00", Timezone: "UTC"})
	require.NoError(t, err)
	_, err = service.SetEscalationChain(ctx, "alice", []string{"push"})
	require.NoError(t, err)

	router := mux.NewRouter()
	recipientsH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := recipientsconnect.NewRecipientsServiceClient(server.Client(), server.URL)

	recipient, err := client.GetRecipient(ctx, connect.NewRequest(&v1.GetRecipientRequest{}))
	require.NoError(t, err)
	require.Equal(t, "alice", recipient.Msg.GetSubject())

	devices, err := client.ListDevices(ctx, connect.NewRequest(&v1.ListDevicesRequest{}))
	require.NoError(t, err)
	require.Len(t, devices.Msg.GetDevices(), 1)
	require.Equal(t, device.ID, devices.Msg.GetDevices()[0].GetId())

	windows, err := client.ListQuietWindows(ctx, connect.NewRequest(&v1.ListQuietWindowsRequest{}))
	require.NoError(t, err)
	require.Len(t, windows.Msg.GetWindows(), 1)
	require.Equal(t, window.ID, windows.Msg.GetWindows()[0].GetId())

	chain, err := client.GetEscalationChain(ctx, connect.NewRequest(&v1.GetEscalationChainRequest{}))
	require.NoError(t, err)
	require.Equal(t, recipient.Msg.GetId(), chain.Msg.GetRecipientId())
	require.Len(t, chain.Msg.GetSteps(), 1)
	require.Equal(t, "push", chain.Msg.GetSteps()[0].GetChannel())

	// A configured but unregistered subject is reported as absent. The read
	// path must not create the durable recipient record as a side effect.
	service.SetDefaultRecipient("nobody")
	_, err = client.GetRecipient(ctx, connect.NewRequest(&v1.GetRecipientRequest{}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())
	chain, err = client.GetEscalationChain(ctx, connect.NewRequest(&v1.GetEscalationChainRequest{}))
	require.NoError(t, err)
	require.Empty(t, chain.Msg.GetRecipientId())
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM recipients WHERE subject='nobody'`).Scan(&count))
	require.Zero(t, count)

	_, err = client.UpsertDevice(ctx, connect.NewRequest(&v1.UpsertDeviceRequest{Name: "No auth"}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
}
