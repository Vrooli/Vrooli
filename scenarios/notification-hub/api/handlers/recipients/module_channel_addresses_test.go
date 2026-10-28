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
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/recipients/recipients_v1connect"

	recipientsH "notification-hub/handlers/recipients"
	"notification-hub/internal/hub"
	"notification-hub/internal/modules"
)

func ownerRequest[T any](message *T, subject string) *connect.Request[T] {
	req := connect.NewRequest(message)
	req.Header().Set("X-Vrooli-Identity-Subject", subject)
	return req
}

func TestChannelAddressRPCsUseStableCreateAndExactOwnerIDUpdate(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	service.SetDefaultRecipient("alice")
	device, err := service.UpsertDevice(ctx, "alice", hub.Device{Name: "Phone", MachineID: "local"})
	require.NoError(t, err)
	foreignDevice, err := service.UpsertDevice(ctx, "bob", hub.Device{Name: "Phone", MachineID: "local"})
	require.NoError(t, err)

	router := mux.NewRouter()
	recipientsH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := connectv1.NewRecipientsServiceClient(server.Client(), server.URL)
	_, err = client.ListChannelAddresses(ctx, connect.NewRequest(&v1.ListChannelAddressesRequest{}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
	create := &v1.CreateChannelAddressRequest{Id: "stable-email", DeviceId: device.ID, Channel: "email", Address: "first@example.test", ApprovedLabels: []string{"public"}}
	created, err := client.CreateChannelAddress(ctx, ownerRequest(create, "alice"))
	require.NoError(t, err)
	retry, err := client.CreateChannelAddress(ctx, ownerRequest(create, "alice"))
	require.NoError(t, err)
	require.Equal(t, created.Msg, retry.Msg)

	conflict := &v1.CreateChannelAddressRequest{Id: "other-email", DeviceId: device.ID, Channel: "email", Address: "overwrite@example.test", ApprovedLabels: []string{"critical"}}
	_, err = client.CreateChannelAddress(ctx, ownerRequest(conflict, "alice"))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeAlreadyExists, connectErr.Code())

	updated, err := client.UpdateChannelAddress(ctx, ownerRequest(&v1.UpdateChannelAddressRequest{Id: created.Msg.GetId(), Address: "updated@example.test", ApprovedLabels: []string{"private"}}, "alice"))
	require.NoError(t, err)
	require.Equal(t, created.Msg.GetId(), updated.Msg.GetId())
	require.Equal(t, "updated@example.test", updated.Msg.GetAddress())
	require.Equal(t, []string{"private"}, updated.Msg.GetApprovedLabels())

	_, err = client.UpdateChannelAddress(ctx, ownerRequest(&v1.UpdateChannelAddressRequest{Id: created.Msg.GetId(), Address: "foreign@example.test"}, "bob"))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())
	_, err = client.CreateChannelAddress(ctx, ownerRequest(&v1.CreateChannelAddressRequest{Id: "foreign-device", DeviceId: foreignDevice.ID, Channel: "email", Address: "hidden@example.test"}, "alice"))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())
	_, err = client.UpdateChannelAddress(ctx, ownerRequest(&v1.UpdateChannelAddressRequest{Id: "missing", Address: "nobody@example.test"}, "alice"))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())

	listed, err := client.ListChannelAddresses(ctx, ownerRequest(&v1.ListChannelAddressesRequest{}, "alice"))
	require.NoError(t, err)
	require.Len(t, listed.Msg.GetAddresses(), 1)
	require.Equal(t, "updated@example.test", listed.Msg.GetAddresses()[0].GetAddress())
	listed, err = client.ListChannelAddresses(ctx, ownerRequest(&v1.ListChannelAddressesRequest{}, "bob"))
	require.NoError(t, err)
	require.Empty(t, listed.Msg.GetAddresses())
}

func TestLegacyUpsertChannelAddressCannotOverwriteNaturalKey(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	service := hub.New(database.NewFromPrimary(primary), nil, nil)
	device, err := service.UpsertDevice(ctx, "alice", hub.Device{Name: "Phone", MachineID: "local"})
	require.NoError(t, err)
	router := mux.NewRouter()
	recipientsH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := connectv1.NewRecipientsServiceClient(server.Client(), server.URL)
	request := connect.NewRequest(&v1.UpsertChannelAddressRequest{Id: "legacy-first", DeviceId: device.ID, Channel: "email", Address: "first@example.test"})
	request.Header().Set("X-Vrooli-Identity-Subject", "alice")
	_, err = client.UpsertChannelAddress(ctx, request)
	require.NoError(t, err)
	request = connect.NewRequest(&v1.UpsertChannelAddressRequest{Id: "legacy-second", DeviceId: device.ID, Channel: "email", Address: "overwrite@example.test"})
	request.Header().Set("X-Vrooli-Identity-Subject", "alice")
	_, err = client.UpsertChannelAddress(ctx, request)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeAlreadyExists, connectErr.Code())
	request = connect.NewRequest(&v1.UpsertChannelAddressRequest{Id: "legacy-first", DeviceId: device.ID, Channel: "email", Address: "exact-update@example.test", ApprovedLabels: []string{"private"}})
	request.Header().Set("X-Vrooli-Identity-Subject", "alice")
	_, err = client.UpsertChannelAddress(ctx, request)
	require.NoError(t, err)
	listed, err := client.ListChannelAddresses(ctx, ownerRequest(&v1.ListChannelAddressesRequest{}, "alice"))
	require.NoError(t, err)
	require.Len(t, listed.Msg.GetAddresses(), 1)
	require.Equal(t, "exact-update@example.test", listed.Msg.GetAddresses()[0].GetAddress())
}
