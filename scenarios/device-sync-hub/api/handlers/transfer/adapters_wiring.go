package transfer

import (
	"context"
	"errors"

	"device-sync-hub/internal/devices"
	"device-sync-hub/internal/realtime"
	internaltransfer "device-sync-hub/internal/transfer"
)

// deviceTrustChecker adapts the devices domain to transfer.TrustChecker. It
// lives at the handler layer so the transfer domain never imports devices'
// concrete service — it depends only on the narrow TrustChecker interface.
type deviceTrustChecker struct {
	svc devices.Service
}

// Compile-time guarantee.
var _ internaltransfer.TrustChecker = deviceTrustChecker{}

func (c deviceTrustChecker) IsTrustedDevice(ctx context.Context, ownerID, deviceID string) (bool, error) {
	d, err := c.svc.Get(ctx, ownerID, deviceID)
	if err != nil {
		var notFound devices.ErrDeviceNotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, err
	}
	return d.TrustState == devices.TrustTrusted, nil
}

// hubNotifier adapts the realtime hub to transfer.Notifier, translating a
// transfer.Item into the id-only fan-out the hub speaks. A nil hub yields the
// service's own no-op default (handled in transfer.NewService).
type hubNotifier struct {
	hub *realtime.Hub
}

// Compile-time guarantee.
var _ internaltransfer.Notifier = hubNotifier{}

func (n hubNotifier) ItemArrived(_ context.Context, item internaltransfer.Item) {
	n.hub.EmitItemArrived(item.OwnerID, item.ID, item.TargetDeviceID, item.OriginDeviceID)
}

func (n hubNotifier) ItemDeleted(_ context.Context, item internaltransfer.Item) {
	n.hub.EmitItemDeleted(item.OwnerID, item.ID, item.TargetDeviceID, item.OriginDeviceID)
}
