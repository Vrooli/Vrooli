//go:build darwin

package securestore

func absentBackendFix() string {
	return "build with cgo enabled so the macOS Keychain adapter is compiled in"
}

// hostBoundFix has nothing to add on macOS: the host-bound wrap is a
// systemd-creds facility, so it is never the reason a store stays locked here.
func hostBoundFix() string { return "" }

// PendingGroupGrant has no meaning here: the native wrap on this platform is a
// per-user facility, not a device a group grants access to.
func PendingGroupGrant() string { return "" }

func nativeStorageStrength() (string, string) { return "", "" }

// GrantableTPMDevice has no answer here: systemd-creds and the tss device
// group are Linux facilities.
func GrantableTPMDevice() (device, group string, found bool) { return "", "", false }
