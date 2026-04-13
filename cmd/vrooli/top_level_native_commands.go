package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/projectcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/project"
)

func renderTopLevelStatusResponse(w io.Writer, format cliout.Format, resp topLevelStatusResponse) error {
	return projectcli.RenderStatusReport(w, format, projectcli.StatusResponse{
		Options: projectcli.StatusOptions{
			ResourcesOnly: resp.Options.ResourcesOnly,
			ScenariosOnly: resp.Options.ScenariosOnly,
		},
		Report: resp.Report,
	})
}

func runTopLevelResourceCommandWithApp(app *App, ctx *commandContext, args []string) error {
	if len(args) == 0 || wantsCommandHelp(args) {
		return runResourceCommandWithApp(app, ctx, nil, args)
	}
	controller, err := app.newResourceController(ctx)
	if err != nil {
		return err
	}
	return runResourceCommandWithApp(app, ctx, controller, args)
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

func normalizeResourceSubcommand(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
