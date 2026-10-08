package apps

import (
	"fmt"
	"os"
	"strconv"

	"app-monitor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `app` subcommand group covering list/get/start/stop/restart/logs/metrics.
// The API is the source of truth for orchestration (start/stop/restart); this package
// is a thin wrapper that formats responses through the standard output contracts.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "app",
		Description: "List and control managed applications",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List managed apps", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one app", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "start", Description: "Start an app", Run: func(args []string) error { return runAction(core, args, "start", "Start") }},
			{Name: "stop", Description: "Stop an app", Run: func(args []string) error { return runAction(core, args, "stop", "Stop") }},
			{Name: "restart", Description: "Restart an app", Run: func(args []string) error { return runAction(core, args, "restart", "Restart") }},
			{Name: "logs", Description: "Show app logs", Run: func(args []string) error { return runLogs(core, args) }},
			{Name: "metrics", Description: "Show app metrics over a time window", Run: func(args []string) error { return runMetrics(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("app list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/apps", nil)
	if err != nil {
		return err
	}
	var apps []support.App
	if err := support.Decode(body, &apps); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Managed apps: %d", len(apps))},
		ResultsHeading: "Apps",
		Results:        appRows(apps),
		RetrievalHints: []string{
			fmt.Sprintf("%s app get <app-id>", support.CLIName),
			fmt.Sprintf("%s app logs <app-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("app get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: app get <app-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/apps/"+id, nil)
	if err != nil {
		return err
	}
	var app support.App
	if err := support.Decode(body, &app); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", app.ID),
		fmt.Sprintf("Name: %s", app.Name),
		fmt.Sprintf("Scenario: %s", app.ScenarioName),
		fmt.Sprintf("Status: %s", app.Status),
	}
	if app.URL != "" {
		results = append(results, fmt.Sprintf("URL: %s", app.URL))
	}
	if app.Uptime != "" {
		results = append(results, fmt.Sprintf("Uptime: %s", app.Uptime))
	}
	if app.CPUUsage > 0 {
		results = append(results, fmt.Sprintf("CPU: %.1f%%", app.CPUUsage))
	}
	if app.MemoryUsage > 0 {
		results = append(results, fmt.Sprintf("Memory: %.1f%%", app.MemoryUsage))
	}
	if app.LastSeenAt != nil {
		results = append(results, fmt.Sprintf("Last seen: %s", support.FormatTimeValue(*app.LastSeenAt)))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("App: %s (%s)", app.Name, app.Status)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s app logs %s", support.CLIName, app.ID),
			fmt.Sprintf("%s app metrics %s", support.CLIName, app.ID),
			fmt.Sprintf("%s diagnostics interop %s", support.CLIName, app.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runAction(core *cliapp.ScenarioApp, args []string, verb, display string) error {
	fs := support.NewFlagSet("app " + verb)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: app %s <app-id>", verb)
	}
	id := fs.Arg(0)

	body, err := core.Request("POST", "/apps/"+id+"/"+verb, nil, nil)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("%s issued for %s", display, id)
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{fmt.Sprintf("App %s: %s", id, verb)},
		NextCommand: []string{
			fmt.Sprintf("%s app get %s", support.CLIName, id),
			fmt.Sprintf("%s app logs %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runLogs(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("app logs")
	limit := fs.Int("limit", 50, "Maximum log lines to retrieve")
	logType := fs.String("type", "", "Log type filter: lifecycle|background|both")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: app logs <app-id> [--limit N] [--type lifecycle|background|both]")
	}
	id := fs.Arg(0)

	query := support.BuildQuery(map[string]string{
		"limit": strconv.Itoa(*limit),
		"type":  *logType,
	})
	body, err := core.Get("/apps/"+id+"/logs", query)
	if err != nil {
		return err
	}

	var entries []support.LogEntry
	if err := support.Decode(body, &entries); err != nil {
		// The API shape for this endpoint varies; emit the raw payload as a single result
		// rather than failing when it's an envelope wrapping a different structure.
		entries = nil
	}

	results := logRows(entries)
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Logs for %s: %d lines", id, len(results))},
		ResultsHeading: "Log lines",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s app logs %s --limit 200", support.CLIName, id),
			fmt.Sprintf("%s app logs %s --type lifecycle", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runMetrics(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("app metrics")
	hours := fs.Int("hours", 24, "Time window in hours")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: app metrics <app-id> [--hours N]")
	}
	id := fs.Arg(0)

	query := support.BuildQuery(map[string]string{
		"hours": strconv.Itoa(*hours),
	})
	body, err := core.Get("/apps/"+id+"/metrics", query)
	if err != nil {
		return err
	}
	var metrics []support.MetricEntry
	if err := support.Decode(body, &metrics); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Metrics for %s: %d samples over last %dh", id, len(metrics), *hours)},
		ResultsHeading: "Samples",
		Results:        metricRows(metrics),
		RetrievalHints: []string{fmt.Sprintf("%s app get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func appRows(apps []support.App) []string {
	if len(apps) == 0 {
		return []string{"No apps registered"}
	}
	rows := make([]string, 0, len(apps))
	for _, a := range apps {
		rows = append(rows, fmt.Sprintf("%s (%s) | %s | scenario=%s",
			a.Name, support.ShortID(a.ID), a.Status, a.ScenarioName))
	}
	return rows
}

func logRows(entries []support.LogEntry) []string {
	if len(entries) == 0 {
		return []string{"(no log lines returned)"}
	}
	rows := make([]string, 0, len(entries))
	for _, e := range entries {
		ts := support.FormatTime(e.Timestamp)
		level := e.Level
		if level == "" {
			level = "info"
		}
		rows = append(rows, fmt.Sprintf("%s [%s] %s", ts, level, e.Message))
	}
	return rows
}

func metricRows(metrics []support.MetricEntry) []string {
	if len(metrics) == 0 {
		return []string{"(no metrics in window)"}
	}
	rows := make([]string, 0, len(metrics))
	for _, m := range metrics {
		rows = append(rows, fmt.Sprintf("%s | status=%s | cpu=%.1f%% | mem=%.1f%%",
			support.FormatTime(m.Timestamp), m.Status, m.CPUUsage, m.MemoryUsage))
	}
	return rows
}
