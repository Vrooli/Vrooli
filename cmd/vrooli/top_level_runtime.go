package main

import (
	"io"

	"github.com/vrooli/vrooli/internal/cli/topcli"
)

func runTopLevelResourceCommandWithApp(app *App, ctx *commandContext, args []string) error {
	if len(args) == 0 || wantsCommandHelp(args) {
		return runResourceCommandWithApp(app, ctx, nil, args)
	}
	controller, err := app.newResourceController(ctx)
	if err != nil {
		return err
	}
	return runResourceCommandWithApp(app, ctx, controller, args)
}

func runProjectLifecyclePhaseCommandWithApp(app *App, ctx *commandContext, phase string, args []string) error {
	if wantsCommandHelp(args) {
		topcli.RenderProjectLifecycleHelp(ctx.Stdout, phase)
		return nil
	}

	controller, err := app.newProjectController(ctx)
	if err != nil {
		return err
	}
	return controller.RunProjectPhase(phase, args)
}

func wantsCommandHelp(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "--help", "-h", "help":
			return true
		}
	}
	return false
}

func runCleanupCommand(root string, parsed parsedArgs, stdout, stderr io.Writer) error {
	app, ctx := newConfiguredCommandContext(root, parsed.globals, stdout, stderr)
	return runTopLevelCleanupCommand(app, ctx, parsed.args)
}
