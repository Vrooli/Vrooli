package platform

import (
	"fmt"
	"runtime"
	"strings"
)

// Current returns the host identity using the naming used by resource
// acquisition contracts.
func Current() (string, string) {
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "macos"
	}
	return osName, runtime.GOARCH
}

func CheckCurrent() error {
	osName, arch := Current()
	return Check(osName, arch)
}

// Check rejects targets before the parser artifact is opened. WASI is
// intentionally available on the five desktop pairs produced by the build
// lane; the gate remains explicit so unsupported targets fail with a useful
// reason rather than a runtime error from Wazero.
func Check(osName, arch string) error {
	osName = strings.ToLower(strings.TrimSpace(osName))
	arch = strings.ToLower(strings.TrimSpace(arch))
	if osName == "darwin" {
		osName = "macos"
	}
	if osName != "linux" && osName != "macos" && osName != "windows" {
		return fmt.Errorf("unsupported platform %s-%s: doc-parse WASI artifacts are produced only for linux, macos, and windows", osName, arch)
	}
	if arch != "amd64" && arch != "arm64" {
		return fmt.Errorf("unsupported platform %s-%s: doc-parse artifacts are produced only for amd64 and arm64", osName, arch)
	}
	if osName == "windows" && arch == "arm64" {
		return fmt.Errorf("unsupported platform %s-%s: the current desktop build lane produces windows-amd64 only", osName, arch)
	}
	return nil
}
