package forest

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "forest"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"ForestService.GetFrontier":       cliapp.ProtoList(h.frontierCall, h.frontierReport),
		"ForestService.RunCompactionPass": cliapp.ProtoMutation(h.compactCall, h.compactReport),
		"ForestService.RebuildForest":     cliapp.ProtoMutation(h.rebuildCall, h.rebuildReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("forest: load manifest: %w", err)
	}
	return group, nil
}
