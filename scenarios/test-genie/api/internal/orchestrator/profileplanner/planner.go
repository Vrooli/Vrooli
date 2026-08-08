package profileplanner

import (
	"math"
	"sort"
	"strings"
	"time"

	"test-genie/internal/orchestrator/phasepolicy"
)

type EstimateSource string

const (
	EstimateSourceScenarioHistory EstimateSource = "scenario_history"
	EstimateSourceBlendedHistory  EstimateSource = "blended_history"
	EstimateSourceGlobalHistory   EstimateSource = "global_history"
	EstimateSourceUnknown         EstimateSource = "unknown"
)

type EstimateConfidence string

const (
	EstimateConfidenceHigh   EstimateConfidence = "high"
	EstimateConfidenceMedium EstimateConfidence = "medium"
	EstimateConfidenceLow    EstimateConfidence = "low"
)

type ProfileStrategy string

const (
	StrategyBudgetFastFeedback ProfileStrategy = "budget_fast_feedback"
	StrategyBudgetSmoke        ProfileStrategy = "budget_smoke"
)

const (
	FitModeAdditive = "additive"
	FitModeMakespan = "makespan"

	ReasonSelectedRequired         = "selected_required"
	ReasonSelectedBudgetFit        = "selected_budget_fit"
	ReasonSelectedUnknownCost      = "selected_unknown_cost"
	ReasonBudgetExceededByRequired = "budget_exceeded_by_required"
	ReasonOmittedBudget            = "omitted_budget_exceeded"
	ReasonOmittedUnknown           = "omitted_unknown_estimate"
	ReasonOmittedExplicitOnly      = "omitted_explicit_only"
	ReasonOmittedNeverDefault      = "omitted_never_by_default"

	historyPercentile        = 0.90
	primaryScenarioSamples   = 5
	maxScenarioSamplesPerKey = 20
	maxGlobalSamplesPerKey   = 50
)

type Profile struct {
	Name          string
	BudgetSeconds int
	Strategy      ProfileStrategy
	// ConcurrencyGranted is true only when the scheduler has granted capacity
	// for phases to overlap. Plan previews leave this false and therefore fit
	// against additive wall-clock duration.
	ConcurrencyGranted bool
}

type Candidate struct {
	Name             string
	DisplayName      string
	TimeoutSeconds   int
	Policy           phasepolicy.Policy
	Order            int
	ConcurrencyMode  string
	ConcurrencyGroup string
}

type Sample struct {
	ScenarioName string
	PhaseName    string
	Status       string
	// DurationMilliseconds is the planner's explicit input unit. The execution
	// repository performs the single storage-to-planner conversion.
	DurationMilliseconds int64
	// DurationSeconds supports fixture callers and legacy run projections. When
	// both fields are present, milliseconds are authoritative.
	DurationSeconds int
	CompletedAt     time.Time
}

type Estimate struct {
	DurationSeconds     int
	Source              EstimateSource
	Confidence          EstimateConfidence
	SampleSize          int
	Unknown             bool
	PointSampleCount    int
	CensoredSampleCount int
	ExcludedSampleCount int
}

type Decision struct {
	Candidate Candidate
	Estimate  Estimate
	Selected  bool
	Reasons   []string
}

type Plan struct {
	Profile                       Profile
	Selected                      []Decision
	Omitted                       []Decision
	EstimatedTotalSeconds         int
	UnknownEstimateCount          int
	SelectedUnknownEstimates      int
	RequiredEstimatedTotalSeconds int
	BudgetOverflowSeconds         int
	BudgetExceededByRequired      bool
	FitMode                       string
}

type sampleBucket struct {
	pointDurations []int
	censoredCount  int
	excludedCount  int
}

func (b sampleBucket) totalCount() int {
	return len(b.pointDurations) + b.censoredCount + b.excludedCount
}

type Estimator struct {
	scenarioName    string
	scenarioBuckets map[string]sampleBucket
	globalBuckets   map[string]sampleBucket
}

// RunSample is a terminal full-run duration. It stays separate from phase
// history because startup/orchestration costs cannot be recovered by summing
// independent phase observations.
type RunSample struct {
	DurationSeconds      int
	DurationMilliseconds int64
	TerminalOutcome      string
	CompletedAt          time.Time
}

func NewEstimator(scenarioName string, samples []Sample) Estimator {
	scenarioBuckets := make(map[string]sampleBucket)
	globalBuckets := make(map[string]sampleBucket)
	for _, sample := range samples {
		key := normalize(sample.PhaseName)
		if key == "" {
			continue
		}
		class := classifySample(sample)
		isScenario := strings.EqualFold(strings.TrimSpace(sample.ScenarioName), strings.TrimSpace(scenarioName))
		if isScenario {
			bucket := scenarioBuckets[key]
			if bucket.totalCount() < maxScenarioSamplesPerKey {
				addSample(&bucket, sample, class)
				scenarioBuckets[key] = bucket
			}
			continue
		}
		bucket := globalBuckets[key]
		if bucket.totalCount() < maxGlobalSamplesPerKey {
			addSample(&bucket, sample, class)
			globalBuckets[key] = bucket
		}
	}
	return Estimator{scenarioName: scenarioName, scenarioBuckets: scenarioBuckets, globalBuckets: globalBuckets}
}

func addSample(bucket *sampleBucket, sample Sample, class sampleClass) {
	duration := sampleDurationSeconds(sample)
	switch class {
	case samplePoint:
		bucket.pointDurations = append(bucket.pointDurations, duration)
	case sampleCensored:
		bucket.censoredCount++
	default:
		bucket.excludedCount++
	}
}

// EstimateComparableRun returns a conservative P90 from exact full-run
// history. Failed and timed-out runs remain elapsed evidence at run level;
// phase-level censoring is handled separately by NewEstimator.
func EstimateComparableRun(samples []RunSample) Estimate {
	durations := make([]int, 0, len(samples))
	for _, sample := range samples {
		duration := sample.DurationSeconds
		if sample.DurationMilliseconds > 0 {
			duration = maxInt(0, int(math.Ceil(float64(sample.DurationMilliseconds)/1000)))
		}
		if duration > 0 {
			durations = append(durations, duration)
		}
	}
	if len(durations) == 0 {
		return Estimate{Source: EstimateSourceUnknown, Confidence: EstimateConfidenceLow, Unknown: true}
	}
	return Estimate{
		DurationSeconds:  percentileSeconds(durations, historyPercentile),
		Source:           EstimateSourceScenarioHistory,
		Confidence:       confidenceFor(EstimateSourceScenarioHistory, len(durations), 0, 0),
		SampleSize:       len(durations),
		PointSampleCount: len(durations),
	}
}

func (e Estimator) Estimate(phaseName string, timeoutSeconds int) Estimate {
	key := normalize(phaseName)
	return estimatePhase(timeoutSeconds, e.scenarioBuckets[key], e.globalBuckets[key])
}

func PlanProfile(profile Profile, candidates []Candidate, estimator Estimator) Plan {
	plan := Plan{Profile: profile, FitMode: FitModeAdditive}
	if profile.ConcurrencyGranted {
		plan.FitMode = FitModeMakespan
	}

	ordered := append([]Candidate(nil), candidates...)
	sort.SliceStable(ordered, func(i, j int) bool { return candidateRank(ordered[i]) < candidateRank(ordered[j]) })
	decisions := make([]Decision, 0, len(ordered))
	for _, candidate := range ordered {
		estimate := estimator.Estimate(candidate.Name, candidate.TimeoutSeconds)
		if estimate.Unknown {
			plan.UnknownEstimateCount++
		}
		decision := Decision{Candidate: candidate, Estimate: estimate}
		if omitReason := defaultOmitReason(candidate.Policy); omitReason != "" {
			decision.Reasons = []string{omitReason}
		}
		decisions = append(decisions, decision)
	}

	// Required phases retain policy/rank order. Optional phases are a
	// cost-ascending greedy fill, using the prior rank only as a tie-breaker.
	sort.SliceStable(decisions, func(i, j int) bool {
		ri := isRequired(decisions[i].Candidate.Policy)
		rj := isRequired(decisions[j].Candidate.Policy)
		if ri != rj {
			return ri
		}
		if ri {
			return candidateRank(decisions[i].Candidate) < candidateRank(decisions[j].Candidate)
		}
		if decisions[i].Estimate.DurationSeconds != decisions[j].Estimate.DurationSeconds {
			return decisions[i].Estimate.DurationSeconds < decisions[j].Estimate.DurationSeconds
		}
		return candidateRank(decisions[i].Candidate) < candidateRank(decisions[j].Candidate)
	})

	for _, decision := range decisions {
		if len(decision.Reasons) > 0 {
			plan.Omitted = append(plan.Omitted, decision)
			continue
		}
		if isRequired(decision.Candidate.Policy) {
			decision.Selected = true
			decision.Reasons = []string{ReasonSelectedRequired}
			plan.Selected = append(plan.Selected, decision)
			plan.RequiredEstimatedTotalSeconds += decision.Estimate.DurationSeconds
			if decision.Estimate.Unknown {
				plan.SelectedUnknownEstimates++
			}
			continue
		}
		if decision.Estimate.Unknown {
			// A missing history sample earns one bounded observation. This is
			// deliberately selected even when its cost is not yet known.
			decision.Selected = true
			decision.Reasons = []string{ReasonSelectedUnknownCost}
			plan.Selected = append(plan.Selected, decision)
			plan.SelectedUnknownEstimates++
			continue
		}

		candidateTotal := estimatedPlanTotal(profile, append(append([]Decision(nil), plan.Selected...), decision))
		if profile.BudgetSeconds > 0 && candidateTotal > profile.BudgetSeconds {
			// A required overflow cannot be repaired by silently dropping cheap
			// optional coverage. Keep an optional phase that fits the nominal
			// budget on its own and expose the overflow condition on the plan.
			if plan.RequiredEstimatedTotalSeconds > profile.BudgetSeconds && decision.Estimate.DurationSeconds <= profile.BudgetSeconds {
				decision.Selected = true
				decision.Reasons = []string{ReasonSelectedBudgetFit, ReasonBudgetExceededByRequired}
				plan.Selected = append(plan.Selected, decision)
				continue
			}
			decision.Reasons = []string{ReasonOmittedBudget}
			plan.Omitted = append(plan.Omitted, decision)
			continue
		}
		decision.Selected = true
		decision.Reasons = []string{ReasonSelectedBudgetFit}
		plan.Selected = append(plan.Selected, decision)
	}

	plan.EstimatedTotalSeconds = estimatedPlanTotal(profile, plan.Selected)
	if profile.BudgetSeconds > 0 && plan.RequiredEstimatedTotalSeconds > profile.BudgetSeconds {
		plan.BudgetExceededByRequired = true
		plan.BudgetOverflowSeconds = plan.RequiredEstimatedTotalSeconds - profile.BudgetSeconds
	}
	return plan
}

func estimatedPlanTotal(profile Profile, decisions []Decision) int {
	if !profile.ConcurrencyGranted {
		total := 0
		for _, decision := range decisions {
			total += decision.Estimate.DurationSeconds
		}
		return total
	}
	// Selection is cost-ordered, but the executor schedules in catalog order.
	// Reconstruct that order before applying the scheduler's contiguous-batch
	// makespan model.
	scheduled := append([]Decision(nil), decisions...)
	sort.SliceStable(scheduled, func(i, j int) bool {
		return scheduled[i].Candidate.Order < scheduled[j].Candidate.Order
	})
	// The scheduler groups contiguous parallel-safe/provider-serial phases and
	// runs exclusive phases as singleton chains. This is the same conservative
	// grouping contract used by the executor; no capacity is opened here.
	total := 0
	batch := 0
	providers := map[string]struct{}{}
	flush := func() { total += batch; batch = 0; providers = map[string]struct{}{} }
	for _, decision := range scheduled {
		mode := strings.ToLower(strings.TrimSpace(decision.Candidate.ConcurrencyMode))
		if mode != "parallel-safe" && mode != "provider-serial" {
			flush()
			total += decision.Estimate.DurationSeconds
			continue
		}
		if mode == "provider-serial" {
			group := decision.Candidate.ConcurrencyGroup
			if group == "" {
				group = decision.Candidate.Name
			}
			if _, exists := providers[group]; exists {
				flush()
			}
			providers[group] = struct{}{}
		}
		if decision.Estimate.DurationSeconds > batch {
			batch = decision.Estimate.DurationSeconds
		}
	}
	flush()
	return total
}

func estimatePhase(timeoutSeconds int, scenario, global sampleBucket) Estimate {
	scenarioCount := len(scenario.pointDurations)
	globalCount := len(global.pointDurations)
	censored := scenario.censoredCount + global.censoredCount
	excluded := scenario.excludedCount + global.excludedCount
	base := Estimate{CensoredSampleCount: censored, ExcludedSampleCount: excluded}
	switch {
	case scenarioCount >= primaryScenarioSamples:
		base.DurationSeconds = percentileSeconds(scenario.pointDurations, historyPercentile)
		base.Source, base.SampleSize = EstimateSourceScenarioHistory, scenarioCount
		return finishEstimate(base, scenarioCount, censored, excluded, base.Source, globalCount)
	case scenarioCount > 0 && globalCount > 0:
		globalWeight := minInt(globalCount, primaryScenarioSamples)
		base.DurationSeconds = weightedBlend(percentileSeconds(scenario.pointDurations, historyPercentile), scenarioCount, percentileSeconds(global.pointDurations, historyPercentile), globalWeight)
		base.Source, base.SampleSize = EstimateSourceBlendedHistory, scenarioCount+globalWeight
		return finishEstimate(base, scenarioCount, censored, excluded, base.Source, globalCount)
	case scenarioCount > 0:
		base.DurationSeconds = percentileSeconds(scenario.pointDurations, historyPercentile)
		base.Source, base.SampleSize = EstimateSourceScenarioHistory, scenarioCount
		return finishEstimate(base, scenarioCount, censored, excluded, base.Source, globalCount)
	case globalCount > 0:
		base.DurationSeconds = percentileSeconds(global.pointDurations, historyPercentile)
		base.Source, base.SampleSize = EstimateSourceGlobalHistory, globalCount
		return finishEstimate(base, scenarioCount, censored, excluded, base.Source, globalCount)
	default:
		base.DurationSeconds = clampNonNegative(timeoutSeconds)
		base.Source, base.Confidence, base.Unknown = EstimateSourceUnknown, EstimateConfidenceLow, true
		return base
	}
}

func finishEstimate(estimate Estimate, point, censored, excluded int, source EstimateSource, global int) Estimate {
	estimate.PointSampleCount = point
	estimate.CensoredSampleCount = censored
	estimate.ExcludedSampleCount = excluded
	estimate.Confidence = confidenceFor(source, point, global, censored)
	return estimate
}

func defaultOmitReason(policy phasepolicy.Policy) string {
	switch policy.Selection {
	case phasepolicy.SelectionExplicitOnly:
		return ReasonOmittedExplicitOnly
	case phasepolicy.SelectionNeverByDefault:
		return ReasonOmittedNeverDefault
	default:
		return ""
	}
}

func candidateRank(candidate Candidate) int {
	rank := 0
	if !isRequired(candidate.Policy) {
		rank += 1000
	}
	if candidate.Policy.ProviderReadiness == phasepolicy.ProviderReadinessRequiredWhenApplicable {
		rank += 100
	}
	if candidate.Policy.ProviderLifecycle == phasepolicy.ProviderLifecycleStartIfNeeded || candidate.Policy.ProviderLifecycle == phasepolicy.ProviderLifecycleRestartBeforeProbe {
		rank += 100
	}
	if candidate.Order > 0 {
		rank += candidate.Order
	}
	return rank
}

func isRequired(policy phasepolicy.Policy) bool {
	if policy.IsZero() {
		return true
	}
	return policy.ResultGating == phasepolicy.ResultGatingGating && policy.Unavailable == phasepolicy.UnavailableFail && policy.Selection == phasepolicy.SelectionDefaultWhenApplicable
}

type sampleClass int

const (
	samplePoint sampleClass = iota
	sampleCensored
	sampleExcluded
)

func classifySample(sample Sample) sampleClass {
	if sampleDurationSeconds(sample) <= 0 {
		return sampleExcluded
	}
	switch normalize(sample.Status) {
	case phasepolicy.StatusPassed, "success", "succeeded", "failed", "failure", "errored":
		return samplePoint
	case "timeout", "timed_out", "aborted":
		return sampleCensored
	default:
		return sampleExcluded
	}
}

func sampleDurationSeconds(sample Sample) int {
	if sample.DurationMilliseconds > 0 {
		return maxInt(0, int(math.Ceil(float64(sample.DurationMilliseconds)/1000)))
	}
	return clampNonNegative(sample.DurationSeconds)
}

func confidenceFor(source EstimateSource, scenarioCount, globalCount, censored int) EstimateConfidence {
	var confidence EstimateConfidence
	switch source {
	case EstimateSourceScenarioHistory:
		switch {
		case scenarioCount >= 10:
			confidence = EstimateConfidenceHigh
		case scenarioCount >= 3:
			confidence = EstimateConfidenceMedium
		default:
			confidence = EstimateConfidenceLow
		}
	case EstimateSourceBlendedHistory:
		if scenarioCount >= 2 && globalCount >= 5 {
			confidence = EstimateConfidenceMedium
		} else {
			confidence = EstimateConfidenceLow
		}
	case EstimateSourceGlobalHistory:
		if globalCount >= 12 {
			confidence = EstimateConfidenceMedium
		} else {
			confidence = EstimateConfidenceLow
		}
	default:
		confidence = EstimateConfidenceLow
	}
	if censored > 0 {
		if confidence == EstimateConfidenceHigh {
			return EstimateConfidenceMedium
		}
		return EstimateConfidenceLow
	}
	return confidence
}

func percentileSeconds(values []int, percentile float64) int {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int(nil), values...)
	sort.Ints(sorted)
	if len(sorted) == 1 {
		return sorted[0]
	}
	if percentile <= 0 {
		return sorted[0]
	}
	if percentile >= 1 {
		return sorted[len(sorted)-1]
	}
	index := int(math.Round(percentile * float64(len(sorted)-1)))
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func weightedBlend(left, leftWeight, right, rightWeight int) int {
	totalWeight := leftWeight + rightWeight
	if totalWeight <= 0 {
		return 0
	}
	return int(math.Round(float64(left*leftWeight+right*rightWeight) / float64(totalWeight)))
}

func normalize(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
func clampNonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
