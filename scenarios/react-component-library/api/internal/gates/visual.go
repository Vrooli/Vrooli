package gates

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"react-component-library/internal/librarywalk"
)

// ValidateVisual checks the durable image produced by the preview capture
// path. Browser assertions remain owned by the experience suite, while this
// gate makes the stored artifact requirement measurable for every renderable
// catalog asset.
func ValidateVisual(scope Scope) (Result, error) {
	assets, err := loadAssets(scope)
	if err != nil {
		return Result{}, err
	}
	manifestPaths, err := librarywalk.Glob(filepath.Join(scope.Root, "scenarios", "react-component-library", "library", "*", "*", "component.json"))
	if err != nil {
		return Result{}, err
	}
	manifests := make(map[string]struct {
		path   string
		latest string
	}, len(manifestPaths))
	for _, candidate := range manifestPaths {
		data, readErr := os.ReadFile(candidate)
		if readErr != nil {
			return Result{}, readErr
		}
		var manifest struct {
			CatalogID string `json:"catalogId"`
			Latest    string `json:"latest"`
		}
		if jsonErr := json.Unmarshal(data, &manifest); jsonErr != nil {
			return Result{}, jsonErr
		}
		if manifest.CatalogID != "" {
			manifests[manifest.CatalogID] = struct {
				path   string
				latest string
			}{path: candidate, latest: manifest.Latest}
		}
	}
	result := Result{}
	for _, asset := range assets {
		if !renderableAssetKind(asset.Asset.Kind) {
			continue
		}
		manifest, ok := manifests[asset.Asset.ID]
		// Catalog entries adopted from another scenario can be renderable in
		// the catalog while having no local released source directory. There is
		// no version-local preview artifact to validate for those entries; the
		// materialized local release set is the visual gate's evidence boundary.
		if !ok {
			continue
		}
		result.Inspected++
		if strings.TrimSpace(manifest.latest) == "" {
			result.Findings = append(result.Findings, Finding{Code: "catalog.visual_preview_missing", AssetID: asset.Asset.ID, Message: "renderable asset has no latest materialized version", Remediation: "publish or materialize the latest version, then run react-component-library preview capture --asset <library-id> --version <version>"})
			continue
		}
		versionDir := filepath.Join(filepath.Dir(manifest.path), "versions", manifest.latest)
		previewPath := filepath.Join(versionDir, "preview.png")
		file, openErr := os.Open(previewPath)
		if os.IsNotExist(openErr) {
			result.Findings = append(result.Findings, Finding{Code: "catalog.visual_preview_missing", AssetID: asset.Asset.ID, File: repoRel(scope.Root, previewPath), Message: fmt.Sprintf("latest version %s has no preview.png", manifest.latest), Remediation: "Run react-component-library preview capture --asset <library-id> --version <version> and inspect the stored image."})
			continue
		}
		if openErr != nil {
			return Result{}, openErr
		}
		_, _, decodeErr := image.DecodeConfig(file)
		_ = file.Close()
		if decodeErr != nil {
			result.Findings = append(result.Findings, Finding{Code: "catalog.visual_preview_invalid", AssetID: asset.Asset.ID, File: repoRel(scope.Root, previewPath), Message: decodeErr.Error(), Remediation: "Replace preview.png with a valid PNG produced by the governed preview capture command."})
		}
	}
	return nonEmpty(result, "visual"), nil
}

func renderableAssetKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "primitive", "component", "pattern", "page-template", "navigation":
		return true
	default:
		return false
	}
}
