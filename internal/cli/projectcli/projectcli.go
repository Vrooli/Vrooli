package projectcli

import (
	"fmt"
	"io"
	"strconv"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/project"
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
}

type LocksResponse struct {
	CleanReport *control.StopReport
	List        []maintenance.LockInfo
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
		return cliout.WriteSuccessJSON(w, "orphans", resp.List)
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
		return cliout.WriteSuccessJSON(w, "locks", resp.List)
	}
	if len(resp.List) == 0 {
		_, _ = fmt.Fprintln(w, "No port locks found.")
		return nil
	}
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
	_, _ = fmt.Fprintf(w, "Orphans detected: %d\n", diagnostic.OrphanCount)
	_, _ = fmt.Fprintln(w, "Recommended actions:")
	for _, recommendation := range diagnostic.Recommendations {
		_, _ = fmt.Fprintf(w, "  - %s\n", recommendation)
	}
	return nil
}
