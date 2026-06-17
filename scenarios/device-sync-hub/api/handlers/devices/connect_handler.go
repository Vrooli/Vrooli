// Package devices is the HTTP/Connect transport edge for the devices domain.
// It is intentionally thin: resolve the owner identity (for owner-gated RPCs),
// decode the request, call internal/devices.Service, translate the result and
// any typed error. All policy lives in the service; all persistence in the
// repository.
package devices

import (
	"context"
	"log"

	"device-sync-hub/internal/auth"
	internaldevices "device-sync-hub/internal/devices"

	"connectrpc.com/connect"

	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/devices"
)

// Deps wires the seams the Connect devices handler needs.
type Deps struct {
	Service internaldevices.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect handler for the devices service.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// ---- Owner-gated RPCs -------------------------------------------------------

func (h *connectHandler) ListDevices(ctx context.Context, _ *connect.Request[devicesv1.ListDevicesRequest]) (*connect.Response[devicesv1.ListDevicesResponse], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, internaldevices.ToConnectError(err)
	}
	list, err := h.deps.Service.List(ctx, owner.OwnerID)
	if err != nil {
		h.deps.Logger.Printf("devices.ListDevices: %v", err)
		return nil, internaldevices.ToConnectError(err)
	}
	resp := &devicesv1.ListDevicesResponse{Devices: make([]*devicesv1.Device, 0, len(list))}
	for _, d := range list {
		resp.Devices = append(resp.Devices, deviceToProto(d))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetDevice(ctx context.Context, req *connect.Request[devicesv1.GetDeviceRequest]) (*connect.Response[devicesv1.GetDeviceResponse], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, internaldevices.ToConnectError(err)
	}
	d, err := h.deps.Service.Get(ctx, owner.OwnerID, req.Msg.Id)
	if err != nil {
		return nil, internaldevices.ToConnectError(err)
	}
	return connect.NewResponse(&devicesv1.GetDeviceResponse{Device: deviceToProto(d)}), nil
}

func (h *connectHandler) IssuePairingCode(ctx context.Context, req *connect.Request[devicesv1.IssuePairingCodeRequest]) (*connect.Response[devicesv1.IssuePairingCodeResponse], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, internaldevices.ToConnectError(err)
	}
	pc, err := h.deps.Service.IssuePairingCode(ctx, owner.OwnerID, req.Msg.DeviceName)
	if err != nil {
		h.deps.Logger.Printf("devices.IssuePairingCode: %v", err)
		return nil, internaldevices.ToConnectError(err)
	}
	return connect.NewResponse(&devicesv1.IssuePairingCodeResponse{PairingCode: pairingCodeToProto(pc)}), nil
}

func (h *connectHandler) ApprovePairing(ctx context.Context, req *connect.Request[devicesv1.ApprovePairingRequest]) (*connect.Response[devicesv1.ApprovePairingResponse], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, internaldevices.ToConnectError(err)
	}
	d, err := h.deps.Service.Approve(ctx, owner.OwnerID, req.Msg.DeviceId)
	if err != nil {
		return nil, internaldevices.ToConnectError(err)
	}
	return connect.NewResponse(&devicesv1.ApprovePairingResponse{Device: deviceToProto(d)}), nil
}

func (h *connectHandler) RenameDevice(ctx context.Context, req *connect.Request[devicesv1.RenameDeviceRequest]) (*connect.Response[devicesv1.RenameDeviceResponse], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, internaldevices.ToConnectError(err)
	}
	d, err := h.deps.Service.Rename(ctx, owner.OwnerID, req.Msg.DeviceId, req.Msg.Name)
	if err != nil {
		return nil, internaldevices.ToConnectError(err)
	}
	return connect.NewResponse(&devicesv1.RenameDeviceResponse{Device: deviceToProto(d)}), nil
}

func (h *connectHandler) RevokeDevice(ctx context.Context, req *connect.Request[devicesv1.RevokeDeviceRequest]) (*connect.Response[devicesv1.RevokeDeviceResponse], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, internaldevices.ToConnectError(err)
	}
	d, err := h.deps.Service.Revoke(ctx, owner.OwnerID, req.Msg.DeviceId)
	if err != nil {
		h.deps.Logger.Printf("devices.RevokeDevice: %v", err)
		return nil, internaldevices.ToConnectError(err)
	}
	return connect.NewResponse(&devicesv1.RevokeDeviceResponse{Device: deviceToProto(d)}), nil
}

// ---- Open RPCs (called by not-yet-paired devices) ---------------------------

func (h *connectHandler) RedeemPairingCode(ctx context.Context, req *connect.Request[devicesv1.RedeemPairingCodeRequest]) (*connect.Response[devicesv1.RedeemPairingCodeResponse], error) {
	issued, err := h.deps.Service.RedeemPairingCode(ctx, req.Msg.Code, profileFromProto(req.Msg.Profile))
	if err != nil {
		h.deps.Logger.Printf("devices.RedeemPairingCode: %v", err)
		return nil, internaldevices.ToConnectError(err)
	}
	return connect.NewResponse(&devicesv1.RedeemPairingCodeResponse{
		Device:      deviceToProto(issued.Device),
		DeviceToken: issued.Token,
	}), nil
}

func (h *connectHandler) RequestPairing(ctx context.Context, req *connect.Request[devicesv1.RequestPairingRequest]) (*connect.Response[devicesv1.RequestPairingResponse], error) {
	issued, err := h.deps.Service.RequestPairing(ctx, profileFromProto(req.Msg.Profile))
	if err != nil {
		h.deps.Logger.Printf("devices.RequestPairing: %v", err)
		return nil, internaldevices.ToConnectError(err)
	}
	return connect.NewResponse(&devicesv1.RequestPairingResponse{
		Device:      deviceToProto(issued.Device),
		DeviceToken: issued.Token,
	}), nil
}
