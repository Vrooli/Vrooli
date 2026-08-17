package bindings

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"time"

	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
)

// Load-bearing condition constants. The exercise window is one day because
// the Phase 8 ledger calibration observed the operational sweep and normal
// reads at a daily-or-faster cadence (the slowest legitimate caller must be
// re-measured if it is ever reported dormant). Seven days exceeds ordinary
// restart and rollout windows, so a sustained flag means degradation survived
// more than one normal recovery opportunity.
const (
	conditionSustainedWindow = 7 * 24 * time.Hour
	conditionExerciseWindow  = 24 * time.Hour
)

type conditionBand struct {
	failureRate     float64
	degradationRate float64
}

type freshnessMetadata struct {
	sourcePath      string
	sourceMtime     time.Time
	generationMtime time.Time
}

// Calibrated 2026-08-12 from 639 instrumented bindings after the operator
// sweep: failure-rate p50=0.50, p95=1.00, max=1.00; refusal/degradation-rate
// p95=0.00, max=1.00. A leg is degraded only when its observed bad-outcome
// majority exceeds 0.50; a refusal majority uses the same conservative band.
// Latency p95 was 13,266ms at the distribution p95 and is reported as a
// measure, not used as a health verdict until a latency SLO is authored.
var conditionBands = conditionBand{failureRate: 0.50, degradationRate: 0.50}

const defaultConditionWindow = conditionExerciseWindow

func (r *Registry) Conditions(ctx context.Context, bindingID, scenario string, window time.Duration) (*bindingsv1.GetBindingConditionResponse, error) {
	r = r.active()
	if window <= 0 {
		window = defaultConditionWindow
	}
	if r.recorder == nil {
		return nil, fmt.Errorf("binding invocation ledger is unavailable")
	}
	now := time.Now().UTC()
	rows, err := r.recorder.ListInvocations(ctx, now.Add(-window), bindingID, scenario)
	if err != nil {
		return nil, err
	}
	sustainedRows, err := r.recorder.ListInvocations(ctx, now.Add(-conditionSustainedWindow), bindingID, scenario)
	if err != nil {
		return nil, err
	}
	byBinding := make(map[string][]Invocation)
	for _, row := range rows {
		byBinding[row.BindingID] = append(byBinding[row.BindingID], row)
	}
	sustainedByBinding := make(map[string][]Invocation)
	for _, row := range sustainedRows {
		sustainedByBinding[row.BindingID] = append(sustainedByBinding[row.BindingID], row)
	}
	response := &bindingsv1.GetBindingConditionResponse{WindowSeconds: int64(window / time.Second)}
	byScenario := make(map[string][]*bindingsv1.BindingCondition)
	for _, binding := range r.bindings {
		if bindingID != "" && binding.GetId() != bindingID || scenario != "" && binding.GetScenario() != scenario {
			continue
		}
		response.TotalBindings++
		metadata := freshnessMetadata{generationMtime: r.generationMtime}
		metadata.sourcePath = r.manifestPaths[binding.GetScenario()]
		if metadata.sourcePath == "" {
			metadata.sourcePath = filepath.Join("scenarios", binding.GetScenario(), "cli", "manifest.json")
		}
		metadata.sourceMtime = r.manifestMtimes[binding.GetScenario()]
		condition := conditionForWithFreshness(binding, byBinding[binding.GetId()], sustainedByBinding[binding.GetId()], metadata)
		if len(byBinding[binding.GetId()]) > 0 {
			response.InstrumentedBindings++
		}
		response.Conditions = append(response.Conditions, condition)
		byScenario[binding.GetScenario()] = append(byScenario[binding.GetScenario()], condition)
	}
	for _, scenario := range sortedScenarioNames(byScenario) {
		conditions := byScenario[scenario]
		rollup := &bindingsv1.ScenarioCondition{Scenario: scenario, BindingCount: int32(len(conditions))}
		for _, condition := range conditions {
			switch condition.GetStatus() {
			case bindingsv1.ConditionStatus_CONDITION_STATUS_DEGRADED:
				rollup.DegradedBindings++
			case bindingsv1.ConditionStatus_CONDITION_STATUS_HEALTHY:
				rollup.HealthyBindings++
			case bindingsv1.ConditionStatus_CONDITION_STATUS_DORMANT, bindingsv1.ConditionStatus_CONDITION_STATUS_UNINSTRUMENTED:
				rollup.DormantBindings++
			}
		}
		switch {
		case rollup.DegradedBindings > 0:
			rollup.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_DEGRADED
			rollup.Verdict = fmt.Sprintf("scenario has %d degraded binding(s)", rollup.DegradedBindings)
		case rollup.HealthyBindings > 0:
			rollup.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_HEALTHY
			rollup.Verdict = fmt.Sprintf("scenario has %d healthy binding(s)", rollup.HealthyBindings)
		default:
			rollup.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_DORMANT
			rollup.Verdict = "scenario has no exercised bindings in the requested window"
		}
		response.ScenarioConditions = append(response.ScenarioConditions, rollup)
	}
	return response, nil
}

func sortedScenarioNames(groups map[string][]*bindingsv1.BindingCondition) []string {
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SustainedDegradedCount is the measure-facing projection of the same
// condition logic returned by GetBindingCondition. Keeping it on Registry
// prevents the measure from inventing a second health calculation.
func (r *Registry) SustainedDegradedCount(ctx context.Context) int {
	response, err := r.Conditions(ctx, "", "", defaultConditionWindow)
	if err != nil {
		return 0
	}
	count := 0
	for _, condition := range response.GetConditions() {
		if condition.GetSustainedDegradation() {
			count++
		}
	}
	return count
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
		if row.InvocationClass == "probe_invalid_argument" || row.InvocationClass == "probe_timeout" {
			continue
		}
		total++
		seen[row.BindingID] = struct{}{}
		if row.InvocationClass == "target_failed" || row.InvocationClass == "target_unavailable" {
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
	return conditionForWithSustained(binding, rows, rows, artifactMtime)
}

func conditionForWithFreshness(binding interface {
	GetId() string
	GetScenario() string
}, rows, sustainedRows []Invocation, metadata freshnessMetadata,
) *bindingsv1.BindingCondition {
	condition := conditionForWithSustained(binding, rows, sustainedRows, metadata.generationMtime)
	freshness := condition.Freshness
	freshness.SourcePath = metadata.sourcePath
	freshness.SourceMtime = formatFreshnessTime(metadata.sourceMtime)
	freshness.GenerationMtime = formatFreshnessTime(metadata.generationMtime)
	switch {
	case metadata.generationMtime.IsZero():
		freshness.Family.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_UNINSTRUMENTED
		freshness.Family.Reason = "freshness generation timestamp unavailable"
		freshness.DriftStatus = freshness.Family.Status
		freshness.DriftReason = freshness.Family.Reason
	case metadata.sourceMtime.IsZero():
		freshness.Family.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_UNINSTRUMENTED
		freshness.Family.Reason = fmt.Sprintf("freshness source timestamp unavailable: %s", metadata.sourcePath)
		freshness.DriftStatus = freshness.Family.Status
		freshness.DriftReason = freshness.Family.Reason
	case metadata.sourceMtime.After(metadata.generationMtime):
		freshness.Family.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_DEGRADED
		freshness.Family.Reason = fmt.Sprintf("freshness source newer than generation: %s source=%s generation=%s", metadata.sourcePath, formatFreshnessTime(metadata.sourceMtime), formatFreshnessTime(metadata.generationMtime))
		freshness.DriftStatus = freshness.Family.Status
		freshness.DriftReason = freshness.Family.Reason
	default:
		freshness.Family.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_HEALTHY
		freshness.Family.Reason = fmt.Sprintf("freshness generation covers %s", metadata.sourcePath)
		freshness.DriftStatus = freshness.Family.Status
		freshness.DriftReason = freshness.Family.Reason
	}
	return condition
}

func formatFreshnessTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func conditionForWithSustained(binding interface {
	GetId() string
	GetScenario() string
}, rows, sustainedRows []Invocation, artifactMtime time.Time,
) *bindingsv1.BindingCondition {
	condition := &bindingsv1.BindingCondition{BindingId: binding.GetId(), Scenario: binding.GetScenario()}
	serving := &bindingsv1.ServingCondition{Family: &bindingsv1.ConditionFamily{Status: bindingsv1.ConditionStatus_CONDITION_STATUS_UNINSTRUMENTED, Reason: "serving has no invocations in window"}}
	exercise := &bindingsv1.ExerciseCondition{Family: &bindingsv1.ConditionFamily{Status: bindingsv1.ConditionStatus_CONDITION_STATUS_DORMANT, Reason: "exercise.invocations=0"}}
	callers := map[string]struct{}{}
	latencies := make([]int64, 0, len(rows))
	servingRows := make([]Invocation, 0, len(rows))
	failed, refused := 0, 0
	var latest time.Time
	for _, row := range rows {
		if row.InvocationClass == "" {
			row.InvocationClass = classifyInvocation(row)
		}
		if row.InvocationClass == "probe_invalid_argument" || row.InvocationClass == "probe_timeout" {
			serving.ProbeInvocations++
			continue
		}
		servingRows = append(servingRows, row)
		if row.Origin == "synthetic" {
			serving.SyntheticInvocations++
		} else {
			serving.OrganicInvocations++
		}
		if row.Origin == "organic" {
			exercise.Invocations++
			caller := row.SessionID
			if caller == "" {
				caller = row.ProgramID
			}
			if caller != "" {
				callers[caller] = struct{}{}
			}
			if row.OccurredAt.After(latest) {
				latest = row.OccurredAt
			}
		} else {
			exercise.SyntheticInvocations++
		}
		latencies = append(latencies, row.LatencyMS)
		if row.Outcome == "failed" {
			failed++
		}
		if row.Outcome == "refused" {
			refused++
		}
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	if len(servingRows) > 0 {
		serving.Family.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_HEALTHY
		serving.Family.Reason = "serving observations measured"
		serving.FailureRate = float64(failed) / float64(len(servingRows))
		serving.DegradationRate = float64(refused) / float64(len(servingRows))
		serving.LatencyP50Ms = nearestRank(latencies, 0.50)
		serving.LatencyP95Ms = nearestRank(latencies, 0.95)
		if serving.FailureRate > conditionBands.failureRate {
			serving.Family.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_DEGRADED
			serving.Family.Reason = fmt.Sprintf("serving.failure_rate=%.4f", serving.FailureRate)
		} else if serving.DegradationRate > conditionBands.degradationRate {
			serving.Family.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_DEGRADED
			serving.Family.Reason = fmt.Sprintf("serving.degradation_rate=%.4f", serving.DegradationRate)
		}
	}
	if exercise.Invocations > 0 {
		exercise.Family.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_HEALTHY
		exercise.Family.Reason = "exercise.invocations>0"
	}
	if serving.Family.Status == bindingsv1.ConditionStatus_CONDITION_STATUS_DEGRADED && sustainedDegradation(sustainedRows) {
		condition.SustainedDegradation = true
		condition.SustainedDegradationReason = fmt.Sprintf("serving degraded across the %s sustained window", conditionSustainedWindow)
	}
	exercise.DistinctCallers = int64(len(callers))
	if !latest.IsZero() {
		exercise.LastInvokedAt = latest.UTC().Format(time.RFC3339Nano)
	}
	freshness := &bindingsv1.FreshnessCondition{Family: &bindingsv1.ConditionFamily{Status: bindingsv1.ConditionStatus_CONDITION_STATUS_UNINSTRUMENTED, Reason: "freshness source metadata unavailable"}, DriftStatus: bindingsv1.ConditionStatus_CONDITION_STATUS_UNINSTRUMENTED, DriftReason: "freshness source metadata unavailable"}
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
	case serving.Family.Status == bindingsv1.ConditionStatus_CONDITION_STATUS_UNINSTRUMENTED:
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

// sustainedDegradation treats the invocation ledger as an observation stream:
// all observations in the sustained window must be bad outcomes and the
// stream must span the full window. A later success clears the promotion flag
// because the condition was not continuously degraded.
func sustainedDegradation(rows []Invocation) bool {
	if len(rows) == 0 {
		return false
	}
	first, last := rows[0].OccurredAt, rows[0].OccurredAt
	for _, row := range rows {
		if row.OccurredAt.Before(first) {
			first = row.OccurredAt
		}
		if row.OccurredAt.After(last) {
			last = row.OccurredAt
		}
		if row.Outcome != "failed" && row.Outcome != "refused" {
			return false
		}
	}
	return !first.IsZero() && last.Sub(first) >= conditionSustainedWindow
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
