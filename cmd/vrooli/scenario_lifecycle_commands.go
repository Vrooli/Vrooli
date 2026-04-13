package main

import (
	"fmt"
	"io"
	"time"

	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
)

func runScenarioUISmokeCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return app.runScenarioTestGenieCommand(ctx, buildUISmokeArgs(ctx.Globals, args))
}

func runScenarioCompletenessCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return app.runScenarioCompletenessSubprocessCommand(ctx, buildScenarioCompletenessArgs(ctx.Globals, args))
}

func runScenarioRequirementsRequest(app *App, ctx *commandContext, req scenariocli.RequirementsRequest) (cliout.Format, struct{}, error) {
	if req.Snapshot {
		return cliout.FormatHuman, struct{}{}, runScenarioRequirementsSnapshot(ctx.Root, req.Args[1:], ctx.Stdout)
	}

	home, err := ctx.HomeDir()
	if err != nil {
		return "", struct{}{}, err
	}
	cliPath, err := app.locateTestGenieCLI(ctx.Root, home)
	if err != nil {
		return "", struct{}{}, err
	}

	commandArgs, workdir, err := translateScenarioRequirementsArgs(ctx.Root, ctx.Globals, req.Args)
	if err != nil {
		return "", struct{}{}, err
	}
	if err := app.runScenarioExternalCommand(ctx, scenarioExternalCommandSpec{
		name: cliPath,
		args: commandArgs,
		dir:  workdir,
	}); err != nil {
		return "", struct{}{}, err
	}
	return cliout.FormatHuman, struct{}{}, nil
}

func runScenarioHealFromSandboxRequest(app *App, ctx *commandContext, req scenariocli.HealFromSandboxRequest) (cliout.Format, scenariocli.HealFromSandboxResponse, error) {
	home, err := ctx.HomeDir()
	if err != nil {
		return "", scenariocli.HealFromSandboxResponse{}, err
	}
	affected, err := orchestrator.SandboxAffectedScenarios(home, req.MergedPath)
	if err != nil {
		return "", scenariocli.HealFromSandboxResponse{}, err
	}
	resp := scenariocli.HealFromSandboxResponse{
		Affected: append([]string(nil), affected...),
		DryRun:   req.DryRun,
	}
	if len(affected) == 0 || req.DryRun {
		return cliout.FormatHuman, resp, nil
	}

	runner, err := app.newScenarioLifecycleRunner(ctx)
	if err != nil {
		return "", scenariocli.HealFromSandboxResponse{}, err
	}
	for _, name := range affected {
		if stopErr := runner.Stop(name, lifecycle.StopOptions{}); stopErr != nil {
			_, _ = fmt.Fprintf(ctx.Stderr, "heal-from-sandbox: stop %s failed: %v\n", name, stopErr)
		}
	}
	time.Sleep(1 * time.Second)
	for _, name := range affected {
		if startErr := app.launchDetachedScenario(ctx.Root, ctx.Globals, "start", name); startErr != nil {
			return "", scenariocli.HealFromSandboxResponse{}, startErr
		}
		resp.StoppedCount++
	}
	return cliout.FormatHuman, resp, nil
}

func showScenarioRequirementsHelp(w io.Writer) {
	_, _ = fmt.Fprint(w, scenariocli.RequirementsHelpText())
}
