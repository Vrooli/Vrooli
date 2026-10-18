package hub_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	hub "notification-hub/internal/hub"
)

func TestChannelsStatusUsesConfiguredOrLocalMachineCacheKeys(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()

	for _, requestedMachine := range []string{"unknown-one", "unknown-two"} {
		status, err := service.ChannelsStatus(ctx, "alice", requestedMachine)
		require.NoError(t, err)
		require.Len(t, status, 1)
		require.Equal(t, "local", status[0].MachineID)
	}
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM machine_channel_status`).Scan(&count))
	require.Equal(t, 1, count, "unrecognized request IDs must not grow the status cache")

	device, err := service.UpsertDevice(ctx, "alice", hub.Device{Name: "Remote", MachineID: "configured-machine"})
	require.NoError(t, err)
	_, err = service.UpsertChannelAddress(ctx, "alice", hub.ChannelAddress{DeviceID: device.ID, Channel: "email", Address: "configured"})
	require.NoError(t, err)
	status, err := service.ChannelsStatus(ctx, "alice", "another-unrecognized-id")
	require.NoError(t, err)
	require.Len(t, status, 1)
	require.Equal(t, "configured-machine", status[0].MachineID)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM machine_channel_status`).Scan(&count))
	require.Equal(t, 2, count, "configured target IDs remain the only non-local cache keys")
}
