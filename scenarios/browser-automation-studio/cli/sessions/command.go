package sessions

import (
	"browser-automation-studio/cli/internal/appctx"
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

func Commands(ctx *appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Sessions",
		Commands: []cliapp.Command{
			{
				Name:        "session",
				NeedsAPI:    true,
				Description: "Manage session profiles (list, create, show, rename, delete, clear-storage)",
				Run: func(args []string) error {
					return runSession(ctx, args)
				},
			},
		},
	}
}

func printSessionHelp(cliName string) {
	fmt.Printf("Usage: %s session <subcommand> [options]\n\n", cliName)
	fmt.Println("Manage browser session profiles for reusing authentication state across workflow executions.")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  list                      List all session profiles")
	fmt.Println("  create [name]             Create a new session profile")
	fmt.Println("  show <id|name>            Show profile details and storage state")
	fmt.Println("  rename <id|name> <new>    Rename a session profile")
	fmt.Println("  delete <id|name>          Delete a session profile")
	fmt.Println("  clear-storage <id|name>   Clear storage state (cookies, localStorage)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --json    Output in JSON format (supported by all subcommands)")
	fmt.Println("  --force   Skip confirmation prompt (delete only)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s session create \"Dev Account\"\n", cliName)
	fmt.Printf("  %s session show \"Dev Account\"\n", cliName)
	fmt.Printf("  %s session rename \"Dev Account\" \"Production Account\"\n", cliName)
	fmt.Printf("  %s session list --json\n", cliName)
	fmt.Printf("  %s session delete \"Dev Account\" --force\n", cliName)
	fmt.Println()
}

func runSession(ctx *appctx.Context, args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printSessionHelp(ctx.Name)
		if len(args) == 0 {
			return fmt.Errorf("subcommand required")
		}
		return nil
	}

	subcommand := args[0]
	switch subcommand {
	case "list":
		return runList(ctx, args[1:])
	case "create":
		return runCreate(ctx, args[1:])
	case "show":
		return runShow(ctx, args[1:])
	case "rename":
		return runRename(ctx, args[1:])
	case "delete":
		return runDelete(ctx, args[1:])
	case "clear-storage":
		return runClearStorage(ctx, args[1:])
	default:
		return fmt.Errorf("unknown session command: %s", subcommand)
	}
}
