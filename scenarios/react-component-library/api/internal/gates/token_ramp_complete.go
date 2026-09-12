package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ValidateTokenRampComplete(scope Scope) (Result, error) {
	root := scope.Root
	sources, err := activeLibrarySources(scope)
	if err != nil {
		return Result{}, err
	}
	filteredSources := sources[:0]
	for _, path := range sources {
		if !isSupplementalImplementation(path) {
			filteredSources = append(filteredSources, path)
		}
	}
	sources = filteredSources
	rampRaw, err := os.ReadFile(filepath.Join(root, "scenarios", "react-component-library", "ui", "src", "design-tokens.css"))
	if err != nil {
		return Result{}, err
	}
	ramp := map[string]struct{}{}
	for _, match := range cssVarDeclGateRE.FindAllStringSubmatch(string(rampRaw), -1) {
		ramp[match[1]] = struct{}{}
	}
	baseProperties, err := activeAssetDeclarations(filepath.Join(root, "scenarios", "react-component-library", "library", "foundations", "BaseStyles"))
	if err != nil {
		return Result{}, fmt.Errorf("read shared BaseStyles vocabulary: %w", err)
	}
	for property := range baseProperties {
		ramp[property] = struct{}{}
	}
	result := Result{Inspected: len(sources)}
	for _, path := range sources {
		raw, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		text := string(raw)
		declared := map[string]struct{}{}
		for _, match := range cssVarDeclGateRE.FindAllStringSubmatch(text, -1) {
			declared[match[1]] = struct{}{}
		}
		for _, match := range cssVarRefGateRE.FindAllStringSubmatch(text, -1) {
			property := match[1]
			if _, local := declared[property]; local || strings.HasPrefix(property, "--rcl-") || strings.HasSuffix(property, "-") {
				continue
			}
			if _, published := ramp[property]; !published {
				result.Findings = append(result.Findings, Finding{
					Code:        "catalog.token_ramp_complete",
					AssetID:     implementationName(path),
					File:        repoRel(root, path),
					Line:        lineOf(raw, match[0]),
					Message:     fmt.Sprintf("consumes %s, which the canonical ramp does not publish", property),
					Remediation: fmt.Sprintf("Either publish %s in ui/src/design-tokens.css so every adopting scenario inherits it, or switch this reference to a property the ramp already declares. An unpublished property resolves to nothing in a consumer that has not copied this file, so the asset silently loses the styling this reference was meant to apply.", property),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
				})
			}
		}
	}
	return nonEmpty(result, "token-ramp-complete"), nil
}

// ValidateReleasedVersionImmutable compares every indexed released version
// with its current on-disk entry and companion files. It is intentionally a
// corpus gate, independent of the indexer's write path, so direct filesystem
// edits remain observable.
