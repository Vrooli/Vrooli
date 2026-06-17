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
	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/devices/devices_v1connect"
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
	h   devicesconnect.DevicesServiceHandler
	svc internaldevices.Service
}

func ownerCtx(ownerID string) context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{OwnerID: ownerID})
}

func TestSetupOwnerDeviceRequiresOwner(t *testing.T) {
	t.Parallel()
	hh := newHandler(t)

	_, err := hh.h.SetupOwnerDevice(context.Background(), connect.NewRequest(&devicesv1.SetupOwnerDeviceRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "bootstrap still needs an authenticated owner")
}

// [REQ:REQ-P0-003] Owner bootstrap admits the caller to the trust group as TRUSTED.
func TestSetupOwnerDeviceTrustsCaller(t *testing.T) {
	t.Parallel()
	hh := newHandler(t)

	resp, err := hh.h.SetupOwnerDevice(ownerCtx("owner-1"), connect.NewRequest(&devicesv1.SetupOwnerDeviceRequest{
		Profile: &devicesv1.DeviceProfile{DeviceName: "Workstation", Kind: "laptop"},
	}))
	require.NoError(t, err)
	assert.Equal(t, devicesv1.TrustState_TRUST_STATE_TRUSTED, resp.Msg.Device.TrustState)
	assert.Equal(t, "Workstation", resp.Msg.Device.Name)
	assert.NotEmpty(t, resp.Msg.DeviceToken, "one-time token returned")
}

// [REQ:REQ-P0-005] Single-owner hub rejects a second identity with PermissionDenied.
func TestSetupOwnerDeviceRejectsSecondIdentity(t *testing.T) {
	t.Parallel()
	hh := newHandler(t)

	_, err := hh.h.SetupOwnerDevice(ownerCtx("owner-1"), connect.NewRequest(&devicesv1.SetupOwnerDeviceRequest{}))
	require.NoError(t, err)

	_, err = hh.h.SetupOwnerDevice(ownerCtx("owner-2"), connect.NewRequest(&devicesv1.SetupOwnerDeviceRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "single-owner hub rejects a second identity")
}

func TestListDevicesRequiresOwner(t *testing.T) {
	t.Parallel()
	hh := newHandler(t)

	_, err := hh.h.ListDevices(context.Background(), connect.NewRequest(&devicesv1.ListDevicesRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "owner-gated RPC rejects an anonymous caller")
}

func TestSetupIssueRedeemList(t *testing.T) {
	t.Parallel()
	hh := newHandler(t)
	ctx := ownerCtx("owner-1")

	_, err := hh.h.SetupOwnerDevice(ctx, connect.NewRequest(&devicesv1.SetupOwnerDeviceRequest{
		Profile: &devicesv1.DeviceProfile{DeviceName: "Workstation", Kind: "laptop"},
	}))
	require.NoError(t, err)

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
	require.Len(t, list.Msg.Devices, 2, "owner device + redeemed phone")
}

func TestRedeemInvalidCodeMapsToInvalidArgument(t *testing.T) {
	t.Parallel()
	hh := newHandler(t)
	_, err := hh.h.RedeemPairingCode(context.Background(), connect.NewRequest(&devicesv1.RedeemPairingCodeRequest{Code: "BAD00-BAD00"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
