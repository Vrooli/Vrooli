package main

import (
	"fmt"
	"runtime"
	"strings"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

// parsePlatformKey parses a platform key (e.g., "linux-x64") into GOOS and GOARCH.
func parsePlatformKey(key string) (string, string, error) {
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
	case "mac":
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

// expandShorthandToHostArch expands shorthand platform names to goos/goarch using host architecture.
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
	case "win", "windows":
		return "windows", goarch
	case "mac", "darwin":
		return "darwin", goarch
	}
	return "", ""
}

// expandShorthandPlatform expands shorthand platform names to architecture-specific keys.
// For example, "win" -> ["win-x64", "win-arm64", "windows-x64", "windows-arm64"]
func expandShorthandPlatform(platform string) []string {
	archs := []string{"x64", "arm64", "amd64", "aarch64"}
	var keys []string

	switch platform {
	case "win", "windows":
		for _, arch := range archs {
			keys = append(keys, "win-"+arch, "windows-"+arch)
		}
	case "mac", "darwin":
		for _, arch := range archs {
			keys = append(keys, "darwin-"+arch, "mac-"+arch)
		}
	case "linux":
		for _, arch := range archs {
			keys = append(keys, "linux-"+arch)
		}
	}
	return keys
}

// aliasPlatformKey returns an alias for a platform key (e.g., "win-x64" -> "windows-x64").
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

// normalizeRuntimePlatform ensures the runtime/CLI staging uses arch-suffixed directories
// that match the Electron runtimePlatformKey (e.g., linux-x64, win-x64, darwin-x64).
func normalizeRuntimePlatform(platform string) string {
	switch platform {
	case "linux":
		return "linux-x64"
	case "mac", "darwin":
		return "darwin-x64"
	case "win", "windows":
		return "win-x64"
	default:
		return platform
	}
}

// runtimeBinaryName returns the runtime binary filename for a given GOOS.
func runtimeBinaryName(goos string) string {
	if goos == "windows" {
		return "runtime.exe"
	}
	return "runtime"
}

// runtimeCtlBinaryName returns the runtimectl binary filename for a given GOOS.
func runtimeCtlBinaryName(goos string) string {
	if goos == "windows" {
		return "runtimectl.exe"
	}
	return "runtimectl"
}

// resolveBinaryForPlatform resolves the binary entry for a service on a given platform.
func resolveBinaryForPlatform(svc bundlemanifest.Service, platform string) (bundlemanifest.Binary, bool) {
	keys := []string{platform}
	if alias := aliasPlatformKey(platform); alias != "" {
		keys = append(keys, alias)
	}
	// Try exact and aliased matches first
	for _, key := range keys {
		if bin, ok := svc.Binaries[key]; ok {
			return bin, true
		}
	}
	// For shorthand platforms (win, mac, linux), try architecture-specific keys
	archKeys := expandShorthandPlatform(platform)
	for _, key := range archKeys {
		if bin, ok := svc.Binaries[key]; ok {
			return bin, true
		}
	}
	return bundlemanifest.Binary{}, false
}
