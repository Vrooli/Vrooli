package retrieval

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "retrieval"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"RetrievalService.Search":  cliapp.ProtoList(h.searchCall, h.searchReport),
		"RetrievalService.Ambient": cliapp.ProtoList(h.ambientCall, h.ambientReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("retrieval: load manifest: %w", err)
	}
	return group, nil
}
