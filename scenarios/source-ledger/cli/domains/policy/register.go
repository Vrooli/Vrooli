package policy

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "policy"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"ScopesService.GetPolicy":   cliapp.ProtoList(h.showCall, h.showReport),
		"ScopesService.SetPolicy":   cliapp.ProtoMutation(h.setCall, h.setReport),
		"ScopesService.ResetPolicy": cliapp.ProtoMutation(h.resetCall, h.resetReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("policy: load manifest: %w", err)
	}
	return group, nil
}
