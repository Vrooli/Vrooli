//go:build windows

package hostinventory

import "golang.org/x/sys/windows"

func currentElevation() ElevationCapability {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return ElevationCapability{Mechanism: "windows-token-unknown"}
	}
	defer token.Close()
	if token.IsElevated() {
		return ElevationCapability{Elevated: true, CanElevate: true, Mechanism: "windows-token"}
	}
	return ElevationCapability{Mechanism: "windows-uac"}
}
