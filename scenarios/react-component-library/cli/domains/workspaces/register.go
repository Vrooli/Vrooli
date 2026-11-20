// Package workspaces is the CLI's durable composition surface. Mirrors the
// API's Connect-RPC WorkspaceService. Command surface loads from
// cli/manifest.json via cliapp.LoadFromManifestPrimitives.
package workspaces

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "workspaces"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"WorkspaceService.ListWorkspaces":         cliapp.ProtoList(h.listCall, h.listReport),
		"WorkspaceService.GetWorkspace":           cliapp.ProtoList(h.getCall, h.getReport),
		"WorkspaceService.CreateWorkspace":        cliapp.ProtoMutation(h.createCall, h.createReport),
		"WorkspaceService.SaveWorkspace":          cliapp.ProtoMutation(h.saveCall, h.saveReport),
		"WorkspaceService.DeleteWorkspace":        cliapp.ProtoMutation(h.deleteCall, h.deleteReport),
		"WorkspaceService.ForkWorkspace":          cliapp.ProtoMutation(h.forkCall, h.forkReport),
		"WorkspaceService.ValidateWorkspace":      cliapp.ProtoList(h.validateCall, h.validateReport),
		"WorkspaceService.ListWorkspaceRevisions": cliapp.ProtoList(h.revisionsCall, h.revisionsReport),
		"WorkspaceService.AssistWorkspace":        cliapp.ProtoMutation(h.assistCall, h.assistReport),
		// Local binding: promote a built workspace into the reviewed shelf.
		"export-seed": cliapp.ContentAction(h.exportSeedCall),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("workspaces: load from manifest: %w", err)
	}
	return group, nil
}
