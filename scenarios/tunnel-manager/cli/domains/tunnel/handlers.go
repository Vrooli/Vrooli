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

func (h *handlers) status(ctx cliapp.RunContext) error {
	resp, err := h.client.GetStatus(context.Background(), connect.NewRequest(&tunnelv1.GetStatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get tunnel status", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Status == nil {
		return fmt.Errorf("server returned no status")
	}
	results := []string{formatStatus(resp.Msg.Status)}
	if resp.Msg.LatestMetrics != nil {
		results = append(results, "latest metrics: "+formatSample(resp.Msg.LatestMetrics))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Tunnel is %s (score %d).", resp.Msg.Status.Status, resp.Msg.Status.Score)},
		ResultsHeading: "Status",
		Results:        results,
		RetrievalHints: []string{
			"`tunnel metrics` — show the scraped metrics time-series",
			"`tunnel scrape` — force one scrape now",
		},
	})
}

func (h *handlers) metrics(ctx cliapp.RunContext) error {
	req := &tunnelv1.ListMetricsRequest{}
	if v := strings.TrimSpace(ctx.Flag("from")); v != "" {
		ts, err := parseTime(v)
		if err != nil {
			return fmt.Errorf("--from must be RFC3339: %w", err)
		}
		req.From = timestamppb.New(ts)
	}
	if v := strings.TrimSpace(ctx.Flag("to")); v != "" {
		ts, err := parseTime(v)
		if err != nil {
			return fmt.Errorf("--to must be RFC3339: %w", err)
		}
		req.To = timestamppb.New(ts)
	}
	resp, err := h.client.ListMetrics(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("list tunnel metrics", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no metrics response")
	}
	results := make([]string, 0, len(resp.Msg.Samples))
	for _, s := range resp.Msg.Samples {
		results = append(results, formatSample(s))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d metrics sample(s).", len(resp.Msg.Samples))},
		ResultsHeading: "Metrics",
		Results:        results,
		RetrievalHints: []string{
			"`tunnel scrape` — capture a fresh sample",
			"`tunnel status` — show composite health",
		},
	})
}

func (h *handlers) scrape(ctx cliapp.RunContext) error {
	resp, err := h.client.Scrape(context.Background(), connect.NewRequest(&tunnelv1.ScrapeRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("scrape tunnel metrics", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Sample == nil {
		return fmt.Errorf("server returned no sample")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Scraped metrics sample %s.", resp.Msg.Sample.Id)},
		Changes: []string{formatSample(resp.Msg.Sample)},
		NextCommand: []string{
			"`tunnel metrics` — show the metrics time-series",
			"`tunnel status` — show composite health",
		},
	})
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
