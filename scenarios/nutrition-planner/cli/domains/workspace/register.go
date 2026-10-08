package workspace

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "workspace"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"WorkspaceService.ListWorkspaces": h.list,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("workspace: load from manifest: %w", err)
	}
	return group, nil
}
