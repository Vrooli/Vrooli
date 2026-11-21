//go:build windows

package securestore

func absentBackendFix() string {
	return "ensure the Windows Credential Manager service is present; it ships with every supported Windows release"
}

// hostBoundFix has nothing to add on Windows for the same reason it has nothing
// to add on macOS: systemd-creds is a Linux facility.
func hostBoundFix() string { return "" }

// PendingGroupGrant has no meaning here: the native wrap on this platform is a
// per-user facility, not a device a group grants access to.
func PendingGroupGrant() string { return "" }

func nativeStorageStrength() (string, string) { return "", "" }

// GrantableTPMDevice has no answer here: systemd-creds and the tss device
// group are Linux facilities.
func GrantableTPMDevice() (device, group string, found bool) { return "", "", false }
