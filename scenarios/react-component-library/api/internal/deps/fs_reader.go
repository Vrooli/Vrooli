package deps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FSPackageJSONReader resolves <scenariosRoot>/<scenario>/package.json
// with a traversal guard. Mirrors the adoptions FSScenarioFileReader
// shape so production wires both off the same scenariosRoot.
type FSPackageJSONReader struct {
	Root string
}

// NewFSPackageJSONReader constructs the production reader. Root must
// be an absolute path on disk.
func NewFSPackageJSONReader(root string) *FSPackageJSONReader {
	return &FSPackageJSONReader{Root: root}
}

var _ PackageJSONReader = (*FSPackageJSONReader)(nil)

func (r *FSPackageJSONReader) Read(_ context.Context, scenario string) ([]byte, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, fmt.Errorf("scenario required")
	}
	if strings.ContainsAny(scenario, "/\\") || strings.Contains(scenario, "..") {
		return nil, fmt.Errorf("invalid scenario name %q", scenario)
	}
	full := filepath.Join(r.Root, scenario, "package.json")
	cleaned := filepath.Clean(full)
	rootClean := filepath.Clean(r.Root) + string(filepath.Separator)
	if !strings.HasPrefix(cleaned, rootClean) {
		return nil, fmt.Errorf("resolved path escapes root")
	}
	return os.ReadFile(cleaned)
}
