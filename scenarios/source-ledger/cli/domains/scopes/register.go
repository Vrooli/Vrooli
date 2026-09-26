package scopes

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "scopes"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"ScopesService.CreateScope": cliapp.ProtoMutation(h.createCall, h.createReport),
		"ScopesService.ListScopes":  cliapp.ProtoList(h.listCall, h.listReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("scopes: load manifest: %w", err)
	}
	return group, nil
}
