package workflows

import (
	"browser-automation-studio/cli/internal/appctx"
	"fmt"

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

func printWorkflowHelp(cliName string) {
	fmt.Printf("Usage: %s workflow <subcommand> [options]\n\n", cliName)
	fmt.Println("Manage and execute browser automation workflows.")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  create    Create a new workflow")
	fmt.Println("  execute   Execute a workflow (by ID, name, file, or inline steps)")
	fmt.Println("  list      List all workflows")
	fmt.Println("  lint      Validate workflow JSON syntax")
	fmt.Println("  delete    Delete a workflow")
	fmt.Println("  versions  Show workflow version history")
	fmt.Println()
	fmt.Printf("Use '%s workflow <subcommand> --help' for subcommand details.\n", cliName)
	fmt.Println()
}

func runWorkflow(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		printWorkflowHelp(ctx.Name)
		return fmt.Errorf("subcommand required")
	}

	subcommand := args[0]
	switch subcommand {
	case "--help", "-h":
		printWorkflowHelp(ctx.Name)
		return nil
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
