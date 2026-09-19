package validation_run

import (
	"path/filepath"
	"strings"

	manifest "development-toolchain-validator/internal/manifest"
)

// NormalizeDiffPaths strips generated physical roots or logical golden roots
// from agent-manager diff paths before manifest evaluation. Agent-manager
// normally reports paths relative to ProjectRoot; this also handles absolute
// paths defensively so generated temp dirs do not leak into manifests.
func NormalizeDiffPaths(files []manifest.DiffFile, g MaterializedGolden) []manifest.DiffFile {
	if len(files) == 0 {
		return nil
	}
	out := make([]manifest.DiffFile, 0, len(files))
	for _, f := range files {
		f.Path = normalizeDiffPath(f.Path, g)
		out = append(out, f)
	}
	return out
}

func normalizeDiffPath(path string, g MaterializedGolden) string {
	p := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	if p == "." {
		return ""
	}
	for _, root := range []string{g.PhysicalPath, g.LogicalRoot} {
		root = filepath.ToSlash(filepath.Clean(strings.TrimSpace(root)))
		if root == "." || root == "" {
			continue
		}
		if p == root {
			return ""
		}
		if strings.HasPrefix(p, root+"/") {
			return strings.TrimPrefix(p, root+"/")
		}
	}
	return strings.TrimPrefix(p, "./")
}
