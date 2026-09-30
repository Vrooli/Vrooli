//go:build !linux

package namespace

import (
	"runtime"
	"strings"
	"testing"
)

// TestGetKernelVersion_OffLinux asserts the non-Linux build returns a
// non-empty, GOOS-descriptive string with no syscalls. Tagged !linux so it
// gives compile + behavior coverage of kernel_other.go on darwin/windows.
func TestGetKernelVersion_OffLinux(t *testing.T) {
	v := getKernelVersion()
	if v == "" {
		t.Fatal("getKernelVersion() off Linux returned empty string")
	}
	if !strings.Contains(v, runtime.GOOS) {
		t.Errorf("getKernelVersion() = %q, want it to mention GOOS %q", v, runtime.GOOS)
	}
}
