package events

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"

	eventsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/events"
	eventsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/events/events_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"web-console/cli/internal/support"
)

// Register exposes `web-console events` as a flat command since the
// events surface is a single read-only Connect RPC
// (EventsService.List).
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

func newClient(core *cliapp.ScenarioApp) eventsconnect.EventsServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return eventsconnect.NewEventsServiceClient(httpClient, baseURL)
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("events")
	limit := fs.Int("limit", 50, "Maximum events to return (max 1000)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	resp, err := newClient(core).List(context.Background(), connect.NewRequest(&eventsv1.ListRequest{
		Limit: int32(*limit),
	}))
	if err != nil {
		return cliapp.WrapAPIError("events list", err, nil)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Events returned: %d (limit=%d, total=%d)", len(resp.Msg.GetEvents()), *limit, resp.Msg.GetTotal())},
		ResultsHeading: "Events",
		Results:        eventRows(resp.Msg.GetEvents()),
		RetrievalHints: []string{fmt.Sprintf("%s events --limit 200", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func eventRows(events []*eventsv1.Event) []string {
	if len(events) == 0 {
		return []string{"(no recent events)"}
	}
	rows := make([]string, 0, len(events))
	for _, e := range events {
		rows = append(rows, fmt.Sprintf("%s | %s | session=%s",
			support.FormatTime(e.GetTimestamp()), e.GetType(), support.ShortID(e.GetSessionId())))
	}
	return rows
}
