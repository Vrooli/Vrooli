package graph

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "graph"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"GraphService.DescribeTechTree": h.describe,
		"GraphService.GetNeighborhood":  h.neighbors,
		"GraphService.FindPath":         h.path,
		"GraphService.ListAncestors":    h.ancestors,
		"GraphService.ExportTechTree":   h.export,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("graph: load from manifest: %w", err)
	}
	return group, nil
}
