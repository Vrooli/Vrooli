package main

type scenarioExternalCommandSpec struct {
	name string
	args []string
	dir  string
}

func (app *App) runScenarioExternalCommand(ctx *commandContext, spec scenarioExternalCommandSpec) error {
	return app.runScenarioSubprocess(scenarioSubprocessSpec{
		name:   spec.name,
		args:   spec.args,
		dir:    spec.dir,
		env:    app.commandEnv(ctx.Root, ctx.Globals),
		stdout: ctx.Stdout,
		stderr: ctx.Stderr,
	})
}

func (app *App) buildScenarioTestGenieSpec(ctx *commandContext, args []string) (scenarioExternalCommandSpec, error) {
	home, err := ctx.HomeDir()
	if err != nil {
		return scenarioExternalCommandSpec{}, err
	}
	cliPath, err := app.locateTestGenieCLI(ctx.Root, home)
	if err != nil {
		return scenarioExternalCommandSpec{}, err
	}
	return scenarioExternalCommandSpec{
		name: cliPath,
		args: args,
		dir:  ctx.Root,
	}, nil
}

func (app *App) runScenarioTestGenieCommand(ctx *commandContext, args []string) error {
	spec, err := app.buildScenarioTestGenieSpec(ctx, args)
	if err != nil {
		return err
	}
	return app.runScenarioExternalCommand(ctx, spec)
}

func (app *App) buildScenarioCompletenessSpec(ctx *commandContext, args []string) (scenarioExternalCommandSpec, error) {
	cliPath, err := app.locateScenarioCompletenessCLI(ctx.Root)
	if err != nil {
		return scenarioExternalCommandSpec{}, err
	}
	return scenarioExternalCommandSpec{
		name: cliPath,
		args: args,
		dir:  ctx.Root,
	}, nil
}

func (app *App) runScenarioCompletenessSubprocessCommand(ctx *commandContext, args []string) error {
	spec, err := app.buildScenarioCompletenessSpec(ctx, args)
	if err != nil {
		return err
	}
	return app.runScenarioExternalCommand(ctx, spec)
}

func buildUISmokeArgs(globals globalOptions, args []string) []string {
	commandArgs := []string{"ui-smoke"}
	commandArgs = append(commandArgs, args...)
	return append(commandArgs, passthroughFlags(globals, commandArgs)...)
}

func buildScenarioCompletenessArgs(globals globalOptions, args []string) []string {
	commandArgs := append([]string{}, args...)
	if globals.json && !containsArg(commandArgs, "--json") && !containsArg(commandArgs, "--format") {
		commandArgs = append(commandArgs, "--json")
	}
	return commandArgs
}
