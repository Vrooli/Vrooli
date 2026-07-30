package recall

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "recall"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{"RecallService.Recall": cliapp.ProtoList(h.recallCall, h.recallReport), "RecallService.Wake": cliapp.ProtoList(h.wakeCall, h.wakeReport), "RecallService.ListSiblingEvents": cliapp.ProtoList(h.siblingsCall, h.siblingsReport)})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("recall: load manifest: %w", err)
	}
	g.DefaultSubcommand = "recall"
	return g, nil
}

func Commands(core *cliapp.ScenarioApp) []cliapp.Command {
	h := newHandlers(core)
	return []cliapp.Command{
		cliapp.Command{Name: "recall", Description: "Semantically recall memories", Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{{Name: "query", Required: true, Description: "Semantic query"}},
			Flags:       []cliapp.Flag{{Name: "limit", Description: "Maximum hits"}},
		}}.WithPrimitive(cliapp.ProtoList(h.recallCall, h.recallReport)),
		cliapp.Command{Name: "wake", Description: "Render bounded ambient memory", Args: cliapp.ArgSchema{
			Flags: []cliapp.Flag{{Name: "budget", Description: "Line budget"}},
		}}.WithPrimitive(cliapp.ProtoList(h.wakeCall, h.wakeReport)),
		cliapp.Command{Name: "siblings", Description: "List memories from the same agent run", Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{{Name: "entry-id", Required: true, Description: "Memory entry id"}},
		}}.WithPrimitive(cliapp.ProtoList(h.siblingsCall, h.siblingsReport)),
	}
}
