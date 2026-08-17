package dependencyhealth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/packages/deployability"
)

type portabilityToolManifest struct {
	Name           string            `json:"name"`
	Platforms      []string          `json:"platforms"`
	Packages       map[string]string `json:"packages"`
	DefaultPackage string            `json:"defaultPackage"`
	Handler        string            `json:"handler"`
	Manual         bool              `json:"manual"`
	Source         json.RawMessage   `json:"source"`
	Acquisition    json.RawMessage   `json:"acquisition"`
}

// validateToolAcquisitionCoverage validates the repository-wide tool contract
// while the portability provider is running. This is deliberately manifest
// driven: acquisition policy must not grow a second list of tool names.
func validateToolAcquisitionCoverage(repoRoot string) error {
	paths, err := filepath.Glob(filepath.Join(repoRoot, "internal", "tools", "*", "tool.json"))
	if err != nil {
		return fmt.Errorf("enumerate tool manifests: %w", err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read tool manifest %q: %w", path, err)
		}
		var manifest portabilityToolManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return fmt.Errorf("decode tool manifest %q: %w", path, err)
		}
		platforms := make([]deployability.HostOS, 0, len(manifest.Platforms))
		for _, rawOS := range manifest.Platforms {
			if hostOS, ok := portabilityNormalizeOS(rawOS); ok {
				platforms = append(platforms, hostOS)
			}
		}
		source := ""
		if len(manifest.Source) > 0 && string(manifest.Source) != "null" {
			var sourceManifest struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(manifest.Source, &sourceManifest); err != nil {
				return fmt.Errorf("decode source in tool manifest %q: %w", path, err)
			}
			source = sourceManifest.Type
		}
		// A tool's governed acquisition declaration is itself a source target.
		// Do not require a legacy `source` field when the manifest already
		// carries checksum-verified per-platform acquisition targets.
		if source == "" && hasAcquisitionTargets(manifest.Acquisition) {
			source = "declared-acquisition"
		}
		var acquisition *binaryfetch.Acquisition
		if len(manifest.Acquisition) > 0 && string(manifest.Acquisition) != "null" {
			var parsed binaryfetch.Acquisition
			if err := json.Unmarshal(manifest.Acquisition, &parsed); err != nil {
				return fmt.Errorf("decode acquisition in tool manifest %q: %w", path, err)
			}
			acquisition = &parsed
		}
		fallbacks := packageFallbacks(manifest.Packages, manifest.DefaultPackage, platforms)
		declaration := deployability.AcquisitionCoverageDeclaration{
			Name:             manifest.Name,
			Platforms:        platforms,
			PackageFallbacks: fallbacks,
			Acquisition:      acquisition,
			Handler:          strings.TrimSpace(manifest.Handler),
			Manual:           manifest.Manual,
		}
		if err := deployability.ValidateAcquisitionCoverage(declaration); err != nil {
			return fmt.Errorf("tool %q: %w", manifest.Name, err)
		}
	}
	return nil
}

func packageFallbacks(packages map[string]string, defaultPackage string, platforms []deployability.HostOS) map[deployability.HostOS]string {
	result := make(map[deployability.HostOS]string)
	for _, platform := range platforms {
		result[platform] = strings.TrimSpace(defaultPackage)
	}
	for manager, packageName := range packages {
		packageName = strings.TrimSpace(packageName)
		if packageName == "" {
			continue
		}
		var platform deployability.HostOS
		switch strings.ToLower(strings.TrimSpace(manager)) {
		case "brew":
			platform = deployability.HostOSMacOS
		case "winget", "choco", "scoop":
			platform = deployability.HostOSWindows
		default:
			platform = deployability.HostOSLinux
		}
		result[platform] = packageName
	}
	return result
}

func hasAcquisitionTargets(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	var acquisition struct {
		Targets []json.RawMessage `json:"targets"`
	}
	if err := json.Unmarshal(raw, &acquisition); err != nil {
		return false
	}
	return len(acquisition.Targets) > 0
}
