package index

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "index"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"CodeFactsService.GetIndexStatus":          cliapp.ProtoList(h.statusCall, h.statusReport),
		"CodeFactsService.ReconcileIndex":          cliapp.ProtoMutation(h.reconcileCall, h.mutationReport),
		"CodeFactsService.Reindex":                 cliapp.ProtoMutation(h.reindexCall, h.mutationReport),
		"CodeFactsService.CancelIndexJob":          cliapp.ProtoMutation(h.cancelCall, h.mutationReport),
		"CodeFactsService.PromoteIndexGeneration":  cliapp.ProtoMutation(h.promoteCall, h.mutationReport),
		"CodeFactsService.RollbackIndexGeneration": cliapp.ProtoMutation(h.rollbackCall, h.mutationReport),
		"CodeFactsService.CleanupIndex":            cliapp.ProtoMutation(h.cleanupCall, h.mutationReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("index: load from manifest: %w", err)
	}
	return group, nil
}
