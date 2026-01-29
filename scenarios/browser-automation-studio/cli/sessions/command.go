package sessions

import (
	"fmt"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func Commands(ctx *appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Sessions",
		Commands: []cliapp.Command{
			{
				Name:        "session",
				NeedsAPI:    true,
				Description: "Manage session profiles (list, create, show, delete, clear-storage)",
				Run: func(args []string) error {
					return runSession(ctx, args)
				},
			},
		},
	}
}

func runSession(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s session <list|create|show|delete|clear-storage>", ctx.Name)
	}

	subcommand := args[0]
	switch subcommand {
	case "list":
		return runList(ctx, args[1:])
	case "create":
		return runCreate(ctx, args[1:])
	case "show":
		return runShow(ctx, args[1:])
	case "delete":
		return runDelete(ctx, args[1:])
	case "clear-storage":
		return runClearStorage(ctx, args[1:])
	default:
		return fmt.Errorf("unknown session command: %s", subcommand)
	}
}
