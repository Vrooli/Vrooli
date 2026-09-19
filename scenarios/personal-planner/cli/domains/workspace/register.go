package workspace

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "workspace"

func Register(c *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(c)
	b := map[string]cliapp.PrimitiveHandler{
		"WorkspaceService.GetProfile":          cliapp.ProtoList(h.profileCall, h.profileReport),
		"WorkspaceService.UpdateProfile":       cliapp.ProtoMutation(h.updateCall, h.updateReport),
		"WorkspaceService.ListAvailability":    cliapp.ProtoList(h.availabilityCall, h.availabilityReport),
		"WorkspaceService.ReplaceAvailability": cliapp.ProtoMutation(h.replaceAvailabilityCall, h.replaceAvailabilityReport),
	}
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, b)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("workspace: load from manifest: %w", err)
	}
	return g, nil
}
