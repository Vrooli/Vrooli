package embedding

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "embedding"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	doc, err := cliapp.ParseManifest(manifest)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("embedding: parse manifest: %w", err)
	}
	declared := doc.FindGroup(GroupName)
	if declared == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("embedding: group %q is not declared", GroupName)
	}
	handlers := map[string]cliapp.PrimitiveHandler{
		"inventory":      cliapp.Action(h.inventoryCall, h.report),
		"retarget-plan":  cliapp.Action(h.planCall, h.report),
		"apply-shadow":   cliapp.Action(h.applyShadowCall, h.report),
		"record-compare": cliapp.Action(h.recordCompareCall, h.report),
		"abort":          cliapp.Action(h.abortCall, h.report),
		"cutover":        cliapp.Action(h.cutoverCall, h.report),
		"rollback":       cliapp.Action(h.rollbackCall, h.report),
	}
	commands := make([]cliapp.Command, 0, len(declared.Commands))
	for _, command := range declared.Commands {
		args, err := cliapp.ManifestArgs(command)
		if err != nil {
			return cliapp.SubcommandGroup{}, fmt.Errorf("embedding command %q: %w", command.Name, err)
		}
		handler, ok := handlers[command.Name]
		if !ok {
			return cliapp.SubcommandGroup{}, fmt.Errorf("embedding command %q has no local handler", command.Name)
		}
		commands = append(commands, cliapp.Command{
			Name: command.Name, Description: command.Description, Args: args,
			Architecture: command.Architecture.CommandArchitecture(),
		}.WithPrimitive(handler))
	}
	return cliapp.SubcommandGroup{Name: GroupName, Description: declared.Description, Subcommands: commands, NeedsAPI: false}, nil
}
