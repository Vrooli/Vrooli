package workflows

import (
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{Title: "Workflows", Commands: []cliapp.Command{support.Command("workflow", "Validate and reconcile scenario workflows", deps.Workflow)}}
}
