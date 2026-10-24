package tunnel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	tunnelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/tunnel"
	tunnelconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/tunnel/tunnel_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client tunnelconnect.TunnelServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: tunnelconnect.NewTunnelServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) statusCall(_ cliapp.OperationContext) (*tunnelv1.GetStatusResponse, error) {
	resp, err := h.client.GetStatus(context.Background(), connect.NewRequest(&tunnelv1.GetStatusRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get tunnel status", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Status == nil {
		return nil, fmt.Errorf("server returned no status")
	}
	return resp.Msg, nil
}

func (h *handlers) statusReport(_ cliapp.OperationContext, message *tunnelv1.GetStatusResponse) cliapp.ListReport {
	results := []string{formatStatus(message.Status)}
	if message.LatestMetrics != nil {
		results = append(results, "latest metrics: "+formatSample(message.LatestMetrics))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Tunnel is %s (score %d).", message.Status.Status, message.Status.Score)},
		ResultsHeading: "Status",
		Results:        results,
		RetrievalHints: []string{
			"`tunnel metrics` — show the scraped metrics time-series",
			"`tunnel scrape` — force one scrape now",
		},
	}
}

func (h *handlers) metricsCall(ctx cliapp.OperationContext) (*tunnelv1.ListMetricsResponse, error) {
	req := &tunnelv1.ListMetricsRequest{}
	if v := strings.TrimSpace(ctx.Flag("from")); v != "" {
		ts, err := parseTime(v)
		if err != nil {
			return nil, fmt.Errorf("--from must be RFC3339: %w", err)
		}
		req.From = timestamppb.New(ts)
	}
	if v := strings.TrimSpace(ctx.Flag("to")); v != "" {
		ts, err := parseTime(v)
		if err != nil {
			return nil, fmt.Errorf("--to must be RFC3339: %w", err)
		}
		req.To = timestamppb.New(ts)
	}
	resp, err := h.client.ListMetrics(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("list tunnel metrics", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no metrics response")
	}
	return resp.Msg, nil
}

func (h *handlers) metricsReport(_ cliapp.OperationContext, message *tunnelv1.ListMetricsResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.Samples))
	for _, s := range message.Samples {
		results = append(results, formatSample(s))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d metrics sample(s).", len(message.Samples))},
		ResultsHeading: "Metrics",
		Results:        results,
		RetrievalHints: []string{
			"`tunnel scrape` — capture a fresh sample",
			"`tunnel status` — show composite health",
		},
	}
}

func (h *handlers) scrapeCall(_ cliapp.OperationContext) (*tunnelv1.ScrapeResponse, error) {
	resp, err := h.client.Scrape(context.Background(), connect.NewRequest(&tunnelv1.ScrapeRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("scrape tunnel metrics", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Sample == nil {
		return nil, fmt.Errorf("server returned no sample")
	}
	return resp.Msg, nil
}

func (h *handlers) scrapeReport(_ cliapp.OperationContext, message *tunnelv1.ScrapeResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Scraped metrics sample %s.", message.Sample.Id)},
		Changes: []string{formatSample(message.Sample)},
		NextCommand: []string{
			"`tunnel metrics` — show the metrics time-series",
			"`tunnel status` — show composite health",
		},
	}
}

// parseTime accepts RFC3339 / RFC3339Nano timestamps.
func parseTime(v string) (time.Time, error) {
	if ts, err := time.Parse(time.RFC3339Nano, v); err == nil {
		return ts, nil
	}
	return time.Parse(time.RFC3339, v)
}

func formatStatus(s *tunnelv1.TunnelStatus) string {
	if s == nil {
		return "(nil)"
	}
	msg := s.Message
	if msg == "" {
		msg = "ok"
	}
	return fmt.Sprintf("%s — systemd=%s ready=%s (%dms) score=%d [%s]",
		s.Status, s.Systemd, s.Ready, s.ReadyLatencyMs, s.Score, msg)
}

func formatSample(s *tunnelv1.MetricsSample) string {
	if s == nil {
		return "(nil)"
	}
	at := ""
	if s.ScrapedAt != nil {
		at = s.ScrapedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("ha_connections=%d request_errors=%g active_streams=%d smoothed_rtt_ms=%g [at=%s, id=%s]",
		s.HaConnections, s.RequestErrors, s.ActiveStreams, s.SmoothedRttMs, at, s.Id)
}
