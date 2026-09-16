package targetmodel

import (
	"fmt"
	"strings"
)

// Canonical desktop platform vocabulary.
//
// Desktop packaging is architecture-aware and uses arch-qualified target
// identifiers such as "linux-x64", "darwin-arm64", and "win-x64". The desktop
// download catalog and the electron update feed are operating-system scoped and
// use the vocabulary below. A single projection between the two prevents a
// producer-specific spelling from reaching a destination that keys assets by
// operating system.
const (
	DesktopOSWindows = "windows"
	DesktopOSMac     = "mac"
	DesktopOSLinux   = "linux"

	DesktopArchAMD64 = "amd64"
	DesktopArchARM64 = "arm64"
)

// ProjectDesktopPlatform maps a Vrooli execution target identifier or desktop
// platform spelling to the canonical operating-system platform and
// architecture.
//
// It accepts OS-only spellings ("linux", "mac", "darwin", "win") and
// architecture-qualified target identifiers ("linux-x64", "linux-amd64",
// "darwin-arm64", "win-x64", "windows-amd64"). An architecture-less spelling
// returns an empty architecture, which callers must treat as "unspecified"
// rather than "all architectures".
//
// It returns an error for an unknown OS or architecture instead of defaulting,
// because artifact selection and channel promotion must never guess a machine.
func ProjectDesktopPlatform(target string) (platform string, architecture string, err error) {
	raw := strings.ToLower(strings.TrimSpace(target))
	if raw == "" {
		return "", "", fmt.Errorf("desktop platform is empty")
	}
	osToken, archToken := splitDesktopTarget(raw)
	osPlatform, ok := canonicalDesktopOS(osToken)
	if !ok {
		return "", "", fmt.Errorf("unsupported desktop OS %q", osToken)
	}
	if archToken == "" {
		return osPlatform, "", nil
	}
	arch, ok := canonicalDesktopArch(archToken)
	if !ok {
		return "", "", fmt.Errorf("unsupported desktop architecture %q", archToken)
	}
	return osPlatform, arch, nil
}

// splitDesktopTarget separates an OS token from an optional architecture token
// at the first `-`, `_`, or `/` separator. An architecture remainder is
// case-normalized to use `-` so spellings such as `x86_64` resolve to the same
// architecture as `x86-64`.
func splitDesktopTarget(raw string) (osToken, archToken string) {
	index := -1
	for _, separator := range []rune{'-', '_', '/'} {
		if at := strings.IndexRune(raw, separator); at >= 0 && (index == -1 || at < index) {
			index = at
		}
	}
	if index == -1 {
		return raw, ""
	}
	archToken = strings.ReplaceAll(raw[index+1:], "_", "-")
	return raw[:index], strings.Trim(archToken, "-")
}

func canonicalDesktopOS(value string) (string, bool) {
	switch value {
	case "linux":
		return DesktopOSLinux, true
	case "mac", "macos", "darwin", "osx":
		return DesktopOSMac, true
	case "win", "windows":
		return DesktopOSWindows, true
	default:
		return "", false
	}
}

func canonicalDesktopArch(value string) (string, bool) {
	switch value {
	case "amd64", "x64", "x86-64":
		return DesktopArchAMD64, true
	case "arm64", "aarch64", "armv8":
		return DesktopArchARM64, true
	default:
		return "", false
	}
}
