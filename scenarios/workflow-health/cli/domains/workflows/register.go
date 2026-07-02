package workflows

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "workflows"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"WorkflowSearchService.SearchWorkflows": h.search,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("workflows: load from manifest: %w", err)
	}
	return group, nil
}
