package support

import "github.com/vrooli/cli-core/cliapp"

type CommandFunc func(args []string) error

type Dependencies struct {
	Profile     CommandFunc
	Task        CommandFunc
	Run         CommandFunc
	Runner      CommandFunc
	Settings    CommandFunc
	Maintenance CommandFunc
}

func Command(name, description string, run CommandFunc) cliapp.Command {
	return cliapp.Command{
		Name:        name,
		NeedsAPI:    true,
		Description: description,
		Run:         run,
	}
}
