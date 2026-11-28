// Package search is the CLI's unified retrieval surface. Mirrors the API's
// Connect-RPC SearchService. Command surface loads from cli/manifest.json via
// cliapp.LoadFromManifestPrimitives.
package search

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "search"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"SearchService.Search":          cliapp.ProtoList(h.queryCall, h.queryReport),
		"SearchService.Suggest":         cliapp.ProtoList(h.suggestCall, h.suggestReport),
		"SearchService.GetSearchStatus": cliapp.ProtoList(h.statusCall, h.statusReport),
		"SearchService.Reindex":         cliapp.ProtoMutation(h.reindexCall, h.reindexReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("search: load from manifest: %w", err)
	}
	return group, nil
}
