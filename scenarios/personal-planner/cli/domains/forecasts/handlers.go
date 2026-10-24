package forecasts

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/forecasts"
	c "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/forecasts/forecasts_v1connect"
)

type handlers struct{ client c.ForecastsServiceClient }

func newHandlers(app *cliapp.ScenarioApp) *handlers {
	httpClient, base := cliapp.NewConnectHTTPClient(app)
	return &handlers{client: c.NewForecastsServiceClient(httpClient, base)}
}

func (h *handlers) getCall(cxt cliapp.OperationContext) (*v.GetForecastResponse, error) {
	days := int32(28)
	if raw := cxt.Flag("horizon-days"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 32)
		if err != nil { return nil, fmt.Errorf("horizon-days must be an integer: %w", err) }
		days = int32(parsed)
	}
	r, err := h.client.GetForecast(context.Background(), connect.NewRequest(&v.GetForecastRequest{LocalDate: cxt.Flag("local-date"), Timezone: cxt.Flag("timezone"), HorizonDays: days}))
	if err != nil { return nil, cliapp.WrapAPIError("get forecast", err, nil) }
	if r == nil || r.Msg == nil || r.Msg.Forecast == nil { return nil, fmt.Errorf("server returned no forecast") }
	return r.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, m *v.GetForecastResponse) cliapp.MutationReport {
	x := m.Forecast
	central := x.CentralFinish
	if central == "" { central = "not available" }
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Forecast: %s; cautious: %s; risk: %s.", central, valueOr(x.CautiousFinish, "not available"), x.RiskState)}, Changes: []string{x.Explanation}, NextCommand: []string{"`plan` — compare the outlook with accepted capacity"}}
}

func (h *handlers) historyCall(cxt cliapp.OperationContext) (*v.ListForecastSnapshotsResponse, error) {
	limit := int32(8)
	if raw := cxt.Flag("limit"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 32)
		if err != nil { return nil, fmt.Errorf("limit must be an integer: %w", err) }
		limit = int32(parsed)
	}
	r, err := h.client.ListForecastSnapshots(context.Background(), connect.NewRequest(&v.ListForecastSnapshotsRequest{Limit: limit}))
	if err != nil { return nil, cliapp.WrapAPIError("list forecast history", err, nil) }
	if r == nil || r.Msg == nil { return nil, fmt.Errorf("server returned no forecast history") }
	return r.Msg, nil
}

func (h *handlers) historyReport(_ cliapp.OperationContext, m *v.ListForecastSnapshotsResponse) cliapp.ListReport {
	out := make([]string, 0, len(m.Snapshots))
	for _, x := range m.Snapshots {
		out = append(out, fmt.Sprintf("%s — central %s; cautious %s; %s", x.GeneratedAt, valueOr(x.CentralFinish, "not available"), valueOr(x.CautiousFinish, "not available"), valueOr(x.RiskState, "unknown")))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d forecast snapshot(s).", len(m.Snapshots))}, ResultsHeading: "Recent forecasts", Results: out}
}

func valueOr(value, fallback string) string { if value == "" { return fallback }; return value }
