package main

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/project"
)

type topLevelCommandAction[Req any, Resp any] struct {
	parse  func(globals globalOptions, args []string) (Req, error)
	run    func(app *App, ctx *commandContext, req Req) (cliout.Format, Resp, error)
	render func(w io.Writer, format cliout.Format, resp Resp) error
}

func executeTopLevelCommand[Req any, Resp any](app *App, ctx *commandContext, args []string, action topLevelCommandAction[Req, Resp]) error {
	req, err := action.parse(ctx.Globals, args)
	if err != nil {
		var helpErr commandHelpError
		if errors.As(err, &helpErr) {
			_, _ = fmt.Fprintln(ctx.Stdout, helpErr.message)
			return nil
		}
		return err
	}
	format, resp, err := action.run(app, ctx, req)
	if err != nil {
		return err
	}
	return action.render(ctx.Stdout, format, resp)
}

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

type topLevelOrphansListResponse struct {
	Items []maintenance.SystemProcess
}

type topLevelLocksListResponse struct {
	Items []maintenance.LockInfo
}

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
	return writeDoctorReport(w, format, resp)
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
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "data", report)
	}
	for _, item := range report.Stopped {
		_, _ = fmt.Fprintf(w, "Stopped %s\n", item.Name)
	}
	for _, item := range report.Failed {
		_, _ = fmt.Fprintf(w, "Failed %s: %s\n", item.Name, item.Error)
	}
	return nil
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

func runTopLevelOrphansRequest(app *App, ctx *commandContext, req topLevelOrphansRequest) (cliout.Format, any, error) {
	controller, err := app.newMaintenanceController(ctx)
	if err != nil {
		return "", nil, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", nil, err
	}
	if req.Kill {
		report, err := controller.KillOrphans()
		if err != nil {
			return "", nil, err
		}
		return format, report, nil
	}
	items, err := controller.ListOrphans()
	if err != nil {
		return "", nil, err
	}
	return format, topLevelOrphansListResponse{Items: items}, nil
}

func renderTopLevelOrphansResponse(w io.Writer, format cliout.Format, resp any) error {
	switch typed := resp.(type) {
	case control.StopReport:
		if format == cliout.FormatJSON {
			return writeSuccessData(w, "data", typed)
		}
		for _, item := range typed.Stopped {
			_, _ = fmt.Fprintf(w, "Stopped orphan PID %s (%s)\n", item.Name, item.Message)
		}
		for _, item := range typed.Failed {
			_, _ = fmt.Fprintf(w, "Failed orphan PID %s: %s\n", item.Name, item.Error)
		}
		if len(typed.Stopped) == 0 && len(typed.Failed) == 0 {
			_, _ = fmt.Fprintln(w, "No orphaned Vrooli processes found.")
		}
		return nil
	case topLevelOrphansListResponse:
		if format == cliout.FormatJSON {
			return writeSuccessData(w, "orphans", typed.Items)
		}
		if len(typed.Items) == 0 {
			_, _ = fmt.Fprintln(w, "No orphaned Vrooli processes found.")
			return nil
		}
		rows := make([][]string, 0, len(typed.Items))
		for _, item := range typed.Items {
			rows = append(rows, []string{strconv.Itoa(item.PID), strconv.Itoa(item.PPID), item.Command})
		}
		return cliout.RenderTable(w, []string{"PID", "PPID", "Command"}, rows)
	default:
		return fmt.Errorf("unexpected orphans response type %T", resp)
	}
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

func runTopLevelLocksRequest(app *App, ctx *commandContext, req topLevelLocksRequest) (cliout.Format, any, error) {
	controller, err := app.newMaintenanceController(ctx)
	if err != nil {
		return "", nil, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", nil, err
	}
	if req.Clean {
		report, err := controller.CleanStaleLocks()
		if err != nil {
			return "", nil, err
		}
		return format, report, nil
	}
	items, err := controller.ListLocks()
	if err != nil {
		return "", nil, err
	}
	return format, topLevelLocksListResponse{Items: items}, nil
}

func renderTopLevelLocksResponse(w io.Writer, format cliout.Format, resp any) error {
	switch typed := resp.(type) {
	case control.StopReport:
		if format == cliout.FormatJSON {
			return writeSuccessData(w, "data", typed)
		}
		for _, item := range typed.Stopped {
			_, _ = fmt.Fprintf(w, "Removed stale lock for port %s\n", item.Name)
		}
		for _, item := range typed.Failed {
			_, _ = fmt.Fprintf(w, "Failed to remove lock for port %s: %s\n", item.Name, item.Error)
		}
		if len(typed.Stopped) == 0 && len(typed.Failed) == 0 {
			_, _ = fmt.Fprintln(w, "No stale port locks found.")
		}
		return nil
	case topLevelLocksListResponse:
		if format == cliout.FormatJSON {
			return writeSuccessData(w, "locks", typed.Items)
		}
		if len(typed.Items) == 0 {
			_, _ = fmt.Fprintln(w, "No port locks found.")
			return nil
		}
		rows := make([][]string, 0, len(typed.Items))
		for _, item := range typed.Items {
			status := "active"
			if item.Stale {
				status = "stale"
			}
			rows = append(rows, []string{
				strconv.Itoa(item.Port),
				item.Scenario,
				strconv.Itoa(item.PID),
				status,
			})
		}
		return cliout.RenderTable(w, []string{"Port", "Scenario", "PID", "Status"}, rows)
	default:
		return fmt.Errorf("unexpected locks response type %T", resp)
	}
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
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "diagnostic", diagnostic)
	}

	_, _ = fmt.Fprintf(w, "Port %d\n", diagnostic.Port)
	if diagnostic.Scenario != "" {
		_, _ = fmt.Fprintf(w, "Scenario: %s\n", diagnostic.Scenario)
	}
	if diagnostic.ListenerInspection.Available {
		if strings.TrimSpace(diagnostic.ListenerInspection.Tool) != "" {
			_, _ = fmt.Fprintf(w, "Listener inspection: available via %s\n", diagnostic.ListenerInspection.Tool)
		} else {
			_, _ = fmt.Fprintln(w, "Listener inspection: available")
		}
	} else {
		_, _ = fmt.Fprintf(w, "Listener inspection: unavailable (%s)\n", diagnostic.ListenerInspection.Reason)
	}
	if diagnostic.InUse {
		_, _ = fmt.Fprintln(w, "Listeners:")
		for _, listener := range diagnostic.Listeners {
			_, _ = fmt.Fprintf(w, "  PID %d  zombie=%t  %s\n", listener.PID, listener.Zombie, listener.Command)
		}
	} else {
		_, _ = fmt.Fprintln(w, "Listeners: none")
	}
	if diagnostic.Lock != nil {
		_, _ = fmt.Fprintf(w, "Lock: %s (scenario=%s pid=%d stale=%t)\n", diagnostic.Lock.Path, diagnostic.Lock.Scenario, diagnostic.Lock.PID, diagnostic.Lock.Stale)
	} else {
		_, _ = fmt.Fprintln(w, "Lock: none")
	}
	_, _ = fmt.Fprintf(w, "Orphans detected: %d\n", diagnostic.OrphanCount)
	_, _ = fmt.Fprintln(w, "Recommended actions:")
	for _, recommendation := range diagnostic.Recommendations {
		_, _ = fmt.Fprintf(w, "  - %s\n", recommendation)
	}
	return nil
}
