package main

func (app *App) runTopLevelBuildCommand(ctx *commandContext, args []string) error {
	return runTopLevelBuildCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelDeployCommand(ctx *commandContext, args []string) error {
	return runTopLevelDeployCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelCleanCommand(ctx *commandContext, args []string) error {
	return runTopLevelCleanCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelBackupCommand(ctx *commandContext, args []string) error {
	return runTopLevelBackupCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelRestoreCommand(ctx *commandContext, args []string) error {
	return runTopLevelRestoreCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runInfoCommand(ctx *commandContext, args []string) error {
	return runInfoCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runScenarioCommand(ctx *commandContext, args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		showScenarioHelp(ctx.Stdout)
		return nil
	}

	handler, ok := scenarioCommands[args[0]]
	if !ok {
		return newUnknownScenarioCommandError(args[0])
	}
	return handler(app, ctx, args[1:])
}

func (app *App) runTopLevelStatusCommand(ctx *commandContext, args []string) error {
	return runTopLevelStatusCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelDoctorCommand(ctx *commandContext, args []string) error {
	return runTopLevelDoctorCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelStopCommand(ctx *commandContext, args []string) error {
	return runTopLevelStopCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelResourceCommand(ctx *commandContext, args []string) error {
	return runTopLevelResourceCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelOrphansCommand(ctx *commandContext, args []string) error {
	return runTopLevelOrphansCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelLocksCommand(ctx *commandContext, args []string) error {
	return runTopLevelLocksCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelDiagnosePortCommand(ctx *commandContext, args []string) error {
	return runTopLevelDiagnosePortCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runScenarioStartCommand(ctx *commandContext, args []string) error {
	return runScenarioStartCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioStopCommand(ctx *commandContext, args []string) error {
	return runScenarioStopCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runScenarioRestartCommand(ctx *commandContext, args []string) error {
	return runScenarioRestartCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioListCommand(ctx *commandContext, args []string) error {
	return runScenarioListCommand(ctx.Root, ctx.Globals, args, ctx.Stdout)
}

func (app *App) runScenarioInfoCommand(ctx *commandContext, args []string) error {
	return runScenarioInfoCommand(ctx.Root, ctx.Globals, args, ctx.Stdout)
}

func (app *App) runScenarioStatusCommand(ctx *commandContext, args []string) error {
	return runScenarioStatusCommand(ctx.Root, ctx.Globals, args, ctx.Stdout)
}

func (app *App) runScenarioRunCommand(ctx *commandContext, args []string) error {
	return runScenarioRunCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runScenarioSetupCommand(ctx *commandContext, args []string) error {
	return runScenarioSetupCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runScenarioStartAllCommand(ctx *commandContext, args []string) error {
	return runScenarioStartAllCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runScenarioStopAllCommand(ctx *commandContext, args []string) error {
	return runScenarioStopAllCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runScenarioTestCommand(ctx *commandContext, args []string) error {
	return runScenarioTestCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runScenarioLogsCommand(ctx *commandContext, args []string) error {
	return runScenarioLogsCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runScenarioOpenCommand(ctx *commandContext, args []string) error {
	return runScenarioOpenCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioPortCommand(ctx *commandContext, args []string) error {
	return runScenarioPortCommand(ctx.Root, ctx.Globals, args, ctx.Stdout)
}

func (app *App) runScenarioUISmokeCommand(ctx *commandContext, args []string) error {
	return runScenarioUISmokeCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioRequirementsCommand(ctx *commandContext, args []string) error {
	return runScenarioRequirementsCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioTemplateCommand(ctx *commandContext, args []string) error {
	return runScenarioTemplateCommand(ctx.Root, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func (app *App) runScenarioGenerateCommand(ctx *commandContext, args []string) error {
	return runScenarioGenerateCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioCompletenessCommand(ctx *commandContext, args []string) error {
	return runScenarioCompletenessCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioHealFromSandboxCommand(ctx *commandContext, args []string) error {
	return runScenarioHealFromSandboxCommandWithApp(app, ctx, args)
}
