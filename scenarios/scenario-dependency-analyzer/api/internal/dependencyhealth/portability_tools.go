package dependencyhealth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/packages/deployability"
)

type portabilityToolManifest struct {
	Name      string            `json:"name"`
	Platforms []string          `json:"platforms"`
	Packages  map[string]string `json:"packages"`
	Handler   string            `json:"handler"`
	Manual    bool              `json:"manual"`
	Source    json.RawMessage   `json:"source"`
}

// validateToolMacOSAcquisitions validates the repository-wide tool contract
// while the portability provider is running. This is deliberately manifest
// driven: acquisition policy must not grow a second list of tool names.
func validateToolMacOSAcquisitions(repoRoot string) error {
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
		declaration := deployability.ToolAcquisitionDeclaration{
			Platforms: platforms,
			Brew:      manifest.Packages["brew"],
			Source:    source,
			Handler:   strings.TrimSpace(manifest.Handler),
			Manual:    manifest.Manual,
		}
		if err := deployability.ValidateMacOSAcquisition(declaration); err != nil {
			return fmt.Errorf("tool %q: %w", manifest.Name, err)
		}
	}
	return nil
}
