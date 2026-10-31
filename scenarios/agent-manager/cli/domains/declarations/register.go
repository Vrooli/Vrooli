package declarations

import (
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title:    "Declarations",
		Commands: []cliapp.Command{support.Command("declarations", "Reconcile a scenario's unified profile + workflow declarations", deps.Declarations)},
	}
}
