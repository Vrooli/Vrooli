// Package libspec owns the grammar and traversal for React Component Library
// source references. Callers must use this package instead of inventing a
// package-specific regular expression or filesystem walk.
package libspec

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const Prefix = "@vrooli/react-component-library/"

var (
	grammar = regexp.MustCompile(`^@vrooli/react-component-library/([A-Za-z][A-Za-z0-9-]*)(?:/(\d+|\d+\.\d+\.\d+))?$`)
	release = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
)

type (
	Specifier struct{ Name, Selector string }
	Scope     struct{ Assets map[string]bool }
)

func Parse(value string) (Specifier, bool, error) {
	match := grammar.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return Specifier{}, false, nil
	}
	return Specifier{Name: match[1], Selector: match[2]}, true, nil
}

func ParseAll(source string) []Specifier {
	result := []Specifier{}
	seen := map[Specifier]bool{}
	for _, token := range strings.FieldsFunc(source, func(r rune) bool { return strings.ContainsRune("\"'` \t\r\n,;(){}", r) }) {
		if spec, ok, _ := Parse(token); ok && !seen[spec] {
			seen[spec] = true
			result = append(result, spec)
		}
	}
	return result
}

func IsRelease(value string) bool { return release.MatchString(value) }

func Walk(root string, scope Scope, visit func(path string) error) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
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
	if _, ok, _ := Parse(value); !ok {
		return fmt.Errorf("invalid library specifier %q", value)
	}
	return nil
}
