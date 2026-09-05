package components

import (
	"os"
	"path/filepath"
	"strings"
)

// experienceContractPath makes the scenario-level experience declaration the
// authority. The version-local fallback exists only for isolated legacy test
// fixtures and disappears naturally once those fixtures are migrated.
func experienceContractPath(sourceRoot, slug, assetRoot, version string) string {
	repoRoot := repositoryRootFromLibrary(sourceRoot)
	if repoRoot != "" {
		canonical := filepath.Join(repoRoot, "scenarios", "react-component-library", "experience", "components", kebabAssetName(slug)+".json")
		if _, err := os.Stat(canonical); err == nil {
			return canonical
		}
	}
	return filepath.Join(sourceRoot, assetRoot, slug, "versions", version, "experience-contract.json")
}

func kebabAssetName(value string) string {
	value = strings.TrimSpace(value)
	var out []rune
	for i, r := range value {
		if r >= 'A' && r <= 'Z' && i > 0 {
			out = append(out, '-')
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			out = append(out, []rune(strings.ToLower(string(r)))...)
		}
	}
	return string(out)
}
