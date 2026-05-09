package timeline

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Commands(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Forensics",
		Commands: []cliapp.Command{
			{
				Name:        "timeline",
				NeedsAPI:    true,
				Description: "Show chronological system events and correlation hints",
				Run: func(args []string) error {
					return systemTimeline(core, args)
				},
			},
		},
	}
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "actions",
		Description: "Inspect timeline, status transitions, uptime, and action history",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "history", Description: "Show recent recovery action history", Run: func(args []string) error { return actionHistory(core, args) }},
			{Name: "timeline", Description: "Show recent timeline events", Run: func(args []string) error { return timeline(core, args) }},
			{Name: "transitions", Description: "Show recent status transitions", Run: func(args []string) error { return transitions(core, args) }},
			{Name: "uptime", Description: "Show uptime summary", Run: func(args []string) error { return uptime(core, args) }},
			{Name: "trends", Description: "Show check trends", Run: func(args []string) error { return trends(core, args) }},
		},
	}
}

func systemTimeline(core *cliapp.ScenarioApp, args []string) error {
	if len(args) > 0 && args[0] == "refresh" {
		return refreshSystemTimeline(core, args[1:])
	}
	fs := support.NewFlagSet("timeline")
	since := fs.String("since", "72h", "Start time or duration ago")
	until := fs.String("until", "", "End time")
	category := fs.String("category", "", "Comma-separated categories")
	severity := fs.String("severity", "", "Comma-separated severities")
	source := fs.String("source", "", "Comma-separated sources")
	limit := fs.Int("limit", 100, "Maximum events")
	correlate := fs.Bool("correlate", true, "Include deterministic correlation hints")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	query := url.Values{
		"limit": []string{strconv.Itoa(*limit)},
	}
	if strings.TrimSpace(*since) != "" {
		query.Set("since", strings.TrimSpace(*since))
	}
	if strings.TrimSpace(*until) != "" {
		query.Set("until", strings.TrimSpace(*until))
	}
	if strings.TrimSpace(*category) != "" {
		query.Set("category", strings.TrimSpace(*category))
	}
	if strings.TrimSpace(*severity) != "" {
		query.Set("severity", strings.TrimSpace(*severity))
	}
	if strings.TrimSpace(*source) != "" {
		query.Set("source", strings.TrimSpace(*source))
	}
	if *correlate {
		query.Set("correlate", "true")
	}
	body, err := core.Get("/system-events", query)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	var resp support.SystemEventsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	summary := []string{fmt.Sprintf("System events: %d", resp.Count)}
	if len(resp.Sources) > 0 {
		sourceStates := make([]string, 0, len(resp.Sources))
		for _, source := range resp.Sources {
			sourceStates = append(sourceStates, fmt.Sprintf("%s=%s", source.Source, source.Status))
		}
		summary = append(summary, "Sources: "+strings.Join(sourceStates, ", "))
	}
	var results []string
	for _, event := range resp.Events {
		results = append(results, fmt.Sprintf("%s [%s/%s] %s: %s", event.OccurredAt.Format("2006-01-02 15:04:05Z07:00"), event.Severity, event.Category, event.Title, event.Summary))
	}
	if len(results) == 0 {
		results = []string{"No system events matched the selected filters."}
	}
	if len(resp.Correlations) > 0 {
		summary = append(summary, "Correlation hints:")
		for _, hint := range resp.Correlations {
			summary = append(summary, fmt.Sprintf("- %s: %s", hint.Title, hint.Summary))
		}
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Events",
		Results:        results,
		RetrievalHints: []string{"vrooli-autoheal timeline refresh", "vrooli-autoheal timeline --since 7d --category kernel,driver --json"},
	})
}

func refreshSystemTimeline(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("timeline refresh")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Request("POST", "/system-events/refresh", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	var resp support.SystemEventsRefreshResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("System event refresh completed in %dms.", resp.DurationMs),
			fmt.Sprintf("Inserted: %d, deduped: %d", resp.Ingested, resp.Deduped),
		},
		ResultsHeading: "Sources",
		Results:        systemEventSourceLines(resp.Sources),
	})
}

func systemEventSourceLines(sources []support.SystemEventSource) []string {
	if len(sources) == 0 {
		return []string{"No source status reported."}
	}
	lines := make([]string, 0, len(sources))
	for _, source := range sources {
		line := fmt.Sprintf("%s (%s): %s", source.Source, source.Platform, source.Status)
		if source.LastError != "" {
			line += " - " + source.LastError
		}
		lines = append(lines, line)
	}
	return lines
}

func actionHistory(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("actions history")
	checkID := fs.String("check-id", "", "Filter by check id")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	query := url.Values{}
	if *checkID != "" {
		query.Set("checkId", *checkID)
	}
	body, err := core.Get("/actions/history", query)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	var resp support.ActionLogsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	results := make([]string, 0, len(resp.Logs))
	for _, log := range resp.Logs {
		results = append(results, fmt.Sprintf("%s %s/%s success=%s: %s", log.Timestamp, log.CheckID, log.ActionID, support.BoolWord(log.Success), log.Message))
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Recovery action history entries: %d", resp.Total)},
		ResultsHeading: "Actions",
		Results:        results,
		RetrievalHints: []string{"vrooli-autoheal check actions <check-id>"},
	})
}

func timeline(core *cliapp.ScenarioApp, args []string) error {
	return renderJSONOnlyGet(core, "/timeline", args)
}

func transitions(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("actions transitions")
	hours := fs.Int("hours", 24, "Transition window in hours")
	limit := fs.Int("limit", 50, "Maximum transitions")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	query := url.Values{
		"hours": []string{strconv.Itoa(*hours)},
		"limit": []string{strconv.Itoa(*limit)},
	}
	body, err := core.Get("/transitions", query)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
	return nil
}

func uptime(core *cliapp.ScenarioApp, args []string) error {
	return renderJSONOnlyGet(core, "/uptime", args)
}

func trends(core *cliapp.ScenarioApp, args []string) error {
	return renderJSONOnlyGet(core, "/checks/trends", args)
}

func renderJSONOnlyGet(core *cliapp.ScenarioApp, path string, args []string) error {
	fs := support.NewFlagSet(path)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Get(path, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
	return nil
}
