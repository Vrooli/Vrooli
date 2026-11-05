package components

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CatalogKindDirectories reads the catalog vocabulary and returns the
// authored library roots that can contain registry manifests. The catalog's
// semantic kinds intentionally collapse onto the small set of filesystem
// roots used by the library (for example pattern and navigation assets are
// component assets, while runtime-service assets live under services).
func CatalogKindDirectories(root string) []string {
	var config struct {
		Gates []struct {
			AppliesTo []string `json:"appliesTo"`
		} `json:"gates"`
	}
	for _, candidate := range []string{
		filepath.Join(root, "catalog", "config.json"),
		filepath.Join(root, "..", "catalog", "config.json"),
		filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json"),
	} {
		data, err := os.ReadFile(filepath.Clean(candidate))
		if err != nil || json.Unmarshal(data, &config) != nil {
			continue
		}
		break
	}

	kinds := map[string]struct{}{}
	for _, gate := range config.Gates {
		for _, kind := range gate.AppliesTo {
			kinds[strings.TrimSpace(kind)] = struct{}{}
		}
	}
	if len(kinds) == 0 {
		// Isolated indexer fixtures do not carry the repository catalog config.
		// Keep their historical vocabulary while production always resolves the
		// list above from catalog/config.json.
		kinds = map[string]struct{}{
			"component": {}, "foundation": {}, "primitive": {}, "runtime-hook": {}, "runtime-service": {},
		}
	}

	directories := map[string]struct{}{}
	for kind := range kinds {
		directory := "components"
		switch kind {
		case "foundation":
			directory = "foundations"
		case "primitive":
			directory = "primitives"
		case "runtime-hook":
			directory = "hooks"
		case "runtime-service":
			directory = "services"
		}
		directories[directory] = struct{}{}
	}
	out := make([]string, 0, len(directories))
	for directory := range directories {
		out = append(out, directory)
	}
	sort.Strings(out)
	return out
}
