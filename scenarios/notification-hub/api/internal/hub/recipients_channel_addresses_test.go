package hub_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"

	hub "notification-hub/internal/hub"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestChannelAddressCreateRetryConflictAndOwnerScope(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	device, err := service.UpsertDevice(ctx, "alice", hub.Device{Name: "Phone", MachineID: "local"})
	require.NoError(t, err)
	foreign, err := service.UpsertDevice(ctx, "bob", hub.Device{Name: "Phone", MachineID: "local"})
	require.NoError(t, err)

	original := hub.ChannelAddress{ID: "stable-email-1", DeviceID: device.ID, Channel: "email", Address: "first@example.test", ApprovedLabels: []string{"public", "sensitive"}}
	created, err := service.CreateChannelAddress(ctx, "alice", original)
	require.NoError(t, err)
	require.Equal(t, original.ID, created.ID)

	// Repeating the same stable request after an uncertain response is a readback.
	retried, err := service.CreateChannelAddress(ctx, "alice", original)
	require.NoError(t, err)
	require.Equal(t, created, retried)

	// Reusing an ID for a different payload and a new ID for the occupied
	// natural key both conflict; neither can mutate the prior row.
	changedRetry := original
	changedRetry.Address = "attacker@example.test"
	_, err = service.CreateChannelAddress(ctx, "alice", changedRetry)
	require.ErrorIs(t, err, hub.ErrChannelAddressConflict)
	duplicate := original
	duplicate.ID = "stable-email-2"
	duplicate.Address = "second@example.test"
	_, err = service.CreateChannelAddress(ctx, "alice", duplicate)
	require.ErrorIs(t, err, hub.ErrChannelAddressConflict)
	read, err := service.ListChannelAddresses(ctx, "alice")
	require.NoError(t, err)
	require.Equal(t, []hub.ChannelAddress{created}, read)

	// ID-only updates cannot reach a row owned by another recipient.
	_, err = service.UpdateChannelAddress(ctx, "bob", hub.ChannelAddress{ID: created.ID, Address: "foreign@example.test", ApprovedLabels: []string{"critical"}})
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = service.CreateChannelAddress(ctx, "alice", hub.ChannelAddress{ID: "foreign-device", DeviceID: foreign.ID, Channel: "email", Address: "secret@example.test"})
	require.ErrorIs(t, err, sql.ErrNoRows)
	_ = foreign
}

func TestChannelAddressUpdateRequiresExactOwnerScopedID(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	device, err := service.UpsertDevice(ctx, "alice", hub.Device{Name: "Phone", MachineID: "local"})
	require.NoError(t, err)
	other, err := service.UpsertDevice(ctx, "bob", hub.Device{Name: "Phone", MachineID: "local"})
	require.NoError(t, err)
	created, err := service.CreateChannelAddress(ctx, "alice", hub.ChannelAddress{ID: "email-owner", DeviceID: device.ID, Channel: "email", Address: "owner@example.test", ApprovedLabels: []string{"public"}})
	require.NoError(t, err)

	_, err = service.UpdateChannelAddress(ctx, "alice", hub.ChannelAddress{ID: "unknown", Address: "new@example.test"})
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = service.UpdateChannelAddress(ctx, "bob", hub.ChannelAddress{ID: created.ID, Address: "foreign@example.test"})
	require.ErrorIs(t, err, sql.ErrNoRows)
	updated, err := service.UpdateChannelAddress(ctx, "alice", hub.ChannelAddress{ID: created.ID, Address: "updated@example.test", ApprovedLabels: []string{"private"}})
	require.NoError(t, err)
	require.Equal(t, created.ID, updated.ID)
	require.Equal(t, device.ID, updated.DeviceID)
	require.Equal(t, "email", updated.Channel)
	require.Equal(t, "updated@example.test", updated.Address)
	require.Equal(t, []string{"private"}, updated.ApprovedLabels)

	addresses, err := service.ListChannelAddresses(ctx, "bob")
	require.NoError(t, err)
	require.Empty(t, addresses)
	_ = other
}

func TestChannelAddressConcurrentDifferentIDsNeverOverwrite(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	device, err := service.UpsertDevice(ctx, "alice", hub.Device{Name: "Phone", MachineID: "local"})
	require.NoError(t, err)
	start := make(chan struct{})
	results := make(chan struct {
		address hub.ChannelAddress
		err     error
	}, 2)
	var workers sync.WaitGroup
	for _, address := range []hub.ChannelAddress{
		{ID: uuid.NewString(), DeviceID: device.ID, Channel: "email", Address: "one@example.test", ApprovedLabels: []string{"public"}},
		{ID: uuid.NewString(), DeviceID: device.ID, Channel: "email", Address: "two@example.test", ApprovedLabels: []string{"critical"}},
	} {
		address := address
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			created, createErr := service.CreateChannelAddress(ctx, "alice", address)
			results <- struct {
				address hub.ChannelAddress
				err     error
			}{created, createErr}
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	var winner *hub.ChannelAddress
	conflicts := 0
	for result := range results {
		if result.err == nil {
			copy := result.address
			winner = &copy
		} else if errors.Is(result.err, hub.ErrChannelAddressConflict) {
			conflicts++
		} else {
			require.NoError(t, result.err)
		}
	}
	require.NotNil(t, winner)
	require.Equal(t, 1, conflicts)
	addresses, err := service.ListChannelAddresses(ctx, "alice")
	require.NoError(t, err)
	require.Equal(t, []hub.ChannelAddress{*winner}, addresses)
}
