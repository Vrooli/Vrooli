package deployability

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const acquisitionDarwin = string(hostreqspec.PlatformDarwin)

// requiredAcquisitionArchitectures lists, per host OS, the architectures an
// arch-selected acquisition must account for. Linux fleet nodes run on both
// amd64 and arm64, so a release-backed tool that declares Linux targets must
// say what happens on each: a verified target, a package fallback, or an
// explicit unsupported reason. Without that, a missing target reads as an
// installable route until an arm64 host tries to install it. macOS and
// Windows are held to OS-level coverage only.
var requiredAcquisitionArchitectures = map[HostOS][]string{
	HostOSLinux: {"amd64", "arm64"},
}

// AcquisitionCoverageDeclaration is the manifest subset needed to prove that
// a tool has a deterministic path on every platform it claims. The path may
// be a checksum-verified acquisition target, a package fallback, a custom
// handler, an explicitly manual workflow, or an explicit unsupported target.
type AcquisitionCoverageDeclaration struct {
	Name string
	// Platforms lists the host OSes the tool claims. Nil means the manifest
	// declares nothing, which is read as all host OSes; an empty non-nil
	// slice means the manifest explicitly claims none.
	Platforms        []HostOS
	PackageFallbacks map[HostOS]string
	Acquisition      *binaryfetch.Acquisition
	Handler          string
	Manual           bool
}

// ValidateAcquisitionCoverage is the single acquisition-route rule for tool
// manifests. Every claimed OS needs at least one declared route or an
// explicit reason why it is unsupported. In addition, every OS for which the
// acquisition declares targets must resolve each of its required
// architectures to a target, an explicit unsupported reason, or a package
// fallback, because the runtime installer selects targets by OS and
// architecture together.
func ValidateAcquisitionCoverage(declaration AcquisitionCoverageDeclaration) error {
	name := strings.TrimSpace(declaration.Name)
	if name == "" {
		name = "tool"
	}
	platforms := declaration.Platforms
	if platforms == nil {
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
	return validateArchitectureCoverage(name, declaration)
}

// validateArchitectureCoverage applies the architecture rule independently
// of the claimed platforms: the runtime installer acts on declared targets
// whatever a platform_status says, so declared targets must be complete.
func validateArchitectureCoverage(name string, declaration AcquisitionCoverageDeclaration) error {
	if declaration.Acquisition == nil {
		return nil
	}
	platforms := make([]HostOS, 0, len(requiredAcquisitionArchitectures))
	for platform := range requiredAcquisitionArchitectures {
		platforms = append(platforms, platform)
	}
	sort.Slice(platforms, func(i, j int) bool { return platforms[i] < platforms[j] })
	for _, platform := range platforms {
		if !acquisitionTargetsPlatform(declaration.Acquisition, platform) {
			continue
		}
		if strings.TrimSpace(declaration.PackageFallbacks[platform]) != "" {
			continue
		}
		for _, arch := range requiredAcquisitionArchitectures[platform] {
			if acquisitionResolvesArchitecture(declaration.Acquisition, platform, arch) {
				continue
			}
			return fmt.Errorf("tool %q declares %s acquisition targets but neither a %s/%s target, a package fallback, nor an explicit unsupported reason", name, platform, platform, arch)
		}
	}
	return nil
}

// acquisitionTargetsPlatform reports whether any target can apply to the OS,
// meaning the OS's route depends on arch-selected targets.
func acquisitionTargetsPlatform(acquisition *binaryfetch.Acquisition, platform HostOS) bool {
	for _, target := range acquisition.Targets {
		targetOS, constrained := target.When["os"]
		if !constrained || normalizeFactOS(targetOS) == string(platform) {
			return true
		}
	}
	return false
}

// acquisitionResolvesArchitecture uses the installer's own first-match
// resolution, so the rule cannot disagree with what an install would select.
func acquisitionResolvesArchitecture(acquisition *binaryfetch.Acquisition, platform HostOS, arch string) bool {
	_, err := acquisition.Resolve(binaryfetch.Facts{"os": goosFor(platform), "arch": arch})
	if err == nil {
		return true
	}
	var unsupported *binaryfetch.UnsupportedError
	return errors.As(err, &unsupported)
}

func acquisitionCoversPlatform(acquisition *binaryfetch.Acquisition, platform HostOS) bool {
	if acquisition == nil {
		return false
	}
	for _, target := range acquisition.Targets {
		if targetOS, ok := target.When["os"]; ok && normalizeFactOS(targetOS) != string(platform) {
			continue
		}
		if strings.TrimSpace(target.Unsupported) != "" || strings.TrimSpace(target.URL) != "" {
			return true
		}
		// An OCI image is a Linux filesystem tree, so it only covers Linux. A
		// macOS or Windows target pointing at one stages an ELF binary that
		// cannot execute there, which previously read as genuine coverage and
		// let the capability ledger report support that does not exist.
		if strings.TrimSpace(target.Image) != "" && platform == HostOSLinux {
			return true
		}
	}
	return false
}

// toolManifestCoverage is the subset of internal/tools/<name>/tool.json the
// acquisition rule reads.
type toolManifestCoverage struct {
	Name           string   `json:"name"`
	Platforms      []string `json:"platforms"`
	PlatformStatus map[string]struct {
		Status string `json:"status"`
	} `json:"platform_status"`
	Packages       map[string]string        `json:"packages"`
	DefaultPackage string                   `json:"defaultPackage"`
	Handler        string                   `json:"handler"`
	Manual         bool                     `json:"manual"`
	Acquisition    *binaryfetch.Acquisition `json:"acquisition"`
}

// ToolManifestAcquisitionCoverage maps one tool.json document onto the
// coverage declaration. platform_status is the canonical platform claim; the
// legacy platforms array is read only when a manifest has no platform_status.
// A platform is claimed unless its status is ineligible (unsupported,
// not_applicable) or not_implemented, since neither has anything to acquire.
func ToolManifestAcquisitionCoverage(data []byte) (AcquisitionCoverageDeclaration, error) {
	var manifest toolManifestCoverage
	if err := json.Unmarshal(data, &manifest); err != nil {
		return AcquisitionCoverageDeclaration{}, err
	}
	platforms, err := claimedToolPlatforms(manifest)
	if err != nil {
		return AcquisitionCoverageDeclaration{}, fmt.Errorf("tool %q: %w", manifest.Name, err)
	}
	return AcquisitionCoverageDeclaration{
		Name:             manifest.Name,
		Platforms:        platforms,
		PackageFallbacks: toolPackageFallbacks(manifest.Packages, manifest.DefaultPackage, platforms),
		Acquisition:      manifest.Acquisition,
		Handler:          strings.TrimSpace(manifest.Handler),
		Manual:           manifest.Manual,
	}, nil
}

// ValidateToolManifestAcquisitionCoverage applies ValidateAcquisitionCoverage
// to every <name>/tool.json in manifests and reports every failing tool, not
// only the first. The control plane passes its embedded manifests; scenario
// callers pass the repository's internal/tools directory.
func ValidateToolManifestAcquisitionCoverage(manifests fs.FS) error {
	paths, err := fs.Glob(manifests, "*/tool.json")
	if err != nil {
		return fmt.Errorf("enumerate tool manifests: %w", err)
	}
	sort.Strings(paths)
	var failures []error
	for _, path := range paths {
		data, err := fs.ReadFile(manifests, path)
		if err != nil {
			failures = append(failures, fmt.Errorf("read %s: %w", path, err))
			continue
		}
		declaration, err := ToolManifestAcquisitionCoverage(data)
		if err != nil {
			failures = append(failures, fmt.Errorf("decode %s: %w", path, err))
			continue
		}
		if err := ValidateAcquisitionCoverage(declaration); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", path, err))
		}
	}
	return errors.Join(failures...)
}

func claimedToolPlatforms(manifest toolManifestCoverage) ([]HostOS, error) {
	if len(manifest.PlatformStatus) > 0 {
		platforms := make([]HostOS, 0, len(manifest.PlatformStatus))
		for rawOS, declaration := range manifest.PlatformStatus {
			platform := normalizeHostOS(HostOS(rawOS))
			if platform == "" {
				return nil, fmt.Errorf("platform_status names unknown OS %q", rawOS)
			}
			status, err := ParsePlatformStatus(declaration.Status)
			if err != nil {
				return nil, fmt.Errorf("platform_status.%s: %w", rawOS, err)
			}
			if status.Qualification() == QualificationIneligible || status == StatusNotImplemented {
				continue
			}
			platforms = append(platforms, platform)
		}
		sort.Slice(platforms, func(i, j int) bool { return platforms[i] < platforms[j] })
		return platforms, nil
	}
	if len(manifest.Platforms) == 0 {
		return nil, nil
	}
	platforms := make([]HostOS, 0, len(manifest.Platforms))
	for _, rawOS := range manifest.Platforms {
		platform := normalizeHostOS(HostOS(rawOS))
		if platform == "" {
			return nil, fmt.Errorf("platforms names unknown OS %q", rawOS)
		}
		platforms = append(platforms, platform)
	}
	return platforms, nil
}

// toolPackageFallbacks maps declared packages onto the OS whose package
// manager installs them. defaultPackage applies to every claimed OS.
func toolPackageFallbacks(packages map[string]string, defaultPackage string, platforms []HostOS) map[HostOS]string {
	result := make(map[HostOS]string)
	if strings.TrimSpace(defaultPackage) != "" {
		claimed := platforms
		if claimed == nil {
			claimed = []HostOS{HostOSLinux, HostOSMacOS, HostOSWindows}
		}
		for _, platform := range claimed {
			result[platform] = strings.TrimSpace(defaultPackage)
		}
	}
	for manager, packageName := range packages {
		packageName = strings.TrimSpace(packageName)
		if packageName == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(manager)) {
		case "brew":
			result[HostOSMacOS] = packageName
		case "winget", "choco", "scoop":
			result[HostOSWindows] = packageName
		default:
			result[HostOSLinux] = packageName
		}
	}
	return result
}

func goosFor(platform HostOS) string {
	if platform == HostOSMacOS {
		return acquisitionDarwin
	}
	return string(platform)
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
	case string(hostreqspec.PlatformLinux):
		return string(HostOSLinux)
	case string(hostreqspec.PlatformWindows), "win32":
		return string(HostOSWindows)
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}
