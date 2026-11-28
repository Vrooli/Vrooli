// Package libspec owns the grammar for published React Component Library
// specifiers. It is shared by the API and CLI modules.
package libspec

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const Prefix = "@vrooli/react-component-library/"

var (
	grammar = regexp.MustCompile(`^@vrooli/react-component-library/([A-Za-z][A-Za-z0-9-]*)(?:/(\d+|\d+\.\d+\.\d+))?$`)
	release = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
)

type Specifier struct{ Name, Selector string }

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
		if specifier, ok, _ := Parse(token); ok && !seen[specifier] {
			seen[specifier] = true
			result = append(result, specifier)
		}
	}
	return result
}

func Rewrite(source string, replace func(Specifier) string) string {
	result := source
	for _, specifier := range ParseAll(source) {
		canonical := Prefix + specifier.Name
		if specifier.Selector != "" {
			canonical += "/" + specifier.Selector
		}
		if replacement := replace(specifier); replacement != "" {
			result = strings.ReplaceAll(result, canonical, replacement)
		}
	}
	return result
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

func IsRelease(value string) bool { return release.MatchString(value) }

func Validate(value string) error {
	if _, ok, _ := Parse(value); !ok {
		return fmt.Errorf("invalid library specifier %q", value)
	}
	return nil
}
