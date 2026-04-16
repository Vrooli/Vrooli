// Package event wraps the /api/v1/events endpoints: create and list.
package event

import (
	"fmt"
	"os"
	"strings"

	"time-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `event` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "event",
		Description: "Manage scheduled events (create, list)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "create", Description: "Create a new scheduled event from a JSON body file", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "list", Aliases: []string{"ls"}, Description: "List scheduled events", Run: func(args []string) error { return runList(core, args) }},
		},
	}
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("event create")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the event payload (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/events", nil, payload)
	if err != nil {
		return err
	}
	var event support.ScheduledEvent
	if err := support.Decode(body, &event); err != nil {
		return err
	}

	changes := []string{fmt.Sprintf("ID: %s", event.ID)}
	if event.Title != "" {
		changes = append(changes, fmt.Sprintf("Title: %s", event.Title))
	}
	if event.StartTime != "" {
		changes = append(changes, fmt.Sprintf("Start: %s", event.StartTime))
	}
	if event.EndTime != "" {
		changes = append(changes, fmt.Sprintf("End: %s", event.EndTime))
	}
	if event.Status != "" {
		changes = append(changes, fmt.Sprintf("Status: %s", event.Status))
	}

	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created event %s", event.ID)},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s event list --organizer %s", support.CLIName, event.OrganizerID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("event list")
	organizerID := fs.String("organizer", "", "Filter by organizer ID")
	startDate := fs.String("start-date", "", "Earliest start_time (RFC3339)")
	endDate := fs.String("end-date", "", "Latest end_time (RFC3339)")
	status := fs.String("status", "", "Filter by status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"organizer_id": *organizerID,
		"start_date":   *startDate,
		"end_date":     *endDate,
		"status":       *status,
	})

	body, err := core.Get("/events", query)
	if err != nil {
		return err
	}
	var resp support.EventsListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	rows := make([]string, 0, len(resp.Events))
	for _, e := range resp.Events {
		id := e.ID
		if len(id) > 8 {
			id = id[:8]
		}
		parts := []string{fmt.Sprintf("%s (%s)", e.Title, id)}
		if e.StartTime != "" {
			parts = append(parts, e.StartTime)
		}
		if e.Status != "" {
			parts = append(parts, fmt.Sprintf("status=%s", e.Status))
		}
		rows = append(rows, strings.Join(parts, " | "))
	}
	if len(rows) == 0 {
		rows = []string{"(no events)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Events: %d", resp.Count)},
		ResultsHeading: "Events",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s event create --body-file ./event.json", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
