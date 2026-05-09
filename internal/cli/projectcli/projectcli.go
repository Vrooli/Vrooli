package projectcli

import (
	"fmt"
	"io"
	"strconv"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/templatevalidation"
)

type StatusOptions struct {
	ResourcesOnly bool
	ScenariosOnly bool
}

type StatusResponse struct {
	Options StatusOptions
	Report  project.StatusReport
}

type OrphansResponse struct {
	KillReport *control.StopReport
	List       []maintenance.SystemProcess
	DryRun     bool
}

type LocksResponse struct {
	CleanReport   *control.StopReport
	List          []maintenance.LockInfo
	RuntimeClaims []maintenance.RuntimeClaimInfo
}

type TemplateValidationCleanupResponse struct {
	Result templatevalidation.CleanupResult
}

func RenderStatusReport(w io.Writer, format cliout.Format, resp StatusResponse) error {
	if format == cliout.FormatJSON {
		return cliout.WriteSuccessJSON(w, "status", resp.Report)
	}

	if !resp.Options.ScenariosOnly {
		_, _ = fmt.Fprintln(w, "Resources")
		rows := make([][]string, 0, len(resp.Report.Resources))
		for _, item := range resp.Report.Resources {
			healthy := "n/a"
			if item.Healthy != nil {
				if *item.Healthy {
					healthy = "healthy"
				} else {
					healthy = "unhealthy"
				}
			}
			rows = append(rows, []string{
				item.Resource.Name,
				cliout.BoolLabel(item.Resource.Enabled),
				cliout.BoolLabel(item.Running),
				healthy,
				item.Message,
			})
		}
		_ = cliout.RenderTable(w, []string{"Name", "Enabled", "Running", "Health", "Status"}, rows)
		_, _ = fmt.Fprintln(w)
	}

	if !resp.Options.ResourcesOnly {
		_, _ = fmt.Fprintln(w, "Scenarios")
		rows := make([][]string, 0, len(resp.Report.Scenarios))
		for _, item := range resp.Report.Scenarios {
			health := ""
			if item.Health != nil {
				health = fmt.Sprint(item.Health)
			}
			rows = append(rows, []string{
				item.Name,
				item.Status,
				fmt.Sprintf("%d", item.Processes),
				health,
				item.Runtime,
			})
		}
		_ = cliout.RenderTable(w, []string{"Name", "Status", "Processes", "Health", "Runtime"}, rows)
		_, _ = fmt.Fprintln(w)
	}

	if resp.Report.Maintenance != nil {
		_, _ = fmt.Fprintln(w, "Maintenance")
		health := resp.Report.Maintenance.HealthSnapshot()
		rows := [][]string{
			{"Tracked processes", strconv.Itoa(resp.Report.Maintenance.TrackedProcesses)},
			{"Running tracked", strconv.Itoa(resp.Report.Maintenance.RunningTracked)},
			{"Child processes", strconv.Itoa(resp.Report.Maintenance.ChildProcesses)},
			{"Zombie processes", strconv.Itoa(resp.Report.Maintenance.ZombieProcesses)},
			{"Orphan processes", strconv.Itoa(resp.Report.Maintenance.OrphanProcesses)},
			{"Overall health", health.OverallStatus},
		}
		_ = cliout.RenderTable(w, []string{"Check", "Value"}, rows)
	}
	return nil
}

func RenderDoctorReport(w io.Writer, format cliout.Format, report project.DoctorReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteSuccessJSON(w, "checks", report.Checks)
	}

	rows := make([][]string, 0, len(report.Checks))
	for _, item := range report.Checks {
		rows = append(rows, []string{item.Name, item.Status, item.Message})
	}
	return cliout.RenderTable(w, []string{"Check", "Status", "Message"}, rows)
}

func RenderStopReport(w io.Writer, format cliout.Format, report control.StopReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteSuccessJSON(w, "data", report)
	}
	for _, item := range report.Stopped {
		_, _ = fmt.Fprintf(w, "Stopped %s\n", item.Name)
	}
	for _, item := range report.Failed {
		_, _ = fmt.Fprintf(w, "Failed %s: %s\n", item.Name, item.Error)
	}
	return nil
}

func RenderOrphansResponse(w io.Writer, format cliout.Format, resp OrphansResponse) error {
	if resp.KillReport != nil {
		typed := *resp.KillReport
		if format == cliout.FormatJSON {
			return cliout.WriteSuccessJSON(w, "data", typed)
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
	}
	if format == cliout.FormatJSON {
		if resp.DryRun {
			return cliout.WriteSuccessJSON(w, "dry_run", struct {
				Orphans []maintenance.SystemProcess `json:"orphans"`
			}{Orphans: resp.List})
		}
		return cliout.WriteSuccessJSON(w, "orphans", resp.List)
	}
	if resp.DryRun {
		if len(resp.List) == 0 {
			_, _ = fmt.Fprintln(w, "[dry-run] No orphaned Vrooli processes would be killed.")
			return nil
		}
		_, _ = fmt.Fprintf(w, "[dry-run] %d orphaned Vrooli process(es) would be killed:\n", len(resp.List))
		rows := make([][]string, 0, len(resp.List))
		for _, item := range resp.List {
			rows = append(rows, []string{strconv.Itoa(item.PID), strconv.Itoa(item.PPID), item.Command})
		}
		if err := cliout.RenderTable(w, []string{"PID", "PPID", "Command"}, rows); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(w, "Re-run without --dry-run to send SIGTERM (then SIGKILL on unresponsive processes).")
		return nil
	}
	if len(resp.List) == 0 {
		_, _ = fmt.Fprintln(w, "No orphaned Vrooli processes found.")
		return nil
	}
	rows := make([][]string, 0, len(resp.List))
	for _, item := range resp.List {
		rows = append(rows, []string{strconv.Itoa(item.PID), strconv.Itoa(item.PPID), item.Command})
	}
	return cliout.RenderTable(w, []string{"PID", "PPID", "Command"}, rows)
}

func RenderLocksResponse(w io.Writer, format cliout.Format, resp LocksResponse) error {
	if resp.CleanReport != nil {
		typed := *resp.CleanReport
		if format == cliout.FormatJSON {
			return cliout.WriteSuccessJSON(w, "data", typed)
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
	}
	if format == cliout.FormatJSON {
		return cliout.WriteSuccessFields(w, map[string]any{
			"locks":           resp.List,
			"registry_claims": resp.RuntimeClaims,
		})
	}
	if len(resp.List) == 0 && len(resp.RuntimeClaims) == 0 {
		_, _ = fmt.Fprintln(w, "No port locks found.")
		return nil
	}
	if len(resp.RuntimeClaims) > 0 {
		_, _ = fmt.Fprintln(w, "Registry claims")
		rows := make([][]string, 0, len(resp.RuntimeClaims))
		for _, item := range resp.RuntimeClaims {
			rows = append(rows, []string{strconv.Itoa(item.Port), item.Scenario, item.InstanceStatus, item.ClaimStatus})
		}
		if err := cliout.RenderTable(w, []string{"Port", "Scenario", "Lease", "Claim"}, rows); err != nil {
			return err
		}
		if len(resp.List) > 0 {
			_, _ = fmt.Fprintln(w)
		}
	}
	if len(resp.List) > 0 {
		_, _ = fmt.Fprintln(w, "Legacy lock files")
		rows := make([][]string, 0, len(resp.List))
		for _, item := range resp.List {
			status := "active"
			if item.Stale {
				status = "stale"
			}
			rows = append(rows, []string{strconv.Itoa(item.Port), item.Scenario, strconv.Itoa(item.PID), status})
		}
		return cliout.RenderTable(w, []string{"Port", "Scenario", "PID", "Status"}, rows)
	}
	return nil
}

func RenderTemplateValidationCleanupResponse(w io.Writer, format cliout.Format, resp TemplateValidationCleanupResponse) error {
	result := resp.Result
	if format == cliout.FormatJSON {
		return cliout.WriteFieldsWithSuccess(w, len(result.Failures) == 0, map[string]any{"cleanup": result})
	}
	_, _ = fmt.Fprintln(w, "Template validation cleanup")
	_, _ = fmt.Fprintf(w, "Status: %s\n", result.Message)
	if len(result.Eligible) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Eligible")
		for _, run := range result.Eligible {
			_, _ = fmt.Fprintf(w, "  %s  %s  age=%s  retained=%t  %s\n", run.Marker.RunID, run.Marker.Template, run.Age, run.Marker.Retained, run.Marker.TempRoot)
		}
	}
	if len(result.Removed) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Removed")
		for _, run := range result.Removed {
			_, _ = fmt.Fprintf(w, "  %s  %s\n", run.Marker.RunID, run.Marker.TempRoot)
		}
	}
	if len(result.Skipped) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Skipped")
		for _, skipped := range result.Skipped {
			label := skipped.Path
			if skipped.Run != nil {
				label = skipped.Run.Marker.RunID
			}
			_, _ = fmt.Fprintf(w, "  %s  %s\n", label, skipped.Reason)
		}
	}
	if len(result.Failures) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Failures")
		for _, failure := range result.Failures {
			label := failure.Path
			if failure.Run != nil {
				label = failure.Run.Marker.RunID
			}
			if label == "" {
				label = "cleanup"
			}
			_, _ = fmt.Fprintf(w, "  %s  %s\n", label, failure.Error)
		}
	}
	if result.NeedsProtoGenerate {
		_, _ = fmt.Fprintln(w)
		if result.DryRun {
			_, _ = fmt.Fprintln(w, "Next steps: rerun without --dry-run to remove proto artifacts and regenerate packages/proto outputs.")
		} else if !result.ProtoGenerateRan {
			_, _ = fmt.Fprintln(w, "Next steps: run `cd packages/proto && make generate` to refresh proto outputs.")
		}
	}
	return nil
}

func RenderPortDiagnostic(w io.Writer, format cliout.Format, diagnostic maintenance.PortDiagnostic) error {
	if format == cliout.FormatJSON {
		return cliout.WriteSuccessJSON(w, "diagnostic", diagnostic)
	}

	_, _ = fmt.Fprintf(w, "Port %d\n", diagnostic.Port)
	if diagnostic.Scenario != "" {
		_, _ = fmt.Fprintf(w, "Scenario: %s\n", diagnostic.Scenario)
	}
	if diagnostic.ListenerInspection.Available {
		if diagnostic.ListenerInspection.Tool != "" {
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
	if len(diagnostic.RegistryClaims) > 0 {
		_, _ = fmt.Fprintln(w, "Registry claims:")
		for _, claim := range diagnostic.RegistryClaims {
			_, _ = fmt.Fprintf(w, "  %s  scenario=%s instance=%s lease=%s claim=%s\n", claim.ClaimID, claim.Scenario, claim.InstanceID, claim.InstanceStatus, claim.ClaimStatus)
		}
	} else {
		_, _ = fmt.Fprintln(w, "Registry claims: none")
	}
	if len(diagnostic.RegistryProcesses) > 0 {
		_, _ = fmt.Fprintln(w, "Registry process refs:")
		for _, ref := range diagnostic.RegistryProcesses {
			pid := ""
			if ref.PID != nil {
				pid = strconv.Itoa(*ref.PID)
			}
			_, _ = fmt.Fprintf(w, "  %s  pid=%s status=%s step=%s\n", ref.RefID, pid, ref.Status, ref.Step)
		}
	}
	if diagnostic.PortPolicy.EphemeralMin > 0 {
		band := diagnostic.PortPolicy.CanonicalBand
		if band == "" {
			band = "outside canonical bands"
		}
		_, _ = fmt.Fprintf(w, "OS ephemeral range: %d-%d (source=%s)\n",
			diagnostic.PortPolicy.EphemeralMin, diagnostic.PortPolicy.EphemeralMax, diagnostic.PortPolicy.EphemeralSource)
		_, _ = fmt.Fprintf(w, "Inside ephemeral range: %t\n", diagnostic.PortPolicy.InsideEphemeralRange)
		_, _ = fmt.Fprintf(w, "Canonical band: %s\n", band)
	}
	_, _ = fmt.Fprintf(w, "Host orphan Vrooli processes: %d (run `vrooli orphans` to list)\n", diagnostic.HostOrphanCount)
	_, _ = fmt.Fprintln(w, "Recommended actions:")
	for _, recommendation := range diagnostic.Recommendations {
		_, _ = fmt.Fprintf(w, "  - %s\n", recommendation)
	}
	return nil
}
