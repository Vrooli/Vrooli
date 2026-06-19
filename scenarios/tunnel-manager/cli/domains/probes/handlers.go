package probes

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	probesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/probes"
	probesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/probes/probes_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client probesconnect.ProbesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: probesconnect.NewProbesServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) run(ctx cliapp.RunContext) error {
	resp, err := h.client.RunProbes(context.Background(), connect.NewRequest(&probesv1.RunProbesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("run probes", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no probe response")
	}
	results := make([]string, 0, len(resp.Msg.Results))
	for _, r := range resp.Msg.Results {
		results = append(results, formatResult(r))
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Ran %d probe(s).", len(resp.Msg.Results))},
		Changes: results,
		NextCommand: []string{
			"`probes classify` — diagnose per-route reachability",
			"`probes history` — show recent probe history",
		},
	})
}

func (h *handlers) history(ctx cliapp.RunContext) error {
	var limit int32
	if v := strings.TrimSpace(ctx.Flag("limit")); v != "" {
		// Parse with an explicit 32-bit width so the value provably fits int32
		// (the proto Limit field) — avoids the unbounded Atoi→int32 overflow
		// gosec flags as G109.
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return fmt.Errorf("--limit must be an integer: %w", err)
		}
		limit = int32(n)
	}
	resp, err := h.client.ListProbes(context.Background(), connect.NewRequest(&probesv1.ListProbesRequest{
		Subdomain: ctx.Flag("subdomain"),
		Limit:     limit,
	}))
	if err != nil {
		return cliapp.WrapAPIError("list probe history", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no probe history response")
	}
	results := make([]string, 0, len(resp.Msg.Results))
	for _, r := range resp.Msg.Results {
		results = append(results, formatResult(r))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d probe(s).", len(resp.Msg.Results))},
		ResultsHeading: "Probe history",
		Results:        results,
		RetrievalHints: []string{
			"`probes history --subdomain <s>` — filter by route",
			"`probes run` — execute a fresh probe cycle",
		},
	})
}

func (h *handlers) classify(ctx cliapp.RunContext) error {
	resp, err := h.client.Classify(context.Background(), connect.NewRequest(&probesv1.ClassifyRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("classify routes", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no classification response")
	}
	results := make([]string, 0, len(resp.Msg.Classifications))
	for _, c := range resp.Msg.Classifications {
		results = append(results, formatClassification(c))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Classified %d route(s).", len(resp.Msg.Classifications))},
		ResultsHeading: "Classifications",
		Results:        results,
		RetrievalHints: []string{
			"`probes run` — refresh probes before re-classifying",
		},
	})
}

func formatResult(r *probesv1.ProbeResult) string {
	if r == nil {
		return "(nil)"
	}
	kind := strings.TrimPrefix(strings.ToLower(r.Kind.String()), "probe_kind_")
	status := strings.TrimPrefix(strings.ToLower(r.Status.String()), "probe_status_")
	detail := fmt.Sprintf("%dms", r.LatencyMs)
	if r.StatusCode != 0 {
		detail += fmt.Sprintf(", http=%d", r.StatusCode)
	}
	if r.ErrorMsg != "" {
		detail += fmt.Sprintf(", err=%s", r.ErrorMsg)
	}
	return fmt.Sprintf("%s [%s] %s (%s)", r.Subdomain, kind, status, detail)
}

func formatClassification(c *probesv1.RouteClassification) string {
	if c == nil {
		return "(nil)"
	}
	class := strings.TrimPrefix(strings.ToLower(c.Classification.String()), "failure_class_")
	internal := strings.TrimPrefix(strings.ToLower(c.Internal.String()), "probe_status_")
	external := strings.TrimPrefix(strings.ToLower(c.External.String()), "probe_status_")
	return fmt.Sprintf("%s — %s [internal=%s, external=%s] %s", c.Subdomain, class, internal, external, c.Assessment)
}
