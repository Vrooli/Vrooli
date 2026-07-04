package profileplanner

import (
	"math"
	"sort"
	"strings"

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

type Profile struct {
	Name          string
	BudgetSeconds int
	Strategy      ProfileStrategy
}

type Candidate struct {
	Name           string
	DisplayName    string
	TimeoutSeconds int
	Policy         phasepolicy.Policy
	Order          int
}

type Sample struct {
	ScenarioName    string
	PhaseName       string
	Status          string
	DurationSeconds int
}

type Estimate struct {
	DurationSeconds int
	Source          EstimateSource
	Confidence      EstimateConfidence
	SampleSize      int
	Unknown         bool
}

type Decision struct {
	Candidate Candidate
	Estimate  Estimate
	Selected  bool
	Reasons   []string
}

type Plan struct {
	Profile                  Profile
	Selected                 []Decision
	Omitted                  []Decision
	EstimatedTotalSeconds    int
	UnknownEstimateCount     int
	SelectedUnknownEstimates int
}

const (
	historyPercentile         = 0.75
	primaryScenarioSamples    = 5
	maxScenarioSamplesPerKey  = 20
	maxGlobalSamplesPerKey    = 50
	ReasonSelectedRequired    = "selected_required"
	ReasonSelectedBudgetFit   = "selected_budget_fit"
	ReasonOmittedBudget       = "omitted_budget_exceeded"
	ReasonOmittedUnknown      = "omitted_unknown_estimate"
	ReasonOmittedExplicitOnly = "omitted_explicit_only"
	ReasonOmittedNeverDefault = "omitted_never_by_default"
)

type Estimator struct {
	scenarioName    string
	scenarioBuckets map[string][]int
	globalBuckets   map[string][]int
}

func NewEstimator(scenarioName string, samples []Sample) Estimator {
	scenarioBuckets := make(map[string][]int)
	globalBuckets := make(map[string][]int)
	for _, sample := range samples {
		key := normalize(sample.PhaseName)
		if key == "" {
			continue
		}
		duration := clampNonNegative(sample.DurationSeconds)
		if duration == 0 || !sampleCounts(sample) {
			continue
		}
		if strings.EqualFold(sample.ScenarioName, scenarioName) && len(scenarioBuckets[key]) < maxScenarioSamplesPerKey {
			scenarioBuckets[key] = append(scenarioBuckets[key], duration)
			continue
		}
		if len(globalBuckets[key]) < maxGlobalSamplesPerKey {
			globalBuckets[key] = append(globalBuckets[key], duration)
		}
	}
	return Estimator{
		scenarioName:    scenarioName,
		scenarioBuckets: scenarioBuckets,
		globalBuckets:   globalBuckets,
	}
}

func (e Estimator) Estimate(phaseName string, timeoutSeconds int) Estimate {
	key := normalize(phaseName)
	return estimatePhase(timeoutSeconds, e.scenarioBuckets[key], e.globalBuckets[key])
}

func PlanProfile(profile Profile, candidates []Candidate, estimator Estimator) Plan {
	ordered := append([]Candidate(nil), candidates...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return candidateRank(ordered[i]) < candidateRank(ordered[j])
	})

	plan := Plan{Profile: profile}
	for _, candidate := range ordered {
		estimate := estimator.Estimate(candidate.Name, candidate.TimeoutSeconds)
		if estimate.Unknown {
			plan.UnknownEstimateCount++
		}

		decision := Decision{Candidate: candidate, Estimate: estimate}
		if omitReason := defaultOmitReason(candidate.Policy); omitReason != "" {
			decision.Reasons = []string{omitReason}
			plan.Omitted = append(plan.Omitted, decision)
			continue
		}
		if estimate.Unknown && !isRequired(candidate.Policy) {
			decision.Reasons = []string{ReasonOmittedUnknown}
			plan.Omitted = append(plan.Omitted, decision)
			continue
		}

		required := isRequired(candidate.Policy)
		nextTotal := plan.EstimatedTotalSeconds + estimate.DurationSeconds
		if !required && profile.BudgetSeconds > 0 && nextTotal > profile.BudgetSeconds {
			decision.Reasons = []string{ReasonOmittedBudget}
			plan.Omitted = append(plan.Omitted, decision)
			continue
		}

		decision.Selected = true
		if required {
			decision.Reasons = []string{ReasonSelectedRequired}
		} else {
			decision.Reasons = []string{ReasonSelectedBudgetFit}
		}
		plan.Selected = append(plan.Selected, decision)
		plan.EstimatedTotalSeconds = nextTotal
		if estimate.Unknown {
			plan.SelectedUnknownEstimates++
		}
	}
	return plan
}

func estimatePhase(timeoutSeconds int, scenarioDurations, globalDurations []int) Estimate {
	scenarioCount := len(scenarioDurations)
	globalCount := len(globalDurations)
	switch {
	case scenarioCount >= primaryScenarioSamples:
		return Estimate{
			DurationSeconds: percentileSeconds(scenarioDurations, historyPercentile),
			Source:          EstimateSourceScenarioHistory,
			Confidence:      confidenceFor(EstimateSourceScenarioHistory, scenarioCount, globalCount),
			SampleSize:      scenarioCount,
		}
	case scenarioCount > 0 && globalCount > 0:
		globalWeight := minInt(globalCount, primaryScenarioSamples)
		blended := weightedBlend(
			percentileSeconds(scenarioDurations, historyPercentile), scenarioCount,
			percentileSeconds(globalDurations, historyPercentile), globalWeight,
		)
		return Estimate{
			DurationSeconds: blended,
			Source:          EstimateSourceBlendedHistory,
			Confidence:      confidenceFor(EstimateSourceBlendedHistory, scenarioCount, globalCount),
			SampleSize:      scenarioCount + globalWeight,
		}
	case scenarioCount > 0:
		return Estimate{
			DurationSeconds: percentileSeconds(scenarioDurations, historyPercentile),
			Source:          EstimateSourceScenarioHistory,
			Confidence:      confidenceFor(EstimateSourceScenarioHistory, scenarioCount, globalCount),
			SampleSize:      scenarioCount,
		}
	case globalCount > 0:
		return Estimate{
			DurationSeconds: percentileSeconds(globalDurations, historyPercentile),
			Source:          EstimateSourceGlobalHistory,
			Confidence:      confidenceFor(EstimateSourceGlobalHistory, scenarioCount, globalCount),
			SampleSize:      globalCount,
		}
	default:
		return Estimate{
			DurationSeconds: clampNonNegative(timeoutSeconds),
			Source:          EstimateSourceUnknown,
			Confidence:      EstimateConfidenceLow,
			Unknown:         true,
		}
	}
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
	if candidate.Policy.ProviderLifecycle == phasepolicy.ProviderLifecycleStartIfNeeded ||
		candidate.Policy.ProviderLifecycle == phasepolicy.ProviderLifecycleRestartBeforeProbe {
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
	return policy.ResultGating == phasepolicy.ResultGatingGating &&
		policy.Unavailable == phasepolicy.UnavailableFail &&
		policy.Selection == phasepolicy.SelectionDefaultWhenApplicable
}

func sampleCounts(sample Sample) bool {
	switch normalize(sample.Status) {
	case phasepolicy.StatusPassed, "success", "succeeded":
		return true
	default:
		return false
	}
}

func confidenceFor(source EstimateSource, scenarioCount, globalCount int) EstimateConfidence {
	switch source {
	case EstimateSourceScenarioHistory:
		switch {
		case scenarioCount >= 10:
			return EstimateConfidenceHigh
		case scenarioCount >= 3:
			return EstimateConfidenceMedium
		default:
			return EstimateConfidenceLow
		}
	case EstimateSourceBlendedHistory:
		if scenarioCount >= 2 && globalCount >= 5 {
			return EstimateConfidenceMedium
		}
		return EstimateConfidenceLow
	case EstimateSourceGlobalHistory:
		if globalCount >= 12 {
			return EstimateConfidenceMedium
		}
		return EstimateConfidenceLow
	default:
		return EstimateConfidenceLow
	}
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

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func clampNonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
