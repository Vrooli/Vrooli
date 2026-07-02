package playbooks

import (
	"fmt"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func Commands(ctx *appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Playbooks",
		Commands: []cliapp.Command{
			{
				Name:        "playbooks",
				NeedsAPI:    false,
				Description: "Manage playbook registries (order, scaffold, verify)",
				Run: func(args []string) error {
					return runPlaybooks(ctx, args)
				},
			},
		},
	}
}

func runPlaybooks(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s playbooks <order|scaffold|verify>", ctx.Name)
	}

	subcommand := args[0]
	switch subcommand {
	case "order":
		return runOrder(ctx, args[1:])
	case "scaffold":
		return runScaffold(ctx, args[1:])
	case "verify":
		return runVerify(ctx, args[1:])
	default:
		return fmt.Errorf("unknown playbooks command: %s", subcommand)
	}
}
