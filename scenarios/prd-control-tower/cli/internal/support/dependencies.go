package support

import "github.com/vrooli/cli-core/cliapp"

type CommandFunc func(args []string) error

type Dependencies struct {
	ListDrafts   CommandFunc
	PRD          CommandFunc
	Requirements CommandFunc
}

func Command(name, description string, run CommandFunc) cliapp.Command {
	return cliapp.Command{
		Name:        name,
		NeedsAPI:    true,
		Description: description,
		Run:         run,
	}
}
