package discovery

import (
	"path/filepath"
	"sort"

	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/filesystem"
)

func FindContracts(root string) ([]contract.Contract, error) {
	paths, err := filesystem.Find(root, ".flow.json")
	if err != nil {
		return nil, err
	}
	contracts := make([]contract.Contract, 0, len(paths))
	for _, rel := range paths {
		loaded, err := contract.Load(filepath.Join(root, filepath.FromSlash(rel)), rel)
		if err != nil {
			return nil, err
		}
		contracts = append(contracts, loaded)
	}
	sort.Slice(contracts, func(i int, j int) bool {
		return contracts[i].FlowID < contracts[j].FlowID
	})
	return contracts, nil
}

func Filter(contracts []contract.Contract, flowID string) []contract.Contract {
	if flowID == "" {
		return contracts
	}
	var out []contract.Contract
	for _, c := range contracts {
		if c.FlowID == flowID {
			out = append(out, c)
		}
	}
	return out
}
