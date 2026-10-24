package sources

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "sources"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{"SourcesService.ListAdapters": cliapp.ProtoList(h.listCall, h.listReport), "SourcesService.SetAdapterEnabled": cliapp.ProtoMutation(h.enableCall, h.enableReport), "SourcesService.ImportArchive": cliapp.ProtoMutation(h.importCall, h.importReport)})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("sources: load manifest: %w", err)
	}
	return group, nil
}
