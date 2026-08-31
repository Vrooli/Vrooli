package support

import "github.com/vrooli/cli-core/cliapp"

type CommandFunc func(args []string) error

type Dependencies struct {
	Profile          CommandFunc
	Declarations     CommandFunc
	Workflow         CommandFunc
	Task             CommandFunc
	Run              CommandFunc
	RunCommands      []cliapp.Command
	Runner           CommandFunc
	Policy           CommandFunc
	PermissionPolicy CommandFunc
	Settings         CommandFunc
	Maintenance      CommandFunc
	Ops              CommandFunc
	Health           CommandFunc
	Events           CommandFunc
	Findings         CommandFunc
	Subscription     CommandFunc
	ScenarioSmoke    CommandFunc
}

func Command(name, description string, run CommandFunc) cliapp.Command {
	return cliapp.Command{
		Name:        name,
		NeedsAPI:    true,
		Description: description,
		Run:         run,
	}
}

// Subcommand adapts an existing group dispatcher to cliapp's discoverable
// subcommand model while the command-specific flag parsing remains local.
func Subcommand(name, description string, dispatch CommandFunc) cliapp.Command {
	return cliapp.Command{
		Name:        name,
		NeedsAPI:    true,
		Description: description,
		Run: func(args []string) error {
			return dispatch(append([]string{name}, args...))
		},
	}
}

func SubcommandGroup(name, description string, dispatch CommandFunc, commands ...[2]string) cliapp.SubcommandGroup {
	subcommands := make([]cliapp.Command, 0, len(commands))
	for _, command := range commands {
		subcommands = append(subcommands, Subcommand(command[0], command[1], dispatch))
	}
	return cliapp.SubcommandGroup{Name: name, Description: description, NeedsAPI: true, Subcommands: subcommands}
}
