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
	"strings"

	"react-component-library/internal/assetrung"
)

// Asset is one desired-state catalog entry.
type Asset struct {
	ID           string
	Name         string
	Kind         string
	Surface      string
	Rung         assetrung.Rung
	RungName     string
	Domain       string
	DomainOrder  int
	Slot         string
	Delivery     string
	Targets      []string
	Kits         []string
	Priority     string
	PinnedWeight float64
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
	// Name is the directory name under any library asset root.
	Name      string
	LibraryID string
	Path      string
	// Root is "foundations", "hooks", "services", "primitives", or "components".
	Root string
	// CatalogID is the nullable back-reference to a catalog asset. Empty means
	// the implementation is supplemental: real, adopted, and outside the target
	// list. That is legitimate, not an error.
	CatalogID string
	// SupplementalJustification is required when CatalogID is empty so an
	// implementation outside the target catalog is an explicit decision.
	SupplementalJustification string
	Latest                    string
	Slot                      string
	Dependencies              []ManifestDependency
	// ExperienceStateKnown is true for live catalog loading. It lets unit
	// callers retain the small in-memory Compute fixture while production
	// coverage fail-closes when no registered contract exists.
	ExperienceStateKnown bool
	ExperienceRegistered bool
	ExperienceVacuous    bool
}

type ManifestDependency struct {
	LibraryID string
	Version   string
}

type rawAsset struct {
	Kind  string `json:"kind"`
	Asset struct {
		ID       string   `json:"id"`
		Name     string   `json:"name"`
		Kind     string   `json:"kind"`
		Surface  string   `json:"surface"`
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

type catalogDomain struct {
	ID    string `json:"id"`
	Order int    `json:"order"`
}

type rawManifest struct {
	LibraryID                 string `json:"libraryId"`
	CatalogID                 string `json:"catalogId"`
	SupplementalJustification string `json:"x-supplementalJustification"`
	Latest                    string `json:"latest"`
	Slot                      string `json:"slot"`
	Dependencies              []struct {
		LibraryID string `json:"libraryId"`
		Version   string `json:"version"`
	} `json:"dependencies"`
}

// LoadGateDefinitions reads the catalog gate registry without coupling API
// handlers to the authored JSON shape.
func LoadGateDefinitions(configPath string) ([]GateDefinition, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read catalog config: %w", err)
	}
	var doc struct {
		Gates []GateDefinition `json:"gates"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse catalog config: %w", err)
	}
	return doc.Gates, nil
}

// LoadCatalog reads every asset document under catalogDir/assets.
func LoadCatalog(catalogDir string) ([]Asset, error) {
	orders, err := loadDomainOrders(filepath.Join(catalogDir, "config.json"))
	if err != nil {
		return nil, err
	}
	paths, err := filepath.Glob(filepath.Join(catalogDir, "assets", "*", "*.json"))
	if err != nil {
		return nil, fmt.Errorf("scan catalog: %w", err)
	}
	sort.Strings(paths)
	weights := loadPinnedWeights(filepath.Join(catalogDir, "weights.json"))
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
			ID: raw.Asset.ID, Name: raw.Asset.Name, Kind: raw.Asset.Kind, Surface: raw.Asset.Surface,
			Domain: raw.Asset.Domain, DomainOrder: orders[raw.Asset.Domain], Slot: raw.Asset.Slot, Delivery: raw.Asset.Delivery,
			Targets: raw.Asset.Targets, Kits: raw.Asset.Kits,
			Priority: raw.Asset.Target.Priority, Maturity: raw.Asset.Target.Maturity,
			Satisfies: raw.Satisfies, Capabilities: raw.RequiredCapabilities,
			States:       raw.RequiredStates,
			PinnedWeight: weights[raw.Asset.ID],
		}
		rung, err := assetrung.Of(asset.Kind)
		if err != nil {
			return nil, fmt.Errorf("asset %s: %w", asset.ID, err)
		}
		asset.Rung, asset.RungName = rung, rung.Name()
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

func loadPinnedWeights(path string) map[string]float64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]float64{}
	}
	var doc struct {
		Weights map[string]float64 `json:"weights"`
	}
	if json.Unmarshal(data, &doc) != nil || doc.Weights == nil {
		return map[string]float64{}
	}
	return doc.Weights
}

func loadDomainOrders(path string) (map[string]int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]int{}, nil
		}
		return nil, fmt.Errorf("read catalog config: %w", err)
	}
	var config struct {
		Domains []catalogDomain `json:"domains"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse catalog config: %w", err)
	}
	orders := make(map[string]int, len(config.Domains))
	for _, domain := range config.Domains {
		orders[domain.ID] = domain.Order
	}
	return orders, nil
}

// LoadImplementations reads every component.json under the library roots.
func LoadImplementations(libraryDir string) ([]Implementation, error) {
	var out []Implementation
	experience := loadExperienceStates(filepath.Dir(libraryDir))
	for _, root := range []string{"foundations", "hooks", "services", "primitives", "components"} {
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
			implementation := Implementation{
				Name:                      filepath.Base(filepath.Dir(path)),
				LibraryID:                 raw.LibraryID,
				Path:                      path,
				Root:                      root,
				CatalogID:                 raw.CatalogID,
				SupplementalJustification: raw.SupplementalJustification,
				Latest:                    raw.Latest,
				Slot:                      raw.Slot,
				Dependencies: func() []ManifestDependency {
					deps := make([]ManifestDependency, 0, len(raw.Dependencies))
					for _, dep := range raw.Dependencies {
						deps = append(deps, ManifestDependency{LibraryID: dep.LibraryID, Version: dep.Version})
					}
					return deps
				}(),
				ExperienceStateKnown: true,
			}
			if state, ok := experience[implementation.Name+"@"+implementation.Latest]; ok {
				implementation.ExperienceRegistered = true
				implementation.ExperienceVacuous = state.vacuous
			}
			out = append(out, implementation)
		}
	}
	return out, nil
}

type experienceState struct{ vacuous bool }

// loadExperienceStates joins generated scenario component documents back to
// their library source by storyRef. This is intentionally read-only: the
// experience index remains the authority, while coverage refuses to promote a
// library asset that has no registered, substantive contract.
func loadExperienceStates(scenarioRoot string) map[string]experienceState {
	out := map[string]experienceState{}
	paths, _ := filepath.Glob(filepath.Join(scenarioRoot, "experience", "components", "*.json"))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var document struct {
			Component struct {
				StoryRef string `json:"storyRef"`
			} `json:"component"`
			Claims []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"claims"`
		}
		if json.Unmarshal(data, &document) != nil || document.Component.StoryRef == "" {
			continue
		}
		story := filepath.ToSlash(document.Component.StoryRef)
		parts := strings.Split(story, "/")
		version := ""
		name := ""
		for i, part := range parts {
			if part == "versions" && i+1 < len(parts) {
				version = parts[i+1]
				if i > 0 {
					name = parts[i-1]
				}
				break
			}
		}
		if name == "" || version == "" {
			continue
		}
		substantive := false
		for _, claim := range document.Claims {
			if claim.ID != "contract-present" && claim.Type != "" && claim.Type != "custom" {
				substantive = true
				break
			}
		}
		out[name+"@"+version] = experienceState{vacuous: !substantive}
	}
	return out
}
