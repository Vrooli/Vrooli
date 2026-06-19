// Package workspace is the CLI's workspace-domain command surface. It mirrors
// the API's Connect-RPC WorkspaceService — the shared pane layout plus
// group/pane mutations — and is built from the embedded manifest.
package workspace

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

// GroupName is the manifest group name this package owns.
const GroupName = "workspace"

// Register builds the `workspace` subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers. All RPCs go through the
// Connect-RPC WorkspaceService.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"WorkspaceService.GetLayout":   h.layoutGet,
		"WorkspaceService.SaveLayout":  h.layoutSave,
		"WorkspaceService.UpdatePane":  h.paneUpdate,
		"WorkspaceService.DeletePane":  h.paneDelete,
		"WorkspaceService.CreateGroup": h.groupCreate,
		"WorkspaceService.UpdateGroup": h.groupUpdate,
		"WorkspaceService.DeleteGroup": h.groupDelete,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("workspace: load from manifest: %w", err)
	}
	// Preserve the pre-manifest subcommand alias (cli-manifest/v1 has no
	// per-command alias field).
	support.ApplyAliases(group.Subcommands, map[string][]string{
		"layout-get": {"layout"},
	})
	return group, nil
}
