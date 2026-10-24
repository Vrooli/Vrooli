package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"react-component-library/internal/components"
)

// ValidateTokenFallbackLiteral rejects a literal CSS fallback in an active
// library source. A fallback is a silent design-system failure: it makes a
// missing declaration look intentional. The small exemption list is data in
// catalog/config.json because host/runtime contracts are the one case where a
// standalone asset is allowed to remain usable before its host supplies a
// value.
func ValidateTokenFallbackLiteral(scope Scope) (Result, error) {
	exemptions, err := components.ReadTokenFallbackExemptions(filepath.Join(scope.Root, "scenarios", "react-component-library", "catalog", "config.json"))
	if err != nil {
		return Result{}, err
	}
	sources, err := activeLibrarySources(scope)
	if err != nil {
		return Result{}, err
	}
	kept := sources[:0]
	for _, source := range sources {
		if isKeptLibrarySource(scope.Root, source) {
			kept = append(kept, source)
		}
	}
	sources = kept
	result := Result{Inspected: len(sources)}
	for _, source := range sources {
		data, readErr := os.ReadFile(source)
		if readErr != nil {
			return Result{}, readErr
		}
		for _, fallback := range parseTokenFallbacks(string(data)) {
			if _, exempt := exemptions[fallback.Property]; exempt || strings.HasPrefix(strings.TrimSpace(fallback.Value), "var(") {
				continue
			}
			result.Findings = append(result.Findings, Finding{
				Code:        "catalog.token-fallback-literal",
				AssetID:     implementationName(source),
				File:        repoRel(scope.Root, source),
				Line:        lineAt(data, fallback.Offset),
				Message:     fmt.Sprintf("%s carries a literal fallback %q", fallback.Property, fallback.Value),
				Remediation: fmt.Sprintf("Remove the fallback from var(%s). If this property is a documented host/runtime contract, add its exact name and reason to catalog/config.json x-token-fallback-exemptions.", fallback.Property),
				DocsRef:     "docs/reference/style-ownership.md",
			})
		}
	}
	return nonEmpty(result, "token-fallback-literal"), nil
}

func isKeptLibrarySource(root, source string) bool {
	clean := filepath.ToSlash(source)
	marker := "/scenarios/react-component-library/library/"
	index := strings.Index(clean, marker)
	if index < 0 {
		return false
	}
	parts := strings.Split(strings.Trim(clean[index+len(marker):], "/"), "/")
	if len(parts) < 5 || parts[2] != "versions" {
		return false
	}
	manifestPath := filepath.Join(root, "scenarios", "react-component-library", "library", parts[0], parts[1], "component.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return false
	}
	var manifest struct {
		Latest string `json:"latest"`
		Draft  string `json:"draft"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return false
	}
	return parts[3] == manifest.Latest || (manifest.Draft != "" && parts[3] == manifest.Draft)
}
