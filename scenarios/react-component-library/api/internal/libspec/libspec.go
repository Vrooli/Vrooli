// Package libspec owns the grammar and traversal for React Component Library
// source references. Callers must use this package instead of inventing a
// package-specific regular expression or filesystem walk.
package libspec

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	sharedlibspec "github.com/vrooli/vrooli/packages/react-component-library/libspec"
	"react-component-library/internal/librarywalk"
)

const Prefix = sharedlibspec.Prefix

type (
	Specifier = sharedlibspec.Specifier
	Scope     struct{ Assets map[string]bool }
)

// Rewrite visits each canonical library specifier in source and replaces it
// with the callback result. Parsing remains owned by this package; callers
// provide only the policy-specific replacement.
func Rewrite(source string, replace func(Specifier) string) string {
	return sharedlibspec.Rewrite(source, replace)
}

func Parse(value string) (Specifier, bool, error) {
	return sharedlibspec.Parse(value)
}

func ParseAll(source string) []Specifier {
	return sharedlibspec.ParseAll(source)
}

func IsRelease(value string) bool { return sharedlibspec.IsRelease(value) }

func Walk(root string, scope Scope, visit func(path string) error) error {
	return librarywalk.WalkTree(context.TODO(), root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == "node_modules" || entry.Name() == "dist" || entry.Name() == ".retired") {
				return filepath.SkipDir
			}
			if path != root && len(scope.Assets) > 0 && filepath.Dir(path) == root && !scope.Assets[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".ts") || strings.HasSuffix(entry.Name(), ".tsx") {
			return visit(path)
		}
		return nil
	})
}

func Sorted(values []Specifier) []Specifier {
	result := append([]Specifier(nil), values...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].Selector < result[j].Selector
		}
		return result[i].Name < result[j].Name
	})
	return result
}

func Validate(value string) error {
	return sharedlibspec.Validate(value)
}
