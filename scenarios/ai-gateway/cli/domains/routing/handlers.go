package routing

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"connectrpc.com/connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	"google.golang.org/protobuf/proto"

	"ai-gateway/cli/domains/internal/gatewayreq"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client     routingconnect.RoutingServiceClient
	sourceHTTP connect.HTTPClient
}

func renderPreview(ctx cliapp.RunContext, message proto.Message) error {
	response, ok := message.(*routingv1.PreviewRouteResponse)
	if !ok {
		return fmt.Errorf("route preview renderer received %T", message)
	}
	results := routeCandidateLines(response.GetCandidates())
	results = append(results, reasonLines(response.GetPolicyReasons())...)
	return cliapp.RenderProtoList(ctx, response, cliapp.ListReport{Summary: []string{fmt.Sprintf("Route preview valid=%t selected=%s fallback=%t.", response.GetValid(), response.GetSelectedProvider(), response.GetFallbackAllowed())}, ResultsHeading: "Route candidates", Results: results, RetrievalHints: []string{"`routing execute --role <role> --input <text>` — execute and record route evidence"}})
}

func renderExecute(ctx cliapp.RunContext, message proto.Message) error {
	response, ok := message.(*routingv1.ExecuteRouteResponse)
	if !ok {
		return fmt.Errorf("route execution renderer received %T", message)
	}
	results := []string{formatEvidence(response.GetEvidence())}
	if output := response.GetOutputText(); output != "" {
		results = append(results, "output: "+output)
	}
	results = append(results, reasonLines(response.GetPolicyReasons())...)
	return cliapp.RenderProtoMutation(ctx, response, cliapp.MutationReport{Result: []string{fmt.Sprintf("Route execution valid=%t.", response.GetValid())}, Changes: results, NextCommand: []string{"`routing evidence list` — inspect recent route evidence"}})
}

func renderMediaSubmit(ctx cliapp.RunContext, message proto.Message) error {
	response, ok := message.(*routingv1.SubmitMediaResponse)
	if !ok {
		return fmt.Errorf("media submit renderer received %T", message)
	}
	execution := response.GetExecution()
	return renderMediaMutation(ctx, "Submitted", execution)
}

func (h *handlers) mediaSubmit(ctx cliapp.RunContext) error {
	request, err := gatewayreq.FromContext(ctx)
	if err != nil {
		return err
	}
	count, err := parseLimit(ctx.Flag("output-count"))
	if err != nil || count < 1 {
		return fmt.Errorf("--output-count must be a positive integer")
	}
	resp, err := h.client.SubmitMedia(context.Background(), connect.NewRequest(&routingv1.SubmitMediaRequest{Request: request, Prompt: ctx.Flag("prompt"), OutputCount: count, IdempotencyKey: ctx.Flag("idempotency-key"), OutputReference: ctx.Flag("output-reference")}))
	if err != nil {
		return cliapp.WrapAPIError("submit media", err, nil)
	}
	return renderMediaMutation(ctx, "Submitted", resp.Msg.GetExecution())
}

func (h *handlers) mediaShow(ctx cliapp.RunContext) error {
	resp, err := h.client.GetMediaExecution(context.Background(), connect.NewRequest(&routingv1.GetMediaExecutionRequest{ExecutionId: ctx.Positional("execution-id")}))
	if err != nil {
		return cliapp.WrapAPIError("show media execution", err, nil)
	}
	return renderMediaList(ctx, resp.Msg.GetExecution())
}

func (h *handlers) mediaWait(ctx cliapp.RunContext) error {
	resp, err := h.client.WaitMediaExecution(context.Background(), connect.NewRequest(&routingv1.WaitMediaExecutionRequest{ExecutionId: ctx.Positional("execution-id")}))
	if err != nil {
		return cliapp.WrapAPIError("wait for media execution", err, nil)
	}
	return renderMediaList(ctx, resp.Msg.GetExecution())
}

func (h *handlers) mediaCancel(ctx cliapp.RunContext) error {
	resp, err := h.client.CancelMediaExecution(context.Background(), connect.NewRequest(&routingv1.CancelMediaExecutionRequest{ExecutionId: ctx.Positional("execution-id")}))
	if err != nil {
		return cliapp.WrapAPIError("cancel media execution", err, nil)
	}
	return renderMediaMutation(ctx, "Cancelled", resp.Msg.GetExecution())
}

func (h *handlers) mediaRetry(ctx cliapp.RunContext) error {
	resp, err := h.client.RetryMediaExecution(context.Background(), connect.NewRequest(&routingv1.RetryMediaExecutionRequest{ExecutionId: ctx.Positional("execution-id"), IdempotencyKey: ctx.Flag("idempotency-key")}))
	if err != nil {
		return cliapp.WrapAPIError("retry media execution", err, nil)
	}
	return renderMediaMutation(ctx, "Retried", resp.Msg.GetExecution())
}

func renderMediaMutation(ctx cliapp.RunContext, action string, execution *routingv1.MediaExecution) error {
	if execution.GetErrorCode() == "outcome_unknown" {
		if action == "Cancelled" {
			action = "Unchanged"
		} else if action == "Submitted" {
			action = "Retained"
		}
	}
	return cliapp.RenderProtoMutation(ctx, &routingv1.SubmitMediaResponse{Execution: execution}, cliapp.MutationReport{Result: []string{fmt.Sprintf("%s media execution %s.", action, execution.GetExecutionId())}, Changes: mediaLines(execution)})
}

func renderMediaList(ctx cliapp.RunContext, execution *routingv1.MediaExecution) error {
	return cliapp.RenderProtoList(ctx, &routingv1.GetMediaExecutionResponse{Execution: execution}, cliapp.ListReport{Summary: []string{fmt.Sprintf("Media execution %s.", execution.GetExecutionId())}, ResultsHeading: "Media receipt", Results: mediaLines(execution)})
}

func mediaLines(execution *routingv1.MediaExecution) []string {
	if execution == nil {
		return []string{"(no execution returned)"}
	}
	cost := "unknown"
	if execution.ActualCostUsd != nil {
		cost = fmt.Sprintf("%.4f", execution.GetActualCostUsd())
	}
	lines := []string{fmt.Sprintf("status=%s model=%s actual_cost_usd=%s", execution.GetStatus(), execution.GetResolvedModel(), cost)}
	for _, output := range execution.GetOutputs() {
		lines = append(lines, fmt.Sprintf("output=%s media_type=%s bytes=%d", output.GetReference(), output.GetMediaType(), output.GetBytes()))
	}
	if execution.GetErrorCode() != "" {
		lines = append(lines, "error="+execution.GetErrorCode()+": "+strings.TrimSpace(execution.GetErrorMessage()))
	}
	return lines
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	// ExecuteRoute is bounded by GatewayRequest.timeout_ms.  Keep the CLI
	// transport from imposing a shorter, invisible deadline than that explicit
	// execution contract (notably for local-model cold starts).
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	return &handlers{client: routingconnect.NewRoutingServiceClient(httpClient, baseURL), sourceHTTP: httpClient}
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
		InputText: ctx.Flag("input-text"),
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

// evidenceProfile summarizes the bounded recent route-evidence window by
// resolved task role and model. It deliberately reports end-to-end latency
// only; provider load/prefill/decode phases and queue/TTFT are not in this
// evidence contract.
func (h *handlers) evidenceProfile(ctx cliapp.RunContext) error {
	limit, err := parseLimit(ctx.Flag("limit"))
	if err != nil {
		return err
	}
	if limit < 1 || limit > 100 {
		return fmt.Errorf("--limit must be between 1 and 100; the route-evidence API has a 100-row maximum")
	}
	resp, err := h.client.ListRouteEvidence(context.Background(), connect.NewRequest(&routingv1.ListRouteEvidenceRequest{
		Limit:    limit,
		Scenario: ctx.Flag("scenario"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("profile route evidence", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no route evidence profile")
	}
	rows := profileEvidence(resp.Msg.GetEvents())
	window := fmt.Sprintf("latest %d route events", len(resp.Msg.GetEvents()))
	if len(resp.Msg.GetEvents()) == int(limit) {
		window += " (possibly truncated at requested limit)"
	}
	if scenario := strings.TrimSpace(ctx.Flag("scenario")); scenario != "" {
		window += " for scenario " + shortLabel(scenario)
	}
	if len(rows) == 0 {
		return ctx.RenderList(cliapp.ListReport{
			Summary:        []string{"No route evidence in " + window + "."},
			ResultsHeading: "Per-role latency profile",
			RetrievalHints: []string{"This profile includes end-to-end route latency only; it cannot attribute provider phases or queue wait."},
		})
	}
	return ctx.RenderList(cliapp.ListReport{
		Summary:        []string{"Per-role end-to-end latency for " + window + ". Nearest-rank p50/p95 are descriptive of this recent sample."},
		ResultsHeading: "Role and model latency",
		Results:        rows,
		RetrievalHints: []string{"This is a bounded read of metadata-only route evidence, not a fixed quality-suite comparison. Queue, TTFT, load, prefill, decode, context and host/device peaks are unavailable."},
	})
}

type evidenceProfileKey struct {
	role, model, provider string
}

type evidenceProfileBucket struct {
	latencies                  []int64
	stageNS                    map[string][]int64
	stageMissing               map[string]int
	succeeded, failed, blocked int
	firstAt, lastAt            string
}

func profileEvidence(events []*routingv1.RouteEvidence) []string {
	buckets := map[evidenceProfileKey]*evidenceProfileBucket{}
	for _, ev := range events {
		if ev == nil {
			continue
		}
		role := ev.GetTaskRole()
		if strings.TrimSpace(role) == "" {
			role = ev.GetRole()
		}
		key := evidenceProfileKey{role: shortLabel(role), model: shortLabel(ev.GetSelectedModel()), provider: shortLabel(ev.GetSelectedProvider())}
		bucket := buckets[key]
		if bucket == nil {
			bucket = &evidenceProfileBucket{stageNS: map[string][]int64{}, stageMissing: map[string]int{}}
			buckets[key] = bucket
		}
		bucket.latencies = append(bucket.latencies, ev.GetLatencyMs())
		for _, stage := range evidenceStages {
			if value := stage.value(ev); value > 0 {
				bucket.stageNS[stage.name] = append(bucket.stageNS[stage.name], value)
			} else {
				bucket.stageMissing[stage.name]++
			}
		}
		switch strings.ToLower(ev.GetStatus()) {
		case "succeeded":
			bucket.succeeded++
		case "failed":
			bucket.failed++
		case "blocked", "refused":
			bucket.blocked++
		}
		createdAt := ev.GetCreatedAt()
		if bucket.firstAt == "" || createdAt < bucket.firstAt {
			bucket.firstAt = createdAt
		}
		if createdAt > bucket.lastAt {
			bucket.lastAt = createdAt
		}
	}
	keys := make([]evidenceProfileKey, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].role != keys[j].role {
			return keys[i].role < keys[j].role
		}
		if keys[i].model != keys[j].model {
			return keys[i].model < keys[j].model
		}
		return keys[i].provider < keys[j].provider
	})
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		b := buckets[key]
		values := append([]int64(nil), b.latencies...)
		sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
		lines = append(lines, fmt.Sprintf("role=%s model=%s provider=%s n=%d succeeded=%d failed=%d blocked=%d latency_ms[p50=%d p95=%d min=%d max=%d] stages_ns[%s] span=%s..%s",
			labelOrUnknown(key.role), labelOrUnknown(key.model), labelOrUnknown(key.provider), len(values), b.succeeded, b.failed, b.blocked,
			nearestRank(values, 50), nearestRank(values, 95), values[0], values[len(values)-1], formatStageProfile(b), b.firstAt, b.lastAt))
	}
	return lines
}

// evidenceStages are the recorded route timings, in reporting order: the
// gateway's own stages, the resource command, what the resource measured
// inside it, and the overhead outside provider work.
var evidenceStages = []struct {
	name  string
	value func(*routingv1.RouteEvidence) int64
}{
	{"prepare", (*routingv1.RouteEvidence).GetPreparationDurationNs},
	{"plan", (*routingv1.RouteEvidence).GetRoutePlanDurationNs},
	{"role", (*routingv1.RouteEvidence).GetRoleResolutionDurationNs},
	{"eligibility", (*routingv1.RouteEvidence).GetEligibilityDurationNs},
	{"capacity", (*routingv1.RouteEvidence).GetCapacityAdmissionDurationNs},
	{"provider", (*routingv1.RouteEvidence).GetProviderExecutionDurationNs},
	{"provider_runtime", (*routingv1.RouteEvidence).GetProviderRuntimeDurationNs},
	{"admission_wait", (*routingv1.RouteEvidence).GetProviderAdmissionWaitNs},
	{"overhead", (*routingv1.RouteEvidence).GetOverheadDurationNs},
}

func formatStageProfile(bucket *evidenceProfileBucket) string {
	parts := make([]string, 0, len(evidenceStages))
	for _, stage := range evidenceStages {
		name := stage.name
		values := append([]int64(nil), bucket.stageNS[name]...)
		if len(values) == 0 {
			parts = append(parts, fmt.Sprintf("%s=unavailable(%d/%d)", name, bucket.stageMissing[name], bucket.stageMissing[name]))
			continue
		}
		sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
		parts = append(parts, fmt.Sprintf("%s[p50=%d p95=%d n=%d missing=%d]", name, nearestRank(values, 50), nearestRank(values, 95), len(values), bucket.stageMissing[name]))
	}
	return strings.Join(parts, ",")
}

func nearestRank(sortedValues []int64, percentile int) int64 {
	if len(sortedValues) == 0 {
		return 0
	}
	index := (len(sortedValues)*percentile+99)/100 - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sortedValues) {
		index = len(sortedValues) - 1
	}
	return sortedValues[index]
}

func shortLabel(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, strings.TrimSpace(value))
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > 80 {
		return string(runes[:80])
	}
	return string(runes)
}

func labelOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
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
	return fmt.Sprintf("%s status=%s scenario=%s caller=%s operation=%s role=%s provider=%s model=%s locality=%s fallback=%t latency_ms=%d redacted=%t/%t",
		ev.GetEventId(), ev.GetStatus(), ev.GetScenario(), ev.GetCaller(), ev.GetOperation(), ev.GetRole(), ev.GetSelectedProvider(), ev.GetSelectedModel(), ev.GetSelectedLocality(), ev.GetFallbackUsed(), ev.GetLatencyMs(), ev.GetPromptRedacted(), ev.GetResponseRedacted())
}
