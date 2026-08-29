package deployability

import (
	"fmt"
	"strings"

	"github.com/vrooli/binaryfetch"
)

const acquisitionDarwin = "darwin"

// AcquisitionCoverageDeclaration is the manifest subset needed to prove that
// a tool has a deterministic path on every platform it claims. The path may
// be a checksum-verified acquisition target, a package fallback, a custom
// handler, an explicitly manual workflow, or an explicit unsupported target.
type AcquisitionCoverageDeclaration struct {
	Name             string
	Platforms        []HostOS
	PackageFallbacks map[HostOS]string
	Acquisition      *binaryfetch.Acquisition
	Handler          string
	Manual           bool
}

// ValidateAcquisitionCoverage applies the same rule to every claimed platform.
// An omitted platforms list means all supported host OSes. Architecture
// selection remains in the acquisition target predicate; this check ensures
// that each claimed OS has at least one declared route or an explicit reason
// why it is unsupported.
func ValidateAcquisitionCoverage(declaration AcquisitionCoverageDeclaration) error {
	name := strings.TrimSpace(declaration.Name)
	if name == "" {
		name = "tool"
	}
	platforms := declaration.Platforms
	if len(platforms) == 0 {
		platforms = []HostOS{HostOSLinux, HostOSMacOS, HostOSWindows}
	}
	seen := make(map[HostOS]struct{}, len(platforms))
	for _, platform := range platforms {
		platform = normalizeHostOS(platform)
		if platform == "" {
			return fmt.Errorf("%s declares an unknown platform", name)
		}
		if _, exists := seen[platform]; exists {
			continue
		}
		seen[platform] = struct{}{}
		if strings.TrimSpace(declaration.PackageFallbacks[platform]) != "" || declaration.Manual || strings.TrimSpace(declaration.Handler) != "" {
			continue
		}
		if acquisitionCoversPlatform(declaration.Acquisition, platform) {
			continue
		}
		return fmt.Errorf("tool %q claims %s without an acquisition target, package fallback, handler, manual path, or explicit unsupported reason", name, platform)
	}
	return nil
}

func acquisitionCoversPlatform(acquisition *binaryfetch.Acquisition, platform HostOS) bool {
	if acquisition == nil {
		return false
	}
	for _, target := range acquisition.Targets {
		if targetOS, ok := target.When["os"]; ok && normalizeFactOS(targetOS) != string(platform) {
			continue
		}
		if strings.TrimSpace(target.Unsupported) != "" || strings.TrimSpace(target.URL) != "" || strings.TrimSpace(target.Image) != "" {
			return true
		}
	}
	return false
}

func normalizeHostOS(value HostOS) HostOS {
	switch normalizeFactOS(string(value)) {
	case string(HostOSLinux):
		return HostOSLinux
	case string(HostOSMacOS):
		return HostOSMacOS
	case string(HostOSWindows):
		return HostOSWindows
	default:
		return ""
	}
}

func normalizeFactOS(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case acquisitionDarwin, "mac", string(HostOSMacOS):
		return string(HostOSMacOS)
	case "linux":
		return string(HostOSLinux)
	case "windows", "win32":
		return string(HostOSWindows)
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}
