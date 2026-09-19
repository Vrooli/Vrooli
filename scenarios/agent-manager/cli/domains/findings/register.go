package findings

import (
	"agent-manager/cli/internal/support"
	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{Title: "Findings", Commands: []cliapp.Command{support.Command("findings", "List recurring investigation findings", deps.Findings)}}
}
