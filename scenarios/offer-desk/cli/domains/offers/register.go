package offers

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "offers"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	b := map[string]cliapp.PrimitiveHandler{
		"CatalogService.SetDeliverableClass": cliapp.ProtoMutation(h.setClass, setClassReport), "CatalogService.GetMeterInventory": cliapp.ProtoList(h.meters, metersReport), "ReleaseLadderService.GetEnablingDeliverables": cliapp.ProtoList(h.enabling, enablingReport),
		"CatalogService.ListNodes": cliapp.ProtoList(h.list, listReport), "CatalogService.ImportCatalog": cliapp.ProtoMutation(h.catalogImport, catalogImportReport), "CatalogService.MapAccount": cliapp.ProtoMutation(h.catalogMapAccount, catalogMapAccountReport), "CatalogService.MergeNodes": cliapp.ProtoMutation(h.catalogMerge, catalogMergeReport), "CatalogService.RenameNode": cliapp.ProtoMutation(h.rename, renameReport), "CatalogService.VerifyCatalog": cliapp.ProtoListOutcome(h.catalogVerify, catalogVerifyReport, catalogVerifyOutcome), "CatalogService.CreateNode": cliapp.ProtoMutation(h.create, createReport), "CatalogService.Transition": cliapp.ProtoMutation(h.transition, transitionReport), "CatalogService.SetReleaseRank": cliapp.ProtoMutation(h.rank, rankReport), "CatalogService.CreateEdge": cliapp.ProtoMutation(h.edge, edgeReport), "CatalogService.ListEdges": cliapp.ProtoList(h.edgesList, edgesListReport), "GatesService.DeclareTrigger": cliapp.ProtoMutation(h.trigger, triggerReport), "GatesService.AddFact": cliapp.ProtoMutation(h.fact, factReport), "GatesService.Evaluate": cliapp.ProtoMutation(h.evaluate, evaluateReport), "GatesService.Promote": cliapp.ProtoMutation(h.promote, promoteReport), "GatesService.ListProposals": cliapp.ProtoList(h.proposals, proposalsReport), "BoardService.GetBoard": cliapp.ProtoList(h.board, boardReport), "ReleaseLadderService.GetReleaseLadder": cliapp.ProtoList(h.ladder, ladderReport), "ReleaseLadderService.GetPrerequisites": cliapp.ProtoList(h.prerequisites, prerequisitesReport), "SpaceService.GetProjection": cliapp.ProtoList(h.space, spaceReport),
	}
	g, e := cliapp.LoadFromManifestPrimitives(manifest, GroupName, b)
	if e != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("offers: load from manifest: %w", e)
	}
	return g, nil
}
