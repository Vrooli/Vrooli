package workflows

import (
	"fmt"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func Commands(ctx *appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Workflows",
		Commands: []cliapp.Command{
			{
				Name:        "workflow",
				NeedsAPI:    true,
				Description: "Manage workflows (create, execute, list, lint, delete, versions)",
				Run: func(args []string) error {
					return runWorkflow(ctx, args)
				},
			},
		},
	}
}

func runWorkflow(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s workflow <create|execute|list|lint|delete|versions>", ctx.Name)
	}

	subcommand := args[0]
	switch subcommand {
	case "create":
		return runCreate(ctx, args[1:])
	case "execute":
		return runExecute(ctx, args[1:])
	case "list":
		return runList(ctx, args[1:])
	case "lint":
		return runLint(ctx, args[1:])
	case "delete":
		return runDelete(ctx, args[1:])
	case "versions":
		return runVersions(ctx, args[1:])
	default:
		return fmt.Errorf("unknown workflow command: %s", subcommand)
	}
}
