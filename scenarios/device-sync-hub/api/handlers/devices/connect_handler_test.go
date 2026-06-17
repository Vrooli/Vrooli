package devices_test

import (
	"context"
	"testing"

	handlerdevices "device-sync-hub/handlers/devices"
	"device-sync-hub/internal/auth"
	internaldevices "device-sync-hub/internal/devices"
	"device-sync-hub/internal/devices/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/devices"
)

func newHandler(t *testing.T) *handlerHarness {
	t.Helper()
	repo := mocks.NewFakeRepository()
	svc := internaldevices.NewService(internaldevices.Config{
		Repo:    repo,
		Secrets: &mocks.FakeSecrets{},
		Auth:    &mocks.FakeAuth{},
	})
	return &handlerHarness{h: handlerdevices.NewConnectHandler(handlerdevices.Deps{Service: svc}), svc: svc}
}

type handlerHarness struct {
	h interface {
		ListDevices(context.Context, *connect.Request[devicesv1.ListDevicesRequest]) (*connect.Response[devicesv1.ListDevicesResponse], error)
		RedeemPairingCode(context.Context, *connect.Request[devicesv1.RedeemPairingCodeRequest]) (*connect.Response[devicesv1.RedeemPairingCodeResponse], error)
		IssuePairingCode(context.Context, *connect.Request[devicesv1.IssuePairingCodeRequest]) (*connect.Response[devicesv1.IssuePairingCodeResponse], error)
	}
	svc internaldevices.Service
}

func ownerCtx(ownerID string) context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{OwnerID: ownerID})
}

func TestListDevicesRequiresOwner(t *testing.T) {
	t.Parallel()
	hh := newHandler(t)

	_, err := hh.h.ListDevices(context.Background(), connect.NewRequest(&devicesv1.ListDevicesRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "owner-gated RPC rejects an anonymous caller")
}

func TestIssueThenListWithOwner(t *testing.T) {
	t.Parallel()
	hh := newHandler(t)
	ctx := ownerCtx("owner-1")

	issue, err := hh.h.IssuePairingCode(ctx, connect.NewRequest(&devicesv1.IssuePairingCodeRequest{DeviceName: "Phone"}))
	require.NoError(t, err)
	code := issue.Msg.PairingCode.Code
	require.NotEmpty(t, code)

	// Redeem is open — no owner identity in context.
	redeem, err := hh.h.RedeemPairingCode(context.Background(), connect.NewRequest(&devicesv1.RedeemPairingCodeRequest{
		Code:    code,
		Profile: &devicesv1.DeviceProfile{DeviceName: "Phone", Kind: "phone"},
	}))
	require.NoError(t, err)
	assert.Equal(t, devicesv1.TrustState_TRUST_STATE_TRUSTED, redeem.Msg.Device.TrustState)
	assert.NotEmpty(t, redeem.Msg.DeviceToken)

	list, err := hh.h.ListDevices(ctx, connect.NewRequest(&devicesv1.ListDevicesRequest{}))
	require.NoError(t, err)
	require.Len(t, list.Msg.Devices, 1)
	assert.Equal(t, "Phone", list.Msg.Devices[0].Name)
}

func TestRedeemInvalidCodeMapsToInvalidArgument(t *testing.T) {
	t.Parallel()
	hh := newHandler(t)
	_, err := hh.h.RedeemPairingCode(context.Background(), connect.NewRequest(&devicesv1.RedeemPairingCodeRequest{Code: "BAD00-BAD00"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
