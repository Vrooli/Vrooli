package offers

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "offers"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	b := map[string]cliapp.PrimitiveHandler{
		"CatalogService.ListNodes": cliapp.ProtoList(h.list, listReport), "CatalogService.ImportCatalog": cliapp.ProtoMutation(h.catalogImport, catalogImportReport), "CatalogService.CreateNode": cliapp.ProtoMutation(h.create, createReport), "CatalogService.Transition": cliapp.ProtoMutation(h.transition, transitionReport), "CatalogService.CreateEdge": cliapp.ProtoMutation(h.edge, edgeReport), "CatalogService.ListEdges": cliapp.ProtoList(h.edgesList, edgesListReport), "GatesService.DeclareTrigger": cliapp.ProtoMutation(h.trigger, triggerReport), "GatesService.AddFact": cliapp.ProtoMutation(h.fact, factReport), "GatesService.Evaluate": cliapp.ProtoMutation(h.evaluate, evaluateReport), "GatesService.Promote": cliapp.ProtoMutation(h.promote, promoteReport), "GatesService.ListProposals": cliapp.ProtoList(h.proposals, proposalsReport), "BoardService.GetBoard": cliapp.ProtoList(h.board, boardReport), "SpaceService.GetProjection": cliapp.ProtoList(h.space, spaceReport),
	}
	g, e := cliapp.LoadFromManifestPrimitives(manifest, GroupName, b)
	if e != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("offers: load from manifest: %w", e)
	}
	return g, nil
}
