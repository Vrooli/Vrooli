package facets

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "facets"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"FacetsService.ListFacets":      cliapp.ProtoList(h.listCall, h.listReport),
		"FacetsService.CountUnassigned": cliapp.ProtoList(h.unassignedCall, h.unassignedReport),
		"FacetsService.DeleteFacet":     cliapp.ProtoMutation(h.deleteCall, h.deleteReport),
		"FacetsService.SetFacetPolicy":  cliapp.ProtoMutation(h.setCall, h.setReport),
		"FacetsService.AssignFacet":     cliapp.ProtoMutation(h.assignCall, h.assignReport),
		"FacetsService.SetPin":          cliapp.ProtoMutation(h.pinCall, h.pinReport),
		"FacetsService.MarkSuperseded":  cliapp.ProtoMutation(h.supersedeCall, h.supersedeReport),
		"FacetsService.ResolveThread":   cliapp.ProtoMutation(h.resolveCall, h.resolveReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("facets: load manifest: %w", err)
	}
	return group, nil
}
