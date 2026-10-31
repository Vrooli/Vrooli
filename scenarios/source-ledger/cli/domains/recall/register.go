package recall

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "recall"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"RecallService.Recall": cliapp.ProtoList(h.recallCall, h.recallReport),
		"RecallService.Wake":   cliapp.ProtoList(h.wakeCall, h.wakeReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("recall: load manifest: %w", err)
	}
	group.DefaultSubcommand = "recall"
	return group, nil
}
