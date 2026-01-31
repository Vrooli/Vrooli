package ai

import (
	"fmt"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

// Commands returns the AI command group.
func Commands(ctx *appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "AI",
		Commands: []cliapp.Command{
			{
				Name:        "ai",
				NeedsAPI:    true,
				Description: "AI navigation and automation (navigators, navigate)",
				Run: func(args []string) error {
					return runAI(ctx, args)
				},
			},
		},
	}
}

func printAIHelp(cliName string) {
	fmt.Printf("Usage: %s ai <subcommand> [options]\n\n", cliName)
	fmt.Println("AI-powered browser navigation and automation.")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  navigators    List available AI navigation backends")
	fmt.Println("  navigate      Start AI-driven browser navigation")
	fmt.Println()
	fmt.Printf("Use '%s ai <subcommand> --help' for subcommand details.\n", cliName)
	fmt.Println()
}

func runAI(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		printAIHelp(ctx.Name)
		return fmt.Errorf("subcommand required")
	}

	subcommand := args[0]
	switch subcommand {
	case "--help", "-h":
		printAIHelp(ctx.Name)
		return nil
	case "navigators":
		return runNavigators(ctx, args[1:])
	case "navigate":
		return runNavigate(ctx, args[1:])
	default:
		return fmt.Errorf("unknown ai command: %s", subcommand)
	}
}
