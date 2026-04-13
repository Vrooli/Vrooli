package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/projectcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/project"
)

func runTopLevelStatusCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[topLevelStatusOptions, topLevelStatusResponse]{
		parse: parseTopLevelStatusRequestFromContext,
		run:   runTopLevelStatusRequest,
		render: func(w io.Writer, format cliout.Format, resp topLevelStatusResponse) error {
			return projectcli.RenderStatusReport(w, format, projectcli.StatusResponse{
				Options: projectcli.StatusOptions{
					ResourcesOnly: resp.Options.ResourcesOnly,
					ScenariosOnly: resp.Options.ScenariosOnly,
				},
				Report: resp.Report,
			})
		},
	})
}

func runTopLevelDoctorCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[topLevelNoArgsRequest, project.DoctorReport]{
		parse:  parseTopLevelDoctorRequestFromContext,
		run:    runTopLevelDoctorRequest,
		render: renderTopLevelDoctorResponse,
	})
}

func runTopLevelStopCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[topLevelStopRequest, control.StopReport]{
		parse:  parseTopLevelStopRequestFromContext,
		run:    runTopLevelStopRequest,
		render: renderTopLevelStopResponse,
	})
}

func runTopLevelResourceCommandWithApp(app *App, ctx *commandContext, args []string) error {
	if len(args) == 0 || wantsCommandHelp(args) {
		return runResourceCommand(nil, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
	}
	controller, err := app.newResourceController(ctx)
	if err != nil {
		return err
	}
	return runResourceCommand(controller, ctx.Globals, args, ctx.Stdout, ctx.Stderr)
}

func runTopLevelOrphansCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[topLevelOrphansRequest, topLevelOrphansResponse]{
		parse:  parseTopLevelOrphansRequestFromContext,
		run:    runTopLevelOrphansRequest,
		render: renderTopLevelOrphansResponse,
	})
}

func runTopLevelLocksCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[topLevelLocksRequest, topLevelLocksResponse]{
		parse:  parseTopLevelLocksRequestFromContext,
		run:    runTopLevelLocksRequest,
		render: renderTopLevelLocksResponse,
	})
}

func runTopLevelDiagnosePortCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[topLevelDiagnosePortRequest, maintenance.PortDiagnostic]{
		parse:  parseTopLevelDiagnosePortRequestFromContext,
		run:    runTopLevelDiagnosePortRequest,
		render: renderTopLevelDiagnosePortResponse,
	})
}

type topLevelStatusOptions struct {
	Help          bool
	ResourcesOnly bool
	ScenariosOnly bool
	Fast          bool
}

func parseTopLevelStatusArgs(args []string) (topLevelStatusOptions, error) {
	opts := topLevelStatusOptions{Fast: true}
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			opts.Help = true
		case "--resources":
			opts.ResourcesOnly = true
		case "--scenarios":
			opts.ScenariosOnly = true
		case "--fast":
			opts.Fast = true
		case "--no-fast":
			opts.Fast = false
		default:
			return topLevelStatusOptions{}, unknownOptionError("status", arg)
		}
	}
	if opts.ResourcesOnly && opts.ScenariosOnly {
		return topLevelStatusOptions{}, usageErrorf("status", "status accepts only one of --resources or --scenarios")
	}
	return opts, nil
}

type topLevelStatusResponse struct {
	Options topLevelStatusOptions
	Report  project.StatusReport
}

func parseTopLevelStatusRequest(globals globalOptions, args []string) (topLevelStatusOptions, error) {
	opts, err := parseTopLevelStatusArgs(args)
	if err != nil {
		return topLevelStatusOptions{}, err
	}
	if opts.Help {
		return topLevelStatusOptions{}, commandHelpOnly("Usage: vrooli status [--resources|--scenarios] [--fast|--no-fast] [--json]")
	}
	return opts, nil
}

func parseTopLevelStatusRequestFromContext(ctx *commandContext, args []string) (topLevelStatusOptions, error) {
	return parseTopLevelStatusRequest(ctx.Globals, args)
}

func runTopLevelStatusRequest(app *App, ctx *commandContext, opts topLevelStatusOptions) (cliout.Format, topLevelStatusResponse, error) {
	controller, err := app.newProjectController(ctx)
	if err != nil {
		return "", topLevelStatusResponse{}, err
	}
	report, err := controller.Status(project.StatusOptions{
		ResourcesOnly: opts.ResourcesOnly,
		ScenariosOnly: opts.ScenariosOnly,
		Fast:          opts.Fast,
	})
	if err != nil {
		return "", topLevelStatusResponse{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", topLevelStatusResponse{}, err
	}
	return format, topLevelStatusResponse{Options: opts, Report: report}, nil
}

func runProjectLifecyclePhaseCommandWithApp(app *App, ctx *commandContext, phase string, args []string) error {
	if wantsCommandHelp(args) {
		showProjectLifecycleHelp(ctx.Stdout, phase)
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

func showProjectLifecycleHelp(w io.Writer, phase string) {
	_, _ = fmt.Fprintf(w, "Usage: vrooli %s\n", phase)
}

func normalizeResourceSubcommand(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
