package executions

import (
	"fmt"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func Commands(ctx *appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Executions",
		Commands: []cliapp.Command{
			{
				Name:        "execution",
				NeedsAPI:    true,
				Description: "Manage executions (watch, list, stop, export, render)",
				Run: func(args []string) error {
					return runExecution(ctx, args)
				},
			},
		},
	}
}

func runExecution(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s execution <watch|list|stop|export|render|render-video>", ctx.Name)
	}

	subcommand := args[0]
	switch subcommand {
	case "watch":
		return runWatch(ctx, args[1:])
	case "list":
		return runList(ctx, args[1:])
	case "stop":
		return runStop(ctx, args[1:])
	case "export":
		return runExport(ctx, args[1:])
	case "render":
		return runRender(ctx, args[1:])
	case "render-video":
		return runRenderVideo(ctx, args[1:])
	default:
		return fmt.Errorf("unknown execution command: %s", subcommand)
	}
}
