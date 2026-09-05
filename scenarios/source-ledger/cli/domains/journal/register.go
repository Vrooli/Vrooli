package journal

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "journal"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"JournalService.AppendEntry": cliapp.ProtoMutation(h.noteCall, h.noteReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("journal: load manifest: %w", err)
	}
	return group, nil
}
