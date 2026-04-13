package main

func runTopLevelSetupCommand(app *App, ctx *commandContext, args []string) error {
	return app.runTopLevelSetup(ctx, args)
}

func runTopLevelDevelopCommand(app *App, ctx *commandContext, args []string) error {
	return app.runTopLevelDevelop(ctx, args)
}

func runTopLevelBuildCommand(app *App, ctx *commandContext, args []string) error {
	return runProjectLifecyclePhaseCommandWithApp(app, ctx, "build", args)
}

func runTopLevelDeployCommand(app *App, ctx *commandContext, args []string) error {
	return runProjectLifecyclePhaseCommandWithApp(app, ctx, "deploy", args)
}

func runTopLevelCleanCommand(app *App, ctx *commandContext, args []string) error {
	return runProjectLifecyclePhaseCommandWithApp(app, ctx, "clean", args)
}

func runTopLevelBackupCommand(app *App, ctx *commandContext, args []string) error {
	return runProjectLifecyclePhaseCommandWithApp(app, ctx, "backup", args)
}

func runTopLevelRestoreCommand(app *App, ctx *commandContext, args []string) error {
	return runProjectLifecyclePhaseCommandWithApp(app, ctx, "restore", args)
}

func runInfoTopLevelCommand(app *App, ctx *commandContext, args []string) error {
	return runInfoCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func runScenarioRootCommand(app *App, ctx *commandContext, args []string) error {
	if len(args) == 0 {
		showScenarioHelp(ctx.Stdout)
		return nil
	}
	if wantsCommandHelp(args) {
		showScenarioHelp(ctx.Stdout)
		return nil
	}
	handler, ok := scenarioCommands[normalizeSubcommand(args[0])]
	if !ok {
		return newUnknownScenarioCommandError(args[0])
	}
	return handler(app, ctx, args[1:])
}

func runTopLevelCleanupCommand(app *App, ctx *commandContext, args []string) error {
	return runCleanupCommandWithApp(app, ctx, parsedArgs{globals: ctx.Globals, args: args})
}

func runScenarioRunCommand(app *App, ctx *commandContext, args []string) error {
	return bindGlobalCommand(parseScenarioStartRequest, runScenarioStartRequest, renderScenarioLifecycleResponse)(app, ctx, args)
}
