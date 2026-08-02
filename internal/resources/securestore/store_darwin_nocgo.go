//go:build darwin && !cgo

package securestore

// nativeDefault reports an absent backend on a darwin build without cgo. The
// Keychain is reachable only through the Security framework, and the
// `security` command's -w flag would place a credential value in a process
// argument, which this package prohibits. Reporting absence keeps such a build
// honest and still startable: resources that need credentials go unhealthy,
// the control plane does not.
func nativeDefault() Store {
	return Absent("macOS Keychain requires a cgo-enabled build; rebuild with CGO_ENABLED=1")
}
