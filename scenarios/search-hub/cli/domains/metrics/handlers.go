package metrics

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	shmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/measures"
	measuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/measures/measures_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client measuresconnect.MeasuresServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: measuresconnect.NewMeasuresServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) federatedLatencyCall(ctx cliapp.OperationContext) (*shmeasuresv1.FederatedLatencyResponse, error) {
	resp, err := h.client.FederatedLatency(context.Background(), connect.NewRequest(&shmeasuresv1.FederatedLatencyRequest{
		Window: timeWindow(ctx.Flag("window")),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("federated latency", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no federated latency response")
	}
	return resp.Msg, nil
}

func (h *handlers) federatedLatencyReport(ctx cliapp.OperationContext, msg *shmeasuresv1.FederatedLatencyResponse) cliapp.ListReport {
	return cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Federated latency for %s: p50 %dms, p95 %dms.", windowLabel(ctx.Flag("window")), msg.GetP50Ms(), msg.GetP95Ms()),
		},
		ResultsHeading: "Latency",
		Results: []string{
			fmt.Sprintf("p50_ms: %d", msg.GetP50Ms()),
			fmt.Sprintf("p95_ms: %d", msg.GetP95Ms()),
		},
		RetrievalHints: []string{"`--window <token>` — use this_week, last_7d, last_30d, this_month, last_month, or this_quarter"},
	}
}

func (h *handlers) degradedQueryRateCall(ctx cliapp.OperationContext) (*shmeasuresv1.DegradedQueryRateResponse, error) {
	resp, err := h.client.DegradedQueryRate(context.Background(), connect.NewRequest(&shmeasuresv1.DegradedQueryRateRequest{
		Window: timeWindow(ctx.Flag("window")),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("degraded query rate", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no degraded query rate response")
	}
	return resp.Msg, nil
}

func (h *handlers) degradedQueryRateReport(ctx cliapp.OperationContext, msg *shmeasuresv1.DegradedQueryRateResponse) cliapp.ListReport {
	return cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Degraded query rate for %s: %.2f%%.", windowLabel(ctx.Flag("window")), msg.GetRate()*100),
		},
		ResultsHeading: "Query degradation",
		Results: []string{
			fmt.Sprintf("rate: %.6f", msg.GetRate()),
			fmt.Sprintf("degraded_queries: %d", msg.GetDegradedQueries()),
			fmt.Sprintf("total_queries: %d", msg.GetTotalQueries()),
		},
		RetrievalHints: []string{"`federation status` — inspect currently reachable providers"},
	}
}

func (h *handlers) providerDegradationRateCall(ctx cliapp.OperationContext) (*shmeasuresv1.ProviderDegradationRateResponse, error) {
	resp, err := h.client.ProviderDegradationRate(context.Background(), connect.NewRequest(&shmeasuresv1.ProviderDegradationRateRequest{
		Window:     timeWindow(ctx.Flag("window")),
		ProviderId: strings.TrimSpace(ctx.Flag("provider-id")),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("provider degradation rate", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no provider degradation rate response")
	}
	return resp.Msg, nil
}

func (h *handlers) providerDegradationRateReport(ctx cliapp.OperationContext, msg *shmeasuresv1.ProviderDegradationRateResponse) cliapp.ListReport {
	scope := strings.TrimSpace(ctx.Flag("provider-id"))
	if scope == "" {
		scope = "all providers"
	}
	return cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Provider degradation rate for %s over %s: %.2f%%.", scope, windowLabel(ctx.Flag("window")), msg.GetRate()*100),
		},
		ResultsHeading: "Provider degradation",
		Results: []string{
			fmt.Sprintf("rate: %.6f", msg.GetRate()),
			fmt.Sprintf("degraded_count: %d", msg.GetDegradedCount()),
			fmt.Sprintf("times_routed: %d", msg.GetTimesRouted()),
		},
		RetrievalHints: []string{"`--provider-id <provider_id>` — scope the rate to one provider"},
	}
}

func timeWindow(raw string) *measuresv1.TimeWindow {
	return &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{Token: timeWindowToken(raw)}}
}

func timeWindowToken(raw string) measuresv1.TimeWindowToken {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "this_week":
		return measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_WEEK
	case "last_7d":
		return measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D
	case "last_30d":
		return measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_30D
	case "this_month":
		return measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_MONTH
	case "last_month":
		return measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_MONTH
	case "this_quarter":
		return measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_QUARTER
	default:
		return measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_WEEK
	}
}

func windowLabel(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "this_week"
	}
	return strings.TrimSpace(raw)
}
