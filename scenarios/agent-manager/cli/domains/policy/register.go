package policy

import (
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Model Policy",
		Commands: []cliapp.Command{
			support.Command("policy", "Inspect, validate, reload, and explain model policy", deps.Policy),
		},
	}
}
