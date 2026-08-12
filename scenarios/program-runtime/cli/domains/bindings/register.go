package bindings

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	bindingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings/bindings_v1connect"
)

const GroupName = "bindings"

type handlers struct {
	client          bindingsconnect.BindingRegistryServiceClient
	conditionClient bindingsconnect.BindingConditionServiceClient
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: bindingsconnect.NewBindingRegistryServiceClient(httpClient, baseURL), conditionClient: bindingsconnect.NewBindingConditionServiceClient(httpClient, baseURL)}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"BindingRegistryService.ListBindings":                                            cliapp.ProtoList(h.list, h.listReport),
		"BindingRegistryService.ListUnbound":                                             cliapp.ProtoList(h.unbound, h.unboundReport),
		"BindingRegistryService.SweepBindings":                                           cliapp.ProtoList(h.sweep, h.sweepReport),
		"BindingRegistryService.ResolveActCells":                                         cliapp.ProtoList(h.act, h.actReport),
		"BindingRegistryService.DoctorBindings":                                          cliapp.ProtoListEmitUnpopulatedJSON(h.doctor, h.doctorReport),
		"BindingRegistryService.DescribeBinding":                                         cliapp.ProtoList(h.describe, h.describeReport),
		"BindingRegistryService.ResolveIntent":                                           cliapp.ProtoList(h.resolveIntent, h.resolveIntentReport),
		"vrooli.program_runtime.v1.bindings.BindingConditionService.GetBindingCondition": cliapp.ProtoList(h.condition, h.conditionReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("bindings: load from manifest: %w", err)
	}
	return group, nil
}

func (h *handlers) resolveIntent(ctx cliapp.OperationContext) (*bindingsv1.ResolveIntentResponse, error) {
	var limit int32
	if raw := ctx.Flag("limit"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("limit must be a signed 32-bit integer: %w", err)
		}
		limit = int32(parsed)
	}
	r, err := h.client.ResolveIntent(context.Background(), connect.NewRequest(&bindingsv1.ResolveIntentRequest{Intent: ctx.Flag("intent"), Limit: limit}))
	if err != nil {
		return nil, cliapp.WrapAPIError("resolve binding intent", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) resolveIntentReport(_ cliapp.OperationContext, r *bindingsv1.ResolveIntentResponse) cliapp.ListReport {
	items := make([]string, 0, len(r.GetBindings()))
	for _, b := range r.GetBindings() {
		items = append(items, fmt.Sprintf("%s/%s/%s — %s", b.GetScenario(), b.GetGroup(), b.GetCommand(), b.GetEffect()))
	}
	mode := "semantic"
	if r.GetFallback() {
		mode = "local fallback"
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d governed candidate(s); %s; %s", len(items), mode, r.GetReason())}, ResultsHeading: "Candidates", Results: items}
}

func (h *handlers) condition(ctx cliapp.OperationContext) (*bindingsv1.GetBindingConditionResponse, error) {
	window, _ := strconv.ParseInt(ctx.Flag("window-seconds"), 10, 64)
	r, err := h.conditionClient.GetBindingCondition(context.Background(), connect.NewRequest(&bindingsv1.GetBindingConditionRequest{BindingId: ctx.Flag("binding-id"), Scenario: ctx.Flag("scenario"), WindowSeconds: window}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get binding condition", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) conditionReport(_ cliapp.OperationContext, r *bindingsv1.GetBindingConditionResponse) cliapp.ListReport {
	items := make([]string, 0, len(r.GetConditions()))
	verdicts := map[string]int{}
	for _, c := range r.GetConditions() {
		status := c.GetStatus().String()
		verdicts[status]++
		sustained := ""
		if c.GetSustainedDegradation() {
			sustained = "; sustained=" + c.GetSustainedDegradationReason()
		}
		items = append(items, fmt.Sprintf("%s — %s (%s; p50=%dms p95=%dms invocations=%d%s)", c.GetBindingId(), status, c.GetVerdict(), c.GetServing().GetLatencyP50Ms(), c.GetServing().GetLatencyP95Ms(), c.GetExercise().GetInvocations(), sustained))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Binding conditions: verdicts healthy=%d degraded=%d dormant=%d uninstrumented=%d; instrumented=%d/%d; window: %ds.", verdicts["CONDITION_STATUS_HEALTHY"], verdicts["CONDITION_STATUS_DEGRADED"], verdicts["CONDITION_STATUS_DORMANT"], verdicts["CONDITION_STATUS_UNINSTRUMENTED"], r.GetInstrumentedBindings(), r.GetTotalBindings(), r.GetWindowSeconds())}, ResultsHeading: "Conditions", Results: items}
}

func (h *handlers) list(ctx cliapp.OperationContext) (*bindingsv1.ListBindingsResponse, error) {
	reachableOnly := ctx.BoolFlag("reachable-only")
	r, err := h.client.ListBindings(context.Background(), connect.NewRequest(&bindingsv1.ListBindingsRequest{Scenario: ctx.Flag("scenario"), Group: ctx.Flag("group"), ReachableOnly: reachableOnly}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list bindings", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, r *bindingsv1.ListBindingsResponse) cliapp.ListReport {
	items := make([]string, 0, len(r.GetBindings()))
	for _, b := range r.GetBindings() {
		items = append(items, fmt.Sprintf("%s — %s.%s [%s; reachable=%t]", b.GetId(), b.GetService(), b.GetMethod(), b.GetEffect(), b.GetReachable()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d governed binding(s); reachability checked at %s.", len(items), r.GetReachabilityCheckedAt())}, ResultsHeading: "Bindings", Results: items}
}

func (h *handlers) unbound(ctx cliapp.OperationContext) (*bindingsv1.ListUnboundResponse, error) {
	r, err := h.client.ListUnbound(context.Background(), connect.NewRequest(&bindingsv1.ListUnboundRequest{Scenario: ctx.Flag("scenario")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list unbound capabilities", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) unboundReport(_ cliapp.OperationContext, r *bindingsv1.ListUnboundResponse) cliapp.ListReport {
	items := make([]string, 0, len(r.GetCapabilities()))
	for _, c := range r.GetCapabilities() {
		items = append(items, fmt.Sprintf("%s/%s — %s (%s)", c.GetScenario(), c.GetCommand(), c.GetReason().String(), c.GetDetail()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d unbound capability record(s).", len(items))}, ResultsHeading: "Unbound", Results: items}
}

func (h *handlers) sweep(ctx cliapp.OperationContext) (*bindingsv1.SweepBindingsResponse, error) {
	dryRun := !ctx.BoolFlag("execute")
	if ctx.BoolFlag("dry-run") {
		dryRun = true
	}
	r, err := h.client.SweepBindings(context.Background(), connect.NewRequest(&bindingsv1.SweepBindingsRequest{Scenario: ctx.Flag("scenario"), Effect: ctx.Flag("effect"), DryRun: dryRun}))
	if err != nil {
		return nil, cliapp.WrapAPIError("sweep bindings", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) sweepReport(_ cliapp.OperationContext, r *bindingsv1.SweepBindingsResponse) cliapp.ListReport {
	items := make([]string, 0, len(r.GetResults()))
	for _, result := range r.GetResults() {
		if result.GetSkippedReason() != "" {
			items = append(items, fmt.Sprintf("%s — skipped: %s", result.GetBindingId(), result.GetSkippedReason()))
		} else if result.GetAttempted() {
			items = append(items, fmt.Sprintf("%s — %s (%dms)", result.GetBindingId(), result.GetOutcome(), result.GetLatencyMs()))
		}
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Sweep: eligible=%d attempted=%d succeeded=%d failed=%d refused=%d skipped=%d provenance=%s.", r.GetEligible(), r.GetAttempted(), r.GetSucceeded(), r.GetFailed(), r.GetRefused(), r.GetSkipped(), r.GetProvenance())}, ResultsHeading: "Sweep", Results: items}
}

func (h *handlers) act(_ cliapp.OperationContext) (*bindingsv1.ResolveActCellsResponse, error) {
	r, err := h.client.ResolveActCells(context.Background(), connect.NewRequest(&bindingsv1.ResolveActCellsRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("resolve Act cells", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) actReport(_ cliapp.OperationContext, r *bindingsv1.ResolveActCellsResponse) cliapp.ListReport {
	items := make([]string, 0, len(r.GetCells()))
	now, inReach, missing := 0, 0, 0
	for _, c := range r.GetCells() {
		items = append(items, fmt.Sprintf("%s — %s", c.GetId(), c.GetVerdict().String()))
		switch c.GetVerdict() {
		case bindingsv1.ActVerdict_ACT_VERDICT_NOW:
			now++
		case bindingsv1.ActVerdict_ACT_VERDICT_IN_REACH:
			inReach++
		case bindingsv1.ActVerdict_ACT_VERDICT_AUTHORED:
			if strings.EqualFold(c.GetAuthoredStatus(), "missing") {
				missing++
			} else {
				inReach++
			}
		}
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Act cells: %d; NOW=%d, IN-REACH=%d, MISSING=%d (confidence=%s).", len(items), now, inReach, missing, r.GetDenominatorConfidence())}, ResultsHeading: "Act", Results: items}
}

func (h *handlers) doctor(ctx cliapp.OperationContext) (*bindingsv1.DoctorBindingsResponse, error) {
	r, err := h.client.DoctorBindings(context.Background(), connect.NewRequest(&bindingsv1.DoctorBindingsRequest{Scenario: ctx.Flag("scenario")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("doctor bindings", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) doctorReport(_ cliapp.OperationContext, r *bindingsv1.DoctorBindingsResponse) cliapp.ListReport {
	items := make([]string, 0, len(r.GetIssues()))
	for _, issue := range r.GetIssues() {
		items = append(items, fmt.Sprintf("%s %s: %s (%s)", issue.GetBindingId(), issue.GetArgument(), issue.GetReason(), issue.GetRequestType()))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Bindings: %d; callable: %d; uncallable: %d; partial: %d; zero-arg: %d; misroutes: %d; semantic collisions: %d; control binds: %d; required fields: %d; redundant binds: %d; manifests: %d/%d; reachable: %d; unreachable: %d.", r.GetBindings(), r.GetCallable(), r.GetUncallable(), r.GetPartial(), r.GetZeroArg(), r.GetMisroutes(), r.GetFieldCollisions(), r.GetControlFlagsBound(), r.GetRequiredFieldsUnpopulated(), r.GetBindsWhereRenameSuffices(), r.GetManifestScenarios(), r.GetTotalScenarios(), len(r.GetReachableScenarios()), len(r.GetUnreachableScenarios()))},
		ResultsHeading: "Unresolved arguments", Results: items,
	}
}

func (h *handlers) describe(ctx cliapp.OperationContext) (*bindingsv1.DescribeBindingResponse, error) {
	id := ctx.Positional("id")
	r, err := h.client.DescribeBinding(context.Background(), connect.NewRequest(&bindingsv1.DescribeBindingRequest{Id: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError("describe binding", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) describeReport(_ cliapp.OperationContext, r *bindingsv1.DescribeBindingResponse) cliapp.ListReport {
	items := make([]string, 0, len(r.GetArguments()))
	for _, argument := range r.GetArguments() {
		path := argument.GetProtoPath()
		if path == "" {
			path = argument.GetReason()
		}
		items = append(items, fmt.Sprintf("%s -> %s [%s]", argument.GetName(), path, argument.GetKind()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%s (callable=%t, source=%s)", r.GetBinding().GetId(), r.GetCallable(), r.GetResolvedSource())}, ResultsHeading: "Arguments", Results: items}
}
