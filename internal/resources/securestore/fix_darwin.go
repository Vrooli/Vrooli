//go:build darwin

package securestore

func absentBackendFix() string {
	return "build with cgo enabled so the macOS Keychain adapter is compiled in"
}

// hostBoundFix has nothing to add on macOS: the host-bound wrap is a
// systemd-creds facility, so it is never the reason a store stays locked here.
func hostBoundFix() string { return "" }

func nativeStorageStrength() (string, string) { return "", "" }
