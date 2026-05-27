package phases

import "fmt"

// docPathConvention returns the repo-relative documentation location for a
// phase. Every catalog phase has a docs/phases/<name>/README.md by convention,
// so adding a phase auto-derives its doc path with no separate mapping to keep
// in sync.
func docPathConvention(name Name) string {
	key := name.Key()
	if key == "" {
		return ""
	}
	return fmt.Sprintf("scenarios/test-genie/docs/phases/%s/README.md", key)
}

// DocPaths returns the repo-relative documentation paths for a phase. It returns
// nil for names that are not registered in the default catalog, so doc lookups
// stay in lockstep with the catalog rather than a hand-maintained map.
func DocPaths(raw string) []string {
	spec, ok := DefaultCatalog().Lookup(raw)
	if !ok || spec.Doc == "" {
		return nil
	}
	return []string{spec.Doc}
}
