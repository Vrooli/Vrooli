package recordings

import (
	"fmt"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func Commands(ctx *appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Recordings",
		Commands: []cliapp.Command{
			{
				Name:        "recording",
				NeedsAPI:    true,
				Description: "Manage recordings (import)",
				Run: func(args []string) error {
					return runRecording(ctx, args)
				},
			},
		},
	}
}

func runRecording(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("recording command requires a subcommand")
	}

	subcommand := args[0]
	switch subcommand {
	case "import":
		return runImport(ctx, args[1:])
	default:
		return fmt.Errorf("unknown recording command: %s", subcommand)
	}
}
