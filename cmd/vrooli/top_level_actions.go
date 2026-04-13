package main

import (
	"io"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/projectcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/project"
)

type topLevelNoArgsRequest struct{}

type topLevelStopRequest struct {
	Targets []string
}

type topLevelOrphansRequest struct {
	Kill bool
}

type topLevelLocksRequest struct {
	Clean bool
}

type topLevelDiagnosePortRequest struct {
	Port         int
	ScenarioName string
}

type (
	topLevelOrphansResponse = projectcli.OrphansResponse
	topLevelLocksResponse   = projectcli.LocksResponse
)

func parseTopLevelDoctorRequest(globals globalOptions, args []string) (topLevelNoArgsRequest, error) {
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return topLevelNoArgsRequest{}, commandHelpOnly("Usage: vrooli doctor [--json]")
		default:
			return topLevelNoArgsRequest{}, unknownOptionError("doctor", arg)
		}
	}
	return topLevelNoArgsRequest{}, nil
}

func runTopLevelDoctorRequest(app *App, ctx *commandContext, _ topLevelNoArgsRequest) (cliout.Format, project.DoctorReport, error) {
	controller, err := app.newProjectController(ctx)
	if err != nil {
		return "", project.DoctorReport{}, err
	}
	report, err := controller.Doctor()
	if err != nil {
		return "", project.DoctorReport{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", project.DoctorReport{}, err
	}
	return format, report, nil
}

func renderTopLevelDoctorResponse(w io.Writer, format cliout.Format, resp project.DoctorReport) error {
	return projectcli.RenderDoctorReport(w, format, resp)
}

func parseTopLevelStopRequest(globals globalOptions, args []string) (topLevelStopRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return topLevelStopRequest{}, commandHelpOnly("Usage: vrooli stop [all|scenarios|resources|scenario:<name>|resource:<name>|<name>...] [--json]")
		}
	}
	return topLevelStopRequest{Targets: append([]string(nil), args...)}, nil
}

func runTopLevelStopRequest(app *App, ctx *commandContext, req topLevelStopRequest) (cliout.Format, control.StopReport, error) {
	controller, err := app.newProjectController(ctx)
	if err != nil {
		return "", control.StopReport{}, err
	}
	report, err := controller.Stop(project.StopOptions{Args: req.Targets})
	if err != nil {
		return "", control.StopReport{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", control.StopReport{}, err
	}
	return format, report, nil
}

func renderTopLevelStopResponse(w io.Writer, format cliout.Format, report control.StopReport) error {
	return projectcli.RenderStopReport(w, format, report)
}

func parseTopLevelOrphansRequest(globals globalOptions, args []string) (topLevelOrphansRequest, error) {
	req := topLevelOrphansRequest{}
	for _, arg := range args {
		switch arg {
		case "kill":
			req.Kill = true
		case "--help", "-h", "help":
			return topLevelOrphansRequest{}, commandHelpOnly("Usage: vrooli orphans [kill] [--json]")
		default:
			return topLevelOrphansRequest{}, unknownOptionError("orphans", arg)
		}
	}
	return req, nil
}

func runTopLevelOrphansRequest(app *App, ctx *commandContext, req topLevelOrphansRequest) (cliout.Format, topLevelOrphansResponse, error) {
	controller, err := app.newMaintenanceController(ctx)
	if err != nil {
		return "", topLevelOrphansResponse{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", topLevelOrphansResponse{}, err
	}
	if req.Kill {
		report, err := controller.KillOrphans()
		if err != nil {
			return "", topLevelOrphansResponse{}, err
		}
		return format, topLevelOrphansResponse{KillReport: &report}, nil
	}
	items, err := controller.ListOrphans()
	if err != nil {
		return "", topLevelOrphansResponse{}, err
	}
	return format, topLevelOrphansResponse{List: items}, nil
}

func renderTopLevelOrphansResponse(w io.Writer, format cliout.Format, resp topLevelOrphansResponse) error {
	return projectcli.RenderOrphansResponse(w, format, resp)
}

func parseTopLevelLocksRequest(globals globalOptions, args []string) (topLevelLocksRequest, error) {
	req := topLevelLocksRequest{}
	for _, arg := range args {
		switch arg {
		case "clean":
			req.Clean = true
		case "--help", "-h", "help":
			return topLevelLocksRequest{}, commandHelpOnly("Usage: vrooli locks [clean] [--json]")
		default:
			return topLevelLocksRequest{}, unknownOptionError("locks", arg)
		}
	}
	return req, nil
}

func runTopLevelLocksRequest(app *App, ctx *commandContext, req topLevelLocksRequest) (cliout.Format, topLevelLocksResponse, error) {
	controller, err := app.newMaintenanceController(ctx)
	if err != nil {
		return "", topLevelLocksResponse{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", topLevelLocksResponse{}, err
	}
	if req.Clean {
		report, err := controller.CleanStaleLocks()
		if err != nil {
			return "", topLevelLocksResponse{}, err
		}
		return format, topLevelLocksResponse{CleanReport: &report}, nil
	}
	items, err := controller.ListLocks()
	if err != nil {
		return "", topLevelLocksResponse{}, err
	}
	return format, topLevelLocksResponse{List: items}, nil
}

func renderTopLevelLocksResponse(w io.Writer, format cliout.Format, resp topLevelLocksResponse) error {
	return projectcli.RenderLocksResponse(w, format, resp)
}

func parseTopLevelDiagnosePortRequest(globals globalOptions, args []string) (topLevelDiagnosePortRequest, error) {
	if len(args) == 0 {
		return topLevelDiagnosePortRequest{}, newUsageError("usage: vrooli diagnose-port <port> [scenario] [--json]", "diagnose-port")
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		return topLevelDiagnosePortRequest{}, commandHelpOnly("Usage: vrooli diagnose-port <port> [scenario] [--json]")
	}
	port, err := strconv.Atoi(strings.TrimSpace(args[0]))
	if err != nil || port <= 0 {
		return topLevelDiagnosePortRequest{}, usageErrorf("diagnose-port", "invalid port: %s", args[0])
	}

	req := topLevelDiagnosePortRequest{Port: port}
	if len(args) > 1 {
		req.ScenarioName = args[1]
	}
	return req, nil
}

func runTopLevelDiagnosePortRequest(app *App, ctx *commandContext, req topLevelDiagnosePortRequest) (cliout.Format, maintenance.PortDiagnostic, error) {
	controller, err := app.newMaintenanceController(ctx)
	if err != nil {
		return "", maintenance.PortDiagnostic{}, err
	}
	diagnostic, err := controller.DiagnosePort(req.Port, req.ScenarioName)
	if err != nil {
		return "", maintenance.PortDiagnostic{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", maintenance.PortDiagnostic{}, err
	}
	return format, diagnostic, nil
}

func renderTopLevelDiagnosePortResponse(w io.Writer, format cliout.Format, diagnostic maintenance.PortDiagnostic) error {
	return projectcli.RenderPortDiagnostic(w, format, diagnostic)
}
