package retrieval

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "retrieval"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"RetrievalService.Query":   cliapp.ProtoList(h.queryCall, h.queryReport),
		"RetrievalService.Similar": cliapp.ProtoList(h.similarCall, h.similarReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("retrieval: load from manifest: %w", err)
	}
	return group, nil
}
