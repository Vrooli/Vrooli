//go:build !linux

package fsmount

import (
	"errors"
	"testing"
)

// TestSysMountOverlay_UnsupportedOffLinux asserts the non-Linux build of the
// raw mount fast path returns the typed BackendUnsupportedError (matching
// ErrBackendUnsupported). Tagged !linux so it provides compile + behavior
// coverage of mount_other.go on darwin/windows without affecting Linux CI.
func TestSysMountOverlay_UnsupportedOffLinux(t *testing.T) {
	if err := sysMountOverlay("/merged", "lowerdir=a,upperdir=b,workdir=c"); !errors.Is(err, ErrBackendUnsupported) {
		t.Fatalf("sysMountOverlay off Linux = %v, want ErrBackendUnsupported", err)
	}
	if err := sysUnmount("/merged", false); !errors.Is(err, ErrBackendUnsupported) {
		t.Fatalf("sysUnmount off Linux = %v, want ErrBackendUnsupported", err)
	}
}
