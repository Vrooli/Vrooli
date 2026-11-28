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
	goos, goarch, err := p.ParseKey(platform)
	if err != nil {
		return platform
	}

	arch := goarch
	if goarch == "amd64" {
		arch = "x64"
	}
	if goos == "windows" {
		return "win-" + arch
	}
	return goos + "-" + arch
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
	goos, goarch, err := p.ParseKey(platform)
	if err != nil {
		return bundlemanifest.Binary{}, false
	}
	for _, key := range platformBinaryKeys(goos, goarch) {
		if bin, ok := svc.Binaries[key]; ok {
			return bin, true
		}
	}
	// A shorthand manifest entry has no architecture metadata. Treat it as a
	// build-host artifact and never reuse it for a cross-architecture target.
	if goarch == runtime.GOARCH {
		for _, key := range platformOSAliases(goos) {
			if bin, ok := svc.Binaries[key]; ok {
				return bin, true
			}
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
		return "", ""
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

func platformBinaryKeys(goos, goarch string) []string {
	var architectures []string
	switch goarch {
	case "amd64":
		architectures = []string{"amd64", "x64"}
	case "arm64":
		architectures = []string{"arm64", "aarch64"}
	default:
		return nil
	}
	var keys []string
	for _, osName := range platformOSAliases(goos) {
		for _, architecture := range architectures {
			keys = append(keys, osName+"-"+architecture)
		}
	}
	return keys
}

func platformOSAliases(goos string) []string {
	switch goos {
	case "darwin":
		return []string{"darwin", "macos", "mac"}
	case "windows":
		return []string{"windows", "win"}
	case "linux":
		return []string{"linux"}
	default:
		return nil
	}
}
