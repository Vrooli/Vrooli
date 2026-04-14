package projectcli

import (
	"io"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/project"
	projectsetup "github.com/vrooli/vrooli/internal/setup"
)

func SetupHandler[C any](stdout func(C) io.Writer, run func(C, projectsetup.Options) error) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		opts, err := ParseSetupOptions(args)
		if err != nil {
			if renderHelp(stdout(ctx), err) {
				return nil
			}
			return err
		}
		return run(ctx, opts)
	}
}

func DevelopHandler[C any](stdout func(C) io.Writer, run func(C, projectsetup.Options) error) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		opts, err := ParseDevelopOptions(args)
		if err != nil {
			if renderHelp(stdout(ctx), err) {
				return nil
			}
			return err
		}
		return run(ctx, opts)
	}
}

func BuildHandler[C any](stdout func(C) io.Writer, run func(C) error) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		if _, err := ParseBuildRequest(args); err != nil {
			if renderHelp(stdout(ctx), err) {
				return nil
			}
			return err
		}
		return run(ctx)
	}
}

func ProjectPhaseHandler[C any](stdout func(C) io.Writer, phase string, run func(C, []string) error) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		req, err := ParseProjectPhaseRequest(phase, args)
		if err != nil {
			if renderHelp(stdout(ctx), err) {
				return nil
			}
			return err
		}
		return run(ctx, req.Args)
	}
}

func CleanupHandler[C any](stdout func(C) io.Writer, runOrphans func(C, []string) error, runLocks func(C, []string) error) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		req, err := ParseCleanupRequest(args)
		if err != nil {
			if renderHelp(stdout(ctx), err) {
				return nil
			}
			return err
		}
		switch req.Target {
		case "orphans":
			return runOrphans(ctx, append([]string{"kill"}, req.Args...))
		case "locks":
			return runLocks(ctx, append([]string{"clean"}, req.Args...))
		default:
			return nil
		}
	}
}

func LifecycleHandler[C any](stdout func(C) io.Writer, runProtect func(C, []string) error) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		req, err := ParseLifecycleRequest(args)
		if err != nil {
			if renderHelp(stdout(ctx), err) {
				return nil
			}
			return err
		}
		switch req.Subcommand {
		case "protect":
			return runProtect(ctx, req.Args)
		default:
			return nil
		}
	}
}

func StatusHandler[C any](stdout func(C) io.Writer, outputFormat func(C) (cliout.Format, error), run func(C, StatusRequest) (project.StatusReport, error)) rootcli.Handler[C] {
	return rootcli.BindGlobalCommand(stdout,
		func(ctx C, args []string) (StatusRequest, error) {
			return ParseStatusRequest(args)
		},
		func(ctx C, req StatusRequest) (cliout.Format, StatusResponse, error) {
			report, err := run(ctx, req)
			if err != nil {
				return "", StatusResponse{}, err
			}
			format, err := outputFormat(ctx)
			if err != nil {
				return "", StatusResponse{}, err
			}
			return format, StatusResponse{
				Options: StatusOptions{
					ResourcesOnly: req.ResourcesOnly,
					ScenariosOnly: req.ScenariosOnly,
				},
				Report: report,
			}, nil
		},
		RenderStatusResponse,
	)
}

func DoctorHandler[C any](stdout func(C) io.Writer, outputFormat func(C) (cliout.Format, error), run func(C) (project.DoctorReport, error)) rootcli.Handler[C] {
	return rootcli.BindGlobalCommand(stdout,
		func(ctx C, args []string) (NoArgsRequest, error) {
			return ParseDoctorRequest(args)
		},
		func(ctx C, _ NoArgsRequest) (cliout.Format, project.DoctorReport, error) {
			report, err := run(ctx)
			if err != nil {
				return "", project.DoctorReport{}, err
			}
			format, err := outputFormat(ctx)
			if err != nil {
				return "", project.DoctorReport{}, err
			}
			return format, report, nil
		},
		RenderDoctorResponse,
	)
}

func StopHandler[C any](stdout func(C) io.Writer, outputFormat func(C) (cliout.Format, error), run func(C, StopRequest) (control.StopReport, error)) rootcli.Handler[C] {
	return rootcli.BindGlobalCommand(stdout,
		func(ctx C, args []string) (StopRequest, error) {
			return ParseStopRequest(args)
		},
		func(ctx C, req StopRequest) (cliout.Format, control.StopReport, error) {
			report, err := run(ctx, req)
			if err != nil {
				return "", control.StopReport{}, err
			}
			format, err := outputFormat(ctx)
			if err != nil {
				return "", control.StopReport{}, err
			}
			return format, report, nil
		},
		RenderStopResponse,
	)
}

func OrphansHandler[C any](stdout func(C) io.Writer, outputFormat func(C) (cliout.Format, error), run func(C, OrphansRequest) (OrphansResponse, error)) rootcli.Handler[C] {
	return rootcli.BindGlobalCommand(stdout,
		func(ctx C, args []string) (OrphansRequest, error) {
			return ParseOrphansRequest(args)
		},
		func(ctx C, req OrphansRequest) (cliout.Format, OrphansResponse, error) {
			resp, err := run(ctx, req)
			if err != nil {
				return "", OrphansResponse{}, err
			}
			format, err := outputFormat(ctx)
			if err != nil {
				return "", OrphansResponse{}, err
			}
			return format, resp, nil
		},
		RenderOrphansResponse,
	)
}

func LocksHandler[C any](stdout func(C) io.Writer, outputFormat func(C) (cliout.Format, error), run func(C, LocksRequest) (LocksResponse, error)) rootcli.Handler[C] {
	return rootcli.BindGlobalCommand(stdout,
		func(ctx C, args []string) (LocksRequest, error) {
			return ParseLocksRequest(args)
		},
		func(ctx C, req LocksRequest) (cliout.Format, LocksResponse, error) {
			resp, err := run(ctx, req)
			if err != nil {
				return "", LocksResponse{}, err
			}
			format, err := outputFormat(ctx)
			if err != nil {
				return "", LocksResponse{}, err
			}
			return format, resp, nil
		},
		RenderLocksResponse,
	)
}

func DiagnosePortHandler[C any](stdout func(C) io.Writer, outputFormat func(C) (cliout.Format, error), run func(C, DiagnosePortRequest) (maintenance.PortDiagnostic, error)) rootcli.Handler[C] {
	return rootcli.BindGlobalCommand(stdout,
		func(ctx C, args []string) (DiagnosePortRequest, error) {
			return ParseDiagnosePortRequest(args)
		},
		func(ctx C, req DiagnosePortRequest) (cliout.Format, maintenance.PortDiagnostic, error) {
			resp, err := run(ctx, req)
			if err != nil {
				return "", maintenance.PortDiagnostic{}, err
			}
			format, err := outputFormat(ctx)
			if err != nil {
				return "", maintenance.PortDiagnostic{}, err
			}
			return format, resp, nil
		},
		RenderPortDiagnostic,
	)
}

func renderHelp(w io.Writer, err error) bool {
	return rootcli.HandleHelp(w, err)
}
