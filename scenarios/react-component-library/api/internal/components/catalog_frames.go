package components

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// CatalogFrameRegistryFromDir loads only the catalog fields needed by story
// frame validation. The catalog remains the source of truth; this is a
// read-only projection and never writes implementation status into it.
func CatalogFrameRegistryFromDir(catalogDir string) (CatalogFrameRegistry, error) {
	paths, err := filepath.Glob(filepath.Join(catalogDir, "assets", "*", "*.json"))
	if err != nil {
		return nil, fmt.Errorf("scan catalog assets: %w", err)
	}
	sort.Strings(paths)
	registry := catalogFrameRegistry{assets: make(map[string]CatalogFrameAsset)}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read catalog asset %s: %w", path, err)
		}
		var doc rawCatalogFrameAsset
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("parse catalog asset %s: %w", path, err)
		}
		if doc.Kind != "catalog-asset" || doc.Asset.ID == "" {
			continue
		}
		asset := CatalogFrameAsset{ID: doc.Asset.ID, Kind: doc.Asset.Kind, Targets: doc.Asset.Targets}
		for _, region := range doc.Regions {
			asset.Regions = append(asset.Regions, region.ID)
		}
		for _, expect := range doc.Expects {
			asset.Expects = append(asset.Expects, CatalogFramePort{Capability: expect.Capability, TypeArguments: append([]string(nil), expect.TypeArguments...)})
		}
		if doc.Fixture.Satisfies != nil {
			asset.FixtureSatisfies = &CatalogFramePort{Capability: doc.Fixture.Satisfies.Capability, TypeArguments: append([]string(nil), doc.Fixture.Satisfies.TypeArguments...)}
		}
		registry.assets[asset.ID] = asset
	}
	return registry, nil
}

type catalogFrameRegistry struct {
	assets map[string]CatalogFrameAsset
}

func (r catalogFrameRegistry) LookupCatalogFrameAsset(id string) (CatalogFrameAsset, bool) {
	asset, ok := r.assets[id]
	return asset, ok
}

var _ CatalogFrameRegistry = catalogFrameRegistry{}

type rawCatalogFrameAsset struct {
	Kind  string `json:"kind"`
	Asset struct {
		ID      string   `json:"id"`
		Kind    string   `json:"kind"`
		Targets []string `json:"targets"`
	} `json:"asset"`
	Expects []struct {
		Capability    string   `json:"capability"`
		TypeArguments []string `json:"typeArguments"`
	} `json:"expects"`
	Regions []struct {
		ID string `json:"id"`
	} `json:"regions"`
	Fixture struct {
		Satisfies *struct {
			Capability    string   `json:"capability"`
			TypeArguments []string `json:"typeArguments"`
		} `json:"satisfies"`
	} `json:"fixture"`
}
