// Package portcontract derives host obligations from the catalog closure and
// the experience-manager capability registry.
package portcontract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"react-component-library/internal/assetgraph"
	"react-component-library/internal/catalogcoverage"
)

type Port struct {
	CapabilityID        string
	DemandingAssets     []assetgraph.Node
	CandidateSatisfiers []assetgraph.Node
}

type Contract struct {
	AssetID       string
	ClosureCount  int
	SelfContained bool
	UnmetPorts    []Port
}

type registryGroup struct {
	Capabilities []struct {
		ID     string   `json:"id"`
		Facets []string `json:"facets"`
	} `json:"capabilities"`
}

func Build(repoRoot, assetID string) (Contract, error) {
	assets, err := catalogcoverage.LoadCatalog(filepath.Join(repoRoot, "scenarios", "react-component-library", "catalog"))
	if err != nil {
		return Contract{}, err
	}
	index, err := assetgraph.Build(assets)
	if err != nil {
		return Contract{}, err
	}
	closure, err := index.Closure(assetID)
	if err != nil {
		return Contract{}, err
	}
	ports, err := loadPorts(filepath.Join(repoRoot, "scenarios", "experience-manager", "capabilities", "capabilities"))
	if err != nil {
		return Contract{}, err
	}
	byID := map[string]catalogcoverage.Asset{}
	for _, asset := range assets {
		byID[asset.ID] = asset
	}
	demanders := map[string][]assetgraph.Node{}
	satisfied := map[string]bool{}
	for _, node := range closure {
		asset := byID[node.ID]
		for _, capability := range asset.Capabilities {
			if ports[capability] {
				demanders[capability] = append(demanders[capability], node)
			}
		}
		for _, capability := range asset.Satisfies {
			if ports[capability] {
				satisfied[capability] = true
			}
		}
	}
	candidates := map[string][]assetgraph.Node{}
	for _, asset := range assets {
		node, nodeErr := index.Node(asset.ID)
		if nodeErr != nil {
			continue
		}
		for _, capability := range asset.Satisfies {
			if ports[capability] {
				candidates[capability] = append(candidates[capability], node)
			}
		}
	}
	ids := make([]string, 0, len(demanders))
	for capability := range demanders {
		if !satisfied[capability] {
			ids = append(ids, capability)
		}
	}
	sort.Strings(ids)
	contract := Contract{AssetID: assetID, ClosureCount: len(closure), SelfContained: len(ids) == 0}
	for _, capability := range ids {
		demanding := append([]assetgraph.Node(nil), demanders[capability]...)
		candidate := append([]assetgraph.Node(nil), candidates[capability]...)
		sortNodes(demanding)
		sortNodes(candidate)
		contract.UnmetPorts = append(contract.UnmetPorts, Port{CapabilityID: capability, DemandingAssets: demanding, CandidateSatisfiers: candidate})
	}
	return contract, nil
}

func loadPorts(dir string) (map[string]bool, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	ports := map[string]bool{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var group registryGroup
		if err := json.Unmarshal(data, &group); err != nil {
			return nil, fmt.Errorf("parse capability registry %s: %w", path, err)
		}
		for _, capability := range group.Capabilities {
			for _, facet := range capability.Facets {
				if facet == "port" {
					ports[capability.ID] = true
				}
			}
		}
	}
	return ports, nil
}

func sortNodes(nodes []assetgraph.Node) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Rung != nodes[j].Rung {
			return nodes[i].Rung > nodes[j].Rung
		}
		return nodes[i].ID < nodes[j].ID
	})
}
