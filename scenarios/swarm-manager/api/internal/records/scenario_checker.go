package records

import (
	"os"
	"strings"
)

// NewDirectoryScenarioChecker builds a ScenarioChecker that treats the
// immediate subdirectories of the given roots (typically <repo>/scenarios,
// <repo>/packages, <repo>/resources), plus any extra slugs (e.g. "vrooli" for
// repo-level work), as known. Roots are re-scanned on every call so newly
// created scenarios are picked up without a restart; the fleet is small
// enough that three ReadDirs per record creation are negligible.
func NewDirectoryScenarioChecker(roots []string, extra ...string) ScenarioChecker {
	return func(slug string) (bool, string) {
		slug = strings.ToLower(strings.TrimSpace(slug))
		if slug == "" {
			return true, "" // emptiness is the service's error to raise, not a warning
		}
		known := make(map[string]struct{})
		for _, e := range extra {
			known[strings.ToLower(strings.TrimSpace(e))] = struct{}{}
		}
		for _, root := range roots {
			entries, err := os.ReadDir(root)
			if err != nil {
				continue // a missing root disables that source, never blocks
			}
			for _, ent := range entries {
				if ent.IsDir() && !strings.HasPrefix(ent.Name(), ".") {
					known[strings.ToLower(ent.Name())] = struct{}{}
				}
			}
		}
		if _, ok := known[slug]; ok {
			return true, ""
		}
		best, bestDist := "", -1
		for k := range known {
			d := levenshtein(slug, k)
			if bestDist == -1 || d < bestDist || (d == bestDist && k < best) {
				best, bestDist = k, d
			}
		}
		if bestDist == -1 || bestDist > 2 {
			return false, ""
		}
		return false, best
	}
}
