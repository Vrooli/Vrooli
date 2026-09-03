package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"react-component-library/internal/librarywalk"
	"strings"
)

func ValidateManifestIdentity(scope Scope) (Result, error) {
	root := scope.Root
	result := Result{}
	catalog, err := loadAssets(scope)
	if err != nil {
		return Result{}, err
	}
	knownCatalogIDs := make(map[string]bool, len(catalog))
	for _, asset := range catalog {
		knownCatalogIDs[asset.Asset.ID] = true
	}
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		paths, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return Result{}, err
		}
		for _, manifest := range paths {
			if len(scope.Assets) > 0 && !scopeReportsAsset(scope, implementationName(manifest)) {
				continue
			}
			data, err := os.ReadFile(manifest)
			if err != nil {
				return Result{}, err
			}
			var doc struct {
				CatalogID string `json:"catalogId"`
				LibraryID string `json:"libraryId"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				return Result{}, err
			}
			result.Inspected++
			if doc.CatalogID == "" {
				result.Findings = append(result.Findings, Finding{Code: "catalog.manifest_identity", AssetID: doc.LibraryID, File: repoRel(root, manifest), Message: "manifest omits required catalogId", Remediation: "Add catalogId to the manifest and keep it stable for the asset's catalog projection.", DocsRef: "docs/concepts/ARCHITECTURE.md#catalog-graph-projection"})
				continue
			}
			if strings.HasPrefix(doc.CatalogID, "react-component-library:") && doc.CatalogID != doc.LibraryID {
				result.Findings = append(result.Findings, Finding{Code: "catalog.manifest_identity", AssetID: doc.CatalogID, File: repoRel(root, manifest), Message: fmt.Sprintf("catalogId %q does not equal libraryId %q", doc.CatalogID, doc.LibraryID), Remediation: "Use the manifest's libraryId as its library-prefixed catalogId.", DocsRef: "docs/concepts/ARCHITECTURE.md#catalog-graph-projection"})
				continue
			}
			if !knownCatalogIDs[doc.CatalogID] && doc.CatalogID != doc.LibraryID {
				result.Findings = append(result.Findings, Finding{Code: "catalog.manifest_identity", AssetID: doc.CatalogID, File: repoRel(root, manifest), Message: fmt.Sprintf("catalogId %q has no matching catalog projection or library identity", doc.CatalogID), Remediation: "Add the catalog projection or use the manifest's library-prefixed identity.", DocsRef: "docs/concepts/ARCHITECTURE.md#catalog-graph-projection"})
			}
		}
	}
	return nonEmpty(result, "manifest-identity"), nil
}

// ValidateManifestMetadata keeps authored assets discoverable and prevents
// transitional catalog escape hatches from becoming permanent public state.
