package registry

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "registry"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"RegistryService.ListTemplates": cliapp.ProtoList(h.listCall, h.listReport),
		"RegistryService.GetTemplate":   cliapp.ProtoList(h.showCall, h.showReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("registry: load from manifest: %w", err)
	}
	return group, nil
}
