package workloadowner

import (
	"context"
	"fmt"
	"runtime"

	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/shell"
)

// CommandRunner is the deliberately small seam used by the live enumerator.
// It keeps the ownership classifier testable without giving it a host-repair
// capability.
type CommandRunner = shell.Runner

type systemCommandRunner = shell.OSRunner

type LiveReport struct {
	Report       Report     `json:"report"`
	Observed     []Workload `json:"observed"`
	Unread       []string   `json:"unread,omitempty"`
	EvidenceNote string     `json:"evidence_note,omitempty"`
}

// Enumerate observes the platform-native workload registries that are
// available. Unsupported registries are explicit unread evidence; they are
// never represented as an empty successful list.
func Enumerate(ctx context.Context, runner CommandRunner) (LiveReport, error) {
	if runner == nil {
		runner = systemCommandRunner{}
	}
	out := LiveReport{EvidenceNote: "classification is computed from observed registries and supplied declarations"}
	if path, err := runner.LookPath("docker"); err == nil && path != "" {
		data, runErr := runner.Run(ctx, path, "ps", "-a", "--format", "{{json .}}")
		if runErr != nil {
			out.Unread = append(out.Unread, "containers: docker query failed: "+runErr.Error())
		} else if workloads, parseErr := ParseDockerPS(data); parseErr != nil {
			return LiveReport{}, parseErr
		} else {
			out.Observed = append(out.Observed, workloads...)
		}
	} else if runtime.GOOS == string(hostreqspec.PlatformLinux) || runtime.GOOS == string(hostreqspec.PlatformDarwin) || runtime.GOOS == string(hostreqspec.PlatformWindows) {
		out.Unread = append(out.Unread, "containers: docker is unavailable")
	}

	switch runtime.GOOS {
	case string(hostreqspec.PlatformLinux):
		if path, err := runner.LookPath("systemctl"); err == nil {
			data, runErr := runner.Run(ctx, path, "list-units", "--all", "--no-legend", "--no-pager")
			if runErr != nil {
				out.Unread = append(out.Unread, "service units: systemctl query failed: "+runErr.Error())
			} else {
				out.Observed = append(out.Observed, ParseServiceUnits(data)...)
			}
		} else {
			out.Unread = append(out.Unread, "service units: systemctl is unavailable")
		}
	case string(hostreqspec.PlatformDarwin):
		out.Unread = append(out.Unread, "service units: launchd enumeration is not implemented")
	case string(hostreqspec.PlatformWindows):
		out.Unread = append(out.Unread, "service units and scheduled tasks: native enumeration is not implemented")
	default:
		out.Unread = append(out.Unread, fmt.Sprintf("service units: unsupported platform %q", runtime.GOOS))
	}
	return out, nil
}

// RedactForPosture removes command lines from informational unmanaged rows.
// Keeping this at the output boundary makes accidental leakage difficult when
// callers construct reports themselves.
func RedactForPosture(report *Report) {
	if report == nil || report.Posture == WholeHost {
		return
	}
	for i := range report.Informational {
		report.Informational[i].CommandLine = ""
	}
	for i := range report.Findings {
		if report.Findings[i].Class == Unmanaged {
			report.Findings[i].CommandLine = ""
		}
	}
}
