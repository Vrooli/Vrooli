//go:build !linux

package fsmount

import "runtime"

// sysMountOverlay has no implementation off Linux: the kernel-overlay
// backend depends on syscall.Mount("overlay", ...), which only Linux
// provides. It returns a typed BackendUnsupportedError (matching
// ErrBackendUnsupported) so callers surface a clear platform error instead
// of shelling out to a `mount` binary that cannot help.
func sysMountOverlay(_, _ string) error {
	return &BackendUnsupportedError{Backend: BackendKernelOverlay.String(), GOOS: runtime.GOOS}
}

// sysUnmount has no implementation off Linux for the same reason. In
// practice SystemMounter.Unmount returns early (IsMountPoint is false when
// the `mountpoint` binary is absent), so this path is unreachable at
// runtime; it exists to keep the seam compiling across platforms.
func sysUnmount(_ string, _ bool) error {
	return &BackendUnsupportedError{Backend: BackendKernelOverlay.String(), GOOS: runtime.GOOS}
}
