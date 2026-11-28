//go:build windows

package hostinventory

import platformgo "github.com/vrooli/platform-go"

func currentElevation() ElevationCapability {
	elevated, err := platformgo.CurrentProcessElevated()
	if err != nil {
		return ElevationCapability{Mechanism: "windows-token-unknown"}
	}
	if elevated {
		return ElevationCapability{Elevated: true, CanElevate: true, Mechanism: "windows-token"}
	}
	return ElevationCapability{Mechanism: "windows-uac"}
}
