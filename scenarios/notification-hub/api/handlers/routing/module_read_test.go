package routing_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/routing/routing_v1connect"

	routingH "notification-hub/handlers/routing"
	"notification-hub/internal/hub"
	"notification-hub/internal/modules"
)

func TestConfiguredRecipientCanReadChannelStatusWithoutIdentity(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	serviceDB := database.NewFromPrimary(primary)
	service := hub.New(serviceDB, nil, nil)
	service.SetDefaultRecipient("alice")

	router := mux.NewRouter()
	routingH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := routingconnect.NewRoutingServiceClient(server.Client(), server.URL)

	for _, machineID := range []string{"unconfigured-one", "unconfigured-two"} {
		response, err := client.ChannelsStatus(ctx, connect.NewRequest(&v1.ChannelsStatusRequest{MachineId: machineID}))
		require.NoError(t, err, "ordinary configured-recipient status reads must not require an owner login")
		require.Len(t, response.Msg.GetChannels(), 1)
		require.Equal(t, "local", response.Msg.GetChannels()[0].GetMachineId())
	}
	var count int
	require.NoError(t, serviceDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM machine_channel_status`).Scan(&count))
	require.Equal(t, 1, count, "unknown machine IDs must not create unbounded status-cache rows")

	device, err := service.UpsertDevice(ctx, "alice", hub.Device{Name: "Remote", MachineID: "configured-machine"})
	require.NoError(t, err)
	_, err = service.UpsertChannelAddress(ctx, "alice", hub.ChannelAddress{DeviceID: device.ID, Channel: "email", Address: "configured"})
	require.NoError(t, err)
	response, err := client.ChannelsStatus(ctx, connect.NewRequest(&v1.ChannelsStatusRequest{MachineId: "another-unconfigured-id"}))
	require.NoError(t, err)
	require.Len(t, response.Msg.GetChannels(), 1)
	require.Equal(t, "configured-machine", response.Msg.GetChannels()[0].GetMachineId(), "configured target identity remains authoritative")
	require.NoError(t, serviceDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM machine_channel_status`).Scan(&count))
	require.Equal(t, 2, count, "unknown request IDs must not be persisted when configured targets exist")
}
