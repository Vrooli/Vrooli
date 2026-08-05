// Package catalogcoverage joins the desired-state asset catalog against the
// implementations on disk and reports what exists, what is planned, and what
// exists without ever having been planned.
//
// The join is deliberately file-based rather than going through the registry
// database. Coverage must be answerable about assets that have no implementation
// at all — which is most of them — and a database of implementations cannot
// describe absence.
package catalogcoverage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Asset is one desired-state catalog entry.
type Asset struct {
	ID           string
	Name         string
	Kind         string
	Domain       string
	Slot         string
	Delivery     string
	Targets      []string
	Kits         []string
	Priority     string
	Maturity     string
	Requires     []string
	Suggests     []string
	Expects      []string
	Satisfies    []string
	Capabilities []string
	States       []string
}

// Implementation is one on-disk component manifest.
type Implementation struct {
	// Name is the directory name under library/components or library/hooks.
	Name string
	// Root is "components" or "hooks".
	Root string
	// CatalogID is the nullable back-reference to a catalog asset. Empty means
	// the implementation is supplemental: real, adopted, and outside the target
	// list. That is legitimate, not an error.
	CatalogID string
	Latest    string
	Slot      string
}

type rawAsset struct {
	Kind  string `json:"kind"`
	Asset struct {
		ID       string   `json:"id"`
		Name     string   `json:"name"`
		Kind     string   `json:"kind"`
		Domain   string   `json:"domain"`
		Slot     string   `json:"slot"`
		Delivery string   `json:"delivery"`
		Targets  []string `json:"targets"`
		Kits     []string `json:"kits"`
		Target   struct {
			Priority string `json:"priority"`
			Maturity string `json:"maturity"`
		} `json:"target"`
	} `json:"asset"`
	Dependencies struct {
		Requires []struct {
			Asset string `json:"asset"`
		} `json:"requires"`
		Suggests []struct {
			Asset string `json:"asset"`
		} `json:"suggests"`
	} `json:"dependencies"`
	Expects []struct {
		Capability string `json:"capability"`
	} `json:"expects"`
	Satisfies            []string `json:"satisfies"`
	RequiredCapabilities []string `json:"requiredCapabilities"`
	RequiredStates       []string `json:"requiredStates"`
}

type rawManifest struct {
	LibraryID string `json:"libraryId"`
	CatalogID string `json:"catalogId"`
	Latest    string `json:"latest"`
	Slot      string `json:"slot"`
}

// LoadCatalog reads every asset document under catalogDir/assets.
func LoadCatalog(catalogDir string) ([]Asset, error) {
	paths, err := filepath.Glob(filepath.Join(catalogDir, "assets", "*", "*.json"))
	if err != nil {
		return nil, fmt.Errorf("scan catalog: %w", err)
	}
	sort.Strings(paths)
	out := make([]Asset, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var raw rawAsset
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if raw.Kind != "catalog-asset" {
			continue
		}
		asset := Asset{
			ID: raw.Asset.ID, Name: raw.Asset.Name, Kind: raw.Asset.Kind,
			Domain: raw.Asset.Domain, Slot: raw.Asset.Slot, Delivery: raw.Asset.Delivery,
			Targets: raw.Asset.Targets, Kits: raw.Asset.Kits,
			Priority: raw.Asset.Target.Priority, Maturity: raw.Asset.Target.Maturity,
			Satisfies: raw.Satisfies, Capabilities: raw.RequiredCapabilities,
			States: raw.RequiredStates,
		}
		for _, r := range raw.Dependencies.Requires {
			asset.Requires = append(asset.Requires, r.Asset)
		}
		for _, s := range raw.Dependencies.Suggests {
			asset.Suggests = append(asset.Suggests, s.Asset)
		}
		for _, e := range raw.Expects {
			asset.Expects = append(asset.Expects, e.Capability)
		}
		out = append(out, asset)
	}
	return out, nil
}

// LoadImplementations reads every component.json under the library roots.
func LoadImplementations(libraryDir string) ([]Implementation, error) {
	var out []Implementation
	for _, root := range []string{"components", "hooks"} {
		paths, err := filepath.Glob(filepath.Join(libraryDir, root, "*", "component.json"))
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", root, err)
		}
		sort.Strings(paths)
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", path, err)
			}
			var raw rawManifest
			if err := json.Unmarshal(data, &raw); err != nil {
				return nil, fmt.Errorf("parse %s: %w", path, err)
			}
			out = append(out, Implementation{
				Name:      filepath.Base(filepath.Dir(path)),
				Root:      root,
				CatalogID: raw.CatalogID,
				Latest:    raw.Latest,
				Slot:      raw.Slot,
			})
		}
	}
	return out, nil
}
