package profiles

import (
	"errors"
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "profiles"

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	connectCommands := newConnectCommands(app)
	return cliapp.CommandGroup{
		Title: "Profiles",
		Commands: []cliapp.Command{
			{Name: "profile", NeedsAPI: true, Description: "Profile management commands", Run: route(connectCommands)},
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

func route(connectCommands *connectCommands) func([]string) error {
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
			return connectCommands.export(rest)
		case "import":
			return connectCommands.importProfile(rest)
		case "update":
			return connectCommands.update(rest)
		case "set":
			return errors.New("profile set is retired; use profile update with --name, --scenario, or --tier")
		case "swap":
			return errors.New("profile swap is retired; use swaps apply <profile-id> <from> <to>")
		case "versions":
			return connectCommands.versions(rest)
		case "analyze":
			return errors.New("profile analyze is retired; use analyze <scenario>")
		case "save":
			return errors.New("profile save is retired; typed profile mutations create the durable version")
		case "diff":
			return errors.New("profile diff is unavailable in the typed profile contract; use profile versions")
		case "rollback":
			return errors.New("profile rollback is unavailable in the typed profile contract; use a new profile update")
		default:
			return fmt.Errorf("unknown profile subcommand: %s", sub)
		}
	}
}
