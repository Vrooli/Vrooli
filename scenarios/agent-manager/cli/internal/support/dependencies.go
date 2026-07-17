package support

import "github.com/vrooli/cli-core/cliapp"

type CommandFunc func(args []string) error

type Dependencies struct {
	Profile          CommandFunc
	Declarations     CommandFunc
	Workflow         CommandFunc
	Task             CommandFunc
	Run              CommandFunc
	Runner           CommandFunc
	Policy           CommandFunc
	PermissionPolicy CommandFunc
	Settings         CommandFunc
	Maintenance      CommandFunc
	Ops              CommandFunc
	Health           CommandFunc
	Events           CommandFunc
}

func Command(name, description string, run CommandFunc) cliapp.Command {
	return cliapp.Command{
		Name:        name,
		NeedsAPI:    true,
		Description: description,
		Run:         run,
	}
}
