package facets

import (
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "facets"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"FacetsService.ListFacets":         cliapp.ProtoList(h.listCall, h.listReport),
		"FacetsService.AssignFacet":        cliapp.ProtoMutation(h.assignCall, h.assignReport),
		"FacetsService.SetPin":             cliapp.ProtoMutation(h.pinCall, h.pinReport),
		"FacetsService.ListPinProposals":   cliapp.ProtoList(h.proposalsCall, h.proposalsReport),
		"FacetsService.ListPinCandidates":  cliapp.ProtoList(h.candidatesCall, h.candidatesReport),
		"FacetsService.ResolvePinProposal": cliapp.ProtoMutation(h.resolveProposalCall, h.proposalReport),
		"FacetsService.MarkSuperseded":     cliapp.ProtoMutation(h.supersedeCall, h.supersedeReport),
		"FacetsService.ResolveThread":      cliapp.ProtoMutation(h.resolveThreadCall, h.threadReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("facets: load manifest: %w", err)
	}
	return g, nil
}
