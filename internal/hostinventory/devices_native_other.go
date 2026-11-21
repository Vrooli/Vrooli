//go:build !darwin && !windows

package hostinventory

import "context"

func collectNativeDevices(_ context.Context, _ CommandRunner) NativeDeviceObservation {
	return unavailableNativeDevices("", IntegrityProbeUnsupported, errNativeDevicePlatform)
}
