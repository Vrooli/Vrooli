package intents

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "intents"

// Register builds the intents group from the embedded manifest. Both commands
// are local (no dedicated RPC): `run` drives the public Query RPC exactly as an
// agent would, and `coverage` reads authored files only.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	doc, err := cliapp.ParseManifest(manifest)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("intents: parse manifest: %w", err)
	}
	declared := doc.FindGroup(GroupName)
	if declared == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("intents: group %q is not declared", GroupName)
	}
	h := newHandlers(core)
	commands := make([]cliapp.Command, 0, len(declared.Commands))
	for _, command := range declared.Commands {
		args, err := cliapp.ManifestArgs(command)
		if err != nil {
			return cliapp.SubcommandGroup{}, fmt.Errorf("intents command %q: %w", command.Name, err)
		}
		var handler cliapp.PrimitiveHandler
		needsAPI := false
		switch command.Name {
		case "run":
			handler = cliapp.ActionWithExit(h.runCall, h.runReport, h.runExit)
			needsAPI = true
		case "coverage":
			handler = cliapp.Action(h.coverageCall, h.coverageReport)
		default:
			return cliapp.SubcommandGroup{}, fmt.Errorf("intents command %q has no local handler", command.Name)
		}
		commands = append(commands, cliapp.Command{
			Name: command.Name, Description: command.Description, Args: args,
			NeedsAPI: needsAPI, Architecture: command.Architecture.CommandArchitecture(),
		}.WithPrimitive(handler))
	}
	return cliapp.SubcommandGroup{Name: GroupName, Description: declared.Description, Subcommands: commands}, nil
}
