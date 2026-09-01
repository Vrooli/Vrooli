package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"react-component-library/internal/themes"
)

func ValidateFallbackParity(scope Scope) (Result, error) {
	root := scope.Root
	tokens, err := themes.ReadTokenFile(filepath.Join(root, "templates", "design", "_base", "tokens.css"))
	if err != nil {
		return Result{}, fmt.Errorf("read canonical token vocabulary: %w", err)
	}
	canonical := make(map[string]string, len(tokens))
	for _, token := range tokens {
		canonical[token.Name] = normalizeTokenValue(token.Value)
	}
	sources, err := activeLibrarySources(scope)
	if err != nil {
		return Result{}, err
	}
	byVersion := map[string][]string{}
	for _, source := range sources {
		if !strings.Contains(filepath.ToSlash(source), "/library/components/") {
			continue
		}
		byVersion[filepath.Dir(source)] = append(byVersion[filepath.Dir(source)], source)
	}
	result := Result{}
	for _, versionSources := range byVersion {
		declared := map[string]bool{}
		contents := map[string][]byte{}
		for _, source := range versionSources {
			data, readErr := os.ReadFile(source)
			if readErr != nil {
				return Result{}, readErr
			}
			contents[source] = data
			for _, match := range regexp.MustCompile(`(?m)(--[A-Za-z0-9_-]+)\s*:`).FindAllSubmatch(data, -1) {
				declared[string(match[1])] = true
			}
		}
		for _, source := range versionSources {
			data := contents[source]
			result.Inspected++
			for _, fallback := range parseTokenFallbacks(string(data)) {
				want, known := canonical[fallback.Property]
				if !known || declared[fallback.Property] || strings.HasPrefix(fallback.Property, "--rcl-") {
					continue
				}
				if normalizeTokenValue(fallback.Value) == want {
					continue
				}
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.fallback_parity", File: repoRel(root, source), Line: lineAt(data, fallback.Offset),
					Message:     fmt.Sprintf("%s fallback %q disagrees with canonical value %q", fallback.Property, fallback.Value, tokenValue(tokens, fallback.Property)),
					Remediation: fmt.Sprintf("Use var(%s, %s), or declare the property inside this component version when it is genuinely component-owned.", fallback.Property, tokenValue(tokens, fallback.Property)),
					DocsRef:     "docs/design/TOKEN-DICTIONARY.md",
				})
			}
		}
	}
	return result, nil
}

// ValidateKitCompatibility blocks versions whose token contract cannot be
// satisfied by the registered kit vocabulary.
