package events

import (
	"fmt"
	"os"
	"strings"

	"calendar/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `event` subcommand group covering CRUD on /api/v1/events.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "event",
		Description: "Create, list, and manage calendar events",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List events", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one event", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Description: "Create a new event", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", Description: "Update an event", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Description: "Delete an event", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("event list")
	startDate := fs.String("start-date", "", "Start of date range (RFC3339)")
	endDate := fs.String("end-date", "", "End of date range (RFC3339)")
	eventType := fs.String("type", "", "Filter by event type")
	search := fs.String("search", "", "Search events by title/description")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"start_date": *startDate,
		"end_date":   *endDate,
		"event_type": *eventType,
		"search":     *search,
	})
	body, err := core.Get("/events", query)
	if err != nil {
		return err
	}
	var resp support.EventListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Events: %d", resp.TotalCount)}
	if resp.Timezone != "" {
		summary = append(summary, fmt.Sprintf("Timezone: %s", resp.Timezone))
	}
	if resp.HasMore {
		summary = append(summary, "(results capped — narrow filters with --start-date/--end-date/--type)")
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Events",
		Results:        eventRows(resp.Events),
		RetrievalHints: []string{
			fmt.Sprintf("%s event get <event-id>", support.CLIName),
			fmt.Sprintf("%s event list --search <query>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("event get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: event get <event-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/events/"+id, nil)
	if err != nil {
		return err
	}
	var event support.Event
	if err := support.Decode(body, &event); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", event.ID),
		fmt.Sprintf("Title: %s", event.Title),
	}
	if event.StartTime != "" {
		results = append(results, fmt.Sprintf("Start: %s", support.FormatTime(event.StartTime)))
	}
	if event.EndTime != "" {
		results = append(results, fmt.Sprintf("End: %s", support.FormatTime(event.EndTime)))
	}
	if event.Timezone != "" {
		results = append(results, fmt.Sprintf("Timezone: %s", event.Timezone))
	}
	if event.EventType != "" {
		results = append(results, fmt.Sprintf("Type: %s", event.EventType))
	}
	if event.Location != "" {
		results = append(results, fmt.Sprintf("Location: %s", event.Location))
	}
	if event.Status != "" {
		results = append(results, fmt.Sprintf("Status: %s", event.Status))
	}
	if event.Description != "" {
		results = append(results, fmt.Sprintf("Description: %s", event.Description))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Event: %s", event.Title)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s event update %s --title \"...\"", support.CLIName, event.ID),
			fmt.Sprintf("%s event delete %s", support.CLIName, event.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("event create")
	title := fs.String("title", "", "Event title (required)")
	start := fs.String("start", "", "Start time, RFC3339 (required)")
	end := fs.String("end", "", "End time, RFC3339 (required)")
	location := fs.String("location", "", "Event location")
	eventType := fs.String("type", "", "Event type (meeting, appointment, etc.)")
	description := fs.String("description", "", "Event description")
	timezone := fs.String("timezone", "", "Event timezone (e.g. America/Los_Angeles)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*title) == "" {
		return fmt.Errorf("--title is required")
	}
	if strings.TrimSpace(*start) == "" {
		return fmt.Errorf("--start is required (RFC3339)")
	}
	if strings.TrimSpace(*end) == "" {
		return fmt.Errorf("--end is required (RFC3339)")
	}

	payload := map[string]interface{}{
		"title":      *title,
		"start_time": *start,
		"end_time":   *end,
	}
	if strings.TrimSpace(*location) != "" {
		payload["location"] = *location
	}
	if strings.TrimSpace(*eventType) != "" {
		payload["event_type"] = *eventType
	}
	if strings.TrimSpace(*description) != "" {
		payload["description"] = *description
	}
	if strings.TrimSpace(*timezone) != "" {
		payload["timezone"] = *timezone
	}

	body, err := core.Request("POST", "/events", nil, payload)
	if err != nil {
		return err
	}

	var resp support.EventCreateResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	eventID := ""
	if id, ok := resp.Event["id"].(string); ok {
		eventID = id
	}

	changes := []string{fmt.Sprintf("Title: %s", *title)}
	if eventID != "" {
		changes = append(changes, fmt.Sprintf("ID: %s", eventID))
	}
	if resp.RemindersScheduled > 0 {
		changes = append(changes, fmt.Sprintf("Reminders scheduled: %d", resp.RemindersScheduled))
	}

	next := []string{}
	if eventID != "" {
		next = append(next, fmt.Sprintf("%s event get %s", support.CLIName, eventID))
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created event %q", *title)},
		Changes:     changes,
		NextCommand: next,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("event update")
	title := fs.String("title", "", "New title")
	start := fs.String("start", "", "New start time, RFC3339")
	end := fs.String("end", "", "New end time, RFC3339")
	location := fs.String("location", "", "New location")
	eventType := fs.String("type", "", "New event type")
	description := fs.String("description", "", "New description")
	timezone := fs.String("timezone", "", "New timezone")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: event update <event-id> [--title ...] [--start ...] [--end ...] [--location ...] [--type ...] [--description ...] [--timezone ...]")
	}
	id := fs.Arg(0)

	payload := map[string]interface{}{}
	addStr := func(key, value string) {
		if strings.TrimSpace(value) != "" {
			payload[key] = value
		}
	}
	addStr("title", *title)
	addStr("start_time", *start)
	addStr("end_time", *end)
	addStr("location", *location)
	addStr("event_type", *eventType)
	addStr("description", *description)
	addStr("timezone", *timezone)

	if len(payload) == 0 {
		return fmt.Errorf("no fields to update — pass at least one of --title, --start, --end, --location, --type, --description, --timezone")
	}

	if _, err := core.Request("PUT", "/events/"+id, nil, payload); err != nil {
		return err
	}

	changes := make([]string, 0, len(payload))
	for k, v := range payload {
		changes = append(changes, fmt.Sprintf("%s: %s", k, support.RenderValue(v)))
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated event %s", id)},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s event get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("event delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: event delete <event-id>")
	}
	id := fs.Arg(0)

	if _, err := core.Request("DELETE", "/events/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted event %s", id)},
		NextCommand: []string{fmt.Sprintf("%s event list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func eventRows(events []support.Event) []string {
	if len(events) == 0 {
		return []string{"No events found"}
	}
	rows := make([]string, 0, len(events))
	for _, e := range events {
		etype := e.EventType
		if etype == "" {
			etype = "meeting"
		}
		row := fmt.Sprintf("%s | %s | %s | %s",
			support.ShortID(e.ID),
			support.FormatTime(e.StartTime),
			etype,
			e.Title)
		if e.Location != "" {
			row += fmt.Sprintf(" @ %s", e.Location)
		}
		rows = append(rows, row)
	}
	return rows
}
