package federation

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client routingconnect.RoutingServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: routingconnect.NewRoutingServiceClient(httpClient, baseURL),
	}
}

// status reports federation health: each ACTIVE provider's reachability plus
// whether the classifier and reranker models are available (the latter gates
// automatic routing and unified rerank respectively).
func (h *handlers) statusCall(_ cliapp.OperationContext) (*routingv1.StatusResponse, error) {
	resp, err := h.client.Status(context.Background(), connect.NewRequest(&routingv1.StatusRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("federation status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no status response")
	}
	return resp.Msg, nil
}

func (h *handlers) statusReport(_ cliapp.OperationContext, msg *routingv1.StatusResponse) cliapp.ListReport {
	reachable := 0
	for _, p := range msg.GetProviders() {
		if p.GetReachable() {
			reachable++
		}
	}

	summary := []string{
		fmt.Sprintf("%d/%d active provider(s) reachable.", reachable, len(msg.GetProviders())),
		fmt.Sprintf("Circuit-open share: %.1f%% (federation quorum %.1f%%; degraded=%t).", msg.GetCircuitOpenShare()*100, msg.GetCircuitOpenQuorum()*100, msg.GetFederationDegraded()),
		fmt.Sprintf("Classifier (auto-routing): %s. Reranker (unified ranking): %s.",
			availability(msg.GetClassifierAvailable()), availability(msg.GetRerankerAvailable())),
	}
	stuck := make([]string, 0)
	for _, p := range msg.GetProviders() {
		if p.GetStuck() {
			stuck = append(stuck, p.GetProviderId())
		}
	}
	if len(stuck) == 0 {
		summary = append(summary, "Recovery: no stuck providers.")
	} else {
		summary = append(summary, fmt.Sprintf("Recovery: stuck provider(s): %s.", strings.Join(stuck, ", ")))
	}
	if !msg.GetRerankerAvailable() {
		summary = append(summary, "Reranker unavailable ⇒ queries degrade to honest by-provider grouping.")
	}
	if incubating := msg.GetIncubating(); len(incubating) > 0 {
		summary = append(summary, fmt.Sprintf("Incubating providers: %d.", len(incubating)))
	}
	results := renderProviderHealth(msg.GetProviders())
	if audit := msg.GetAuditProviders(); len(audit) > 0 {
		summary = append(summary, fmt.Sprintf("Audit-only accounting rows: %d.", len(audit)))
		for _, p := range audit {
			results = append(results, fmt.Sprintf("Audit: %s — demotion-window-routed=%d demotion-window-hits=%d", p.GetProviderId(), p.GetTimesRouted(), p.GetTotalHits()))
		}
	}
	for _, p := range msg.GetIncubating() {
		results = append(results, fmt.Sprintf("Incubating: %s — declared=%s — next action: %s", p.GetProviderId(), p.GetDeclaredAt(), p.GetNextAction()))
	}

	return cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Provider health",
		Results:        results,
		RetrievalHints: []string{
			"`providers list` — see every registered provider and its descriptor",
			"`insights` — telemetry: utilization, zero-result rate, latency",
			"`query \"<text>\" --all` — run a federated search",
		},
	}
}

func (h *handlers) repromoteCall(ctx cliapp.OperationContext) (*routingv1.RepromoteResponse, error) {
	providerID := ctx.Positional("provider_id")
	resp, err := h.client.Repromote(context.Background(), connect.NewRequest(&routingv1.RepromoteRequest{ProviderId: providerID}))
	if err != nil {
		return nil, cliapp.WrapAPIError("federation repromote", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no repromotion response")
	}
	return resp.Msg, nil
}

func (h *handlers) repromoteReport(_ cliapp.OperationContext, msg *routingv1.RepromoteResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:  []string{msg.GetMessage()},
		Changes: []string{fmt.Sprintf("provider=%s reset=%t", msg.GetProviderId(), msg.GetReset_())},
	}
}

// renderProviderHealth renders each leaf's reachability and independently
// reported index age.
func renderProviderHealth(providers []*routingv1.ProviderHealth) []string {
	if len(providers) == 0 {
		return []string{"(no active providers registered — run `providers register`)"}
	}
	out := make([]string, 0, len(providers))
	for _, p := range providers {
		marker := "✓"
		if !p.GetReachable() {
			marker = "✗"
		}
		line := fmt.Sprintf("%s %s", marker, p.GetProviderId())
		if lifecycle := p.GetLifecycle(); lifecycle != "" && lifecycle != "production" {
			line += " — lifecycle: " + lifecycle
		}
		if note := p.GetReachability(); note != "" {
			line += " — reachability: " + note
		}
		if age := p.GetIndexAge(); age != "" {
			line += " — index age: " + age
		}
		if state := p.GetRecoveryState(); state != "" {
			line += " — recovery: " + state
		}
		if p.GetDemoted() {
			line += fmt.Sprintf(" — demoted (%s; demotion-window-routed=%d demotion-window-hits=%d)", p.GetDemotionReason(), p.GetTimesRouted(), p.GetTotalHits())
		}
		if p.GetQualityGateOptedOut() {
			line += " — quality gate opt-out: " + p.GetQualityGateOptOutReason()
		}
		if p.GetQualityWithheld() {
			line += fmt.Sprintf(" — withheld (junk leak; run=%s): %s", p.GetQualityEvidenceRunId(), p.GetQualityWithheldReason())
		}
		if !p.GetAutomaticEligible() && p.GetAutomaticExclusionReason() != "" {
			line += " — automatic: " + p.GetAutomaticExclusionReason()
		}
		out = append(out, line)
	}
	return out
}

func availability(ok bool) string {
	if ok {
		return "available"
	}
	return "unavailable"
}
