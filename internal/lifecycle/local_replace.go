package lifecycle

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
)

// localReplaceDirs returns local directory replacements from a Go module.
// modfile is the sole Go-module syntax parser used by lifecycle freshness;
// callers receive paths relative to the module containing goModPath.
func localReplaceDirs(goModPath string) ([]string, error) {
	goModPath = filepath.Clean(goModPath)
	data, err := os.ReadFile(goModPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	parsed, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return nil, err
	}

	base := filepath.Dir(goModPath)
	seen := make(map[string]struct{})
	paths := make([]string, 0, len(parsed.Replace))
	for _, replacement := range parsed.Replace {
		path := strings.TrimSpace(replacement.New.Path)
		if path == "" || (!filepath.IsAbs(path) && !strings.HasPrefix(path, ".")) {
			continue
		}
		absolute, err := filepath.Abs(filepath.Join(base, filepath.FromSlash(path)))
		if err != nil {
			return nil, err
		}
		relative, err := filepath.Rel(base, absolute)
		if err != nil {
			return nil, err
		}
		relative = filepath.Clean(relative)
		if _, ok := seen[relative]; ok {
			continue
		}
		seen[relative] = struct{}{}
		paths = append(paths, relative)
	}
	sort.Strings(paths)
	return paths, nil
}
