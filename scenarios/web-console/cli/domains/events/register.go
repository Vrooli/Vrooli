package events

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"

	eventsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/events"
	eventsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/events/events_v1connect"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client eventsconnect.EventsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: eventsconnect.NewEventsServiceClient(httpClient, baseURL),
	}
}

// Register exposes `web-console events` as a flat command since the events
// surface is a single read-only Connect RPC (EventsService.List). Built from
// the embedded manifest; DefaultSubcommand preserves the flat invocation.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"EventsService.List": h.run,
	}
	group, err := cliapp.LoadFromManifest(manifest, "events", bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("events: load from manifest: %w", err)
	}
	group.DefaultSubcommand = "events"
	return group, nil
}

func (h *handlers) run(rc cliapp.RunContext) error {
	limit := 50
	if raw := rc.Flag("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("invalid --limit %q: %w", raw, err)
		}
		limit = n
	}

	resp, err := h.client.List(context.Background(), connect.NewRequest(&eventsv1.ListRequest{
		Limit: int32(limit),
	}))
	if err != nil {
		return cliapp.WrapAPIError("events list", err, nil)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Events returned: %d (limit=%d, total=%d)", len(resp.Msg.GetEvents()), limit, resp.Msg.GetTotal())},
		ResultsHeading: "Events",
		Results:        eventRows(resp.Msg.GetEvents()),
		RetrievalHints: []string{fmt.Sprintf("%s events --limit 200", support.CLIName)},
	}
	if rc.JSON() {
		return cliapp.PrintReportJSON(rc.Stdout(), report)
	}
	return cliapp.RenderListReport(rc.Stdout(), report)
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
