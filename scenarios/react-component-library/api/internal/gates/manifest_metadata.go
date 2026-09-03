package gates

import (
	"encoding/json"
	"os"
	"path/filepath"
	"react-component-library/internal/librarywalk"
	"strings"
)

func ValidateManifestMetadata(scope Scope) (Result, error) {
	root := scope.Root
	result := Result{}
	catalog, err := loadAssets(scope)
	if err != nil {
		return Result{}, err
	}
	catalogDescriptions := make(map[string]string, len(catalog))
	catalogIDs := make(map[string]bool, len(catalog))
	for _, asset := range catalog {
		catalogIDs[asset.Asset.ID] = true
		catalogDescriptions[asset.Asset.ID] = strings.TrimSpace(asset.Asset.Description)
	}
	for _, kind := range []string{"components"} {
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
				LibraryID                 string `json:"libraryId"`
				CatalogID                 string `json:"catalogId"`
				SupplementalJustification string `json:"x-supplementalJustification"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				return Result{}, err
			}
			result.Inspected++
			assetID := strings.TrimSpace(doc.CatalogID)
			if assetID == "" {
				assetID = doc.LibraryID
			}
			switch {
			case strings.TrimSpace(doc.SupplementalJustification) != "":
				result.Findings = append(result.Findings, Finding{Code: "catalog.manifest_metadata", AssetID: assetID, File: repoRel(root, manifest), Message: "manifest carries x-supplementalJustification", Remediation: "Register the asset against its catalog projection or remove the transitional justification.", DocsRef: "docs/reference/overlay-selection.md"})
			case catalogIDs[assetID] && strings.TrimSpace(catalogDescriptions[assetID]) == "":
				result.Findings = append(result.Findings, Finding{Code: "catalog.manifest_metadata", AssetID: assetID, File: repoRel(root, manifest), Message: "catalog description is empty or catalog projection is missing", Remediation: "Add the asset's user-visible responsibility to its catalog declaration before indexing the library projection.", DocsRef: "docs/concepts/ARCHITECTURE.md#catalog-graph-projection"})
			case strings.HasPrefix(strings.TrimSpace(doc.CatalogID), "react-component-library:") && !catalogIDs[assetID]:
				result.Findings = append(result.Findings, Finding{Code: "catalog.manifest_metadata", AssetID: doc.CatalogID, File: repoRel(root, manifest), Message: "manifest uses a self-referential catalogId", Remediation: "Use the stable domain catalog id, or clear catalogId when no projection exists.", DocsRef: "docs/concepts/ARCHITECTURE.md#catalog-graph-projection"})
			}
		}
	}
	return nonEmpty(result, "manifest-metadata"), nil
}

// ValidateOverlaySurfaceComposition keeps modal and menu behavior on the
// shared overlay substrate. An opt-out is permitted only when the manifest
// carries a non-empty reason, making the exception reviewable.
