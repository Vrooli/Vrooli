package main

func (app *App) runTopLevelBuildCommand(ctx *commandContext, args []string) error {
	return runProjectLifecyclePhaseCommandWithApp(app, ctx, "build", args)
}

func (app *App) runTopLevelDeployCommand(ctx *commandContext, args []string) error {
	return runProjectLifecyclePhaseCommandWithApp(app, ctx, "deploy", args)
}

func (app *App) runTopLevelCleanCommand(ctx *commandContext, args []string) error {
	return runProjectLifecyclePhaseCommandWithApp(app, ctx, "clean", args)
}

func (app *App) runTopLevelBackupCommand(ctx *commandContext, args []string) error {
	return runProjectLifecyclePhaseCommandWithApp(app, ctx, "backup", args)
}

func (app *App) runTopLevelRestoreCommand(ctx *commandContext, args []string) error {
	return runProjectLifecyclePhaseCommandWithApp(app, ctx, "restore", args)
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
	return runTopLevelStatusCommandWithApp(app, ctx, args)
}

func (app *App) runTopLevelDoctorCommand(ctx *commandContext, args []string) error {
	return runTopLevelDoctorCommandWithApp(app, ctx, args)
}

func (app *App) runTopLevelStopCommand(ctx *commandContext, args []string) error {
	return runTopLevelStopCommandWithApp(app, ctx, args)
}

func (app *App) runTopLevelResourceCommand(ctx *commandContext, args []string) error {
	return runTopLevelResourceCommandWithApp(app, ctx, args)
}

func (app *App) runTopLevelOrphansCommand(ctx *commandContext, args []string) error {
	return runTopLevelOrphansCommandWithApp(app, ctx, args)
}

func (app *App) runTopLevelLocksCommand(ctx *commandContext, args []string) error {
	return runTopLevelLocksCommandWithApp(app, ctx, args)
}

func (app *App) runTopLevelDiagnosePortCommand(ctx *commandContext, args []string) error {
	return runTopLevelDiagnosePortCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioStartCommand(ctx *commandContext, args []string) error {
	return runScenarioStartCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioStopCommand(ctx *commandContext, args []string) error {
	return runScenarioStopCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioRestartCommand(ctx *commandContext, args []string) error {
	return runScenarioRestartCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioListCommand(ctx *commandContext, args []string) error {
	return runScenarioListCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioInfoCommand(ctx *commandContext, args []string) error {
	return runScenarioInfoCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioStatusCommand(ctx *commandContext, args []string) error {
	return runScenarioStatusCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioRunCommand(ctx *commandContext, args []string) error {
	return runScenarioStartCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioSetupCommand(ctx *commandContext, args []string) error {
	return runScenarioSetupCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioStartAllCommand(ctx *commandContext, args []string) error {
	return runScenarioStartAllCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioStopAllCommand(ctx *commandContext, args []string) error {
	return runScenarioStopAllCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioTestCommand(ctx *commandContext, args []string) error {
	return runScenarioTestCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioLogsCommand(ctx *commandContext, args []string) error {
	return runScenarioLogsCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioOpenCommand(ctx *commandContext, args []string) error {
	return runScenarioOpenCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioPortCommand(ctx *commandContext, args []string) error {
	return runScenarioPortCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioUISmokeCommand(ctx *commandContext, args []string) error {
	return runScenarioUISmokeCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioRequirementsCommand(ctx *commandContext, args []string) error {
	return runScenarioRequirementsCommandWithApp(app, ctx, args)
}

func (app *App) runScenarioTemplateCommand(ctx *commandContext, args []string) error {
	return runScenarioTemplateCommandWithApp(app, ctx, args)
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
