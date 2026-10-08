package checks

import (
	"path/filepath"
	"strings"

	"ui-health/internal/uiinterop"
)

// sourceFiles returns the shared RunAll source set constrained to subdir.
func sourceFiles(ctx uiinterop.CheckContext, subdir string) []uiinterop.SourceFile {
	if ctx.Sources == nil {
		return uiinterop.WalkUISource(ctx.ScenarioRoot, subdir)
	}
	prefix := filepath.ToSlash(filepath.Clean(filepath.FromSlash(subdir)))
	if prefix == "." {
		return ctx.Sources
	}
	exact := prefix
	prefix += "/"
	var files []uiinterop.SourceFile
	for _, f := range ctx.Sources {
		if f.RelPath == exact || strings.HasPrefix(f.RelPath, prefix) {
			files = append(files, f)
		}
	}
	return files
}
