package topcli

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

type NoArgsRequest struct{}

type StatusRequest struct {
	ResourcesOnly bool
	ScenariosOnly bool
	Fast          bool
}

type StatusResponse struct {
	Options StatusRequest
	Report  project.StatusReport
}

type StopRequest struct {
	Targets []string
}

type OrphansRequest struct {
	Kill bool
}

type LocksRequest struct {
	Clean bool
}

type DiagnosePortRequest struct {
	Port         int
	ScenarioName string
}

type (
	OrphansResponse = projectcli.OrphansResponse
	LocksResponse   = projectcli.LocksResponse
)

func ParseStatusRequest(args []string) (StatusRequest, error) {
	req := StatusRequest{Fast: true}
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return StatusRequest{}, commandHelpOnly(StatusHelpText)
		case "--resources":
			req.ResourcesOnly = true
		case "--scenarios":
			req.ScenariosOnly = true
		case "--fast":
			req.Fast = true
		case "--no-fast":
			req.Fast = false
		default:
			return StatusRequest{}, unknownOptionError("status", arg)
		}
	}
	if req.ResourcesOnly && req.ScenariosOnly {
		return StatusRequest{}, usageErrorf("status", "status accepts only one of --resources or --scenarios")
	}
	return req, nil
}

func RenderStatusResponse(w io.Writer, format cliout.Format, resp StatusResponse) error {
	return projectcli.RenderStatusReport(w, format, projectcli.StatusResponse{
		Options: projectcli.StatusOptions{
			ResourcesOnly: resp.Options.ResourcesOnly,
			ScenariosOnly: resp.Options.ScenariosOnly,
		},
		Report: resp.Report,
	})
}

func ParseDoctorRequest(args []string) (NoArgsRequest, error) {
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return NoArgsRequest{}, commandHelpOnly(DoctorHelpText)
		default:
			return NoArgsRequest{}, unknownOptionError("doctor", arg)
		}
	}
	return NoArgsRequest{}, nil
}

func RenderDoctorResponse(w io.Writer, format cliout.Format, resp project.DoctorReport) error {
	return projectcli.RenderDoctorReport(w, format, resp)
}

func ParseStopRequest(args []string) (StopRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return StopRequest{}, commandHelpOnly(StopHelpText)
		}
	}
	return StopRequest{Targets: append([]string(nil), args...)}, nil
}

func RenderStopResponse(w io.Writer, format cliout.Format, report control.StopReport) error {
	return projectcli.RenderStopReport(w, format, report)
}

func ParseOrphansRequest(args []string) (OrphansRequest, error) {
	req := OrphansRequest{}
	for _, arg := range args {
		switch arg {
		case "kill":
			req.Kill = true
		case "--help", "-h", "help":
			return OrphansRequest{}, commandHelpOnly(OrphansHelpText)
		default:
			return OrphansRequest{}, unknownOptionError("orphans", arg)
		}
	}
	return req, nil
}

func RenderOrphansResponse(w io.Writer, format cliout.Format, resp OrphansResponse) error {
	return projectcli.RenderOrphansResponse(w, format, resp)
}

func ParseLocksRequest(args []string) (LocksRequest, error) {
	req := LocksRequest{}
	for _, arg := range args {
		switch arg {
		case "clean":
			req.Clean = true
		case "--help", "-h", "help":
			return LocksRequest{}, commandHelpOnly(LocksHelpText)
		default:
			return LocksRequest{}, unknownOptionError("locks", arg)
		}
	}
	return req, nil
}

func RenderLocksResponse(w io.Writer, format cliout.Format, resp LocksResponse) error {
	return projectcli.RenderLocksResponse(w, format, resp)
}

func ParseDiagnosePortRequest(args []string) (DiagnosePortRequest, error) {
	if len(args) == 0 {
		return DiagnosePortRequest{}, usageErrorf("diagnose-port", "usage: vrooli diagnose-port <port> [scenario] [--json]")
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		return DiagnosePortRequest{}, commandHelpOnly(DiagnosePortHelpText)
	}
	port, err := strconv.Atoi(strings.TrimSpace(args[0]))
	if err != nil || port <= 0 {
		return DiagnosePortRequest{}, usageErrorf("diagnose-port", "invalid port: %s", args[0])
	}
	req := DiagnosePortRequest{Port: port}
	if len(args) > 1 {
		req.ScenarioName = args[1]
	}
	return req, nil
}

func RenderDiagnosePortResponse(w io.Writer, format cliout.Format, diagnostic maintenance.PortDiagnostic) error {
	return projectcli.RenderPortDiagnostic(w, format, diagnostic)
}
