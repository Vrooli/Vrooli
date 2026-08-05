package forest

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "forest"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{"ForestService.RunCompactionPass": cliapp.ProtoMutation(h.compactCall, h.compactReport), "ForestService.GetFrontier": cliapp.ProtoList(h.frontierCall, h.frontierReport), "ForestService.RebuildForest": cliapp.ProtoMutation(h.rebuildCall, h.rebuildReport)})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("forest: load manifest: %w", err)
	}
	return g, nil
}

func Commands(core *cliapp.ScenarioApp) []cliapp.Command {
	h := newHandlers(core)
	return []cliapp.Command{cliapp.Command{Name: "compact", Description: "Compact eligible memory frontier"}.WithPrimitive(cliapp.ProtoMutation(h.compactCall, h.compactReport)), cliapp.Command{Name: "frontier", Description: "Inspect current forest frontier"}.WithPrimitive(cliapp.ProtoList(h.frontierCall, h.frontierReport)), cliapp.Command{Name: "rebuild-forest", Description: "Rebuild derived memory forest"}.WithPrimitive(cliapp.ProtoMutation(h.rebuildCall, h.rebuildReport))}
}
