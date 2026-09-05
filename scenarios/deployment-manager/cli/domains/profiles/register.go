package profiles

import (
	"errors"
	"fmt"

	profilecmd "deployment-manager/cli/profiles"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "profiles"

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	commands := profilecmd.New(app.APIClient)
	connectCommands := newConnectCommands(app)
	return cliapp.CommandGroup{
		Title: "Profiles",
		Commands: []cliapp.Command{
			{Name: "profile", NeedsAPI: true, Description: "Profile management commands", Run: route(commands, connectCommands)},
		},
	}
}

func RegisterConnect(app *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	commands := newConnectCommands(app)
	return cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"ProfilesService.ListProfiles":        cliapp.ProtoList(commands.listCall, commands.listReport),
		"ProfilesService.CreateProfile":       cliapp.ProtoMutation(commands.createCall, commands.createReport),
		"ProfilesService.GetProfile":          cliapp.ProtoList(commands.showCall, commands.showReport),
		"ProfilesService.UpdateProfile":       cliapp.ProtoMutation(commands.updateCall, commands.updateReport),
		"ProfilesService.DeleteProfile":       cliapp.ProtoMutation(commands.deleteCall, commands.deleteReport),
		"ProfilesService.ListProfileVersions": cliapp.ProtoList(commands.versionsCall, commands.versionsReport),
	})
}

func route(commands *profilecmd.Commands, connectCommands *connectCommands) func([]string) error {
	return func(args []string) error {
		if len(args) == 0 {
			return errors.New("profile subcommand is required")
		}
		sub := args[0]
		rest := args[1:]
		switch sub {
		case "create":
			return connectCommands.create(rest)
		case "list":
			return connectCommands.list(rest)
		case "show":
			return connectCommands.show(rest)
		case "delete":
			return connectCommands.delete(rest)
		case "export":
			return commands.Export(rest)
		case "import":
			return commands.Import(rest)
		case "update":
			return commands.Update(rest)
		case "set":
			return commands.Set(rest)
		case "swap":
			return commands.Swap(rest)
		case "versions":
			return connectCommands.versions(rest)
		case "analyze":
			return commands.Analyze(rest)
		case "save":
			return commands.Save(rest)
		case "diff":
			return commands.Diff(rest)
		case "rollback":
			return commands.Rollback(rest)
		default:
			return fmt.Errorf("unknown profile subcommand: %s", sub)
		}
	}
}
