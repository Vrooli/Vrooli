package discovery

import (
	"fmt"
	"path/filepath"
	"sort"

	"flow-verifier/internal/flows/compile"
	"flow-verifier/internal/flows/contract"
	"flow-verifier/internal/flows/model"
	"flow-verifier/internal/fsadapter"
)

func FindContracts(root string) ([]model.Flow, error) {
	paths, err := fsadapter.Find(root, "/flow/flow.json")
	if err != nil {
		return nil, err
	}
	flows := make([]model.Flow, 0, len(paths))
	for _, rel := range paths {
		raw, err := contract.LoadRaw(filepath.Join(root, filepath.FromSlash(rel)), rel)
		if err != nil {
			return nil, err
		}
		flow, err := compile.Compile(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid temporal flow contract %s:\n%s", rel, err)
		}
		flows = append(flows, flow)
	}
	sort.Slice(flows, func(i int, j int) bool {
		return flows[i].FlowID < flows[j].FlowID
	})
	return flows, nil
}

func Filter(flows []model.Flow, flowID string) []model.Flow {
	if flowID == "" {
		return flows
	}
	var out []model.Flow
	for _, flow := range flows {
		if flow.FlowID == flowID {
			out = append(out, flow)
		}
	}
	return out
}
