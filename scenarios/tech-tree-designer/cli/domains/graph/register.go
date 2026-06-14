package graph

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "graph"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	_ = core
	reserved := func(cliapp.RunContext) error {
		return fmt.Errorf("graph commands are reserved until the graph domain is implemented")
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"GraphService.DescribeTechTree": reserved,
		"GraphService.GetNeighborhood":  reserved,
		"GraphService.FindPath":         reserved,
		"GraphService.ListAncestors":    reserved,
		"GraphService.ExportTechTree":   reserved,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("graph: load from manifest: %w", err)
	}
	return group, nil
}
