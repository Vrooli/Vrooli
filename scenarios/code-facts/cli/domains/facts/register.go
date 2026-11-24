package facts

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "facts"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"CodeFactsService.DescribeCodeFacts":         cliapp.ProtoList(h.describeCall, h.describeReport),
		"CodeFactsService.DescribeComposition":       cliapp.ProtoList(h.compositionCall, h.compositionReport),
		"CodeFactsService.SearchComposition":         cliapp.ProtoList(h.compositionSearchCall, h.compositionSearchReport),
		"flow-search":                                cliapp.Action(h.flowSearchCall, h.flowSearchReport),
		"ambient-seams":                              cliapp.Action(h.ambientSeamsCall, h.ambientSeamsReport),
		"persistence-patterns":                       cliapp.Action(h.persistencePatternsCall, h.persistencePatternsReport),
		"endpoint-anatomy":                           cliapp.Action(h.endpointAnatomyCall, h.endpointAnatomyReport),
		"CodeFactsService.SearchInvariantCandidates": cliapp.ProtoList(h.invariantCandidatesCall, h.invariantCandidatesReport),
		"CodeFactsService.Search":                    cliapp.ProtoList(h.searchCall, h.searchReport),
		"CodeFactsService.ListSurfaces":              cliapp.ProtoList(h.surfacesCall, h.surfacesReport),
		"CodeFactsService.CheckProtoAdoption":        cliapp.ProtoList(h.protoAdoptionCall, h.proofReport),
		"CodeFactsService.CheckEndpointProof":        cliapp.ProtoList(h.endpointProofCall, h.proofReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("facts: load from manifest: %w", err)
	}
	return group, nil
}
