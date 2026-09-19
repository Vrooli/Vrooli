package deps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FSPackageJSONReader resolves a scenario package manifest with a traversal
// guard. Vrooli React scenarios conventionally keep frontend dependencies in
// ui/package.json, while a few scenarios retain package.json at the root, so
// both canonical locations are supported in that order.
type FSPackageJSONReader struct {
	Root string
}

// NewFSPackageJSONReader constructs the production reader. Root must
// be an absolute path on disk.
func NewFSPackageJSONReader(root string) *FSPackageJSONReader {
	return &FSPackageJSONReader{Root: root}
}

var _ PackageJSONReader = (*FSPackageJSONReader)(nil)

// templateScenarioPrefix is the scenario-key form adoption records use for
// vendored template copies: "../templates/scenarios/<id>", resolved relative
// to the scenarios root (matching adoptions.FSScenarioFileReader).
const templateScenarioPrefix = "../templates/scenarios/"

func (r *FSPackageJSONReader) Read(_ context.Context, scenario string) ([]byte, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, fmt.Errorf("scenario required")
	}
	root := r.Root
	name := scenario
	if tmpl, ok := strings.CutPrefix(scenario, templateScenarioPrefix); ok {
		root = filepath.Clean(filepath.Join(r.Root, "..", "templates", "scenarios"))
		name = tmpl
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return nil, fmt.Errorf("invalid scenario name %q", scenario)
	}
	base := filepath.Join(root, name)
	cleaned := filepath.Clean(base)
	rootClean := filepath.Clean(root) + string(filepath.Separator)
	if !strings.HasPrefix(cleaned, rootClean) {
		return nil, fmt.Errorf("resolved path escapes root")
	}
	for _, rel := range []string{"ui/package.json", "package.json"} {
		path := filepath.Join(cleaned, rel)
		bytes, err := os.ReadFile(path)
		if err == nil {
			return bytes, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("scenario %q package.json missing: ui/package.json or package.json", scenario)
}
