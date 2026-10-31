package measures

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"agent-manager/internal/invocationreadmodel"

	"connectrpc.com/connect"
	measurelib "github.com/vrooli/measures-go"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

// FallbackInsights is the Health view of why runs moved off their first
// choice of runner or model.
func (h *Handler) FallbackInsights(ctx context.Context, req *connect.Request[measurepb.FallbackInsightsRequest]) (*connect.Response[measurepb.FallbackInsightsResponse], error) {
	filter, err := filterWithWindow(nil, req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	insights, err := h.store.FallbackInsights(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	sample := insights.RunnerAttempts + insights.RunnerExhausted + insights.ModelAttempts + insights.ModelExhausted + insights.PolicyCandidateEvents
	return connect.NewResponse(&measurepb.FallbackInsightsResponse{
		RunnerAttempts: insights.RunnerAttempts, RunnerExhausted: insights.RunnerExhausted,
		ModelAttempts: insights.ModelAttempts, ModelExhausted: insights.ModelExhausted,
		RunnerPairs: protoFallbackPairs(insights.RunnerPairs), ModelPairs: protoFallbackPairs(insights.ModelPairs),
		RunnerReasons: protoCountRows(insights.RunnerReasons), ModelReasons: protoCountRows(insights.ModelReasons),
		PolicyCandidateEvents: insights.PolicyCandidateEvents, PolicyOutcomes: protoCountRows(insights.PolicyOutcomes), PolicyFailureClasses: protoCountRows(insights.PolicyFailureClasses),
		Validity:     h.validityForSample(sample),
		Provenance:   h.provenanceWithQuery(ctx, filter, definitionFor(FallbackInsights).SourceTable, sample, "SELECT fallback and policy-candidate event counts FROM run_events"),
		DefinitionId: definitionID(FallbackInsights),
	}), nil
}

func protoFallbackPairs(pairs []invocationreadmodel.FallbackPair) []*measurepb.FallbackPairRow {
	out := make([]*measurepb.FallbackPairRow, 0, len(pairs))
	for _, pair := range pairs {
		out = append(out, &measurepb.FallbackPairRow{From: pair.From, To: pair.To, Reason: pair.Reason, Count: pair.Count})
	}
	return out
}

func protoCountRows(rows []invocationreadmodel.CountRow) []*measurepb.FallbackCountRow {
	out := make([]*measurepb.FallbackCountRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, &measurepb.FallbackCountRow{Key: row.Key, Count: row.Count})
	}
	return out
}

func (h *Handler) registerFallbackInsights(registry *measurelib.Registry) error {
	return registry.Register(measurelib.MeasureDeclaration{
		Name: FallbackInsights, Scenario: "agent-manager", Domain: "health",
		Intent:    "Runner and model fallback attempts, exhausted walks and policy-candidate outcomes.",
		Questions: []string{"how often do runs fall back to another runner or model", "why do model fallbacks happen"},
		Params:    map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}},
		Result:    measurelib.Result{Kind: measurelib.ResultTable, ValueField: "reason", Unit: "events", SummaryTemplate: "fallback insights ({window})"},
		Effect:    measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: "FallbackInsights",
	}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		rangeValue, err := h.resolveWindow(request, measurelib.TokenThisWeek)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		insights, err := h.store.FallbackInsights(ctx, invocationreadmodel.Filter{From: &rangeValue.From, To: &rangeValue.To})
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		count := func(value int64) string { return strconv.FormatInt(value, 10) }
		fields := []map[string]string{{
			"runner_attempts": count(insights.RunnerAttempts), "runner_exhausted": count(insights.RunnerExhausted),
			"model_attempts": count(insights.ModelAttempts), "model_exhausted": count(insights.ModelExhausted),
			"policy_candidate_events": count(insights.PolicyCandidateEvents),
		}}
		return measurelib.MeasureResult{Fields: fields, Provenance: measurelib.Provenance{ExecutedQuery: fmt.Sprintf("SELECT fallback event counts FROM run_events WHERE timestamp >= %q AND timestamp < %q", rangeValue.From.UTC().Format(time.RFC3339Nano), rangeValue.To.UTC().Format(time.RFC3339Nano))}}, nil
	})
}
