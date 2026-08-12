package bindings

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
)

const defaultConditionWindow = 24 * time.Hour

func (r *Registry) Conditions(ctx context.Context, bindingID, scenario string, window time.Duration) (*bindingsv1.GetBindingConditionResponse, error) {
	r = r.active()
	if window <= 0 {
		window = defaultConditionWindow
	}
	if r.recorder == nil {
		return nil, fmt.Errorf("binding invocation ledger is unavailable")
	}
	rows, err := r.recorder.ListInvocations(ctx, time.Now().UTC().Add(-window), bindingID, scenario)
	if err != nil {
		return nil, err
	}
	byBinding := make(map[string][]Invocation)
	for _, row := range rows {
		byBinding[row.BindingID] = append(byBinding[row.BindingID], row)
	}
	response := &bindingsv1.GetBindingConditionResponse{WindowSeconds: int64(window / time.Second)}
	for _, binding := range r.bindings {
		if bindingID != "" && binding.GetId() != bindingID || scenario != "" && binding.GetScenario() != scenario {
			continue
		}
		response.TotalBindings++
		condition := conditionFor(binding, byBinding[binding.GetId()], r.artifactMtime)
		if len(byBinding[binding.GetId()]) > 0 {
			response.InstrumentedBindings++
		}
		response.Conditions = append(response.Conditions, condition)
	}
	return response, nil
}

func (r *Registry) InvocationMeasures(ctx context.Context) (total, failureRatePercent, dormant int) {
	r = r.active()
	if r.recorder == nil {
		return 0, 0, len(r.bindings)
	}
	rows, err := r.recorder.ListInvocations(ctx, time.Now().UTC().Add(-defaultConditionWindow), "", "")
	if err != nil {
		return 0, 0, len(r.bindings)
	}
	seen := make(map[string]struct{})
	failed := 0
	for _, row := range rows {
		total++
		seen[row.BindingID] = struct{}{}
		if row.Outcome == "failed" {
			failed++
		}
	}
	if total > 0 {
		failureRatePercent = int(float64(failed) / float64(total) * 100)
	}
	dormant = len(r.bindings) - len(seen)
	if dormant < 0 {
		dormant = 0
	}
	return total, failureRatePercent, dormant
}

func conditionFor(binding interface {
	GetId() string
	GetScenario() string
}, rows []Invocation, artifactMtime time.Time,
) *bindingsv1.BindingCondition {
	condition := &bindingsv1.BindingCondition{BindingId: binding.GetId(), Scenario: binding.GetScenario()}
	serving := &bindingsv1.ServingCondition{Family: &bindingsv1.ConditionFamily{Status: bindingsv1.ConditionStatus_CONDITION_STATUS_HEALTHY}}
	exercise := &bindingsv1.ExerciseCondition{Family: &bindingsv1.ConditionFamily{Status: bindingsv1.ConditionStatus_CONDITION_STATUS_DORMANT, Reason: "exercise.invocations=0"}}
	callers := map[string]struct{}{}
	latencies := make([]int64, 0, len(rows))
	failed, refused := 0, 0
	var latest time.Time
	for _, row := range rows {
		exercise.Invocations++
		caller := row.SessionID
		if caller == "" {
			caller = row.ProgramID
		}
		if caller != "" {
			callers[caller] = struct{}{}
		}
		latencies = append(latencies, row.LatencyMS)
		if row.Outcome == "failed" {
			failed++
		}
		if row.Outcome == "refused" {
			refused++
		}
		if row.OccurredAt.After(latest) {
			latest = row.OccurredAt
		}
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	if len(rows) > 0 {
		exercise.Family.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_HEALTHY
		exercise.Family.Reason = "exercise.invocations>0"
		serving.FailureRate = float64(failed) / float64(len(rows))
		serving.DegradationRate = float64(refused) / float64(len(rows))
		serving.LatencyP50Ms = nearestRank(latencies, 0.50)
		serving.LatencyP95Ms = nearestRank(latencies, 0.95)
		if failed > 0 {
			serving.Family.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_DEGRADED
			serving.Family.Reason = fmt.Sprintf("serving.failure_rate=%.4f", serving.FailureRate)
		} else if refused > 0 {
			serving.Family.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_DEGRADED
			serving.Family.Reason = fmt.Sprintf("serving.degradation_rate=%.4f", serving.DegradationRate)
		}
	}
	exercise.DistinctCallers = int64(len(callers))
	if !latest.IsZero() {
		exercise.LastInvokedAt = latest.UTC().Format(time.RFC3339Nano)
	}
	freshness := &bindingsv1.FreshnessCondition{Family: &bindingsv1.ConditionFamily{Status: bindingsv1.ConditionStatus_CONDITION_STATUS_UNINSTRUMENTED, Reason: "freshness.drift is not instrumented"}, DriftStatus: bindingsv1.ConditionStatus_CONDITION_STATUS_UNINSTRUMENTED, DriftReason: "declaration-versus-reality drift is not instrumented"}
	if !artifactMtime.IsZero() {
		age := time.Since(artifactMtime)
		if age < 0 {
			age = 0
		}
		freshness.AgeSeconds = int64(age / time.Second)
	}
	condition.Serving = serving
	condition.Freshness = freshness
	condition.Exercise = exercise
	switch {
	case len(rows) == 0:
		condition.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_DORMANT
		condition.Verdict = "DORMANT: exercise.invocations=0"
	case serving.Family.Status == bindingsv1.ConditionStatus_CONDITION_STATUS_DEGRADED:
		condition.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_DEGRADED
		condition.Verdict = serving.Family.Reason
	default:
		condition.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_HEALTHY
		condition.Verdict = "HEALTHY"
	}
	return condition
}

func nearestRank(values []int64, percentile float64) int64 {
	if len(values) == 0 {
		return 0
	}
	rank := int(math.Ceil(percentile * float64(len(values))))
	if rank < 1 {
		rank = 1
	}
	return values[rank-1]
}
