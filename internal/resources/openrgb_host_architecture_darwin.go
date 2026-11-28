//go:build darwin

package resources

import (
	"errors"
	"fmt"
	"runtime"

	"golang.org/x/sys/unix"
)

func verifyOpenRGBNativeHostArchitecture() error {
	translated, err := unix.SysctlUint32("sysctl.proc_translated")
	// Apple's documented ENOENT outcome means translation is unavailable
	// (including an Intel host), rather than an unknown translated process.
	if err != nil && !errors.Is(err, unix.ENOENT) {
		return fmt.Errorf("native OpenRGB translation observation: %w", err)
	}
	if translated != 0 {
		return errors.New("OpenRGB native qualification requires the native control-plane architecture; Rosetta translation is unqualified")
	}
	profile, err := openRGBReleaseProfileFor("darwin", runtime.GOARCH)
	if err != nil {
		return err
	}
	version, err := unix.Sysctl("kern.osproductversion")
	if err != nil {
		return fmt.Errorf("native macOS product version unavailable: %w", err)
	}
	if !openRGBDarwinVersionAtLeast(version, profile.MinimumDarwinVersion) {
		return errors.New("OpenRGB1.1 requires macOS15.0 or later before native setup")
	}
	return nil
}
