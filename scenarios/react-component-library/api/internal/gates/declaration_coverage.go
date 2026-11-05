package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"react-component-library/internal/librarywalk"
)

// ValidateDeclarationCoverage ensures every implementation visible to the
// library walker has a catalog declaration. Missing declarations are reported
// in the corpus bucket because they have no catalog identity to annotate.
func ValidateDeclarationCoverage(scope Scope) (Result, error) {
	root := scope.Root
	known := catalogAssetIDs(root)
	selected := make(map[string]bool, len(scope.Assets))
	for _, id := range scope.Assets {
		selected[id] = true
	}
	result := Result{}
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		manifests, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return Result{}, err
		}
		sort.Strings(manifests)
		for _, manifest := range manifests {
			data, err := os.ReadFile(manifest)
			if err != nil {
				return Result{}, err
			}
			var doc struct {
				CatalogID string `json:"catalogId"`
				LibraryID string `json:"libraryId"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				return Result{}, fmt.Errorf("parse %s: %w", manifest, err)
			}
			id := strings.TrimSpace(doc.CatalogID)
			if !scope.IsFullCorpus() && !selected[id] && !selected[doc.LibraryID] {
				continue
			}
			result.Inspected++
			if known[id] {
				continue
			}
			reportedID := id
			if reportedID == "" {
				reportedID = strings.TrimSpace(doc.LibraryID)
			}
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.declaration_coverage", AssetID: "__corpus__.declaration-coverage",
				File: repoRel(root, manifest), Message: fmt.Sprintf("implemented asset %q has no catalog declaration", reportedID),
				Remediation: "Author the catalog declaration before publishing or indexing this implementation.", DocsRef: "docs/concepts/ASSET-DERIVATION.md",
			})
		}
	}
	return nonEmpty(result, "declaration-coverage"), nil
}
