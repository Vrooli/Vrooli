// Package retirement coordinates withdrawal above the catalog and component
// packages so neither domain needs a circular dependency on the other.
package retirement

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"react-component-library/internal/assetgraph"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
	"react-component-library/internal/gates"
)

type PreflightResult struct {
	Completed        bool
	CatalogID        string
	CatalogPath      string
	RequiredBy       []string
	SuggestedBy      []string
	SourceReferences []string
}

func (r PreflightResult) Ready() bool {
	return r.Completed && len(r.RequiredBy) == 0 && len(r.SuggestedBy) == 0 && len(r.SourceReferences) == 0
}

func Preflight(ctx context.Context, root string, component components.Component) (PreflightResult, error) {
	assetRoot, err := verifiedAssetRoot(root, component)
	if err != nil {
		return PreflightResult{}, err
	}
	assets, err := catalogcoverage.LoadCatalogContext(ctx, filepath.Join(root, "scenarios/react-component-library/catalog"))
	if err != nil {
		return PreflightResult{}, err
	}
	result, err := checkCatalog(assets, component.CatalogID)
	if err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	result.SourceReferences, err = gates.RetirementSourceReferences(root, component.Slug, assetRoot)
	if err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	result.Completed = true
	return result, nil
}

func verifiedAssetRoot(root string, c components.Component) (string, error) {
	parts := strings.Split(c.ManifestPath, "/")
	if len(parts) != 3 || parts[2] != "component.json" || parts[1] != c.Slug || c.Slug == "" ||
		strings.HasPrefix(parts[0], ".") || strings.Contains(c.ManifestPath, "\\") ||
		filepath.ToSlash(filepath.Clean(c.ManifestPath)) != c.ManifestPath || c.LibraryID == "" || c.CatalogID == "" {
		return "", fmt.Errorf("retirement requires a canonical manifest path and matching asset identity")
	}
	library := filepath.Join(root, "scenarios/react-component-library/library")
	manifestPath := filepath.Join(library, c.ManifestPath)
	for _, path := range []string{library, filepath.Join(library, parts[0]), filepath.Dir(manifestPath), manifestPath} {
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("retirement refuses symlink path: %s", path)
		}
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", err
	}
	var manifest struct {
		LibraryID string `json:"libraryId"`
		CatalogID string `json:"catalogId"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", err
	}
	if manifest.LibraryID != c.LibraryID || manifest.CatalogID != c.CatalogID {
		return "", fmt.Errorf("retirement manifest identity does not match indexed asset")
	}
	return filepath.Dir(manifestPath), nil
}

func checkCatalog(assets []catalogcoverage.Asset, id string) (PreflightResult, error) {
	result := PreflightResult{CatalogID: id}
	index, err := assetgraph.Build(assets)
	if err != nil {
		return result, err
	}
	direct, _, err := index.Dependents(id)
	if err != nil {
		return result, err
	}
	for _, asset := range assets {
		if asset.ID == id {
			result.CatalogPath = asset.DeclarationPath
			if asset.Maturity != "deprecated" {
				return result, fmt.Errorf("catalog asset %s must be deprecated before retirement", id)
			}
		}
		for _, suggested := range asset.Suggests {
			if suggested == id {
				result.SuggestedBy = append(result.SuggestedBy, asset.ID)
				break
			}
		}
	}
	for _, node := range direct {
		result.RequiredBy = append(result.RequiredBy, node.ID)
	}
	sort.Strings(result.RequiredBy)
	sort.Strings(result.SuggestedBy)
	return result, nil
}
