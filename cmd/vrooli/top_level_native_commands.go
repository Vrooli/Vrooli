package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/shell"
)

func runTopLevelStatusCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeTopLevelCommand(app, ctx, args, topLevelCommandAction[topLevelStatusOptions, topLevelStatusResponse]{
		parse: parseTopLevelStatusRequest,
		run:   runTopLevelStatusRequest,
		render: func(w io.Writer, format cliout.Format, resp topLevelStatusResponse) error {
			return writeProjectStatusReport(w, format, resp.Report, resp.Options)
		},
	})
}

func runTopLevelDoctorCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeTopLevelCommand(app, ctx, args, topLevelCommandAction[topLevelNoArgsRequest, project.DoctorReport]{
		parse:  parseTopLevelDoctorRequest,
		run:    runTopLevelDoctorRequest,
		render: renderTopLevelDoctorResponse,
	})
}

func runTopLevelStopCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeTopLevelCommand(app, ctx, args, topLevelCommandAction[topLevelStopRequest, control.StopReport]{
		parse:  parseTopLevelStopRequest,
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
	return executeTopLevelCommand(app, ctx, args, topLevelCommandAction[topLevelOrphansRequest, any]{
		parse:  parseTopLevelOrphansRequest,
		run:    runTopLevelOrphansRequest,
		render: renderTopLevelOrphansResponse,
	})
}

func runTopLevelLocksCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeTopLevelCommand(app, ctx, args, topLevelCommandAction[topLevelLocksRequest, any]{
		parse:  parseTopLevelLocksRequest,
		run:    runTopLevelLocksRequest,
		render: renderTopLevelLocksResponse,
	})
}

func runTopLevelDiagnosePortCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeTopLevelCommand(app, ctx, args, topLevelCommandAction[topLevelDiagnosePortRequest, maintenance.PortDiagnostic]{
		parse:  parseTopLevelDiagnosePortRequest,
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

func showStatusHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli status [--resources|--scenarios] [--fast|--no-fast] [--json]")
}

func showDoctorHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli doctor [--json]")
}

func showStopHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli stop [all|scenarios|resources|scenario:<name>|resource:<name>|<name>...] [--json]")
}

func boolLabel(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func installedCommand(name string) bool {
	_, err := shell.LookPath(name)
	return err == nil
}

func normalizeResourceSubcommand(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
