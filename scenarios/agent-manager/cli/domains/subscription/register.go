package subscription

import (
	"agent-manager/cli/internal/support"
	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{Title: "Subscription", Commands: []cliapp.Command{support.Command("subscription", "Record and inspect subscription billing periods", deps.Subscription)}}
}
