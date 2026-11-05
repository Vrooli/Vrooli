package bundle

import (
	"fmt"
	"runtime"
	"strings"

	bundlemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
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

	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid platform key %q", key)
	}
	goos := parts[0]
	switch goos {
	case "win":
		goos = "windows"
	case "mac", "macos":
		goos = "darwin"
	case "darwin", "linux", "windows":
	default:
		return "", "", fmt.Errorf("unsupported platform os %q", goos)
	}

	goarch := parts[1]
	switch goarch {
	case "x64", "amd64":
		goarch = "amd64"
	case "arm64", "aarch64":
		goarch = "arm64"
	default:
		return "", "", fmt.Errorf("unsupported arch %q", goarch)
	}

	return goos, goarch, nil
}

// NormalizeRuntime normalizes a platform key for runtime staging.
func (p *defaultPlatformResolver) NormalizeRuntime(platform string) string {
	switch platform {
	case "linux", "linux-amd64", "linux-x64":
		return "linux-x64"
	case "linux-arm64", "linux-aarch64":
		return "linux-arm64"
	case "mac", "macos", "darwin", "mac-x64", "macos-x64", "macos-amd64", "darwin-amd64", "darwin-x64":
		return "darwin-x64"
	case "mac-arm64", "macos-arm64", "darwin-arm64":
		return "darwin-arm64"
	case "win", "windows", "win-x64", "windows-amd64", "windows-x64":
		return "win-x64"
	case "win-arm64", "windows-arm64":
		return "win-arm64"
	default:
		return platform
	}
}

// RuntimeBinaryName returns the runtime binary name for a GOOS.
func (p *defaultPlatformResolver) RuntimeBinaryName(goos string) string {
	if goos == "windows" {
		return "runtime.exe"
	}
	return "runtime"
}

// RuntimeCtlBinaryName returns the runtimectl binary name for a GOOS.
func (p *defaultPlatformResolver) RuntimeCtlBinaryName(goos string) string {
	if goos == "windows" {
		return "runtimectl.exe"
	}
	return "runtimectl"
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
	case "amd64":
		goarch = "amd64"
	case "arm64":
		goarch = "arm64"
	default:
		goarch = "amd64" // default fallback
	}

	switch platform {
	case "linux":
		return "linux", goarch
	case "win", "windows":
		return "windows", goarch
	case "mac", "macos", "darwin":
		return "darwin", goarch
	}
	return "", ""
}

func expandShorthandPlatform(platform string) []string {
	archs := []string{"x64", "arm64", "amd64", "aarch64"}
	var keys []string

	switch platform {
	case "win", "windows":
		for _, arch := range archs {
			keys = append(keys, "win-"+arch, "windows-"+arch)
		}
	case "mac", "macos", "darwin":
		for _, arch := range archs {
			keys = append(keys, "darwin-"+arch, "mac-"+arch, "macos-"+arch)
		}
	case "linux":
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
	if strings.HasPrefix(key, "macos-") {
		return "darwin-" + strings.TrimPrefix(key, "macos-")
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
