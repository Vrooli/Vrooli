//go:build windows

package resources

import (
	"debug/pe"
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

func verifyOpenRGBNativeHostArchitecture() error {
	profile, err := openRGBReleaseProfileFor("windows", "amd64")
	if err != nil {
		return err
	}
	version := windows.RtlGetVersion()
	if !openRGBWindowsVersionAtLeast(version.MajorVersion, version.MinorVersion, version.BuildNumber, profile.MinimumWindowsBuild) {
		return errors.New("OpenRGB1.1 Qt6.8 requires native Windows10 version1809 (build17763) or later")
	}
	var process, native uint16
	if err := windows.IsWow64Process2(windows.CurrentProcess(), &process, &native); err != nil {
		return fmt.Errorf("native OpenRGB architecture observation: %w", err)
	}
	if process != pe.IMAGE_FILE_MACHINE_UNKNOWN || native != pe.IMAGE_FILE_MACHINE_AMD64 {
		return errors.New("OpenRGB1.1 requires native Windows x64; Windows ARM64 and emulated process profiles are unsupported")
	}
	return nil
}
