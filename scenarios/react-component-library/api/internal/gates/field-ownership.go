package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ValidateFieldOwnership(scope Scope) (Result, error) {
	root := scope.Root
	selected := map[string]bool{}
	for _, id := range scope.Assets {
		selected[id] = true
	}
	var paths []string
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		matches, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return Result{}, err
		}
		paths = append(paths, matches...)
	}
	sort.Strings(paths)
	result := Result{}
	owned := map[string]bool{"description": true, "displayName": true, "kind": true, "slot": true, "tags": true, "category": true, "designStyles": true, "requires": true, "entry": true}
	for _, manifest := range paths {
		data, err := os.ReadFile(manifest)
		if err != nil {
			return Result{}, err
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			return Result{}, fmt.Errorf("parse %s: %w", manifest, err)
		}
		var identity struct {
			CatalogID string `json:"catalogId"`
			LibraryID string `json:"libraryId"`
		}
		_ = json.Unmarshal(data, &identity)
		assetID := identity.CatalogID
		if assetID == "" {
			assetID = identity.LibraryID
		}
		if len(selected) > 0 && !selected[assetID] {
			continue
		}
		result.Inspected++
		for field := range owned {
			if _, present := raw[field]; present {
				result.Findings = append(result.Findings, Finding{Code: "catalog.field_ownership", AssetID: assetID, File: repoRel(root, manifest), Message: fmt.Sprintf("manifest carries catalog-owned field %q", field), Remediation: fmt.Sprintf("Remove %q from component.json and derive it from the catalog declaration.", field), DocsRef: "docs/concepts/ASSET-DERIVATION.md"})
			}
		}
		versionRoot := filepath.Join(filepath.Dir(manifest), "versions")
		if _, statErr := os.Stat(versionRoot); os.IsNotExist(statErr) {
			continue
		}
		_ = filepath.WalkDir(versionRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || (!strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx")) {
				return walkErr
			}
			body, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if ownedHeaderRE.Match(body) {
				result.Findings = append(result.Findings, Finding{Code: "catalog.field_ownership", AssetID: assetID, File: repoRel(root, path), Line: lineOf(body, "@description"), Message: "source header carries catalog-owned field \"description\"", Remediation: "Remove @description from the source header; keep the description in the catalog declaration.", DocsRef: "docs/concepts/ASSET-DERIVATION.md"})
			}
			return nil
		})
	}
	return nonEmpty(result, "field-ownership"), nil
}
