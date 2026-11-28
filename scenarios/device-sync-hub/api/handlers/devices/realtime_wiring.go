package devices

import (
	internaldevices "device-sync-hub/internal/devices"
	internalrealtime "device-sync-hub/internal/realtime"
)

// hubPairingNotifier adapts the realtime hub to devices.PairingNotifier so the
// devices service can push an approve/reject banner to the owner's online
// devices when a new device joins via the fallback request path. Lives at the
// handler layer so the devices domain depends only on its narrow
// PairingNotifier interface, never on the realtime hub's concrete type.
type hubPairingNotifier struct {
	hub *internalrealtime.Hub
}

// Compile-time guarantee.
var _ internaldevices.PairingNotifier = hubPairingNotifier{}

func (n hubPairingNotifier) PairingRequested(ownerID string, device internaldevices.Device) {
	n.hub.EmitPairingRequested(ownerID, internalrealtime.PairingInfo{
		DeviceID: device.ID,
		Name:     device.Name,
		Kind:     device.Kind,
	})
}
