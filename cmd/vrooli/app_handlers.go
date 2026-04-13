package main

import (
	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	projectapp "github.com/vrooli/vrooli/internal/app/project"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/contractcli"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cli/topcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/project"
)

func runTopLevelSetupCommand(app *App, ctx *commandContext, args []string) error {
	return app.runTopLevelSetup(ctx, args)
}

func runTopLevelDevelopCommand(app *App, ctx *commandContext, args []string) error {
	return app.runTopLevelDevelop(ctx, args)
}

func runTopLevelBuildCommand(app *App, ctx *commandContext, args []string) error {
	return app.runTopLevelBuild(ctx, args)
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
	req, err := topcli.ParseInfoRequest(args)
	if err != nil {
		return err
	}
	format, err := formatFromJSON(ctx.Globals.json)
	if err != nil {
		return err
	}
	return topcli.RunInfo(ctx.Root, format, req, ctx.Stdout, ctx.Stderr)
}

func runScenarioRootCommand(app *App, ctx *commandContext, args []string) error {
	if len(args) == 0 {
		scenariocli.RenderCommandHelp(ctx.Stdout)
		return nil
	}
	if wantsCommandHelp(args) {
		scenariocli.RenderCommandHelp(ctx.Stdout)
		return nil
	}
	handler, ok := scenarioCommands[commandtree.NormalizeName(args[0])]
	if !ok {
		return newUnknownScenarioCommandError(args[0])
	}
	return handler(app, ctx, args[1:])
}

func runTopLevelCleanupCommand(app *App, ctx *commandContext, args []string) error {
	req, err := topcli.ParseCleanupRequest(args)
	if err != nil {
		if helpErr, ok := err.(interface{ HelpText() string }); ok {
			if helpErr.HelpText() == "" {
				topcli.RenderCleanupHelp(ctx.Stdout)
				return nil
			}
			_, _ = ctx.Stdout.Write([]byte(helpErr.HelpText()))
			return nil
		}
		return err
	}
	switch req.Target {
	case "orphans":
		return runTopLevelOrphansCommand(app, ctx, append([]string{"kill"}, req.Args...))
	case "locks":
		return runTopLevelLocksCommand(app, ctx, append([]string{"clean"}, req.Args...))
	default:
		return nil
	}
}

func runScenarioRunCommand(app *App, ctx *commandContext, args []string) error {
	return bindGlobalCommand(
		func(globals globalOptions, args []string) (scenariocli.StartRequest, error) {
			return scenariocli.ParseStartRequest(globals.json, args)
		},
		runScenarioStartRequest,
		renderScenarioLifecycleResponse,
	)(app, ctx, args)
}

func runTopLevelStatusCommand(app *App, ctx *commandContext, args []string) error {
	return bindGlobalCommand(
		func(globals globalOptions, args []string) (topcli.StatusRequest, error) {
			return topcli.ParseStatusRequest(args)
		},
		func(app *App, ctx *commandContext, req topcli.StatusRequest) (cliout.Format, topcli.StatusResponse, error) {
			command, err := app.newProjectCommandService(ctx)
			if err != nil {
				return "", topcli.StatusResponse{}, err
			}
			report, err := command.Status(projectapp.StatusRequest{
				ResourcesOnly: req.ResourcesOnly,
				ScenariosOnly: req.ScenariosOnly,
				Fast:          req.Fast,
			})
			if err != nil {
				return "", topcli.StatusResponse{}, err
			}
			format, err := parseOutputFormat(ctx.Globals)
			if err != nil {
				return "", topcli.StatusResponse{}, err
			}
			return format, topcli.StatusResponse{Options: req, Report: report}, nil
		},
		topcli.RenderStatusResponse,
	)(app, ctx, args)
}

func runTopLevelDoctorCommand(app *App, ctx *commandContext, args []string) error {
	return bindGlobalCommand(
		func(globals globalOptions, args []string) (topcli.NoArgsRequest, error) {
			return topcli.ParseDoctorRequest(args)
		},
		func(app *App, ctx *commandContext, _ topcli.NoArgsRequest) (cliout.Format, project.DoctorReport, error) {
			command, err := app.newProjectCommandService(ctx)
			if err != nil {
				return "", project.DoctorReport{}, err
			}
			report, err := command.Doctor()
			if err != nil {
				return "", project.DoctorReport{}, err
			}
			format, err := parseOutputFormat(ctx.Globals)
			if err != nil {
				return "", project.DoctorReport{}, err
			}
			return format, report, nil
		},
		topcli.RenderDoctorResponse,
	)(app, ctx, args)
}

func runTopLevelStopCommand(app *App, ctx *commandContext, args []string) error {
	return bindGlobalCommand(
		func(globals globalOptions, args []string) (topcli.StopRequest, error) {
			return topcli.ParseStopRequest(args)
		},
		func(app *App, ctx *commandContext, req topcli.StopRequest) (cliout.Format, control.StopReport, error) {
			command, err := app.newProjectCommandService(ctx)
			if err != nil {
				return "", control.StopReport{}, err
			}
			report, err := command.Stop(projectapp.StopRequest{Targets: req.Targets})
			if err != nil {
				return "", control.StopReport{}, err
			}
			format, err := parseOutputFormat(ctx.Globals)
			if err != nil {
				return "", control.StopReport{}, err
			}
			return format, report, nil
		},
		topcli.RenderStopResponse,
	)(app, ctx, args)
}

func runTopLevelOrphansCommand(app *App, ctx *commandContext, args []string) error {
	return bindGlobalCommand(
		func(globals globalOptions, args []string) (topcli.OrphansRequest, error) {
			return topcli.ParseOrphansRequest(args)
		},
		func(app *App, ctx *commandContext, req topcli.OrphansRequest) (cliout.Format, topcli.OrphansResponse, error) {
			command, err := app.newProjectCommandService(ctx)
			if err != nil {
				return "", topcli.OrphansResponse{}, err
			}
			format, err := parseOutputFormat(ctx.Globals)
			if err != nil {
				return "", topcli.OrphansResponse{}, err
			}
			resp, err := command.Orphans(projectapp.OrphansRequest{Kill: req.Kill})
			if err != nil {
				return "", topcli.OrphansResponse{}, err
			}
			return format, topcli.OrphansResponse{List: resp.List, KillReport: resp.KillReport}, nil
		},
		topcli.RenderOrphansResponse,
	)(app, ctx, args)
}

func runTopLevelLocksCommand(app *App, ctx *commandContext, args []string) error {
	return bindGlobalCommand(
		func(globals globalOptions, args []string) (topcli.LocksRequest, error) {
			return topcli.ParseLocksRequest(args)
		},
		func(app *App, ctx *commandContext, req topcli.LocksRequest) (cliout.Format, topcli.LocksResponse, error) {
			command, err := app.newProjectCommandService(ctx)
			if err != nil {
				return "", topcli.LocksResponse{}, err
			}
			format, err := parseOutputFormat(ctx.Globals)
			if err != nil {
				return "", topcli.LocksResponse{}, err
			}
			resp, err := command.Locks(projectapp.LocksRequest{Clean: req.Clean})
			if err != nil {
				return "", topcli.LocksResponse{}, err
			}
			return format, topcli.LocksResponse{List: resp.List, CleanReport: resp.CleanReport}, nil
		},
		topcli.RenderLocksResponse,
	)(app, ctx, args)
}

func runTopLevelDiagnosePortCommand(app *App, ctx *commandContext, args []string) error {
	return bindGlobalCommand(
		func(globals globalOptions, args []string) (topcli.DiagnosePortRequest, error) {
			return topcli.ParseDiagnosePortRequest(args)
		},
		func(app *App, ctx *commandContext, req topcli.DiagnosePortRequest) (cliout.Format, maintenance.PortDiagnostic, error) {
			command, err := app.newProjectCommandService(ctx)
			if err != nil {
				return "", maintenance.PortDiagnostic{}, err
			}
			diagnostic, err := command.DiagnosePort(projectapp.DiagnosePortRequest{Port: req.Port, ScenarioName: req.ScenarioName})
			if err != nil {
				return "", maintenance.PortDiagnostic{}, err
			}
			format, err := parseOutputFormat(ctx.Globals)
			if err != nil {
				return "", maintenance.PortDiagnostic{}, err
			}
			return format, diagnostic, nil
		},
		topcli.RenderDiagnosePortResponse,
	)(app, ctx, args)
}

func newContractCommandService() contractapp.Service {
	return contractapp.Service{
		ResolveRootFn:     contractcli.ResolveRoot,
		ValidateFn:        contractcli.Validate,
		ShowFn:            contractcli.LoadShowOutput,
		ResolveScenarioFn: contractcli.ResolveScenario,
		MatchGlobFn:       contractcli.MatchGlob,
	}
}
