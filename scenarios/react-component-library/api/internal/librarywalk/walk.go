// Package librarywalk owns filesystem traversal policy for the component
// library. Callers get one consistent exclusion policy, including quarantine.
package librarywalk

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Scope struct{ Assets map[string]struct{} }

func FullCorpus() Scope { return Scope{} }

// Kinds returns live asset-kind directories, including support. Generated and
// quarantined directories are excluded by the same traversal policy.
func Kinds(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	kinds := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != ".retired" && entry.Name() != "node_modules" && entry.Name() != "dist" {
			kinds = append(kinds, entry.Name())
		}
	}
	sort.Strings(kinds)
	return kinds, nil
}

// Sources returns authored library source files, applying the same exclusions
// and optional asset filter as Walk. Asset filters accept either the directory
// name or its lowercase dotted catalog form (for example Button or
// controls.button).
func Sources(root string, scope Scope) ([]string, error) {
	var paths []string
	err := Walk(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || (filepath.Ext(path) != ".ts" && filepath.Ext(path) != ".tsx" && filepath.Ext(path) != ".css") {
			return nil
		}
		if len(scope.Assets) > 0 {
			asset := filepath.Base(filepath.Dir(path))
			if filepath.Base(filepath.Dir(filepath.Dir(path))) == "versions" {
				asset = filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(path))))
			}
			matched := false
			for candidate := range scope.Assets {
				if candidate == asset || candidate == strings.ToLower(asset) || strings.HasSuffix(candidate, "."+strings.ToLower(asset)) {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		}
		paths = append(paths, path)
		return nil
	})
	return paths, err
}

// Walk traverses a library-related tree while excluding generated, dependency,
// and retired trees. The callback is intentionally compatible with WalkDir so
// existing gate logic remains small while traversal policy stays centralized.
func Walk(root string, fn func(string, os.DirEntry, error) error) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return fn(path, entry, err)
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".retired", "node_modules", "dist":
				if path != root {
					return filepath.SkipDir
				}
			}
		}
		return fn(path, entry, nil)
	})
}
