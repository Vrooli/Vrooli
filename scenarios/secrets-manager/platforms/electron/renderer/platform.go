package bundle

import (
	"fmt"
	"runtime"
	"strings"

	bundlemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

const (
	platformWin        = "win"
	platformMac        = "mac"
	platformDarwin     = "darwin"
	platformLinux      = "linux"
	platformWindows    = "windows"
	platformAMD64      = "amd64"
	platformArm64      = "arm64"
	platformWinX64     = "win-x64"
	platformRuntime    = "runtime"
	platformRuntimeCtl = "runtimectl"
	platformExe        = ".exe"
	platformWarning    = "warning"
)

// defaultPlatformResolver is the default implementation of PlatformResolver.
type defaultPlatformResolver struct{}

// ParseKey parses a platform key into GOOS and GOARCH.
func (p *defaultPlatformResolver) ParseKey(key string) (string, string, error) {
	parts := strings.Split(key, "-")

	// Handle shorthand platforms (win, mac, linux) without architecture
	if len(parts) == 1 {
		goos, goarch := expandShorthandToHostArch(key)
		if goos != "" {
			return goos, goarch, nil
		}
		return "", "", fmt.Errorf("invalid platform key %q", key)
	}

	if len(parts) != 2 { //nolint:mnd // platform identifier has OS and architecture parts
		return "", "", fmt.Errorf("invalid platform key %q", key)
	}
	goos := parts[0]
	switch goos {
	case platformWin:
		goos = platformWindows
	case platformMac:
		goos = platformDarwin
	case platformDarwin, platformLinux, platformWindows:
	default:
		return "", "", fmt.Errorf("unsupported platform os %q", goos)
	}

	goarch := parts[1]
	switch goarch {
	case "x64", platformAMD64:
		goarch = platformAMD64
	case platformArm64, "aarch64":
		goarch = platformArm64
	default:
		return "", "", fmt.Errorf("unsupported arch %q", goarch)
	}

	return goos, goarch, nil
}

// NormalizeRuntime normalizes a platform key for runtime staging.
func (p *defaultPlatformResolver) NormalizeRuntime(platform string) string {
	switch platform {
	case platformLinux, "linux-amd64", "linux-x64":
		return "linux-x64"
	case "linux-arm64", "linux-aarch64":
		return "linux-arm64"
	case platformMac, platformDarwin, "mac-x64", "darwin-amd64", "darwin-x64":
		return "darwin-x64"
	case "mac-arm64", "darwin-arm64":
		return "darwin-arm64"
	case platformWin, platformWindows, platformWinX64, "windows-amd64", "windows-x64":
		return platformWinX64
	case "win-arm64", "windows-arm64":
		return "win-arm64"
	default:
		return platform
	}
}

// RuntimeBinaryName returns the runtime binary name for a GOOS.
func (p *defaultPlatformResolver) RuntimeBinaryName(goos string) string {
	if goos == platformWindows {
		return "runtime.exe"
	}
	return platformRuntime
}

// RuntimeCtlBinaryName returns the runtimectl binary name for a GOOS.
func (p *defaultPlatformResolver) RuntimeCtlBinaryName(goos string) string {
	if goos == platformWindows {
		return "runtimectl.exe"
	}
	return platformRuntimeCtl
}

// ResolveBinaryForPlatform resolves the binary entry for a service on a platform.
func (p *defaultPlatformResolver) ResolveBinaryForPlatform(svc bundlemanifest.Service, platform string) (bundlemanifest.Binary, bool) {
	keys := []string{platform}
	if alias := aliasPlatformKey(platform); alias != "" {
		keys = append(keys, alias)
		if architectureAlias := aliasArchitecturePlatformKey(alias); architectureAlias != "" {
			keys = append(keys, architectureAlias)
		}
	}
	if alias := aliasArchitecturePlatformKey(platform); alias != "" {
		keys = append(keys, alias)
	}
	// Try exact and aliased matches first
	for _, key := range keys {
		if bin, ok := svc.Binaries[key]; ok {
			return bin, true
		}
	}
	// For shorthand platforms, try architecture-specific keys
	archKeys := expandShorthandPlatform(platform)
	for _, key := range archKeys {
		if bin, ok := svc.Binaries[key]; ok {
			return bin, true
		}
	}
	return bundlemanifest.Binary{}, false
}

// Helper functions

func expandShorthandToHostArch(platform string) (string, string) {
	goarch := runtime.GOARCH
	switch goarch {
	case platformAMD64:
		goarch = platformAMD64
	case platformArm64:
		goarch = platformArm64
	default:
		goarch = platformAMD64 // default fallback
	}

	switch platform {
	case platformLinux:
		return platformLinux, goarch
	case platformWin, platformWindows:
		return platformWindows, goarch
	case platformMac, platformDarwin:
		return platformDarwin, goarch
	}
	return "", ""
}

func expandShorthandPlatform(platform string) []string {
	archs := []string{"x64", platformArm64, platformAMD64, "aarch64"}
	var keys []string

	switch platform {
	case platformWin, platformWindows:
		for _, arch := range archs {
			keys = append(keys, "win-"+arch, "windows-"+arch)
		}
	case platformMac, platformDarwin:
		for _, arch := range archs {
			keys = append(keys, "darwin-"+arch, "mac-"+arch)
		}
	case platformLinux:
		for _, arch := range archs {
			keys = append(keys, "linux-"+arch)
		}
	}
	return keys
}

func aliasPlatformKey(key string) string {
	if strings.HasPrefix(key, "windows-") {
		return "win-" + strings.TrimPrefix(key, "windows-")
	}
	if strings.HasPrefix(key, "win-") {
		return "windows-" + strings.TrimPrefix(key, "win-")
	}
	if strings.HasPrefix(key, "darwin-") {
		return "mac-" + strings.TrimPrefix(key, "darwin-")
	}
	if strings.HasPrefix(key, "mac-") {
		return "darwin-" + strings.TrimPrefix(key, "mac-")
	}
	return ""
}

// aliasArchitecturePlatformKey treats the two common architecture spellings as
// one target. Resource deployment uses Go's amd64 spelling while desktop
// manifests conventionally use x64, so both must resolve the same binary.
func aliasArchitecturePlatformKey(key string) string {
	switch {
	case strings.HasSuffix(key, "-amd64"):
		return strings.TrimSuffix(key, "-amd64") + "-x64"
	case strings.HasSuffix(key, "-x64"):
		return strings.TrimSuffix(key, "-x64") + "-amd64"
	case strings.HasSuffix(key, "-aarch64"):
		return strings.TrimSuffix(key, "-aarch64") + "-arm64"
	case strings.HasSuffix(key, "-arm64"):
		return strings.TrimSuffix(key, "-arm64") + "-aarch64"
	default:
		return ""
	}
}
