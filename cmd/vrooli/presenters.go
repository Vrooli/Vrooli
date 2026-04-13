package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/resources"
)

func writeScenarioLifecycleItems(w io.Writer, format cliout.Format, items []scenarioLifecycleItemOutput) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success":   true,
			"scenarios": items,
		})
	}

	for _, item := range items {
		switch item.Status {
		case "already_running":
			_, _ = fmt.Fprintf(w, "Scenario '%s' is already running", item.Name)
		case "stopped":
			_, _ = fmt.Fprintf(w, "Stopped scenario '%s'", item.Name)
		default:
			_, _ = fmt.Fprintf(w, "Started scenario '%s'", item.Name)
		}
		if item.Health != "" {
			_, _ = fmt.Fprintf(w, " (%s)", item.Health)
		}
		_, _ = fmt.Fprintln(w)
		if len(item.Ports) > 0 {
			_, _ = fmt.Fprintf(w, "  Ports: %s\n", formatPortMap(item.Ports))
		}
		if len(item.FailedDependencies) > 0 {
			_, _ = fmt.Fprintf(w, "  Failed dependencies: %s\n", strings.Join(item.FailedDependencies, ", "))
		}
	}
	return nil
}

func writeScenarioBatchReport(w io.Writer, format cliout.Format, verb string, started []scenarioLifecycleItemOutput, stopped []string, failed []scenarioBatchFailure) error {
	if format == cliout.FormatJSON {
		data := map[string]any{
			"failed": failed,
		}
		if len(started) > 0 {
			data["started"] = started
		}
		if len(stopped) > 0 {
			data["stopped"] = stopped
		}
		return cliout.WriteJSON(w, map[string]any{
			"success": true,
			"data":    data,
		})
	}

	if len(started) == 0 && len(stopped) == 0 && len(failed) == 0 {
		_, _ = fmt.Fprintln(w, "No running scenarios found")
		return nil
	}

	if len(started) > 0 {
		_, _ = fmt.Fprintf(w, "%s %d scenarios\n", verb, len(started))
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "%s scenarios:\n", verb)
		for _, item := range started {
			_, _ = fmt.Fprintf(w, "  %s: %s\n", item.Name, item.Status)
		}
	}

	if len(stopped) > 0 {
		_, _ = fmt.Fprintf(w, "%s %d scenarios\n", verb, len(stopped))
	}

	if len(failed) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "Failed to %s:\n", strings.ToLower(verb))
		for _, item := range failed {
			_, _ = fmt.Fprintf(w, "  %s: %s\n", item.Name, item.Error)
		}
	}
	return nil
}

func writeProjectStatusReport(w io.Writer, format cliout.Format, report project.StatusReport, opts topLevelStatusOptions) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "status", report)
	}

	if !opts.ScenariosOnly {
		_, _ = fmt.Fprintln(w, "Resources")
		rows := make([][]string, 0, len(report.Resources))
		for _, item := range report.Resources {
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
				boolLabel(item.Resource.Enabled),
				boolLabel(item.Running),
				healthy,
				item.Message,
			})
		}
		_ = cliout.RenderTable(w, []string{"Name", "Enabled", "Running", "Health", "Status"}, rows)
		_, _ = fmt.Fprintln(w)
	}

	if !opts.ResourcesOnly {
		_, _ = fmt.Fprintln(w, "Scenarios")
		rows := make([][]string, 0, len(report.Scenarios))
		for _, item := range report.Scenarios {
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

	if report.Maintenance != nil {
		_, _ = fmt.Fprintln(w, "Maintenance")
		health := report.Maintenance.HealthSnapshot()
		rows := [][]string{
			{"Tracked processes", strconv.Itoa(report.Maintenance.TrackedProcesses)},
			{"Running tracked", strconv.Itoa(report.Maintenance.RunningTracked)},
			{"Child processes", strconv.Itoa(report.Maintenance.ChildProcesses)},
			{"Zombie processes", strconv.Itoa(report.Maintenance.ZombieProcesses)},
			{"Orphan processes", strconv.Itoa(report.Maintenance.OrphanProcesses)},
			{"Overall health", health.OverallStatus},
		}
		_ = cliout.RenderTable(w, []string{"Check", "Value"}, rows)
	}
	return nil
}

func writeDoctorReport(w io.Writer, format cliout.Format, report project.DoctorReport) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "checks", report.Checks)
	}

	rows := make([][]string, 0, len(report.Checks))
	for _, item := range report.Checks {
		rows = append(rows, []string{item.Name, item.Status, item.Message})
	}
	return cliout.RenderTable(w, []string{"Check", "Status", "Message"}, rows)
}

func writeResourceList(w io.Writer, format cliout.Format, items []resources.Resource) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "resources", items)
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		decision := ""
		if item.ControlMode == "legacy-adapter" {
			decision = item.LegacyAdapter.FinalDisposition
		}
		rows = append(rows, []string{
			item.Name,
			boolLabel(item.Enabled),
			item.ControlMode,
			item.Driver,
			item.PortabilityTier,
			decision,
			boolLabel(item.Registered),
		})
	}
	return cliout.RenderTable(w, []string{"Name", "Enabled", "Control", "Driver", "Portability", "Decision", "Registered"}, rows)
}

func writeResourceStatuses(w io.Writer, format cliout.Format, items []resources.Status) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "resources", items)
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
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
			boolLabel(item.Resource.Enabled),
			item.Resource.ControlMode,
			boolLabel(item.Running),
			healthy,
			item.Message,
		})
	}
	return cliout.RenderTable(w, []string{"Name", "Enabled", "Control", "Running", "Health", "Status"}, rows)
}

func writeResourceStatus(w io.Writer, format cliout.Format, item resources.Status) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success":   true,
			"name":      item.Resource.Name,
			"installed": item.Installed,
			"running":   item.Running,
			"healthy":   item.Healthy,
			"status":    item.Message,
			"resource":  item,
		})
	}
	rows := [][]string{
		{"Name", item.Resource.Name},
		{"Enabled", boolLabel(item.Resource.Enabled)},
		{"Control", item.Resource.ControlMode},
		{"Driver", item.Resource.Driver},
		{"Portability", item.Resource.PortabilityTier},
		{"Installed", boolLabel(item.Installed)},
		{"Running", boolLabel(item.Running)},
	}
	if item.Healthy != nil {
		rows = append(rows, []string{"Healthy", boolLabel(*item.Healthy)})
	}
	if item.StatusCode != "" {
		rows = append(rows, []string{"Status Code", item.StatusCode})
	}
	rows = append(rows, []string{"Status", item.Message})
	if item.Resource.ControlMode == "legacy-adapter" {
		rows = append(rows,
			[]string{"Adapter Owner", item.Resource.LegacyAdapter.Owner},
			[]string{"Decision Deadline", item.Resource.LegacyAdapter.DecisionDeadline},
			[]string{"Final Disposition", item.Resource.LegacyAdapter.FinalDisposition},
			[]string{"Legacy CLI", item.Resource.LegacyAdapter.LegacyCLIPath},
		)
		if item.Resource.LegacyAdapter.Notes != "" {
			rows = append(rows, []string{"Adapter Notes", item.Resource.LegacyAdapter.Notes})
		}
	}
	if item.ProbeError != "" {
		rows = append(rows, []string{"Probe Error", item.ProbeError})
	}
	return cliout.RenderTable(w, []string{"Field", "Value"}, rows)
}

func writeControlReport(w io.Writer, format cliout.Format, successKey string, positiveVerb string, report any, stopped []control.ResultItem, failed []control.ResultItem) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success":  true,
			successKey: report,
		})
	}
	for _, item := range stopped {
		_, _ = fmt.Fprintf(w, "%s %s\n", positiveVerb, item.Name)
	}
	for _, item := range failed {
		_, _ = fmt.Fprintf(w, "Failed %s: %s\n", item.Name, item.Error)
	}
	return nil
}
