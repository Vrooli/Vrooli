package files

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "files"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"FilesService.Search":    cliapp.ProtoList(h.searchCall, h.searchReport),
		"FilesService.ListFeeds": cliapp.ProtoList(h.feedsCall, h.feedsReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("files: load manifest: %w", err)
	}
	group.DefaultSubcommand = "search"
	return group, nil
}
