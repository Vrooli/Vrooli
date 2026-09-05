//go:build !linux

package namespace

import "runtime"

// getKernelVersion has no syscall-backed implementation off Linux. There
// is no unprivileged-overlayfs kernel gate to satisfy on other platforms
// (Check short-circuits: `unshare` is absent so CanCreateUserNamespace is
// false, and /proc/self/uid_map is absent so InUserNamespace is false), so
// this returns a GOOS descriptor purely for diagnostics. It never parses
// as a numeric version, which correctly makes IsKernelAtLeast report false.
func getKernelVersion() string {
	return "non-linux (" + runtime.GOOS + ")"
}
