package main

import (
	"io"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
)

func formatFromJSON(jsonFlag bool) (cliout.Format, error) {
	return cliout.ParseFormat("", jsonFlag)
}

func parseOutputFormat(globals globalOptions) (cliout.Format, error) {
	return formatFromJSON(globals.json)
}

func (ctx *commandContext) outputFormat(forceJSON bool) (cliout.Format, error) {
	return formatFromJSON(ctx.Globals.json || forceJSON)
}

func (ctx *commandContext) executionContextForFormat(format cliout.Format) *commandContext {
	execCtx := *ctx
	if format == cliout.FormatJSON {
		execCtx.Stdout = ctx.Stderr
	}
	return &execCtx
}

func (app *App) newScenarioServiceForFormat(ctx *commandContext, forceJSON bool) (*orchestrator.Service, cliout.Format, error) {
	format, err := ctx.outputFormat(forceJSON)
	if err != nil {
		return nil, "", err
	}
	service, err := app.newScenarioService(ctx.executionContextForFormat(format))
	if err != nil {
		return nil, "", err
	}
	return service, format, nil
}

func (app *App) newScenarioLifecycleRunnerForFormat(ctx *commandContext, forceJSON bool) (*lifecycle.Runner, cliout.Format, error) {
	format, err := ctx.outputFormat(forceJSON)
	if err != nil {
		return nil, "", err
	}
	runner, err := app.newScenarioLifecycleRunner(ctx.executionContextForFormat(format))
	if err != nil {
		return nil, "", err
	}
	return runner, format, nil
}

func writeSuccessData(w io.Writer, key string, value any) error {
	return cliout.WriteSuccessJSON(w, key, value)
}
