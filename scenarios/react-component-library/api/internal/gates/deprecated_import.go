package gates

import (
	"fmt"
	"os"

	"github.com/vrooli/vrooli/packages/react-component-library/libspec"
)

func ValidateDeprecatedImports(scope Scope) (Result, error) {
	root := scope.Root
	deprecated, err := deprecatedLibraryVersions(root)
	if err != nil {
		return Result{}, err
	}
	sources, err := activeLibrarySources(scope)
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: len(sources)}
	for _, path := range sources {
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		for _, specifier := range libspec.ParseAll(string(data)) {
			if specifier.Selector == "" || !contains(deprecated[specifier.Name], specifier.Selector) {
				continue
			}
			importPath := libspec.Prefix + specifier.Name + "/" + specifier.Selector
			result.Findings = append(result.Findings, Finding{Code: "catalog.deprecated-import", AssetID: implementationName(path), File: repoRel(root, path), Line: lineOf(data, importPath), Message: fmt.Sprintf("imports deprecated %s@%s", specifier.Name, specifier.Selector), Remediation: fmt.Sprintf("Import %s at its non-deprecated published version instead of pinning %s.", specifier.Name, importPath), DocsRef: "docs/concepts/ARCHITECTURE.md#version-lifecycle"})
		}
	}
	return nonEmpty(result, "deprecated-import"), nil
}

type consumerPin struct {
	Asset     string
	Version   string
	Scenarios map[string]bool
	Files     []string
}

type consumerPinManifest struct {
	CatalogID          string   `json:"catalogId"`
	LibraryID          string   `json:"libraryId"`
	Latest             string   `json:"latest"`
	DeprecatedVersions []string `json:"deprecatedVersions"`
	Root               string
}

// ValidateConsumerPins inspects the exact asset-version surface imported by
// scenarios and groups each defect with every affected consumer.
