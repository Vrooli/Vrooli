package events

import (
	"fmt"
	"os"
	"strconv"

	"web-console/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `web-console events` as a flat command since the events
// surface is a single read-only list endpoint (GET /api/v1/events).
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Events",
		Commands: []cliapp.Command{
			{
				Name:        "events",
				Description: "Show recent lifecycle events from the in-memory ring buffer",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runList(core, args) },
			},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("events")
	limit := fs.Int("limit", 50, "Maximum events to return (max 1000)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"limit": strconv.Itoa(*limit),
	})
	body, err := core.Get("/events", query)
	if err != nil {
		return err
	}

	var records []support.EventRecord
	if err := support.Decode(body, &records); err != nil {
		records = nil
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Events returned: %d (limit=%d)", len(records), *limit)},
		ResultsHeading: "Events",
		Results:        eventRows(records),
		RetrievalHints: []string{fmt.Sprintf("%s events --limit 200", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func eventRows(records []support.EventRecord) []string {
	if len(records) == 0 {
		return []string{"(no recent events)"}
	}
	rows := make([]string, 0, len(records))
	for _, e := range records {
		ts := e.Timestamp
		if ts == "" {
			ts = e.OccurredAt
		}
		rows = append(rows, fmt.Sprintf("%s | %s | session=%s",
			support.FormatTime(ts), e.Type, support.ShortID(e.SessionID)))
	}
	return rows
}
