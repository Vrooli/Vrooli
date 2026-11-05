package strategy

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "strategy"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	doc, err := cliapp.ParseManifest(manifest)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("strategy: parse manifest: %w", err)
	}
	declared := doc.FindGroup(GroupName)
	if declared == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("strategy: group %q is not declared", GroupName)
	}
	h := newHandlers(core)
	commands := make([]cliapp.Command, 0, len(declared.Commands))
	for _, command := range declared.Commands {
		args, err := cliapp.ManifestArgs(command)
		if err != nil {
			return cliapp.SubcommandGroup{}, fmt.Errorf("strategy command %q: %w", command.Name, err)
		}
		var handler cliapp.PrimitiveHandler
		switch command.Name {
		case "benchmark":
			handler = cliapp.Action(h.benchmarkCall, h.benchmarkReport)
		case "list":
			handler = cliapp.ProtoList(h.listCall, h.listReport)
		case "show":
			handler = cliapp.ProtoList(h.showCall, h.showReport)
		case "compare":
			handler = cliapp.Action(h.compareCall, h.compareReport)
		default:
			return cliapp.SubcommandGroup{}, fmt.Errorf("strategy command %q has no local handler", command.Name)
		}
		commands = append(commands, cliapp.Command{
			Name: command.Name, Description: command.Description, Args: args,
			NeedsAPI: true, Architecture: command.Architecture.CommandArchitecture(),
		}.WithPrimitive(handler))
	}
	return cliapp.SubcommandGroup{Name: GroupName, Description: declared.Description, Subcommands: commands}, nil
}
