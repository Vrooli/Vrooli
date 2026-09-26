package themes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FSDesignMDReader resolves <scenariosRoot>/<scenario>/DESIGN.md with
// a traversal guard. Mirrors the deps/adoptions FS reader shape so
// production wires all three off the same scenariosRoot.
type FSDesignMDReader struct {
	Root string
}

func NewFSDesignMDReader(root string) *FSDesignMDReader {
	return &FSDesignMDReader{Root: root}
}

var _ DesignMDReader = (*FSDesignMDReader)(nil)

func (r *FSDesignMDReader) Read(_ context.Context, scenario string) ([]byte, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, fmt.Errorf("scenario required")
	}
	if strings.ContainsAny(scenario, "/\\") || strings.Contains(scenario, "..") {
		return nil, fmt.Errorf("invalid scenario name %q", scenario)
	}
	full := filepath.Join(r.Root, scenario, "DESIGN.md")
	cleaned := filepath.Clean(full)
	rootClean := filepath.Clean(r.Root) + string(filepath.Separator)
	if !strings.HasPrefix(cleaned, rootClean) {
		return nil, fmt.Errorf("resolved path escapes root")
	}
	return os.ReadFile(cleaned)
}
