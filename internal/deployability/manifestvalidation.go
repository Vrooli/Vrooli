package deployability

import (
	"fmt"
	"sort"
	"strings"
)

// ManifestDeclaration is the loader-neutral shape of one capability-bearing
// manifest. Path is the file the declaration was read from so a rejection can
// name it; the resolver package never globs for manifests itself.
type ManifestDeclaration struct {
	Path       string
	Name       string
	Capability string
	Role       string
	// Platforms is keyed by the raw OS token the manifest authored, so a
	// rejection can quote what was written rather than a normalized guess.
	Platforms map[string]PlatformDeclaration
}

// ValidateManifestDeclarations rejects any manifest that names a capability
// outside the vocabulary, carries an unknown capability role, or authors a
// platform status outside the platform status vocabulary. Every rejection
// names the offending file and token.
func ValidateManifestDeclarations(declarations []ManifestDeclaration, vocabulary []string) error {
	allowed := make(map[string]struct{}, len(vocabulary))
	for _, capability := range vocabulary {
		capability = strings.TrimSpace(capability)
		if capability == "" {
			return fmt.Errorf("capability vocabulary contains an empty entry")
		}
		if _, duplicate := allowed[capability]; duplicate {
			return fmt.Errorf("capability vocabulary duplicates %q", capability)
		}
		allowed[capability] = struct{}{}
	}
	for _, item := range declarations {
		location := declarationLocation(item)
		capability := strings.TrimSpace(item.Capability)
		if capability == "" {
			return fmt.Errorf("%s: capability manifest %q has no capability", location, item.Name)
		}
		if _, ok := allowed[capability]; !ok {
			return fmt.Errorf("%s: capability manifest %q declares unknown capability %q", location, item.Name, capability)
		}
		switch strings.TrimSpace(item.Role) {
		case "primary", "peer", "control":
		default:
			return fmt.Errorf("%s: capability manifest %q has invalid capability_role %q", location, item.Name, item.Role)
		}
		for _, osName := range sortedKeys(item.Platforms) {
			if _, err := ParsePlatformStatus(item.Platforms[osName].Status); err != nil {
				return fmt.Errorf("%s: capability manifest %q declares %s %w", location, item.Name, osName, err)
			}
		}
	}
	return nil
}

func declarationLocation(item ManifestDeclaration) string {
	if path := strings.TrimSpace(item.Path); path != "" {
		return path
	}
	return "<unknown manifest path>"
}

func sortedKeys(platforms map[string]PlatformDeclaration) []string {
	keys := make([]string, 0, len(platforms))
	for key := range platforms {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
