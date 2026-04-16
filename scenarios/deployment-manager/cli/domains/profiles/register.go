package profiles

import (
	"errors"
	"fmt"

	profilecmd "deployment-manager/cli/profiles"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	commands := profilecmd.New(app.APIClient)
	return cliapp.CommandGroup{
		Title: "Profiles",
		Commands: []cliapp.Command{
			{Name: "profiles", NeedsAPI: true, Description: "List deployment profiles", Run: commands.List},
			{Name: "profile", NeedsAPI: true, Description: "Profile management commands", Run: route(commands)},
		},
	}
}

func route(commands *profilecmd.Commands) func([]string) error {
	return func(args []string) error {
		if len(args) == 0 {
			return errors.New("profile subcommand is required")
		}
		sub := args[0]
		rest := args[1:]
		switch sub {
		case "create":
			return commands.Create(rest)
		case "list":
			return commands.List(rest)
		case "show":
			return commands.Show(rest)
		case "delete":
			return commands.Delete(rest)
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
			return commands.Versions(rest)
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
