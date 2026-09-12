//go:build windows

package changedetect

import "testing"

// TestIsCharDevWhiteout_WindowsAlwaysFalse asserts the Windows stub reports
// no overlayfs whiteouts: they are a Linux overlay construct that cannot
// exist on a Windows filesystem. Tagged windows for compile + behavior
// coverage of whiteout_windows.go.
func TestIsCharDevWhiteout_WindowsAlwaysFalse(t *testing.T) {
	if isCharDevWhiteout("C:\\any\\path") {
		t.Fatal("isCharDevWhiteout on Windows returned true, want false")
	}
}
