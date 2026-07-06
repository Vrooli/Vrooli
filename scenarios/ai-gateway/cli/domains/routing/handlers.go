package routing

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"

	"ai-gateway/cli/domains/internal/gatewayreq"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client routingconnect.RoutingServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: routingconnect.NewRoutingServiceClient(httpClient, baseURL)}
}

func (h *handlers) preview(ctx cliapp.RunContext) error {
	req, err := gatewayreq.FromContext(ctx)
	if err != nil {
		return err
	}
	resp, err := h.client.PreviewRoute(context.Background(), connect.NewRequest(&routingv1.PreviewRouteRequest{Request: req}))
	if err != nil {
		return cliapp.WrapAPIError("preview route", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no route preview")
	}
	results := routeCandidateLines(resp.Msg.GetCandidates())
	results = append(results, reasonLines(resp.Msg.GetPolicyReasons())...)
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Route preview valid=%t selected=%s fallback=%t.", resp.Msg.GetValid(), resp.Msg.GetSelectedProvider(), resp.Msg.GetFallbackAllowed())},
		ResultsHeading: "Route candidates",
		Results:        results,
		RetrievalHints: []string{"`routing execute --role <role> --input <text>` — execute and record route evidence"},
	})
}

func (h *handlers) execute(ctx cliapp.RunContext) error {
	req, err := gatewayreq.FromContext(ctx)
	if err != nil {
		return err
	}
	resp, err := h.client.ExecuteRoute(context.Background(), connect.NewRequest(&routingv1.ExecuteRouteRequest{
		Request:   req,
		InputText: ctx.Flag("input"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("execute route", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no execution response")
	}
	results := []string{formatEvidence(resp.Msg.GetEvidence())}
	if out := resp.Msg.GetOutputText(); out != "" {
		results = append(results, "output: "+out)
	}
	results = append(results, reasonLines(resp.Msg.GetPolicyReasons())...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Route execution valid=%t.", resp.Msg.GetValid())},
		Changes: results,
		NextCommand: []string{
			"`routing evidence list` — inspect recent route evidence",
		},
	})
}

func (h *handlers) evidenceList(ctx cliapp.RunContext) error {
	limit, err := parseLimit(ctx.Flag("limit"))
	if err != nil {
		return err
	}
	resp, err := h.client.ListRouteEvidence(context.Background(), connect.NewRequest(&routingv1.ListRouteEvidenceRequest{
		Limit:    limit,
		Scenario: ctx.Flag("scenario"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list route evidence", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no route evidence list")
	}
	results := make([]string, 0, len(resp.Msg.GetEvents()))
	for _, event := range resp.Msg.GetEvents() {
		results = append(results, formatEvidence(event))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d route event(s).", len(resp.Msg.GetEvents()))},
		ResultsHeading: "Route evidence",
		Results:        results,
		RetrievalHints: []string{"`routing evidence show <event-id>` — inspect one event"},
	})
}

func (h *handlers) evidenceShow(ctx cliapp.RunContext) error {
	id := ctx.Positional("event-id")
	resp, err := h.client.GetRouteEvidence(context.Background(), connect.NewRequest(&routingv1.GetRouteEvidenceRequest{EventId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("show route evidence %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetEvent() == nil {
		return fmt.Errorf("server returned no route evidence event")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched route event %s.", resp.Msg.GetEvent().GetEventId())},
		ResultsHeading: "Route evidence",
		Results:        []string{formatEvidence(resp.Msg.GetEvent())},
	})
}

func (h *handlers) health(ctx cliapp.RunContext) error {
	resp, err := h.client.ListProviderHealth(context.Background(), connect.NewRequest(&routingv1.ListProviderHealthRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list provider health", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no provider health")
	}
	items := resp.Msg.GetItems()
	results := make([]string, 0, len(items))
	for _, ph := range items {
		results = append(results, fmt.Sprintf(
			"%s role=%s kind=%s state=%s effective=%s consecutive_failures=%d last_failure_class=%s cooldown_until=%s",
			ph.GetProvider(), ph.GetRole(), ph.GetKind().String(), ph.GetState(), ph.GetEffectiveState(),
			ph.GetConsecutiveFailures(), ph.GetLastFailureClass(), ph.GetCooldownUntil()))
	}
	summary := fmt.Sprintf("Found %d provider-health record(s).", len(items))
	if len(items) == 0 {
		summary = "No provider circuit-breaker state recorded (all providers healthy)."
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Provider health",
		Results:        results,
		RetrievalHints: []string{"An `open`/`half_open` effective state explains why a provider is suppressed or probing."},
	})
}

func parseLimit(value string) (int32, error) {
	if value == "" {
		return 20, nil
	}
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("--limit must be an integer: %w", err)
	}
	return int32(n), nil
}

func routeCandidateLines(candidates []*routingv1.RouteCandidate) []string {
	lines := make([]string, 0, len(candidates))
	for _, c := range candidates {
		lines = append(lines, fmt.Sprintf("%s role=%s locality=%s selected=%t fallback=%t reasons=%v", c.GetProvider(), c.GetRole(), c.GetLocality(), c.GetSelected(), c.GetFallbackEligible(), c.GetReasons()))
	}
	return lines
}

func reasonLines(reasons []string) []string {
	lines := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		lines = append(lines, "policy: "+reason)
	}
	return lines
}

func formatEvidence(ev *routingv1.RouteEvidence) string {
	if ev == nil {
		return "(no evidence)"
	}
	return fmt.Sprintf("%s status=%s scenario=%s operation=%s provider=%s locality=%s fallback=%t latency_ms=%d redacted=%t/%t",
		ev.GetEventId(), ev.GetStatus(), ev.GetScenario(), ev.GetOperation(), ev.GetSelectedProvider(), ev.GetSelectedLocality(), ev.GetFallbackUsed(), ev.GetLatencyMs(), ev.GetPromptRedacted(), ev.GetResponseRedacted())
}
