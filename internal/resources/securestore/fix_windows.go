//go:build windows

package securestore

func absentBackendFix() string {
	return "ensure the Windows Credential Manager service is present; it ships with every supported Windows release"
}

// hostBoundFix has nothing to add on Windows for the same reason it has nothing
// to add on macOS: systemd-creds is a Linux facility.
func hostBoundFix() string { return "" }

func nativeStorageStrength() (string, string) { return "", "" }
