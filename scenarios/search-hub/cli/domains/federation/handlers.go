package federation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
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

// status reports federation health: each ACTIVE provider's reachability, the
// active routing strategy, and whether unified reranking is available. The
// retired classifier field remains compatibility-only.
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

func (h *handlers) providerStatusCall(ctx cliapp.OperationContext) (*structpb.Struct, error) {
	providerID := strings.TrimSpace(ctx.Positional("provider_id"))
	resp, err := h.client.ProviderStatus(context.Background(), connect.NewRequest(&routingv1.ProviderStatusRequest{ProviderId: providerID}))
	if err != nil {
		return nil, cliapp.WrapAPIError("federation provider-status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no provider status response")
	}
	return providerStatusCLIResponse(resp.Msg, resp.Header())
}

func providerStatusCLIResponse(msg *routingv1.ProviderStatusResponse, headers http.Header) (*structpb.Struct, error) {
	body, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal provider status response: %w", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode provider status response: %w", err)
	}
	if projection := selectedStatusProjectionFromHeaders(headers); len(projection) > 0 {
		payload["status_projection"] = projection
	}
	result, err := structpb.NewStruct(payload)
	if err != nil {
		return nil, fmt.Errorf("build provider status CLI response: %w", err)
	}
	return result, nil
}

func selectedStatusProjectionFromHeaders(headers http.Header) map[string]any {
	if headers == nil {
		return nil
	}
	fields := []struct{ name, header string }{
		{"identity_match", "X-Search-Hub-Selected-Owner-Identity-Match"},
		{"status_rpc_reachability", "X-Search-Hub-Selected-Owner-Status-RPC-Reachability"},
		{"endpoint_reachability", "X-Search-Hub-Selected-Owner-Endpoint-Reachability"},
		{"freshness", "X-Search-Hub-Selected-Owner-Freshness"},
		{"supported_fallback", "X-Search-Hub-Selected-Owner-Supported-Fallback"},
	}
	projection := make(map[string]any, len(fields))
	for _, field := range fields {
		if value := strings.TrimSpace(headers.Get(field.header)); value != "" {
			projection[field.name] = value
		}
	}
	if len(projection) == 0 {
		return nil
	}
	return projection
}

func (h *handlers) providerStatusReport(_ cliapp.OperationContext, payload *structpb.Struct) cliapp.ListReport {
	msg := providerStatusProto(payload)
	observed := "unknown"
	if msg.GetObservedAt() != nil {
		observed = msg.GetObservedAt().AsTime().UTC().Format(time.RFC3339)
	}
	indexAge, freshnessBudget, lastIndexed := "unreported", "unreported", "unreported"
	if health := msg.GetHealth(); health != nil {
		if health.GetIndexAge() != "" {
			indexAge = health.GetIndexAge()
		}
		if health.GetFreshnessBudget() != "" {
			freshnessBudget = health.GetFreshnessBudget()
		}
		if health.GetLastIndexedAt() != nil {
			lastIndexed = health.GetLastIndexedAt().AsTime().UTC().Format(time.RFC3339)
		}
	}
	summary := []string{
		fmt.Sprintf("Selected provider %s health read at %s. Search Hub resolved its scenario route and read its declared status/freshness endpoint when available; it did not execute a search.", msg.GetProviderId(), observed),
		fmt.Sprintf("Source freshness: age=%s budget=%s last-indexed=%s.", indexAge, freshnessBudget, lastIndexed),
	}
	var projection map[string]any
	if payload != nil {
		if value, ok := payload.GetFields()["status_projection"]; ok && value.GetStructValue() != nil {
			projection = value.GetStructValue().AsMap()
		}
	}
	if len(projection) > 0 {
		summary = append(summary, selectedStatusProjectionSummary(projection))
	}
	return cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Selected provider health",
		Results:        renderProviderHealth([]*routingv1.ProviderHealth{msg.GetHealth()}),
		RetrievalHints: selectedProviderReadinessHints(msg),
	}
}

func providerStatusProto(payload *structpb.Struct) *routingv1.ProviderStatusResponse {
	message := &routingv1.ProviderStatusResponse{}
	if payload == nil {
		return message
	}
	body, err := json.Marshal(payload.AsMap())
	if err != nil {
		return message
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, message); err != nil {
		return &routingv1.ProviderStatusResponse{}
	}
	return message
}

func selectedStatusProjectionSummary(projection map[string]any) string {
	fields := []struct{ name, label string }{
		{"identity_match", "identity"},
		{"status_rpc_reachability", "status-RPC"},
		{"endpoint_reachability", "owner-endpoint"},
		{"freshness", "freshness"},
		{"supported_fallback", "supported-fallback"},
	}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		if value, ok := projection[field.name]; ok {
			parts = append(parts, fmt.Sprintf("%s=%v", field.label, value))
		}
	}
	return "Selected status projection: " + strings.Join(parts, "; ") + "."
}

func selectedProviderReadinessHints(msg *routingv1.ProviderStatusResponse) []string {
	const noFallback = "Supported fallback: not established by this readiness read. Use only a fallback documented by the assigned task or owner contract; do not infer permission to substitute another owner."
	if msg == nil || msg.GetHealth() == nil {
		return []string{"Selected-owner readiness is unavailable; task answerability is unknown.", noFallback}
	}
	health := msg.GetHealth()
	freshness := strings.ToLower(strings.TrimSpace(health.GetIndexAge()))
	switch {
	case !health.GetReachable():
		return []string{"The selected owner's route is unreachable. This does not establish that another owner is approved or that the task succeeded.", noFallback}
	case strings.HasPrefix(freshness, "unreported:"):
		return []string{"The selected owner's route resolved, but its declared freshness source is unreported. Reachability does not establish freshness or task answerability.", noFallback}
	case strings.HasPrefix(freshness, "not_applicable:"):
		return []string{"The selected owner's route resolved, but no freshness source is declared. Do not treat route reachability as freshness or task answerability.", noFallback}
	case health.GetDegraded():
		return []string{"The selected owner's route resolved, but readiness is degraded. Review the reported freshness and degraded stages before acting.", noFallback}
	default:
		return []string{"The selected owner's route and reported freshness are health signals only; run the owner-declared operation to prove task answerability.", noFallback}
	}
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
		// The interactive LLM classifier was retired; the wire field is kept for
		// older clients and is always false, so it is never rendered as a fault.
		fmt.Sprintf("Auto-routing: %s (the LLM classifier is retired, not down). Reranker (unified ranking): %s.",
			activeStrategySummary(msg.GetActiveStrategy()), availability(msg.GetRerankerAvailable())),
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
			"`query \"<task>\" --explain` — inspect the actual selector and route-specific fallback details for that task",
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
		} else if p.GetDegraded() {
			marker = "!"
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
		if p.GetDegraded() {
			reasons := p.GetDegradedStages()
			if len(reasons) == 0 {
				reasons = []string{"provider health reports degraded"}
			}
			line += " — degraded: " + strings.Join(reasons, ", ")
		}
		if state := p.GetRecoveryState(); state != "" {
			line += " — zero-yield lifecycle: " + state
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

func activeStrategySummary(name string) string {
	if name == "" {
		return "strategy not reported"
	}
	return "strategy " + name
}

func availability(ok bool) string {
	if ok {
		return "available"
	}
	return "unavailable"
}
