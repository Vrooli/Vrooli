package main

import (
	"fmt"
	"io"

	scenarioapp "github.com/vrooli/vrooli/internal/app/scenario"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
)

func runScenarioStartRequest(app *App, ctx *commandContext, req scenariocli.StartRequest) (cliout.Format, []scenariocli.LifecycleItemOutput, error) {
	service, format, err := app.newScenarioServiceForFormat(ctx, req.JSON)
	if err != nil {
		return "", nil, err
	}
	command := scenarioapp.Service{Scenarios: service, OpenURL: app.openScenarioURL, Format: format}
	items, err := command.Start(req)
	return command.Format, items, err
}

func runScenarioValidateEnvRequest(app *App, ctx *commandContext, req scenariocli.ValidateEnvRequest) (cliout.Format, scenariocli.ValidateEnvResponse, error) {
	format, err := formatFromJSON(req.JSON)
	if err != nil {
		return "", scenariocli.ValidateEnvResponse{}, err
	}
	validator, err := app.newResourceController(ctx)
	if err != nil {
		return "", scenariocli.ValidateEnvResponse{}, err
	}
	command := scenarioapp.Service{Validator: validator, Format: format}
	resp, err := command.ValidateEnv(req)
	return command.Format, resp, err
}

func runScenarioStopRequest(app *App, ctx *commandContext, req scenariocli.StopRequest) (cliout.Format, []scenariocli.LifecycleItemOutput, error) {
	runner, format, err := app.newScenarioLifecycleRunnerForFormat(ctx, req.JSON)
	if err != nil {
		return "", nil, err
	}
	command := scenarioapp.Service{Runner: runner, Format: format}
	items, err := command.Stop(req)
	return command.Format, items, err
}

func runScenarioRestartRequest(app *App, ctx *commandContext, req scenariocli.RestartRequest) (cliout.Format, []scenariocli.LifecycleItemOutput, error) {
	service, format, err := app.newScenarioServiceForFormat(ctx, req.JSON)
	if err != nil {
		return "", nil, err
	}
	command := scenarioapp.Service{Scenarios: service, OpenURL: app.openScenarioURL, Format: format}
	items, err := command.Restart(req)
	return command.Format, items, err
}

func runScenarioListRequest(app *App, ctx *commandContext, req scenariocli.ListRequest) (cliout.Format, scenariocli.ListResponse, error) {
	format, err := ctx.outputFormat(req.JSON)
	if err != nil {
		return "", scenariocli.ListResponse{}, err
	}
	service, err := app.newQuietScenarioService(ctx)
	if err != nil {
		return "", scenariocli.ListResponse{}, err
	}
	command := scenarioapp.Service{Scenarios: service, Format: format}
	resp, err := command.List(req)
	return command.Format, resp, err
}

func runScenarioInfoRequest(app *App, ctx *commandContext, req scenariocli.InfoRequest) (cliout.Format, scenariocli.InfoOutput, error) {
	format, err := ctx.outputFormat(req.JSON)
	if err != nil {
		return "", scenariocli.InfoOutput{}, err
	}
	service, err := app.newQuietScenarioService(ctx)
	if err != nil {
		return "", scenariocli.InfoOutput{}, err
	}
	command := scenarioapp.Service{Scenarios: service, Format: format}
	resp, err := command.Info(req)
	return command.Format, resp, err
}

func runScenarioStatusRequest(app *App, ctx *commandContext, req scenariocli.StatusRequest) (cliout.Format, scenariocli.StatusResponse, error) {
	format, err := ctx.outputFormat(req.JSON)
	if err != nil {
		return "", scenariocli.StatusResponse{}, err
	}
	service, err := app.newQuietScenarioService(ctx)
	if err != nil {
		return "", scenariocli.StatusResponse{}, err
	}
	command := scenarioapp.Service{Scenarios: service, Format: format}
	resp, err := command.Status(req)
	return command.Format, resp, err
}

func (app *App) newQuietScenarioService(ctx *commandContext) (*orchestrator.Service, error) {
	quietCtx := *ctx
	quietCtx.Stdout = io.Discard
	quietCtx.Stderr = io.Discard
	return app.newScenarioService(&quietCtx)
}

func runScenarioSetupRequest(app *App, ctx *commandContext, req scenariocli.SetupRequest) (cliout.Format, lifecycle.PhaseResult, error) {
	runner, format, err := app.newScenarioLifecycleRunnerForFormat(ctx, req.JSON)
	if err != nil {
		return "", lifecycle.PhaseResult{}, err
	}
	command := scenarioapp.Service{Runner: runner, Format: format}
	result, err := command.Setup(req)
	return command.Format, result, err
}

func runScenarioTestRequest(app *App, ctx *commandContext, req scenariocli.TestRequest) (cliout.Format, struct{}, error) {
	runner, _, err := app.newScenarioLifecycleRunnerForFormat(ctx, false)
	if err != nil {
		return "", struct{}{}, err
	}
	command := scenarioapp.Service{Runner: runner, Format: cliout.FormatHuman}
	return cliout.FormatHuman, struct{}{}, command.Test(req)
}

func runScenarioStartAllRequest(app *App, ctx *commandContext, req scenariocli.StartAllRequest) (cliout.Format, scenariocli.BatchResponse, error) {
	service, format, err := app.newScenarioServiceForFormat(ctx, req.JSON)
	if err != nil {
		return "", scenariocli.BatchResponse{}, err
	}
	command := scenarioapp.Service{Scenarios: service, Format: format}
	resp, err := command.StartAll()
	return command.Format, resp, err
}

func runScenarioStopAllRequest(app *App, ctx *commandContext, req scenariocli.StopAllRequest) (cliout.Format, scenariocli.BatchResponse, error) {
	service, format, err := app.newScenarioServiceForFormat(ctx, req.JSON)
	if err != nil {
		return "", scenariocli.BatchResponse{}, err
	}
	command := scenarioapp.Service{Scenarios: service, Format: format}
	resp, err := command.StopAll()
	return command.Format, resp, err
}

func runScenarioPortRequest(app *App, ctx *commandContext, req scenariocli.PortRequest) (cliout.Format, scenariocli.PortResponse, error) {
	format, err := formatFromJSON(req.JSON)
	if err != nil {
		return "", scenariocli.PortResponse{}, err
	}
	service, err := app.newQuietScenarioService(ctx)
	if err != nil {
		return "", scenariocli.PortResponse{}, err
	}
	command := scenarioapp.Service{Scenarios: service, Format: format}
	resp, err := command.Port(req)
	if err != nil {
		if req.PortName == "" {
			return "", scenariocli.PortResponse{}, runtimeErrorf("Start the scenario before querying runtime ports", err.Error())
		}
		return "", scenariocli.PortResponse{}, runtimeErrorf("Inspect the scenario status or start the scenario first", err.Error())
	}
	return command.Format, resp, nil
}

func runScenarioOpenRequest(app *App, ctx *commandContext, req scenariocli.OpenRequest) (cliout.Format, scenariocli.OpenOutput, error) {
	format, err := formatFromJSON(req.JSON)
	if err != nil {
		return "", scenariocli.OpenOutput{}, err
	}
	service, err := app.newQuietScenarioService(ctx)
	if err != nil {
		return "", scenariocli.OpenOutput{}, err
	}
	command := scenarioapp.Service{Scenarios: service, OpenURL: app.openScenarioURL, Format: format}
	resp, err := command.Open(req)
	if err != nil {
		return "", scenariocli.OpenOutput{}, err
	}
	if !req.PrintURL && !req.JSON {
		resolved, resolveErr := command.Scenarios.ResolvePort(req.ScenarioName, req.PortName)
		if resolveErr != nil {
			return "", scenariocli.OpenOutput{}, resolveErr
		}
		_, _ = fmt.Fprintf(ctx.Stderr, "Opening %s at %s\n", req.ScenarioName, resolved.URL)
		return cliout.FormatHuman, scenariocli.OpenOutput{}, nil
	}
	return command.Format, resp, nil
}

func renderScenarioLifecycleResponse(w io.Writer, format cliout.Format, items []scenariocli.LifecycleItemOutput) error {
	return scenariocli.WriteLifecycleItems(w, format, items)
}

func renderScenarioListResponse(w io.Writer, format cliout.Format, resp scenariocli.ListResponse) error {
	return scenariocli.RenderListResponse(w, format, resp)
}

func renderScenarioInfoResponse(w io.Writer, format cliout.Format, resp scenariocli.InfoOutput) error {
	return scenariocli.RenderInfoResponse(w, format, resp)
}

func renderScenarioStatusResponse(w io.Writer, format cliout.Format, resp scenariocli.StatusResponse) error {
	return scenariocli.RenderStatusResponse(w, format, resp)
}

func renderScenarioSetupResponse(w io.Writer, format cliout.Format, result lifecycle.PhaseResult) error {
	return scenariocli.RenderSetupResponse(w, format, result)
}
func renderScenarioTestResponse(w io.Writer, format cliout.Format, _ struct{}) error { return nil }
func renderScenarioBatchResponse(w io.Writer, format cliout.Format, resp scenariocli.BatchResponse) error {
	return scenariocli.WriteBatchReport(w, format, resp)
}

func renderScenarioPortResponse(w io.Writer, format cliout.Format, resp scenariocli.PortResponse) error {
	return scenariocli.RenderPortResponse(w, format, resp)
}

func renderScenarioOpenResponse(w io.Writer, _ cliout.Format, resp scenariocli.OpenOutput) error {
	return scenariocli.RenderOpenResponse(w, resp)
}
