package federation

import (
	"context"
	"fmt"

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
func (h *handlers) status(ctx cliapp.RunContext) error {
	resp, err := h.client.Status(context.Background(), connect.NewRequest(&routingv1.StatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("federation status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no status response")
	}
	msg := resp.Msg

	reachable := 0
	for _, p := range msg.GetProviders() {
		if p.GetReachable() {
			reachable++
		}
	}

	summary := []string{
		fmt.Sprintf("%d/%d active provider(s) reachable.", reachable, len(msg.GetProviders())),
		fmt.Sprintf("Classifier (auto-routing): %s. Reranker (unified ranking): %s.",
			availability(msg.GetClassifierAvailable()), availability(msg.GetRerankerAvailable())),
	}
	if !msg.GetRerankerAvailable() {
		summary = append(summary, "Reranker unavailable ⇒ queries degrade to honest by-provider grouping.")
	}

	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Provider health",
		Results:        renderProviderHealth(msg.GetProviders()),
		RetrievalHints: []string{
			"`providers list` — see every registered provider and its descriptor",
			"`insights` — telemetry: utilization, zero-result rate, latency",
			"`query \"<text>\" --all` — run a federated search",
		},
	})
}

// renderProviderHealth renders each leaf's reachability + freshness note.
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
		if note := p.GetFreshness(); note != "" {
			line += " — " + note
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
